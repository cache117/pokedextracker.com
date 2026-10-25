# FireRed / LeafGreen data proposal

This branch proposes FireRed/LeafGreen dex support using the existing game
family, game records, Pokémon identities and seed schemas. It is based on
upstream `5f7578378475c41f88a3763acd804c842c0d582a` and contains no personal
tracker features, Windows setup or custom importer.

## Changes for review

| File                                    | Proposed change                                                                           |
| --------------------------------------- | ----------------------------------------------------------------------------------------- |
| `seeds/game_families.csv`               | Set the existing `fire_red_leaf_green` family to published. Retain its other fields.      |
| `seeds/dex_types.csv`                   | Add Regional (151 species) and Full National (386 species) templates.                     |
| `seeds/dex_types_pokemon.csv`           | Add 537 memberships using existing canonical Pokémon records, ordered by National number. |
| `seeds/locations.csv`                   | Add 772 rows, with separate guidance for FireRed and LeafGreen.                           |
| `app/components/pages/Tracker/Info.tsx` | Map FR/LG's Serebii detail link to `pokedex-rs`.                                          |

**Dex-type IDs 26 and 27 are provisional.** They are free in this upstream
snapshot; maintainers must assign production IDs and remap both the templates
and their memberships. Both existing `fire_red` and `leaf_green` game records
are reused. No new Pokémon records or evolution-table edits are proposed.

The Regional list includes Mew; the National list includes all 386 Generation
III species, including Mythical Pokémon. These are complete checklists, not a
claim that every entry is currently obtainable on Nintendo Switch.

## Switch and original GBA guidance

The locations distinguish current supported Switch methods from historical GBA
alternatives. Switch post-game tickets provide Lugia, Ho-Oh and Deoxys. Local
wireless FR/LG trades remain valid, including version exclusives and trade
evolutions. Pokémon HOME transfers are outward only; no future compatibility
with Ruby/Sapphire/Emerald or Colosseum/XD is assumed.

The [Switch audit](SWITCH-COMPATIBILITY-AUDIT.md) reviews all 772 entries and
identifies 171 species without a current supported Switch acquisition route.
That count is a documented inference from the referenced encounter tables and
connectivity, not an official Nintendo completion total. The audit also
corrects Hitmontop's missing in-game method and covers roaming-beast differences,
unreleased Altering Cave encounters, Espeon/Umbreon and the HOME Celebi reward.

[Location provenance](location-provenance.csv) records the original per-species
reference URLs, retrieval timestamps and page hashes, plus supplementary sources.
[Per-entry review](switch-availability-audit.csv) records each game/species decision.
The review covers platform compatibility of the sourced acquisition methods;
it is not a new survey of every wild encounter slot, rate or level.

## How maintainers can use this branch

Upstream's [seed documentation](../../seeds/README.md) explains that seeds are
one-way exports from production. These edits provide concrete proposed data
diffs for an issue; merging CSV edits alone does not change the live site.
Maintainers can review/import the agreed data through their normal production
process, assign IDs, and refresh the seed snapshot afterward. Fork pull
requests changing seed CSVs are rejected by the existing seed-data check.
This branch leaves that policy and CI workflow unchanged.

The CSV proposal was validated in PostgreSQL temporary tables: field types,
text-array encoding, Pokémon/game identities, membership counts/order and legacy
versus array location text. The checks ran in a transaction that was rolled back.
The underlying location dataset also passed repeat-import, owner-override and
saved-progress checks in an isolated personal tracker test database. Those
personal test/import tools are not included in this branch.
