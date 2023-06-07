package index

import (
	"log"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/net/html"

	"github.com/flofriday/websearch/model"
	"github.com/flofriday/websearch/queue"
	"github.com/flofriday/websearch/store"
)

type IndexerPool struct {
	discoverQueue queue.Queue[*url.URL]
	documentQueue queue.Queue[*model.Document]
	documentStore store.DocumentStore
	indexStore    store.IndexStore
	workerCount   int
}

func NewIndexerPool(
	discoverQueue queue.Queue[*url.URL],
	documentQueue queue.Queue[*model.Document],
	documentStore store.DocumentStore,
	indexStore store.IndexStore,
	workerCount int,
) *IndexerPool {
	return &IndexerPool{
		discoverQueue: discoverQueue,
		documentQueue: documentQueue,
		documentStore: documentStore,
		indexStore:    indexStore,
		workerCount:   workerCount,
	}
}

func (p *IndexerPool) Run() {
	var wg sync.WaitGroup
	for i := 0; i < p.workerCount; i++ {
		wg.Add(1)
		go func() {
			p.indexLoop()
			wg.Done()
		}()
	}
	wg.Wait()
	log.Println("IndexerPool Done")
	p.discoverQueue.Close()
}

func (p *IndexerPool) indexLoop() {
	for {
		document, err := p.documentQueue.Get()
		if err != nil {
			break
		}

		documentView, words, links, err := parseHTML(document.Content)
		if err != nil {
			log.Printf("WARNING: could not parse the following document %v because %v", document.Url.String(), err.Error())
			continue
		}

		documentView.Index = document.Index
		documentView.Url = document.Url
		documentView.Icon = nil
		err = p.documentStore.Put(documentView)
		if err != nil {
			log.Printf("WARNING: Unable to store doc %v because '%v'", documentView, err.Error())
			continue
		}

		for _, link := range links {
			tmpUrl, err := url.Parse(link)
			if err != nil {
				continue
			}

			absoluteUrl := document.Url.ResolveReference(tmpUrl)
			if !absoluteUrl.IsAbs() {
				continue
			}
			p.discoverQueue.Put(absoluteUrl)
		}

		abc := tf_idf(words)
		p.indexStore.PutAllWords(document.Index, abc)
	}
}

func parseHTML(text string) (*model.DocumentView, []string, []string, error) {
	doc, err := html.Parse(strings.NewReader(text))
	if err != nil {
		return nil, []string{}, nil, err
	}

	docView := &model.DocumentView{}
	links := []string{}
	words := []string{}

	var f func(*html.Node)
	f = func(n *html.Node) {
		// Add title
		if n.Type == html.ElementNode && n.Data == "title" {
			if docView.Title == "" {
				docView.Title = getInnerText(n)
			}
		}

		if n.Type == html.TextNode {
			words = append(words, strings.Split(n.Data, " ")...)
		}

		// Add discovered links
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					links = append(links, a.Val)
					break
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if n.Type == html.ElementNode && (n.Data == "style" || n.Data == "script") {
				continue
			}
			f(c)
		}
	}
	f(doc)

	return docView, words, links, nil
}

func getInnerText(n *html.Node) string {
	text := ""
	if n.Type == html.TextNode {
		text += n.Data
	} else {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			text += getInnerText(c)
		}
	}
	return text
}
