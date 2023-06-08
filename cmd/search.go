package cmd

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/flofriday/websearch/query"
	"github.com/flofriday/websearch/store"
)

func Search(queryText string) {
	db, err := sql.Open("sqlite3", "index.db?_journal=WAL")
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

	queryResult, err := queryEngine.Find(queryText, 6)
	if err != nil {
		log.Fatalf("Unable to create result because: '%v'\n", err)
	}

	fmt.Printf("Found %v results for \"%v\"\n", queryResult.TotalDocs, queryText)
	fmt.Println()

	for i, doc := range queryResult.Documents {
		fmt.Printf("%d) %s\n%s\n", i+1, doc.Title, doc.Url.String())
		fmt.Println()
	}
}
