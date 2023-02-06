package serra

import (
	"log"
	"os"
)

func getMongoDBURI() string {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("You must set your 'MONGODB_URI' environmental variable. See\n\t https://docs.mongodb.com/drivers/go/current/usage-examples/#environment-variable")
	}
	return uri
}

// Returns configured human readable name for
// the configured currency of the user
func getCurrency() string {
	switch os.Getenv("SERRA_CURRENCY") {
	case "EUR":
		return "EUR"
	case "USD":
		return "USD"
	}
	// default
	LogMessage("Warning: You did not configure SERRA_CURRENCY. Assuming \"USD\"", "yellow")
	return "USD"
}
