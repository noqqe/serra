# Recording the README gifs

Every gif in the README is generated from a [VHS](https://github.com/charmbracelet/vhs)
tape in this directory, so they can be re-recorded whenever serra's output
changes instead of being re-screenshotted by hand.

```bash
task gifs                 # re-record all of them into ../imgs
task gif -- stats         # re-record just imgs/stats.gif
```

Or drive `vhs` yourself, from *this* directory — it resolves both `Source`
and `Output` against the working directory, not against the tape:

```bash
cd vhs && vhs stats.tape
```

## What you need

* [`vhs`](https://github.com/charmbracelet/vhs) — `brew install vhs`
* `docker`, for the throwaway database described below
* network access — seeding pulls real card data from Scryfall

Record locally, not in CI: `serra` opens an audio device to play its
success/error cues, and that panics on a machine without one.

## The demo database

The tapes never touch a real collection. `demo-db.sh` spins up a disposable
MongoDB on port `27117` and seeds it from `seed/cards.txt`, a fixed list of
~30 cards across four sets:

```bash
vhs/demo-db.sh up      # start and seed (no-op if already seeded)
vhs/demo-db.sh reset    # wipe and seed again from scratch
vhs/demo-db.sh down     # remove the container
vhs/demo-db.sh uri      # print the MONGODB_URI the tapes use
```

Two things the seeding does that `serra add` alone would not:

* **Set metadata** is imported straight from `api.scryfall.com/sets`.
  serra normally only writes the `sets` collection from `serra update`,
  which downloads the entire Scryfall bulk file — too much for a 30 card
  demo. `set <code>` and `missing` need it.
* **Value history** is back-filled by `seed/history.js`. A freshly seeded
  collection has a single price snapshot per entry, which leaves `stats`,
  `set`, `tops` and `flops` with nothing to report. The script walks today's
  real prices backwards over 15 monthly snapshots, with a per-entry drift
  derived from the entry's ID so the shape is the same on every re-seed.

Prices come from Scryfall at seeding time, so the numbers in the gifs move
with the real market — the *shape* of the history is what is pinned, not the
values.

## Layout

| File | What it is |
| --- | --- |
| `common.tape` | shared `Set` directives, sourced first by every tape |
| `setup.tape` | hidden shell prep: `cd` to the repo root, point `MONGODB_URI` at the demo database, set a clean prompt |
| `<command>.tape` | one tape per serra command, writing `../imgs/<command>.gif` |
| `demo-db.sh` | the disposable database |
| `seed/cards.txt` | the demo collection, one `serra add` invocation per line |
| `seed/history.js` | the value-history back-fill |

A tape that needs a taller or shorter terminal overrides `Set Height`
between `Source common.tape` and `Source setup.tape` — VHS applies all
settings before the recording starts, so the last one wins.

## Adding a tape

1. Write `vhs/<name>.tape`, starting with:

   ```
   Output ../imgs/<name>.gif
   Source common.tape
   Source setup.tape
   ```

2. Add `<name>` to `GIF_TAPES` in `Taskfile.yaml`. The order there is not
   cosmetic: read-only tapes run first, then `add`/`remove`, which are
   written to cancel each other out so a full run leaves the demo database
   where it started, and `update` last because it pulls the bulk file.
3. Reference `imgs/<name>.gif` from the README.
