package serra

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func storage_connect() *mongo.Client {

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("You must set your 'MONGODB_URI' environmental variable. See\n\t https://docs.mongodb.com/drivers/go/current/usage-examples/#environment-variable")
	}

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	// defer func() {
	// 	if err := client.Disconnect(context.TODO()); err != nil {
	// 		panic(err)
	// 	}
	// }()

	return client
}

func storage_add(coll *mongo.Collection, card *Card) error {

	_, err := coll.InsertOne(context.TODO(), card)
	if err != nil {
		return err
	}
	return nil

}

func storage_find(coll *mongo.Collection) ([]Card, error) {

	opts := options.Find().SetSort(bson.D{{"collectornumber", 1}})
	cursor, err := coll.Find(context.TODO(), bson.D{{}}, opts)
	if err != nil {
		log.Fatal(err)
	}

	// Get a list of all returned documents and print them out.
	// See the mongo.Cursor documentation for more examples of using cursors.
	var results []Card
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
		return []Card{}, err
	}
	return results, nil

}

func storage_aggregate(coll *mongo.Collection, groupstage bson.D) ([]primitive.M, error) {

	// db.cards.aggregate([ {$group: { _id: "$setname", sum: { $sum: "$prices.eur"}}}])
	opts := options.Aggregate().SetMaxTime(2 * time.Second)
	cursor, err := coll.Aggregate(
		context.TODO(),
		mongo.Pipeline{groupstage},
		opts)
	if err != nil {
		log.Fatal(err)
	}

	// Get a list of all returned documents and print them out.
	// See the mongo.Cursor documentation for more examples of using cursors.
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
	}
	return results, nil

}
