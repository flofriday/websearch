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
}

func NewCurator(input queue.Queue[*url.URL], output queue.Queue[*model.Target]) *Curator {
	return &Curator{
		input:  input,
		output: output,
	}
}

func (c *Curator) Run() {
	for {
		url, err := c.input.Get()
		if err != nil {
			log.Println("Curator is exiting")
			break
		}

		if _, ok := c.seenURLs[url.String()]; ok {
			// Already seen
			continue
		}

		// FIXME: Add additional url filters here

		target := &model.Target{
			Index: c.idCounter,
			Url:   url,
		}
		c.idCounter++

		//log.Printf("Discovered %v\n", c.idCounter)
		if c.idCounter > 10 {
			c.output.Close()
			break
		}

		c.output.Put(target)
	}
	log.Println("Currator done")
}
