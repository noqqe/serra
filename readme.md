# serra

# Install

    go build .
    ./serra

# Todo

mvp

* single view for card, with history

optional

* termui overview
* add - do search for cards by name

# What its not

* Gives a shit about conditions (NM, M, GD...)
* If the card is foil

# Cheatsheet

Find cards that increased prices

    db.cards.find({$expr: {$gt: [{$arrayElemAt: ["$serra_prices", -2]}, {$arrayElemAt: ["$serra_prices", -1]}]}}, {name:1})

# MongoDB Operations

Do a database dump

    mongodump  -u root -p root --authenticationDatabase admin -d serra -o /backup/

Do a collection export to json

    mongoexport  -u root -p root --authenticationDatabase admin -d serra -c cards
