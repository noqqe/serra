package serra

import (
	"fmt"
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

		client := storage_connect()
		coll := &Collection{client.Database("serra").Collection("cards")}
		defer storage_disconnect(client)

		// fetch all cards in set
		cards, err := coll.storage_find(bson.D{{"set", setname[0]}}, bson.D{{"collectornumber", 1}})
		if (err != nil) || len(cards) == 0 {
			LogMessage(fmt.Sprintf("Error: Set %s not found or no card in your collection.", setname[0]), "red")
			return err
		}

		// fetch set informations
		setcoll := &Collection{client.Database("serra").Collection("sets")}
		sets, _ := setcoll.storage_find_set(bson.D{{"code", setname[0]}}, bson.D{{"_id", 1}})
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
		for _, m := range misses {
			ncard, err := fetch_card(fmt.Sprintf("%s/%s", setname[0], m))
			if err != nil {
				continue
			}
			fmt.Printf("%.02f %s\t%s (%s)\n", ncard.getValue(), getCurrency(), ncard.Name, ncard.SetName)
		}
		return nil
	},
}
