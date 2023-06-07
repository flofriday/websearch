package model

import "net/url"

type Target struct {
	Index int64
	Url   *url.URL
}
