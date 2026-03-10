package serra

import (
	"context"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Collection Struct
// reason: https://siongui.github.io/2017/02/11/go-add-method-function-to-type-in-external-package/
type StorageClient struct {
	*mongo.Client
}

// Returns configured human readable name for
// the configured currency of the user
// HACK: ugly, rework
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

func storageConnect() StorageClient {
	l := Logger()
	uri := getMongoDBURI()

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		l.Fatalf("Could not connect to mongodb at %s", uri)
	}

	return StorageClient{client}
}

func storageDisconnect(client StorageClient) error {
	if err := client.Disconnect(context.TODO()); err != nil {
		return err
	}
	return nil
}
