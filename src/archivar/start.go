package archivar

import (
	"fmt"
	"time"
)

const (
	version string = "0.0.1"
)

// Main Wrapper for a set of tests
func Start(set_file string) {

	var s Setfile
	s.ReadFile(set_file)

	LogMessage(fmt.Sprintf("Archivar %v\n", version), "green")

	fmt.Printf("Your Challenge: %s\n", s.Description)

	// Loop over different challenges
	for _, entry := range s.Cards {
		card := fetch(entry)
		fmt.Println(card)
		time.Sleep(100 * time.Millisecond)
	}

}
