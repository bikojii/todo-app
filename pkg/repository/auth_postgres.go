package repository

import (
	todo "github.com/bikojii/todo-app"
	"github.com/jmoiron/sqlx"
)

type AuthPostgres struct{ db *sqlx.DB }

func NewAuthPostgres(db *sqlx.DB) *AuthPostgres { return &AuthPostgres{db: db} }

func (r *AuthPostgres) CreateUser(user todo.User) (int, error) {
	var id int
	err := r.db.QueryRow("INSERT INTO users (name, username, password_hash) VALUES ($1, $2, $3) RETURNING id", user.Name, user.Username, user.Password).Scan(&id)
	return id, userError(err)
}

func (r *AuthPostgres) GetUser(username string) (todo.User, error) {
	var user todo.User
	err := r.db.Get(&user, "SELECT id, password_hash FROM users WHERE username = $1", username)
	return user, mapError(err)
}

func (r *AuthPostgres) UpdatePassword(userId int, hash string) error {
	result, err := r.db.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", hash, userId)
	return checkAffected(result, err)
}
