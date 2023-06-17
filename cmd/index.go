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
	numIndexers := runtime.NumCPU() * 2
	numDownloaders := numIndexers * 5

	// Setup the dependencies
	discoverQueue := queue.NewChannelQueue[*url.URL](make(chan *url.URL, 100))
	requestQueue := queue.NewChannelQueue[*model.Request](make(chan *model.Request, docLimit))
	responseQueue := queue.NewChannelQueue[*model.Response](make(chan *model.Response, 100))
	documentQueue := queue.NewChannelQueue[*model.Response](make(chan *model.Response, numIndexers*2))

	os.Remove(sqliteFile)
	db, err := sql.Open("sqlite3", sqliteFile+"?_journal=WAL&_synchronous=OFF")
	if err != nil {
		log.Fatal("Unable to connect to the db!")
	}
	defer db.Close()

	sqlDocumentStore, err := store.NewSQLDocumentStore(db)
	if err != nil {
		log.Fatalf("Unable to connect to the document store '%v'\n", err)
	}
	sqlIndexStore, err := store.NewSQLIndexStore(db)
	if err != nil {
		log.Fatalf("Unable to connect to the index store '%v'\n", err)
	}

	curator := curate.NewCurator(discoverQueue, requestQueue, responseQueue, documentQueue, docLimit)
	downloaderPool := download.NewDownloaderPool(requestQueue, responseQueue, numDownloaders)
	indexerPool := index.NewIndexerPool(discoverQueue, documentQueue, sqlDocumentStore, sqlIndexStore, numIndexers)

	// Insert the seed into the discoverQueue
	seed := []string{"https://en.wikipedia.org/wiki/Computer", "https://en.wikipedia.org/wiki/Medicine"}
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
			s2, _ := requestQueue.Size()
			s3, _ := responseQueue.Size()
			s4, _ := documentQueue.Size()
			log.Printf("Completed: %v, DiscoverQ: %v, RequestQ: %v, ResponseQ: %v, DocumentQ: %v", cnt, s1, s2, s3, s4)
			time.Sleep(time.Millisecond * 1000)
		}
	}()
	wg.Wait()

	// Print the final statistics
	log.Println("Optimize DB")
	sqlIndexStore.Optimize()
	db.Exec("")
	log.Println("")
	cnt, _ := sqlDocumentStore.Count()
	duration := time.Since(startTime)
	log.Println(" --- Statistics --- ")
	log.Printf("Downloaded and indexed %v document in %v\n", cnt, duration)
	if cnt > 0 {
		log.Printf("Average time per document: %v\n", time.Duration(int64(duration)/cnt))
	}
}
