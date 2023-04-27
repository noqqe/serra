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
	// Connect to the DB & load the collection
	client := storageConnect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storageDisconnect(client)

	// Loop over different cards
	for _, card := range cards {
		// Extract collector number and set name from input & remove leading zeros
		collectorNumber := strings.TrimLeft(strings.Split(card, "/")[1], "0")
		setName := strings.Split(card, "/")[0]

		// Fetch card from scryfall
		c, err := findCardByCollectorNumber(coll, setName, collectorNumber)
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
			continue
		}

		if foil && c.SerraCountFoil < 1 {
			LogMessage(fmt.Sprintf("Error: No Foil \"%s\" in the Collection.", c.Name), "red")
			continue
		}

		if !foil && c.SerraCount < 1 {
			LogMessage(fmt.Sprintf("Error: No normal \"%s\" in the Collection.", c.Name), "red")
			continue
		}

		if foil && c.SerraCountFoil == 1 && c.SerraCount == 0 || !foil && c.SerraCount == 1 && c.SerraCountFoil == 0 {
			coll.storageRemove(bson.M{"_id": c.ID})
			LogMessage(fmt.Sprintf("\"%s\" (%.2f%s) removed from the Collection.", c.Name, c.getValue(foil), getCurrency()), "green")
		} else {
			modifyCardCount(coll, c, -count, foil)
		}
	}

	return nil
}
