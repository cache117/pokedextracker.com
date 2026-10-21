# Personal Windows fork

GitHub: https://github.com/cache117/pokedextracker.com (`origin`).
Original project: https://github.com/pokedextracker/pokedextracker.com (`upstream`).
The default `personal` branch contains our customizations. `master` is a clean
upstream branch. Keep personal database contents and credentials outside Git.

## Upstream baseline

Merged through `e74d9d23cfc9b593bf2c129f37b80e52a83a7a70` (2026-10-19 review).
This includes the unified Go server, upstream browser tests, Node 26.12.0 and
pnpm 12.14.0. Upstream's structure, production seed CSVs, dependency lockfile,
licenses and normal development/deployment code are retained.

## Personal changes

- Native Windows launch/stop scripts in `personal/`, using the surrounding
  installation's portable PostgreSQL and compiled application. The server binds
  only to 127.0.0.1 and serves both the website and `/api` on port 9898, following
  upstream's unified executable. There is no separate Node web/proxy server.
- Local database/session credentials come from `../private.local.json`.
  Analytics and error reporting stay disabled locally, including the frontend
  configuration injected by the unified server.
- Owner-editable source catalogue, platform/group labels, HOME/Bank routes,
  ownership, HOME encounter filtering and personal encounter overrides. Only
  the first local account can change the catalogue; other accounts can browse.
- A transactional `cmd/personal-seed-import` updates reference data without
  replacing users, dexes, captures, personal sources or encounter overrides.
  Upstream's replacement seeder does not know about personal source games.
- The historical 10-record importer and public-profile bootstrap importer are
  retained for provenance/recovery. Reference refreshes now use `seeds/`.

## Running

Use the surrounding `Open Pokedextracker.cmd` and `Stop Pokedextracker.cmd`.
Site: http://localhost:9898/u/cache117. Source editor: http://localhost:9898/manage.
Login details remain in `../Login details.txt`. Progress is in `../data/postgres`.
The compiled app does not require Node to run. Node/pnpm are build tools only.
No Windows service, startup task or public network listener is installed.

Upstream's `cmd/dev` launcher still uses Unix file locks/process groups and does
not compile on Windows. Keep it unchanged for Linux/macOS; use `personal/` on
Windows. GitHub CI runs upstream's Linux checks, including its browser tests.

## Build

Use the tools pinned in `mise.toml`. With Go, Node 26 and pnpm available, install
with `pnpm install --frozen-lockfile`, then run `personal/Build-Local.ps1`.
The script type-checks application and browser tests, builds the frontend into
upstream's `internal/webapp/dist`, copies public assets, and then builds the Go
binary that embeds it. It also builds the data importer and migrations.
Optional `-GoExe`, `-NodeExe` and `-PnpmScript` parameters support portable tools.

Stop the tracker, replace `../runtime/api.exe` with the generated
`../runtime/api-next.exe`, and start again. Runtime files, build assets, database
contents and credentials are not source files to commit or push.

## Updating

Fetch `upstream`, review its changes, merge `upstream/master` into `personal`,
resolve conflicts and run the checks before updating the live installation.
Push the verified personal branch to `origin`. The clean GitHub `master` branch
can independently be fast-forwarded from `upstream/master`. Syncing `master`
alone does not rebuild or upgrade the personal app; never force-push over the
personal commits to synchronize a fork.

For data refreshes, run `personal/Import-UpstreamData.ps1` only after checking a
copy of the database. It pins imports to the merged upstream revision, verifies
that the seed files match that revision, saves a database backup, and imports
all eight reference tables in one transaction. It refuses identity conflicts or
changes that would hide caught Pokémon from upstream templates. The source CSVs
stay exactly as upstream published them; personal encounter edits belong in the
manager's override layer. Do not run `mise db:reset` on the personal database.

## Verification

The fork's CI workflow runs on pushes to `personal` as well as `master`.
It retains upstream's frontend/backend, browser, seed-data and image checks.
Local regression checks include configuration/telemetry isolation and catalogue
routes under `/api` with authentication required.

`personal/Test-Local.ps1 -ApiUrl http://127.0.0.1:9899 -DatabaseName
pokedex_tracker_unified_trial` verifies owner/non-owner/anonymous permissions,
invalid and stale edits, HOME vs regional filtering, encounter overrides,
repeat imports, branched evolutions, Z-A templates and saved progress. Run it
only against a copied trial/test database and a separately launched server; it
refuses the normal personal database/port. Test edits are restored afterward.

The complete upstream snapshot has 1,454 Pokémon records, 25 dex types,
13,424 memberships, 684 box labels, 17,514 locations and 716 evolutions.
Z-A regional has 237 entries, Full Lumiose 380, and Hyperspace 148 (forms included).
Existing personal progress is verified against the pre-upgrade backup.
New templates do not create collections automatically; use Create a New Dex.

## FireRed and LeafGreen (Generation III)

The personal seed importer applies `cmd/personal-seed-import/fire_red_leaf_green.sql`
after the unmodified upstream CSVs, within the same protected transaction. It
publishes upstream's existing `fire_red` and `leaf_green` games and adds normal
`dex_types` / `dex_types_pokemon` rows: Regional (151) and Full National (386).
The website and API use their normal create/edit/track flows without a parallel
custom dex implementation. Serebii links use the Generation III Pokédex.

Membership follows upstream's X/Y and OR/AS base-species policy: `form IS NULL`,
`national_order >= 0`, and original Generation I-III identities. Both lists
include Mythical Pokémon; National also includes species requiring other games
or events. Unown and Deoxys each occupy one species entry. Later evolutions,
regional forms and duplicate encounter identities are excluded. Sources:
https://www.serebii.net/fireredleafgreen/kantopokedex.shtml and
https://www.serebii.net/pokedex-rs/386.shtml.

Personal template IDs 1000001 and 1000002 avoid upstream's generated IDs; the
import refuses collisions or identity changes requiring review. Refreshes
reapply publication and memberships, preserving collections and captures.
If upstream adds official FR/LG templates, review a deliberate migration of
personal collections to those IDs instead of deleting or replacing templates.
No production seed CSVs are edited and no upstream schema changes are needed.

The personal importer also loads `cmd/personal-seed-import/fire_red_leaf_green_locations.json`:
386 species with separate FireRed/LeafGreen acquisition lists (772 game entries).
Each record includes a source URL, retrieval time and raw-source SHA-256; curated
prerequisites include additional references. The dataset was collected from
Serebii's Generation III location tables, with reviewed corrections for event
tickets, trade evolutions and unreleased Altering Cave encounters. See
`personal/FRLG-LOCATIONS.md` for scope, sources, editing and verification.

This data covers the original GBA releases. The separate personal Switch
rerelease sources are unchanged. Evolution panels still use upstream's shared
data, filtered to the selected dex; acquisition entries explicitly identify
trade-evolution items and National Dex restrictions where applicable.

Run `personal/Test-FireRedLeafGreen.ps1` with the same trial-server/database
parameters as `Test-Local.ps1` to check both games, both lists, evolution scope,
capturing and repeat-import preservation. Tests create disposable trial users.

Pre-unified-server backups: `../backups/before-unified-upstream.dump` and
`../backups/before-unified-api.exe`. Earlier source checkouts are retained in
`../backups/legacy-api` and `../backups/legacy-web`. `source/` is the active repo.
