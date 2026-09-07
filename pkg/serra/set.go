package serra

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	setCmd.Flags().StringVarP(&sortBy, "sort", "s", "release", "How to sort cards (release/value)")
	setCmd.Flags().StringVarP(&setType, "type", "t", "all", "Filter on set type (core/expansion/masters/commander/all)")
	rootCmd.AddCommand(setCmd)
}

type SetsResult struct {
	ID      string
	Code    string
	Value   float64
	Count   int64
	Unique  int64
	Release string
}

var setCmd = &cobra.Command{
	Aliases: []string{"cards"},
	Use:     "set [set]",
	Short:   "Search & show sets from your collection",
	Long: `Search and show sets from your collection.
If you directly put a setcode as an argument, it will be displayed
otherwise you'll get a list of sets as a search result.`,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, sets []string) error {
		if len(sets) == 0 {
			setList := Sets(sortBy, setType)
			showSetList(setList)
		} else {
			for _, set := range sets {
				ShowSet(set)
			}
		}
		return nil
	},
}

func Sets(sortBy string, filter string) []SetsResult {
	l := Logger()

	cardFilter := bson.D{}
	if filter != "all" {
		cardFilter = append(cardFilter, bson.E{"settype", filter})
	}

	owned, err := OwnedCards(cardFilter)
	if err != nil {
		l.Error("Error fetching sets:", err)
		return nil
	}

	bySet := map[string]*SetsResult{}
	order := []string{}
	for _, c := range owned {
		r, ok := bySet[c.SetName]
		if !ok {
			r = &SetsResult{ID: c.SetName, Code: c.Set, Release: c.ReleasedAt}
			bySet[c.SetName] = r
			order = append(order, c.SetName)
		}
		r.Value += c.getValue()*float64(c.Count) + c.getFoilValue()*float64(c.CountFoil) + c.getEtchedValue()*float64(c.CountEtched)
		r.Count += c.Count + c.CountFoil + c.CountEtched
		r.Unique++
	}

	sets := make([]SetsResult, 0, len(order))
	for _, k := range order {
		sets = append(sets, *bySet[k])
	}

	switch sortBy {
	case "value":
		sort.Slice(sets, func(i, j int) bool { return sets[i].Value < sets[j].Value })
	default:
		sort.Slice(sets, func(i, j int) bool { return sets[i].Release < sets[j].Release })
	}

	return sets
}

func showSetList(sets []SetsResult) {

	client := storageConnect()
	setscoll := client.getSetsCollection()

	for _, set := range sets {
		setobj, _ := setscoll.FindSetByCode(set.Code)
		fmt.Printf("* %s %s (%s)\n", fmt.Sprintf("%.4s", set.Release), Purple(set.ID), Cyan(set.Code))
		fmt.Printf("  Cards: %s Total: %d \n", Yellow("%d/%d", set.Unique, setobj.CardCount), set.Count)
		fmt.Printf("  Value: %s%s\n", Pink("%.2f", set.Value), Pink(getCurrency()))
		fmt.Println()
	}
}

func ShowSet(setname string) error {

	client := storageConnect()
	l := Logger()
	defer storageDisconnect(client)

	cards, err := OwnedCards(bson.D{{"set", setname}})
	if (err != nil) || len(cards) == 0 {
		l.Errorf("Set %s not found or no card in your collection.", setname)
		return err
	}

	// sort cards by value, most valuable first, for the "Most valuable cards" section
	sort.Slice(cards, func(i, j int) bool { return cards[i].getValue() > cards[j].getValue() })

	// fetch set informations
	setcoll := client.getSetsCollection()
	set, err := setcoll.FindSetByCode(setname)
	if err != nil {
		l.Errorf("Set %s not found or no card in your collection.", setname)
		return err
	}

	var normalValue, foilValue, etchedValue float64
	var normalCount, foilCount, etchedCount int64
	for _, c := range cards {
		normalValue += c.getValue() * float64(c.Count)
		foilValue += c.getFoilValue() * float64(c.CountFoil)
		etchedValue += c.getEtchedValue() * float64(c.CountEtched)
		normalCount += c.Count
		foilCount += c.CountFoil
		etchedCount += c.CountEtched
	}
	totalValue := normalValue + foilValue + etchedValue

	ri := rarityBreakdown(cards)

	fmt.Printf("%s\n", Green(set.Name))
	fmt.Printf("Type: %s\n", set.SetType)
	fmt.Printf("Released: %s\n", set.ReleasedAt)
	fmt.Printf("Set Cards: %d/%d\n", len(cards), set.CardCount)
	fmt.Printf("Total Cards: %d\n", normalCount+foilCount+etchedCount)
	fmt.Printf("Foil Cards: %d\n", foilCount)

	fmt.Printf("\n%s\n", Purple("Current Value"))
	fmt.Printf("Total: %dx %s%s\n", normalCount+foilCount+etchedCount, Yellow("%.2f", totalValue), Yellow(getCurrency()))
	fmt.Printf("Normal: %dx %s%s\n", normalCount, Yellow("%.2f", normalValue), Yellow(getCurrency()))
	fmt.Printf("Foil: %dx %s%s\n", foilCount, Yellow("%.2f", foilValue), Yellow(getCurrency()))
	if etchedCount > 0 {
		fmt.Printf("Etched: %dx %s%s\n", etchedCount, Yellow("%.2f", etchedValue), Yellow(getCurrency()))
	}

	fmt.Printf("\n%s\n", Purple("Rarities"))
	fmt.Printf("Mythics: %.0f\n", ri.Mythics)
	fmt.Printf("Rares: %.0f\n", ri.Rares)
	fmt.Printf("Uncommons: %.0f\n", ri.Uncommons)
	fmt.Printf("Commons: %.0f\n", ri.Commons)

	fmt.Printf("\n%s\n", Pink("Price History"))
	showPriceHistory(set.PriceList, "* ", true)

	fmt.Printf("\n%s\n", Pink("Most valuable cards"))

	// Calc counter to show 10 cards or less
	ccards := min(10, len(cards))
	for _, card := range cards[:ccards] {
		fmt.Printf("* %s (%s/%s) %s%s\n", Purple(card.Name), set.Code, card.CollectorNumber, Yellow("%.2f", card.getValue()), Yellow(getCurrency()))
	}

	return nil
}
