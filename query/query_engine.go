package query

import (
	"sort"
	"strings"

	"github.com/flofriday/websearch/fp"
	"github.com/flofriday/websearch/model"
	"github.com/flofriday/websearch/store"
)

type QueryEngine struct {
	IndexStore    store.IndexStore
	DocumentStore store.DocumentStore
}

type QueryResult struct {
	Documents []*model.DocumentView
	TotalDocs int64
}

//queryEngine := query.NewQueryEngine(sqlIndexStore, sqlDocumentStore)
//documents := queryEngine.find(query)

type rankedIndex struct {
	index int64
	rank  float64
}

func (e *QueryEngine) Find(text string, number int) (*QueryResult, error) {
	words := strings.Split(text, " ")
	words = fp.Map(words, Normalize)

	indexRanks := map[int64]float64{}
	for _, word := range words {
		indecies, frequencies, err := e.IndexStore.Get(word)
		if err != nil {
			return nil, err
		}

		for i, index := range indecies {
			if _, ok := indexRanks[index]; !ok {
				indexRanks[index] = 0.0
			}
			indexRanks[index] += frequencies[i]
		}
	}

	// FIXME: Well, the internet does have more than 2,147,483,647 pages
	totalDocs := int64(len(indexRanks))

	rankedDocs := []rankedIndex{}
	for k, v := range indexRanks {
		rankedDocs = append(rankedDocs, rankedIndex{index: k, rank: v})
	}
	sort.Slice(rankedDocs, func(i, j int) bool {
		return rankedDocs[i].rank > rankedDocs[j].rank
	})

	if len(rankedDocs) > number {
		rankedDocs = rankedDocs[:number]
	}

	docs := []*model.DocumentView{}
	for _, rankedIndex := range rankedDocs {
		doc, err := e.DocumentStore.Get(rankedIndex.index)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}

	return &QueryResult{
		Documents: docs,
		TotalDocs: totalDocs,
	}, nil
}
