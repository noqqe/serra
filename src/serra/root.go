package serra

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	version string = "0.0.1"
)

// Add
func Add(cards []string) {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}

	// Loop over different cards
	for _, card := range cards {
		// Fetch card from scryfall
		c, err := fetch_card(card)
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
			continue
		}

		// Write card to mongodb
		err = coll.storage_add(c)

		// If duplicate key, increase count of card
		if mongo.IsDuplicateKeyError(err) {
			increase_count_of_card(coll, c)
			continue
		}

		// If error, print error and continue
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
			continue
		}

		// Give feedback of successfully added card
		LogMessage(fmt.Sprintf("\"%s\" (%.2f Eur) added to Collection.", c.Name, c.Prices.Eur), "green")
	}
	storage_disconnect(client)
}

func Cards() {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}

	sort := bson.D{{"collectornumber", 1}}
	filter := bson.D{{}}
	cards, _ := coll.storage_find(filter, sort)

	for _, card := range cards {
		fmt.Printf("%s (%s) %.2f\n", card.Name, card.Set, card.Prices.Eur)
	}
	storage_disconnect(client)
}

func Sets() {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}

	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", "$setname"},
			{"value", bson.D{{"$sum", bson.D{{"$multiply", bson.A{"$prices.eur", "$serra_count"}}}}}},
			{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
		}},
	}

	sets, _ := coll.storage_aggregate(groupStage)
	for _, set := range sets {
		fmt.Printf("* %s (%.2f Eur) %.0f\n", set["_id"], set["value"], set["count"])
	}
	storage_disconnect(client)

}

func ShowSet(setname string) error {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}

	sort := bson.D{{"collectornumber", 1}}
	filter := bson.D{{"set", setname}}
	cards, err := coll.storage_find(filter, sort)
	if (err != nil) || len(cards) == 0 {
		LogMessage(fmt.Sprintf("Error: Set %s not found or no card in your collection.", setname), "red")
		return err
	}

	// print
	fmt.Printf("%s\n", cards[0].SetName)
	for _, card := range cards {
		fmt.Printf("%dx %d %s %.2f\n", card.SerraCount, card.CollectorNumber, card.Name, card.Prices.Eur)
	}
	storage_disconnect(client)
	return nil

}

func Update() {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}

	sort := bson.D{{"_id", 1}}
	filter := bson.D{{}}
	cards, _ := coll.storage_find(filter, sort)

	for i, card := range cards {
		fmt.Printf("Updating (%d/%d): %s (%s)...\n", i+1, len(cards), card.Name, card.SetName)

		// TODO fetch new card

		updated_card, err := fetch_card(fmt.Sprintf("%s/%d", card.Set, card.CollectorNumber))
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
			continue
		}

		filter := bson.M{"_id": bson.M{"$eq": card.ID}}

		update := bson.M{
			"$set": bson.M{"serra_updated": primitive.NewDateTimeFromTime(time.Now()), "prices": updated_card.Prices},
			"$push": bson.M{"serra_prices": bson.M{"date": primitive.NewDateTimeFromTime(time.Now()),
				"value": updated_card.Prices.Eur}}}

		coll.storage_update(filter, update)
	}

	storage_disconnect(client)
}
