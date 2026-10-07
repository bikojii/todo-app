package repository

import (
	"fmt"
	"strings"

	"github.com/bikojii/todo-app"
	"github.com/jmoiron/sqlx"
)

type TodoListPostgres struct {
	db *sqlx.DB
}

func NewTodoListPostgres(db *sqlx.DB) *TodoListPostgres {
	return &TodoListPostgres{db: db}
}

func (r *TodoListPostgres) Create(userId int, list todo.TodoList) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}

	defer tx.Rollback()
	var id int
	createListQuery := fmt.Sprintf("INSERT INTO %s (title, description) VALUES ($1, $2) RETURNING id", todoListsTable)
	row := tx.QueryRow(createListQuery, list.Title, list.Description)
	if err := row.Scan(&id); err != nil {
		return 0, err
	}

	createUsersListQuery := fmt.Sprintf("INSERT INTO %s (user_id, list_id) VALUES ($1, $2)", usersListsTable)
	_, err = tx.Exec(createUsersListQuery, userId, id)
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()

}

func (r *TodoListPostgres) GetAll(userId int) ([]todo.TodoList, error) {
	lists := make([]todo.TodoList, 0)

	query := fmt.Sprintf(`SELECT tl.id, tl.title, tl.description FROM %s tl
                                       INNER JOIN %s ul on tl.id = ul.list_id WHERE ul.user_id = $1`,
		todoListsTable, usersListsTable)
	err := r.db.Select(&lists, query, userId)

	return lists, err
}

func (r *TodoListPostgres) GetById(userId int, listId int) (todo.TodoList, error) {
	var list todo.TodoList
	query := fmt.Sprintf(`SELECT tl.id, tl.title, tl.description FROM %s tl
                                       INNER JOIN %s ul on tl.id = ul.list_id WHERE ul.user_id = $1 AND ul.list_id = $2`,
		todoListsTable, usersListsTable)
	err := r.db.Get(&list, query, userId, listId)

	return list, mapError(err)
}

func (r *TodoListPostgres) Delete(userId, listId int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int
	err = tx.QueryRow("SELECT tl.id FROM todo_lists tl JOIN users_lists ul ON ul.list_id = tl.id WHERE ul.user_id = $1 AND tl.id = $2 FOR UPDATE OF tl", userId, listId).Scan(&id)
	if err != nil {
		return mapError(err)
	}
	_, err = tx.Exec(`DELETE FROM todo_items ti WHERE EXISTS (
        SELECT 1 FROM lists_items li WHERE li.item_id = ti.id AND li.list_id = $1
        ) AND NOT EXISTS (
        SELECT 1 FROM lists_items li WHERE li.item_id = ti.id AND li.list_id <> $1
    )`, listId)
	if err != nil {
		return err
	}
	result, err := tx.Exec("DELETE FROM todo_lists WHERE id = $1", listId)
	if err := checkAffected(result, err); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *TodoListPostgres) Update(userId, listId int, input todo.UpdateListInput) error {
	if err := input.Validate(); err != nil {
		return err
	}
	setValues := make([]string, 0)
	args := make([]interface{}, 0)
	argId := 1

	if input.Title != nil {
		setValues = append(setValues, fmt.Sprintf("title=$%d", argId))
		args = append(args, *input.Title)
		argId++
	}

	if input.Description != nil {
		setValues = append(setValues, fmt.Sprintf("description=$%d", argId))
		args = append(args, *input.Description)
		argId++
	}

	setQuery := strings.Join(setValues, ", ")

	query := fmt.Sprintf("UPDATE %s tl SET %s FROM %s ul WHERE tl.id = ul.list_id AND ul.list_id = $%d AND ul.user_id = $%d",
		todoListsTable, setQuery, usersListsTable, argId, argId+1)
	args = append(args, listId, userId)

	result, err := r.db.Exec(query, args...)
	return checkAffected(result, err)
}
