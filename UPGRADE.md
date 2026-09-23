# Upgrading serra

Download the release for your platform from the
[releases page](https://github.com/noqqe/serra/releases), or on macOS:

    brew upgrade noqqe/tap/serra

Manual install:

    wget https://github.com/noqqe/serra/releases/download/5.0.0/serra_Darwin_x86_64.tar.gz
    tar zxfv serra_Darwin_x86_64.tar.gz
    ./serra --version

## Schema versioning

serra records the schema version of its database in a `meta` collection.
Every command warns if the database is behind the version the binary
expects, and refuses to run if it is ahead (i.e. you downgraded the binary).

    ./serra migrate --status   # show current/expected version, change nothing
    ./serra migrate            # bring the database up to date

`migrate` is idempotent — it is always safe to run more than once.

## 4.x.x → 5.x.x

**This upgrade requires a migration.** The database schema changed: the
`cards` collection used to hold both cached Scryfall data and your ownership
data (count, value history, added/updated timestamps) in the same document.
These are now split:

* `cards` only holds cached Scryfall data, one document per printing.
* `inventory` holds ownership data, with one entry per card/finish
  (normal/foil/etched)/language/condition combination, carrying its own
  `count` and value history.

`add`/`remove` now support `--language` and `--condition` flags (defaulting
to `en`/`nm`) to track those per copy, and `--etched` to add/remove the
etched finish.

### 1. Back up first

The migration rewrites documents in place. It is well-behaved and
re-runnable, but this is your collection — take a dump before you start:

    mongodump -d serra -o ./backup/

(see [mongodb_queries.md](mongodb_queries.md) for the authenticated variant)

### 2. Run the migration

After upgrading the binary:

    ./serra migrate

This does two things:

1. Splits every legacy `cards` document that carries ownership data into a
   clean `cards` document plus one `inventory` entry per finish you owned.
   Existing price history is carried over, narrowed down to the finish it
   belongs to. Copies are assumed to be `en`/`nm`, since older versions did
   not track language or condition — adjust afterwards if you kept that
   information elsewhere.
2. Applies any pending schema migrations and records the schema version, so
   future upgrades have a stored baseline to count from rather than an
   assumption.

It is safe to run more than once. Converted documents have their old fields
cleared, so a second run finds nothing left to do and reports `0 cards, 0
inventory entries`. A partial failure can simply be resumed by running it
again: anything that did not make it across keeps its old format and gets
picked up on the next run.

### 3. Verify

    ./serra migrate --status
    ./serra stats

`migrate --status` should report the database and expected schema version as
the same number and no longer mention the pre-5.0 format, and `stats` should
show your collection again.

If you skip the migration, serra notices: a 4.x database records no schema
version to compare against, so instead of trusting the version number serra
checks whether ownership data is still sitting in the old format, and warns
on every command until you convert it. Your data is not lost in the
meantime — it just hasn't been moved into `inventory` yet, so the collection
reads as empty.

### Downgrading

Going back to a 4.x binary after migrating is not supported: it knows nothing
about the `inventory` collection and would show an empty collection. Restore
the dump if you need to go back.

## 3.x.x → 4.x.x

No extra steps needed.

## 2.x.x → 3.x.x

No extra steps needed. Only new web interface and foil support.

## 1.5.3 → 2.0.0

In this stage of the development of serra, I was breaking the original
database "schema" without migration.

Sadly you need to export the cards from MongoDB and import them again using
`serra add` commands. I wrote a little helper script in python to export all
the cards in format `set/number` and generate the queries:

```
python3 export.py > add_commands.sh

head add_commands.sh
./serra add 5ed/3 -c 1
./serra add mmq/2 -c 1
./serra add p02/4 -c 1
./serra add chr/44 -c 1

<do the upgrade of serra (download new binary)>
<delete the old mongodb or just empty it completely>

bash add_commands.sh
```
