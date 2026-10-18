# Personal Windows fork

This checkout is a local Git fork of https://github.com/pokedextracker/pokedextracker.com.
The `upstream` remote points to that repository; the `personal` branch carries the
personal changes. Upstream history and its README, licenses, CSV seeds, dependency
locks, and normal directory layout are retained. No GitHub repository was created
or published for this local fork.

## Upstream baseline

2026-10-17: `7c32e95eada504e9b73fef700f44d234265cb790`.
The upstream repository now includes both the React frontend and Go API.
The old separate API repository is no longer the update source for this fork.

## Personal changes

- Loopback-only Windows launch/stop scripts and a same-origin static/API server
  in `personal/`. The surrounding portable installation supplies `runtime/`,
  `data/`, `logs/`, and `private.local.json`; those are outside this Git checkout.
- Local configuration uses the installation's database password and JWT secret;
  local frontend analytics/error reporting are disabled.
- Owner-editable source catalogue, platforms/groups, HOME/Bank routes, ownership,
  source filtering, and encounter overrides. Imported data remains separate.
- A transactional `cmd/personal-seed-import` merges the upstream reference CSVs
  into a populated personal database. Upstream's normal dev seeder replaces its
  reference tables; this personal importer retains custom source games and checks
  that users, dexes, captures, settings, and overrides are unchanged.
- Historical ten-record download/import tools and the original public-profile
  bootstrap importer are retained for provenance and recovery; full reference
  refreshes now use `seeds/`, without requests to the production Pokémon API.

## Run on this PC

Use the surrounding installation's `Open Pokedextracker.cmd` and
`Stop Pokedextracker.cmd`. Login details are in `Login details.txt`, outside this
checkout. Site: http://localhost:9898. Source editor: http://localhost:9898/manage.
Existing caught progress remains in the installation's `data/postgres` directory.

The upstream `cmd/dev` launcher uses Unix process groups and file locks and does
not compile natively on Windows. This fork keeps that upstream code unchanged;
use the scripts in `personal/` for the Windows installation.

## Build and update

With Go, Node.js and Yarn 1 installed, install the pinned frontend dependencies
with `yarn install --frozen-lockfile`, then run `personal/Build-Local.ps1` from
PowerShell. It builds the frontend into upstream's `build/web` directory and the
portable API/importer/migration executables into the surrounding `runtime/`.
Stop the app, replace `runtime/api.exe` with `runtime/api-next.exe`, then start it.
Build outputs, dependencies, account credentials, and personal database contents
must not be committed or pushed.

To update, fetch `upstream`, review its changes, and merge `upstream/master` into
`personal`. Resolve conflicts while keeping the personal extensions, then build
and verify against a copy of the database. Run
`personal/Import-UpstreamData.ps1` against the personal installation only after
that verification. It backs up the database before merging all eight seed files
in a transaction. The importer refuses identity conflicts and changes that would
hide caught Pokémon from upstream dex templates. Do not run `mise db:reset`
against the personal database. Avoid the upstream replacement seeder for this
installation; it does not know about personal source games or overrides.

The seed CSVs stay exactly as upstream published them. Put personal encounter
changes in the source editor, then export the configuration for portability.
The source filters affect HOME encounter suggestions; regional dex membership
continues to follow upstream templates. Newly available templates do not create
collections automatically: choose Add a dex on your profile when you want one.

## Verified baseline

- 26 upstream game families, 44 games, 1,454 Pokémon, 25 dex types,
  13,424 dex memberships, 684 box labels, 17,514 locations, 716 evolutions.
- Existing personal accounts, all 24 saved dexes, captures, catalogue settings,
  custom games, and encounter overrides preserved.
- Z-A regional checklist: 237 entries; Mega Dimension Full Lumiose: 380;
  Hyperspace: 148. Counts include the forms used by upstream.
- Trial creation/capture of each Z-A type and a repeat import with a personal
  override passed. TypeScript/production build and catalogue/config Go tests pass.
- The untouched upstream development-tool tests cannot compile on Windows;
  the personal Windows launch path is verified separately.

Full pre-upgrade database backup: `../backups/before-upstream-fork.dump`.
The old source checkouts are retained in `../backups/legacy-api` and
`../backups/legacy-web` for recovery; `source/` is now the active development repo.
