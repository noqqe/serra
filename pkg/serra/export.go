package serra

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"

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

		// filter out cards that do not reach the minimum amount (--min-count)
		// this is done after query result because find query constructed does not support
		// aggregating fields (of count and countFoil).
		temp := cardList[:0]
		for _, card := range cardList {
			if (card.SerraCount + card.SerraCountFoil) >= count {
				temp = append(temp, card)
			}
		}
		cardList = temp

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

func exportTCGPowertools(cards []Card) {

	// TCGPowertools.com Example
	// idProduct,quantity,name,set,condition,language,isFoil,isPlayset,isSigned,isFirstEd,price,comment
	// 260009,1,Totally Lost,Gatecrash,GD,English,true,true,,,1000,
	// 260009,1,Totally Lost,Gatecrash,NM,English,true,true,,,1000,

	fmt.Println("quantity,cardmarketId,name,set,condition,language,isFoil,isPlayset,price,comment")
	for _, card := range cards {

		// nonfoil
		if card.SerraCount > 0 {
			fmt.Printf("%d,%.0f,%s,%s,EX,German,false,false,%.2f,\n", card.SerraCount, card.CardmarketID, card.Name, card.SetName, card.getValue())
		}

		// foil
		if card.SerraCountFoil > 0 {
			fmt.Printf("%d,%.0f,%s,%s,EX,German,true,false,%.2f,\n", card.SerraCountFoil, card.CardmarketID, card.Name, card.SetName, card.getFoilValue())
		}
	}
}

func exportMoxfield(cards []Card) {

	// Structure
	// https://www.moxfield.com/help/importing-collection
	records := [][]string{{
		"Count", "Name", "Edition", "Condition", "Language", "Finish", "Collector Number", "Alter", "Proxy", "Purchase Price"}}

	w := csv.NewWriter(os.Stdout)

	for _, card := range cards {

		// nonfoil
		if card.SerraCount > 0 {
			records = append(records,
				[]string{fmt.Sprintf("%d", card.SerraCount), card.Name, card.Set, "NM", "English", "Non-foil", card.CollectorNumber, "FALSE", "FALSE", fmt.Sprintf("%.2f", card.getValue())})
		}

		// foil
		if card.SerraCountFoil > 0 {
			records = append(records,
				[]string{fmt.Sprintf("%d", card.SerraCountFoil), card.Name, card.Set, "NM", "English", "Foil", card.CollectorNumber, "FALSE", "FALSE", fmt.Sprintf("%.2f", card.getFoilValue())})
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

func exportJSON(cards []Card) {
	ehj, _ := json.MarshalIndent(cards, "", "  ")
	fmt.Println(string(ehj))
}
