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

	docs, err := e.DocumentStore.GetAll(
		fp.Map(rankedDocs, func(i rankedIndex) int64 { return i.index }),
	)
	if err != nil {
		return nil, err
	}

	return &QueryResult{
		Documents: docs,
		TotalDocs: totalDocs,
	}, nil
}
