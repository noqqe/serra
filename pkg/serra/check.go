package serra

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	checkCmd.Flags().StringVarP(&set, "set", "s", "", "Filter by set code (usg/mmq/vow)")
	checkCmd.Flags().BoolVarP(&detail, "detail", "d", false, "Show details for cards (url)")
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Aliases:       []string{"c"},
	Use:           "check",
	Short:         "Check if a card is in your collection",
	Long:          "Check if a card is in your collection. Useful for list comparsions",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, cards []string) error {
		checkCards(cards, detail)
		return nil
	},
}

func checkCards(cards []string, detail bool) error {
	client := storageConnect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storageDisconnect(client)
	l := Logger()

	// Loop over different cards
	for _, card := range cards {

		if !strings.Contains(card, "/") {
			l.Errorf("Invalid card format %s. Needs to be set/collector number i.e. \"usg/13\"", card)
			continue
		}

		// Extract collector number and set name from card input & trim any leading 0 from collector number
		collectorNumber := strings.TrimLeft(strings.Split(card, "/")[1], "0")
		setName := strings.ToLower(strings.Split(card, "/")[0])

		if collectorNumber == "" {
			l.Errorf("Invalid card format %s. Needs to be set/collector number i.e. \"usg/13\"", card)
			continue
		}

		// Check if card is already in collection
		co, err := coll.storageFind(bson.D{{"set", setName}, {"collectornumber", collectorNumber}}, bson.D{}, 0, 0)
		if err != nil {
			l.Warn(err)
			continue
		}

		// If Card is in collection, print yes.
		if len(co) >= 1 {
			c := co[0]
			fmt.Printf("PRESENT %s \"%s\" (%s, %.2f%s) %s\n", card, c.Name, c.Rarity, c.getValue(foil), getCurrency(), strings.Replace(c.ScryfallURI, "?utm_source=api", "", 1))
			continue
		} else {
			if detail {
				// fetch card from scyrfall if --detail was given
				c, _ := fetchCard(setName, collectorNumber)
				fmt.Printf("MISSING %s \"%s\" (%s, %.2f%s) %s\n", card, c.Name, c.Rarity, c.getValue(foil), getCurrency(), strings.Replace(c.ScryfallURI, "?utm_source=api", "", 1))
			} else {
				// Just print, the card name was not found
				fmt.Printf("MISSING \"%s\"\n", card)
			}
		}
	}
	storageDisconnect(client)
	return nil
}
