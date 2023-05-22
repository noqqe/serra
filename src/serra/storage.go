package serra

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Total struct {
	ID    string       `json:"id" bson:"_id"`
	Value []PriceEntry `bson:"value"`
}

// https://siongui.github.io/2017/02/11/go-add-method-function-to-type-in-external-package/
type Collection struct {
	*mongo.Collection
}

// Returns configured human readable name for
// the configured currency of the user
func getCurrencyField(foil bool) string {
	switch os.Getenv("SERRA_CURRENCY") {
	case "EUR":
		if foil {
			return "$prices.eur_foil"
		}
		return "$prices.eur"
	case "USD":
		if foil {
			return "$prices.usd_foil"
		}
		return "$prices.usd"
	default:
		if foil {
			return "$prices.usd_foil"
		}
		return "$prices.usd"
	}
}

func storageConnect() *mongo.Client {
	uri := getMongoDBURI()

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		LogMessage(fmt.Sprintf("Could not connect to mongodb at %s", uri), "red")
		os.Exit(1)
	}

	return client

}

func (coll Collection) storageAdd(card *Card) error {

	card.SerraUpdated = primitive.NewDateTimeFromTime(time.Now())

	_, err := coll.InsertOne(context.TODO(), card)
	if err != nil {
		return err
	}
	return nil

}

func (coll Collection) storageAddSet(set *Set) (*mongo.InsertOneResult, error) {

	id, err := coll.InsertOne(context.TODO(), set)
	if err != nil {
		return id, err
	}
	return id, err

}

func (coll Collection) storageAddTotal(p PriceEntry) error {

	// create total object if not exists...
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

func (coll Collection) storageFind(filter, sort bson.D) ([]Card, error) {
	opts := options.Find().SetSort(sort)
	cursor, err := coll.Find(context.TODO(), filter, opts)
	if err != nil {
		LogMessage(fmt.Sprintf("Could not query data due to connection errors to database: %s", err.Error()), "red")
		os.Exit(1)
	}

	var results []Card
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
		return []Card{}, err
	}
	return results, nil

}

func (coll Collection) storageFindSet(filter, sort bson.D) ([]Set, error) {
	opts := options.Find().SetSort(sort)

	cursor, err := coll.Find(context.TODO(), filter, opts)
	if err != nil {
		LogMessage(fmt.Sprintf("Could not query set data due to connection errors to database: %s", err.Error()), "red")
		os.Exit(1)
	}

	var results []Set
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
		return []Set{}, err
	}

	return results, nil
}

func (coll Collection) storageFindTotal() (Total, error) {
	var total Total

	err := coll.FindOne(context.TODO(), bson.D{{"_id", "1"}}).Decode(&total)
	if err != nil {
		LogMessage(fmt.Sprintf("Could not query total data due to connection errors to database: %s", err.Error()), "red")
		os.Exit(1)
	}

	return total, nil
}

func (coll Collection) storageRemove(filter bson.M) error {
	_, err := coll.DeleteOne(context.TODO(), filter)
	if err != nil {
		LogMessage(fmt.Sprintf("Could remove card data due to connection errors to database: %s", err.Error()), "red")
		os.Exit(1)
	}

	return nil
}

func (coll Collection) storageAggregate(pipeline mongo.Pipeline) ([]primitive.M, error) {
	opts := options.Aggregate()

	cursor, err := coll.Aggregate(
		context.TODO(),
		pipeline,
		opts)
	if err != nil {
		LogMessage(fmt.Sprintf("Could not aggregate data due to connection errors to database: %s", err.Error()), "red")
		os.Exit(1)
	}

	// Get a list of all returned documents and print them out.
	// See the mongo.Cursor documentation for more examples of using cursors.
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
	}

	return results, nil
}

func (coll Collection) storageUpdate(filter, update bson.M) error {
	// Call the driver's UpdateOne() method and pass filter and update to it
	_, err := coll.UpdateOne(
		context.Background(),
		filter,
		update,
	)
	if err != nil {
		LogMessage(fmt.Sprintf("Could not update data due to connection errors to database: %s", err.Error()), "red")
		os.Exit(1)
	}

	return nil
}

func storageDisconnect(client *mongo.Client) error {
	if err := client.Disconnect(context.TODO()); err != nil {
		return err
	}
	return nil
}
