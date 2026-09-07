package serra

import (
	"errors"
	"fmt"
	"strings"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
)

func init() {
	removeCmd.Flags().Int64VarP(&count, "count", "c", 1, "Amount of cards to remove")
	removeCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Spin up interactive terminal")
	removeCmd.Flags().StringVarP(&set, "set", "s", "", "Filter by set code (usg/mmq/vow)")
	removeCmd.Flags().BoolVarP(&foil, "foil", "f", false, "Remove foil variant of card")
	removeCmd.Flags().BoolVarP(&etched, "etched", "", false, "Remove etched foil variant of card")
	removeCmd.Flags().StringVarP(&language, "language", "l", DefaultLanguage, "Language of the card (en, de, fr, ...)")
	removeCmd.Flags().StringVarP(&condition, "condition", "", DefaultCondition, "Condition of the card (nm, lp, mp, hp, dmg)")
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
	cardsColl := client.getCardsCollection()
	invColl := client.getInventoryCollection()
	l := Logger()
	defer storageDisconnect(client)

	setName, collectorNumber, err := parseCardID(cardID)
	if err != nil {
		return err
	}

	finish := FinishNonfoil
	if foil {
		finish = FinishFoil
	}
	if etched {
		finish = FinishEtched
	}

	card, err := cardsColl.FindCardByCollectorNumber(setName, collectorNumber)
	if err != nil {
		l.Error(err)
		return err
	}

	id := inventoryID(card.ID, finish, language, condition)
	entry, err := invColl.FindInventoryEntry(id)
	if err != nil {
		l.Errorf("No \"%s\" (%s, %s, %s) in the collection", card.Name, language, condition, finish)
		return errors.New("card variant not in collection")
	}

	if entry.Count < count {
		l.Errorf("Only %d \"%s\" (%s, %s, %s) in the collection, cannot remove %d", entry.Count, card.Name, language, condition, finish, count)
		return errors.New("not enough copies in collection")
	}

	remaining := entry.Count - count
	if remaining <= 0 {
		invColl.RemoveInventoryEntry(id)
		l.Infof("\"%s\" (%.2f%s%s) removed", card.Name, card.valueForFinish(finish), getCurrency(), finishSuffix(finish))
	} else {
		invColl.SetInventoryCount(id, remaining)
		l.Warnf("Reduced card amount of \"%s\" (%.2f%s%s) from %d to %d", card.Name, card.valueForFinish(finish), getCurrency(), finishSuffix(finish), entry.Count, remaining)
	}

	return nil
}
