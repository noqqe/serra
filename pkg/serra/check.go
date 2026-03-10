package serra

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
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
		for _, card := range cards {
			checkCard(card, detail)
		}
		return nil
	},
}

func checkCard(cardID string, detail bool) error {
	client := storageConnect()
	coll := client.getCardsCollection()
	defer storageDisconnect(client)

	// Loop over different cards
	setName, collectorNumber, err := parseCardID(cardID)
	if err != nil {
		return err
	}

	card, err := findCardByCollectorNumber(coll, setName, collectorNumber)
	if err == nil {
		fmt.Printf("PRESENT %s \"%s\" (%s, %.2f%s) %s\n", cardID, card.Name, card.Rarity, card.getValue(), getCurrency(), strings.Replace(card.ScryfallURI, "?utm_source=api", "", 1))
	} else {
		if detail {
			// fetch card from scyrfall if --detail was given
			card, _ := fetchCard(setName, collectorNumber)
			fmt.Printf("MISSING %s \"%s\" (%s, %.2f%s) %s\n", cardID, card.Name, card.Rarity, card.getValue(), getCurrency(), strings.Replace(card.ScryfallURI, "?utm_source=api", "", 1))
		} else {
			// Just print, the card name was not found
			fmt.Printf("MISSING \"%s\"\n", cardID)
		}
	}
	storageDisconnect(client)
	return nil
}
