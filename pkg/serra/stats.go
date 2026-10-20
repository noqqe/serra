package serra

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
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

	owned, err := OwnedCardsWithoutHistory(bson.D{})
	if err != nil {
		Logger().Error("Error fetching stats:", err)
		return
	}

	// Show Value Stats
	showValueStats(owned, totalcoll)

	// Price Distribution
	showPriceDistributionStats(owned)

	// Rarities
	showRarityStats(owned)

	// Reserved List
	showReservedListStats(owned)

	// Colors
	showColorStats(owned)

	// Types
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

// countedMainTypes are the card types counted as main types; everything else
// left of the "—" in a type line (Legendary, Basic, Kindred, Battle, ...) is
// ignored.
var countedMainTypes = map[string]bool{
	"Instant": true, "Creature": true, "Sorcery": true, "Enchantment": true,
	"Artifact": true, "Land": true, "Planeswalker": true,
}

// splitTypeLine splits a type line like "Legendary Artifact Creature — Human
// Wizard" into its main types (Artifact, Creature) and sub types (Human,
// Wizard). Only types in countedMainTypes are reported as main types. Faces
// of multi-faced cards ("... // ...") are combined, with each type reported at
// most once per card.
func splitTypeLine(typeLine string) (mainTypes, subTypes []string) {
	seen := map[string]bool{}
	add := func(list *[]string, kind, t string) {
		if seen[kind+t] {
			return
		}
		seen[kind+t] = true
		*list = append(*list, t)
	}
	for _, face := range strings.Split(typeLine, "//") {
		main, sub, _ := strings.Cut(face, "—")
		for _, t := range strings.Fields(main) {
			if countedMainTypes[t] {
				add(&mainTypes, "main:", t)
			}
		}
		for _, t := range strings.Fields(sub) {
			add(&subTypes, "sub:", t)
		}
	}
	return mainTypes, subTypes
}

func showTypeStats(owned []OwnedCard) {
	mainCounts := map[string]int64{}
	subCounts := map[string]int64{}
	for _, c := range owned {
		n := c.Count + c.CountFoil + c.CountEtched
		mainTypes, subTypes := splitTypeLine(c.TypeLine)
		for _, t := range mainTypes {
			mainCounts[t] += n
		}
		for _, t := range subTypes {
			subCounts[t] += n
		}
	}

	printTypeCounts("Main Types", mainCounts, 0)
	printTypeCounts("Sub Types (Top 10)", subCounts, 10)
}

// printTypeCounts prints counts sorted descending, limited to the top n
// entries (n <= 0 means all).
func printTypeCounts(title string, counts map[string]int64, n int) {
	type entry struct {
		Type  string
		Count int64
	}
	entries := make([]entry, 0, len(counts))
	for t, c := range counts {
		entries = append(entries, entry{t, c})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count != entries[j].Count {
			return entries[i].Count > entries[j].Count
		}
		return entries[i].Type < entries[j].Type
	})
	if n > 0 && len(entries) > n {
		entries = entries[:n]
	}

	fmt.Printf("\n%s\n", Green(title))
	for _, e := range entries {
		fmt.Printf("%s: %s\n", e.Type, Purple("%d", e.Count))
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

// priceDistributionSteps are the lower bounds of the price buckets used by
// showPriceDistributionStats, in descending order.
var priceDistributionSteps = []float64{100, 80, 50, 40, 30, 20, 10, 5, 1, 0.5, 0.2}

// showPriceDistributionStats counts every owned copy into a price bucket,
// using the price of the finish that copy actually has. Copies without a
// price in the configured currency are counted separately.
func showPriceDistributionStats(owned []OwnedCard) {
	counts := make([]int64, len(priceDistributionSteps)+2)
	noPrice := len(priceDistributionSteps) + 1
	bucket := func(price float64) int {
		if price <= 0 {
			return noPrice
		}
		for i, step := range priceDistributionSteps {
			if price >= step {
				return i
			}
		}
		return len(priceDistributionSteps)
	}

	for _, c := range owned {
		counts[bucket(c.getValue())] += c.Count
		counts[bucket(c.getFoilValue())] += c.CountFoil
		counts[bucket(c.getEtchedValue())] += c.CountEtched
	}

	cur := getCurrency()
	fmt.Printf("\n%s\n", Green("Price Distribution"))
	for i, step := range priceDistributionSteps {
		var label string
		if i == 0 {
			label = fmt.Sprintf(">= %.2f%s", step, cur)
		} else {
			label = fmt.Sprintf("%.2f - %.2f%s", step, priceDistributionSteps[i-1], cur)
		}
		fmt.Printf("%s: %s\n", label, Purple("%d", counts[i]))
	}
	last := priceDistributionSteps[len(priceDistributionSteps)-1]
	fmt.Printf("< %.2f%s: %s\n", last, cur, Purple("%d", counts[len(priceDistributionSteps)]))
	fmt.Printf("No price: %s\n", Purple("%d", counts[noPrice]))
}
