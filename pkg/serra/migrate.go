package serra

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// toDateTime best-effort converts a decoded legacy timestamp (typically a
// primitive.DateTime) back into a primitive.DateTime, falling back to now if
// it is missing or of an unexpected type.
func toDateTime(v interface{}) primitive.DateTime {
	if dt, ok := v.(primitive.DateTime); ok {
		return dt
	}
	return primitive.NewDateTimeFromTime(time.Now())
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "One-time migration of the legacy cards collection into cards + inventory",
	Long: `Older versions of serra stored ownership data (count, value history,
added/updated timestamps) directly inside the "cards" collection, alongside
the cached Scryfall data. This command splits any such legacy documents into
a pure Scryfall "cards" collection and a new "inventory" collection (one
entry per card/finish/language/condition combination).

It is safe to run multiple times.`,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMigration()
	},
}

// legacyCard mirrors the pre-migration shape of a "cards" document, where
// ownership data lived alongside the Scryfall data.
type legacyCard struct {
	Card `bson:",inline"`

	Count       int64        `bson:"serra_count"`
	CountFoil   int64        `bson:"serra_count_foil"`
	CountEtched int64        `bson:"serra_count_etched"`
	PriceList   []PriceEntry `bson:"serra_prices"`
	Created     interface{}  `bson:"serra_created"`
	Updated     interface{}  `bson:"serra_updated"`
}

func runMigration() error {
	l := Logger()
	client := storageConnect()
	defer storageDisconnect(client)

	cardsColl := client.getCardsCollection()
	invColl := client.getInventoryCollection()

	cursor, err := cardsColl.Find(context.TODO(), bson.D{})
	if err != nil {
		l.Fatalf("Could not query cards collection: %s", err.Error())
		return err
	}

	var legacyCards []legacyCard
	if err := cursor.All(context.TODO(), &legacyCards); err != nil {
		l.Fatalf("Could not decode cards collection: %s", err.Error())
		return err
	}

	migratedCards := 0
	migratedEntries := 0

	for _, lc := range legacyCards {
		// Nothing to migrate for this doc: either already pure Scryfall
		// data, or an orphaned cache entry with no ownership at all.
		if lc.Count == 0 && lc.CountFoil == 0 && lc.CountEtched == 0 {
			continue
		}

		created := toDateTime(lc.Created)
		updated := toDateTime(lc.Updated)

		// Re-save the card as pure Scryfall data (strips the legacy
		// serra_* fields once written back).
		card := lc.Card
		if err := cardsColl.UpsertCard(&card); err != nil {
			l.Errorf("Could not migrate card %s: %s", card.ID, err.Error())
			continue
		}
		migratedCards++

		finishes := []struct {
			Finish string
			Count  int64
		}{
			{FinishNonfoil, lc.Count},
			{FinishFoil, lc.CountFoil},
			{FinishEtched, lc.CountEtched},
		}

		for _, f := range finishes {
			if f.Count <= 0 {
				continue
			}

			history := make([]PriceEntry, len(lc.PriceList))
			for i, p := range lc.PriceList {
				history[i] = priceEntryForFinish(p, f.Finish)
			}
			if len(history) == 0 {
				history = []PriceEntry{priceEntryForFinish(card.Prices, f.Finish)}
			}

			entry := InventoryEntry{
				ID:              inventoryID(card.ID, f.Finish, DefaultLanguage, DefaultCondition),
				CardID:          card.ID,
				Set:             card.Set,
				CollectorNumber: card.CollectorNumber,
				Finish:          f.Finish,
				Language:        DefaultLanguage,
				Condition:       DefaultCondition,
				Count:           f.Count,
				ValueHistory:    history,
				Created:         created,
				Updated:         updated,
			}

			if err := invColl.upsertMigratedEntry(&entry); err != nil {
				l.Errorf("Could not migrate inventory entry %s: %s", entry.ID, err.Error())
				continue
			}
			migratedEntries++
		}
	}

	l.Infof("Migration finished: %d cards, %d inventory entries", migratedCards, migratedEntries)
	return nil
}
