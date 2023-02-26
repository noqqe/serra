package serra

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

func init() {
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
	Name string `form:"name"`
	Set  string `form:"set"`
	Sort string `form:"sort"`
}

func startWeb() error {

	router := gin.Default()
	router.LoadHTMLGlob("templates/*.tmpl")
	router.Static("/assets", "./assets")

	// Landing Page
	router.GET("/", landingPage)

	router.Run(":8080")
	return nil
}

func landingPage(c *gin.Context) {

	var query Query
	if c.ShouldBind(&query) == nil {
		cards := Cards("", query.Set, query.Sort, query.Name, "", "")
		sets := Sets("release")
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title": "Serraaaa",
			"cards": cards,
			"sets":  sets,
		})
	}
}
