package serra

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	rootCmd.AddCommand(missingCmd)
}

var missingCmd = &cobra.Command{
	Aliases: []string{"m"},
	Use:     "missing <set>...",
	Short:   "Display missing cards from a set",
	Long: `In case you are a set collector, you can generate a list of
cards you dont own (yet) :)`,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, setNames []string) error {
		client := storageConnect()
		coll := client.getCardsCollection()
		l := Logger()
		defer storageDisconnect(client)

		for _, setName := range setNames {
			// fetch all cards in set
			cards, err := coll.storageFind(bson.D{{"set", setName}}, bson.D{{"collectornumber", 1}}, 0, 0)
			if (err != nil) || len(cards) == 0 {
				l.Errorf("Set %s not found or no card in your collection.", setName)
				return err
			}

			// fetch set informations
			setcoll := client.getSetsCollection()
			set, err := findSetByCode(setcoll, setName)
			if err != nil {
				l.Errorf("Set %s not found. Make sure to have it in your collection.", setName)
				return err
			}

			fmt.Printf("Missing cards in %s\n", set.Name)

			// generate set with all setnumbers
			var (
				completeSet []string
				i           int64
			)
			for i = 1; i <= set.CardCount; i++ {
				completeSet = append(completeSet, strconv.FormatInt(i, 10))
			}

			// iterate over all cards in collection
			var inCollection []string
			for _, c := range cards {
				inCollection = append(inCollection, c.CollectorNumber)
			}

			misses := missing(inCollection, completeSet)

			// Fetch all missing cards
			missingCards := []*Card{}
			for _, m := range misses {
				card, err := fetchCard(setName, m)
				if err != nil {
					continue
				}

				missingCards = append(missingCards, card)
			}

			// Sort the missing cards by ID
			sort.Slice(missingCards, func(i, j int) bool {
				id1, _ := strconv.Atoi(missingCards[i].CollectorNumber)
				id2, _ := strconv.Atoi(missingCards[j].CollectorNumber)
				return id1 < id2
			})

			for _, card := range missingCards {
				fmt.Printf("%s\t(%s, %s)\t%s%s\t%s (%s)\n", Purple("%s/%s", card.Set, card.CollectorNumber), string([]rune(card.Rarity)[0]), DarkGray("https://scryfall.com/card/%s/%s", card.Set, card.CollectorNumber), Green("%.02f", card.getValue()), Green(getCurrency()), card.Name, card.SetName)
			}
		}

		return nil
	},
}
