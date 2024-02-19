package serra

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
	"unicode"

	"github.com/charmbracelet/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Rarities struct {
	Rares, Uncommons, Commons, Mythics float64
}

var (
	Icon        = "\U0001F9D9\U0001F3FC"
	Reset       = "\033[0m"
	Background  = "\033[38;5;59m"
	CurrentLine = "\033[38;5;60m"
	Foreground  = "\033[38;5;231m"
	Comment     = "\033[38;5;103m"
	Cyan        = "\033[38;5;159m"
	Green       = "\033[38;5;120m"
	Orange      = "\033[38;5;222m"
	Pink        = "\033[38;5;212m"
	Purple      = "\033[38;5;183m"
	Red         = "\033[38;5;210m"
	Yellow      = "\033[38;5;229m"
)

func Logger() *log.Logger {

	l := log.New(os.Stderr)
	l.SetReportTimestamp(false)
	return l
}

func modifyCardCount(coll *Collection, c *Card, amount int64, foil bool) error {

	// find already existing card
	sort := bson.D{{"_id", 1}}
	searchFilter := bson.D{{"_id", c.ID}}
	l := Logger()
	storedCards, err := coll.storageFind(searchFilter, sort, 0, 0)
	if err != nil {
		return err
	}
	storedCard := storedCards[0]

	// update card amount
	var update bson.M
	if foil {
		update = bson.M{
			"$set": bson.M{"serra_count_foil": storedCard.SerraCountFoil + amount},
		}
	} else {
		update = bson.M{
			"$set": bson.M{"serra_count": storedCard.SerraCount + amount},
		}
	}

	coll.storageUpdate(bson.M{"_id": bson.M{"$eq": c.ID}}, update)

	var total int64
	if foil {
		total = storedCard.SerraCountFoil + amount
		if amount < 0 {
			l.Warnf("Reduced card amount of \"%s\" (%.2f%s, foil) from %d to %d", storedCard.Name, storedCard.getValue(true), getCurrency(), storedCard.SerraCountFoil, total)
		} else {
			l.Warnf("Increased card amount of \"%s\" (%.2f%s, foil) from %d to %d", storedCard.Name, storedCard.getValue(true), getCurrency(), storedCard.SerraCountFoil, total)
		}
	} else {
		total = storedCard.SerraCount + amount
		if amount < 0 {
			l.Warnf("Reduced card amount of \"%s\" (%.2f%s) from %d to %d", storedCard.Name, storedCard.getValue(false), getCurrency(), storedCard.SerraCount, total)
		} else {
			l.Warnf("Increased card amount of \"%s\" (%.2f%s) from %d to %d", storedCard.Name, storedCard.getValue(false), getCurrency(), storedCard.SerraCount, total)
		}
	}

	return nil
}

func findCardByCollectorNumber(coll *Collection, setCode string, collectorNumber string) (*Card, error) {
	sort := bson.D{{"_id", 1}}
	searchFilter := bson.D{{"set", setCode}, {"collectornumber", collectorNumber}}
	storedCards, err := coll.storageFind(searchFilter, sort, 0, 0)
	if err != nil {
		return &Card{}, err
	}

	if len(storedCards) < 1 {
		return &Card{}, errors.New("Card not found")
	}

	return &storedCards[0], nil
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
	storedSets, err := coll.storageFindSet(bson.D{{"code", setcode}}, bson.D{{"_id", 1}})
	if err != nil {
		return &Set{}, err
	}

	if len(storedSets) < 1 {
		return &Set{}, errors.New("Set not found")
	}

	return &storedSets[0], nil
}

func convertManaSymbols(sym []interface{}) string {
	var mana string

	if len(sym) == 0 {
		mana = mana + "None" //probibited sign for lands
	}

	for _, v := range sym {
		switch v {
		case "B":
			mana = mana + "Black" //black
		case "R":
			mana = mana + "Red" //red
		case "G":
			mana = mana + "Green" //green
		case "U":
			mana = mana + "Blue" //blue
		case "W":
			mana = mana + "White" //white
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
			if getCurrency() == EUR {
				value = e.Eur + e.EurFoil
			} else {
				value = e.Usd + e.UsdFoil
			}
		} else {
			if getCurrency() == EUR {
				value = e.Eur
			} else {
				value = e.Usd
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
		return math.NaN(), errors.New("non-numeric type could not be converted to float")
	}
}
