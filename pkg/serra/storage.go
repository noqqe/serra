package serra

import (
	"context"
	"slices"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// StorageClient is a wrapper around the mongo.Client struct to add methods for
// reason: https://siongui.github.io/2017/02/11/go-add-method-function-to-type-in-external-package/
type StorageClient struct {
	*mongo.Client
}

// storageConnect connects to the MongoDB database using the URI from
// environment variables and returns a StorageClient.
func storageConnect() StorageClient {
	l := Logger()
	uri := getMongoDBURI()

	// Card documents carry the full Scryfall payload, so reads like stats
	// move tens of MB. Compression is negotiated with the server and cuts
	// that several-fold; compressors set in the URI still take precedence.
	opts := options.Client().SetCompressors([]string{"zstd", "snappy"}).ApplyURI(uri)
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		l.Fatalf("Could not connect to mongodb at %s", uri)
	}

	storageClient := StorageClient{client}
	storageClient.CheckSchemaVersion()
	return storageClient
}

// storageDisconnect disconnects from the MongoDB database and returns an error
// if the disconnection fails.
func storageDisconnect(client StorageClient) error {
	if err := client.Disconnect(context.TODO()); err != nil {
		return err
	}
	return nil
}

// staleIndexes are indexes on fields that are never stored under that name
// (Card has no bson tags, so collector_number is stored as collectornumber).
// They can never be used by a query and only slow down writes.
var staleIndexes = map[string][]string{
	"cards": {"collector_number_1"},
}

// ensureIndexes creates the indexes the set / set+collector number lookups
// rely on and drops known stale ones. It is idempotent: creating an index
// that already exists is a no-op, and missing stale indexes are skipped.
func (client StorageClient) ensureIndexes() error {
	l := Logger()
	db := client.Database("serra")
	setNumber := mongo.IndexModel{Keys: bson.D{{"set", 1}, {"collectornumber", 1}}}

	for _, coll := range []string{"cards", "inventory"} {
		if _, err := db.Collection(coll).Indexes().CreateOne(context.TODO(), setNumber); err != nil {
			l.Fatalf("Could not create index on %s: %v", coll, err)
			return err
		}
	}

	for coll, names := range staleIndexes {
		existing, err := db.Collection(coll).Indexes().ListSpecifications(context.TODO())
		if err != nil {
			l.Fatalf("Could not list indexes of %s: %v", coll, err)
			return err
		}
		for _, spec := range existing {
			if !slices.Contains(names, spec.Name) {
				continue
			}
			if _, err := db.Collection(coll).Indexes().DropOne(context.TODO(), spec.Name); err != nil {
				l.Fatalf("Could not drop stale index %s on %s: %v", spec.Name, coll, err)
				return err
			}
			l.Infof("Dropped stale index %s on %s", spec.Name, coll)
		}
	}
	return nil
}
