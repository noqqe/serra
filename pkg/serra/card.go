package serra

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	cardCmd.Flags().StringVarP(&artist, "artist", "a", "", "Filter by name of artist")
	cardCmd.Flags().StringVarP(&rarity, "rarity", "r", "", "Filter by rarity of cards (mythic, rare, uncommon, common)")
	cardCmd.Flags().StringVarP(&set, "set", "e", "", "Filter by set code (usg/mmq/vow)")
	cardCmd.Flags().StringVarP(&sortBy, "sort", "s", "name", "How to sort cards (value/number/name/added/count)")
	cardCmd.Flags().StringVarP(&name, "name", "n", "", "Name of the card (regex compatible)")
	cardCmd.Flags().Int64VarP(&cmc, "cmc", "m", -1, "Cumulative mana cost of card")
	cardCmd.Flags().StringVarP(&color, "color", "i", "", "Color identity of card (w,u,b,r,g)")
	cardCmd.Flags().StringVarP(&oracle, "oracle", "o", "", "Contains string in card text")
	cardCmd.Flags().StringVarP(&cardType, "type", "t", "", "Contains string in card type line")
	cardCmd.Flags().Int64VarP(&count, "min-count", "c", 0, "Occource more than X in your collection")
	cardCmd.Flags().BoolVarP(&detail, "detail", "d", false, "Show details for cards (url)")
	cardCmd.Flags().BoolVarP(&reserved, "reserved", "w", false, "If card is on reserved list")
	cardCmd.Flags().BoolVarP(&foil, "foil", "f", false, "If card is foil list")
	cardCmd.Flags().StringVarP(&is, "is", "y", "", "If card has certain attribute")
	cardCmd.Flags().StringVarP(&isNot, "isnot", "x", "", "If card does not have certain attribute")
	cardCmd.Flags().StringVarP(&legal, "legal", "l", "", "Filter by format legality (standard, pioneer, modern, legacy, vintage, commander, pauper, premodern, ...)")
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
			cardList := Cards(rarity, set, sortBy, name, oracle, cardType, reserved, foil, 0, 0, is, isNot, legal)
			showCardList(cardList, detail)
		} else {
			for _, card := range cards {
				showCard(card)
			}
		}
		return nil
	},
}

// OwnedCard combines Scryfall card data with the aggregated inventory data
// (across all finishes/languages/conditions) owned for that card.
type OwnedCard struct {
	Card

	Count       int64
	CountFoil   int64
	CountEtched int64
	Created     primitive.DateTime
	Updated     primitive.DateTime
	Entries     []InventoryEntry
}

// OwnedCards joins the inventory collection with the cards collection,
// returning only cards that are actually owned (i.e. have at least one
// inventory entry), matching the given Scryfall attribute filter.
func OwnedCards(cardFilter bson.D) ([]OwnedCard, error) {
	client := storageConnect()
	defer storageDisconnect(client)

	invColl := client.getInventoryCollection()
	cardsColl := client.getCardsCollection()

	entries, err := invColl.FindInventoryEntries(bson.D{}, bson.D{}, 0, 0)
	if err != nil {
		return nil, err
	}

	type aggregate struct {
		Count, CountFoil, CountEtched int64
		Created, Updated              primitive.DateTime
		Entries                       []InventoryEntry
	}

	byCard := map[string]*aggregate{}
	order := []string{}
	for _, e := range entries {
		a, ok := byCard[e.CardID]
		if !ok {
			a = &aggregate{Created: e.Created, Updated: e.Updated}
			byCard[e.CardID] = a
			order = append(order, e.CardID)
		}

		switch e.Finish {
		case FinishFoil:
			a.CountFoil += e.Count
		case FinishEtched:
			a.CountEtched += e.Count
		default:
			a.Count += e.Count
		}

		if e.Created < a.Created {
			a.Created = e.Created
		}
		if e.Updated > a.Updated {
			a.Updated = e.Updated
		}
		a.Entries = append(a.Entries, e)
	}

	ids := make([]string, 0, len(order))
	for _, id := range order {
		ids = append(ids, id)
	}

	filter := append(bson.D{{"_id", bson.D{{"$in", ids}}}}, cardFilter...)
	cards, err := cardsColl.FindCards(filter, bson.D{}, 0, 0)
	if err != nil {
		return nil, err
	}

	owned := make([]OwnedCard, 0, len(cards))
	for _, c := range cards {
		a := byCard[c.ID]
		owned = append(owned, OwnedCard{
			Card:        c,
			Count:       a.Count,
			CountFoil:   a.CountFoil,
			CountEtched: a.CountEtched,
			Created:     a.Created,
			Updated:     a.Updated,
			Entries:     a.Entries,
		})
	}

	return owned, nil
}

// FindOwnedCard returns the single owned card for a given set + collector
// number, including its inventory breakdown.
func FindOwnedCard(setCode, collectorNumber string) (*OwnedCard, error) {
	owned, err := OwnedCards(bson.D{{"set", setCode}, {"collectornumber", collectorNumber}})
	if err != nil {
		return nil, err
	}
	if len(owned) < 1 {
		return nil, errors.New("Card not found")
	}
	return &owned[0], nil
}

// Cards fetches card based on search parameters
// TODO:Create search object instead of a bazillion parameters
func Cards(rarity, set, sortBy, name, oracle, cardType string, reserved, foil bool, skip, limit int64, is, isNot, legal string) []OwnedCard {
	filter := bson.D{}

	switch rarity {
	case "uncommon":
		filter = append(filter, bson.E{"rarity", "uncommon"})
	case "common":
		filter = append(filter, bson.E{"rarity", "common"})
	case "rare":
		filter = append(filter, bson.E{"rarity", "rare"})
	case "mythic":
		filter = append(filter, bson.E{"rarity", "mythic"})
	}

	if len(set) > 0 {
		filter = append(filter, bson.E{"set", set})
	}

	if len(name) > 0 {
		filter = append(filter, bson.E{"name", bson.D{{"$regex", ".*" + name + ".*"}, {"$options", "i"}}})
	}

	if len(artist) > 0 {
		filter = append(filter, bson.E{"artist", bson.D{{"$regex", ".*" + artist + ".*"}, {"$options", "i"}}})
	}

	if cmc > -1 {
		filter = append(filter, bson.E{"cmc", cmc})
	}

	if len(oracle) > 0 {
		filter = append(filter, bson.E{"oracletext", bson.D{{"$regex", ".*" + oracle + ".*"}, {"$options", "i"}}})
	}

	if len(cardType) > 0 {
		filter = append(filter, bson.E{"typeline", bson.D{{"$regex", ".*" + cardType + ".*"}, {"$options", "i"}}})
	}

	if len(color) > 0 {
		colorArr := strings.Split(strings.ToUpper(color), ",")
		filter = append(filter, bson.E{"coloridentity", colorArr})
	}

	if len(is) > 0 {
		filter = append(filter, bson.E{is, true})
	}

	if len(isNot) > 0 {
		filter = append(filter, bson.E{isNot, false})
	}

	if reserved {
		filter = append(filter, bson.E{"reserved", true})
	}

	if len(legal) > 0 {
		filter = append(filter, bson.E{"legalities." + strings.ToLower(legal), "legal"})
	}

	cards, _ := OwnedCards(filter)

	if foil {
		temp := cards[:0]
		for _, card := range cards {
			if card.CountFoil > 0 {
				temp = append(temp, card)
			}
		}
		cards = temp
	}

	// filter out cards that do not reach the minimum amount (--min-count)
	temp := cards[:0]
	for _, card := range cards {
		if (card.Count + card.CountFoil) >= count {
			temp = append(temp, card)
		}
	}
	cards = temp

	switch sortBy {
	case "value":
		sort.Slice(cards, func(i, j int) bool { return cards[i].getValue() < cards[j].getValue() })
	case "number":
		// This is needed because collectornumbers are strings (ie. "23a") but still we
		// want it to be sorted numerically ... 1,2,3,10,11,100.
		sort.Slice(cards, func(i, j int) bool {
			return filterForDigits(cards[i].CollectorNumber) < filterForDigits(cards[j].CollectorNumber)
		})
	case "added":
		sort.Slice(cards, func(i, j int) bool { return cards[i].Created < cards[j].Created })
	case "count":
		sort.Slice(cards, func(i, j int) bool {
			return (cards[i].Count + cards[i].CountFoil) < (cards[j].Count + cards[j].CountFoil)
		})
	default:
		sort.Slice(cards, func(i, j int) bool { return cards[i].Name < cards[j].Name })
	}

	if skip > 0 || limit > 0 {
		if skip > int64(len(cards)) {
			return []OwnedCard{}
		}
		end := int64(len(cards))
		if limit > 0 && skip+limit < end {
			end = skip + limit
		}
		cards = cards[skip:end]
	}

	return cards
}

func showCard(cardID string) error {
	setCode, collectorNumber, err := parseCardID(cardID)
	if err != nil {
		return err
	}

	l := Logger()
	card, err := FindOwnedCard(setCode, collectorNumber)
	if err != nil {
		// Not (or no longer) in the inventory - fall back to the cached
		// Scryfall data so the card can still be displayed.
		client := storageConnect()
		defer storageDisconnect(client)

		scryfallCard, ferr := client.getCardsCollection().FindCardByCollectorNumber(setCode, collectorNumber)
		if ferr != nil {
			l.Errorf("Card %s not found", cardID)
			return ferr
		}
		card = &OwnedCard{Card: *scryfallCard}
	}

	showCardDetails(card)
	return nil
}

func showCardList(cards []OwnedCard, detail bool) {

	var total float64
	if detail {
		for _, card := range cards {
			fmt.Printf("* %dx %s (%s/%s) %s%s %s\n", card.Count+card.CountFoil+card.CountEtched, Purple(card.Name), card.Set, card.CollectorNumber, Yellow("%.2f", card.getValue()), Yellow(getCurrency()), DarkGray(strings.Replace(card.ScryfallURI, "?utm_source=api", "", 1)))
			total = total + card.getValue()*float64(card.Count) + card.getFoilValue()*float64(card.CountFoil) + card.getEtchedValue()*float64(card.CountEtched)
		}
	} else {
		for _, card := range cards {
			fmt.Printf("* %dx %s (%s/%s) %s%s\n", card.Count+card.CountFoil+card.CountEtched, Purple(card.Name), card.Set, card.CollectorNumber, Yellow("%.2f", card.getValue()), Yellow(getCurrency()))
			total = total + card.getValue()*float64(card.Count) + card.getFoilValue()*float64(card.CountFoil) + card.getEtchedValue()*float64(card.CountEtched)
		}
	}

	fmt.Printf("\nTotal Value: %s%s\n", Yellow("%.2f", total), Yellow(getCurrency()))

}

func showCardDetails(card *OwnedCard) error {
	inInventory := card.Count+card.CountFoil+card.CountEtched > 0

	fmt.Printf("%s (%s/%s)\n", Purple(card.Name), card.Set, card.CollectorNumber)
	if inInventory {
		fmt.Printf("Status: %s\n", Green("In Inventory"))
		fmt.Printf("Added: %s\n", stringToTime(card.Created))
	} else {
		fmt.Printf("Status: %s\n", Red("Not in Inventory"))
	}
	fmt.Printf("Rarity: %s\n", card.Rarity)
	fmt.Printf("Scryfall: %s\n", strings.Replace(card.ScryfallURI, "?utm_source=api", "", 1))

	fmt.Printf("\n%s\n", Green("Current Values"))
	fmt.Printf("* Normal: %dx %s%s %s\n", card.Count, Yellow("%.2f", card.getValue()), Yellow(getCurrency()), DarkGray("(%.2f)", float64(card.Count)*card.getValue()))
	fmt.Printf("* Foil: %dx %s%s %s\n", card.CountFoil, Yellow("%.2f", card.getFoilValue()), Yellow(getCurrency()), DarkGray("(%.2f)", float64(card.CountFoil)*card.getFoilValue()))
	if card.CountEtched > 0 {
		fmt.Printf("* Etched: %dx %s%s %s\n", card.CountEtched, Yellow("%.2f", card.getEtchedValue()), Yellow(getCurrency()), DarkGray("(%.2f)", float64(card.CountEtched)*card.getEtchedValue()))
	}

	if inInventory {
		for _, e := range card.Entries {
			fmt.Printf("\n%s\n", Green(fmt.Sprintf("Value History (%s, %s, %s)", e.Finish, e.Language, e.Condition)))
			showPriceHistory(e.ValueHistory, "* ", false)
		}
	} else if len(card.PriceHistory) > 0 {
		// Not owned, so there's no per-entry value history - fall back to the
		// Scryfall price history cached for every card regardless of
		// ownership, narrowed down to the finishes this printing exists in.
		if card.Nonfoil {
			fmt.Printf("\n%s\n", Green("Value History (Nonfoil)"))
			showPriceHistory(priceHistoryForFinish(card.PriceHistory, FinishNonfoil), "* ", false)
		}
		if card.Foil {
			fmt.Printf("\n%s\n", Green("Value History (Foil)"))
			showPriceHistory(priceHistoryForFinish(card.PriceHistory, FinishFoil), "* ", false)
		}
	}
	return nil
}
