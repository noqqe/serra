package serra

import (
	"fmt"
	"time"

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

		client := storage_connect()
		defer storage_disconnect(client)

		// update sets
		setscoll := &Collection{client.Database("serra").Collection("sets")}
		coll := &Collection{client.Database("serra").Collection("cards")}
		totalcoll := &Collection{client.Database("serra").Collection("total")}

		sets, _ := fetch_sets()
		for _, set := range sets.Data {
			setscoll.storage_add_set(&set)
			cards, _ := coll.storage_find(bson.D{{"set", set.Code}}, bson.D{{"_id", 1}})

			// if no cards in collection for this set, skip it
			if len(cards) == 0 {
				continue
			}

			// Workaround for:
			// "The field 'serra_prices' must be an array but is of type null in document"
			// I dont know why I need to initialize this array alement as an empty array, but $push does
			// not work on null fields.
			if len(set.SerraPrices) == 0 {
				set_init := bson.M{
					"$set": bson.M{"serra_prices": bson.A{}},
				}
				setscoll.storage_update(bson.M{"code": bson.M{"$eq": set.Code}}, set_init)
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
				updated_card, err := fetch_card(fmt.Sprintf("%s/%s", card.Set, card.CollectorNumber))
				if err != nil {
					LogMessage(fmt.Sprintf("%v", err), "red")
					continue
				}

				update := bson.M{
					"$set": bson.M{"serra_updated": primitive.NewDateTimeFromTime(time.Now()), "prices": updated_card.Prices, "collectornumber": updated_card.CollectorNumber},
					"$push": bson.M{"serra_prices": bson.M{"date": primitive.NewDateTimeFromTime(time.Now()),
						"value": updated_card.Prices.Eur}},
				}
				coll.storage_update(bson.M{"_id": bson.M{"$eq": card.ID}}, update)
			}
			fmt.Println()

			// update set value sum

			// calculate value summary
			matchStage := bson.D{
				{"$match", bson.D{
					{"set", set.Code},
				}},
			}
			groupStage := bson.D{
				{"$group", bson.D{
					{"_id", "$set"},
					{"value", bson.D{{"$sum", bson.D{{"$multiply", bson.A{"$prices.eur", "$serra_count"}}}}}},
				}}}
			setvalue, _ := coll.storage_aggregate(mongo.Pipeline{matchStage, groupStage})

			// do the update
			set_update := bson.M{
				"$set": bson.M{"serra_updated": primitive.NewDateTimeFromTime(time.Now())},
				"$push": bson.M{"serra_prices": bson.M{"date": primitive.NewDateTimeFromTime(time.Now()),
					"value": setvalue[0]["value"]}},
			}
			// fmt.Printf("Set %s%s%s (%s) is now worth %s%.02f EUR%s\n", Pink, set.Name, Reset, set.Code, Yellow, setvalue[0]["value"], Reset)
			setscoll.storage_update(bson.M{"code": bson.M{"$eq": set.Code}}, set_update)
		}

		// calculate total summary over all sets
		overall_value := mongo.Pipeline{
			bson.D{{"$match",
				bson.D{{"serra_prices", bson.D{{"$type", "array"}}}}}},
			bson.D{{"$project",
				bson.D{{"name", true}, {"totalValue", bson.D{{"$arrayElemAt", bson.A{"$serra_prices", -1}}}}}}},
			bson.D{{"$group", bson.D{{"_id", nil}, {"total", bson.D{{"$sum", "$totalValue.value"}}}}}},
		}
		ostats, _ := setscoll.storage_aggregate(overall_value)
		fmt.Printf("\n%sUpdating total value of collection to: %s%.02f EUR%s\n", Green, Yellow, ostats[0]["total"].(float64), Reset)
		totalcoll.storage_add_total(ostats[0]["total"].(float64))

		return nil
	},
}
