package serra

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	webCmd.Flags().StringVarP(&address, "address", "a", "0.0.0.0", "Address to listen on")
	webCmd.Flags().Uint64VarP(&port, "port", "p", 8080, "Port to listen on")
	rootCmd.AddCommand(webCmd)
}

var webCmd = &cobra.Command{
	Aliases:       []string{"a"},
	Use:           "web",
	Short:         "Startup web interface",
	Long:          "Start a tiny web interface to have a web view of your collection",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		startWeb()
		return nil
	},
}

type Query struct {
	Name  string `form:"name"`
	Set   string `form:"set"`
	Sort  string `form:"sort"`
	Limit int64  `form:"limit"`
	Page  int64  `form:"page"`
}

func startWeb() error {
	router := gin.Default()
	router.LoadHTMLGlob("templates/*.tmpl")
	router.Static("/assets", "./assets")

	// Landing page
	router.GET("/", landingPage)

	router.Run(address + ":" + strconv.FormatUint(port, 10))
	return nil
}

func landingPage(c *gin.Context) {
	var query Query
	if c.ShouldBind(&query) == nil {

		// Construct per Page results "limit"
		strLimit := c.DefaultQuery("limit", "500")
		limit, _ := strconv.ParseInt(strLimit, 10, 64)
		if limit == 0 {
			limit = 500
		}

		// Fetch all sets for Dropdown
		sets := Sets("release")

		// Fetch all results based on filter criteria
		cards := Cards("", query.Set, query.Sort, query.Name, "", "", false, false, query.Page*int64(limit), limit)

		// Construct quick way for counting results
		filter := bson.D{}
		client := storageConnect()
		coll := &Collection{client.Database("serra").Collection("cards")}

		if query.Set != "" {
			filter = append(filter, bson.E{"set", query.Set})
		}

		if query.Name != "" {
			filter = append(filter, bson.E{"name", bson.D{{"$regex", ".*" + query.Name + ".*"}, {"$options", "i"}}})
		}

		counts, _ := coll.storageAggregate(mongo.Pipeline{
			bson.D{
				{"$match", filter},
			},
			bson.D{
				{"$group", bson.D{
					{"_id", nil},
					{"count", bson.D{{"$sum", 1}}},
				}}},
		})
		defer storageDisconnect(client)

		// Catch index error on no results
		var numCards int32
		if len(counts) != 0 {
			numCards = counts[0]["count"].(int32)
		}

		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title":    "Serra",
			"cards":    cards,
			"sets":     sets,
			"query":    query,
			"version":  Version,
			"prevPage": query.Page - 1,
			"page":     query.Page,
			"nextPage": query.Page + 1,
			"limit":    limit,
			"numCards": int64(numCards),
			"numPages": int64(numCards) / limit,
		})
	}
}
