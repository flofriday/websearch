package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/flofriday/websearch/cmd"
	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "websearch",
		Usage: "A search engine for the web, just for fun 🥳",
		Commands: []*cli.Command{
			{
				Name:  "index",
				Usage: "build an index",
				Flags: []cli.Flag{
					&cli.Int64Flag{
						Name:  "number",
						Value: 1000,
						Usage: "The number of documents to index",
					},
				},
				Action: func(cCtx *cli.Context) error {
					cmd.CrawlAndIndex(cCtx.Int64("number"))
					return nil
				},
			},
			{
				Name:  "server",
				Usage: "search the index from the comfort of your browser",
				Action: func(cCtx *cli.Context) error {
					cmd.Serve()
					return nil
				},
			},
			{
				Name:      "search",
				Usage:     "search the index from the command line",
				ArgsUsage: "query",
				Action: func(cCtx *cli.Context) error {
					if len(cCtx.Args().Slice()) == 0 {
						fmt.Fprintln(os.Stderr, "usage: websearch search query")
						fmt.Fprintln(os.Stderr, "Run 'websearch search --help' for more infos.")
						return nil
					}
					cmd.Search(strings.Join(cCtx.Args().Slice(), " "))
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
