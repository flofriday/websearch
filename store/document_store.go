package store

import "github.com/flofriday/websearch/model"

type DocumentStore interface {
	Put(*model.DocumentView) error
	Get(index int64) (*model.DocumentView, error)
	GetAll(index []int64) ([]*model.DocumentView, error)
}
