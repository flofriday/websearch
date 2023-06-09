package curate

import (
	"log"
	"net/url"
	"strings"

	"github.com/flofriday/websearch/model"
	"github.com/flofriday/websearch/queue"
)

type Curator struct {
	input  queue.Queue[*url.URL]
	output queue.Queue[*model.Target]

	// FIXME: If that ever becomes a bottle-neck, a tries datastucture would fit
	// quite nice for this usecase.
	seenURLs  map[string]bool
	idCounter int64
	limit     int64
}

func NewCurator(input queue.Queue[*url.URL], output queue.Queue[*model.Target], limit int64) *Curator {
	return &Curator{
		input:     input,
		output:    output,
		seenURLs:  map[string]bool{},
		idCounter: 0,
		limit:     limit,
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

func (c *Curator) Run() {
	for {
		url, err := c.input.Get()
		if err != nil {
			log.Println("Curator is exiting, discoverqueue broken")
			break
		}
		url = normalize(url)

		if !isUseful(url) {
			continue
		}

		if _, ok := c.seenURLs[url.String()]; ok {
			// Already seen
			continue
		}
		c.seenURLs[url.String()] = true

		// FIXME: Add additional url filters here

		target := &model.Target{
			Index: c.idCounter,
			Url:   url,
		}
		c.idCounter++

		// FIXME: This is the wrong place to limit the size, because the
		// submitted documents here can still fail later down the pipeline.
		// We should probably do this by monitoring the documentstore.
		if c.idCounter > c.limit {
			break
		}

		c.output.Put(target)
	}

	// Close the output queue because we have submitted enough documents
	log.Println("Close download queue")
	c.output.Close()

	// Keep draining the discover queue
	for {
		_, err := c.input.Get()
		if err != nil {
			break
		}
	}

	log.Println("Currator done")
}
