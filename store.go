package main

import (
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
)

// Store wraps database access for the application.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store backed by db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func isDuplicateEntry(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
