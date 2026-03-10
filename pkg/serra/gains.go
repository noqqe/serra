package serra

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	rootCmd.AddCommand(topsCmd)
	rootCmd.AddCommand(flopsCmd)
	topsCmd.Flags().Float64VarP(&limit, "limit", "l", 0, "Minimum card price to be shown in analysis")
	topsCmd.Flags().BoolVarP(&sinceLastUpdate, "since-last-update", "u", false, "Show gains since last update")
	topsCmd.Flags().BoolVarP(&sinceBeginning, "since-beginning", "b", true, "Show gains since beginning of records")
	flopsCmd.Flags().Float64VarP(&limit, "limit", "l", 0, "Minimum card price to be shown in analysis")
	flopsCmd.Flags().BoolVarP(&sinceLastUpdate, "since-last-update", "u", false, "Show losses since last update")
	flopsCmd.Flags().BoolVarP(&sinceBeginning, "since-beginning", "b", true, "Show losses since beginning of records")
}

var topsCmd = &cobra.Command{
	Aliases:       []string{"t"},
	Use:           "tops",
	Short:         "What cards gained most value",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		Gains(limit, -1)
		return nil
	},
}

var flopsCmd = &cobra.Command{
	Aliases:       []string{"f"},
	Use:           "flops",
	Short:         "What cards lost most value",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		Gains(limit, 1)
		return nil
	},
}

func Gains(limit float64, sort int) error {

	client := storageConnect()
	coll := client.getCardsCollection()
	setcoll := client.getSetsCollection()
	defer storageDisconnect(client)

	var old int
	if sinceBeginning {
		old = 0
	}
	if sinceLastUpdate {
		old = -2
	}

	currencyField := "$serra_prices.usd"
	if getCurrency() == EUR {
		currencyField = "$serra_prices.eur"
	}

	cardRaisePipeline := mongo.Pipeline{
		bson.D{{"$project",
			bson.D{
				{"name", true},
				{"set", true},
				{"collectornumber", true},
				{"old",
					bson.D{{"$arrayElemAt",
						bson.A{currencyField, old},
					}},
				},
				{"current",
					bson.D{{"$arrayElemAt",
						bson.A{currencyField, -1},
					}},
				},
			},
		}},
		bson.D{{"$match",
			bson.D{{"old", bson.D{{"$gt", limit}}}},
		}},
		bson.D{{"$project",
			bson.D{
				{"name", true},
				{"set", true},
				{"old", true},
				{"current", true},
				{"collectornumber", true},
				{"rate",
					bson.D{{"$subtract",
						bson.A{
							bson.D{{"$divide",
								bson.A{"$current",
									bson.D{{"$divide",
										bson.A{"$old", 100},
									}},
								},
							}},
							100,
						},
					}},
				},
			},
		}},
		bson.D{{"$sort",
			bson.D{{"rate", sort}}}},
		bson.D{{"$limit", 20}},
	}
	cardRaise, _ := coll.AggregateCards(cardRaisePipeline)

	setRaisePipeline := mongo.Pipeline{
		bson.D{{"$project",
			bson.D{
				{"name", true},
				{"code", true},
				{"old",
					bson.D{{"$arrayElemAt",
						bson.A{currencyField, old},
					}},
				},
				{"current",
					bson.D{{"$arrayElemAt",
						bson.A{currencyField, -1},
					}},
				},
			},
		}},
		bson.D{{"$match",
			bson.D{{"old", bson.D{{"$gt", limit}}}},
		}},
		bson.D{{"$project",
			bson.D{
				{"name", true},
				{"code", true},
				{"old", true},
				{"current", true},
				{"rate",
					bson.D{{"$subtract",
						bson.A{
							bson.D{{"$divide",
								bson.A{"$current",
									bson.D{{"$divide",
										bson.A{"$old", 100},
									}},
								},
							}},
							100,
						},
					}},
				},
			},
		}},
		bson.D{{"$sort",
			bson.D{{"rate", sort}}}},
		bson.D{{"$limit", 10}},
	}
	setRaise, _ := setcoll.AggregateSet(setRaisePipeline)

	// TODO: bring back color coding for gains and losses
	fmt.Printf("%s\n", Purple("Cards"))
	// print each card
	for _, e := range cardRaise {
		fmt.Printf("%+.0f%% %s %s (%.2f->%s%s) \n", e["rate"], e["name"], Yellow("(%s/%s)", e["set"], fmt.Sprint(e["collectornumber"])), e["old"], Green("%.2f", e["current"]), Green(getCurrency()))
	}

	fmt.Printf("\n%s\n", Purple("Sets"))
	for _, e := range setRaise {
		fmt.Printf("%+.0f%% %s %s (%.2f->%s%s)\n", e["rate"], e["name"], Yellow("(%s)", e["code"]), e["old"], Green("%.2f", e["current"]), Green(getCurrency()))
	}
	return nil

}
