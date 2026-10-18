package pokemon

import (
	"context"
	"time"

	"github.com/pokedextracker/pokedextracker.com/pkg/catalog"
	"github.com/pokedextracker/pokedextracker.com/pkg/games"
)

func (svc *Service) applyPersonalOverrides(ctx context.Context, p *Pokemon) (catalog.Settings, error) {
	cfg, err := catalog.LoadSettings(ctx, svc.db)
	if err != nil {
		return cfg.Settings, err
	}
	var overrides []catalog.Override
	if _, err = svc.db.QueryContext(ctx, &overrides, `SELECT pokemon_id,game_id,values FROM personal_location_overrides WHERE pokemon_id=?`, p.ID); err != nil {
		return cfg.Settings, err
	}
	for _, o := range overrides {
		next := make([]*Location, 0, len(p.Locations)+1)
		for _, l := range p.Locations {
			if l.Game.ID != o.GameID {
				next = append(next, l)
			}
		}
		if len(o.Values) > 0 {
			g := &games.Game{}
			if err = svc.db.ModelContext(ctx, g).Relation("GameFamily").Where("g.id = ?", o.GameID).Select(); err != nil {
				return cfg.Settings, err
			}
			next = append(next, &Location{GameID: g.ID, Game: g, PokemonID: p.ID, Values: o.Values})
		}
		p.Locations = next
	}
	return cfg.Settings, nil
}
func filterPersonalLocations(locations []*Location, cfg catalog.Settings) []*Location {
	out := make([]*Location, 0, len(locations))
	for _, l := range locations {
		if catalog.Allowed(cfg, l.Game.ID, time.Now()) {
			out = append(out, l)
		}
	}
	return out
}
