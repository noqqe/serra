package serra

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
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
		if query.Limit == 0 {
			query.Limit = 500
		}
		cards := Cards("", query.Set, query.Sort, query.Name, "", "", false, false, query.Page, query.Limit)
		numCards := len(Cards("", query.Set, query.Sort, query.Name, "", "", false, false, 0, 0))
		sets := Sets("release")

		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title":    "Serra",
			"cards":    cards,
			"sets":     sets,
			"query":    query,
			"version":  Version,
			"prevPage": query.Page - 1,
			"page":     query.Page,
			"nextPage": query.Page + 1,
			"limit":    query.Limit,
			"numCards": numCards,
			"numPages": int64(numCards) / query.Limit,
		})
	}
}
