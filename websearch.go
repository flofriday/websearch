package main

import (
	"database/sql"
	"log"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/flofriday/websearch/curate"
	"github.com/flofriday/websearch/download"
	"github.com/flofriday/websearch/index"
	"github.com/flofriday/websearch/model"
	"github.com/flofriday/websearch/queue"
	"github.com/flofriday/websearch/store"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	discoverQueue := queue.NewChannelQueue[*url.URL](make(chan *url.URL, 10_000))
	downloadQueue := queue.NewChannelQueue[*model.Target](make(chan *model.Target, 100))
	documentQueue := queue.NewChannelQueue[*model.Document](make(chan *model.Document, 100))

	db, err := sql.Open("sqlite3", "index.db")
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

	curator := curate.NewCurator(discoverQueue, downloadQueue)
	downloaderPool := download.NewDownloaderPool(discoverQueue, downloadQueue, documentQueue, 10)
	indexerPool := index.NewIndexerPool(discoverQueue, documentQueue, sqlDocumentStore, sqlIndexStore, int64(runtime.NumCPU()))

	seed := []string{"https://en.wikipedia.org/wiki/SerenityOS"}
	for _, item := range seed {
		url, _ := url.Parse(item)
		discoverQueue.Put(url)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		curator.Run()
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		downloaderPool.Run()
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		indexerPool.Run()
		wg.Done()
	}()

	go func() {
		for {
			s1, _ := discoverQueue.Size()
			s2, _ := downloadQueue.Size()
			s3, _ := documentQueue.Size()
			log.Printf("DisQ: %v, TarQ: %v, DocQ: %v", s1, s2, s3)
			time.Sleep(time.Millisecond * 1000)
		}
	}()
	wg.Wait()
}
