# Changelog

Notable changes per release. For releases before 5.0.0, see the
[GitHub releases page](https://github.com/noqqe/serra/releases).

## 5.0.0

The biggest release so far, and a breaking one: the database schema changed.
**Run `./serra migrate` once after upgrading** — see [UPGRADE.md](UPGRADE.md).

### Breaking

* **Split `cards` and `inventory` into separate collections.** The `cards`
  collection used to hold cached Scryfall data *and* your ownership data
  (count, value history, timestamps) in one document. Ownership now lives in
  its own `inventory` collection, with one entry per
  card/finish/language/condition combination. `serra migrate` converts an
  existing database in place.
* **Database schema versioning.** A `meta` collection records the schema
  version the database is at. Every command warns when the database is behind
  the binary, and refuses to run when it is ahead (i.e. you downgraded).
  `serra migrate --status` shows current vs. expected without changing
  anything.
* **Detection of pre-5.0 databases.** A 4.x database records no schema
  version, so the version check alone cannot catch it and the collection
  would silently read as empty. serra now looks for ownership data left in
  the old format and warns until `serra migrate` has been run. The check only
  runs when the `inventory` collection is empty and is time-bounded, so it
  costs nothing once you are migrated.

### Added

* **Track prices for every card, not just the ones you own.** Each `serra
  update` imports the full Scryfall bulk file and appends a price snapshot to
  every printing. Price history is no longer limited to your collection.
* **`--language` and `--condition` on `add`/`remove`** (defaulting to
  `en`/`nm`), plus **`--etched`** for the etched-foil finish. Copies are
  tracked separately per combination, so a NM English copy and a played
  German one of the same card are distinct inventory entries.
* **`--all` on `tops`/`flops`** to report gains/losses across the entire
  cached card database instead of only what you own — useful for spotting
  movement on cards you are considering buying.
* **`--legal` filter on `card`** to filter by format legality (`standard`,
  `modern`, `commander`, `pauper`, `premodern`, ...).
* **Price history for cards you do not own.** `serra card <set>/<number>`
  now falls back to the cached Scryfall price history when the card is not
  in your inventory, instead of just erroring out.
* **Inventory indicator in `card` output**, making it obvious at a glance
  whether a printing is `In Inventory` or `Not in Inventory`.
* **Sound cues** on add/remove and for high-value cards.

### Changed

* **`missing` checks against the local database instead of Scryfall**, which
  makes it dramatically faster and usable offline.

### Fixed

* **`serra migrate` could reset value history when run a second time.** The
  migration re-saved each card as pure Scryfall data, but the upsert merges
  into the existing document instead of replacing it, so the legacy
  `serra_*` ownership fields survived. A later run therefore found the same
  documents still looking legacy, converted them again, and overwrote the
  inventory entry's accumulated value history with the stale snapshot frozen
  in `serra_prices`. The legacy fields are now cleared once their data has
  been converted, which also makes re-running the migration a genuine no-op.
* **`serra migrate` never recorded the schema version.** The `meta` document
  was only written when a migration step actually ran, and since version 1 is
  a baseline with no step, no database was ever stamped — `migrate --status`
  reported version 1 because that is the assumed default, not because
  anything had recorded it. The version is now recorded once the database is
  up to date, and `migrate --status` distinguishes a recorded version from an
  assumed one.
* **Interactive add ranges only added the last card.** `12-15` built the card
  ID inside the loop but called `addCard` once afterwards, so only the last
  card of a range was stored. The foil/amount shortcut is now parsed before
  the range is expanded, so it applies to every card in the range too.
* **Duplicate-key error when updating a brand-new set.** `updateSet` inserted
  a newly-seen set, then inserted it again as part of a remove/add dance that
  was a no-op, hitting MongoDB's unique `_id` constraint. The set was created
  without its first price snapshot and only self-healed on the next run.
  Replaced with a single atomic upsert.
* **Price decrease on the most recent snapshot was sometimes hidden.**
  `showPriceHistory` always printed the last entry on an increase, but the
  decrease branch compared against an index that never occurs, so a drop on
  the latest snapshot only showed if it exceeded 5%. Zero "before" values no
  longer produce `+Inf%`/`NaN%` output either.
* **`stats` database connection error.**

### Internal

* `CLAUDE.md` documenting the architecture, build commands and the sharp
  edges of the codebase.
