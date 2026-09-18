package serra

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
)

func init() {
	addCmd.Flags().Int64VarP(&count, "count", "c", 1, "Amount of cards to add")
	addCmd.Flags().BoolVarP(&unique, "unique", "u", false, "Only add card if not existent yet")
	addCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Spin up interactive terminal")
	addCmd.Flags().StringVarP(&set, "set", "s", "", "Filter by set code (usg/mmq/vow)")
	addCmd.Flags().BoolVarP(&foil, "foil", "f", false, "Add foil variant of card")
	addCmd.Flags().BoolVarP(&etched, "etched", "", false, "Add etched foil variant of card")
	addCmd.Flags().StringVarP(&language, "language", "l", DefaultLanguage, "Language of the card (en, de, fr, ...)")
	addCmd.Flags().StringVarP(&condition, "condition", "", DefaultCondition, "Condition of the card (nm, lp, mp, hp, dmg)")
	rootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Aliases:       []string{"a"},
	Use:           "add",
	Short:         "Add a card to your collection",
	Long:          "Adds a card from scryfall to your collection. Amount can be modified using flags",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, cards []string) error {
		if interactive {
			addCardsInteractive(unique, set)
		} else {
			for _, card := range cards {
				addCard(card, unique, count)
			}
		}
		return nil
	},
}

func addCardsInteractive(unique bool, set string) {
	l := Logger()
	if len(set) == 0 {
		l.Fatal("Option --set <set> must be given in interactive mode")
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

		// default is no foil
		foil = false

		// default is count 1
		count = 1

		fields := strings.Split(line, " ")

		// Are there extra arguments?
		if len(fields) == 2 {

			// foil shortcut
			if fields[1] == "f" {
				foil = true
			}

			// amount shortcut
			if amount, err := strconv.Atoi(fields[1]); err == nil {
				if amount > 1 {
					count = int64(amount)
				}
			}
		}

		// Detect if input contains a dash, if it does it means the user wants to add a range of cards
		if strings.Contains(fields[0], "-") {
			// Split input into two parts
			parts := strings.Split(fields[0], "-")
			// Check if both parts are numbers
			if len(parts) == 2 {
				if start, err := strconv.Atoi(parts[0]); err == nil {
					if end, err := strconv.Atoi(parts[1]); err == nil {
						// Loop over range and add each card individually
						for i := start; i <= end; i++ {
							addCard(fmt.Sprintf("%s/%d", set, i), unique, count)
						}
						continue
					}
				}
			}
		}

		cardID := fmt.Sprintf("%s/%s", set, fields[0])
		addCard(cardID, unique, count)
	}

}

func addCard(cardID string, unique bool, count int64) error {
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

	// Make sure Scryfall data for this card is cached
	card, err := cardsColl.FindCardByCollectorNumber(setName, collectorNumber)
	if err != nil {
		card, err = fetchCard(setName, collectorNumber)
		if err != nil {
			l.Warn(err)
			return err
		}
		if err := cardsColl.UpsertCard(card); err != nil {
			l.Warn(err)
			return err
		}
	}

	// Check if this exact variant (finish/language/condition) is already in the collection
	id := inventoryID(card.ID, finish, language, condition)
	if _, err := invColl.FindInventoryEntry(id); err == nil && unique {
		go playSoundNegative()
		l.Warnf("%dx \"%s\" (%s, %s%s%s) not added, because it already exists", count, card.Name, card.Rarity, card.getColoredValueForFinish(finish), getCurrency(), finishSuffix(finish))
		return nil
	}

	snapshot := priceEntryForFinish(card.Prices, finish)

	entry, err := invColl.IncrementInventory(card.ID, setName, collectorNumber, finish, language, condition, count, snapshot)
	if err != nil {
		l.Warn(err)
		return err
	}

	go playSoundPositive()
	l.Infof("%dx \"%s\" (%s, %s%s%s) added, now %d in collection", count, card.Name, card.Rarity, card.getColoredValueForFinish(finish), getCurrency(), finishSuffix(finish), entry.Count)

	return nil
}
