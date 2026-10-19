// personal-seed-import merges upstream reference CSVs without replacing user data
// or the personal catalogue. It deliberately does not call the dev seed reset.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pokedextracker/pokedextracker.com/pkg/config"
	"github.com/pokedextracker/pokedextracker.com/pkg/database"
)

type tableSpec struct {
	name string
	keys []string
}

var tables = []tableSpec{
	{"game_families", []string{"id"}}, {"games", []string{"id"}},
	{"pokemon", []string{"id"}}, {"dex_types", []string{"id"}},
	{"dex_types_pokemon", []string{"dex_type_id", "pokemon_id"}},
	{"boxes", []string{"game_family_id", "regional", "pokemon_id"}},
	{"locations", []string{"game_id", "pokemon_id"}},
	{"evolutions", []string{"evolving_pokemon_id", "evolved_pokemon_id"}},
}
var identifier = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
var revisionPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)

func quote(s string) string { return `"` + s + `"` }
func quoted(cols []string) string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = quote(c)
	}
	return strings.Join(out, ",")
}

var errImportRejected = errors.New("reference import rejected")

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) != 3 || !revisionPattern.MatchString(os.Args[2]) {
		return fmt.Errorf("%w: usage: personal-seed-import seeds-directory upstream-commit-sha", errImportRejected)
	}
	if os.Getenv("ENVIRONMENT") != "local" {
		return fmt.Errorf("%w: only local instances may use this importer", errImportRejected)
	}
	cfg, err := config.New()
	if err != nil {
		return err
	}
	db, err := database.New("personal-seed-import", cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	counts := map[string]int{}
	err = db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		// No application write can race this import. Reference FKs stay enabled.
		if _, err := tx.Exec(`SELECT pg_advisory_xact_lock(170017)`); err != nil {
			return err
		}
		protected := []string{"users", "dexes", "captures", "personal_catalog_settings", "personal_location_overrides"}
		fingerprints := map[string]string{}
		for _, table := range protected {
			if _, err := tx.Exec(`LOCK TABLE ` + quote(table) + ` IN SHARE MODE`); err != nil {
				return err
			}
			var sum string
			if _, err := tx.QueryOne(pg.Scan(&sum), `SELECT md5(COALESCE(string_agg(row_to_json(t)::text,E'\n' ORDER BY row_to_json(t)::text),'')) FROM `+quote(table)+` t`); err != nil {
				return err
			}
			fingerprints[table] = sum
		}
		if _, err := tx.Exec(`ALTER TABLE boxes DROP CONSTRAINT boxes_pkey; ALTER TABLE boxes ADD PRIMARY KEY(game_family_id,regional,pokemon_id)`); err != nil {
			return err
		}
		for _, spec := range tables {
			raw, err := os.ReadFile(filepath.Join(os.Args[1], spec.name+".csv"))
			if err != nil {
				return err
			}
			rows, err := csv.NewReader(bytes.NewReader(raw)).ReadAll()
			if err != nil {
				return err
			}
			if len(rows) < 2 {
				return fmt.Errorf("%w: empty seed table %s", errImportRejected, spec.name)
			}
			columns := rows[0]
			seen := map[string]bool{}
			for _, c := range columns {
				if !identifier.MatchString(c) || seen[c] {
					return fmt.Errorf("%w: invalid CSV header for %s", errImportRejected, spec.name)
				}
				seen[c] = true
			}
			for _, k := range spec.keys {
				if !seen[k] {
					return fmt.Errorf("%w: missing primary key for %s", errImportRejected, spec.name)
				}
			}
			temp := "seed_" + spec.name
			if _, err := tx.Exec(`CREATE TEMP TABLE ` + quote(temp) + ` (LIKE ` + quote(spec.name) + ` INCLUDING DEFAULTS) ON COMMIT DROP`); err != nil {
				return err
			}
			if _, err := tx.CopyFrom(bytes.NewReader(raw), `COPY `+quote(temp)+` (`+quoted(columns)+`) FROM STDIN WITH(FORMAT csv,HEADER)`); err != nil {
				return fmt.Errorf("stage %s: %w", spec.name, err)
			}
			if spec.name == "pokemon" {
				var conflicts int
				if _, err := tx.QueryOne(pg.Scan(&conflicts), `SELECT count(*) FROM pokemon p JOIN seed_pokemon s USING(id) WHERE p.national_id<>s.national_id OR p.form IS DISTINCT FROM s.form`); err != nil {
					return err
				}
				if conflicts > 0 {
					return fmt.Errorf("%w: upstream Pokémon IDs conflict with %d local identities", errImportRejected, conflicts)
				}
			}
			sets := []string{}
			for _, c := range columns {
				key := false
				for _, k := range spec.keys {
					key = key || k == c
				}
				if !key {
					sets = append(sets, quote(c)+`=EXCLUDED.`+quote(c))
				}
			}
			sql := `INSERT INTO ` + quote(spec.name) + ` (` + quoted(columns) + `) SELECT ` + quoted(columns) + ` FROM ` + quote(temp) + ` ON CONFLICT (` + quoted(spec.keys) + `) DO UPDATE SET ` + strings.Join(sets, ",")
			if _, err := tx.Exec(sql); err != nil {
				return fmt.Errorf("merge %s: %w", spec.name, err)
			}
			counts[spec.name] = len(rows) - 1
		}
		// Canonical memberships for upstream templates; local-only templates survive.
		// Refuse to hide a caught Pokémon as a side effect of updating a template.
		var lost int
		if _, err := tx.QueryOne(pg.Scan(&lost), `SELECT count(*) FROM captures c JOIN dexes d ON d.id=c.dex_id JOIN dex_types_pokemon p ON p.dex_type_id=d.dex_type_id AND p.pokemon_id=c.pokemon_id WHERE c.captured AND d.dex_type_id IN(SELECT id FROM seed_dex_types) AND NOT EXISTS(SELECT 1 FROM seed_dex_types_pokemon s WHERE s.dex_type_id=p.dex_type_id AND s.pokemon_id=p.pokemon_id)`); err != nil {
			return err
		}
		if lost > 0 {
			return fmt.Errorf("%w: refusing to hide %d caught Pokémon absent from upstream templates", errImportRejected, lost)
		}
		if _, err := tx.Exec(`DELETE FROM dex_types_pokemon p WHERE p.dex_type_id IN(SELECT id FROM seed_dex_types) AND NOT EXISTS(SELECT 1 FROM seed_dex_types_pokemon s WHERE s.dex_type_id=p.dex_type_id AND s.pokemon_id=p.pokemon_id)`); err != nil {
			return err
		}
		if err := importPersonalDexes(tx); err != nil {
			return fmt.Errorf("personal dex extensions: %w", err)
		}
		// Personal names stay personal; the source CSVs remain byte-for-byte upstream.
		if _, err := tx.Exec(`UPDATE games g SET name=j->>'name' FROM personal_catalog_settings c, jsonb_array_elements(c.document->'games') j WHERE g.id=j->>'id'`); err != nil {
			return err
		}
		sum := sha256.New()
		for _, spec := range tables {
			raw, err := os.ReadFile(filepath.Join(os.Args[1], spec.name+".csv"))
			if err != nil {
				return err
			}
			sum.Write([]byte(spec.name + "\x00"))
			sum.Write(raw)
		}
		hash := hex.EncodeToString(sum.Sum(nil))
		url := "https://github.com/pokedextracker/pokedextracker.com/tree/" + os.Args[2] + "/seeds"
		if _, err := tx.Exec(`INSERT INTO personal_reference_imports(pokemon_id,source_url,retrieved_at,sha256) SELECT id,?,?,? FROM seed_pokemon ON CONFLICT(pokemon_id) DO UPDATE SET source_url=EXCLUDED.source_url,retrieved_at=EXCLUDED.retrieved_at,sha256=EXCLUDED.sha256`, url, time.Now().UTC(), hash); err != nil {
			return err
		}
		for _, table := range []string{"pokemon", "dex_types"} {
			if _, err := tx.Exec(`SELECT setval(pg_get_serial_sequence(?, 'id'), COALESCE((SELECT max(id) FROM `+quote(table)+`),1), true)`, table); err != nil {
				return err
			}
		}
		for _, table := range protected {
			var sum string
			if _, err := tx.QueryOne(pg.Scan(&sum), `SELECT md5(COALESCE(string_agg(row_to_json(t)::text,E'\n' ORDER BY row_to_json(t)::text),'')) FROM `+quote(table)+` t`); err != nil {
				return err
			}
			if sum != fingerprints[table] {
				return fmt.Errorf("%w: protected table %s changed; rolling back", errImportRejected, table)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, spec := range tables {
		fmt.Printf("Merged %s: %d upstream rows\n", spec.name, counts[spec.name])
	}
	fmt.Println("Applied personal FireRed/LeafGreen templates: Regional (151) and Full National (386).")
	fmt.Println("Verified users, dexes, captures, personal settings, and encounter overrides unchanged.")
	return nil
}
