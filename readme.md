# serra

Serra is my personal *Magic: The Gathering* collection tracker.

It began as a holiday project in winter 2021/2022 because I was frustrated of
Collection Tracker Websites that are:

* Pain to use
* Want ~$10 a month
* Don't have the features I want

So I started my own Collection Tracker using [Golang](https://golang.org),
[MongoDB](https://mongodb.com) and [Scryfall](https://scryfall.com) to have
an overview in what cards you own and what value they have.

**What Serra does**

* Tracks prices
* Calculates statistics
* Query/filter all of your cards
* Shows what cards/sets do best in value development.

**What Serra does not**

* Does not verify condition/language input against a fixed enum - anything you type is stored as-is

# What's new in 5.0.0

5.0.0 is a **breaking release**: the database schema changed. After upgrading
the binary, run `./serra migrate` once (see [Upgrade Notes](#4xx---5xx)).

Highlights:

* **`cards` and `inventory` are now separate collections.** Cached Scryfall
  data and your ownership data no longer share a document.
* **Language, condition and etched-foil tracking.** `add`/`remove` take
  `--language`, `--condition` and `--etched`, and each combination is its own
  inventory entry.
* **Price history for every card, not just the ones you own.** Every `update`
  imports the full Scryfall bulk file, so you can look at the price
  development of cards you are only thinking about buying.
* **`--all` on `tops`/`flops`** to analyse the whole card database instead of
  just your collection, and **`--legal`** to filter cards by format legality.
* **`missing` runs against the local database**, which makes it much faster.
* **Schema versioning.** serra now tells you when your database needs
  migrating instead of misbehaving quietly.

The full list, including bugfixes, is in [CHANGELOG.md](CHANGELOG.md).

## A note on how this release was built

Most of 5.0.0 was written together with [Claude](https://claude.ai). This is
a hobby project I maintain in the evenings, and the honest reason is time: I
have a long list of changes I want in serra and no realistic chance of doing
all of them by hand. Pairing with an LLM is what got the schema split, the
per-copy language/condition tracking and a pile of long-standing bugfixes
over the line instead of sitting in my notes for another year.

Everything in this release was reviewed and tested by me before it shipped,
and the commits that were written this way say so in their trailers. I'm
mentioning it here because I think it's worth being upfront about, not
because it changes what serra does.

# Quickstart

## Install Binaries

on macOS you can use

    brew install noqqe/tap/serra

on Linux/BSD/Windows you can download binaries from

    https://github.com/noqqe/serra/releases

## Spin up Database

To run serra, a MongoDB Database is required. The best way is to setup one by yourself. Any way it connects is fine. 
    
    docker run -d -p 27017:27017 --name mongo mongo:8

For "production" use this docker command is not recommended since it does not have persistent storage.

## Configure the Database

Configure `serra` via Environment variables

    export MONGODB_URI='mongodb://localhost:27017'
    export SERRA_CURRENCY=USD # or EUR

After that, you can add a card

    ./serra add usg/17
    ./serra update

Start exploring :) (the more cards you add, the more fun it is)

# Usage

The overall usage is described in `--help` text. But below are some examples.
```
Usage:
  serra [command]

Available Commands:
  add         Add a card to your collection
  card        Search & show cards from your collection
  check       Check if a card is in your collection
  completion  Generate the autocompletion script for the specified shell
  flops       What cards lost most value
  help        Help about any command
  migrate     Bring the database up to the schema this version of serra expects
  missing     Display missing cards from a set
  remove      Remove a card from your collection
  set         Search & show sets from your collection
  stats       Shows statistics of the collection
  tops        What cards gained most value
  update      Update card values from scryfall
  web         Startup web interface

Flags:
  -h, --help      help for serra
  -v, --version   version for serra

Use "serra [command] --help" for more information about a command.
```

## Add

To add a card to your collection.

![](https://github.com/noqqe/serra/blob/main/imgs/add.png)

## Cards

Query all of your cards with filters

![](https://github.com/noqqe/serra/blob/main/imgs/cards.png)

## Sets

List all your sets

![](https://github.com/noqqe/serra/blob/main/imgs/sets.png)

## Set

Show details of a single set

![](https://github.com/noqqe/serra/blob/main/imgs/set.png)

## Stats

Calculate some stats for all of your cards

![](https://github.com/noqqe/serra/blob/main/imgs/stats.png)

## Tops

Show what cards/set gained most value

![](https://github.com/noqqe/serra/blob/main/imgs/tops.png)

## Flops

Show what cards/set lost most value

![](https://github.com/noqqe/serra/blob/main/imgs/flops.png)

## Update

The update mechanism iterates over each card in your collection and fetches
its price. After all cards you own in a set are updated, the set value will
update. After all Sets are updated, the whole collection value is updated.

Every run also imports the entire Scryfall bulk file into the `cards`
collection, appending a price snapshot to every printing - not just the ones
you own. This lets you track price history for any card, owned or not.

![](https://github.com/noqqe/serra/blob/main/imgs/update.png)

## Check

To add a card to your collection.

![](https://github.com/noqqe/serra/blob/main/imgs/check.png)

## Adding all those cards, manually?

Yes. While there are serveral OCR/Photo Scanners for mtg cards, I found they
are not accurate enough. They guess Editions wrong, they have problems with
blue/black cards and so on.

I add my cards the `add --interactive` feature, since they are sorted by editions
anyways.

```
> ./serra add --interactive --unique --set one
one> 1
1x "Against All Odds" (uncommon, 0.06 USD) added to Collection.
one> 1
Not adding "Against All Odds" (uncommon, 0.06 USD) to Collection because it already exists.
one> 3
1x "Apostle of Invasion" (uncommon, 0.03 USD) added to Collection.
```

It also supports ranges of cards 
```
dmr> 1-3
1x "Auramancer" (common, 0.02$) added to Collection.
1x "Battle Screech" (uncommon, 0.09$) added to Collection.
1x "Cleric of the Forward Order" (common, 0.01$) added to Collection.
```

Its basically typing 2-3 digit numbers and hitting enter. I was way faster
with this approach then Smartphone scanners.

# Upgrade

If you want to upgrade, go to
[releases](https://github.com/noqqe/serra/releases) Page and download the
corresponding release for your platform.

For example:
```
wget https://github.com/noqqe/serra/releases/download/3.10.0/serra_Darwin_x86_64.tar.gz
tar zxfv serra_Darwin_x86_64.tar.gz
./serra 
```

## Database Schema Versioning

serra tracks which schema version its database is at in a `meta` collection.
Every command warns if the database is behind the version this build
expects, and refuses to run if it's ahead (i.e. you downgraded the binary).

Run `serra migrate` to bring the database up to date, or `serra migrate
--status` to check the current/expected version without changing anything.
It is always safe to run more than once.

## Upgrade Notes

### 4.x.x -> 5.x.x

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

#### Back up first

The migration rewrites documents in place. It is well-behaved and re-runnable,
but this is your collection — take a dump before you start:

    mongodump -d serra -o ./backup/

(see `mongodb_queries.md` for the authenticated variant)

#### Run the migration

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

#### Verify

    ./serra migrate --status
    ./serra stats

`migrate --status` should report the database and expected schema version as
the same number, no longer mention the pre-5.0 format, and `stats` should
show your collection again.

If you skip the migration, serra notices: a 4.x database records no schema
version to compare against, so instead of trusting the version number serra
checks whether ownership data is still sitting in the old format, and warns
on every command until you convert it. Your data is not lost in the
meantime — it just hasn't been moved into `inventory` yet, so the collection
reads as empty.

#### Downgrading

Going back to a 4.x binary after migrating is not supported: it knows nothing
about the `inventory` collection and would show an empty collection. Restore
the dump if you need to go back.

### 3.x.x -> 4.x.x

No extra steps needed

### 2.x.x -> 3.x.x

No extra steps needed. Only new Webinterface and Foil support

### 1.5.3 -> 2.0.0 

In this stage of the development of serra, I was breaking the original database
"schema" without migration. 

Sadly you need to export the cards from the mongodb and import it again using
`serra add ` commands

I wrote a little helper script in python to export all the cards in format
set/number and generate some queries

```
python3 export.py > add_commands.sh

head add_commands.sh
./serra add 5ed/3 -c 1
./serra add mmq/2 -c 1
./serra add p02/4 -c 1
./serra add chr/44 -c 1
./serra add 4ed/291 -c 1
./serra add 4ed/292 -c 1
./serra add mir/2 -c 1
./serra add usg/231 -c 1
./serra add mir/155 -c 1
./serra add pcy/29 -c 2

<do the upgrade of serra (download new binary>

<delete the old mongodb or just empty it completly>

bash add_commands.sh
```

# Development

## Install

    task build
    ./serra

(`task build` bakes the version string in via `git describe`. A plain `go
build ./cmd/serra` works too, it just reports its version as `unknown`.)

