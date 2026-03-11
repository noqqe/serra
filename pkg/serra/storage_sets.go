package serra

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SetList struct {
	Data []Set `json:"data"`
}

type Set struct {
	PriceList []PriceEntry       `bson:"serra_prices"`
	Created   primitive.DateTime `bson:"serra_created"`
	Updated   primitive.DateTime `bson:"serra_updated"`
	CardCount int64              `json:"card_count" bson:"cardcount"`

	Code        string `json:"code"`
	Digital     bool   `json:"digital"`
	FoilOnly    bool   `json:"foil_only"`
	IconSvgURI  string `json:"icon_svg_uri"`
	ID          string `json:"id" bson:"_id"`
	Name        string `json:"name"`
	NonfoilOnly bool   `json:"nonfoil_only"`
	Object      string `json:"object"`
	ReleasedAt  string `json:"released_at"`
	ScryfallURI string `json:"scryfall_uri"`
	SearchURI   string `json:"search_uri"`
	SetType     string `json:"set_type"`
	TcgplayerID int64  `json:"tcgplayer_id"`
	URI         string `json:"uri"`
}

type SetsCollection struct {
	*mongo.Collection
}

func (client StorageClient) getSetsCollection() SetsCollection {
	return SetsCollection{client.Database("serra").Collection("sets")}
}

// AddSet adds a set to the collection. If the set already exists, an error is returned.
func (coll SetsCollection) AddSet(set *Set) (*mongo.InsertOneResult, error) {
	id, err := coll.InsertOne(context.TODO(), set)
	if err != nil {
		return id, err
	}
	return id, err
}

// RemoveSet removes a set from the collection. If the set does not exist, an error is returned.
func (coll SetsCollection) RemoveSet(set *Set) error {
	l := Logger()

	filter := bson.M{"_id": set.ID}
	_, err := coll.DeleteOne(context.TODO(), filter)
	if err != nil {
		l.Fatalf("Could remove set due to connection errors to database: %s", err.Error())
		return err
	}

	return nil
}

// FindSet returns a list of sets by a given filter and sort options.
func (coll SetsCollection) FindSet(filter, sort bson.D) ([]Set, error) {
	l := Logger()
	opts := options.Find().SetSort(sort)

	cursor, err := coll.Find(context.TODO(), filter, opts)
	if err != nil {
		l.Fatalf("Could not query set data due to connection errors to database: %s", err.Error())
	}

	var results []Set
	if err = cursor.All(context.TODO(), &results); err != nil {
		l.Fatal(err)
		return []Set{}, err
	}

	return results, nil
}

// FindSetByCode finds a set in the collection by its code. Returns an error if not found
func (coll SetsCollection) FindSetByCode(setcode string) (*Set, error) {
	storedSets, err := coll.FindSet(bson.D{{"code", setcode}}, bson.D{{"_id", 1}})
	if err != nil {
		return &Set{}, err
	}

	if len(storedSets) < 1 {
		return &Set{}, errors.New("Set not found")
	}

	return &storedSets[0], nil
}

func (coll SetsCollection) UpdateSet(filter, update bson.M) error {
	l := Logger()
	// Call the driver's UpdateOne() method and pass filter and update to it
	_, err := coll.UpdateOne(
		context.Background(),
		filter,
		update,
	)
	if err != nil {
		l.Fatalf("Could not update data due to connection errors to database: %s", err.Error())
	}

	return nil
}

func (coll SetsCollection) AggregateSet(pipeline mongo.Pipeline) ([]primitive.M, error) {
	l := Logger()
	opts := options.Aggregate()

	cursor, err := coll.Aggregate(
		context.TODO(),
		pipeline,
		opts)
	if err != nil {
		l.Fatalf("Could not aggregate data due to connection errors to database: %s", err.Error())
		return []primitive.M{}, err
	}

	// Get a list of all returned documents and print them out.
	// See the mongo.Cursor documentation for more examples of using cursors.
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		l.Fatal(err)
		return []primitive.M{}, err
	}

	return results, nil
}
