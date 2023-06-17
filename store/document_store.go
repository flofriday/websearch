package store

import "github.com/flofriday/websearch/model"

type DocumentStore interface {
	Put(*model.Document) error
	Get(index int64) (*model.Document, error)
	GetAll(index []int64) ([]*model.Document, error)
	Count() (int64, error)
}
