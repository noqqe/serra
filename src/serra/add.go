package serra

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	addCmd.Flags().Int64VarP(&count, "count", "c", 1, "Amount of cards to add")
	addCmd.Flags().BoolVarP(&unique, "unique", "u", false, "Only add card if not existent yet")
	rootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Aliases:       []string{"a"},
	Use:           "add",
	Short:         "Add a card to your collection",
	Long:          "Adds a card from scryfall to your collection. Amount can be modified using flags",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, cards []string) error {

		client := storage_connect()
		coll := &Collection{client.Database("serra").Collection("cards")}
		defer storage_disconnect(client)

		// Loop over different cards
		for _, card := range cards {

			// Check if card is already in collection
			co, _ := coll.storage_find(bson.D{{"set", strings.Split(card, "/")[0]}, {"collectornumber", strings.Split(card, "/")[1]}}, bson.D{})

			if len(co) >= 1 {
				c := co[0]

				if unique {
					LogMessage(fmt.Sprintf("Not adding \"%s\" (%s, %.2f Eur) to Collection because it already exists.", c.Name, c.Rarity, c.Prices.Eur), "red")
					continue
				}

				modify_count_of_card(coll, &c, count)
				// Give feedback of successfully added card
				LogMessage(fmt.Sprintf("%dx \"%s\" (%s, %.2f Eur) added to Collection.", c.SerraCount, c.Name, c.Rarity, c.Prices.Eur), "green")

				// If card is not already in collection, fetching from scyfall
			} else {

				// Fetch card from scryfall
				c, err := fetch_card(card)
				if err != nil {
					LogMessage(fmt.Sprintf("%v", err), "red")
					continue
				}

				// Write card to mongodb
				c.SerraCount = count
				err = coll.storage_add(c)
				if err != nil {
					LogMessage(fmt.Sprintf("%v", err), "red")
					continue
				}

				// Give feedback of successfully added card
				LogMessage(fmt.Sprintf("%dx \"%s\" (%s, %.2f Eur) added to Collection.", c.SerraCount, c.Name, c.Rarity, c.Prices.Eur), "green")
			}
		}
		storage_disconnect(client)
		return nil
	},
}
