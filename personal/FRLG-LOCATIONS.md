# FireRed / LeafGreen locations

The personal reference importer embeds
`cmd/personal-seed-import/fire_red_leaf_green_locations.json`. The reviewed file
contains National Dex numbers 1–386 with separate `fire_red` and `leaf_green`
lists. These become 772 ordinary upstream `locations` rows, linked to the same
canonical Pokémon used by both FR/LG dex templates. The original `seeds/*.csv`
files are unchanged.

This is acquisition guidance for the original Generation III GBA games:
wild locations, starters, gifts, fossils, breeding, evolutions, NPC trades,
trades from other Generation III games, and historical event prerequisites.
Some methods require another game, compatible hardware, or a previously
distributed event item/Pokémon. Listing all 386 species does not imply all of
them can be caught on a fresh FR/LG save. Percentages and encounter levels are
not included. Lists are sourced routes, not every possible indirect
breeding/evolution/trading combination. They do not apply Switch rerelease
changes to the GBA games or populate the separate Switch source IDs.

## Sources and review

- Each species links to `https://www.serebii.net/pokedex-rs/NNN.shtml` and records
  the retrieval timestamp and SHA-256 of the downloaded reference page. Only
  location/acquisition facts are checked into Git; full pages remain in the
  task's private work cache.
- Historical GBA MysticTicket/AuroraTicket requirements:
  https://www.serebii.net/red_green/tickets.shtml.
- Hoenn/GameCube trading prerequisites:
  https://www.serebii.net/fireredleafgreen/compatibility.shtml.
- National Dex restrictions:
  https://www.serebii.net/fireredleafgreen/evolution.shtml.
- Old Sea Map context:
  https://www.serebii.net/itemdex/oldseamap.shtml.
- Serebii's per-species pages list eight unreleased-event species at Altering
  Cave without a warning. These entries are corrected to trade/import routes,
  with the unreleased encounter explicitly identified. Review references:
  https://www.serebii.net/fireredleafgreen/alteringcave.shtml and
  https://bulbapedia.bulbagarden.net/wiki/Changing_Cave.
- Trade-evolution item names agree with the upstream evolution snapshot.
  Deoxys retains the single species entry used by upstream; location text notes
  Attack Forme in FireRed and Defense Forme in LeafGreen.

The initial snapshot was collected on 2026-10-21. Corrections are intentionally
reviewed rather than automatically downloading and overwriting data at startup.
Shared evolution panels are not a complete independently audited historical
evolution database.

## Editing and importing

For a personal correction, use Manage Sources to edit that Pokémon's encounter
text. It lives in the owner override layer and takes precedence over imports.
For a correction to share through the fork, edit the appropriate game list in
the JSON file, keep its original provenance, and add review URLs to
`additional_sources` when introducing new facts. Record substantial edits in
the Git commit message. Do not edit production seed CSVs.

After a JSON edit, rebuild `cmd/personal-seed-import` using Go 1.20.5, or run
`personal/Build-Local.ps1` to build all tools. Run
`personal/Import-UpstreamData.ps1` to back up and refresh the local database.
Neither a website rebuild nor server restart is required for location changes.
The running website reads the database; reload an already-open detail panel.

The existing transaction imports upstream first, then the personal templates
and location dataset. Personal FR/LG baseline entries therefore take precedence
over any overlapping upstream seed rows; owner overrides take precedence over
both. If upstream later publishes official FR/LG data, review and reconcile
the overlap before retiring the personal addition. Users, dexes, captures,
source settings and overrides are fingerprinted and must remain unchanged
through the import, or the entire transaction rolls back. Missing, duplicate,
blank or mismatched Pokémon records also reject the import.

## Verification

`go test ./cmd/personal-seed-import` checks dataset completeness, version
differences, event requirements and rejection of malformed/incomplete files.

Run `personal/Test-FireRedLeafGreenLocations.ps1` against a separately copied
trial/test database and loopback server. It checks all 772 baselines against
the checked-in file, representative Regional/National API responses, repeat
imports and owner override preservation, then restores the trial settings.
It refuses the normal personal database and port. The optional `-ImporterExe`
argument selects a newly built importer for pre-release testing.

Use `personal/Test-FireRedLeafGreen.ps1` and `personal/Test-Local.ps1` on that
same isolated server for saved dex/catch and existing catalogue regressions.
