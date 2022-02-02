# serra

Serra is my personal Magic: The Gathering collection tool.
It uses a MongoDB and [Scryfall](https://scryfall.com).

## What Serra does

* tracks prices
* calculates statistics
* shows what cards/sets do best in value development.

## What Serra does not

* Does not give a shit about conditions (NM, M, GD...)
* Does not track if card is foil or not (may come in the future)
* Its not configurable to have Dollar/US Prices

# Quickstart

Spin up a MongoDB (you can use docker-compose)

    docker-compose up -d

Add a card

    ./serra add usg/17

Start exploring :) (the more cards you add, the more fun it is)

# Usage

```
./serra
Usage:
  serra add <cardid>... [--count=<number>]
  serra remove <cardid>...
  serra cards [--rarity=<rarity>] [--set=<setcode>] [--sort=<sort>]
  serra card <cardid>...
  serra tops [--limit=<limit>]
  serra flops [--limit=<limit>]
  serra missing <setcode>
  serra set <setcode>
  serra sets
  serra update
  serra stats
```

## Add

To add a card to your collection

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

Show what cards/se gained most value
![](https://github.com/noqqe/serra/blob/main/imgs/tops.png)

## Update

The update mechanism iterates over each card in your collection and fetches
its price. After all cards you own in a set are updated, the set value will
update. After all Sets are updated, the whole collection value is updated.

# Install

    go build .
    ./serra

# Todo

* prices since the beginning
* calculate biggest gaining cards (Up XX %!)
* display (+/-) % in sets overview
* display (+/-) % in history views
* Reserved List
* Mythic

# Cheatsheet Queries

Find cards that increased prices

    db.cards.find({$expr: {$gt: [{$arrayElemAt: ["$serra_prices", -2]}, {$arrayElemAt: ["$serra_prices", -1]}]}}, {name:1})

Update card Price

		db.cards.update(
		{'_id':'8fa2ecf9-b53c-4f1d-9028-ca3820d043cb'},
		{$set:{'serra_updated':ISODate("2021-11-02T09:28:56.504Z")},
		$push: {"serra_prices": { date: ISODate("2021-11-02T09:28:56.504Z"), value: 0.1 }}});

Set value

    db.cards.aggregate([{ $group: { _id: { set: "$set" }, value: { $sum: { $multiply: ["$prices.eur", "$serra_count"] } }, count: { $sum: 1 } } }])

Color distribution

     db.cards.aggregate([{ $group: { _id: { color: "$colors" }, count: { $sum: 1 } } }])

Calculate value of all sets

    db.sets.aggregate({$match: {serra_prices: {$exists: true}}}, {$project: {name: 1, "totalValue": {$arrayElemAt: ["$serra_prices", -1]} }}, {$group: {_id: null, total: {$sum: "$totalValue.value" }}})

Calculate what cards gained most value in percent

    db.cards.aggregate({$project: {set: 1, collectornumber:1, name: 1, "old": {$arrayElemAt: ["$serra_prices.value", -2]}, "current": {$arrayElemAt: ["$serra_prices.value", -1]} }}, {$match: {old: {$gt: 2}}} ,{$project: {name: 1,set:1,collectornumber:1,current:1, "rate": {$subtract: [{$divide: ["$current", {$divide: ["$old", 100]}]}, 100]} }}, {$sort: { rate: -1}})

# MongoDB Operations

A few commands that do backups and exports of your data inside of the docker
container.

Do a database dump

    mongodump  -u root -p root --authenticationDatabase admin -d serra -o /backup/

Do a collection export to json

    mongoexport  -u root -p root --authenticationDatabase admin -d serra -c cards > /backup/cards.json
    mongoexport  -u root -p root --authenticationDatabase admin -d serra -c sets > /backup/sets.json
    mongoexport  -u root -p root --authenticationDatabase admin -d serra -c total > /backup/total.json
