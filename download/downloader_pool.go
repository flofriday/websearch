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
	requestQueue  queue.Queue[*model.Request]
	responseQueue queue.Queue[*model.Response]
	workerCount   int
}

func NewDownloaderPool(
	requestQueue queue.Queue[*model.Request],
	responseQueue queue.Queue[*model.Response],
	workerCount int,
) *DownloaderPool {
	return &DownloaderPool{
		requestQueue:  requestQueue,
		responseQueue: responseQueue,
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
	p.responseQueue.Close()
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
		request, err := p.requestQueue.Get()
		if err != nil {
			break
		}

		redirects := []*url.URL{}
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			for _, req := range via {
				redirects = append(redirects, req.URL)
			}
			return nil
		}

		// FIXME: Security-wise we must dissallow any requests that are to our
		// local network
		resp, err := client.Get(request.Url.String())
		if err != nil {
			log.Printf("WARNING: Could not download: %v\n", request.Url.String())
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("WARNING: Could not download: %v\n", request.Url.String())
			continue
		}

		// FIXME: can this fail, if it is not valid utf-8?
		content := string(body)
		p.responseQueue.Put(&model.Response{
			Index:      request.Index,
			Url:        resp.Request.URL,
			Content:    content,
			Redirected: redirects,
		})
	}
}
