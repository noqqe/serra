# serra

# Install

    go build .
    ./serra

# Todo

mvp

* refactor cards/set to scryfall.go
* refactor storage to have a storage object.
* single view for card, with history
* docker with local mounted volume and git ignore
* mongodb backup container

optional

* termui overview
* add - do search for cards by name

# What its not

* Gives a shit about conditions (NM, M, GD...)
* If the card is foil

# Cheatsheet

Find cards that increased prices

    db.cards.find({$expr: {$gt: [{$arrayElemAt: ["$serra_prices", -2]}, {$arrayElemAt: ["$serra_prices", -1]}]}}, {name:1})
