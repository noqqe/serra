package serra

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	removeCmd.Flags().Int64VarP(&count, "count", "c", 1, "Amount of cards to remove")
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Aliases:       []string{"a"},
	Use:           "remove",
	Short:         "Remove a card from your collection",
	Long:          "Removes a card from your collection. Amount can be modified using flags",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, cards []string) error {

		client := storage_connect()
		coll := &Collection{client.Database("serra").Collection("cards")}
		defer storage_disconnect(client)

		// Loop over different cards
		for _, card := range cards {
			// Fetch card from scryfall
			c, err := find_card_by_setcollectornumber(coll, strings.Split(card, "/")[0], strings.Split(card, "/")[1])
			if err != nil {
				LogMessage(fmt.Sprintf("%v", err), "red")
				continue
			}

			if c.SerraCount > 1 {
				modify_count_of_card(coll, c, -1)
			} else {
				coll.storage_remove(bson.M{"_id": c.ID})
				LogMessage(fmt.Sprintf("\"%s\" (%.2f %s) removed from the Collection.", c.Name, c.getValue(), getCurrency()), "green")
			}

		}
		return nil
	},
}
