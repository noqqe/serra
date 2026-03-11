package serra

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
	RunE: func(cmd *cobra.Command, args []string) error {

		client := storageConnect()
		l := Logger()
		defer storageDisconnect(client)

		// update sets
		setscoll := client.getSetsCollection()
		coll := client.getCardsCollection()
		totalcoll := client.getTotalCollection()

		// predefine query for set analysis. used for total stats later
		projectStage := bson.D{{"$project",
			bson.D{
				{"serra_count", true},
				{"serra_count_foil", true},
				{"set", true},
				{"last_price", bson.D{{"$arrayElemAt", bson.A{"$serra_prices", -1}}}}}}}
		groupStage := bson.D{
			{"$group", bson.D{
				{"_id", ""},
				{"eur", bson.D{{"$sum", bson.D{{"$multiply", bson.A{"$last_price.eur", "$serra_count"}}}}}},
				{"eurfoil", bson.D{{"$sum", bson.D{{"$multiply", bson.A{"$last_price.eur_foil", "$serra_count_foil"}}}}}},
				{"usd", bson.D{{"$sum", bson.D{{"$multiply", bson.A{"$last_price.usd", "$serra_count"}}}}}},
				{"usdfoil", bson.D{{"$sum", bson.D{{"$multiply", bson.A{"$last_price.usd_foil", "$serra_count_foil"}}}}}},
			}}}

		// Fetch bulk file
		l.Info("Fetching bulk data from scryfall...")
		downloadURL, err := fetchBulkDownloadURL()
		if err != nil {
			l.Error("Could not extract bulk download URL:", err)
			return err
		}
		l.Infof("Found latest bulkfile url: %s", downloadURL)

		l.Info("Downloading bulk data file...")
		bulkFilePath, err := downloadBulkData(downloadURL)
		if err != nil {
			l.Error("Could not fetch bulk json from scryfall", err)
			return err
		}

		l.Info("Loading bulk data file...")
		updatedCards, err := loadBulkFile(bulkFilePath)
		if err != nil {
			l.Error("Could not load bulk file:", err)
			return err
		}
		l.Infof("Successfully loaded %d cards. Starting Update.", len(updatedCards))

		// Fetch sets from scryfall to be able to update set values after updating cards
		sets, err := fetchSets()
		if err != nil {
			l.Fatal("Could not fetch sets from scryfall:", err)
			return err
		}

		var storedSet *Set

		for _, updatedSet := range sets.Data {

			updatedSet.PriceList = []PriceEntry{}

			// fetch storedSet
			storedSet, err = setscoll.FindSetByCode(updatedSet.Code)
			if err != nil {
				setscoll.AddSet(&updatedSet)
			}

			// fetch all cards in collection for this set
			cards, _ := coll.FindCards(bson.D{{"set", updatedSet.Code}}, bson.D{{"_id", 1}}, 0, 0)

			// if no cards in collection for this set, skip it
			if len(cards) == 0 {
				continue
			}

			bar := progressbar.NewOptions(len(cards),
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

			for _, storedCard := range cards {
				bar.Add(1)

				// fetch updatedCard from bulk file
				updatedCard, err := getCardFromBulk(updatedCards, storedCard.Set, storedCard.CollectorNumber)
				if err != nil {
					l.Error(err)
					continue
				}

				// extend price entry from updatedCard with current timestamp
				updatedCard.Prices.Date = primitive.NewDateTimeFromTime(time.Now())

				// merge PriceList of storedCard with updatedCard.
				updatedCard.PriceList = storedCard.PriceList
				updatedCard.PriceList = append(updatedCard.PriceList, updatedCard.Prices)

				// set timestamp
				updatedCard.Created = storedCard.Created
				updatedCard.Updated = primitive.NewDateTimeFromTime(time.Now())

				// set count and foil count from storedCard to updatedCard
				updatedCard.Count = storedCard.Count
				updatedCard.CountFoil = storedCard.CountFoil

				// delete storedCard
				coll.RemoveCard(&storedCard)

				// add updatedCard to database
				coll.AddCard(updatedCard)

			}
			fmt.Println()

			// calculate value summary
			matchStage := bson.D{{"$match", bson.D{{"set", updatedSet.Code}}}}
			setValue, _ := coll.AggregateCards(mongo.Pipeline{matchStage, projectStage, groupStage})

			// extend set price list with new value entry
			updatedSet.PriceList = storedSet.PriceList

			// create empty priceEntry and use mapdecode to put aggregate result into PriceEntry struct
			priceEntry := PriceEntry{}
			s := setValue[0]
			priceEntry.Date = primitive.NewDateTimeFromTime(time.Now())
			mapstructure.Decode(s, &priceEntry)
			updatedSet.PriceList = append(updatedSet.PriceList, priceEntry)

			// set timestamp
			updatedSet.Created = storedSet.Created
			updatedSet.Updated = primitive.NewDateTimeFromTime(time.Now())

			setscoll.RemoveSet(storedSet)
			setscoll.AddSet(&updatedSet)

		}

		totalValue, _ := coll.AggregateCards(mongo.Pipeline{projectStage, groupStage})

		// create empty priceEntry and use mapdecode to put aggregate result into PriceEntry struct
		t := PriceEntry{}
		t.Date = primitive.NewDateTimeFromTime(time.Now())
		mapstructure.Decode(totalValue[0], &t)

		// HACK: This is here to be able to fetch currency from
		// constructed new priceentry
		tmpCard := Card{}
		tmpCard.Prices = t

		l.Infof("Updating total value of collection to: %s%s\n", Yellow("%.02f", tmpCard.getValue()+tmpCard.getFoilValue()), Yellow(getCurrency()))
		totalcoll.AddTotal(t)

		return nil
	},
}
