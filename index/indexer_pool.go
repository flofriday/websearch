package index

import (
	"log"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/net/html"

	"github.com/flofriday/websearch/fp"
	"github.com/flofriday/websearch/model"
	"github.com/flofriday/websearch/query"
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

		documentView, words, links, err := parseHTML(document.Content, document.Url)
		if err != nil {
			log.Printf("WARNING: could not parse the following document %v because %v", document.Url.String(), err.Error())
			continue
		}

		documentView.Index = document.Index
		err = p.documentStore.Put(documentView)
		if err != nil {
			log.Printf("WARNING: Unable to store doc %v because '%v'", documentView, err.Error())
			continue
		}

		for _, link := range links {
			p.discoverQueue.Put(link)
		}

		words = fp.Map(words, query.Normalize)
		frequencies := tf_idf(words)
		p.indexStore.PutAllWords(document.Index, frequencies)
	}
}

func parseHTML(text string, baseURL *url.URL) (*model.DocumentView, []string, []*url.URL, error) {
	doc, err := html.Parse(strings.NewReader(text))
	if err != nil {
		return nil, []string{}, nil, err
	}

	docView := &model.DocumentView{
		Url: baseURL,
	}
	links := []*url.URL{}
	words := []string{}
	altDesc := ""

	// FIXME: We should be using a higher level HTML parser. Often there are
	// multiple ways to extract an icon or description. And we should try them
	// all in an order that provides the best results. This iterative parsing
	// we try at the moment is not quite suited for that.
	var f func(*html.Node)
	f = func(n *html.Node) {
		// Add title
		if n.Type == html.ElementNode && n.Data == "title" {
			if docView.Title == "" {
				docView.Title = getInnerText(n)
			}
		}

		// Add the description
		if n.Type == html.ElementNode &&
			n.Data == "meta" &&
			fp.Any(n.Attr, func(a html.Attribute) bool {
				return a.Key == "name" && a.Val == "description"
			}) {
			for _, attr := range n.Attr {
				if attr.Key == "content" {
					docView.Description = attr.Val
				}
			}
		}

		// Add favicon
		if n.Type == html.ElementNode &&
			n.Data == "link" &&
			fp.Any(n.Attr, func(a html.Attribute) bool {
				return a.Key == "rel" && a.Val == "icon"
			}) {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					icon, err := parseUrlFrom(attr.Val, baseURL)
					if err != nil {
						break
					}
					docView.Icon = icon
				}
			}
		}

		// Sumup all text
		if n.Type == html.TextNode {
			words = append(words, strings.Split(n.Data, " ")...)

			// FIXME: This really doesn't work but we really need a better html
			// parser for this.
			if len(altDesc) < 350 {
				altDesc += n.Data + " "
			}
		}

		// Add discovered links
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					link, err := parseUrlFrom(a.Val, baseURL)
					if err != nil {
						continue
					}
					links = append(links, link)
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

	if docView.Description == "" {
		docView.Description = altDesc + "..."
	}
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

func parseUrlFrom(link string, baseURL *url.URL) (*url.URL, error) {
	tmpUrl, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	absoluteUrl := baseURL.ResolveReference(tmpUrl)
	if !absoluteUrl.IsAbs() {
		return nil, err
	}

	return absoluteUrl, err
}
