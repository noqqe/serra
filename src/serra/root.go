package serra

import (
	"fmt"
)

const (
	version string = "0.0.1"
)

// Add
func Add(cards []string) {
	LogMessage(fmt.Sprintf("Archivar %v\n", version), "green")

	client := storage_connect()
	coll := client.Database("serra").Collection("cards")

	// Loop over different cards
	for _, card := range cards {

		// Fetch card from scryfall
		c, err := fetch(card)
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
		}

		// Write card to mongodb
		err = storage_add(coll, c)
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
			continue
		}

		LogMessage(fmt.Sprintf("\"%s\" (%s Eur) added to Collection.", c.Name, c.Prices.Eur), "purple")
	}

}

func List() {
	LogMessage(fmt.Sprintf("Archivar %v\n", version), "green")

	client := storage_connect()
	coll := client.Database("serra").Collection("cards")
	storage_find(coll)

}
