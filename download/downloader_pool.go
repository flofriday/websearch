package download

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/flofriday/websearch/model"
	"github.com/flofriday/websearch/queue"
)

type DownloaderPool struct {
	downloadQueue queue.Queue[*model.Target]
	discoverQueue queue.Queue[*url.URL]
	documentQueue queue.Queue[*model.Document]
	workerCount   int
}

func NewDownloaderPool(
	discoverQueue queue.Queue[*url.URL],
	downloadQueue queue.Queue[*model.Target],
	documentQueue queue.Queue[*model.Document],
	workerCount int,
) *DownloaderPool {
	return &DownloaderPool{
		downloadQueue: downloadQueue,
		discoverQueue: discoverQueue,
		documentQueue: documentQueue,
		workerCount:   workerCount,
	}
}

func (p *DownloaderPool) Run() {
	var wg sync.WaitGroup
	for i := 0; i < p.workerCount; i++ {
		wg.Add(1)
		go func() {
			p.downloadLoop()
			wg.Done()
		}()
	}
	wg.Wait()
	p.documentQueue.Close()
	log.Println("DownloadPool Done")
}

func (p *DownloaderPool) newClient() *http.Client {
	// FIXME: We should somehow tell the curator that we had a redirect and not
	// issue this final URL again.
	// Also maybe we already have downloaded the final destination.
	return &http.Client{
		Jar:     nil,
		Timeout: 5 * time.Second,
	}
}

func (p *DownloaderPool) downloadLoop() {
	client := p.newClient()

	for {
		target, err := p.downloadQueue.Get()
		if err != nil {
			break
		}

		// FIXME: Security-wise we must dissallow any requests that are to our
		// local network
		resp, err := client.Get(target.Url.String())
		if err != nil {
			log.Printf("WARNING: Could not download: %v\n", target.Url.String())
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("WARNING: Could not download: %v\n", target.Url.String())
			continue
		}

		// FIXME: can this fail, if it is not valid utf-8?
		content := string(body)
		p.documentQueue.Put(&model.Document{
			Index:   target.Index,
			Url:     resp.Request.URL,
			Content: content,
		})
	}
}
