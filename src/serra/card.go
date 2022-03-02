package serra

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	cardCmd.Flags().StringVarP(&rarity, "rarity", "r", "", "Filter by rarity of cards (mythic, rare, uncommon, common)")
	cardCmd.Flags().StringVarP(&set, "set", "e", "", "Filter by set code (usg/mmq/vow)")
	cardCmd.Flags().StringVarP(&sort, "sort", "s", "name", "How to sort cards (value/number/name)")
	rootCmd.AddCommand(cardCmd)
}

var cardCmd = &cobra.Command{
	Aliases: []string{"cards"},
	Use:     "card [card]",
	Short:   "Search & show cards from your collection",
	Long: `Search and show cards from your collection.
If you directly put a card as an argument, it will be displayed
otherwise you'll get a list of cards as a search result.`,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, cards []string) error {
		if len(cards) == 0 {
			Cards(rarity, set, sort)
		} else {
			ShowCard(cards)
		}
		return nil
	},
}

func ShowCard(cardids []string) {

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storage_disconnect(client)

	for _, v := range cardids {

		cards, _ := coll.storage_find(bson.D{{"set", strings.Split(v, "/")[0]}, {"collectornumber", strings.Split(v, "/")[1]}}, bson.D{{"name", 1}})

		for _, card := range cards {
			show_card_details(&card)
		}
	}
}

func Cards(rarity, set, sort string) {

	var total float64
	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storage_disconnect(client)

	filter := bson.D{}

	switch rarity {
	case "uncommon":
		filter = append(filter, bson.E{"rarity", "uncommon"})
	case "common":
		filter = append(filter, bson.E{"rarity", "common"})
	case "rare":
		filter = append(filter, bson.E{"rarity", "rare"})
	}

	var sortStage bson.D
	switch sort {
	case "value":
		sortStage = bson.D{{"prices.eur", 1}}
	case "number":
		sortStage = bson.D{{"collectornumber", 1}}
	case "name":
		sortStage = bson.D{{"name", 1}}
	default:
		sortStage = bson.D{{"name", 1}}
	}

	if len(set) > 0 {
		filter = append(filter, bson.E{"set", set})
	}

	cards, _ := coll.storage_find(filter, sortStage)

	for _, card := range cards {
		LogMessage(fmt.Sprintf("* %dx %s%s%s (%s/%s) %s%.2f EUR%s", card.SerraCount, Purple, card.Name, Reset, card.Set, card.CollectorNumber, Yellow, card.Prices.Eur, Reset), "normal")
		total = total + card.Prices.Eur*float64(card.SerraCount)
	}
	fmt.Printf("\nTotal Value: %s%.2f EUR%s\n", Yellow, total, Reset)

}
