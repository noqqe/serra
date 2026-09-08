package serra

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	rootCmd.AddCommand(statsCmd)
}

var statsCmd = &cobra.Command{
	Aliases:       []string{"stats"},
	Use:           "stats",
	Short:         "Shows statistics of the collection",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		Stats()
		return nil
	},
}

func Stats() {
	client := storageConnect()
	defer storageDisconnect(client)
	totalcoll := client.getTotalCollection()

	owned, err := OwnedCards(bson.D{})
	if err != nil {
		Logger().Error("Error fetching stats:", err)
		return
	}

	// Show Value Stats
	showValueStats(owned, totalcoll)

	// Rarities
	showRarityStats(owned)

	// Reserved List
	showReservedListStats(owned)

	// Colors
	showColorStats(owned)

	// Colors
	showTypeStats(owned)

	// Artists
	showArtistStats(owned)

	// Mana Curve of Collection
	showManaCurveStats(owned)

	// Show cards added per month
	showCardsAddedPerMonth(owned)
}

func showValueStats(owned []OwnedCard, totalcoll TotalCollection) {
	var value, valueFoil, valueEtched float64
	var countNormal, countFoil, countEtched int64

	for _, c := range owned {
		value += c.getValue() * float64(c.Count)
		valueFoil += c.getFoilValue() * float64(c.CountFoil)
		valueEtched += c.getEtchedValue() * float64(c.CountEtched)
		countNormal += c.Count
		countFoil += c.CountFoil
		countEtched += c.CountEtched
	}
	countAll := countNormal + countFoil + countEtched
	totalValue := value + valueFoil + valueEtched

	fmt.Printf("%s\n", Green("Cards"))
	fmt.Printf("Total: %s\n", Yellow("%d", countAll))
	fmt.Printf("Unique: %s\n", Purple("%d", len(owned)))
	fmt.Printf("Normal: %s\n", Purple("%d", countNormal))
	fmt.Printf("Foil: %s\n", Purple("%d", countFoil))
	if countEtched > 0 {
		fmt.Printf("Etched: %s\n", Purple("%d", countEtched))
	}

	// Total Value
	fmt.Printf("\n%s\n", Green("Total Value"))
	fmt.Printf("Total: %s%s\n", Pink("%.2f", totalValue), Pink(getCurrency()))
	fmt.Printf("Normal: %s%s\n", Pink("%.2f", value), Pink(getCurrency()))
	fmt.Printf("Foils: %s%s\n", Pink("%.2f", valueFoil), Pink(getCurrency()))
	if countAll > 0 {
		fmt.Printf("Average Card: %s%s\n", Pink("%.2f", totalValue/float64(countAll)), Pink(getCurrency()))
	}

	total, _ := totalcoll.FindTotal()
	fmt.Printf("History: \n")
	showPriceHistory(total.Value, "* ", true)
}

func showReservedListStats(owned []OwnedCard) {
	var countReserved int
	for _, c := range owned {
		if c.Reserved {
			countReserved++
		}
	}
	fmt.Printf("Reserved List: %s\n", Yellow("%d", countReserved))
}

func showRarityStats(owned []OwnedCard) {
	ri := rarityBreakdown(owned)
	fmt.Printf("\n%s\n", Green("Rarity"))
	fmt.Printf("Mythics: %s\n", Pink("%.0f", ri.Mythics))
	fmt.Printf("Rares: %s\n", Pink("%.0f", ri.Rares))
	fmt.Printf("Uncommons: %s\n", Yellow("%.0f", ri.Uncommons))
	fmt.Printf("Commons: %s\n", Purple("%.0f", ri.Commons))
}

func showTypeStats(owned []OwnedCard) {
	counts := map[string]int64{}
	for _, c := range owned {
		counts[c.TypeLine] += c.Count + c.CountFoil + c.CountEtched
	}

	type entry struct {
		TypeLine string
		Count    int64
	}
	entries := make([]entry, 0, len(counts))
	for t, c := range counts {
		entries = append(entries, entry{t, c})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Count > entries[j].Count })
	if len(entries) > 10 {
		entries = entries[:10]
	}

	fmt.Printf("\n%s\n", Green("Types (Top 10)"))
	for _, e := range entries {
		fmt.Printf("%s: %s\n", e.TypeLine, Purple("%d", e.Count))
	}
}

func showCardsAddedPerMonth(owned []OwnedCard) {
	fmt.Printf("\n%s\n", Green("Cards added over time"))

	type monthKey struct{ Year, Month int }
	counts := map[monthKey]int{}
	for _, c := range owned {
		t := stringToTime(c.Created)
		var year, month int
		fmt.Sscanf(t, "%d-%d-", &year, &month)
		counts[monthKey{year, month}]++
	}

	keys := make([]monthKey, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Year != keys[j].Year {
			return keys[i].Year < keys[j].Year
		}
		return keys[i].Month < keys[j].Month
	})

	for _, k := range keys {
		fmt.Printf("%d-%02d: %s\n", k.Year, k.Month, Purple("%d", counts[k]))
	}
}

func showManaCurveStats(owned []OwnedCard) {
	counts := map[float64]int{}
	for _, c := range owned {
		counts[c.Cmc]++
	}

	cmcs := make([]float64, 0, len(counts))
	for cmc := range counts {
		cmcs = append(cmcs, cmc)
	}
	sort.Float64s(cmcs)

	fmt.Printf("\n%s\n", Green("Mana Curve"))
	for _, cmc := range cmcs {
		fmt.Printf("%.0f: %s\n", cmc, Purple("%d", counts[cmc]))
	}
}

func showArtistStats(owned []OwnedCard) {
	counts := map[string]int{}
	for _, c := range owned {
		counts[c.Artist]++
	}

	type entry struct {
		Artist string
		Count  int
	}
	entries := make([]entry, 0, len(counts))
	for a, c := range counts {
		entries = append(entries, entry{a, c})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Count > entries[j].Count })
	if len(entries) > 10 {
		entries = entries[:10]
	}

	fmt.Printf("\n%s\n", Green("Artists (Top 10)"))
	for _, e := range entries {
		fmt.Printf("%s: %s\n", e.Artist, Purple("%d", e.Count))
	}
}

func showColorStats(owned []OwnedCard) {
	counts := map[string]int64{}
	for _, c := range owned {
		if len(c.ColorIdentity) != 1 {
			continue
		}
		counts[c.ColorIdentity[0]] += c.Count + c.CountFoil + c.CountEtched
	}

	type entry struct {
		Color string
		Count int64
	}
	entries := make([]entry, 0, len(counts))
	for color, c := range counts {
		entries = append(entries, entry{color, c})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Count > entries[j].Count })

	fmt.Printf("\n%s\n", Green("Colors"))
	for _, e := range entries {
		fmt.Printf("%s: %s\n", convertManaSymbols([]any{e.Color}), Purple("%d", e.Count))
	}
}
