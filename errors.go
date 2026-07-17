package main

import "errors"

var (
	ErrNotFound            = errors.New("resource not found")
	ErrInvalidInput        = errors.New("invalid input")
	ErrForbidden           = errors.New("forbidden")
	ErrDuplicateValue      = errors.New("duplicate value")
	ErrConflict            = errors.New("conflict")
	ErrInsufficientCredits = errors.New("insufficient credits")
	ErrInvalidUserID       = errors.New("invalid X-UserID")
)
