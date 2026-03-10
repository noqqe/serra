package serra

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Total struct {
	ID    string       `json:"id" bson:"_id"`
	Value []PriceEntry `bson:"value"`
}

type TotalCollection struct {
	*mongo.Collection
}

func (client StorageClient) getTotalCollection() TotalCollection {
	return TotalCollection{client.Database("serra").Collection("total")}
}

// AddTotal adds a price entry to the total collection
func (coll TotalCollection) AddTotal(p PriceEntry) error {
	// create total object if not exists...
	// HACK: Make this only create an entry if none exists...
	coll.InsertOne(context.TODO(), Total{ID: "1", Value: []PriceEntry{}})

	// update object as intended...
	filter := bson.D{{"_id", "1"}}
	update := bson.M{"$push": bson.M{"value": p}}

	_, err := coll.UpdateOne(
		context.Background(),
		filter,
		update,
	)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// FindTotal returns the total value of the collection
func (coll TotalCollection) FindTotal() (Total, error) {
	var total Total
	l := Logger()

	err := coll.FindOne(context.TODO(), bson.D{{"_id", "1"}}).Decode(&total)
	if err != nil {
		l.Fatalf("Could not query total data due to connection errors to database: %s", err.Error())
	}

	return total, nil
}
