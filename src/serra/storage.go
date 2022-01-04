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

// https://siongui.github.io/2017/02/11/go-add-method-function-to-type-in-external-package/
type Collection struct {
	*mongo.Collection
}

func storage_connect() *mongo.Client {

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("You must set your 'MONGODB_URI' environmental variable. See\n\t https://docs.mongodb.com/drivers/go/current/usage-examples/#environment-variable")
	}

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	return client
}

func (coll Collection) storage_add(card *Card) error {

	card.SerraUpdated = primitive.NewDateTimeFromTime(time.Now())

	_, err := coll.InsertOne(context.TODO(), card)
	if err != nil {
		return err
	}
	return nil

}

func (coll Collection) storage_find(filter, sort bson.D) ([]Card, error) {

	opts := options.Find().SetSort(sort)
	cursor, err := coll.Find(context.TODO(), filter, opts)
	if err != nil {
		log.Fatal(err)
	}

	var results []Card
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
		return []Card{}, err
	}
	return results, nil

}

func (coll Collection) storage_remove(filter bson.M) error {

	_, err := coll.DeleteOne(context.TODO(), filter)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil

}

func (coll Collection) storage_aggregate(groupstage bson.D) ([]primitive.M, error) {

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

func (coll Collection) storage_update(filter, update bson.M) error {

	// Call the driver's UpdateOne() method and pass filter and update to it
	_, err := coll.UpdateOne(
		context.Background(),
		filter,
		update,
	)
	if err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

func storage_disconnect(client *mongo.Client) error {
	if err := client.Disconnect(context.TODO()); err != nil {
		return err
	}
	return nil
}
