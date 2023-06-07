package model

import "net/url"

// FIXME: probably should just be in the server
type DocumentView struct {
	Index       int64
	Title       string
	Description string
	Url         *url.URL
	Icon        *url.URL
}
