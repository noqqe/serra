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

// GetSetByCode returns a set from the set list by its code. If the set is not found, an error is returned.
func (s SetList) GetSetByCode(code string) *Set {
	setList := s.Data
	for i := range setList {
		if setList[i].Code == code {
			return &setList[i]
		}
	}
	return nil
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

// UpsertSet replaces a set document outright, inserting it if it does not
// exist yet, keyed by its Scryfall ID. Used by update to write back a set's
// refreshed data plus appended price history in a single atomic step.
func (coll SetsCollection) UpsertSet(set *Set) error {
	l := Logger()

	_, err := coll.ReplaceOne(context.TODO(), bson.M{"_id": set.ID}, set, options.Replace().SetUpsert(true))
	if err != nil {
		l.Fatalf("Could not upsert set due to connection errors to database: %s", err.Error())
	}
	return err
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
