// Back-fills the demo database with a value history.
//
// A freshly seeded collection has exactly one price snapshot per inventory
// entry, which leaves `stats`, `set`, `tops` and `flops` with nothing to
// show. This walks today's real Scryfall prices backwards over MONTHS
// monthly snapshots and writes them to `inventory.value_history`, rolls
// them up into `sets.serra_prices` and `total.value`, and backdates when
// each entry was added - exactly the shape `serra update` would have
// produced had it been run once a month.
//
// The per-entry drift is derived from the entry's ID, so re-seeding the
// same card list produces the same shape (the absolute numbers still move
// with Scryfall's real prices).
//
// Run with: mongosh --quiet serra --file history.js

const MONTHS = 15;

// Share of entries that trend upwards over the whole window. The rest
// trend down, so `flops` has something to report.
const SHARE_RISING = 0.62;

// FNV-1a, so an entry's drift is a pure function of its ID.
function hash(str) {
  let h = 2166136261 >>> 0;
  for (let i = 0; i < str.length; i++) {
    h ^= str.charCodeAt(i);
    h = Math.imul(h, 16777619) >>> 0;
  }
  return h >>> 0;
}

// Plain LCG - we want repeatable, not cryptographic.
function rng(seed) {
  let s = seed >>> 0;
  return function () {
    s = (Math.imul(s, 1664525) + 1013904223) >>> 0;
    return s / 4294967296;
  };
}

function round2(v) {
  return Math.round(v * 100) / 100;
}

const now = new Date();

// Snapshot i, counted from the oldest (0) to today (MONTHS - 1). Older
// snapshots are pinned to mid-month: subtracting a month from the 29th
// through 31st rolls over into the following month on the short ones.
function snapshotDate(i) {
  if (i === MONTHS - 1) {
    return now;
  }
  const d = new Date(now.getFullYear(), now.getMonth(), 15, 12, 0, 0);
  d.setMonth(d.getMonth() - (MONTHS - 1 - i));
  return d;
}

// The price a given finish of a card is worth, in both currencies.
function priceFor(card, finish) {
  const p = (card && card.prices) || {};
  switch (finish) {
    case "foil":
      return { eur: p.eur_foil || 0, usd: p.usd_foil || 0 };
    case "etched":
      return { eur: 0, usd: p.usd_etched || 0 };
    default:
      return { eur: p.eur || 0, usd: p.usd || 0 };
  }
}

// A market-wide swing every card rides along with, mean-zero so it does not
// move the long term trend. Without it every card drifts at its own steady
// rate, the collection total grows by a smooth few percent a month, and
// serra's history view - which only prints a snapshot once it moved more
// than 5% - has almost nothing to print.
const marketFactors = (function () {
  const r = rng(hash("serra-vhs-market"));
  const f = new Array(MONTHS);
  f[MONTHS - 1] = 1;
  for (let i = MONTHS - 2; i >= 0; i--) {
    const step = (r() - 0.5) * 0.15;
    f[i] = f[i + 1] / (1 + step);
  }
  return f;
})();

// Multipliers to apply to today's price, oldest first. The last one is
// always 1 so the newest snapshot matches the real current price.
function factors(seed) {
  const r = rng(seed);
  // Where this card ends up over the whole window: mostly up, with a real
  // left tail so `flops` is not just the least profitable gainers.
  const total = r() < SHARE_RISING ? 0.15 + r() * 1.05 : -(0.2 + r() * 0.35);

  const f = new Array(MONTHS);
  for (let i = 0; i < MONTHS; i++) {
    const trend = Math.pow(1 + total, -(MONTHS - 1 - i) / (MONTHS - 1));
    const noise = i === MONTHS - 1 ? 1 : 1 + (r() - 0.5) * 0.03;
    f[i] = trend * marketFactors[i] * noise;
  }
  return f;
}

// Running totals per set code, and for the collection as a whole. Foil and
// non-foil are kept apart because that is how serra stores set/total
// history: eur/usd hold the non-foil sum, eur_foil/usd_foil the foil one.
const perSet = {};
const overall = emptyBuckets();

function emptyBuckets() {
  return {
    eur: new Array(MONTHS).fill(0),
    usd: new Array(MONTHS).fill(0),
    eurFoil: new Array(MONTHS).fill(0),
    usdFoil: new Array(MONTHS).fill(0),
  };
}

let entries = 0;

db.inventory.find({}).forEach(function (entry) {
  const card = db.cards.findOne({ _id: entry.card_id });
  if (!card) {
    print("skipping " + entry._id + ": no cached card");
    return;
  }

  const seed = hash(entry._id);
  const price = priceFor(card, entry.finish);
  const f = factors(seed);

  const history = [];
  for (let i = 0; i < MONTHS; i++) {
    const eur = round2(price.eur * f[i]);
    const usd = round2(price.usd * f[i]);
    history.push({
      date: snapshotDate(i),
      eur: eur,
      eur_foil: 0,
      tix: 0,
      usd: usd,
      usd_etched: 0,
      usd_foil: 0,
    });

    if (!perSet[entry.set]) {
      perSet[entry.set] = emptyBuckets();
    }
    const foil = entry.finish !== "nonfoil";
    const bucket = perSet[entry.set];
    const key = foil ? "eurFoil" : "eur";
    const keyUsd = foil ? "usdFoil" : "usd";
    bucket[key][i] += eur * entry.count;
    bucket[keyUsd][i] += usd * entry.count;
    overall[key][i] += eur * entry.count;
    overall[keyUsd][i] += usd * entry.count;
  }

  // Backdate when this entry was added, so `stats` has more than a single
  // month in its "cards added" breakdown.
  const addedAt = snapshotDate(Math.floor(rng(seed ^ 0x5bf03635)() * MONTHS));

  db.inventory.updateOne(
    { _id: entry._id },
    { $set: { value_history: history, created: addedAt, updated: now } }
  );
  entries++;
});

// Turn a bucket of running sums into the price history serra expects.
function rollup(bucket) {
  const history = [];
  for (let i = 0; i < MONTHS; i++) {
    history.push({
      date: snapshotDate(i),
      eur: round2(bucket.eur[i]),
      eur_foil: round2(bucket.eurFoil[i]),
      tix: 0,
      usd: round2(bucket.usd[i]),
      usd_etched: 0,
      usd_foil: round2(bucket.usdFoil[i]),
    });
  }
  return history;
}

let sets = 0;
Object.keys(perSet).forEach(function (code) {
  const res = db.sets.updateOne(
    { code: code },
    { $set: { serra_prices: rollup(perSet[code]), serra_updated: now } }
  );
  if (res.matchedCount === 0) {
    print("warning: set " + code + " is not in the sets collection");
    return;
  }
  sets++;
});

db.total.replaceOne({ _id: "1" }, { _id: "1", value: rollup(overall) }, { upsert: true });

print("backfilled " + MONTHS + " snapshots for " + entries + " inventory entries and " + sets + " sets");
