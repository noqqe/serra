# serra

# Install

    go build .
    ./serra

# Todo

* prices since the beginning
* calculate biggest gaining cards (Up XX %!)
* display (+/-) % in sets overview
* display (+/-) % in history views


# What its not

* Gives a shit about conditions (NM, M, GD...)
* If the card is foil

# Cheatsheet

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

Do a database dump

    mongodump  -u root -p root --authenticationDatabase admin -d serra -o /backup/

Do a collection export to json

    mongoexport  -u root -p root --authenticationDatabase admin -d serra -c cards > /backup/cards.json
    mongoexport  -u root -p root --authenticationDatabase admin -d serra -c sets > /backup/sets.json
    mongoexport  -u root -p root --authenticationDatabase admin -d serra -c total > /backup/total.json
