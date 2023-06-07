package store

import (
	"database/sql"
	"errors"
	"net/url"

	"github.com/flofriday/websearch/model"
)

type SQLDocumentStore struct {
	db *sql.DB
}

func NewSQLDocumentStore(db *sql.DB) (*SQLDocumentStore, error) {
	store := &SQLDocumentStore{
		db: db,
	}

	// Create tables if they don't exist
	err := store.createTables()
	if err != nil {
		return nil, err
	}

	return store, nil
}

func (s *SQLDocumentStore) Put(doc *model.DocumentView) error {
	icon := ""
	if doc.Icon != nil {
		icon = doc.Icon.String()
	}
	_, err := s.db.Exec("INSERT INTO documents (id, title, description, url, icon) VALUES (?, ?, ?, ?, ?)", doc.Index, doc.Title, doc.Description, doc.Url.String(), icon)
	if err != nil {
		return err
	}

	return nil
}

func (s *SQLDocumentStore) Get(index int64) (*model.DocumentView, error) {
	row := s.db.QueryRow("SELECT title, description, url, icon FROM documents WHERE id = ?", index)

	doc := &model.DocumentView{}
	var urlStr, iconStr string

	err := row.Scan(&doc.Title, &doc.Description, &urlStr, &iconStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Document not found
		}
		return nil, err
	}

	urlObj, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	doc.Url = urlObj

	iconObj, err := url.Parse(iconStr)
	if err != nil {
		iconObj = nil
	}
	doc.Icon = iconObj

	return doc, nil
}

func (s *SQLDocumentStore) GetAll(index []int64) ([]*model.DocumentView, error) {
	rows, err := s.db.Query("SELECT title, description, url, icon FROM documents WHERE id IN (?)", index)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var documents []*model.DocumentView

	for rows.Next() {
		doc := &model.DocumentView{}
		var urlStr, iconStr string

		err := rows.Scan(&doc.Title, &doc.Description, &urlStr, &iconStr)
		if err != nil {
			return nil, err
		}

		urlObj, err := url.Parse(urlStr)
		if err != nil {
			return nil, err
		}
		doc.Url = urlObj

		iconObj, err := url.Parse(iconStr)
		if err != nil {
			iconObj = nil
		}
		doc.Icon = iconObj

		documents = append(documents, doc)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return documents, nil
}

func (s *SQLDocumentStore) createTables() error {
	// Create documents table if it doesn't exist
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS documents (
		id INTEGER PRIMARY KEY,
		title TEXT,
		description TEXT,
		url TEXT,
		icon TEXT
	);`)
	if err != nil {
		return err
	}

	return nil
}

func (s *SQLDocumentStore) Count() (int64, error) {
	row := s.db.QueryRow("SELECT COUNT(*) FROM documents")

	var count int64
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
