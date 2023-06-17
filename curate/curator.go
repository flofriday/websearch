package curate

import (
	"log"
	"net/url"
	"strings"
	"sync"

	"github.com/flofriday/websearch/model"
	"github.com/flofriday/websearch/queue"
)

type Curator struct {
	discoverQueue queue.Queue[*url.URL]
	requestQueue  queue.Queue[*model.Request]
	responseQueue queue.Queue[*model.Response]
	documentQueue queue.Queue[*model.Response]

	// FIXME: If that ever becomes a bottle-neck, a tries datastucture would fit
	// quite nice for this usecase.
	seenURLs    map[string]bool
	indexedURLs map[string]bool
	idCounter   int64
	limit       int64
	lock        sync.RWMutex
}

// FIXME: The constructor here makes sense but since it need so many arguments
// maybe a single option argument would be nicer
func NewCurator(
	discoverQueue queue.Queue[*url.URL],
	requestQueue queue.Queue[*model.Request],
	responseQueue queue.Queue[*model.Response],
	documentQueue queue.Queue[*model.Response],
	limit int64,
) *Curator {
	return &Curator{
		discoverQueue: discoverQueue,
		requestQueue:  requestQueue,
		responseQueue: responseQueue,
		documentQueue: documentQueue,
		seenURLs:      map[string]bool{},
		indexedURLs:   map[string]bool{},
		idCounter:     0,
		limit:         limit,
	}
}

// Removes part of the url which should not make a difference for our usecase
func normalize(link *url.URL) *url.URL {
	link.Fragment = ""
	link.Path = strings.TrimRight(link.Path, "/")
	return link
}

// Filters the url, wether it should be blocked or not
func isUseful(link *url.URL) bool {
	// FIXME: Implement filtering, like images, etc
	return true
}

func (c *Curator) addSeenURL(link *url.URL) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.seenURLs[link.String()] = true
}

func (c *Curator) hasSeenURL(link *url.URL) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	_, ok := c.seenURLs[link.String()]
	return ok
}

func (c *Curator) Run() {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		c.curateDiscover()
		wg.Done()
	}()
	go func() {
		c.curateResponse()
		wg.Done()
	}()
	wg.Wait()

	log.Println("Currator done")
}

// Curate the discovered URLs and decide which should be passed on to the
// request queue and which ones should be filtered out.
func (c *Curator) curateDiscover() {
	for {
		uri, err := c.discoverQueue.Get()
		if err != nil {
			log.Println("Curator is exiting, discoverqueue broken")
			break
		}
		uri = normalize(uri)

		if !isUseful(uri) {
			continue
		}

		if c.hasSeenURL(uri) {
			// Already seen
			continue
		}
		c.addSeenURL(uri)

		// FIXME: Add additional url filters here

		target := &model.Request{
			Index: c.idCounter,
			Url:   uri,
		}
		c.idCounter++

		// FIXME: This is the wrong place to limit the size, because the
		// submitted documents here can still fail later down the pipeline.
		// We should probably do this by monitoring the documentstore.
		if c.idCounter > c.limit {
			break
		}

		c.requestQueue.Put(target)
	}

	// Close the output queue because we have submitted enough documents
	log.Println("Close request queue")
	c.requestQueue.Close()

	// Keep draining the discover queue
	for {
		_, err := c.discoverQueue.Get()
		if err != nil {
			break
		}
	}
}

// Curate the response queue and decide which of those should be indexed.
// Here we need to filter the URL again because of redirects and we can also maybe
// filter the body somewhat.
func (c *Curator) curateResponse() {
	for {
		response, err := c.responseQueue.Get()
		if err != nil {
			break
		}

		uri := normalize(response.Url)

		if !isUseful(uri) {
			continue
		}

		if _, ok := c.indexedURLs[uri.String()]; ok {
			// Already indexed
			continue
		}

		c.addSeenURL(uri)
		c.indexedURLs[uri.String()] = true
		for _, redirectUri := range response.Redirected {
			c.addSeenURL(redirectUri)
			c.indexedURLs[redirectUri.String()] = true
		}

		// FIXME: Add additional url filters here

		c.documentQueue.Put(response)
	}

	// Close the output queue because we have submitted enough documents
	log.Println("Close document queue")
	c.documentQueue.Close()
}
