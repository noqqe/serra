package serra

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	rootCmd.AddCommand(topsCmd)
	rootCmd.AddCommand(flopsCmd)
	topsCmd.Flags().Float64VarP(&limit, "limit", "l", 0, "Minimum card price to be shown in analysis")
	topsCmd.Flags().BoolVarP(&sinceLastUpdate, "since-last-update", "u", false, "Show gains since last update")
	topsCmd.Flags().BoolVarP(&sinceBeginning, "since-beginning", "b", true, "Show gains since beginning of records")
	flopsCmd.Flags().Float64VarP(&limit, "limit", "l", 0, "Minimum card price to be shown in analysis")
	flopsCmd.Flags().BoolVarP(&sinceLastUpdate, "since-last-update", "u", false, "Show losses since last update")
	flopsCmd.Flags().BoolVarP(&sinceBeginning, "since-beginning", "b", true, "Show losses since beginning of records")
}

var topsCmd = &cobra.Command{
	Aliases:       []string{"t"},
	Use:           "tops",
	Short:         "What cards gained most value",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		Gains(limit, -1)
		return nil
	},
}

var flopsCmd = &cobra.Command{
	Aliases:       []string{"f"},
	Use:           "flops",
	Short:         "What cards lost most value",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		Gains(limit, 1)
		return nil
	},
}

// historicValue returns the value of a value history snapshot in the
// configured currency. Both inventory entry and set/total value history
// entries hold their relevant value in the Eur/Usd fields.
func historicValue(p PriceEntry) float64 {
	if getCurrency() == EUR {
		return p.Eur
	}
	return p.Usd
}

type gainRate struct {
	Old, Current, Rate float64
}

// rateAt computes the rate of change between the "old" (0 = beginning, -2 =
// second to last) and the last entry of a value history, or ok=false if
// there is not enough history or the old value is below limit.
func rateAt(history []float64, old int, limit float64) (gainRate, bool) {
	if len(history) == 0 {
		return gainRate{}, false
	}
	oldIdx := old
	if oldIdx < 0 {
		oldIdx = len(history) + oldIdx
	}
	if oldIdx < 0 || oldIdx >= len(history) {
		return gainRate{}, false
	}

	oldVal := history[oldIdx]
	curVal := history[len(history)-1]
	if oldVal <= limit {
		return gainRate{}, false
	}

	rate := (curVal/(oldVal/100) - 100)
	return gainRate{Old: oldVal, Current: curVal, Rate: rate}, true
}

func Gains(limit float64, sortDir int) error {
	client := storageConnect()
	invColl := client.getInventoryCollection()
	cardsColl := client.getCardsCollection()
	setscoll := client.getSetsCollection()
	defer storageDisconnect(client)

	var old int
	if sinceBeginning {
		old = 0
	}
	if sinceLastUpdate {
		old = -2
	}

	// Card (inventory entry) level gains
	entries, _ := invColl.FindInventoryEntries(bson.D{}, bson.D{}, 0, 0)

	type cardGain struct {
		Entry InventoryEntry
		gainRate
	}
	var cardGains []cardGain
	for _, e := range entries {
		history := make([]float64, len(e.ValueHistory))
		for i, p := range e.ValueHistory {
			history[i] = historicValue(p)
		}
		if r, ok := rateAt(history, old, limit); ok {
			cardGains = append(cardGains, cardGain{e, r})
		}
	}

	sort.Slice(cardGains, func(i, j int) bool {
		if sortDir < 0 {
			return cardGains[i].Rate > cardGains[j].Rate
		}
		return cardGains[i].Rate < cardGains[j].Rate
	})
	if len(cardGains) > 20 {
		cardGains = cardGains[:20]
	}

	ids := make([]string, 0, len(cardGains))
	seen := map[string]bool{}
	for _, g := range cardGains {
		if !seen[g.Entry.CardID] {
			seen[g.Entry.CardID] = true
			ids = append(ids, g.Entry.CardID)
		}
	}
	cardsByID, _ := cardsColl.FindCardsByIDs(ids)

	// Set level gains
	sets, _ := setscoll.FindSet(bson.D{}, bson.D{})
	type setGain struct {
		Set Set
		gainRate
	}
	var setGains []setGain
	for _, s := range sets {
		history := make([]float64, len(s.PriceList))
		for i, p := range s.PriceList {
			history[i] = historicValue(p)
		}
		if r, ok := rateAt(history, old, limit); ok {
			setGains = append(setGains, setGain{s, r})
		}
	}

	sort.Slice(setGains, func(i, j int) bool {
		if sortDir < 0 {
			return setGains[i].Rate > setGains[j].Rate
		}
		return setGains[i].Rate < setGains[j].Rate
	})
	if len(setGains) > 10 {
		setGains = setGains[:10]
	}

	fmt.Printf("%s\n", Purple("Cards"))
	for _, g := range cardGains {
		c := cardsByID[g.Entry.CardID]
		fmt.Printf("%+.0f%% %s %s (%.2f->%s%s) \n", g.Rate, c.Name, Yellow("(%s/%s)", c.Set, c.CollectorNumber), g.Old, Green("%.2f", g.Current), Green(getCurrency()))
	}

	fmt.Printf("\n%s\n", Purple("Sets"))
	for _, g := range setGains {
		fmt.Printf("%+.0f%% %s %s (%.2f->%s%s)\n", g.Rate, g.Set.Name, Yellow("(%s)", g.Set.Code), g.Old, Green("%.2f", g.Current), Green(getCurrency()))
	}
	return nil

}
