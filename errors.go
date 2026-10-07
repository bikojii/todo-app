package todo

import "errors"

var (
	ErrNotFound     = errors.New("resource not found")
	ErrConflict     = errors.New("username already exists")
	ErrInvalidInput = errors.New("invalid input")
	ErrCredentials  = errors.New("invalid username or password")
	ErrToken        = errors.New("invalid or expired token")
)
