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

	// Loop over different challenges
	for _, card := range cards {
		c, err := fetch(card)
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
		}
		fmt.Println(c)
	}

}
