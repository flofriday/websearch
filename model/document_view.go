package model

import "net/url"

// FIXME: probably should just be in the server
type Document struct {
	Index       int64
	Title       string
	Description string
	Url         *url.URL
	Icon        *url.URL
}
