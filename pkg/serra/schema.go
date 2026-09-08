package serra

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CurrentSchemaVersion is the schema version this build of serra expects the
// database to be at. Bump it whenever a migration is appended to
// schemaMigrations.
const CurrentSchemaVersion = 1

// schemaMigration describes a single, ordered step that brings the database
// from one schema version to the next. Up must be idempotent, since
// ApplySchemaMigrations may be re-run after a partial failure.
type schemaMigration struct {
	Version     int
	Description string
	Up          func(StorageClient) error
}

// schemaMigrations lists every migration in ascending version order.
// Version 1 is the baseline this versioning scheme was introduced with (the
// cards+inventory split, with price history tracked for all cards) and has
// no migration attached to it - it's simply where the counting starts.
//
// Append future migrations here, e.g.:
//
//	{Version: 2, Description: "...", Up: migrateToV2},
var schemaMigrations = []schemaMigration{}

const schemaVersionDocID = "schema_version"

type schemaMeta struct {
	ID      string `bson:"_id"`
	Version int    `bson:"version"`
}

func (client StorageClient) getMetaCollection() *mongo.Collection {
	return client.Database("serra").Collection("meta")
}

// getSchemaVersion returns the database's current schema version. A database
// with no version recorded yet is assumed to already be at the baseline
// version 1 - it predates schema versioning, not necessarily the schema
// itself.
func (client StorageClient) getSchemaVersion() (int, error) {
	var meta schemaMeta
	err := client.getMetaCollection().FindOne(context.TODO(), bson.M{"_id": schemaVersionDocID}).Decode(&meta)
	if err == mongo.ErrNoDocuments {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	return meta.Version, nil
}

// setSchemaVersion records the database's schema version.
func (client StorageClient) setSchemaVersion(version int) error {
	_, err := client.getMetaCollection().UpdateOne(
		context.TODO(),
		bson.M{"_id": schemaVersionDocID},
		bson.M{"$set": bson.M{"version": version}},
		options.Update().SetUpsert(true),
	)
	return err
}

// CheckSchemaVersion warns if the database is behind the schema this build
// of serra expects, and fatally errors if it is ahead (a newer serra
// version has touched this database - downgrading is not supported).
func (client StorageClient) CheckSchemaVersion() {
	l := Logger()

	version, err := client.getSchemaVersion()
	if err != nil {
		l.Warnf("Could not determine database schema version: %s", err.Error())
		return
	}

	if version > CurrentSchemaVersion {
		l.Fatalf("Database schema version %d is newer than this version of serra supports (%d). Please upgrade serra.", version, CurrentSchemaVersion)
	}

	if version < CurrentSchemaVersion {
		l.Warnf("Database schema is out of date (version %d, expected %d). Run 'serra migrate' to update it.", version, CurrentSchemaVersion)
	}
}

// ApplySchemaMigrations brings the database up to CurrentSchemaVersion,
// running each pending migration in order and recording progress after each
// one, so that a failure partway through can be resumed by running it again.
func (client StorageClient) ApplySchemaMigrations() error {
	l := Logger()

	version, err := client.getSchemaVersion()
	if err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	if version > CurrentSchemaVersion {
		l.Fatalf("Database schema version %d is newer than this version of serra supports (%d). Please upgrade serra.", version, CurrentSchemaVersion)
	}

	if version == CurrentSchemaVersion {
		l.Infof("Database schema is already at the latest version (%d)", version)
		return nil
	}

	for _, m := range schemaMigrations {
		if m.Version <= version {
			continue
		}

		l.Infof("Migrating database schema to version %d: %s", m.Version, m.Description)
		if err := m.Up(client); err != nil {
			return fmt.Errorf("migrating to schema version %d: %w", m.Version, err)
		}
		if err := client.setSchemaVersion(m.Version); err != nil {
			return fmt.Errorf("recording schema version %d: %w", m.Version, err)
		}
	}

	l.Infof("Database schema is now at version %d", CurrentSchemaVersion)
	return nil
}
