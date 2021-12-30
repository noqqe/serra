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
		c, err := fetch(card)
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

}

func List() {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := client.Database("serra").Collection("cards")
	cards, _ := storage_find(coll)
	for _, card := range cards {
		fmt.Printf("%s (%s) %.2f\n", card.Name, card.Set, card.Prices.Eur)
	}
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

}
