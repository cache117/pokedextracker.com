package catalog

import (
	"testing"
	"time"
)

func TestSourceEligibilityAtClosure(t *testing.T) {
	cutoff := time.Date(2027, 2, 26, 3, 0, 0, 0, time.UTC)
	s := Settings{HomeMode: "current", Routes: []Route{{ID: "bank", ReachesHome: true, Enabled: true, EndsAt: &cutoff}}, Games: []Game{{ID: "sun", Included: true, RouteID: "bank", Groups: []string{"3DS"}}}}
	if !Allowed(s, "sun", cutoff.Add(-time.Second)) || Allowed(s, "sun", cutoff) {
		t.Fatal("route must expire at the exact closure instant")
	}
	s.HomeMode = "all"
	if !Allowed(s, "sun", cutoff) {
		t.Fatal("historical sources must remain available in all mode")
	}
	s.HiddenGroups = []string{"3DS"}
	if Allowed(s, "sun", cutoff) {
		t.Fatal("explicitly hidden groups must stay hidden")
	}
}
func TestOwnedAndUnknownRoutes(t *testing.T) {
	s := Settings{HomeMode: "owned", Routes: []Route{{ID: "home", ReachesHome: true, Enabled: true}}, Games: []Game{{ID: "switch", Included: true, RouteID: "home"}}}
	if Allowed(s, "switch", time.Now()) {
		t.Fatal("unowned game should be hidden")
	}
	s.Games[0].Owned = true
	if !Allowed(s, "switch", time.Now()) {
		t.Fatal("owned supported game should be shown")
	}
	s.Games[0].RouteID = "unknown"
	if Allowed(s, "switch", time.Now()) {
		t.Fatal("unknown route must not be treated as HOME compatible")
	}
}

func TestExplicitSourceFiltersAcrossModes(t *testing.T) {
	now := time.Now()
	for _, mode := range []string{"all", "current", "owned"} {
		t.Run(mode, func(t *testing.T) {
			s := Settings{Version: 1, HomeMode: mode,
				Routes: []Route{{ID: "bank", Name: "Bank", ReachesHome: true, Enabled: true}, {ID: "home", Name: "HOME", ReachesHome: true, Enabled: true}},
				Games:  []Game{{ID: "sun", Name: "Sun", Generation: 7, Platform: "3DS", Included: true, Owned: true, RouteID: "bank"}, {ID: "violet", Name: "Violet", Generation: 9, Platform: "Switch", Included: true, Owned: true, RouteID: "home"}},
			}
			if err := Validate(s, nil); err != nil {
				t.Fatal(err)
			}
			if !Allowed(s, "sun", now) || !Allowed(s, "violet", now) {
				t.Fatal("older configuration without new filters must retain its behavior")
			}
			s.HiddenRoutes = []string{"bank"}
			if Allowed(s, "sun", now) || !Allowed(s, "violet", now) {
				t.Fatal("hiding Bank must leave direct HOME sources available in every mode")
			}
			s.HiddenRoutes = nil
			s.HiddenPlatforms = []string{"3DS"}
			if Allowed(s, "sun", now) || !Allowed(s, "violet", now) {
				t.Fatal("hidden platforms must be respected independently of transfer routes")
			}
			// A newly added source inherits the existing platform rule.
			s.Games = append(s.Games, Game{ID: "future_3ds", Platform: "3DS", Included: true, Owned: true, RouteID: "home"})
			if Allowed(s, "future_3ds", now) {
				t.Fatal("new games on a hidden platform must also stay hidden")
			}
			s.HiddenPlatforms = nil
			if !Allowed(s, "sun", now) {
				t.Fatal("clearing a filter must restore the source without editing the game")
			}
		})
	}
}

func TestValidateHiddenSourceRules(t *testing.T) {
	s := Settings{Version: 1, HomeMode: "all", Routes: []Route{{ID: "bank", Name: "Bank"}}}
	s.HiddenRoutes = []string{"missing"}
	if Validate(s, nil) == nil {
		t.Fatal("unknown hidden routes should be rejected")
	}
	s.HiddenRoutes = []string{"bank"}
	s.HiddenPlatforms = []string{""}
	if Validate(s, nil) == nil {
		t.Fatal("blank hidden platforms should be rejected")
	}
	s.HiddenPlatforms = []string{"3DS"}
	if err := Validate(s, nil); err != nil {
		t.Fatal(err)
	}
}
