package serra

import (
	"errors"
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
			removeCardsInteractive(set)
		} else {
			for _, card := range cards {
				removeCard(card, count)
			}
		}
		return nil
	},
}

func removeCardsInteractive(set string) {
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

		card := fmt.Sprintf("%s/%s", set, strings.TrimSpace(line))
		removeCard(card, count)
	}

}

func removeCard(cardID string, count int64) error {
	// Connect to the DB & load the collection
	client := storageConnect()
	coll := client.getCardsCollection()
	l := Logger()
	defer storageDisconnect(client)

	// Loop over different cards

	setName, collectorNumber, err := parseCardID(cardID)
	if err != nil {
		return err
	}

	// Fetch card from scryfall
	c, err := coll.FindCardByCollectorNumber(setName, collectorNumber)
	if err != nil {
		l.Error(err)
		return err
	}

	if foil && c.CountFoil < 1 {
		l.Errorf("No foil \"%s\" in the collection", c.Name)
		return errors.New("no foil card in collection")
	}

	if !foil && c.Count < 1 {
		l.Errorf("No normal \"%s\" in the collection", c.Name)
		return errors.New("no normal card in collection")
	}

	if foil && c.CountFoil == 1 && c.Count == 0 || !foil && c.Count == 1 && c.CountFoil == 0 {
		coll.RemoveCards(bson.M{"_id": c.ID})
		// TODO: Show foil price
		l.Infof("\"%s\" (%.2f%s) removed", c.Name, c.getValue(), getCurrency())
	} else {
		coll.ModifyCardCount(c, -count, foil)
	}

	return nil
}
