package serra

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Rarities struct {
	Rares, Uncommons, Commons, Mythics float64
}

func modify_count_of_card(coll *Collection, c *Card, amount int64) error {

	// find already existing card
	sort := bson.D{{"_id", 1}}
	search_filter := bson.D{{"_id", c.ID}}
	stored_cards, err := coll.storage_find(search_filter, sort)
	if err != nil {
		return err
	}
	stored_card := stored_cards[0]

	// update card amount
	update_filter := bson.M{"_id": bson.M{"$eq": c.ID}}
	update := bson.M{
		"$set": bson.M{"serra_count": stored_card.SerraCount + amount},
	}
	coll.storage_update(update_filter, update)

	LogMessage(fmt.Sprintf("Updating Card \"%s\" amount to %d", stored_card.Name, stored_card.SerraCount+amount), "purple")
	return nil
}

func find_card_by_setcollectornumber(coll *Collection, setcode string, collectornumber string) (*Card, error) {

	sort := bson.D{{"_id", 1}}
	search_filter := bson.D{{"set", setcode}, {"collectornumber", collectornumber}}
	stored_cards, err := coll.storage_find(search_filter, sort)
	if err != nil {
		return &Card{}, err
	}

	if len(stored_cards) < 1 {
		return &Card{}, errors.New("Card not found")
	}

	return &stored_cards[0], nil
}

func stringToTime(s primitive.DateTime) string {
	return time.UnixMilli(int64(s)).Format("2006-01-02")
}

// missing compares two slices and returns slice of differences
func missing(a, b []string) []string {
	type void struct{}
	// create map with length of the 'a' slice
	ma := make(map[string]void, len(a))
	diffs := []string{}
	// Convert first slice to map with empty struct (0 bytes)
	for _, ka := range a {
		ma[ka] = void{}
	}
	// find missing values in a
	for _, kb := range b {
		if _, ok := ma[kb]; !ok {
			diffs = append(diffs, kb)
		}
	}
	return diffs
}

func find_set_by_code(coll *Collection, setcode string) (*Set, error) {

	stored_sets, err := coll.storage_find_set(bson.D{{"code", setcode}}, bson.D{{"_id", 1}})
	if err != nil {
		return &Set{}, err
	}

	if len(stored_sets) < 1 {
		return &Set{}, errors.New("Set not found")
	}

	return &stored_sets[0], nil
}

func show_card_details(card *Card) error {
	fmt.Printf("%s%s%s (%s/%s)\n", Purple, card.Name, Reset, card.Set, card.CollectorNumber)
	fmt.Printf("Added: %s\n", stringToTime(card.SerraCreated))
	fmt.Printf("Count: %dx\n", card.SerraCount)
	fmt.Printf("Rarity: %s\n", card.Rarity)
	fmt.Printf("Scryfall: %s\n", strings.Replace(card.ScryfallURI, "?utm_source=api", "", 1))
	fmt.Printf("Current Value: %s%.2f %s%s\n", Yellow, card.getValue(), getCurrency(), Reset)

	fmt.Printf("\n%sHistory%s\n", Green, Reset)
	print_price_history(card.SerraPrices, "* ")
	fmt.Println()
	return nil
}

func convert_mana_symbols(sym []interface{}) string {
	var mana string

	if len(sym) == 0 {
		// mana = mana + "\U0001F6AB" //probibited sign for lands
		mana = mana + "None" //probibited sign for lands
	}

	for _, v := range sym {
		switch v {
		case "B":
			mana = mana + "Black" //black
			//mana = mana + "\U000026AB" //black
		case "R":
			mana = mana + "Red" //red
			// mana = mana + "\U0001F534" //red
		case "G":
			mana = mana + "Green" //green
			// mana = mana + "\U0001F7E2" //green
		case "U":
			mana = mana + "Blue" //blue
			//mana = mana + "\U0001F535" //blue
		case "W":
			mana = mana + "White" //white
			// mana = mana + "\U000026AA" //white
		}
	}
	return mana

}

func convert_rarities(rar []primitive.M) Rarities {

	// this is maybe the ugliest way someone could choose to verify, if a rarity type is missing
	// [
	// { _id: { rarity: 'common' }, count: 20 },
	// { _id: { rarity: 'uncommon' }, count: 2 }
	// ]
	// if a result like this is there, 1 rarity type "rare" is not in the array. and needs to be
	// initialized with 0, otherwise we get a panic

	var ri Rarities
	for _, r := range rar {
		switch r["_id"] {
		case "rare":
			ri.Rares = r["count"].(float64)
		case "uncommon":
			ri.Uncommons = r["count"].(float64)
		case "common":
			ri.Commons = r["count"].(float64)
		case "mythic":
			ri.Mythics = r["count"].(float64)
		}
	}
	return ri

}

func print_price_history(prices []PriceEntry, prefix string) {

	var before float64
	for _, e := range prices {

		// TODO: Make currency configurable
		value := e.Usd
		if getCurrency() == "EUR" {
			value = e.Eur
		}

		if value > before && before != 0 {
			fmt.Printf("%s%s%s %.2f %s%s (%+.2f%%)\n", prefix, stringToTime(e.Date), Green, value, getCurrency(), Reset, (value/before*100)-100)
		} else if value < before {
			fmt.Printf("%s%s%s %.2f %s%s (%+.2f%%)\n", prefix, stringToTime(e.Date), Red, value, getCurrency(), Reset, (value/before*100)-100)
		} else {
			fmt.Printf("%s%s %.2f %s%s\n", prefix, stringToTime(e.Date), value, getCurrency(), Reset)
		}
		before = value
	}
}
