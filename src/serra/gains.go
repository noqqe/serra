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
	flopsCmd.Flags().Float64VarP(&limit, "limit", "l", 0, "Minimum card price to be shown in analysis")
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

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	setcoll := &Collection{client.Database("serra").Collection("sets")}
	defer storage_disconnect(client)

	raise_pipeline := mongo.Pipeline{
		bson.D{{"$project",
			bson.D{
				{"name", true},
				{"set", true},
				{"collectornumber", true},
				{"old",
					bson.D{{"$arrayElemAt",
						bson.A{"$serra_prices.value", 0},
					}},
				},
				{"current",
					bson.D{{"$arrayElemAt",
						bson.A{"$serra_prices.value", -1},
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
	raise, _ := coll.storage_aggregate(raise_pipeline)

	sraise_pipeline := mongo.Pipeline{
		bson.D{{"$project",
			bson.D{
				{"name", true},
				{"code", true},
				{"old",
					bson.D{{"$arrayElemAt",
						bson.A{"$serra_prices.value", 0},
					}},
				},
				{"current",
					bson.D{{"$arrayElemAt",
						bson.A{"$serra_prices.value", -1},
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
	sraise, _ := setcoll.storage_aggregate(sraise_pipeline)

	// percentage coloring
	var p_color string
	if sort == 1 {
		p_color = Red
	} else {
		p_color = Green
	}

	fmt.Printf("%sCards%s\n", Purple, Reset)
	// print each card
	for _, e := range raise {
		fmt.Printf("%s%+.0f%%%s %s %s(%s/%s)%s (%.2f->%s%.2f EUR%s) \n", p_color, e["rate"], Reset, e["name"], Yellow, e["set"], e["collectornumber"], Reset, e["old"], Green, e["current"], Reset)
	}

	fmt.Printf("\n%sSets%s\n", Purple, Reset)
	for _, e := range sraise {
		fmt.Printf("%s%+.0f%%%s %s %s(%s)%s (%.2f->%s%.2f EUR%s) \n", p_color, e["rate"], Reset, e["name"], Yellow, e["code"], Reset, e["old"], Green, e["current"], Reset)
	}
	return nil

}
