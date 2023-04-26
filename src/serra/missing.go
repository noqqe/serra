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
	RunE: func(cmd *cobra.Command, setname []string) error {
		client := storageConnect()
		coll := &Collection{client.Database("serra").Collection("cards")}
		defer storageDisconnect(client)

		// fetch all cards in set
		cards, err := coll.storageFind(bson.D{{"set", setname[0]}}, bson.D{{"collectornumber", 1}})
		if (err != nil) || len(cards) == 0 {
			LogMessage(fmt.Sprintf("Error: Set %s not found or no card in your collection.", setname[0]), "red")
			return err
		}

		// fetch set informations
		setcoll := &Collection{client.Database("serra").Collection("sets")}
		sets, _ := setcoll.storageFindSet(bson.D{{"code", setname[0]}}, bson.D{{"_id", 1}})
		set := sets[0]

		LogMessage(fmt.Sprintf("Missing cards in %s", sets[0].Name), "green")

		// generate set with all setnumbers
		var complete_set []string
		var i int64
		for i = 1; i <= set.CardCount; i++ {
			complete_set = append(complete_set, strconv.FormatInt(i, 10))
		}

		// iterate over all cards in collection
		var in_collection []string
		for _, c := range cards {
			in_collection = append(in_collection, c.CollectorNumber)
		}

		misses := missing(in_collection, complete_set)

		// Fetch all missing cards
		missingCards := []*Card{}
		for _, m := range misses {
			card, err := fetchCard(fmt.Sprintf("%s/%s", setname[0], m))
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
			fmt.Printf("%s%s/%s%s %s%.02f%s%s\t%s (%s)\n", Purple, card.Set, card.CollectorNumber, Reset, Green, card.getValue(false), Reset, getCurrency(), card.Name, card.SetName)
		}

		return nil
	},
}
