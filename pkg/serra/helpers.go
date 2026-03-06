package serra

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/log"
	termColor "github.com/fatih/color"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Rarities struct {
	Rares, Uncommons, Commons, Mythics float64
}

var (
	DarkGray  = termColor.New(termColor.FgHiBlack).SprintfFunc()
	LightGray = termColor.New(termColor.FgWhite).SprintfFunc()
	Cyan      = termColor.New(termColor.FgHiCyan).SprintfFunc()
	Green     = termColor.New(termColor.FgGreen).SprintfFunc()
	Pink      = termColor.New(termColor.FgHiMagenta).SprintfFunc()
	Purple    = termColor.New(termColor.FgMagenta).SprintfFunc()
	Red       = termColor.New(termColor.FgRed).SprintfFunc()
	Yellow    = termColor.New(termColor.FgYellow).SprintfFunc()
)

func Logger() *log.Logger {

	l := log.New(os.Stderr)
	l.SetReportTimestamp(false)
	return l
}

// modifyCardCount modifies the amount of a card in the collection by a given
// amount in foil or nonfoil
func modifyCardCount(coll *Collection, c *Card, amount int64, foil bool) error {

	l := Logger()
	storedCard, err := findCardByCollectorNumber(coll, c.Set, c.CollectorNumber)
	if err != nil {
		return err
	}

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
			l.Warnf("Reduced card amount of \"%s\" (%.2f%s, foil) from %d to %d", storedCard.Name, storedCard.getFoilValue(), getCurrency(), storedCard.SerraCountFoil, total)
		} else {
			l.Warnf("Increased card amount of \"%s\" (%.2f%s, foil) from %d to %d", storedCard.Name, storedCard.getFoilValue(), getCurrency(), storedCard.SerraCountFoil, total)
		}
	} else {
		total = storedCard.SerraCount + amount
		if amount < 0 {
			l.Warnf("Reduced card amount of \"%s\" (%.2f%s) from %d to %d", storedCard.Name, storedCard.getValue(), getCurrency(), storedCard.SerraCount, total)
		} else {
			l.Warnf("Increased card amount of \"%s\" (%.2f%s) from %d to %d", storedCard.Name, storedCard.getValue(), getCurrency(), storedCard.SerraCount, total)
		}
	}

	return nil
}

// Find a card in the collection by set code and collector number. Returns an error if not found
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

// findSetByCode finds a set in the collection by its code. Returns an error if not found
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

// parseCardID parses a card ID in the format
// "set/collector_number" and returns the set code and collector number.
// Returns an error if the format is invalid
func parseCardID(cardID string) (string, string, error) {
	l := Logger()
	if len(strings.Split(cardID, "/")) < 2 || strings.Split(cardID, "/")[1] == "" {
		l.Warnf("Invalid card %s", cardID)
		return "", "", errors.New("set code and collector number must be provided in format set/collector_number (ex. usg/23)")
	}

	setCode := strings.Split(cardID, "/")[0]
	collectorNumber := strings.Split(cardID, "/")[1]

	if collectorNumber == "" {
		l.Errorf("Invalid card format %s. Needs to be set/collector number i.e. \"usg/13\"", cardID)
		return "", "", errors.New("set code and collector number must be provided in format set/collector_number (ex. usg/23)")
	}

	return setCode, collectorNumber, nil
}

func convertManaSymbols(sym []any) string {
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

// HACK:
// this is maybe the ugliest way someone could choose to verify, if a rarity type is missing
// [
// { _id: { rarity: 'common' }, count: 20 },
// { _id: { rarity: 'uncommon' }, count: 2 }
// ]
// if a result like this is there, 1 rarity type "rare" is not in the array. and needs to be
// initialized with 0, otherwise we get a panic
func convertRarities(rar []primitive.M) Rarities {

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
	last := len(prices)
	for i, e := range prices {

		var value float64

		// check if a total sum is going to be printed
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

		// calculate percent difference
		diffPercent := (value / before * 100) - 100

		// always display first history entry
		if i == 0 {
			fmt.Printf("%s%s %.2f%s\n", prefix, stringToTime(e.Date), value, getCurrency())
			before = value
			continue
		}

		// price increased or first or last element in history
		if (value >= before && before != 0) && (diffPercent > 5 || i+1 == last) {
			fmt.Printf("%s%s %s%s (%.2f%%, %+.2f%s)\n", prefix, stringToTime(e.Date), Green("%+.2f", value), Green(getCurrency()), diffPercent, value-before, getCurrency())
		}

		// price decreased or first or last element in history
		if (value < before) && (diffPercent < -5 || i == last) {
			fmt.Printf("%s%s %s%s (%.2f%%, %+.2f%s)\n", prefix, stringToTime(e.Date), Red("%+.2f", value), Red(getCurrency()), diffPercent, value-before, getCurrency())
		}

		before = value

	}
}

func filterForDigits(str string) int {
	var numStr strings.Builder
	for _, c := range str {
		if unicode.IsDigit(c) {
			numStr.WriteString(string(c))
		}
	}
	s, _ := strconv.Atoi(numStr.String())
	return s
}

func getFloat64(unknown any) (float64, error) {
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
