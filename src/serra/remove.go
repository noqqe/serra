package serra

import (
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	removeCmd.Flags().Int64VarP(&count, "count", "c", 1, "Amount of cards to remove")
	removeCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Spin up interactive terminal")
	removeCmd.Flags().StringVarP(&set, "set", "s", "", "Filter by set code (usg/mmq/vow)")
	removeCmd.Flags().BoolVarP(&foil, "foil", "f", false, "Remove foil variant of card")
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Aliases:       []string{"a"},
	Use:           "remove",
	Short:         "Remove a card from your collection",
	Long:          "Removes a card from your collection. Amount can be modified using flags",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, cards []string) error {

		if interactive {
			removeCardsInteractive(unique, set)
		} else {
			removeCards(cards, count)
		}
		return nil
	},
}

func removeCardsInteractive(unique bool, set string) {

	if len(set) == 0 {
		LogMessage("Error: --set must be given in interactive mode", "red")
		os.Exit(1)
	}

	rl, err := readline.New(fmt.Sprintf("%s> ", set))
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}

		// construct card input for addCards
		card := []string{}
		card = append(card, fmt.Sprintf("%s/%s", set, strings.TrimSpace(line)))

		removeCards(card, count)
	}

}

func removeCards(cards []string, count int64) error {

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

		if foil && c.SerraCountFoil < 1 {
			LogMessage(fmt.Sprintf("Error: No Foil \"%s\" in the Collection.", c.Name), "red")
			continue
		}

		if !foil && c.SerraCount < 1 {
			LogMessage(fmt.Sprintf("Error: No Non-Foil \"%s\" in the Collection.", c.Name), "red")
			continue
		}

		if foil && c.SerraCountFoil == 1 && c.SerraCount == 0 || !foil && c.SerraCount == 1 && c.SerraCountFoil == 0 {
			coll.storage_remove(bson.M{"_id": c.ID})
			LogMessage(fmt.Sprintf("\"%s\" (%.2f %s) removed from the Collection.", c.Name, c.getValue(foil), getCurrency()), "green")
		} else {
			modify_count_of_card(coll, c, -1, foil)
		}

	}
	return nil
}
