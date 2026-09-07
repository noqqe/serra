package serra

import (
	"context"

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

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		l.Fatalf("Could not connect to mongodb at %s", uri)
	}

	return StorageClient{client}
}

// storageDisconnect disconnects from the MongoDB database and returns an error
// if the disconnection fails.
func storageDisconnect(client StorageClient) error {
	if err := client.Disconnect(context.TODO()); err != nil {
		return err
	}
	return nil
}
