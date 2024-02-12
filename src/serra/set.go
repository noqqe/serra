package serra

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	setCmd.Flags().StringVarP(&sortby, "sort", "s", "release", "How to sort cards (release/value)")
	rootCmd.AddCommand(setCmd)
}

var setCmd = &cobra.Command{
	Aliases: []string{"cards"},
	Use:     "set [set]",
	Short:   "Search & show sets from your collection",
	Long: `Search and show sets from your collection.
If you directly put a setcode as an argument, it will be displayed
otherwise you'll get a list of sets as a search result.`,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, set []string) error {
		if len(set) == 0 {
			setList := Sets(sortby)
			showSetList(setList)
		} else {
			ShowSet(set[0])
		}
		return nil
	},
}

func Sets(sort string) []primitive.M {

	client := storageConnect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storageDisconnect(client)

	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", "$setname"},
			{"value", bson.D{{"$sum", bson.D{{"$multiply", bson.A{getCurrencyField(false), "$serra_count"}}}}}},
			{"value_foil", bson.D{{"$sum", bson.D{{"$multiply", bson.A{getCurrencyField(true), "$serra_count_foil"}}}}}},
			{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
			{"unique", bson.D{{"$sum", 1}}},
			{"code", bson.D{{"$last", "$set"}}},
			{"release", bson.D{{"$last", "$releasedat"}}},
		}},
	}

	var sortStage bson.D
	switch sort {
	case "release":
		sortStage = bson.D{
			{"$sort", bson.D{
				{"release", 1},
			}}}
	case "value":
		sortStage = bson.D{
			{"$sort", bson.D{
				{"value", 1},
			}}}
	}

	sets, _ := coll.storageAggregate(mongo.Pipeline{groupStage, sortStage})
	return sets

}

func showSetList(sets []primitive.M) {

	client := storageConnect()
	setscoll := &Collection{client.Database("serra").Collection("sets")}

	for _, set := range sets {
		setobj, _ := findSetByCode(setscoll, set["code"].(string))
		fmt.Printf("* %s %s%s%s (%s%s%s)\n", set["release"].(string)[0:4], Purple, set["_id"], Reset, Cyan, set["code"], Reset)
		fmt.Printf("  Cards: %s%d/%d%s Total: %.0f \n", Yellow, set["unique"], setobj.CardCount, Reset, set["count"])
		fmt.Printf("  Value: %s%.2f%s%s\n", Pink, set["value"], getCurrency(), Reset)
		fmt.Println()
	}
}

func ShowSet(setname string) error {

	client := storageConnect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	l := Logger()
	defer storageDisconnect(client)

	// fetch all cards in set ordered by currently used currency
	cardSortCurrency := bson.D{{"prices.usd", -1}}
	if getCurrency() == EUR {
		cardSortCurrency = bson.D{{"prices.eur", -1}}
	}
	cards, err := coll.storageFind(bson.D{{"set", setname}}, cardSortCurrency, 0, 0)
	if (err != nil) || len(cards) == 0 {
		l.Errorf("Set %s not found or no card in your collection.", setname)
		return err
	}

	// fetch set informations
	setcoll := &Collection{client.Database("serra").Collection("sets")}
	sets, _ := setcoll.storageFindSet(bson.D{{"code", setname}}, bson.D{{"_id", 1}})

	// set values
	matchStage := bson.D{
		{"$match", bson.D{
			{"set", setname},
		}},
	}
	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", "$setname"},
			{"value", bson.D{{"$sum", bson.D{{"$multiply", bson.A{getCurrencyField(false), "$serra_count"}}}}}},
			{"value_foil", bson.D{{"$sum", bson.D{{"$multiply", bson.A{getCurrencyField(true), "$serra_count_foil"}}}}}},
			{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
			{"count_foil", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count_foil"}}}}}},
		}},
	}
	stats, _ := coll.storageAggregate(mongo.Pipeline{matchStage, groupStage})

	// set rarities
	matchStage = bson.D{
		{"$match", bson.D{
			{"set", setname},
		}},
	}
	groupStage = bson.D{
		{"$group", bson.D{
			{"_id", "$rarity"},
			{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
		}}}

	sortStage := bson.D{
		{"$sort", bson.D{
			{"_id", 1},
		}}}
	rar, _ := coll.storageAggregate(mongo.Pipeline{matchStage, groupStage, sortStage})

	ri := convertRarities(rar)

	fmt.Printf("%s%s%s\n", Green, sets[0].Name, Reset)
	fmt.Printf("Released: %s\n", sets[0].ReleasedAt)
	fmt.Printf("Set Cards: %d/%d\n", len(cards), sets[0].CardCount)
	fmt.Printf("Total Cards: %.0f\n", stats[0]["count"])
	fmt.Printf("Foil Cards: %.0f\n", stats[0]["count_foil"])

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
	totalValue := normalValue + foilValue

	normalCount, _ := getFloat64(stats[0]["count"])
	foilCount, _ := getFloat64(stats[0]["count_foil"])

	fmt.Printf("\n%sCurrent Value%s\n", Purple, Reset)
	fmt.Printf("Total: %.0fx %s%.2f%s%s\n", normalCount+foilCount, Yellow, totalValue, getCurrency(), Reset)
	fmt.Printf("Normal: %.0fx %s%.2f%s%s\n", stats[0]["count"], Yellow, normalValue, getCurrency(), Reset)
	fmt.Printf("Foil: %.0fx %s%.2f%s%s\n", stats[0]["count_foil"], Yellow, foilValue, getCurrency(), Reset)

	fmt.Printf("\n%sRarities%s\n", Purple, Reset)
	fmt.Printf("Mythics: %.0f\n", ri.Mythics)
	fmt.Printf("Rares: %.0f\n", ri.Rares)
	fmt.Printf("Uncommons: %.0f\n", ri.Uncommons)
	fmt.Printf("Commons: %.0f\n", ri.Commons)

	fmt.Printf("\n%sPrice History:%s\n", Pink, Reset)
	showPriceHistory(sets[0].SerraPrices, "* ", true)

	fmt.Printf("\n%sMost valuable cards%s\n", Pink, Reset)

	// Calc counter to show 10 cards or less
	ccards := 0
	if len(cards) < 10 {
		ccards = len(cards)
	} else {
		ccards = 10
	}

	for i := 0; i < ccards; i++ {
		card := cards[i]
		fmt.Printf("* %s%s%s (%s/%s) %s%.2f%s%s\n", Purple, card.Name, Reset, sets[0].Code, card.CollectorNumber, Yellow, card.getValue(false), getCurrency(), Reset)
	}

	return nil
}
