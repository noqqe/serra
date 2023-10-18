package serra

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	exportCmd.Flags().StringVarP(&set, "set", "e", "", "Filter by set code (usg/mmq/vow)")
	exportCmd.Flags().StringVarP(&format, "format", "f", "tcgpowertools", "Choose format to export (tcgpowertools/json)")
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
		cardList := Cards(rarity, set, sortby, name, oracle, cardType, reserved, foil, 0, 0)

		switch format {
		case "tcgpowertools":
			exportTCGPowertools(cardList)
		case "json":
			exportJson(cardList)
		}
		return nil
	},
}

func exportTCGPowertools(cards []Card) {

	// TCGPowertools.com Example
	// idProduct,quantity,name,set,condition,language,isFoil,isPlayset,isSigned,isFirstEd,price,comment
	// 260009,1,Totally Lost,Gatecrash,GD,English,true,true,,,1000,
	// 260009,1,Totally Lost,Gatecrash,NM,English,true,true,,,1000,

	for _, card := range cards {
		fmt.Printf("%.0f,%d,%s,%s,EX,German,false,false,,,%.2f,\n", card.CardmarketID, card.SerraCount, card.Name, card.SetName, card.getValue(false))
	}
}

func exportJson(cards []Card) {
	ehj, _ := json.MarshalIndent(cards, "", "  ")
	fmt.Println(string(ehj))
}
