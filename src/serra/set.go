package serra

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	setCmd.Flags().StringVarP(&sort, "sort", "s", "release", "How to sort cards (release/value)")
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
			setList := Sets(sort)
			show_set_list(setList)
		} else {
			ShowSet(set[0])
		}
		return nil
	},
}

func Sets(sort string) []primitive.M {

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storage_disconnect(client)

	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", "$setname"},
			{"value", bson.D{{"$sum", bson.D{{"$multiply", bson.A{getCurrencyField(), "$serra_count"}}}}}},
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

	sets, _ := coll.storage_aggregate(mongo.Pipeline{groupStage, sortStage})
	return sets

}

func show_set_list(sets []primitive.M) {

	client := storage_connect()
	setscoll := &Collection{client.Database("serra").Collection("sets")}

	for _, set := range sets {
		setobj, _ := find_set_by_code(setscoll, set["code"].(string))
		fmt.Printf("* %s %s%s%s (%s%s%s)\n", set["release"].(string)[0:4], Purple, set["_id"], Reset, Cyan, set["code"], Reset)
		fmt.Printf("  Cards: %s%d/%d%s Total: %.0f \n", Yellow, set["unique"], setobj.CardCount, Reset, set["count"])
		fmt.Printf("  Value: %s%.2f %s%s\n", Pink, set["value"], getCurrency(), Reset)
		fmt.Println()
	}
}

func ShowSet(setname string) error {

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storage_disconnect(client)

	// fetch all cards in set ordered by currently used currency
	cardSortCurrency := bson.D{{"prices.usd", -1}}
	if getCurrency() == "EUR" {
		cardSortCurrency = bson.D{{"prices.eur", -1}}
	}
	cards, err := coll.storage_find(bson.D{{"set", setname}}, cardSortCurrency)
	if (err != nil) || len(cards) == 0 {
		LogMessage(fmt.Sprintf("Error: Set %s not found or no card in your collection.", setname), "red")
		return err
	}

	// fetch set informations
	setcoll := &Collection{client.Database("serra").Collection("sets")}
	sets, _ := setcoll.storage_find_set(bson.D{{"code", setname}}, bson.D{{"_id", 1}})

	// set values
	matchStage := bson.D{
		{"$match", bson.D{
			{"set", setname},
		}},
	}
	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", "$setname"},
			{"value", bson.D{{"$sum", bson.D{{"$multiply", bson.A{getCurrencyField(), "$serra_count"}}}}}},
			{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
		}},
	}
	stats, _ := coll.storage_aggregate(mongo.Pipeline{matchStage, groupStage})

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
	rar, _ := coll.storage_aggregate(mongo.Pipeline{matchStage, groupStage, sortStage})

	ri := convert_rarities(rar)

	LogMessage(fmt.Sprintf("%s", sets[0].Name), "green")
	LogMessage(fmt.Sprintf("Set Cards: %d/%d", len(cards), sets[0].CardCount), "normal")
	LogMessage(fmt.Sprintf("Total Cards: %.0f", stats[0]["count"]), "normal")
	LogMessage(fmt.Sprintf("Total Value: %.2f %s", stats[0]["value"], getCurrency()), "normal")
	LogMessage(fmt.Sprintf("Released: %s", sets[0].ReleasedAt), "normal")
	LogMessage(fmt.Sprintf("Mythics: %.0f", ri.Mythics), "normal")
	LogMessage(fmt.Sprintf("Rares: %.0f", ri.Rares), "normal")
	LogMessage(fmt.Sprintf("Uncommons: %.0f", ri.Uncommons), "normal")
	LogMessage(fmt.Sprintf("Commons: %.0f", ri.Commons), "normal")
	fmt.Printf("\n%sPrice History:%s\n", Pink, Reset)
	print_price_history(sets[0].SerraPrices, "* ")

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
		fmt.Printf("* %dx %s%s%s (%s/%s) %s%.2f %s%s\n", card.SerraCount, Purple, card.Name, Reset, sets[0].Code, card.CollectorNumber, Yellow, card.getValue(), getCurrency(), Reset)
	}

	return nil
}
