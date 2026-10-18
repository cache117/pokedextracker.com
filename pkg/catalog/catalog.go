// Package catalog holds editable personal source rules, independently of imports.
package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/go-pg/pg/v10"
)

type Route struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	ReachesHome bool       `json:"reaches_home"`
	Enabled     bool       `json:"enabled"`
	OneWay      bool       `json:"one_way"`
	StartsAt    *time.Time `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at"`
}
type Game struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Generation int      `json:"generation"`
	Platform   string   `json:"platform"`
	Groups     []string `json:"groups"`
	RouteID    string   `json:"route_id"`
	Included   bool     `json:"included"`
	Owned      bool     `json:"owned"`
}
type Settings struct {
	Version      int      `json:"version"`
	HomeMode     string   `json:"home_mode"`
	HiddenGroups []string `json:"hidden_groups"`
	Routes       []Route  `json:"routes"`
	Games        []Game   `json:"games"`
}
type Override struct {
	PokemonID int      `json:"pokemon_id"`
	GameID    string   `json:"game_id"`
	Values    []string `json:"values" pg:",array"`
}
type Export struct {
	Revision  int        `json:"revision"`
	Settings  Settings   `json:"settings"`
	Overrides []Override `json:"overrides"`
}

func Load(ctx context.Context, db *pg.DB) (Export, error)         { return load(ctx, db, true) }
func LoadSettings(ctx context.Context, db *pg.DB) (Export, error) { return load(ctx, db, false) }
func load(ctx context.Context, db *pg.DB, includeOverrides bool) (Export, error) {
	var out Export
	var raw string
	_, err := db.QueryOneContext(ctx, pg.Scan(&out.Revision, &raw), `SELECT revision,document::text FROM personal_catalog_settings WHERE id=1`)
	if err != nil {
		return out, err
	}
	if err = json.Unmarshal([]byte(raw), &out.Settings); err != nil {
		return out, err
	}
	if out.Settings.Version == 0 {
		cutoff := time.Date(2027, 2, 26, 3, 0, 0, 0, time.UTC)
		out.Settings = Settings{Version: 1, HomeMode: "all", HiddenGroups: []string{}, Games: []Game{}, Routes: []Route{
			{ID: "home", Name: "Direct to HOME", ReachesHome: true, Enabled: true},
			{ID: "home_storage", Name: "Already in Pokémon HOME", ReachesHome: true, Enabled: true},
			{ID: "bank", Name: "Through Pokémon Bank", ReachesHome: true, Enabled: true, OneWay: true, EndsAt: &cutoff},
			{ID: "unknown", Name: "Unknown / no HOME route"},
		}}
	}
	var existing []struct {
		ID         string
		Name       string
		Generation int
	}
	_, err = db.QueryContext(ctx, &existing, `SELECT g.id,g.name,gf.generation FROM games g JOIN game_families gf ON gf.id=g.game_family_id ORDER BY g."order" DESC`)
	if err != nil {
		return out, err
	}
	present := map[string]bool{}
	for _, g := range out.Settings.Games {
		present[g.ID] = true
	}
	for _, g := range existing {
		if present[g.ID] {
			continue
		}
		platform, route := "3DS", "bank"
		if g.Generation < 6 {
			route = "unknown"
			platform = map[int]string{1: "Game Boy", 2: "Game Boy Color", 3: "Game Boy Advance", 4: "DS", 5: "DS"}[g.Generation]
		}
		if g.Generation >= 8 {
			platform, route = "Switch", "home"
		}
		if g.ID == "home" {
			platform, route = "Service", "home_storage"
		}
		out.Settings.Games = append(out.Settings.Games, Game{ID: g.ID, Name: g.Name, Generation: g.Generation, Platform: platform, Groups: []string{platform}, RouteID: route, Included: true})
	}
	out.Overrides = []Override{}
	if includeOverrides {
		_, err = db.QueryContext(ctx, &out.Overrides, `SELECT pokemon_id,game_id,values FROM personal_location_overrides ORDER BY pokemon_id,game_id`)
	}
	return out, err
}

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,59}$`)

func Validate(s Settings, overrides []Override) error {
	if s.Version != 1 {
		return fmt.Errorf("unsupported catalogue version")
	}
	if s.HomeMode != "all" && s.HomeMode != "current" && s.HomeMode != "owned" {
		return fmt.Errorf("choose all, current, or owned HOME sources")
	}
	if len(s.Games) > 1000 || len(s.Routes) > 100 || len(overrides) > 50000 {
		return fmt.Errorf("configuration is too large")
	}
	routes := map[string]bool{}
	games := map[string]bool{}
	for _, r := range s.Routes {
		if !idPattern.MatchString(r.ID) || routes[r.ID] || len(r.Name) == 0 || len(r.Name) > 200 {
			return fmt.Errorf("invalid or duplicate transfer route")
		}
		if r.StartsAt != nil && r.EndsAt != nil && !r.StartsAt.Before(*r.EndsAt) {
			return fmt.Errorf("route start must be before its end")
		}
		routes[r.ID] = true
	}
	for _, g := range s.Games {
		if !idPattern.MatchString(g.ID) || games[g.ID] || len(g.Name) == 0 || len(g.Name) > 200 || len(g.Platform) > 100 || g.Generation < 1 || g.Generation > 100 || len(g.Groups) > 30 {
			return fmt.Errorf("invalid or duplicate game")
		}
		if !routes[g.RouteID] {
			return fmt.Errorf("unknown route for %s", g.Name)
		}
		for _, tag := range g.Groups {
			if len(tag) == 0 || len(tag) > 100 {
				return fmt.Errorf("invalid game group")
			}
		}
		games[g.ID] = true
	}
	if len(s.HiddenGroups) > 100 {
		return fmt.Errorf("too many hidden groups")
	}
	for _, group := range s.HiddenGroups {
		if len(group) > 100 {
			return fmt.Errorf("group name too long")
		}
	}
	seen := map[string]bool{}
	for _, o := range overrides {
		key := fmt.Sprintf("%d:%s", o.PokemonID, o.GameID)
		if o.PokemonID <= 0 || !games[o.GameID] || len(o.Values) > 100 || seen[key] {
			return fmt.Errorf("invalid or duplicate encounter override")
		}
		seen[key] = true
		for _, v := range o.Values {
			if len(v) == 0 || len(v) > 2000 {
				return fmt.Errorf("encounter lines must contain 1–2000 characters")
			}
		}
	}
	return nil
}

// Allowed checks the route at the current instant; no scheduled task is required.
func Allowed(s Settings, gameID string, now time.Time) bool {
	for _, g := range s.Games {
		if g.ID != gameID {
			continue
		}
		if !g.Included {
			return false
		}
		for _, group := range g.Groups {
			for _, hidden := range s.HiddenGroups {
				if group == hidden {
					return false
				}
			}
		}
		if s.HomeMode == "all" {
			return true
		}
		if s.HomeMode == "owned" && !g.Owned {
			return false
		}
		for _, r := range s.Routes {
			if r.ID == g.RouteID {
				return r.ReachesHome && r.Enabled && (r.StartsAt == nil || !now.Before(*r.StartsAt)) && (r.EndsAt == nil || now.Before(*r.EndsAt))
			}
		}
		return false
	}
	return false
}
