package catalog

import "testing"

func TestCollectionFilterShape(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		filters []CollectionFilter
		invalid bool
	}{
		{name: "older exports", filters: nil},
		{name: "all obtainable", filters: []CollectionFilter{{DexID: 1, HideUnavailable: true, UnavailablePokemonIDs: []int{}}}},
		{name: "duplicate collection", filters: []CollectionFilter{{DexID: 1}, {DexID: 1}}, invalid: true},
		{name: "duplicate species", filters: []CollectionFilter{{DexID: 1, UnavailablePokemonIDs: []int{151, 151}}}, invalid: true},
		{name: "invalid species", filters: []CollectionFilter{{DexID: 1, UnavailablePokemonIDs: []int{-1}}}, invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateCollectionFilterShape(tt.filters)
			if (err != nil) != tt.invalid {
				t.Fatalf("validation result: %v", err)
			}
		})
	}
}
