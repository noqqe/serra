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

* Does not care about conditions (NM, M, GD...)
* Does not track etched cards. Only normal and foil.

# Quickstart

### Install Binaries

on macOS you can use

    brew install noqqe/tap/serra

on Linux/BSD/Windows you can download binaries from

    https://github.com/noqqe/serra/releases

### Spin up Database

To run serra, a MongoDB Database is required. The best way is to setup one by yourself. Any way it connects is fine. 
    

You can also use the docker-compose setup included in this Repo:

    docker-compose up -d

### Configure the Database

Configure `serra` via Environment variables

    export MONGODB_URI='mongodb://root:root@localhost:27017'
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
  completion  Generate the autocompletion script for the specified shell
  flops       What cards lost most value
  help        Help about any command
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

![](https://github.com/noqqe/serra/blob/main/imgs/update.png)

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

# Development

## Install

    go build .
    ./serra

## MongoDB Operations

A few commands that do backups and exports of your data inside of the docker
container.

Do a database dump

    mongodump  -u root -p root --authenticationDatabase admin -d serra -o /backup/

Do a collection export to json

    mongoexport  -u root -p root --authenticationDatabase admin -d serra -c cards > /backup/cards.json
    mongoexport  -u root -p root --authenticationDatabase admin -d serra -c sets > /backup/sets.json
    mongoexport  -u root -p root --authenticationDatabase admin -d serra -c total > /backup/total.json
