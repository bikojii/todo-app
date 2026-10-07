package repository

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	todo "github.com/bikojii/todo-app"
	"github.com/jmoiron/sqlx"
)

func mockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		db.Close()
	})
	return sqlx.NewDb(db, "sqlmock"), mock
}

func TestDeleteItemOwnership(t *testing.T) {
	for _, tc := range []struct {
		name     string
		affected int64
		want     error
	}{{"owner", 1, nil}, {"foreign or absent", 0, todo.ErrNotFound}} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := mockDB(t)
			mock.ExpectExec("DELETE FROM todo_items").WithArgs(2, 7).WillReturnResult(sqlmock.NewResult(0, tc.affected))
			err := NewTodoItemPostgres(db).Delete(2, 7)
			if !errors.Is(err, tc.want) {
				t.Fatalf("delete: %v", err)
			}
		})
	}
}

func TestUpdatesAndMissingRows(t *testing.T) {
	db, mock := mockDB(t)
	items, lists := NewTodoItemPostgres(db), NewTodoListPostgres(db)
	if err := items.Update(2, 7, todo.UpdateItemInput{}); !errors.Is(err, todo.ErrInvalidInput) {
		t.Fatal(err)
	}
	if err := lists.Update(2, 7, todo.UpdateListInput{}); !errors.Is(err, todo.ErrInvalidInput) {
		t.Fatal(err)
	}
	done := false
	mock.ExpectExec("UPDATE todo_items").WithArgs(false, 2, 7).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := items.Update(2, 7, todo.UpdateItemInput{Done: &done}); err != nil {
		t.Fatal(err)
	}
	title := "Updated"
	mock.ExpectExec("UPDATE todo_lists").WithArgs(title, 7, 2).WillReturnResult(sqlmock.NewResult(0, 0))
	if err := lists.Update(2, 7, todo.UpdateListInput{Title: &title}); !errors.Is(err, todo.ErrNotFound) {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT ti.id").WithArgs(7, 2).WillReturnError(sql.ErrNoRows)
	if _, err := items.GetById(2, 7); !errors.Is(err, todo.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestCreateItemUsesOwnershipAndDone(t *testing.T) {
	db, mock := mockDB(t)
	repo := NewTodoItemPostgres(db)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT tl.id").WithArgs(2, 7).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
	mock.ExpectQuery("INSERT INTO todo_items").WithArgs("task", "", true).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(11))
	mock.ExpectExec("INSERT INTO lists_items").WithArgs(7, 11).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	id, err := repo.Create(2, 7, todo.TodoItem{Title: "task", Done: true})
	if err != nil || id != 11 {
		t.Fatalf("id = %d, error = %v", id, err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT tl.id").WithArgs(3, 7).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()
	if _, err := repo.Create(3, 7, todo.TodoItem{Title: "forbidden"}); !errors.Is(err, todo.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestDeleteListTransaction(t *testing.T) {
	for _, fail := range []bool{false, true} {
		db, mock := mockDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT tl.id").WithArgs(2, 7).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
		if fail {
			mock.ExpectExec("DELETE FROM todo_items").WithArgs(7).WillReturnError(errors.New("database failure"))
			mock.ExpectRollback()
		} else {
			mock.ExpectExec("DELETE FROM todo_items").WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 2))
			mock.ExpectExec("DELETE FROM todo_lists").WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
		}
		err := NewTodoListPostgres(db).Delete(2, 7)
		if (err != nil) != fail {
			t.Fatalf("delete error = %v", err)
		}
	}
}
