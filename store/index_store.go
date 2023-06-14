package store

type IndexStore interface {
	PutWord(index int64, word string, frequency float64) error
	PutAllWords(index int64, words map[string]float64) error
	// PutRank(index int64, rank int64) error
	Get(word string) ([]int64, []float64, error)
	Optimize() error
}
