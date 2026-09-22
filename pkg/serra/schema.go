package serra

import (
	"context"
	"fmt"
	"time"

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

// getSchemaVersion returns the database's current schema version, and
// whether that version was actually recorded in the meta collection or is
// merely assumed. A database with no version recorded yet is assumed to
// already be at the baseline version 1 - it predates schema versioning, not
// necessarily the schema itself.
func (client StorageClient) getSchemaVersion() (int, bool, error) {
	var meta schemaMeta
	err := client.getMetaCollection().FindOne(context.TODO(), bson.M{"_id": schemaVersionDocID}).Decode(&meta)
	if err == mongo.ErrNoDocuments {
		return 1, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return meta.Version, true, nil
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

// legacyOwnershipField is the field 4.x and older stored a card's owned
// count in, directly on the cards document.
const legacyOwnershipField = "serra_count"

// legacyCheckTimeout bounds the scan hasLegacyOwnershipData performs, so no
// command can be held up by it for long.
const legacyCheckTimeout = 500 * time.Millisecond

// skipLegacyCheck suppresses the "run serra migrate" hint while the
// migration is running - there is no point telling someone to do the thing
// they are already doing.
var skipLegacyCheck bool

// hasLegacyOwnershipData reports whether the cards collection still holds
// documents in the pre-5.0 shape, where ownership data lived alongside the
// cached Scryfall data.
//
// A 4.x database carries no schema version at all, so getSchemaVersion()
// assumes it is already at the baseline and the version check stays silent.
// Looking for leftover ownership fields is what actually catches it.
//
// The question is only asked when the inventory collection is completely
// empty. That is the state a 4.x database upgrades into - its ownership data
// has not been moved yet - and it is also the only state in which the answer
// is cheap, since a legacy database matches on the first document scanned.
// Anything holding inventory entries skips the check, so the normal case
// costs one O(1) count. A partially migrated database is therefore not
// detected here; 'serra migrate' is re-runnable and reports its own errors.
//
// The scan is bounded by legacyCheckTimeout: a database that has imported
// the full Scryfall bulk file but owns nothing yet has nothing to find, and
// every command would otherwise pay for proving it.
func (client StorageClient) hasLegacyOwnershipData() bool {
	entries, err := client.getInventoryCollection().EstimatedDocumentCount(context.TODO())
	if err != nil || entries > 0 {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), legacyCheckTimeout)
	defer cancel()

	opts := options.FindOne().SetProjection(bson.M{"_id": 1})
	filter := bson.M{legacyOwnershipField: bson.M{"$exists": true}}

	return client.getCardsCollection().FindOne(ctx, filter, opts).Err() == nil
}

// CheckSchemaVersion warns if the database is behind the schema this build
// of serra expects, and fatally errors if it is ahead (a newer serra
// version has touched this database - downgrading is not supported). It also
// catches databases predating schema versioning entirely, which record no
// version to compare against.
func (client StorageClient) CheckSchemaVersion() {
	l := Logger()

	version, _, err := client.getSchemaVersion()
	if err != nil {
		l.Warnf("Could not determine database schema version: %s", err.Error())
		return
	}

	if version > CurrentSchemaVersion {
		l.Fatalf("Database schema version %d is newer than this version of serra supports (%d). Please upgrade serra.", version, CurrentSchemaVersion)
	}

	if version < CurrentSchemaVersion {
		l.Warnf("Database schema is out of date (version %d, expected %d). Run 'serra migrate' to update it.", version, CurrentSchemaVersion)
		return
	}

	if !skipLegacyCheck && client.hasLegacyOwnershipData() {
		l.Warn("Your database still stores ownership data in the pre-5.0 format, so your collection will show up as empty. Run 'serra migrate' to convert it.")
	}
}

// ApplySchemaMigrations brings the database up to CurrentSchemaVersion,
// running each pending migration in order and recording progress after each
// one, so that a failure partway through can be resumed by running it again.
func (client StorageClient) ApplySchemaMigrations() error {
	l := Logger()

	version, recorded, err := client.getSchemaVersion()
	if err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	if version > CurrentSchemaVersion {
		l.Fatalf("Database schema version %d is newer than this version of serra supports (%d). Please upgrade serra.", version, CurrentSchemaVersion)
	}

	if version == CurrentSchemaVersion {
		if recorded {
			l.Infof("Database schema is already at the latest version (%d)", version)
			return nil
		}

		// The shape is already current but nothing ever wrote it down (the
		// database predates schema versioning). There is nothing to migrate,
		// so just record it - later migrations should count from a stored
		// baseline rather than from this assumption.
		if err := client.setSchemaVersion(version); err != nil {
			return fmt.Errorf("recording schema version %d: %w", version, err)
		}
		l.Infof("Database schema is at the latest version (%d), now recorded", version)
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

	// The newest migration can be older than CurrentSchemaVersion, since a
	// version may be bumped without a step of its own. Record what this build
	// expects rather than whatever the last migration happened to be.
	if err := client.setSchemaVersion(CurrentSchemaVersion); err != nil {
		return fmt.Errorf("recording schema version %d: %w", CurrentSchemaVersion, err)
	}

	l.Infof("Database schema is now at version %d", CurrentSchemaVersion)
	return nil
}
