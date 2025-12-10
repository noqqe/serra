package serra

import (
	"net/http"
	"strconv"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	webCmd.Flags().StringVarP(&address, "address", "a", "0.0.0.0", "Address to listen on")
	webCmd.Flags().Uint64VarP(&port, "port", "p", 8080, "Port to listen on")
	rootCmd.AddCommand(webCmd)
}

// needed for template functions, as Go templates don't support basic math operations
func add(a, b int64) int64 {
	return a + b
}

var webCmd = &cobra.Command{
	Aliases:       []string{"a"},
	Use:           "web",
	Short:         "Startup web interface",
	Long:          "Start a tiny web interface to have a web view of your collection",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := startWeb()
		if err != nil {
			return err
		}
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

type TemplateData struct {
	Title    string
	Cards    []Card
	Sets     []SetsResult
	Query    Query
	Version  string
	PrevPage int64
	Page     int64
	NextPage int64
	Limit    int64
	NumCards int64
	NumPages int64
}

// StartWeb initializes the web server with Chi
func startWeb() error {
	router := chi.NewRouter()

	// Use middlewares
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// Define static assets directory
	fs := http.FileServer(http.Dir("./assets"))
	router.Handle("/assets/*", http.StripPrefix("/assets", fs))

	// Landing page route
	router.Get("/", indexPage)

	// Start the server
	return http.ListenAndServe(address+":"+strconv.FormatUint(port, 10), router)
}

// indexPage handles the landing page, rendering the template with the appropriate data
func indexPage(w http.ResponseWriter, r *http.Request) {

	// Setting up template functions
	templates := template.New("templates")
	templates.Funcs(template.FuncMap{
		"add": add,
	})

	// Load the template files
	tmpl, err := templates.ParseGlob("templates/*.gohtml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// Parse query parameters
	var query Query
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse query parameters", http.StatusBadRequest)
		return
	}
	query.Set = r.FormValue("set")
	query.Sort = r.FormValue("sort")
	query.Name = r.FormValue("name")
	pageStr := r.FormValue("page")
	if pageStr != "" {
		query.Page, _ = strconv.ParseInt(pageStr, 10, 64)
	}

	// Construct per Page results "limit"
	strLimit := r.URL.Query().Get("limit")
	limit, _ := strconv.ParseInt(strLimit, 10, 64)
	if limit == 0 {
		limit = 500
	}

	// Fetch all sets for Dropdown
	sets := Sets("release")

	// Fetch all results based on filter criteria
	cards := Cards("", query.Set, query.Sort, query.Name, "", "", false, false, query.Page*int64(limit), limit, "", "")

	// Construct quick way for counting results
	filter := bson.D{}
	client := storageConnect()
	coll := client.getCardsCollection()

	if query.Set != "" {
		filter = append(filter, bson.E{"set", query.Set})
	}

	if query.Name != "" {
		filter = append(filter, bson.E{"name", bson.D{{"$regex", ".*" + query.Name + ".*"}, {"$options", "i"}}})
	}

	counts, err := coll.AggregateCards(mongo.Pipeline{
		bson.D{
			{"$match", filter},
		},
		bson.D{
			{"$group", bson.D{
				{"_id", nil},
				{"count", bson.D{{"$sum", 1}}},
			}}},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	defer storageDisconnect(client)

	// Catch index error on no results
	var numCards int32
	if len(counts) != 0 {
		numCards = counts[0]["count"].(int32)
	}

	tmplData := TemplateData{
		Title:    "Serra",
		Cards:    cards,
		Sets:     sets,
		Query:    query,
		Version:  Version,
		PrevPage: query.Page - 1,
		Page:     query.Page,
		NextPage: query.Page + 1,
		Limit:    limit,
		NumCards: int64(numCards),
		NumPages: int64(numCards) / limit,
	}

	err = tmpl.ExecuteTemplate(w, "index.gohtml", tmplData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
