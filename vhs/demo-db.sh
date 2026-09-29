#!/usr/bin/env bash
#
# Creates the throwaway MongoDB the README gifs are recorded against.
#
# The tapes deliberately do not talk to a real collection: that would leak
# whatever happens to be in it into the README, and the output would change
# between runs for reasons that have nothing to do with serra. Instead this
# spins up a disposable mongo, seeds it from the fixed card list in
# seed/cards.txt, and back-fills a value history so the commands that report
# on price movement have something to report.
#
#   vhs/demo-db.sh up      # start and seed (no-op if already seeded)
#   vhs/demo-db.sh reset   # wipe and seed again from scratch
#   vhs/demo-db.sh down    # remove the container
#   vhs/demo-db.sh uri     # print the MONGODB_URI the tapes use
#
# Everything is reachable from the tapes via MONGODB_URI, so nothing here
# touches the database your own $MONGODB_URI points at.

set -euo pipefail

CONTAINER="${SERRA_DEMO_CONTAINER:-serra-vhs-demo}"
PORT="${SERRA_DEMO_PORT:-27117}"
IMAGE="${SERRA_DEMO_IMAGE:-mongo:8}"
URI="mongodb://127.0.0.1:${PORT}"

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(dirname "$HERE")"
SERRA="$ROOT/serra"

log() { printf '\033[1;35m==>\033[0m %s\n' "$*" >&2; }
die() { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

# mongosh lives inside the container, so we never need it on the host.
mongosh_eval() { docker exec -i "$CONTAINER" mongosh --quiet serra --eval "$1"; }

# Reads a script from stdin. It has to go in as a --file rather than on
# mongosh's stdin: stdin is evaluated statement by statement like a REPL,
# which breaks function hoisting and echoes every declaration back.
mongosh_file() {
	docker exec -i "$CONTAINER" sh -c 'cat >/tmp/seed.js && mongosh --quiet serra --file /tmp/seed.js'
}

check_prerequisites() {
	command -v docker >/dev/null || die "docker is required to run the demo database"
	docker info >/dev/null 2>&1 || die "the docker daemon is not running"
	command -v python3 >/dev/null || die "python3 is required to import the set list"
	[ -x "$SERRA" ] || die "no serra binary at $SERRA - run 'task build' first"
}

start_container() {
	if docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
		log "reusing running container $CONTAINER"
	elif docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER"; then
		log "starting existing container $CONTAINER"
		docker start "$CONTAINER" >/dev/null
	else
		log "creating container $CONTAINER from $IMAGE on port $PORT"
		docker run -d --name "$CONTAINER" -p "${PORT}:27017" "$IMAGE" >/dev/null
	fi

	log "waiting for mongo to accept connections"
	for _ in $(seq 1 60); do
		if docker exec "$CONTAINER" mongosh --quiet --eval 'db.runCommand({ping:1})' >/dev/null 2>&1; then
			return 0
		fi
		sleep 1
	done
	die "mongo in $CONTAINER did not come up within 60s"
}

is_seeded() {
	[ "$(mongosh_eval 'db.inventory.countDocuments({})' | tr -d '\r')" != "0" ]
}

# serra only ever writes the sets collection from `serra update`, which pulls
# the whole Scryfall bulk file. For a 30 card demo that is absurd, so import
# the set metadata straight from the API instead. The field names below are
# the Go field names lowercased, because Set carries almost no bson tags.
import_sets() {
	log "importing set metadata from Scryfall"
	python3 - <<-'PY' | mongosh_file
	import json, sys, urllib.request

	req = urllib.request.Request(
	    "https://api.scryfall.com/sets",
	    headers={"User-Agent": "serra-vhs-demo", "Accept": "application/json"},
	)
	with urllib.request.urlopen(req, timeout=60) as resp:
	    sets = json.load(resp)["data"]

	docs = [
	    {
	        "_id": s["id"],
	        "cardcount": s.get("card_count", 0),
	        "code": s["code"],
	        "digital": s.get("digital", False),
	        "foilonly": s.get("foil_only", False),
	        "iconsvguri": s.get("icon_svg_uri", ""),
	        "name": s["name"],
	        "nonfoilonly": s.get("nonfoil_only", False),
	        "object": s.get("object", "set"),
	        "releasedat": s.get("released_at", ""),
	        "scryfalluri": s.get("scryfall_uri", ""),
	        "searchuri": s.get("search_uri", ""),
	        "settype": s.get("set_type", ""),
	        "tcgplayerid": s.get("tcgplayer_id", 0),
	        "uri": s.get("uri", ""),
	        "serra_prices": [],
	    }
	    for s in sets
	]

	print("const now = new Date();")
	print("const sets = %s;" % json.dumps(docs))
	print("""
	sets.forEach(function (s) {
	  s.serra_created = now;
	  s.serra_updated = now;
	  db.sets.replaceOne({ _id: s._id }, s, { upsert: true });
	});
	print("imported " + sets.length + " sets");
	""")
	PY
}

# `missing` lists the printings of a set you do *not* own, which it reads
# out of the cards collection - serra's Scryfall cache. Adding 30 cards only
# ever caches those 30, so without this the command has nothing to report.
# serra would normally fill the cache from the bulk file; for four sets the
# search API is enough.
#
# Scryfall's JSON keys map onto serra's bson field names by dropping the
# underscores: Card carries almost no bson tags, so its Mongo field names are
# its Go field names lowercased ("collector_number" -> CollectorNumber ->
# "collectornumber"). The exception is the prices sub-document, which does
# have tags and keeps its underscores.
import_cards() {
	log "caching the printings of the demo sets"
	# The set codes come in as arguments, not on stdin: stdin is where
	# python3 reads the script below from.
	local codes
	codes="$(sed -e 's/#.*//' "$HERE/seed/cards.txt" |
		grep -oE '[a-z0-9]+/[0-9]+' | cut -d/ -f1 | sort -u | tr '\n' ' ')"
	# shellcheck disable=SC2086 # one argument per set code
	python3 - $codes <<-'PY' | mongosh_file
		import json, sys, time, urllib.parse, urllib.request

		BATCH = 250

		def fetch(url):
		    req = urllib.request.Request(
		        url,
		        headers={"User-Agent": "serra-vhs-demo", "Accept": "application/json"},
		    )
		    with urllib.request.urlopen(req, timeout=60) as resp:
		        return json.load(resp)

		def price(value):
		    # Scryfall sends prices as strings or null, serra stores floats.
		    try:
		        return float(value)
		    except (TypeError, ValueError):
		        return 0.0

		def bsonify(value, strip=True):
		    if isinstance(value, dict):
		        return {
		            (k.replace("_", "") if strip else k): bsonify(v, strip)
		            for k, v in value.items()
		        }
		    if isinstance(value, list):
		        return [bsonify(v, strip) for v in value]
		    return value

		def convert(card, now):
		    doc = bsonify({k: v for k, v in card.items() if k not in ("id", "prices")})
		    doc["_id"] = card["id"]
		    doc["prices"] = {
		        "date": now,
		        "eur": price(card.get("prices", {}).get("eur")),
		        "eur_foil": price(card.get("prices", {}).get("eur_foil")),
		        "tix": price(card.get("prices", {}).get("tix")),
		        "usd": price(card.get("prices", {}).get("usd")),
		        "usd_etched": price(card.get("prices", {}).get("usd_etched")),
		        "usd_foil": price(card.get("prices", {}).get("usd_foil")),
		    }
		    doc["price_history"] = [dict(doc["prices"])]
		    return doc

		codes = sys.argv[1:]
		docs = []
		for code in codes:
		    url = "https://api.scryfall.com/cards/search?" + urllib.parse.urlencode(
		        {"q": "set:" + code, "unique": "prints", "order": "set"}
		    )
		    while url:
		        page = fetch(url)
		        docs.extend(page["data"])
		        url = page.get("next_page")
		        time.sleep(0.1)  # Scryfall asks for 50-100ms between requests

		print('const now = new Date();')
		for i in range(0, len(docs), BATCH):
		    batch = [convert(c, {"__date__": True}) for c in docs[i : i + BATCH]]
		    payload = json.dumps(batch).replace('{"__date__": true}', "now")
		    # $setOnInsert: cards serra already cached while adding the demo
		    # collection keep the price history it wrote for them.
		    # _id goes in the filter, not the update: mongo rejects an
		    # immutable field in $setOnInsert.
		    print(
		        "db.cards.bulkWrite(%s.map(function (d) {"
		        " const id = d._id; delete d._id;"
		        " return { updateOne: { filter: { _id: id },"
		        " update: { $setOnInsert: d }, upsert: true } }; }),"
		        " { ordered: false });" % payload
		    )
		print('print("cached " + %d + " printings");' % len(docs))
		PY
}

add_cards() {
	log "adding the demo collection ($(grep -cvE '^\s*(#|$)' "$HERE/seed/cards.txt") cards)"
	# serra resolves ./sounds, ./templates and ./assets relative to the
	# working directory, so every invocation has to happen from the repo root.
	cd "$ROOT"
	while IFS= read -r line; do
		line="${line%%#*}"
		line="$(printf '%s' "$line" | tr -s ' ' | sed -e 's/^ *//' -e 's/ *$//')"
		[ -n "$line" ] || continue
		# shellcheck disable=SC2086 # the arguments are meant to be split
		MONGODB_URI="$URI" "$SERRA" add $line
	done <"$HERE/seed/cards.txt"
}

backfill_history() {
	log "back-filling value history"
	mongosh_file <"$HERE/seed/history.js"
}

seed() {
	import_sets
	import_cards
	add_cards
	backfill_history
	log "demo database ready at $URI"
}

case "${1:-up}" in
up)
	check_prerequisites
	start_container
	if is_seeded; then
		log "already seeded - use 'vhs/demo-db.sh reset' to rebuild it"
	else
		seed
	fi
	;;
reset)
	check_prerequisites
	start_container
	log "dropping the serra database"
	mongosh_eval 'db.dropDatabase()' >/dev/null
	seed
	;;
down)
	log "removing container $CONTAINER"
	docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
	;;
uri)
	printf '%s\n' "$URI"
	;;
*)
	die "unknown command '$1' - expected up, reset, down or uri"
	;;
esac
