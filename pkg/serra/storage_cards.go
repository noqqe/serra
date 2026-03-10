package serra

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CardsCollection struct {
	*mongo.Collection
}

func (client StorageClient) getCardsCollection() CardsCollection {
	return CardsCollection{client.Database("serra").Collection("cards")}
}

func (coll CardsCollection) storageAdd(card *Card) error {

	card.SerraUpdated = primitive.NewDateTimeFromTime(time.Now())

	_, err := coll.InsertOne(context.TODO(), card)
	if err != nil {
		return err
	}
	return nil

}

func (coll CardsCollection) FindCards(filter, sort bson.D, skip, limit int64) ([]Card, error) {
	opts := options.Find().SetSort(sort).SetSkip(skip).SetLimit(limit)
	cursor, err := coll.Find(context.TODO(), filter, opts)
	l := Logger()

	if err != nil {
		l.Fatalf("Could not query data due to connection errors to database: %s", err.Error())
	}

	var results []Card
	if err = cursor.All(context.TODO(), &results); err != nil {
		l.Fatal(err)
		return []Card{}, err
	}
	return results, nil

}

// FindCardByCollectorNumber returns a card by set code and collector number.
// If no card is found, an error is returned.
func (coll CardsCollection) FindCardByCollectorNumber(setCode string, collectorNumber string) (*Card, error) {
	sort := bson.D{{"_id", 1}}
	searchFilter := bson.D{{"set", setCode}, {"collectornumber", collectorNumber}}
	storedCards, err := coll.FindCards(searchFilter, sort, 0, 0)
	if err != nil {
		return &Card{}, err
	}

	if len(storedCards) < 1 {
		return &Card{}, errors.New("Card not found")
	}

	return &storedCards[0], nil
}

// RemoveCards removes cards from the collection by a given filter. If no card
// is found, an error is returned.
func (coll CardsCollection) RemoveCards(filter bson.M) error {
	l := Logger()

	_, err := coll.DeleteOne(context.TODO(), filter)
	if err != nil {
		l.Fatalf("Could remove card data due to connection errors to database: %s", err.Error())
	}

	return nil
}

// AggregateCards aggregates cards in the collection by a given pipeline.
func (coll CardsCollection) AggregateCards(pipeline mongo.Pipeline) ([]primitive.M, error) {
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

func (coll CardsCollection) UpdateCards(filter, update bson.M) error {
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

// ModifyCardCount modifies the amount of a card in the collection by a given
// amount in foil or nonfoil
func (coll CardsCollection) ModifyCardCount(c *Card, amount int64, foil bool) error {

	l := Logger()
	storedCard, err := coll.FindCardByCollectorNumber(c.Set, c.CollectorNumber)
	if err != nil {
		return err
	}

	// update card amount
	var update bson.M
	if foil {
		update = bson.M{
			"$set": bson.M{"serra_count_foil": storedCard.SerraCountFoil + amount},
		}
	} else {
		update = bson.M{
			"$set": bson.M{"serra_count": storedCard.SerraCount + amount},
		}
	}

	coll.UpdateCards(bson.M{"_id": bson.M{"$eq": c.ID}}, update)

	var total int64
	if foil {
		total = storedCard.SerraCountFoil + amount
		if amount < 0 {
			l.Warnf("Reduced card amount of \"%s\" (%.2f%s, foil) from %d to %d", storedCard.Name, storedCard.getFoilValue(), getCurrency(), storedCard.SerraCountFoil, total)
		} else {
			l.Warnf("Increased card amount of \"%s\" (%.2f%s, foil) from %d to %d", storedCard.Name, storedCard.getFoilValue(), getCurrency(), storedCard.SerraCountFoil, total)
		}
	} else {
		total = storedCard.SerraCount + amount
		if amount < 0 {
			l.Warnf("Reduced card amount of \"%s\" (%.2f%s) from %d to %d", storedCard.Name, storedCard.getValue(), getCurrency(), storedCard.SerraCount, total)
		} else {
			l.Warnf("Increased card amount of \"%s\" (%.2f%s) from %d to %d", storedCard.Name, storedCard.getValue(), getCurrency(), storedCard.SerraCount, total)
		}
	}

	return nil
}
