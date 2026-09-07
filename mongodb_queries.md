## MongoDB Operations

A few commands that do backups and exports of your data inside of the docker
container.

Do a database dump

    mongodump  -u root -p root --authenticationDatabase admin -d serra -o /backup/

Do a collection export to json

    mongoexport  -u root -p root --authenticationDatabase admin -d serra -c cards > /backup/cards.json
    mongoexport  -u root -p root --authenticationDatabase admin -d serra -c inventory > /backup/inventory.json
    mongoexport  -u root -p root --authenticationDatabase admin -d serra -c sets > /backup/sets.json
    mongoexport  -u root -p root --authenticationDatabase admin -d serra -c total > /backup/total.json

## Schema

* `cards` - pure Scryfall data, one document per printing, keyed by Scryfall ID (`_id`).
* `inventory` - your ownership data, one document per unique
  `card_id`+`finish`+`language`+`condition` combination:
  * `card_id` references `cards._id`
  * `finish` is `nonfoil`, `foil` or `etched`
  * `count` how many copies you own of that exact combination
  * `value_history` array of `{date, eur, usd}` snapshots
  * `created`/`updated` timestamps
* `sets` - cached Scryfall set data plus a `serra_prices` value history of the set's total value.
* `total` - a single document tracking the value history of the whole collection.

## Cheatsheet Queries

Find cards that increased in value

    db.inventory.find({$expr: {$gt: [{$arrayElemAt: ["$value_history", -2]}, {$arrayElemAt: ["$value_history", -1]}]}}, {card_id:1, finish:1})

Update an inventory entry's value directly

		db.inventory.update(
		{'_id':'<card_id>|nonfoil|en|nm'},
		{$set:{'updated':ISODate("2021-11-02T09:28:56.504Z")},
		$push: {"value_history": { date: ISODate("2021-11-02T09:28:56.504Z"), eur: 0.1, usd: 0.1 }}});

Set value (joins inventory with cards)

    db.inventory.aggregate([
      { $lookup: { from: "cards", localField: "card_id", foreignField: "_id", as: "card" } },
      { $unwind: "$card" },
      { $group: { _id: "$set", value: { $sum: { $multiply: ["$card.prices.eur", "$count"] } }, count: { $sum: "$count" } } }
    ])

Color distribution (joins inventory with cards)

    db.inventory.aggregate([
      { $lookup: { from: "cards", localField: "card_id", foreignField: "_id", as: "card" } },
      { $unwind: "$card" },
      { $group: { _id: { color: "$card.colors" }, count: { $sum: "$count" } } }
    ])

Calculate value of all sets

    db.sets.aggregate({$match: {serra_prices: {$exists: true}}}, {$project: {name: 1, "totalValue": {$arrayElemAt: ["$serra_prices", -1]} }}, {$group: {_id: null, total: {$sum: "$totalValue.value" }}})

Show when cards where added per month of the year

    db.inventory.aggregate({ $project: { month: { $month: "$created" }, year: { $year: "$created" } } }, { $group: { _id: { month: "$month", year: "$year" }, count: { $sum: 1 } } })

Show card count by artists (joins inventory with cards)

    db.inventory.aggregate([
      { $lookup: { from: "cards", localField: "card_id", foreignField: "_id", as: "card" } },
      { $unwind: "$card" },
      { $group: { _id : "$card.artist", total : {$sum:1}}},
      { $sort: {total:-1} }
    ])
