package store

import (
	"database/sql"
	"errors"
	"net/url"

	"github.com/flofriday/websearch/fp"
	"github.com/flofriday/websearch/model"
)

type SQLDocumentStore struct {
	db        *sql.DB
	putStmt   *sql.Stmt
	getStmt   *sql.Stmt
	countStmt *sql.Stmt
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

	store.putStmt, err = db.Prepare("INSERT INTO documents (id, title, description, url, icon) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return nil, err
	}

	store.getStmt, err = db.Prepare("SELECT title, description, url, icon FROM documents WHERE id = ?")
	if err != nil {
		return nil, err
	}

	store.countStmt, err = db.Prepare("SELECT COUNT(*) FROM documents")
	if err != nil {
		return nil, err
	}

	return store, nil
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

func (s *SQLDocumentStore) Put(doc *model.Document) error {
	icon := ""
	if doc.Icon != nil {
		icon = doc.Icon.String()
	}
	// FIXME: Prepared statements are the way to go here
	_, err := s.putStmt.Exec(doc.Index, doc.Title, doc.Description, doc.Url.String(), icon)
	if err != nil {
		return err
	}

	return nil
}

func (s *SQLDocumentStore) Get(index int64) (*model.Document, error) {
	row := s.getStmt.QueryRow(index)

	doc := &model.Document{}
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

func (s *SQLDocumentStore) GetAll(index []int64) ([]*model.Document, error) {
	// FIXME: this should work, but doesn't, enabling this should however
	// improve performance.
	return fp.MapErr(index, s.Get)

	/*
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
	*/
}

func (s *SQLDocumentStore) Count() (int64, error) {
	row := s.countStmt.QueryRow()

	var count int64
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
