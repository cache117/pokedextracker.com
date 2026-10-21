package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
)

//go:embed fire_red_leaf_green_locations.json
var fireRedLeafGreenLocations []byte

type personalLocation struct {
	NationalID        int      `json:"national_id"`
	Name              string   `json:"name"`
	SourceURL         string   `json:"source_url"`
	RetrievedAt       string   `json:"retrieved_at"`
	SourceSHA256      string   `json:"source_sha256"`
	AdditionalSources []string `json:"additional_sources,omitempty"`
	FireRed           []string `json:"fire_red"`
	LeafGreen         []string `json:"leaf_green"`
}

func parsePersonalLocations(raw []byte) ([]personalLocation, error) {
	var rows []personalLocation
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	if len(rows) != 386 {
		return nil, fmt.Errorf("%w: expected 386 FR/LG location records", errImportRejected)
	}
	seen := map[int]bool{}
	for _, row := range rows {
		if row.NationalID < 1 || row.NationalID > 386 || seen[row.NationalID] {
			return nil, fmt.Errorf("%w: duplicate or invalid FR/LG National ID %d", errImportRejected, row.NationalID)
		}
		seen[row.NationalID] = true
		expectedURL := fmt.Sprintf("https://www.serebii.net/pokedex-rs/%03d.shtml", row.NationalID)
		_, dateErr := time.Parse(time.RFC3339Nano, row.RetrievedAt)
		if row.Name == "" || row.SourceURL != expectedURL || len(row.SourceSHA256) != 64 || dateErr != nil {
			return nil, fmt.Errorf("%w: missing FR/LG source metadata for %d", errImportRejected, row.NationalID)
		}
		for _, values := range [][]string{row.FireRed, row.LeafGreen} {
			if len(values) == 0 {
				return nil, fmt.Errorf("%w: missing FR/LG game entry for %d", errImportRejected, row.NationalID)
			}
			for _, value := range values {
				if strings.TrimSpace(value) == "" {
					return nil, fmt.Errorf("%w: blank FR/LG location for %d", errImportRejected, row.NationalID)
				}
			}
		}
	}
	return rows, nil
}

func importPersonalLocations(tx *pg.Tx) error {
	rows, err := parsePersonalLocations(fireRedLeafGreenLocations)
	if err != nil {
		return err
	}
	for _, row := range rows {
		for game, values := range map[string][]string{"fire_red": row.FireRed, "leaf_green": row.LeafGreen} {
			result, err := tx.Exec(`INSERT INTO locations (game_id,pokemon_id,value,values)
SELECT ?,dtp.pokemon_id,?,? FROM dex_types_pokemon dtp JOIN pokemon p ON p.id=dtp.pokemon_id
WHERE dtp.dex_type_id=1000002 AND p.national_id=?
ON CONFLICT (game_id,pokemon_id) DO UPDATE SET value=EXCLUDED.value,values=EXCLUDED.values`,
				game, strings.Join(values, "\n"), pg.Array(values), row.NationalID)
			if err != nil {
				return err
			}
			if result.RowsAffected() != 1 {
				return fmt.Errorf("%w: FR/LG identity mismatch for %s/%d", errImportRejected, game, row.NationalID)
			}
		}
	}
	return nil
}
