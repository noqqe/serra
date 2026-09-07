package serra

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	exportCmd.Flags().StringVarP(&set, "set", "e", "", "Filter by set code (usg/mmq/vow)")
	exportCmd.Flags().StringVarP(&format, "format", "f", "tcgpowertools", "Choose format to export (tcgpowertools/moxfield/json)")
	exportCmd.Flags().Int64VarP(&count, "min-count", "c", 0, "Occource more than X in your collection")
	rootCmd.AddCommand(exportCmd)
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export cards from your collection",
	Long: `Export cards from your collection.
		Your data. Your choice.
		Supports multiple output formats depending on where you want to export your collection.`,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cardList := Cards(rarity, set, sortBy, name, oracle, cardType, reserved, foil, 0, 0, "", "")

		switch format {
		case "tcgpowertools":
			exportTCGPowertools(cardList)
		case "moxfield":
			exportMoxfield(cardList)
		case "json":
			exportJSON(cardList)
		}
		return nil
	},
}

// finishName returns a human readable name for a finish, matching the
// vocabulary used by common import formats.
func finishName(finish string) string {
	switch finish {
	case FinishFoil:
		return "Foil"
	case FinishEtched:
		return "Etched"
	default:
		return "Non-foil"
	}
}

func exportTCGPowertools(cards []OwnedCard) {

	// TCGPowertools.com Example
	// idProduct,quantity,name,set,condition,language,isFoil,isPlayset,isSigned,isFirstEd,price,comment
	// 260009,1,Totally Lost,Gatecrash,GD,English,true,true,,,1000,
	// 260009,1,Totally Lost,Gatecrash,NM,English,true,true,,,1000,

	fmt.Println("quantity,cardmarketId,name,set,condition,language,isFoil,isPlayset,price,comment")
	for _, card := range cards {
		for _, e := range card.Entries {
			if e.Count <= 0 {
				continue
			}
			isFoil := e.Finish == FinishFoil || e.Finish == FinishEtched
			fmt.Printf("%d,%.0f,%s,%s,%s,%s,%t,false,%.2f,\n", e.Count, card.CardmarketID, card.Name, card.SetName, strings.ToUpper(e.Condition), languageName(e.Language), isFoil, card.valueForFinish(e.Finish))
		}
	}
}

func exportMoxfield(cards []OwnedCard) {

	// Structure
	// https://www.moxfield.com/help/importing-collection
	records := [][]string{{
		"Count", "Name", "Edition", "Condition", "Language", "Finish", "Collector Number", "Alter", "Proxy", "Purchase Price"}}

	w := csv.NewWriter(os.Stdout)

	for _, card := range cards {
		for _, e := range card.Entries {
			if e.Count <= 0 {
				continue
			}
			records = append(records,
				[]string{fmt.Sprintf("%d", e.Count), card.Name, card.Set, strings.ToUpper(e.Condition), languageName(e.Language), finishName(e.Finish), card.CollectorNumber, "FALSE", "FALSE", fmt.Sprintf("%.2f", card.valueForFinish(e.Finish))})
		}
	}

	for _, record := range records {
		if err := w.Write(record); err != nil {
			log.Fatalln("error writing record to csv:", err)
		}
	}

	w.Flush()

	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
}

func exportJSON(cards []OwnedCard) {
	ehj, _ := json.MarshalIndent(cards, "", "  ")
	fmt.Println(string(ehj))
}
