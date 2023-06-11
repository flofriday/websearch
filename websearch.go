package main

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"
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
						Name:    "number",
						Aliases: []string{"n"},
						Value:   1000,
						Usage:   "The number of documents to index",
					},
					&cli.StringFlag{
						Name:  "sqlite",
						Value: "./index.db",
						Usage: "Path of the sqlite file",
					},
					&cli.BoolFlag{
						Name:  "profile",
						Value: false,
						Usage: "Start a cpu profile",
					},
				},
				Action: func(cCtx *cli.Context) error {
					if cCtx.Bool("profile") {
						f, err := os.Create("cpu.prof")
						if err != nil {
							log.Fatal(err)
						}
						pprof.StartCPUProfile(f)
						defer pprof.StopCPUProfile()
					}

					cmd.CrawlAndIndex(cCtx.Int64("number"), cCtx.String("sqlite"))
					return nil
				},
			},
			{
				Name:  "server",
				Usage: "search the index from the comfort of your browser",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "addr",
						Value: ":8080",
						Usage: "Port and IP to listen at",
					},
					&cli.StringFlag{
						Name:  "sqlite",
						Value: "./index.db",
						Usage: "Path of the sqlite file",
					},
				},
				Action: func(cCtx *cli.Context) error {
					cmd.Serve(cCtx.String("addr"), cCtx.String("sqlite"))
					return nil
				},
			},
			{
				Name:      "search",
				Usage:     "search the index from the command line",
				ArgsUsage: "query",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "sqlite",
						Value: "./index.db",
						Usage: "Path of the sqlite file",
					},
				},
				Action: func(cCtx *cli.Context) error {
					if len(cCtx.Args().Slice()) == 0 {
						fmt.Fprintln(os.Stderr, "usage: websearch search query")
						fmt.Fprintln(os.Stderr, "Run 'websearch search --help' for more infos.")
						return nil
					}
					cmd.Search(cCtx.String("sqlite"), strings.Join(cCtx.Args().Slice(), " "))
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
