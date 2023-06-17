package model

import "net/url"

type Request struct {
	Index int64
	Url   *url.URL
}
