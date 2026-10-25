package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPersonalLocationData(t *testing.T) {
	rows, err := parsePersonalLocations(fireRedLeafGreenLocations)
	require.NoError(t, err)
	byID := map[int]personalLocation{}
	for _, row := range rows {
		byID[row.NationalID] = row
	}
	require.Contains(t, strings.Join(byID[1].FireRed, " "), "Pallet Town")
	require.NotEqual(t, byID[23].FireRed, byID[23].LeafGreen, "Ekans is version-exclusive")
	require.NotEqual(t, byID[37].FireRed, byID[37].LeafGreen, "Vulpix is version-exclusive")
	for _, id := range []int{249, 250} {
		require.Contains(t, strings.Join(byID[id].FireRed, " "), "MysticTicket")
		require.Contains(t, strings.Join(byID[id].LeafGreen, " "), "MysticTicket")
	}
	// The Switch releases award both tickets in-game; GBA event requirements
	// must not imply those encounters are unavailable on Switch.
	for _, id := range []int{249, 250, 386} {
		for _, values := range [][]string{byID[id].FireRed, byID[id].LeafGreen} {
			require.Contains(t, values[0], "Switch")
			require.Contains(t, values[0], "automatically")
			require.Contains(t, values[0], "Elite Four")
			require.Contains(t, values[0], "Hall of Fame")
			require.Contains(t, strings.Join(values, " "), "Original GBA")
		}
		require.Contains(t, byID[id].AdditionalSources, "https://www.serebii.net/fireredleafgreen/nintendoswitch.shtml")
	}
	require.Contains(t, strings.Join(byID[386].FireRed, " "), "AuroraTicket")
	require.Contains(t, strings.Join(byID[386].LeafGreen, " "), "AuroraTicket")
	require.Contains(t, strings.Join(byID[151].FireRed, " "), "Old Sea Map")
	require.Contains(t, strings.Join(byID[251].FireRed, " "), "event")
	require.Contains(t, strings.Join(byID[385].FireRed, " "), "Bonus Disc")
}

func TestRejectIncompletePersonalLocations(t *testing.T) {
	rows, err := parsePersonalLocations(fireRedLeafGreenLocations)
	require.NoError(t, err)
	for _, mutate := range []func([]personalLocation) []personalLocation{
		func(rows []personalLocation) []personalLocation { return rows[:385] },
		func(rows []personalLocation) []personalLocation { rows[1].NationalID = rows[0].NationalID; return rows },
		func(rows []personalLocation) []personalLocation { rows[0].LeafGreen = nil; return rows },
		func(rows []personalLocation) []personalLocation { rows[0].FireRed = []string{" "}; return rows },
		func(rows []personalLocation) []personalLocation { rows[0].SourceURL = ""; return rows },
	} {
		copyRows := append([]personalLocation(nil), rows...)
		raw, marshalErr := json.Marshal(mutate(copyRows))
		require.NoError(t, marshalErr)
		_, parseErr := parsePersonalLocations(raw)
		require.Error(t, parseErr)
	}
}

func TestSwitchAcquisitionRoutes(t *testing.T) {
	rows, err := parsePersonalLocations(fireRedLeafGreenLocations)
	require.NoError(t, err)
	byID := map[int]personalLocation{}
	for _, row := range rows {
		byID[row.NationalID] = row
	}
	// External-game methods, unreleased Altering Cave species, and evolutions
	// of unavailable parents must not be presented as current Switch routes.
	for _, id := range []int{151, 152, 179, 180, 181, 196, 197, 204, 205, 251, 252, 350, 385} {
		for _, values := range [][]string{byID[id].FireRed, byID[id].LeafGreen} {
			require.Contains(t, values[0], "Not currently obtainable")
			require.Contains(t, values[0], "HOME cannot send")
		}
	}
	require.Contains(t, strings.Join(byID[251].FireRed, " "), "mobile Pokemon HOME app")
	// Preserve legitimate local trades and avoid incorrectly making Hitmontop
	// unavailable just because the old source listed only Colosseum.
	require.Contains(t, byID[37].FireRed[0], "trade locally")
	for _, values := range [][]string{byID[237].FireRed, byID[237].LeafGreen} {
		require.Contains(t, values[0], "Tyrogue")
		require.Contains(t, values[0], "Attack equal to Defense")
	}
	require.Contains(t, strings.Join(byID[243].FireRed, " "), "Roar disappearance bug is fixed")
	require.Contains(t, byID[245].FireRed[0], "Sevii Islands Team Rocket")
	require.Contains(t, byID[360].FireRed[0], "Lax Incense")
}
