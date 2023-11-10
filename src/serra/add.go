package serra

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	addCmd.Flags().Int64VarP(&count, "count", "c", 1, "Amount of cards to add")
	addCmd.Flags().BoolVarP(&unique, "unique", "u", false, "Only add card if not existent yet")
	addCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Spin up interactive terminal")
	addCmd.Flags().StringVarP(&set, "set", "s", "", "Filter by set code (usg/mmq/vow)")
	addCmd.Flags().BoolVarP(&foil, "foil", "f", false, "Add foil variant of card")
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
			addCards(cards, unique, count)
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

		// construct card input for addCards
		card := []string{}

		// Detect if input contains a dash, if it does it means the user wants to add a range of cards
		if strings.Contains(line, "-") {
			// Split input into two parts
			parts := strings.Split(line, "-")
			// Check if both parts are numbers
			if _, err := strconv.Atoi(parts[0]); err == nil {
				if _, err = strconv.Atoi(parts[1]); err == nil {
					// Loop over range and add each card to card slice
					start, _ := strconv.Atoi(parts[0])
					end, _ := strconv.Atoi(parts[1])
					for i := start; i <= end; i++ {
						card = append(card, fmt.Sprintf("%s/%d", set, i))
					}
				}
			}
		} else {
			card = append(card, fmt.Sprintf("%s/%s", set, strings.Split(line, " ")[0]))
		}

		// Are there extra arguments?
		if len(strings.Split(line, " ")) == 2 {

			// foil shortcut
			if strings.Split(line, " ")[1] == "f" {
				foil = true
			}

			// amount shortcut
			if amount, err := strconv.Atoi(strings.Split(line, " ")[1]); err == nil {
				if amount > 1 {
					count = int64(amount)
				}
			}
		}

		addCards(card, unique, count)
	}

}

func addCards(cards []string, unique bool, count int64) error {
	client := storageConnect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	l := Logger()
	defer storageDisconnect(client)

	// Loop over different cards
	for _, card := range cards {
		// Extract collector number and set name from card input & trim any leading 0 from collector number

		if !strings.Contains(card, "/") {
			l.Errorf("Invalid card format %s. Needs to be set/collector number i.e. \"usg/13\"", card)
			continue
		}

		setName := strings.ToLower(strings.Split(card, "/")[0])
		collectorNumber := strings.TrimLeft(strings.Split(card, "/")[1], "0")

		if collectorNumber == "" {
			l.Errorf("Invalid card format %s. Needs to be set/collector number i.e. \"usg/13\"", card)
			continue
		}

		// Check if card is already in collection
		co, err := coll.storageFind(bson.D{{"set", setName}, {"collectornumber", collectorNumber}}, bson.D{}, 0, 0)
		if err != nil {
			l.Error(err)
			continue
		}

		if len(co) >= 1 {
			c := co[0]

			if unique {
				l.Warnf("%dx \"%s\" (%s, %.2f%s) not added, because it already exists", count, c.Name, c.Rarity, c.getValue(foil), getCurrency())
				continue
			}

			modifyCardCount(coll, &c, count, foil)

		} else {
			// Fetch card from scryfall
			c, err := fetchCard(setName, collectorNumber)
			if err != nil {
				l.Warn(err)
				continue
			}

			// Write card to mongodb
			var total int64 = 0
			if foil {
				c.SerraCountFoil = count
				total = c.SerraCountFoil
			} else {
				c.SerraCount = count
				total = c.SerraCount
			}
			err = coll.storageAdd(c)
			if err != nil {
				l.Warn(err)
				continue
			}

			// Give feedback of successfully added card
			if foil {
				l.Infof("%dx \"%s\" (%s, %.2f%s, foil) added", total, c.Name, c.Rarity, c.getValue(foil), getCurrency())
			} else {
				l.Infof("%dx \"%s\" (%s, %.2f%s) added", total, c.Name, c.Rarity, c.getValue(foil), getCurrency())
			}
		}
	}
	storageDisconnect(client)
	return nil
}
