package store

import (
	"database/sql"
)

type SQLIndexStore struct {
	db      *sql.DB
	putStmt *sql.Stmt
	getStmt *sql.Stmt
}

func NewSQLIndexStore(db *sql.DB) (*SQLIndexStore, error) {
	store := &SQLIndexStore{
		db: db,
	}

	// Create tables if they don't exist
	err := store.createTables()
	if err != nil {
		return nil, err
	}

	store.putStmt, err = db.Prepare("INSERT INTO index_words (id, word, frequency) VALUES (?, ?, ?)")
	store.getStmt, err = db.Prepare("SELECT id, frequency FROM index_words WHERE word = ?")

	return store, nil
}

func (s *SQLIndexStore) createTables() error {
	// Create index_words table if it doesn't exist
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS index_words (
		id INTEGER,
		word TEXT,
		frequency FLOAT,
		PRIMARY KEY(id, word)
	);

	CREATE INDEX IF NOT EXISTS word_idx ON index_words (word);
	`)
	if err != nil {
		return err
	}

	return nil
}

func (s *SQLIndexStore) PutWord(index int64, word string, frequency float64) error {
	_, err := s.db.Exec("", index, word, frequency)
	if err != nil {
		return err
	}

	return nil
}

func (s *SQLIndexStore) PutAllWords(index int64, words map[string]float64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	//stmt := tx.Stmt(s.putStmt)
	stmt, err := tx.Prepare("INSERT INTO index_words (id, word, frequency) VALUES (?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for word, frequency := range words {
		_, err := stmt.Exec(index, word, frequency)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func (s *SQLIndexStore) Get(word string) ([]int64, []float64, error) {
	rows, err := s.getStmt.Query(word)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var indexes []int64
	var frequencies []float64

	for rows.Next() {
		var indexID int64
		var frequency float64

		err := rows.Scan(&indexID, &frequency)
		if err != nil {
			return nil, nil, err
		}

		indexes = append(indexes, indexID)
		frequencies = append(frequencies, frequency)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	return indexes, frequencies, nil
}
