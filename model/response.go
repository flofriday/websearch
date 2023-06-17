package model

import "net/url"

type Response struct {
	Index      int64
	Url        *url.URL
	Redirected []*url.URL
	Content    string
}
