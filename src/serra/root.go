package serra

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

const (
	version string = "0.0.1"
)

// Add
func Add(cards []string) {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := client.Database("serra").Collection("cards")

	// Loop over different cards
	for _, card := range cards {
		// Fetch card from scryfall
		c, err := fetch_card(card)
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
			continue
		}

		// Write card to mongodb
		err = storage_add(coll, c)
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
			continue
		}

		LogMessage(fmt.Sprintf("\"%s\" (%.2f Eur) added to Collection.", c.Name, c.Prices.Eur), "purple")
	}
	storage_disconnect(client)

}

func Cards() {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := client.Database("serra").Collection("cards")

	sort := bson.D{{"collectornumber", 1}}
	filter := bson.D{{}}
	cards, _ := storage_find(coll, filter, sort)

	for _, card := range cards {
		fmt.Printf("%s (%s) %.2f\n", card.Name, card.Set, card.Prices.Eur)
	}
	storage_disconnect(client)
}

func Sets() {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := client.Database("serra").Collection("cards")

	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", "$setname"},
			{"sum", bson.D{
				{"$sum", "$prices.eur"},
			}},
		}},
	}

	sets, _ := storage_aggregate(coll, groupStage)
	for _, set := range sets {
		fmt.Printf("* %s (%.2f Eur)\n", set["_id"], set["sum"])
	}
	storage_disconnect(client)

}

func Update() {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := client.Database("serra").Collection("cards")

	sort := bson.D{{"_id", 1}}
	filter := bson.D{{"_id", "0d4f3c1d-d25e-4263-ab2b-19534c852678"}}
	cards, _ := storage_find(coll, filter, sort)

	for _, card := range cards {
		fmt.Printf("%s (%s) %.2f\n", card.Name, card.Set, card.Prices.Eur)

		// db.cards.update({'_id':'8fa2ecf9-b53c-4f1d-9028-ca3820d043cb'},{$set:{'serra_updated':ISODate("2021-11-02T09:28:56.504Z")}, $push: {"serra_prices": { date: ISODate("2021-11-02T09:28:56.504Z"), value: 0.1 }}});

		// Declare an _id filter to get a specific MongoDB document
		filter := bson.M{"_id": bson.M{"$eq": card.ID}}

		// Declare a filter that will change a field's integer value to `42`
		update := bson.M{"$set": bson.M{"textless": true}}
	}

	storage_disconnect(client)
}
