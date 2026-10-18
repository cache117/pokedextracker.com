// Imports a limited batch of cached responses, without network access.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pokedextracker/pokedextracker.com/pkg/config"
	"github.com/pokedextracker/pokedextracker.com/pkg/database"
	"github.com/pokedextracker/pokedextracker.com/pkg/games"
	"github.com/pokedextracker/pokedextracker.com/pkg/pokemon"
)

type record struct {
	ID         int    `json:"id"`
	NationalID int    `json:"national_id"`
	Name       string `json:"name"`
	Locations  []struct {
		Game   games.Game `json:"game"`
		Value  []string   `json:"value"`
		Values []string   `json:"values"`
	} `json:"locations"`
	Family pokemon.EvolutionFamily `json:"evolution_family"`
	Meta   metadata                `json:"-"`
}
type metadata struct {
	SourceURL   string    `json:"source_url"`
	RetrievedAt time.Time `json:"retrieved_at"`
	SHA256      string    `json:"sha256"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) != 2 && len(os.Args) != 4 {
		return fmt.Errorf("usage: reference-sample-import reference-snapshot-folder [start-id count]")
	}
	start, count := 1, 10
	if len(os.Args) == 4 {
		var err error
		start, err = strconv.Atoi(os.Args[2])
		if err != nil || start < 1 {
			return fmt.Errorf("invalid start ID")
		}
		count, err = strconv.Atoi(os.Args[3])
		if err != nil || count < 1 || count > 10 {
			return fmt.Errorf("batch must contain 1 to 10 Pokémon")
		}
	}
	var records []record
	for id := start; id < start+count; id++ {
		raw, err := os.ReadFile(filepath.Join(os.Args[1], fmt.Sprintf("%d.json", id)))
		if err != nil {
			return err
		}
		var r record
		if err = json.Unmarshal(raw, &r); err != nil {
			return err
		}
		meta, err := os.ReadFile(filepath.Join(os.Args[1], fmt.Sprintf("%d.meta.json", id)))
		if err != nil {
			return err
		}
		if err = json.Unmarshal(meta, &r.Meta); err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		if r.ID != id || r.Meta.SHA256 != hex.EncodeToString(sum[:]) || r.Meta.SourceURL != fmt.Sprintf("https://pokedextracker.com/api/pokemon/%d?dex_type=21", id) || r.Meta.RetrievedAt.IsZero() {
			return fmt.Errorf("invalid record or provenance for %d", id)
		}
		// The sample contains simple, unbranched families. Refuse ambiguous mappings.
		if len(r.Family.Pokemon) == 0 || len(r.Family.Evolutions) != len(r.Family.Pokemon)-1 {
			return fmt.Errorf("unexpected evolution stages for %d", id)
		}
		for _, stage := range r.Family.Pokemon {
			if len(stage) != 1 {
				return fmt.Errorf("branched families require an explicit edge mapping; refusing %d", id)
			}
		}
		for _, stage := range r.Family.Evolutions {
			if len(stage) != 1 {
				return fmt.Errorf("ambiguous evolution trigger for %d", id)
			}
		}
		records = append(records, r)
	}
	cfg, err := config.New()
	if err != nil {
		return err
	}
	db, err := database.New("reference-sample-import", cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	locations := 0
	err = db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		for _, r := range records {
			var nationalID int
			if _, err := tx.QueryOne(pg.Scan(&nationalID), `SELECT national_id FROM pokemon WHERE id=?`, r.ID); err != nil {
				return err
			}
			if nationalID != r.NationalID {
				return fmt.Errorf("identity mismatch for %d", r.ID)
			}
			for _, l := range r.Locations {
				if l.Game.ID == "" || len(l.Value) == 0 {
					return fmt.Errorf("incomplete location for %d", r.ID)
				}
				if strings.Join(l.Value, "\x00") != strings.Join(l.Values, "\x00") {
					return fmt.Errorf("conflicting location fields for %d", r.ID)
				}
				if _, err := tx.Exec(`INSERT INTO locations (game_id,pokemon_id,value,"values") VALUES (?,?,?,?) ON CONFLICT (game_id,pokemon_id) DO UPDATE SET value=EXCLUDED.value,"values"=EXCLUDED."values"`, l.Game.ID, r.ID, strings.Join(l.Value, ", "), pg.Array(l.Value)); err != nil {
					return err
				}
				locations++
			}
			familyID := r.Family.Pokemon[0][0].ID
			for _, stage := range r.Family.Pokemon {
				member := stage[0]
				var existing int
				if _, err := tx.QueryOne(pg.Scan(&existing), `SELECT national_id FROM pokemon WHERE id=?`, member.ID); err != nil {
					return err
				}
				if existing != member.NationalID {
					return fmt.Errorf("evolution member mismatch: %d", member.ID)
				}
				if _, err := tx.Exec(`UPDATE pokemon SET evolution_family_id=? WHERE id=?`, familyID, member.ID); err != nil {
					return err
				}
			}
			for i, stage := range r.Family.Evolutions {
				e := *stage[0]
				e.EvolvingPokemonID = r.Family.Pokemon[i][0].ID
				e.EvolvedPokemonID = r.Family.Pokemon[i+1][0].ID
				e.EvolutionFamilyID = familyID
				e.Stage = i + 1
				if _, err := tx.Model(&e).OnConflict("(evolving_pokemon_id,evolved_pokemon_id) DO UPDATE").Set("evolution_family_id=EXCLUDED.evolution_family_id, stage=EXCLUDED.stage, trigger=EXCLUDED.trigger, level=EXCLUDED.level, candy_count=EXCLUDED.candy_count, stone=EXCLUDED.stone, held_item=EXCLUDED.held_item, notes=EXCLUDED.notes").Insert(); err != nil {
					return err
				}
			}
			if _, err := tx.Exec(`INSERT INTO personal_reference_imports (pokemon_id,source_url,retrieved_at,sha256) VALUES (?,?,?,?) ON CONFLICT (pokemon_id) DO UPDATE SET source_url=EXCLUDED.source_url,retrieved_at=EXCLUDED.retrieved_at,sha256=EXCLUDED.sha256`, r.ID, r.Meta.SourceURL, r.Meta.RetrievedAt, r.Meta.SHA256); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("Imported reference details for %d Pokémon and %d game-specific location records.\n", len(records), locations)
	return nil
}
