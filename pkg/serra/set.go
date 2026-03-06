package serra

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	setCmd.Flags().StringVarP(&sortBy, "sort", "s", "release", "How to sort cards (release/value)")
	rootCmd.AddCommand(setCmd)
}

type SetsResult struct {
	ID        string  `bson:"_id"`
	Code      string  `bson:"code"`
	Value     float64 `bson:"value"`
	ValueFoil float64 `bson:"value_foil"`
	Count     int32   `bson:"count"`
	Unique    int32   `bson:"unique"`
	Release   string  `bson:"release"`
}

var setCmd = &cobra.Command{
	Aliases: []string{"cards"},
	Use:     "set [set]",
	Short:   "Search & show sets from your collection",
	Long: `Search and show sets from your collection.
If you directly put a setcode as an argument, it will be displayed
otherwise you'll get a list of sets as a search result.`,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, sets []string) error {
		if len(sets) == 0 {
			setList := Sets(sortBy)
			showSetList(setList)
		} else {
			for _, set := range sets {
				ShowSet(set)
			}
		}
		return nil
	},
}

func Sets(sort string) []SetsResult {

	client := storageConnect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storageDisconnect(client)
	l := Logger()

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

	bsonList, err := coll.storageAggregate(mongo.Pipeline{groupStage, sortStage})

	if err != nil {
		l.Error("Error fetching sets:", err)
		return nil
	}

	sets := []SetsResult{}
	for _, bsonEntry := range bsonList {
		var item SetsResult
		// Unmarshal bson.M into myStruct
		bsonData, err := bson.Marshal(bsonEntry)
		if err != nil {
			l.Error("Error marshalling set result:", err)
			return nil
		}
		err = bson.Unmarshal(bsonData, &item)
		if err != nil {
			l.Error("Error unmarshalling set result:", err)
			return nil
		}
		sets = append(sets, item)
	}
	return sets

}

func showSetList(sets []SetsResult) {

	client := storageConnect()
	setscoll := &Collection{client.Database("serra").Collection("sets")}

	for _, set := range sets {
		setobj, _ := findSetByCode(setscoll, set.Code)
		fmt.Printf("* %s %s (%s)\n", fmt.Sprintf("%.4s", set.Release), Purple(set.ID), Cyan(set.Code))
		fmt.Printf("  Cards: %s Total: %d \n", Yellow("%d/%d", set.Unique, setobj.CardCount), set.Count)
		fmt.Printf("  Value: %s%s\n", Pink("%.2f", set.Value), Pink(getCurrency()))
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
	set, err := findSetByCode(setcoll, setname)
	if err != nil {
		l.Errorf("Set %s not found or no card in your collection.", setname)
		return err
	}

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

	fmt.Printf("%s\n", Green(set.Name))
	fmt.Printf("Released: %s\n", set.ReleasedAt)
	fmt.Printf("Set Cards: %d/%d\n", len(cards), set.CardCount)
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

	fmt.Printf("\n%s\n", Purple("Current Value"))
	fmt.Printf("Total: %.0fx %s%s\n", normalCount+foilCount, Yellow("%.2f", totalValue), Yellow(getCurrency()))
	fmt.Printf("Normal: %.0fx %s%s\n", stats[0]["count"], Yellow("%.2f", normalValue), Yellow(getCurrency()))
	fmt.Printf("Foil: %.0fx %s%s\n", stats[0]["count_foil"], Yellow("%.2f", foilValue), Yellow(getCurrency()))

	fmt.Printf("\n%s\n", Purple("Rarities"))
	fmt.Printf("Mythics: %.0f\n", ri.Mythics)
	fmt.Printf("Rares: %.0f\n", ri.Rares)
	fmt.Printf("Uncommons: %.0f\n", ri.Uncommons)
	fmt.Printf("Commons: %.0f\n", ri.Commons)

	fmt.Printf("\n%s\n", Pink("Price History"))
	showPriceHistory(set.SerraPrices, "* ", true)

	fmt.Printf("\n%s\n", Pink("Most valuable cards"))

	// Calc counter to show 10 cards or less
	ccards := min(10, len(cards))
	for _, card := range cards[:ccards] {
		fmt.Printf("* %s (%s/%s) %s%s\n", Purple(card.Name), set.Code, card.CollectorNumber, Yellow("%.2f", card.getValue()), Yellow(getCurrency()))
	}

	return nil
}
