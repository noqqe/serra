package serra

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Aliases:       []string{"u"},
	Use:           "update",
	Short:         "update card values from scryfall",
	Long:          `the update mechanism iterates over each card in your collection and fetches its price. after all cards you own in a set are updated, the set value will update. after all sets are updated, the whole collection value is updated.`,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, setCodes []string) error {

		l := Logger()

		if len(setCodes) > 0 {
			l.Infof("Starting update for sets: %s", Yellow("%s", strings.Join(setCodes, ", ")))
		} else {
			l.Info("Starting update for whole collection")
		}

		updatedSets, err := fetchSets()
		if err != nil {
			l.Error("Could not fetch updated sets:", err)
			l.Error("Exiting")
			return err
		}

		updatedCards, err := fetchUpdatedCards()
		if err != nil {
			log.Error("Could not fetch updated cards:", err)
			log.Error("Exiting")
			return err
		}

		if err := importBulkCards(updatedCards); err != nil {
			log.Error("Could not import bulk cards:", err)
			log.Error("Exiting")
			return err
		}

		if len(setCodes) > 0 {
			for _, setCode := range setCodes {
				updateCardsOfSet(setCode, updatedSets.GetSetByCode(setCode), updatedCards)
				updateSet(setCode, updatedSets.GetSetByCode(setCode))
			}
		} else {
			for _, set := range updatedSets.Data {
				updateCardsOfSet(set.Code, updatedSets.GetSetByCode(set.Code), updatedCards)
				updateSet(set.Code, updatedSets.GetSetByCode(set.Code))
			}
			updateTotal()

		}
		l.Info("Update finished successfully")
		return nil
	},
}

func fetchUpdatedCards() ([]Card, error) {
	l := Logger()
	updatedCards := []Card{}

	// Fetch bulk file
	l.Info("Fetching bulk data from scryfall...")
	downloadURL, err := fetchBulkDownloadURL()
	if err != nil {
		l.Error("Could not extract bulk download URL:", err)
		return updatedCards, err
	}
	l.Infof("Found latest bulkfile url: %s", downloadURL)

	l.Info("Downloading bulk data file...")
	bulkFilePath, err := downloadBulkData(downloadURL)
	if err != nil {
		l.Error("Could not fetch bulk json from scryfall", err)
		return updatedCards, err
	}

	l.Info("Loading bulk data file...")
	updatedCards, err = loadBulkFile(bulkFilePath)
	if err != nil {
		l.Error("Could not load bulk file:", err)
		return updatedCards, err
	}
	l.Infof("Successfully loaded %d cards. Starting Update.", len(updatedCards))

	return updatedCards, nil
}

// importBulkCards caches every card from the Scryfall bulk file into the
// cards collection, appending a price snapshot to each - regardless of
// whether the card is owned. This is what allows price history to be
// tracked for cards outside of your collection.
func importBulkCards(updatedCards []Card) error {
	l := Logger()
	client := storageConnect()
	defer storageDisconnect(client)

	l.Infof("Importing %d cards from bulk data...", len(updatedCards))
	if err := client.getCardsCollection().UpsertCards(updatedCards); err != nil {
		return err
	}
	l.Info("Finished importing bulk cards.")
	return nil
}

// updateCardsOfSet refreshes the cached Scryfall data for every card owned
// in a set, and appends a new value snapshot to every inventory entry of
// that set.
func updateCardsOfSet(setCode string, updatedSet *Set, updatedCards []Card) error {
	client := storageConnect()
	l := Logger()
	defer storageDisconnect(client)

	invColl := client.getInventoryCollection()

	// fetch all inventory entries owned in this set
	entries, _ := invColl.FindInventoryEntries(bson.D{{"set", setCode}}, bson.D{}, 0, 0)

	// if no cards in collection for this set, skip it
	if len(entries) == 0 {
		return errors.New("no cards in collection for this set, skipping update")
	}

	bar := progressbar.NewOptions(len(entries),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetDescription(fmt.Sprintf("%s, %s\t", updatedSet.ReleasedAt[0:4], Yellow(updatedSet.Code))),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "|",
			BarEnd:        "| " + updatedSet.Name,
		}),
	)

	now := primitive.NewDateTimeFromTime(time.Now())

	for _, entry := range entries {
		bar.Add(1)

		// fetch fresh scryfall data from bulk file (already cached into the
		// cards collection by importBulkCards for the whole bulk file)
		updatedCard, err := getCardFromBulk(updatedCards, setCode, entry.CollectorNumber)
		if err != nil {
			l.Error(err)
			continue
		}

		// Scryfall occasionally re-keys a printing's ID. If that happened,
		// migrate this inventory entry to the new card ID so it keeps
		// pointing at valid Scryfall data.
		if updatedCard.ID != entry.CardID {
			invColl.RemoveInventoryEntry(entry.ID)
			entry.CardID = updatedCard.ID
			entry.ID = inventoryID(updatedCard.ID, entry.Finish, entry.Language, entry.Condition)
			invColl.InsertInventoryEntry(&entry)
		}

		// append finish-specific value snapshot
		snapshot := priceEntryForFinish(updatedCard.Prices, entry.Finish)
		snapshot.Date = now
		invColl.AppendValueHistory(entry.ID, snapshot)
	}
	fmt.Println()

	return nil
}

func updateSet(setCode string, updatedSet *Set) error {
	l := Logger()

	if setCode != updatedSet.Code {
		return errors.New("set code mismatch between stored set and updated set")
	}

	client := storageConnect()
	setscoll := client.getSetsCollection()
	defer storageDisconnect(client)

	// fetch set from database for its price history/created timestamp; a
	// zero-value Set{} (not found - this is a brand-new set) is fine to
	// build on, UpsertSet below will insert it.
	storedSet, _ := setscoll.FindSetByCode(setCode)

	owned, err := OwnedCards(bson.D{{"set", updatedSet.Code}})
	if err != nil || len(owned) == 0 {
		return fmt.Errorf("fetching set stats was not possible for set %s", setCode)
	}

	var eur, eurfoil, usd, usdfoil float64
	for _, c := range owned {
		eur += c.Prices.Eur * float64(c.Count)
		usd += c.Prices.Usd * float64(c.Count)
		eurfoil += c.Prices.EurFoil * float64(c.CountFoil)
		usdfoil += c.Prices.UsdFoil * float64(c.CountFoil)
		// Etched cards have no dedicated history slot on sets/total, fold
		// their value into the foil bucket as the closest analogue.
		usdfoil += c.Prices.UsdEtched * float64(c.CountEtched)
	}

	priceEntry := PriceEntry{
		Date:    primitive.NewDateTimeFromTime(time.Now()),
		Eur:     eur,
		EurFoil: eurfoil,
		Usd:     usd,
		UsdFoil: usdfoil,
	}

	// extend set price list with new value entry
	updatedSet.PriceList = append(storedSet.PriceList, priceEntry)

	// set timestamp
	updatedSet.Created = storedSet.Created
	updatedSet.Updated = primitive.NewDateTimeFromTime(time.Now())

	if err := setscoll.UpsertSet(updatedSet); err != nil {
		l.Error("Could not upsert set during update, skipping set update:", err)
		return err
	}

	return nil
}

func updateTotal() error {
	l := Logger()
	client := storageConnect()
	totalcoll := client.getTotalCollection()
	defer storageDisconnect(client)

	owned, err := OwnedCards(bson.D{})
	if err != nil {
		l.Error("Could not update total value of collection:", err)
		return err
	}

	var eur, eurfoil, usd, usdfoil float64
	for _, c := range owned {
		eur += c.Prices.Eur * float64(c.Count)
		usd += c.Prices.Usd * float64(c.Count)
		eurfoil += c.Prices.EurFoil * float64(c.CountFoil)
		usdfoil += c.Prices.UsdFoil * float64(c.CountFoil)
		usdfoil += c.Prices.UsdEtched * float64(c.CountEtched)
	}

	t := PriceEntry{
		Date:    primitive.NewDateTimeFromTime(time.Now()),
		Eur:     eur,
		EurFoil: eurfoil,
		Usd:     usd,
		UsdFoil: usdfoil,
	}

	total := t.Eur + t.EurFoil
	if getCurrency() != EUR {
		total = t.Usd + t.UsdFoil
	}
	l.Infof("Updating total value of collection to: %s%s\n", Yellow("%.02f", total), Yellow(getCurrency()))

	err = totalcoll.AddTotal(t)
	if err != nil {
		log.Error("Could not update total value of collection:", err)
		return err
	}

	return nil
}
