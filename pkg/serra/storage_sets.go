package serra

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SetsCollection struct {
	*mongo.Collection
}

func (client StorageClient) getSetsCollection() SetsCollection {
	return SetsCollection{client.Database("serra").Collection("sets")}
}

func (coll SetsCollection) storageAddSet(set *Set) (*mongo.InsertOneResult, error) {

	id, err := coll.InsertOne(context.TODO(), set)
	if err != nil {
		return id, err
	}
	return id, err

}

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
