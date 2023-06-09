package cmd

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/flofriday/websearch/query"
	"github.com/flofriday/websearch/store"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
)

func mainHandler(queryEngine *query.QueryEngine) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		query := c.Query("q", "")

		// If there is no question just display the home page
		if query == "" {
			return c.SendFile("web/view/home.html")
		}

		// Get the results
		queryResult, err := queryEngine.Find(query, 20)
		if err != nil {
			return c.Status(500).SendString(fmt.Sprintf("Could not load results: '%v'", err))
		}

		// Render the Results
		return c.Render("results", queryResult)
	}
}

func Serve(addr string, sqliteFile string) {

	// Setup the dependencies
	db, err := sql.Open("sqlite3", sqliteFile+"?_journal=WAL")
	if err != nil {
		log.Fatal("Unable to connect to the db!")
	}

	sqlDocumentStore, err := store.NewSQLDocumentStore(db)
	if err != nil {
		log.Fatalf("Unable to connect to the document store '%v'\n", err)
	}
	sqlIndexStore, err := store.NewSQLIndexStore(db)
	if err != nil {
		log.Fatalf("Unable to connect to the index store '%v'\n", err)
	}

	queryEngine := &query.QueryEngine{
		DocumentStore: sqlDocumentStore,
		IndexStore:    sqlIndexStore,
	}

	// Setup the routes
	templateEngine := html.New("./web/view", ".html")
	app := fiber.New(fiber.Config{
		AppName: "websearch",
		Views:   templateEngine,
	})

	app.Get("/", mainHandler(queryEngine))
	app.Static("/static", "./web/static")
	app.Listen(addr)
}
