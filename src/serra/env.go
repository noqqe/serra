package serra

import (
	"os"
)

const EUR = "€"
const USD = "$"

func getMongoDBURI() string {
	l := Logger()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		l.Fatal("You must set your 'MONGODB_URI' environmental variable. See\n\t https://docs.mongodb.com/drivers/go/current/usage-examples/#environment-variable")
	}

	return uri
}

// Returns configured human readable name for
// the configured currency of the user
func getCurrency() string {
	l := Logger()
	switch os.Getenv("SERRA_CURRENCY") {
	case "EUR":
		return EUR
	case "USD":
		return USD
	default:
		l.Warn("Warning: You did not configure SERRA_CURRENCY. Assuming \"USD\"")
		return "$"
	}
}
