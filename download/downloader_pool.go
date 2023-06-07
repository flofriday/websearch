package download

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/flofriday/websearch/model"
	"github.com/flofriday/websearch/queue"
)

type DownloaderPool struct {
	downloadQueue queue.Queue[*model.Target]
	discoverQueue queue.Queue[*url.URL]
	documentQueue queue.Queue[*model.Document]
	workerCount   int64
}

func NewDownloaderPool(
	discoverQueue queue.Queue[*url.URL],
	downloadQueue queue.Queue[*model.Target],
	documentQueue queue.Queue[*model.Document],
	workerCount int64,
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
	for i := 0; i < int(p.workerCount); i++ {
		wg.Add(1)
		go func() {
			p.downloadLoop()
			wg.Done()
		}()
	}
	wg.Wait()
	p.documentQueue.Close()
	log.Println("DownloadPool done")
}

func (p *DownloaderPool) downloadLoop() {
	for {
		target, err := p.downloadQueue.Get()
		if err != nil {
			break
		}

		// FIXME: is a client faster?
		// FIXME: there are also redirects which we should try to fix
		resp, err := http.Get(target.Url.String())
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
		log.Printf("INFO: Downloaded: %v\n", target.Url.String())
		p.documentQueue.Put(&model.Document{
			Index:   target.Index,
			Url:     resp.Request.URL,
			Content: content,
		})
	}
}
