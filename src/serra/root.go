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

	// Loop over different challenges
	for _, card := range cards {
		c, err := fetch(card)
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
		}
		storage_add(coll, c)
		fmt.Println(c)
	}

}
