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
