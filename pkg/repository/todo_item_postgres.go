package repository

import (
	"fmt"
	"strings"

	"github.com/bikojii/todo-app"
	"github.com/jmoiron/sqlx"
)

type TodoItemPostgres struct {
	db *sqlx.DB
}

func NewTodoItemPostgres(db *sqlx.DB) *TodoItemPostgres {
	return &TodoItemPostgres{db: db}
}

func (r *TodoItemPostgres) Create(userId, listId int, item todo.TodoItem) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}

	defer tx.Rollback()
	var ownedId int
	err = tx.QueryRow("SELECT tl.id FROM todo_lists tl JOIN users_lists ul ON ul.list_id = tl.id WHERE ul.user_id = $1 AND tl.id = $2 FOR UPDATE OF tl", userId, listId).Scan(&ownedId)
	if err != nil {
		return 0, mapError(err)
	}
	var itemId int
	createItemQuery := fmt.Sprintf("INSERT INTO %s (title, description, done) VALUES ($1, $2, $3) RETURNING id",
		todoItemsTable)

	row := tx.QueryRow(createItemQuery, item.Title, item.Description, item.Done)
	err = row.Scan(&itemId)
	if err != nil {
		return 0, err
	}

	createListItemsQuery := fmt.Sprintf("INSERT INTO %s (list_id, item_id) VALUES ($1, $2)",
		listsItemsTable)
	_, err = tx.Exec(createListItemsQuery, listId, itemId)
	if err != nil {
		return 0, err
	}

	return itemId, tx.Commit()
}

func (r *TodoItemPostgres) GetAll(userId, listId int) ([]todo.TodoItem, error) {
	items := make([]todo.TodoItem, 0)
	query := fmt.Sprintf(`SELECT ti.id, ti.title, ti.description, ti.done
        FROM %s ti
        INNER JOIN %s li on li.item_id = ti.id
        INNER JOIN %s ul on ul.list_id = li.list_id
                					WHERE li.list_id = $1 AND ul.user_id = $2`,
		todoItemsTable, listsItemsTable, usersListsTable)
	if err := r.db.Select(&items, query, listId, userId); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *TodoItemPostgres) GetById(userId, itemId int) (todo.TodoItem, error) {
	var item todo.TodoItem
	query := fmt.Sprintf(`SELECT ti.id, ti.title, ti.description, ti.done
        FROM %s ti
        INNER JOIN %s li on li.item_id = ti.id
        INNER JOIN %s ul on ul.list_id = li.list_id
                 					WHERE ti.id = $1 AND ul.user_id = $2`,
		todoItemsTable, listsItemsTable, usersListsTable)
	if err := r.db.Get(&item, query, itemId, userId); err != nil {
		return item, mapError(err)
	}
	return item, nil
}

func (r *TodoItemPostgres) Delete(userId, itemId int) error {
	query := fmt.Sprintf(`DELETE FROM %s ti using %s li, %s ul
       WHERE ti.id = li.item_id AND li.list_id = ul.list_id AND ul.user_id = $1 AND ti.id = $2`,
		todoItemsTable, listsItemsTable, usersListsTable)
	result, err := r.db.Exec(query, userId, itemId)
	return checkAffected(result, err)
}

func (r *TodoItemPostgres) Update(userId, itemId int, input todo.UpdateItemInput) error {
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

	if input.Done != nil {
		setValues = append(setValues, fmt.Sprintf("done=$%d", argId))
		args = append(args, *input.Done)
		argId++
	}

	setQuery := strings.Join(setValues, ", ")

	query := fmt.Sprintf(`UPDATE %s ti SET %s FROM %s li, %s ul
                    WHERE ti.id = li.item_id AND li.list_id = ul.list_id AND ul.user_id = $%d AND ti.id = $%d`,
		todoItemsTable, setQuery, listsItemsTable, usersListsTable, argId, argId+1)
	args = append(args, userId, itemId)

	result, err := r.db.Exec(query, args...)
	return checkAffected(result, err)
}
