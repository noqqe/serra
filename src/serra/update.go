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
	Short:         "Update card values from scryfall",
	Long:          `The update mechanism iterates over each card in your collection and fetches its price. After all cards you own in a set are updated, the set value will update. After all Sets are updated, the whole collection value is updated.`,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {

		client := storageConnect()
		defer storageDisconnect(client)

		// update sets
		setscoll := &Collection{client.Database("serra").Collection("sets")}
		coll := &Collection{client.Database("serra").Collection("cards")}
		totalcoll := &Collection{client.Database("serra").Collection("total")}

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

		sets, _ := fetchSets()
		for _, set := range sets.Data {

			// When downloading new sets, PriceList needs to be initialized
			// This query silently fails if set was already downloaded. Not nice but ok for now.
			set.SerraPrices = []PriceEntry{}
			setscoll.storageAddSet(&set)

			cards, _ := coll.storageFind(bson.D{{"set", set.Code}}, bson.D{{"_id", 1}})

			// if no cards in collection for this set, skip it
			if len(cards) == 0 {
				continue
			}

			bar := progressbar.NewOptions(len(cards),
				progressbar.OptionSetWidth(50),
				progressbar.OptionSetDescription(fmt.Sprintf("%s, %s%s%s\t", set.ReleasedAt[0:4], Yellow, set.Code, Reset)),
				progressbar.OptionEnableColorCodes(true),
				progressbar.OptionShowCount(),
				progressbar.OptionSetTheme(progressbar.Theme{
					Saucer:        "[green]=[reset]",
					SaucerHead:    "[green]>[reset]",
					SaucerPadding: " ",
					BarStart:      "|",
					BarEnd:        fmt.Sprintf("| %s%s%s", Pink, set.Name, Reset),
				}),
			)
			for _, card := range cards {
				bar.Add(1)
				updatedCard, err := fetchCard(card.Set, card.CollectorNumber)
				if err != nil {
					LogMessage(fmt.Sprintf("%v", err), "red")
					continue
				}

				updatedCard.Prices.Date = primitive.NewDateTimeFromTime(time.Now())

				update := bson.M{
					"$set":  bson.M{"serra_updated": primitive.NewDateTimeFromTime(time.Now()), "prices": updatedCard.Prices, "collectornumber": updatedCard.CollectorNumber},
					"$push": bson.M{"serra_prices": updatedCard.Prices},
				}
				coll.storageUpdate(bson.M{"_id": bson.M{"$eq": card.ID}}, update)
			}
			fmt.Println()

			// update set value sum

			// calculate value summary
			matchStage := bson.D{{"$match", bson.D{{"set", set.Code}}}}
			setValue, _ := coll.storageAggregate(mongo.Pipeline{matchStage, projectStage, groupStage})

			p := PriceEntry{}
			s := setValue[0]

			p.Date = primitive.NewDateTimeFromTime(time.Now())

			// fill struct PriceEntry with map from mongoresult
			mapstructure.Decode(s, &p)

			// do the update
			setUpdate := bson.M{
				"$set":  bson.M{"serra_updated": p.Date},
				"$push": bson.M{"serra_prices": p},
			}
			// fmt.Printf("Set %s%s%s (%s) is now worth %s%.02f EUR%s\n", Pink, set.Name, Reset, set.Code, Yellow, setvalue[0]["value"], Reset)
			setscoll.storageUpdate(bson.M{"code": bson.M{"$eq": set.Code}}, setUpdate)
		}

		totalValue, _ := coll.storageAggregate(mongo.Pipeline{projectStage, groupStage})

		t := PriceEntry{}
		t.Date = primitive.NewDateTimeFromTime(time.Now())
		mapstructure.Decode(totalValue[0], &t)

		// This is here to be able to fetch currency from
		// constructed new priceentry
		tmpCard := Card{}
		tmpCard.Prices = t

		fmt.Printf("\n%sUpdating total value of collection to: %s%.02f%s%s\n", Green, Yellow, tmpCard.getValue(false)+tmpCard.getValue(true), getCurrency(), Reset)
		totalcoll.storageAddTotal(t)

		return nil
	},
}
