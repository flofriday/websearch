package cmd

import (
	"database/sql"
	"log"
	"net/url"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/flofriday/websearch/curate"
	"github.com/flofriday/websearch/download"
	"github.com/flofriday/websearch/index"
	"github.com/flofriday/websearch/model"
	"github.com/flofriday/websearch/queue"
	"github.com/flofriday/websearch/store"
)

func CrawlAndIndex(docLimit int64, sqliteFile string) {
	numDownloaders := 100
	numIndexers := runtime.NumCPU()

	// Setup the dependencies
	discoverQueue := queue.NewChannelQueue[*url.URL](make(chan *url.URL, 100))
	downloadQueue := queue.NewChannelQueue[*model.Target](make(chan *model.Target, docLimit))
	documentQueue := queue.NewChannelQueue[*model.Document](make(chan *model.Document, numIndexers*4))

	os.Remove(sqliteFile)
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

	curator := curate.NewCurator(discoverQueue, downloadQueue, docLimit)
	downloaderPool := download.NewDownloaderPool(discoverQueue, downloadQueue, documentQueue, numDownloaders)
	indexerPool := index.NewIndexerPool(discoverQueue, documentQueue, sqlDocumentStore, sqlIndexStore, numIndexers)

	// Insert the seed into the discoverQueue
	seed := []string{"https://en.wikipedia.org/wiki/SerenityOS"}
	for _, item := range seed {
		url, _ := url.Parse(item)
		discoverQueue.Put(url)
	}

	// Start the internal curate-download-index pipeline
	startTime := time.Now()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		curator.Run()
		wg.Done()
	}()
	go func() {
		downloaderPool.Run()
		wg.Done()
	}()
	go func() {
		indexerPool.Run()
		wg.Done()
	}()

	// Log the status every second, until we hit the limit
	go func() {
		for {
			cnt, _ := sqlDocumentStore.Count()
			s1, _ := discoverQueue.Size()
			s2, _ := downloadQueue.Size()
			s3, _ := documentQueue.Size()
			log.Printf("Completed: %v, DiscoverQ: %v, DownloadQ: %v, DocumentQ: %v", cnt, s1, s2, s3)
			time.Sleep(time.Millisecond * 1000)
		}
	}()
	wg.Wait()

	// Print the final statistics
	log.Println("")
	cnt, _ := sqlDocumentStore.Count()
	duration := time.Since(startTime)
	log.Println(" --- Statistics --- ")
	log.Printf("Downloaded and indexed %v document in %v\n", cnt, duration)
	log.Printf("Average time per document: %v\n", time.Duration(int64(duration)/cnt))
}
