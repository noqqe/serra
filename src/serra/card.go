package serra

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	cardCmd.Flags().StringVarP(&rarity, "rarity", "r", "", "Filter by rarity of cards (mythic, rare, uncommon, common)")
	cardCmd.Flags().StringVarP(&set, "set", "e", "", "Filter by set code (usg/mmq/vow)")
	cardCmd.Flags().StringVarP(&sortby, "sort", "s", "name", "How to sort cards (value/number/name/added)")
	cardCmd.Flags().StringVarP(&name, "name", "n", "", "Name of the card (regex compatible)")
	cardCmd.Flags().StringVarP(&oracle, "oracle", "o", "", "Contains string in card text")
	cardCmd.Flags().StringVarP(&cardType, "type", "t", "", "Contains string in card type line")
	cardCmd.Flags().BoolVarP(&detail, "detail", "d", false, "Show details for cards (url)")
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
			card_list := Cards(rarity, set, sortby, name, oracle, cardType)
			show_card_list(card_list, detail)
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
		if len(strings.Split(v, "/")) < 2 || strings.Split(v, "/")[1] == "" {
			LogMessage(fmt.Sprintf("Invalid card %s", v), "red")
			continue
		}

		cards, _ := coll.storage_find(bson.D{{"set", strings.Split(v, "/")[0]}, {"collectornumber", strings.Split(v, "/")[1]}}, bson.D{{"name", 1}})

		for _, card := range cards {
			show_card_details(&card)
		}
	}
}

func Cards(rarity, set, sortby, name, oracle, cardType string) []Card {
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
	switch sortby {
	case "value":
		if getCurrency() == "EUR" {
			sortStage = bson.D{{"prices.eur", 1}}
		} else {
			sortStage = bson.D{{"prices.usd", 1}}
		}
	case "number":
		sortStage = bson.D{{"collectornumber", 1}}
	case "name":
		sortStage = bson.D{{"name", 1}}
	case "added":
		sortStage = bson.D{{"serra_created", 1}}
	default:
		sortStage = bson.D{{"name", 1}}
	}

	if len(set) > 0 {
		filter = append(filter, bson.E{"set", set})
	}

	if len(name) > 0 {
		filter = append(filter, bson.E{"name", bson.D{{"$regex", ".*" + name + ".*"}, {"$options", "i"}}})
	}

	if len(oracle) > 0 {
		filter = append(filter, bson.E{"oracletext", bson.D{{"$regex", ".*" + oracle + ".*"}, {"$options", "i"}}})
	}

	if len(cardType) > 0 {
		filter = append(filter, bson.E{"typeline", bson.D{{"$regex", ".*" + cardType + ".*"}, {"$options", "i"}}})
	}

	cards, _ := coll.storage_find(filter, sortStage)

	// This is needed because collectornumbers are strings (ie. "23a") but still we
	// want it to be sorted numerically ... 1,2,3,10,11,100.
	if sortby == "number" {
		sort.Slice(cards, func(i, j int) bool {
			return filterForDigits(cards[i].CollectorNumber) < filterForDigits(cards[j].CollectorNumber)
		})
	}

	return cards
}

func show_card_list(cards []Card, detail bool) {

	var total float64
	if detail {
		for _, card := range cards {
			fmt.Printf("* %dx %s%s%s (%s/%s) %s%.2f%s %s %s %s\n", card.SerraCount+card.SerraCountFoil+card.SerraCountEtched, Purple, card.Name, Reset, card.Set, card.CollectorNumber, Yellow, card.getValue(false), getCurrency(), Background, strings.Replace(card.ScryfallURI, "?utm_source=api", "", 1), Reset)
			total = total + card.getValue(false)*float64(card.SerraCount) + card.getValue(true)*float64(card.SerraCountFoil)
		}
	} else {
		for _, card := range cards {
			fmt.Printf("* %dx %s%s%s (%s/%s) %s%.2f%s%s\n", card.SerraCount+card.SerraCountFoil+card.SerraCountEtched, Purple, card.Name, Reset, card.Set, card.CollectorNumber, Yellow, card.getValue(false), getCurrency(), Reset)
			total = total + card.getValue(false)*float64(card.SerraCount) + card.getValue(true)*float64(card.SerraCountFoil)
		}
	}

	fmt.Printf("\nTotal Value: %s%.2f%s%s\n", Yellow, total, getCurrency(), Reset)

}

func show_card_details(card *Card) error {
	fmt.Printf("%s%s%s (%s/%s)\n", Purple, card.Name, Reset, card.Set, card.CollectorNumber)
	fmt.Printf("Added: %s\n", stringToTime(card.SerraCreated))
	fmt.Printf("Rarity: %s\n", card.Rarity)
	fmt.Printf("Scryfall: %s\n", strings.Replace(card.ScryfallURI, "?utm_source=api", "", 1))

	fmt.Printf("\n%sCurrent Value%s\n", Green, Reset)
	fmt.Printf("* Normal: %dx %s%.2f%s%s\n", card.SerraCount, Yellow, card.getValue(false), getCurrency(), Reset)
	if card.SerraCountFoil > 0 {
		fmt.Printf("* Foil: %dx %s%.2f%s%s\n", card.SerraCountFoil, Yellow, card.getValue(true), getCurrency(), Reset)
	}

	fmt.Printf("\n%sValue History%s\n", Green, Reset)
	print_price_history(card.SerraPrices, "* ", false)
	fmt.Println()
	return nil
}
