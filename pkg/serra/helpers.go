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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Rarities struct {
	Rares, Uncommons, Commons, Mythics float64
}

var (
	DarkGray  = termColor.RGB(68, 71, 90).SprintfFunc()
	LightGray = termColor.RGB(98, 114, 164).SprintfFunc()
	Cyan      = termColor.RGB(139, 233, 253).SprintfFunc()
	Green     = termColor.RGB(80, 250, 123).SprintfFunc()
	Pink      = termColor.RGB(255, 121, 198).SprintfFunc()
	Purple    = termColor.RGB(189, 147, 249).SprintfFunc()
	Red       = termColor.RGB(255, 85, 85).SprintfFunc()
	Yellow    = termColor.RGB(241, 250, 140).SprintfFunc()
)

func Logger() *log.Logger {

	l := log.New(os.Stderr)
	l.SetReportTimestamp(false)
	return l
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

// convertManaSymbols converts a slice of mana symbols to a string of their
// corresponding colors. If the slice is empty, "None" is returned (prohibited
// sign for lands). The function iterates over the slice and appends the
// corresponding color to the result string based on the symbol.
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
			fmt.Printf("%s%s %s%s (%+.2f%%, %+.2f%s)\n", prefix, stringToTime(e.Date), Green("%.2f", value), Green(getCurrency()), diffPercent, value-before, getCurrency())
		}

		// price decreased or first or last element in history
		if (value < before) && (diffPercent < -5 || i == last) {
			fmt.Printf("%s%s %s%s (%+.2f%%, %+.2f%s)\n", prefix, stringToTime(e.Date), Red("%.2f", value), Red(getCurrency()), diffPercent, value-before, getCurrency())
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
