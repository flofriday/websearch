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
	var docLimit int64 = 100
	numDownloaders := 100
	numIndexers := runtime.NumCPU()

	discoverQueue := queue.NewChannelQueue[*url.URL](make(chan *url.URL, docLimit))
	downloadQueue := queue.NewChannelQueue[*model.Target](make(chan *model.Target, numDownloaders*4))
	documentQueue := queue.NewChannelQueue[*model.Document](make(chan *model.Document, numIndexers*4))

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

	curator := curate.NewCurator(discoverQueue, downloadQueue, docLimit)
	downloaderPool := download.NewDownloaderPool(discoverQueue, downloadQueue, documentQueue, numDownloaders)
	indexerPool := index.NewIndexerPool(discoverQueue, documentQueue, sqlDocumentStore, sqlIndexStore, numIndexers)

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
			cnt, _ := sqlDocumentStore.Count()
			s1, _ := discoverQueue.Size()
			s2, _ := downloadQueue.Size()
			s3, _ := documentQueue.Size()
			log.Printf("Completed: %v, DisQ: %v, TarQ: %v, DocQ: %v", cnt, s1, s2, s3)
			time.Sleep(time.Millisecond * 1000)
		}
	}()
	wg.Wait()
}
