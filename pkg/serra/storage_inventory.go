package serra

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Finishes a card can be owned in.
const (
	FinishNonfoil = "nonfoil"
	FinishFoil    = "foil"
	FinishEtched  = "etched"
)

// Defaults used when a language/condition is not explicitly specified.
const (
	DefaultLanguage  = "en"
	DefaultCondition = "nm"
)

// InventoryEntry represents ownership of a card: how many copies you own of
// a specific printing (CardID), in a specific finish, language and
// condition, plus the value history tracked for that specific combination.
//
// One entry exists per unique combination of CardID+Finish+Language+Condition,
// identified by a deterministic ID so that adding/removing copies can be
// expressed as a single atomic increment/decrement.
type InventoryEntry struct {
	ID              string        `bson:"_id"`
	CardID          string        `bson:"card_id"`
	Set             string        `bson:"set"`
	CollectorNumber string        `bson:"collectornumber"`
	Finish          string        `bson:"finish"`
	Language        string        `bson:"language"`
	Condition       string        `bson:"condition"`
	Count           int64         `bson:"count"`
	ValueHistory    []PriceEntry  `bson:"value_history"`
	Created         bson.DateTime `bson:"created"`
	Updated         bson.DateTime `bson:"updated"`
}

// inventoryID builds the deterministic ID identifying a unique
// CardID+Finish+Language+Condition combination.
func inventoryID(cardID, finish, language, condition string) string {
	return fmt.Sprintf("%s|%s|%s|%s", cardID, finish, language, condition)
}

// finishSuffix returns a human readable suffix used in log messages for
// non-default finishes.
func finishSuffix(finish string) string {
	switch finish {
	case FinishFoil:
		return ", foil"
	case FinishEtched:
		return ", etched"
	default:
		return ""
	}
}

type InventoryCollection struct {
	*mongo.Collection
}

func (client StorageClient) getInventoryCollection() InventoryCollection {
	return InventoryCollection{client.Database("serra").Collection("inventory")}
}

// IncrementInventory adds amount copies to the inventory entry identified by
// cardID+finish+language+condition, creating it (with initialSnapshot as the
// first value history entry) if it does not exist yet.
func (coll InventoryCollection) IncrementInventory(cardID, set, collectorNumber, finish, language, condition string, amount int64, initialSnapshot PriceEntry) (*InventoryEntry, error) {
	l := Logger()
	now := bson.NewDateTimeFromTime(time.Now())

	filter := bson.M{"_id": inventoryID(cardID, finish, language, condition)}
	update := bson.M{
		"$inc": bson.M{"count": amount},
		"$set": bson.M{"updated": now},
		"$setOnInsert": bson.M{
			"card_id":         cardID,
			"set":             set,
			"collectornumber": collectorNumber,
			"finish":          finish,
			"language":        language,
			"condition":       condition,
			"created":         now,
			"value_history":   []PriceEntry{initialSnapshot},
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var result InventoryEntry
	err := coll.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&result)
	if err != nil {
		l.Fatalf("Could not update inventory data due to connection errors to database: %s", err.Error())
		return nil, err
	}
	return &result, nil
}

// upsertMigratedEntry replaces (or inserts) an inventory entry outright,
// used by the one-time legacy migration so that re-running it is safe.
func (coll InventoryCollection) upsertMigratedEntry(entry *InventoryEntry) error {
	opts := options.Replace().SetUpsert(true)
	_, err := coll.ReplaceOne(context.TODO(), bson.M{"_id": entry.ID}, entry, opts)
	return err
}

// SetInventoryCount overwrites the count of an inventory entry.
func (coll InventoryCollection) SetInventoryCount(id string, count int64) error {
	l := Logger()
	now := bson.NewDateTimeFromTime(time.Now())
	_, err := coll.UpdateOne(context.TODO(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"count": count, "updated": now}},
	)
	if err != nil {
		l.Fatalf("Could not update inventory data due to connection errors to database: %s", err.Error())
	}
	return err
}

// valueHistoryAppend is the write AppendValueHistory performs, for batching
// many of them into one BulkWrite.
func valueHistoryAppend(id string, snapshot PriceEntry, now bson.DateTime) mongo.WriteModel {
	return mongo.NewUpdateOneModel().
		SetFilter(bson.M{"_id": id}).
		SetUpdate(bson.M{
			"$push": bson.M{"value_history": snapshot},
			"$set":  bson.M{"updated": now},
		})
}

// BulkWriteInventory applies the given writes in order, in a single round
// trip.
func (coll InventoryCollection) BulkWriteInventory(models []mongo.WriteModel) error {
	l := Logger()
	if len(models) == 0 {
		return nil
	}
	_, err := coll.BulkWrite(context.TODO(), models, options.BulkWrite().SetOrdered(true))
	if err != nil {
		l.Fatalf("Could not update inventory data due to connection errors to database: %s", err.Error())
	}
	return err
}

// RemoveInventoryEntry removes an inventory entry entirely (used when its
// count drops to zero).
func (coll InventoryCollection) RemoveInventoryEntry(id string) error {
	l := Logger()
	_, err := coll.DeleteOne(context.TODO(), bson.M{"_id": id})
	if err != nil {
		l.Fatalf("Could not remove inventory data due to connection errors to database: %s", err.Error())
	}
	return err
}

// FindInventoryEntry returns a single inventory entry by its deterministic ID.
func (coll InventoryCollection) FindInventoryEntry(id string) (*InventoryEntry, error) {
	var entry InventoryEntry
	err := coll.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&entry)
	if err != nil {
		return &InventoryEntry{}, err
	}
	return &entry, nil
}

// FindInventoryEntries returns inventory entries by a given filter, sort and
// pagination options.
func (coll InventoryCollection) FindInventoryEntries(filter, sort bson.D, skip, limit int64) ([]InventoryEntry, error) {
	return coll.findInventoryEntries(filter, options.Find().SetSort(sort).SetSkip(skip).SetLimit(limit))
}

// FindInventoryEntriesWithoutHistory is FindInventoryEntries without
// pagination and with value_history left out of the result. That history
// grows with every `serra update` and makes up most of an entry, so this is
// far cheaper for callers that only need current counts.
func (coll InventoryCollection) FindInventoryEntriesWithoutHistory(filter bson.D) ([]InventoryEntry, error) {
	return coll.findInventoryEntries(filter, options.Find().SetProjection(bson.D{{"value_history", 0}}))
}

func (coll InventoryCollection) findInventoryEntries(filter bson.D, opts *options.FindOptionsBuilder) ([]InventoryEntry, error) {
	l := Logger()

	cursor, err := coll.Find(context.TODO(), filter, opts)
	if err != nil {
		l.Fatalf("Could not query inventory data due to connection errors to database: %s", err.Error())
	}

	var results []InventoryEntry
	if err = cursor.All(context.TODO(), &results); err != nil {
		l.Fatal(err)
		return []InventoryEntry{}, err
	}
	return results, nil
}

// FindInventoryEntriesBySetAndCollectorNumber returns all inventory entries
// (across finishes/languages/conditions) owned for a given set+collector
// number, without needing to know the card's Scryfall ID up front.
func (coll InventoryCollection) FindInventoryEntriesBySetAndCollectorNumber(setCode, collectorNumber string) ([]InventoryEntry, error) {
	return coll.FindInventoryEntries(bson.D{{"set", setCode}, {"collectornumber", collectorNumber}}, bson.D{}, 0, 0)
}

// DistinctCollectorNumbers returns the distinct collector numbers owned in a
// given set.
func (coll InventoryCollection) DistinctCollectorNumbers(setCode string) ([]string, error) {
	l := Logger()
	var numbers []string
	err := coll.Distinct(context.TODO(), "collectornumber", bson.M{"set": setCode}).Decode(&numbers)
	if err != nil {
		l.Fatalf("Could not query inventory data due to connection errors to database: %s", err.Error())
		return nil, err
	}
	return numbers, nil
}
