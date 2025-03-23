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
	Use:     "missing <set>",
	Short:   "Display missing cards from a set",
	Long: `In case you are a set collector, you can generate a list of
cards you dont own (yet) :)`,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, setName []string) error {
		client := storageConnect()
		coll := &Collection{client.Database("serra").Collection("cards")}
		l := Logger()
		defer storageDisconnect(client)

		// fetch all cards in set
		cards, err := coll.storageFind(bson.D{{"set", setName[0]}}, bson.D{{"collectornumber", 1}}, 0, 0)
		if (err != nil) || len(cards) == 0 {
			l.Errorf("Set %s not found or no card in your collection.", setName[0])
			return err
		}

		// fetch set informations
		setcoll := &Collection{client.Database("serra").Collection("sets")}
		sets, _ := setcoll.storageFindSet(bson.D{{"code", setName[0]}}, bson.D{{"_id", 1}})
		set := sets[0]

		fmt.Printf("Missing cards in %s\n", sets[0].Name)

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
			card, err := fetchCard(setName[0], m)
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
			fmt.Printf("%s%s/%s\t%s(%s, %shttps://scryfall.com/card/%s/%s%s)\t%s%.02f%s%s\t%s (%s)\n", Purple, card.Set, card.CollectorNumber, Reset, string([]rune(card.Rarity)[0]), Background, card.Set, card.CollectorNumber, Reset, Green, card.getValue(false), Reset, getCurrency(), card.Name, card.SetName)
		}

		return nil
	},
}
