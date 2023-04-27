package serra

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	rootCmd.AddCommand(statsCmd)
}

var statsCmd = &cobra.Command{
	Aliases:       []string{"stats"},
	Use:           "stats <prefix> <n>",
	Short:         "Shows statistics of the collection",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {

		client := storageConnect()
		coll := &Collection{client.Database("serra").Collection("cards")}
		totalcoll := &Collection{client.Database("serra").Collection("total")}
		defer storageDisconnect(client)

		sets, _ := coll.storageAggregate(mongo.Pipeline{
			bson.D{
				{"$match", bson.D{
					{"coloridentity", bson.D{{"$size", 1}}}}}},
			bson.D{
				{"$group", bson.D{
					{"_id", "$coloridentity"},
					{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
				}}},
			bson.D{
				{"$sort", bson.D{
					{"count", -1},
				}}},
		})
		fmt.Printf("%sColors%s\n", Green, Reset)
		for _, set := range sets {
			x, _ := set["_id"].(primitive.A)
			s := []interface{}(x)
			fmt.Printf("%s: %s%.0f%s\n", convertManaSymbols(s), Purple, set["count"], Reset)
		}

		stats, _ := coll.storageAggregate(mongo.Pipeline{
			bson.D{
				{"$group", bson.D{
					{"_id", nil},
					{"value", bson.D{{"$sum", bson.D{{"$multiply", bson.A{getCurrencyField(false), "$serra_count"}}}}}},
					{"value_foil", bson.D{{"$sum", bson.D{{"$multiply", bson.A{getCurrencyField(true), "$serra_count_foil"}}}}}},
					{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
					{"count_foil", bson.D{{"$sum", "$serra_count_foil"}}},
					{"rarity", bson.D{{"$sum", "$rarity"}}},
					{"unique", bson.D{{"$sum", 1}}},
				}},
			},
			bson.D{
				{"$addFields", bson.D{
					{"count_all", bson.D{{"$sum", bson.A{"$count", "$count_foil"}}}},
				}},
			},
		})

		fmt.Printf("\n%sCards %s\n", Green, Reset)
		fmt.Printf("Total: %s%.0f%s\n", Yellow, stats[0]["count_all"], Reset)
		fmt.Printf("Unique: %s%d%s\n", Purple, stats[0]["unique"], Reset)
		fmt.Printf("Normal: %s%.0f%s\n", Purple, stats[0]["count"], Reset)
		fmt.Printf("Foil: %s%d%s\n", Purple, stats[0]["count_foil"], Reset)

		reserved, _ := coll.storageAggregate(mongo.Pipeline{
			bson.D{
				{"$match", bson.D{
					{"reserved", true}}}},
			bson.D{
				{"$group", bson.D{
					{"_id", nil},
					{"count", bson.D{{"$sum", 1}}},
				}}},
		})

		var count_reserved int32
		if len(reserved) > 0 {
			count_reserved = reserved[0]["count"].(int32)
		}
		fmt.Printf("Reserved List: %s%d%s\n", Yellow, count_reserved, Reset)

		rar, _ := coll.storageAggregate(mongo.Pipeline{
			bson.D{
				{"$group", bson.D{
					{"_id", "$rarity"},
					{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
				}}},
			bson.D{
				{"$sort", bson.D{
					{"_id", 1},
				}}},
		})
		ri := convertRarities(rar)
		fmt.Printf("\n%sRarity%s\n", Green, Reset)
		fmt.Printf("Mythics: %s%.0f%s\n", Pink, ri.Mythics, Reset)
		fmt.Printf("Rares: %s%.0f%s\n", Pink, ri.Rares, Reset)
		fmt.Printf("Uncommons: %s%.0f%s\n", Yellow, ri.Uncommons, Reset)
		fmt.Printf("Commons: %s%.0f%s\n", Purple, ri.Commons, Reset)

		fmt.Printf("\n%sTotal Value%s\n", Green, Reset)
		nf_value, err := getFloat64(stats[0]["value"])
		if err != nil {
			LogMessage(fmt.Sprintf("Error: %v", err), "red")
			nf_value = 0
		}
		foil_value, err := getFloat64(stats[0]["value_foil"])
		if err != nil {
			LogMessage(fmt.Sprintf("Error: %v", err), "red")
			foil_value = 0
		}
		total_value := nf_value + foil_value
		fmt.Printf("Total: %s%.2f%s%s\n", Pink, total_value, getCurrency(), Reset)
		fmt.Printf("Normal: %s%.2f%s%s\n", Pink, nf_value, getCurrency(), Reset)
		fmt.Printf("Foils: %s%.2f%s%s\n", Pink, foil_value, getCurrency(), Reset)
		total, _ := totalcoll.storageFindTotal()

		fmt.Printf("History: \n")
		showPriceHistory(total.Value, "* ", true)
		return nil
	},
}
