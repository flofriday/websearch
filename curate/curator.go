package curate

import (
	"log"
	"net/url"

	"github.com/flofriday/websearch/model"
	"github.com/flofriday/websearch/queue"
)

type Curator struct {
	input     queue.Queue[*url.URL]
	output    queue.Queue[*model.Target]
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

func (c *Curator) Run() {
	for {
		url, err := c.input.Get()
		if err != nil {
			log.Println("Curator is exiting, discoverqueue broken")
			break
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
