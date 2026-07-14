package main

import (
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
)

type Store struct {
	db *sql.DB
}

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
