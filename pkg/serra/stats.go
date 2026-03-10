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
	coll := client.getCardsCollection()
	totalcoll := client.getTotalCollection()
	defer storageDisconnect(client)

	// Show Value Stats
	showValueStats(coll, totalcoll)

	// Rarities
	showRarityStats(coll)

	// Reserved List
	showReservedListStats(coll)

	// Colors
	showColorStats(coll)

	// Artists
	showArtistStats(coll)

	// Mana Curve of Collection
	showManaCurveStats(coll)

	// Show cards added per month
	showCardsAddedPerMonth(coll)
}

func showValueStats(coll CardsCollection, totalcoll TotalCollection) {
	l := Logger()
	// Value and Card Numbers
	stats, _ := coll.AggregateCards(mongo.Pipeline{
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
	fmt.Printf("%s\n", Green("Cards"))
	fmt.Printf("Total: %s\n", Yellow("%.0f", stats[0]["count_all"]))
	fmt.Printf("Unique: %s\n", Purple("%d", stats[0]["unique"]))
	fmt.Printf("Normal: %s\n", Purple("%.0f", stats[0]["count"]))
	fmt.Printf("Foil: %s\n", Purple("%d", stats[0]["count_foil"]))

	// Total Value
	fmt.Printf("\n%s\n", Green("Total Value"))
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
	countAll, err := getFloat64(stats[0]["count_all"])
	if err != nil {
		l.Error(err)
		foilValue = 0
	}
	totalValue := normalValue + foilValue
	fmt.Printf("Total: %s%s\n", Pink("%.2f", totalValue), Pink(getCurrency()))
	fmt.Printf("Normal: %s%s\n", Pink("%.2f", normalValue), Pink(getCurrency()))
	fmt.Printf("Foils: %s%s\n", Pink("%.2f", foilValue), Pink(getCurrency()))
	fmt.Printf("Average Card: %s%s\n", Pink("%.2f", totalValue/countAll), Pink(getCurrency()))
	total, _ := totalcoll.FindTotal()

	fmt.Printf("History: \n")
	showPriceHistory(total.Value, "* ", true)
}

func showReservedListStats(coll CardsCollection) {
	reserved, _ := coll.AggregateCards(mongo.Pipeline{
		bson.D{
			{"$match", bson.D{
				{"reserved", true}}}},
		bson.D{
			{"$group", bson.D{
				{"_id", nil},
				{"count", bson.D{{"$sum", 1}}},
			}}},
	})

	var countReserved int32
	if len(reserved) > 0 {
		countReserved = reserved[0]["count"].(int32)
	}
	fmt.Printf("Reserved List: %s\n", Yellow("%d", countReserved))
}

func showRarityStats(coll CardsCollection) {
	rar, _ := coll.AggregateCards(mongo.Pipeline{
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
	fmt.Printf("\n%s\n", Green("Rarity"))
	fmt.Printf("Mythics: %s\n", Pink("%.0f", ri.Mythics))
	fmt.Printf("Rares: %s\n", Pink("%.0f", ri.Rares))
	fmt.Printf("Uncommons: %s\n", Yellow("%.0f", ri.Uncommons))
	fmt.Printf("Commons: %s\n", Purple("%.0f", ri.Commons))
}

func showCardsAddedPerMonth(coll CardsCollection) {
	fmt.Printf("\n%s\n", Green("Cards added over time"))
	type cardsAddedOverTime struct {
		ID struct {
			Year  int32 `mapstructure:"year"`
			Month int32 `mapstructure:"month"`
		} `mapstructure:"_id"`
		Count int32 `mapstructure:"count"`
	}
	caot, _ := coll.AggregateCards(mongo.Pipeline{
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
	for _, month := range caot {
		thisMonth := new(cardsAddedOverTime)
		mapstructure.Decode(month, thisMonth)
		fmt.Printf("%d-%02d: %s\n", thisMonth.ID.Year, thisMonth.ID.Month, Purple("%d", thisMonth.Count))
	}
}

func showManaCurveStats(coll CardsCollection) {
	cmc, _ := coll.AggregateCards(mongo.Pipeline{
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
	fmt.Printf("\n%s\n", Green("Mana Curve"))
	for _, mc := range cmc {
		fmt.Printf("%.0f: %s\n", mc["_id"], Purple("%d", mc["count"]))
	}
}

func showArtistStats(coll CardsCollection) {
	artists, _ := coll.AggregateCards(mongo.Pipeline{
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
	fmt.Printf("\n%s\n", Green("Top Artists"))
	for _, artist := range artists {
		fmt.Printf("%s: %s\n", artist["_id"], Purple("%d", artist["count"]))
	}
}

func showColorStats(coll CardsCollection) {
	sets, _ := coll.AggregateCards(mongo.Pipeline{
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

	fmt.Printf("\n%s\n", Green("Colors"))
	for _, set := range sets {
		x, _ := set["_id"].(primitive.A)
		s := []any(x)
		fmt.Printf("%s: %s\n", convertManaSymbols(s), Purple("%.0f", set["count"]))
	}
}
