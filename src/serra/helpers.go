package serra

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Rarities struct {
	Rares, Uncommons, Commons, Mythics float64
}

func modifyCardCount(coll *Collection, c *Card, amount int64, foil bool) error {

	// find already existing card
	sort := bson.D{{"_id", 1}}
	search_filter := bson.D{{"_id", c.ID}}
	stored_cards, err := coll.storageFind(search_filter, sort)
	if err != nil {
		return err
	}
	stored_card := stored_cards[0]

	// update card amount
	update_filter := bson.M{"_id": bson.M{"$eq": c.ID}}
	var update bson.M
	if foil {
		update = bson.M{
			"$set": bson.M{"serra_count_foil": stored_card.SerraCountFoil + amount},
		}
	} else {
		update = bson.M{
			"$set": bson.M{"serra_count": stored_card.SerraCount + amount},
		}
	}

	coll.storageUpdate(update_filter, update)

	var total int64
	if foil {
		total = stored_card.SerraCountFoil + amount
	} else {
		total = stored_card.SerraCount + amount
	}
	LogMessage(fmt.Sprintf("Updating Card \"%s\" amount to %d", stored_card.Name, total), "purple")
	return nil
}

func findCardbyCollectornumber(coll *Collection, setcode string, collectornumber string) (*Card, error) {

	sort := bson.D{{"_id", 1}}
	search_filter := bson.D{{"set", setcode}, {"collectornumber", collectornumber}}
	stored_cards, err := coll.storageFind(search_filter, sort)
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

func findSetByCode(coll *Collection, setcode string) (*Set, error) {

	stored_sets, err := coll.storageFindSet(bson.D{{"code", setcode}}, bson.D{{"_id", 1}})
	if err != nil {
		return &Set{}, err
	}

	if len(stored_sets) < 1 {
		return &Set{}, errors.New("Set not found")
	}

	return &stored_sets[0], nil
}

func convertManaSymbols(sym []interface{}) string {
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

func convertRarities(rar []primitive.M) Rarities {

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

func showPriceHistory(prices []PriceEntry, prefix string, total bool) {

	var before float64
	for _, e := range prices {

		var value float64
		if total {
			value = e.Usd + e.UsdFoil + e.UsdEtched
			if getCurrency() == "EUR" {
				value = e.Eur + e.EurFoil
			}
		} else {
			value = e.Usd
			if getCurrency() == "EUR" {
				value = e.Eur
			}
		}

		if value > before && before != 0 {
			fmt.Printf("%s%s%s %.2f%s%s (%+.2f%%, %+.2f%s)\n", prefix, stringToTime(e.Date), Green, value, getCurrency(), Reset, (value/before*100)-100, value-before, getCurrency())
		} else if value < before {
			fmt.Printf("%s%s%s %.2f%s%s (%+.2f%%, %+.2f%s)\n", prefix, stringToTime(e.Date), Red, value, getCurrency(), Reset, (value/before*100)-100, value-before, getCurrency())
		} else {
			fmt.Printf("%s%s %.2f%s%s\n", prefix, stringToTime(e.Date), value, getCurrency(), Reset)
		}
		before = value
	}
}

func filterForDigits(str string) int {
	var numStr string
	for _, c := range str {
		if unicode.IsDigit(c) {
			numStr += string(c)
		}
	}
	s, _ := strconv.Atoi(numStr)
	return s
}

func getFloat64(unknown interface{}) (float64, error) {
	switch i := unknown.(type) {
	case float64:
		return i, nil
	case float32:
		return float64(i), nil
	case int64:
		return float64(i), nil
	case int32:
		return float64(i), nil
	case int:
		return float64(i), nil
	case uint64:
		return float64(i), nil
	case uint32:
		return float64(i), nil
	case uint:
		return float64(i), nil
	default:
		return math.NaN(), errors.New("Non-numeric type could not be converted to float")
	}
}

// Splits string by multiple occurances of substring
// needed for calcManaCosts
func SplitAny(s string, seps string) []string {
	splitter := func(r rune) bool {
		return strings.ContainsRune(seps, r)
	}
	return strings.FieldsFunc(s, splitter)
}

// Converts mana encoding to mana costs
//
//	calcManaCosts("{2}{B}{B}") -> 4
//	calcManaCosts("{4}{G}{G}{G}{G}") -> 7
//	calcManaCosts("{1}{U} // {3}{U}") -> 2 (ignore transform costs)
func calcManaCosts(costs string) int {

	var count int

	for _, c := range SplitAny(costs, "{}") {
		if strings.Contains(c, "//") {
			break
		}

		i, err := strconv.Atoi(c)
		if err != nil {
			count = count + 1
		} else {
			count = count + i
		}
	}

	return count
}

func printUniqueValue(arr []int) map[int]int {
	//Create a dictionary of values for each element
	dict := make(map[int]int)
	for _, num := range arr {
		dict[num] = dict[num] + 1
	}

	return dict

}
