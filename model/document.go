package model

import "net/url"

type Document struct {
	Index   int64
	Url     *url.URL
	Content string
}
