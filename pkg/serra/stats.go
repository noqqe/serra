package serra

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
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
	Use:           "stats",
	Short:         "Shows statistics of the collection",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		Stats()
		return nil
	},
}

func Stats() {
	client := storageConnect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	totalcoll := &Collection{client.Database("serra").Collection("total")}
	defer storageDisconnect(client)

	// Show Value Stats
	showValueStats(coll, totalcoll)

	// Reserved List
	showReservedListStats(coll)

	// Rarities
	showRarityStats(coll)

	// Colors
	showColorStats(coll)

	// Artists
	showArtistStats(coll)

	// Mana Curve of Collection
	showManaCurveStats(coll)

	// Show cards added per month
	showCardsAddedPerMonth(coll)
}

func showValueStats(coll *Collection, totalcoll *Collection) {
	l := Logger()
	// Value and Card Numbers
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
	fmt.Printf("%sCards %s\n", Green, Reset)
	fmt.Printf("Total: %s%.0f%s\n", Yellow, stats[0]["count_all"], Reset)
	fmt.Printf("Unique: %s%d%s\n", Purple, stats[0]["unique"], Reset)
	fmt.Printf("Normal: %s%.0f%s\n", Purple, stats[0]["count"], Reset)
	fmt.Printf("Foil: %s%d%s\n", Purple, stats[0]["count_foil"], Reset)

	// Total Value
	fmt.Printf("\n%sTotal Value%s\n", Green, Reset)
	normalValue, err := getFloat64(stats[0]["value"])
	if err != nil {
		l.Error(err)
		normalValue = 0
	}
	foilValue, err := getFloat64(stats[0]["value_foil"])
	if err != nil {
		l.Error(err)
		foilValue = 0
	}
	count_all, err := getFloat64(stats[0]["count_all"])
	if err != nil {
		l.Error(err)
		foilValue = 0
	}
	totalValue := normalValue + foilValue
	fmt.Printf("Total: %s%.2f%s%s\n", Pink, totalValue, getCurrency(), Reset)
	fmt.Printf("Normal: %s%.2f%s%s\n", Pink, normalValue, getCurrency(), Reset)
	fmt.Printf("Foils: %s%.2f%s%s\n", Pink, foilValue, getCurrency(), Reset)
	fmt.Printf("Average Card: %s%.2f%s%s\n", Pink, totalValue/count_all, getCurrency(), Reset)
	total, _ := totalcoll.storageFindTotal()

	fmt.Printf("History: \n")
	showPriceHistory(total.Value, "* ", true)
}

func showReservedListStats(coll *Collection) {
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
}

func showRarityStats(coll *Collection) {
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
}

func showCardsAddedPerMonth(coll *Collection) {
	fmt.Printf("\n%sCards added over time%s\n", Green, Reset)
	type Caot struct {
		Id struct {
			Year  int32 `mapstructure:"year"`
			Month int32 `mapstructure:"month"`
		} `mapstructure:"_id"`
		Count int32 `mapstructure:"count"`
	}
	caot, _ := coll.storageAggregate(mongo.Pipeline{
		bson.D{
			{"$project", bson.D{
				{"month", bson.D{
					{"$month", "$serra_created"}}},
				{"year", bson.D{
					{"$year", "$serra_created"}},
				}},
			}},
		bson.D{
			{"$group", bson.D{
				{"_id", bson.D{{"month", "$month"}, {"year", "$year"}}},
				{"count", bson.D{{"$sum", 1}}},
			}},
		},
		bson.D{
			{"$sort", bson.D{{"_id.year", 1}, {"_id.month", 1}}},
		},
	})
	for _, mo := range caot {
		moo := new(Caot)
		mapstructure.Decode(mo, moo)
		fmt.Printf("%d-%02d: %s%d%s\n", moo.Id.Year, moo.Id.Month, Purple, moo.Count, Reset)
	}
}

func showManaCurveStats(coll *Collection) {
	cmc, _ := coll.storageAggregate(mongo.Pipeline{
		bson.D{
			{"$group", bson.D{
				{"_id", "$cmc"},
				{"count", bson.D{{"$sum", 1}}},
			}}},
		bson.D{
			{"$sort", bson.D{
				{"_id", 1},
			}}},
	})
	fmt.Printf("\n%sMana Curve%s\n", Green, Reset)
	for _, mc := range cmc {
		fmt.Printf("%.0f: %s%d%s\n", mc["_id"], Purple, mc["count"], Reset)
	}
}

func showArtistStats(coll *Collection) {
	artists, _ := coll.storageAggregate(mongo.Pipeline{
		bson.D{
			{"$group", bson.D{
				{"_id", "$artist"},
				{"count", bson.D{{"$sum", 1}}},
			}}},
		bson.D{
			{"$sort", bson.D{
				{"count", -1},
			}}},
		bson.D{
			{"$limit", 10}},
	})
	fmt.Printf("\n%sTop Artists%s\n", Green, Reset)
	for _, artist := range artists {
		fmt.Printf("%s: %s%d%s\n", artist["_id"].(string), Purple, artist["count"], Reset)
	}
}

func showColorStats(coll *Collection) {
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

	fmt.Printf("\n%sColors%s\n", Green, Reset)
	for _, set := range sets {
		x, _ := set["_id"].(primitive.A)
		s := []interface{}(x)
		fmt.Printf("%s: %s%.0f%s\n", convertManaSymbols(s), Purple, set["count"], Reset)
	}
}
