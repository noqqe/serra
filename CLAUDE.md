# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Serra is a personal CLI (+ tiny web UI) for tracking a *Magic: The Gathering*
card collection. It stores ownership data in MongoDB, enriches it with
Scryfall data/pricing, and exposes commands to add/remove cards, check
collection value, and see price trends. See `readme.md` for user-facing docs
and `mongodb_queries.md` for schema/query cheatsheet.

## Commands

    task build              # the real build (default task) - see Taskfile.yaml
    go build ./cmd/serra    # same binary, but Version stays "unknown"
    go build ./...          # compile-check everything
    go vet ./...            # static checks - not clean, see below
    ./serra <command>       # run it, e.g. ./serra add usg/17

`go build .` does **not** work - the repo root contains no Go files. `task
build` is what to use: it injects `-X
github.com/noqqe/serra/pkg/serra.Version=$(git describe --tags)`, and that
`Version` string feeds `--version`, the web UI footer *and* the `User-Agent`
sent to Scryfall (`queryScryfall()`), so a hand-rolled `go build` produces a
binary that identifies itself as `Serra/unknown`.

`go vet ./...` is **not** clean on a pristine checkout: it emits ~51 "struct
literal uses unkeyed fields" warnings for the unkeyed `bson.E{...}` /
`primitive.E{...}` literals used throughout the query builders. That is the
codebase's established style - compare against that baseline rather than
assuming a change introduced them.

There are no automated tests in this repo (`*_test.go` files do not exist).
There is no linter config beyond `go vet`.

Run the binary **from the repo root**. Three runtime paths are relative to
the working directory: `sounds/*.mp3` (`sound.go` `panic()`s if a file is
missing, and it is called as `go playSound...()`, so the panic takes the
whole process down), plus `templates/*.gohtml` and `./assets` (`web.go`).
`colorizeValue()` fires `go playSoundCash()` for any card worth more than 1,
so even read-only commands touch `./sounds`.

`.tool-versions` pins `golang 1.21.3` and is stale - `go.mod` requires go
1.26.1 and the Dockerfile builds on `golang:1.26-alpine`.

Running against a real database requires:

    export MONGODB_URI='mongodb://localhost:27017'
    export SERRA_CURRENCY=USD   # or EUR

A disposable local MongoDB: `docker run -d -p 27017:27017 --name mongo mongo:8`

Release process (`task release`) tags, pushes, runs `goreleaser release
--clean`, and builds/pushes a Docker image — not something to run without the
user explicitly asking.

## Version control

This is a **colocated Jujutsu repo**: `.jj/` uses a git backend pointing at
`.git`, so git HEAD is normally detached and "the current branch" is not a
meaningful question. Check `jj st` / `jj log` before doing anything
VCS-related, and do not "fix" the detached HEAD.

## Architecture

Everything lives in a single Go package, `pkg/serra`, with `cmd/serra/serra.go`
as a one-line entrypoint calling `serra.Execute()`. Each file in `pkg/serra`
is a self-contained vertical slice: a cobra command (`*Cmd` var + `init()`
registering flags/subcommand on `rootCmd` in `root.go`) plus the logic it
needs. Cobra flag variables (`set`, `count`, `foil`, `language`, ...) are
declared once as package-level vars in `root.go` and reused across commands.

Those globals are a sharp edge: cobra/pflag assigns a flag's default to its
variable at *registration* time (i.e. in `init()`, for every command), and
some query helpers read them directly instead of taking them as parameters
— `Cards()` has a dozen parameters but still reads `artist`, `cmc`, `color`
and `count` (`--min-count`) off the package level. Calling `Cards()` from a
non-CLI context (`web.go`) therefore inherits whatever those defaults are.

### Data model: Scryfall cache vs. ownership

The core architectural split (post schema v1, see "Upgrade Notes" in
`readme.md` for the pre-split legacy shape) is between cached Scryfall data
and personal ownership data, stored in separate Mongo collections:

- **`cards`** (`storage_cards.go`, type `Card`) — pure Scryfall data for every
  printing that has ever been fetched (owned or not), keyed by Scryfall ID.
  Every upsert appends a snapshot to `price_history` via an aggregation
  pipeline (`cardUpsertPipeline`) instead of a plain `$set`, so history is
  preserved without needing to read-then-write.
- **`inventory`** (`storage_inventory.go`, type `InventoryEntry`) — ownership
  data: one document per unique `card_id`+`finish`+`language`+`condition`
  combination, with its own `count` and `value_history`. The document ID is
  a deterministic composite key (`inventoryID()`), which is what makes
  add/remove a single atomic `$inc` rather than a read-modify-write.
- **`sets`** (`storage_sets.go`) — cached Scryfall set metadata plus a
  `serra_prices` value history for the set's total value.
- **`total`** (`storage_total.go`) — single document tracking the whole
  collection's value history over time.
- **`meta`** (`schema.go`) — one document tracking the DB schema version.

`card.go`'s `OwnedCard` type and `OwnedCards()`/`Cards()` functions are the
main join point: they aggregate `inventory` entries per card ID, then join
against `cards` for display/filtering. Most read-facing commands (`card`,
`stats`, `gains`, `web`) go through `Cards()`/`OwnedCards()` rather than
querying `inventory`/`cards` directly.

Finishes (`FinishNonfoil`/`FinishFoil`/`FinishEtched`) and per-finish pricing
are threaded through consistently via `priceEntryForFinish` /
`valueForFinish` — a `PriceEntry` carries all of Scryfall's price fields, and
these helpers narrow it down to the one relevant to a given finish so
`inventory.value_history` only stores what's relevant to that entry.

### Update flow (`update.go`)

`serra update` is the main write-heavy flow tying everything together:
1. Fetch the current set list and the entire Scryfall bulk file
   (`scryfall_bulk.go`, streamed as gzip'd JSONL to avoid loading it all into
   memory at once beyond the final slice).
2. Import every card from the bulk file into `cards` (`importBulkCards`) —
   this is what lets price history be tracked for printings you don't own.
3. For each set, refresh cached data + append value snapshots for every
   inventory entry in that set (`updateCardsOfSet`), then update the set's
   own value history (`updateSet`).
4. Update the whole-collection total (`updateTotal`).

`updateCardsOfSet` also handles Scryfall occasionally re-keying a printing's
ID: if the bulk data's ID differs from the stored `entry.CardID`, the
inventory entry is migrated to a new deterministic ID rather than orphaned.

### Schema migrations (`schema.go`, `migrate.go`)

`schemaMigrations` in `schema.go` is an ordered list of `{Version,
Description, Up}` migrations, applied by `ApplySchemaMigrations()`. Version 1
is the current baseline (the cards/inventory split) and has no `Up` function
of its own. `storageConnect()` calls `CheckSchemaVersion()` on every command
invocation, which only warns/fatals — it never migrates automatically.
`serra migrate` runs the one-time legacy-format migration
(`legacyCard` → split into `cards`+`inventory`) followed by any pending
`schemaMigrations`. Migrations must be idempotent since a partial failure can
be resumed by re-running.

### Other things worth knowing

- `StorageClient` (`storage.go`) wraps `*mongo.Client` so collection-getter
  methods (`getCardsCollection()`, etc.) can be attached to it despite living
  in an external package. Every command connects via `storageConnect()` and
  `defer storageDisconnect(client)`.
- `web.go` serves a minimal Chi-based web UI using the same `Cards()`/
  `OwnedCards()` functions as the CLI, rendering `templates/index.gohtml`.
- `sound.go` plays small audio cues (success/error/cash-register) on
  add/remove/high-value cards — fire-and-forget via `go playSound...()`.
- `Logger()` (`helpers.go`) returns a fresh `charmbracelet/log` logger per
  call; color helpers (`Purple`, `Green`, `Red`, `Yellow`, ...) are
  package-level `fatih/color` funcs used throughout for CLI output.
- `getCurrency()` (`env.go`) reads `SERRA_CURRENCY` and drives which
  Eur/Usd(/Foil/Etched) fields of `PriceEntry` are used everywhere value is
  displayed or compared.
- The Mongo database name is hardcoded to `serra` in every collection getter;
  only the host part of `MONGODB_URI` matters.
- `Card` carries no bson tags apart from `_id` and `price_history`, so its
  Mongo field names are the **lowercased Go field names**, not the Scryfall
  JSON names: `collectornumber`, `typeline`, `oracletext`, `coloridentity`,
  `scryfalluri`. Filters in `card.go`/`web.go` and the queries in
  `mongodb_queries.md` follow that spelling.
- `OwnedCards()` calls `storageConnect()`/`storageDisconnect()` itself, so a
  command that already connected ends up with a second client. It also joins
  in Go rather than in Mongo: it loads *every* inventory entry, aggregates
  per card ID, then fetches the matching `cards` with an `$in` over all owned
  IDs — fine at a personal collection's scale, but it means filtering is
  memory-bound, and `--min-count`/`--foil`/sorting happen after the fetch.
- Storage helpers mostly call `l.Fatalf(...)` on driver errors *and* return
  the error, so a DB failure usually exits the process rather than
  propagating to the cobra `RunE`. Follow that convention when adding to
  `storage_*.go` rather than mixing in a new error style.
