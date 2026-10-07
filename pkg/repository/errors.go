package repository

import (
	"database/sql"
	"errors"

	todo "github.com/bikojii/todo-app"
	"github.com/lib/pq"
)

func mapError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return todo.ErrNotFound
	}
	return err
}

func checkAffected(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return todo.ErrNotFound
	}
	return nil
}

func userError(err error) error {
	var pgErr *pq.Error
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return todo.ErrConflict
	}
	return mapError(err)
}
