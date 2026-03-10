package serra

import (
	"context"
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

func (coll CardsCollection) storageFind(filter, sort bson.D, skip, limit int64) ([]Card, error) {
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

func (coll CardsCollection) storageRemove(filter bson.M) error {
	l := Logger()

	_, err := coll.DeleteOne(context.TODO(), filter)
	if err != nil {
		l.Fatalf("Could remove card data due to connection errors to database: %s", err.Error())
	}

	return nil
}

func (coll CardsCollection) storageAggregate(pipeline mongo.Pipeline) ([]primitive.M, error) {
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

func (coll CardsCollection) storageUpdate(filter, update bson.M) error {
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
