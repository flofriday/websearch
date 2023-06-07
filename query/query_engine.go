package query

import (
	"sort"
	"strings"

	"github.com/flofriday/websearch/model"
	"github.com/flofriday/websearch/store"
)

type QueryEngine struct {
	IndexStore    store.IndexStore
	DocumentStore store.DocumentStore
}

//queryEngine := query.NewQueryEngine(sqlIndexStore, sqlDocumentStore)
//documents := queryEngine.find(query)

type RankedIndex struct {
	Index int64
	Rank  float64
}

func (e *QueryEngine) Find(text string, number int) ([]*model.DocumentView, error) {
	words := strings.Split(text, " ")

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

	rankedDocs := []RankedIndex{}
	for k, v := range indexRanks {
		rankedDocs = append(rankedDocs, RankedIndex{Index: k, Rank: v})
	}
	sort.Slice(rankedDocs, func(i, j int) bool {
		return rankedDocs[i].Rank > rankedDocs[j].Rank
	})

	if len(rankedDocs) > number {
		rankedDocs = rankedDocs[:number]
	}

	/*docs, err := e.DocumentStore.GetAll(
		mapSlice(rankedDocs, func(d RankedIndex) int64 { return d.Index }),
	)
	if err != nil {
		return nil, err
	}*/

	docs := []*model.DocumentView{}
	for _, rankedIndex := range rankedDocs {
		doc, err := e.DocumentStore.Get(rankedIndex.Index)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

func mapSlice[T any, M any](a []T, f func(T) M) []M {
	n := make([]M, len(a))
	for i, e := range a {
		n[i] = f(e)
	}
	return n
}
