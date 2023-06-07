package index

func wordFrequency(words []string) map[string]int64 {
	frequencies := map[string]int64{}
	for _, word := range words {
		if _, ok := frequencies[word]; !ok {
			frequencies[word] = 0
		}
		frequencies[word]++
	}
	return frequencies
}

func tf_idf(words []string) map[string]float64 {
	n := len(words)
	frequencies := wordFrequency(words)
	noramlized := map[string]float64{}
	for k, v := range frequencies {
		noramlized[k] = float64(v) / float64(n)
	}
	return noramlized
}
