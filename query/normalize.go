package query

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// This function normalizes a single term. It should be applied to the terms
// inserted into the index but also to the words inside the query
func Normalize(term string) string {
	form := norm.NFKC
	term = form.String(term)
	term = strings.ToLower(term)
	return term
}
