package serra

import (
	"fmt"
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
	l := Logger()

	if len(set) == 0 {
		l.Fatal("Option --set must be given in interactive mode")
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
	l := Logger()
	defer storageDisconnect(client)

	// Loop over different cards
	for _, card := range cards {

		if !strings.Contains(card, "/") {
			l.Errorf("Invalid card format %s. Needs to be set/collector number i.e. \"usg/13\"", card)
			continue
		}

		// Extract collector number and set name from input & remove leading zeros
		collectorNumber := strings.TrimLeft(strings.Split(card, "/")[1], "0")
		setName := strings.Split(card, "/")[0]

		if collectorNumber == "" {
			l.Errorf("Invalid card format %s. Needs to be set/collector number i.e. \"usg/13\"", card)
			continue
		}

		// Fetch card from scryfall
		c, err := findCardByCollectorNumber(coll, setName, collectorNumber)
		if err != nil {
			l.Error(err)
			continue
		}

		if foil && c.SerraCountFoil < 1 {
			l.Errorf("No foil \"%s\" in the collection", c.Name)
			continue
		}

		if !foil && c.SerraCount < 1 {
			l.Errorf("No normal \"%s\" in the collection", c.Name)
			continue
		}

		if foil && c.SerraCountFoil == 1 && c.SerraCount == 0 || !foil && c.SerraCount == 1 && c.SerraCountFoil == 0 {
			coll.storageRemove(bson.M{"_id": c.ID})
			l.Infof("\"%s\" (%.2f%s) removed", c.Name, c.getValue(foil), getCurrency())
		} else {
			modifyCardCount(coll, c, -count, foil)
		}
	}

	return nil
}
