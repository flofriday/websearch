package index

import (
	"log"
	"net/url"
	"strings"
	"sync"

	"github.com/antchfx/htmlquery"

	"github.com/flofriday/websearch/fp"
	"github.com/flofriday/websearch/model"
	"github.com/flofriday/websearch/query"
	"github.com/flofriday/websearch/queue"
	"github.com/flofriday/websearch/store"
)

const DESCRIPTION_LEN = 200

type IndexerPool struct {
	discoverQueue queue.Queue[*url.URL]
	documentQueue queue.Queue[*model.Response]
	documentStore store.DocumentStore
	indexStore    store.IndexStore
	workerCount   int
}

func NewIndexerPool(
	discoverQueue queue.Queue[*url.URL],
	documentQueue queue.Queue[*model.Response],
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
		response, err := p.documentQueue.Get()
		if err != nil {
			break
		}

		document, words, links, err := parseHTML(response.Content, response.Url)
		if err != nil {
			log.Printf("WARNING: could not parse the following document %v because %v", document.Url.String(), err.Error())
			continue
		}

		document.Index = response.Index
		err = p.documentStore.Put(document)
		if err != nil {
			log.Printf("WARNING: Unable to store doc %v because '%v'", document, err.Error())
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

func parseHTML(text string, baseURL *url.URL) (*model.Document, []string, []*url.URL, error) {

	doc, err := htmlquery.Parse(strings.NewReader(text))
	if err != nil {
		return nil, nil, nil, err
	}
	body := htmlquery.FindOne(doc, "//body")
	bodyText := htmlquery.InnerText(body)

	// Find all links this documents links to
	links := []*url.URL{}
	anchors := htmlquery.Find(body, "//a/@href")
	for _, anchor := range anchors {
		href := htmlquery.SelectAttr(anchor, "href")
		link, err := parseUrlFrom(href, baseURL)
		if err != nil {
			continue
		}
		links = append(links, link)
	}

	// Find the title
	document := &model.Document{
		Title:       "",
		Description: "",
		Url:         baseURL,
	}
	if title := htmlquery.FindOne(doc, "//title"); title != nil {
		document.Title = htmlquery.InnerText(title)
	}
	if document.Title == "" {
		document.Title = baseURL.String()
	}

	// We can try to find the first p tag and  read it's contents. I played a
	// lot around with other search engines and I think this is how most do it
	// some like google seem to have a more sophisticated algorithm, which I
	// couldn't figure out, but maybe the are using AI.
	pTags := htmlquery.Find(body, "//p")
	for _, p := range pTags {
		pText := strings.TrimSpace(htmlquery.InnerText(p))
		if pText == "" {
			continue
		}

		// FIXME: A better trim-off algorithm is needed. One that respects
		// unicode and splits on word boundaries
		if len(pText) > DESCRIPTION_LEN {
			pText = pText[:DESCRIPTION_LEN] + "..."
		}
		document.Description = pText
		break
	}

	if document.Description == "" {
		if len(bodyText) > DESCRIPTION_LEN {
			document.Description = bodyText[:DESCRIPTION_LEN] + "..."
		} else {
			document.Description = bodyText
		}
	}

	// Find the icon
	if iconLink := htmlquery.FindOne(doc, "//link[@rel='icon' or @rel='shortcut icon']/@href"); iconLink != nil {
		if icon, err := parseUrlFrom(htmlquery.SelectAttr(iconLink, "href"), baseURL); err == nil {
			document.Icon = icon
		}
	}

	// Get a list of words in that document
	// FIXME: We need to split on lots more words
	words := strings.Fields(bodyText)
	return document, words, links, nil

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
