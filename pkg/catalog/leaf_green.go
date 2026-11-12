package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/labstack/echo/v5"
	"github.com/pokedextracker/pokedextracker.com/pkg/errcodes"
)

type collectionView struct {
	CollectionFilter
	Revision                     int   `json:"revision"`
	DefaultUnavailablePokemonIDs []int `json:"default_unavailable_pokemon_ids"`
	IsOwner                      bool  `json:"is_owner"`
}

func validateCollectionFilterShape(filters []CollectionFilter) error {
	if len(filters) > 1000 {
		return fmt.Errorf("%w: too many collection filters", errInvalidCatalogue)
	}
	seen := map[int]bool{}
	for _, f := range filters {
		if f.DexID <= 0 || seen[f.DexID] || len(f.UnavailablePokemonIDs) > 386 {
			return fmt.Errorf("%w: invalid or duplicate collection filter", errInvalidCatalogue)
		}
		seen[f.DexID] = true
		ids := map[int]bool{}
		for _, id := range f.UnavailablePokemonIDs {
			if id <= 0 || ids[id] {
				return fmt.Errorf("%w: invalid or duplicate unavailable Pokémon", errInvalidCatalogue)
			}
			ids[id] = true
		}
	}
	return nil
}

func validateCollectionFilter(ctx context.Context, db orm.DB, f CollectionFilter) error {
	var count int
	_, err := db.QueryOneContext(ctx, pg.Scan(&count), `SELECT count(*) FROM dexes d JOIN dex_types dt ON dt.id=d.dex_type_id WHERE d.id=? AND d.user_id=(SELECT min(id) FROM users) AND d.game_id='leaf_green' AND dt.game_family_id='fire_red_leaf_green' AND 'full national'=ANY(dt.tags)`, f.DexID)
	if err != nil {
		return err
	}
	if count != 1 {
		return errcodes.ValidationError("Choose an owner's LeafGreen Full National collection.")
	}
	if len(f.UnavailablePokemonIDs) == 0 {
		return nil
	}
	_, err = db.QueryOneContext(ctx, pg.Scan(&count), `SELECT count(*) FROM dex_types_pokemon WHERE dex_type_id=(SELECT dex_type_id FROM dexes WHERE id=?) AND pokemon_id IN (?)`, f.DexID, pg.In(f.UnavailablePokemonIDs))
	if err != nil {
		return err
	}
	if count != len(f.UnavailablePokemonIDs) {
		return errcodes.ValidationError("The unavailable list must contain Pokémon from this collection.")
	}
	return nil
}

func loadCollectionView(ctx context.Context, db *pg.DB, id int, isOwner bool) (collectionView, error) {
	out := collectionView{IsOwner: isOwner, DefaultUnavailablePokemonIDs: []int{},
		DexID:                 id,
		UnavailablePokemonIDs: []int{}}
	if err := validateCollectionFilter(ctx, db, out.CollectionFilter); err != nil {
		return out, err
	}
	_, err := db.QueryContext(ctx, &out.DefaultUnavailablePokemonIDs, `SELECT m.pokemon_id FROM dex_types_pokemon m JOIN locations l ON l.pokemon_id=m.pokemon_id AND l.game_id='leaf_green' WHERE m.dex_type_id=(SELECT dex_type_id FROM dexes WHERE id=?) AND l.values[1] ILIKE '%Not currently obtainable through supported play or local FR/LG trading.%' ORDER BY m."order"`, id)
	if err != nil {
		return out, err
	}
	out.UnavailablePokemonIDs = out.DefaultUnavailablePokemonIDs
	cfg, err := LoadSettings(ctx, db)
	if err != nil {
		return out, err
	}
	out.Revision = cfg.Revision
	for _, filter := range cfg.Settings.LeafGreenFilters {
		if filter.DexID == id {
			out.CollectionFilter = filter
			break
		}
	}
	return out, nil
}

func registerLeafGreenRoutes(e *echo.Group, db *pg.DB, enforce, optional echo.MiddlewareFunc, owner func(*echo.Context) bool) {
	e.GET("/personal/catalog/leaf-green/:id", func(c *echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return errcodes.ValidationError("Invalid collection.")
		}
		out, err := loadCollectionView(c.Request().Context(), db, id, owner(c))
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, out)
	}, optional)
	e.POST("/personal/catalog/leaf-green/:id", func(c *echo.Context) error {
		if !owner(c) {
			return errcodes.Forbidden("editing this collection's unavailable list")
		}
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return errcodes.ValidationError("Invalid collection.")
		}
		var in struct {
			Revision              int   `json:"revision"`
			HideUnavailable       bool  `json:"hide_unavailable"`
			UnavailablePokemonIDs []int `json:"unavailable_pokemon_ids"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(c.Response(), c.Request().Body, 32768))
		dec.DisallowUnknownFields()
		if err = dec.Decode(&in); err != nil {
			return errcodes.ValidationError("Invalid unavailable list.")
		}
		if err = dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			return errcodes.ValidationError("Unexpected extra configuration.")
		}
		filter := CollectionFilter{DexID: id, HideUnavailable: in.HideUnavailable, UnavailablePokemonIDs: in.UnavailablePokemonIDs}
		if filter.UnavailablePokemonIDs == nil {
			filter.UnavailablePokemonIDs = []int{}
		}
		if err = validateCollectionFilterShape([]CollectionFilter{filter}); err != nil {
			return errcodes.ValidationError(err.Error())
		}
		base, err := LoadSettings(c.Request().Context(), db)
		if err != nil {
			return err
		}
		err = db.RunInTransaction(c.Request().Context(), func(tx *pg.Tx) error {
			var revision int
			if _, err := tx.QueryOne(pg.Scan(&revision), `SELECT revision FROM personal_catalog_settings WHERE id=1 FOR UPDATE`); err != nil {
				return err
			}
			if revision != in.Revision || revision != base.Revision {
				return errcodes.ValidationError("The settings changed in another window. Reload availability before saving.")
			}
			if err := validateCollectionFilter(c.Request().Context(), tx, filter); err != nil {
				return err
			}
			cfg := base.Settings
			found := false
			for i, f := range cfg.LeafGreenFilters {
				if f.DexID == id {
					cfg.LeafGreenFilters[i] = filter
					found = true
					break
				}
			}
			if !found {
				cfg.LeafGreenFilters = append(cfg.LeafGreenFilters, filter)
			}
			encoded, err := json.Marshal(cfg)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`UPDATE personal_catalog_settings SET document=?::jsonb,revision=revision+1 WHERE id=1`, string(encoded))
			return err
		})
		if err != nil {
			return err
		}
		out, err := loadCollectionView(c.Request().Context(), db, id, true)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, out)
	}, enforce)
}
