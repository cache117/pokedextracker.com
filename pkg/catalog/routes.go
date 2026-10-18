package catalog

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-pg/pg/v10"
	"github.com/labstack/echo/v4"
	"github.com/pokedextracker/pokedextracker.com/pkg/auth"
	"github.com/pokedextracker/pokedextracker.com/pkg/errcodes"
)

func RegisterRoutes(e *echo.Echo, db *pg.DB, enforce, optional echo.MiddlewareFunc) {
	owner := func(c echo.Context) bool {
		s := auth.FromContext(c)
		if s == nil {
			return false
		}
		var id int
		_, err := db.QueryOneContext(c.Request().Context(), pg.Scan(&id), `SELECT min(id) FROM users`)
		return err == nil && id == s.ID
	}
	e.GET("/personal/catalog", func(c echo.Context) error {
		out, err := Load(c.Request().Context(), db)
		if err != nil {
			return err
		}
		return c.JSON(200, struct {
			Export
			IsOwner bool `json:"is_owner"`
		}{out, owner(c)})
	}, optional)
	e.GET("/personal/catalog/pokemon", func(c echo.Context) error {
		var rows []struct {
			ID         int     `json:"id"`
			Name       string  `json:"name"`
			NationalID int     `json:"national_id"`
			Form       *string `json:"form"`
		}
		_, err := db.QueryContext(c.Request().Context(), &rows, `SELECT id,name,national_id,form FROM pokemon ORDER BY national_id,form NULLS FIRST`)
		if err != nil {
			return err
		}
		return c.JSON(200, rows)
	}, optional)
	e.GET("/personal/catalog/encounters/:id", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return errcodes.ValidationError("invalid Pokemon")
		}
		var rows []struct {
			GameID string   `json:"game_id"`
			Values []string `json:"values" pg:",array"`
		}
		_, err = db.QueryContext(c.Request().Context(), &rows, `SELECT game_id,values FROM locations WHERE pokemon_id=? ORDER BY game_id`, id)
		if err != nil {
			return err
		}
		if rows == nil {
			rows = make([]struct {
				GameID string   `json:"game_id"`
				Values []string `json:"values" pg:",array"`
			}, 0)
		}
		return c.JSON(200, rows)
	}, optional)
	e.POST("/personal/catalog", func(c echo.Context) error {
		if !owner(c) {
			return errcodes.Forbidden("editing this personal catalogue")
		}
		var in Export
		dec := json.NewDecoder(http.MaxBytesReader(c.Response(), c.Request().Body, 8<<20))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			return errcodes.ValidationError("invalid configuration: " + err.Error())
		}
		if err := dec.Decode(&struct{}{}); err != io.EOF {
			return errcodes.ValidationError("unexpected extra configuration")
		}
		if err := Validate(in.Settings, in.Overrides); err != nil {
			return errcodes.ValidationError(err.Error())
		}
		err := db.RunInTransaction(c.Request().Context(), func(tx *pg.Tx) error {
			var revision int
			if _, err := tx.QueryOne(pg.Scan(&revision), `SELECT revision FROM personal_catalog_settings WHERE id=1 FOR UPDATE`); err != nil {
				return err
			}
			if revision != in.Revision {
				return errcodes.ValidationError("The catalogue changed in another window. Reload it before saving.")
			}
			// Imported games can be renamed but cannot be removed by configuration import.
			var existing []struct{ ID string }
			if _, err := tx.Query(&existing, `SELECT id FROM games`); err != nil {
				return err
			}
			present := map[string]bool{}
			for _, g := range in.Settings.Games {
				present[g.ID] = true
			}
			for _, g := range existing {
				if !present[g.ID] {
					return errcodes.ValidationError("Keep existing games in the catalogue; uncheck Include to hide them.")
				}
			}
			for _, g := range in.Settings.Games {
				var n int
				if _, err := tx.QueryOne(pg.Scan(&n), `SELECT count(*) FROM games WHERE id=?`, g.ID); err != nil {
					return err
				}
				if n == 0 {
					familyID := "personal_" + g.ID
					if _, err := tx.Exec(`INSERT INTO game_families(id,generation,regional_total,national_total,"order",published,regional_support,national_support) VALUES(?,?,0,0,(SELECT COALESCE(max("order"),0)+1 FROM game_families),false,false,false)`, familyID, g.Generation); err != nil {
						return err
					}
					if _, err := tx.Exec(`INSERT INTO games(id,name,game_family_id,"order") VALUES(?,?,?,(SELECT COALESCE(max("order"),0)+1 FROM games))`, g.ID, g.Name, familyID); err != nil {
						return err
					}
				} else {
					if _, err := tx.Exec(`UPDATE games SET name=? WHERE id=?`, g.Name, g.ID); err != nil {
						return err
					}
				}
			}
			for _, o := range in.Overrides {
				var n int
				if _, err := tx.QueryOne(pg.Scan(&n), `SELECT count(*) FROM pokemon WHERE id=?`, o.PokemonID); err != nil {
					return err
				}
				if n == 0 {
					return errcodes.ValidationError("An encounter override refers to an unknown Pokémon.")
				}
			}
			// Replace only the editable overlay. Imported locations and captures stay intact.
			if _, err := tx.Exec(`DELETE FROM personal_location_overrides`); err != nil {
				return err
			}
			for _, o := range in.Overrides {
				if _, err := tx.Exec(`INSERT INTO personal_location_overrides(pokemon_id,game_id,values) VALUES(?,?,?)`, o.PokemonID, o.GameID, pg.Array(o.Values)); err != nil {
					return err
				}
			}
			raw, err := json.Marshal(in.Settings)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`UPDATE personal_catalog_settings SET document=?::jsonb,revision=revision+1 WHERE id=1`, string(raw))
			return err
		})
		if err != nil {
			return err
		}
		out, err := Load(context.Background(), db)
		if err != nil {
			return err
		}
		return c.JSON(200, out)
	}, enforce)
}
