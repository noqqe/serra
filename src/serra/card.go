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
	cardCmd.Flags().Int64VarP(&count, "min-count", "c", 0, "Occource more than X in your collection")
	cardCmd.Flags().BoolVarP(&detail, "detail", "d", false, "Show details for cards (url)")
	cardCmd.Flags().BoolVarP(&reserved, "reserved", "w", false, "If card is on reserved list")
	cardCmd.Flags().BoolVarP(&foil, "foil", "f", false, "If card is foil list")
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
			cardList := Cards(rarity, set, sortby, name, oracle, cardType, reserved, foil, 0, 0)
			showCardList(cardList, detail)
		} else {
			ShowCard(cards)
		}
		return nil
	},
}

func ShowCard(cardids []string) {
	client := storageConnect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storageDisconnect(client)

	for _, v := range cardids {
		if len(strings.Split(v, "/")) < 2 || strings.Split(v, "/")[1] == "" {
			LogMessage(fmt.Sprintf("Invalid card %s", v), "red")
			continue
		}

		cards, _ := coll.storageFind(bson.D{{"set", strings.Split(v, "/")[0]}, {"collectornumber", strings.Split(v, "/")[1]}}, bson.D{{"name", 1}}, 0, 0)

		for _, card := range cards {
			showCardDetails(&card)
		}
	}
}

func Cards(rarity, set, sortby, name, oracle, cardType string, reserved, foil bool, skip, limit int64) []Card {
	client := storageConnect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storageDisconnect(client)

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
		if getCurrency() == EUR {
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

	if reserved {
		filter = append(filter, bson.E{"reserved", true})
	}

	if foil {
		filter = append(filter, bson.E{"serra_count_foil", bson.D{{"$gt", 0}}})
	}

	cards, _ := coll.storageFind(filter, sortStage, skip, limit)

	// This is needed because collectornumbers are strings (ie. "23a") but still we
	// want it to be sorted numerically ... 1,2,3,10,11,100.
	if sortby == "number" {
		sort.Slice(cards, func(i, j int) bool {
			return filterForDigits(cards[i].CollectorNumber) < filterForDigits(cards[j].CollectorNumber)
		})
	}

	// filter out cards that do not reach the minimum amount (--min-count)
	// this is done after query result because find query constructed does not support
	// aggregating fields (of count and countFoil).
	temp := cards[:0]
	for _, card := range cards {
		if (card.SerraCount + card.SerraCountFoil) >= count {
			temp = append(temp, card)
		}
	}
	cards = temp

	return cards
}

func showCardList(cards []Card, detail bool) {

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

func showCardDetails(card *Card) error {
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
	showPriceHistory(card.SerraPrices, "* ", false)
	fmt.Println()
	return nil
}
