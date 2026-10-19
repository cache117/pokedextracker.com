// personal-import bootstraps an EMPTY personal database from a public snapshot.
// It refuses to overwrite existing accounts or progress.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pokedextracker/pokedextracker.com/pkg/captures"
	"github.com/pokedextracker/pokedextracker.com/pkg/config"
	"github.com/pokedextracker/pokedextracker.com/pkg/database"
	"github.com/pokedextracker/pokedextracker.com/pkg/dextypes"
	"github.com/pokedextracker/pokedextracker.com/pkg/games"
	"github.com/pokedextracker/pokedextracker.com/pkg/pokemon"
	"github.com/pokedextracker/pokedextracker.com/pkg/users"
	"golang.org/x/crypto/bcrypt"
)

type snapshot struct {
	Profile  users.User                    `json:"profile"`
	Games    []games.Game                  `json:"games"`
	DexTypes []dextypes.DexType            `json:"dex_types"`
	Captures map[string][]captures.Capture `json:"captures_by_slug"`
}

var errImportRejected = errors.New("reference import rejected")

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("%w: usage: personal-import snapshot.json", errImportRejected)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		return err
	}
	var s snapshot
	if err = json.Unmarshal(raw, &s); err != nil {
		return err
	}
	if len(s.Profile.Dexes) == 0 || os.Getenv("PERSONAL_PASSWORD") == "" {
		return fmt.Errorf("%w: snapshot and PERSONAL_PASSWORD are required", errImportRejected)
	}
	cfg, err := config.New()
	if err != nil {
		return err
	}
	db, err := database.New("personal-import", cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	hash, err := bcrypt.GenerateFromPassword([]byte(os.Getenv("PERSONAL_PASSWORD")), cfg.BcryptCost)
	if err != nil {
		return err
	}
	families := map[string]*games.GameFamily{}
	mons := map[int]*pokemon.Pokemon{}
	membership := map[int][]captures.Capture{}
	for _, g := range s.Games {
		families[g.GameFamily.ID] = g.GameFamily
	}
	for _, d := range s.Profile.Dexes {
		cs, ok := s.Captures[d.Slug]
		if !ok || len(cs) != d.Total {
			return fmt.Errorf("%w: incomplete dex %s", errImportRejected, d.Slug)
		}
		caught := 0
		for _, c := range cs {
			if c.Pokemon == nil || c.Pokemon.GameFamily == nil {
				return fmt.Errorf("%w: missing Pokémon data in %s", errImportRejected, d.Slug)
			}
			if c.Captured {
				caught++
			}
			p := c.Pokemon
			families[p.GameFamily.ID] = p.GameFamily
			mons[p.ID] = &pokemon.Pokemon{ID: p.ID, NationalID: p.NationalID, Name: p.Name, Form: p.Form, GameFamilyID: p.GameFamily.ID, NationalOrder: p.NationalID, EvolutionFamilyID: 0}
		}
		if caught != d.Caught {
			return fmt.Errorf("%w: caught count mismatch in %s", errImportRejected, d.Slug)
		}
		if previous, exists := membership[d.DexType.ID]; exists {
			if len(previous) != len(cs) {
				return fmt.Errorf("%w: conflicting membership for dex type %d", errImportRejected, d.DexType.ID)
			}
			for i := range cs {
				if previous[i].Pokemon.ID != cs[i].Pokemon.ID || previous[i].Pokemon.DexNumber != cs[i].Pokemon.DexNumber {
					return fmt.Errorf("%w: conflicting order or numbering for dex type %d", errImportRejected, d.DexType.ID)
				}
			}
		} else {
			membership[d.DexType.ID] = cs
		}
	}
	err = db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		var n int
		if _, e := tx.QueryOne(pg.Scan(&n), "SELECT count(*) FROM users"); e != nil {
			return e
		}
		if n != 0 {
			return fmt.Errorf("%w: refusing import: database already has accounts", errImportRejected)
		}
		for _, f := range families {
			if _, e := tx.Exec(`INSERT INTO game_families (id,generation,regional_total,national_total,regional_support,national_support,"order",published) VALUES (?,?,?,?,?,?,?,?)`, f.ID, f.Generation, f.RegionalTotal, f.NationalTotal, f.RegionalSupport, f.NationalSupport, f.Order, f.Published); e != nil {
				return e
			}
		}
		for _, g := range s.Games {
			g := g
			g.GameFamilyID = g.GameFamily.ID
			if _, e := tx.Model(&g).Insert(); e != nil {
				return e
			}
		}
		// IDs are preserved because the sprite/data mappings and saved dexes use them.
		ids := make([]int, 0, len(mons))
		for id := range mons {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			if _, e := tx.Model(mons[id]).Insert(); e != nil {
				return e
			}
		}
		// Base types are inserted before references to them.
		for _, dt := range s.DexTypes {
			dt := dt
			if dt.BaseDexTypeID != nil {
				continue
			}
			if _, e := tx.Model(&dt).Insert(); e != nil {
				return e
			}
		}
		for _, dt := range s.DexTypes {
			dt := dt
			if dt.BaseDexTypeID == nil {
				continue
			}
			if _, e := tx.Model(&dt).Insert(); e != nil {
				return e
			}
		}
		for typeID, cs := range membership {
			for i, c := range cs {
				p := c.Pokemon
				row := dextypes.DexTypePokemon{DexTypeID: typeID, PokemonID: p.ID, Box: p.Box, Order: i + 1, DexNumber: p.DexNumber}
				if _, e := tx.Model(&row).Insert(); e != nil {
					return e
				}
			}
		}
		// Hide reference definitions for which this snapshot contains no membership.
		if _, e := tx.Exec(`DELETE FROM dex_types WHERE id NOT IN (SELECT DISTINCT dex_type_id FROM dex_types_pokemon) AND id NOT IN (SELECT base_dex_type_id FROM dex_types WHERE base_dex_type_id IS NOT NULL)`); e != nil {
			return e
		}
		u := s.Profile
		u.Password = string(hash)
		u.Dexes = nil
		u.StripeID = nil
		u.LastIP = nil
		u.LastLogin = nil
		if _, e := tx.Model(&u).Insert(); e != nil {
			return e
		}
		for _, source := range s.Profile.Dexes {
			d := *source
			d.UserID = u.ID
			d.GameID = d.Game.ID
			d.DexTypeID = d.DexType.ID
			if _, e := tx.Model(&d).Insert(); e != nil {
				return e
			}
			for _, c := range s.Captures[d.Slug] {
				if !c.Captured {
					continue
				}
				row := captures.Capture{DexID: d.ID, PokemonID: c.Pokemon.ID, Captured: true, DateCreated: time.Now(), DateModified: time.Now()}
				if _, e := tx.Model(&row).Insert(); e != nil {
					return e
				}
			}
		}
		// Advance sequences past imported explicit IDs.
		for _, table := range []string{"users", "pokemon", "dexes", "dex_types"} {
			if _, e := tx.Exec(fmt.Sprintf(`SELECT setval(pg_get_serial_sequence('%s','id'), GREATEST(COALESCE((SELECT max(id) FROM %s),1),1))`, table, table)); e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("Imported %d dexes, %d Pokémon records and %d populated dex types.\n", len(s.Profile.Dexes), len(mons), len(membership))
	return nil
}
