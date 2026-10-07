package handler

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bikojii/todo-app/pkg/repository"
	"github.com/bikojii/todo-app/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
)

func TestAPIValidationAndOwnership(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	secret := strings.Repeat("a", 32)
	services, err := service.NewService(repository.NewRepository(sqlx.NewDb(db, "sqlmock")), secret)
	if err != nil {
		t.Fatal(err)
	}
	router := NewHandler(services).InitRoutes()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": 2, "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body, auth string, status int) string {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("%s %s: status %d, want %d; %s", method, path, w.Code, status, w.Body.String())
		}
		return w.Body.String()
	}
	request("GET", "/api/lists/", "", "", 401)
	request("GET", "/api/lists/", "", "Basic "+token, 401)
	request("PUT", "/api/items/7", `{}`, "Bearer "+token, 400)
	request("PUT", "/api/lists/7", `{"title":" "}`, "Bearer "+token, 400)
	request("GET", "/api/items/-1", "", "Bearer "+token, 400)
	request("POST", "/auth/sign-up", `{"name":"A","username":"a","password":"short"}`, "", 400)
	request("POST", "/api/lists/", `{`, "Bearer "+token, 400)
	request("POST", "/api/lists/", `{"title":"Task"} {}`, "Bearer "+token, 400)
	request("POST", "/api/lists/", `{"title":"Task","unexpected":true}`, "Bearer "+token, 400)
	request("POST", "/api/lists/", `{"title":"`+strings.Repeat("a", 1<<20)+`"}`, "Bearer "+token, 400)
	mock.ExpectExec("DELETE FROM todo_items").WithArgs(2, 7).WillReturnResult(sqlmock.NewResult(0, 0))
	request("DELETE", "/api/items/7", "", "Bearer "+token, 404)
	mock.ExpectQuery("SELECT tl.id").WithArgs(2).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description"}))
	if got := request("GET", "/api/lists/", "", "bearer "+token, 200); got != `{"data":[]}` {
		t.Fatalf("empty lists: %s", got)
	}
	mock.ExpectQuery("SELECT tl.id").WithArgs(2, 7).WillReturnError(errors.New("sensitive database details"))
	if got := request("GET", "/api/lists/7", "", "Bearer "+token, 500); strings.Contains(got, "sensitive") {
		t.Fatal("database error leaked")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
