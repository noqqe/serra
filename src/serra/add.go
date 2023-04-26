package serra

import (
	"fmt"
	"os"
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
			card = append(card, fmt.Sprintf("%s/%s", set, strings.TrimSpace(line)))
		}

		addCards(card, unique, count)
	}

}

func addCards(cards []string, unique bool, count int64) error {
	client := storageConnect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storageDisconnect(client)

	// Loop over different cards
	for _, card := range cards {
		// Check if card is already in collection
		co, _ := coll.storageFind(bson.D{{"set", strings.Split(card, "/")[0]}, {"collectornumber", strings.Split(card, "/")[1]}}, bson.D{})

		if len(co) >= 1 {
			c := co[0]

			if unique {
				LogMessage(fmt.Sprintf("Not adding \"%s\" (%s, %.2f%s) to Collection because it already exists.", c.Name, c.Rarity, c.getValue(foil), getCurrency()), "red")
				continue
			}

			modifyCardCount(coll, &c, count, foil)

			var total int64 = 0
			if foil {
				total = c.SerraCountFoil + count
			} else {
				total = c.SerraCount + count
			}
			// Give feedback of successfully added card
			LogMessage(fmt.Sprintf("%dx \"%s\" (%s, %.2f%s) added to Collection.", total, c.Name, c.Rarity, c.getValue(foil), getCurrency()), "green")

			// If card is not already in collection, fetching from scyfall
		} else {
			// Fetch card from scryfall
			c, err := fetchCard(card)
			if err != nil {
				LogMessage(fmt.Sprintf("%v", err), "red")
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
				LogMessage(fmt.Sprintf("%v", err), "red")
				continue
			}

			// Give feedback of successfully added card
			LogMessage(fmt.Sprintf("%dx \"%s\" (%s, %.2f%s) added to Collection.", total, c.Name, c.Rarity, c.getValue(foil), getCurrency()), "green")
		}
	}
	storageDisconnect(client)
	return nil
}
