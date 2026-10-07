package handler

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bikojii/todo-app/pkg/repository"
	"github.com/bikojii/todo-app/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func TestPostgresAPI(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("todo_test_%d", time.Now().UnixNano())
	admin, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db, err := sqlx.Connect("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, file := range []string{"000001_init.up.sql", "000002_indexes.up.sql"} {
		migration, err := os.ReadFile(filepath.Join("..", "..", "schema", file))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(migration)); err != nil {
			t.Fatal(err)
		}
	}
	gin.SetMode(gin.TestMode)
	services, err := service.NewService(repository.NewRepository(db), strings.Repeat("a", 32))
	if err != nil {
		t.Fatal(err)
	}
	router := NewHandler(services).InitRoutes()
	request := func(method, path, body, token string, status int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("%s %s: %d, want %d; %s", method, path, w.Code, status, w.Body.String())
		}
		return w.Body.Bytes()
	}
	idFrom := func(body []byte) int {
		t.Helper()
		var response struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatal(err)
		}
		return response.ID
	}
	tokens := make([]string, 2)
	for i, name := range []string{"alice", "bob"} {
		body := fmt.Sprintf(`{"name":%q,"username":%q,"password":"password123"}`, name, name)
		request("POST", "/auth/sign-up", body, "", 201)
		request("POST", "/auth/sign-up", body, "", 409)
		var response struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(request("POST", "/auth/sign-in", fmt.Sprintf(`{"username":%q,"password":"password123"}`, name), "", 200), &response); err != nil {
			t.Fatal(err)
		}
		tokens[i] = response.Token
	}
	request("POST", "/auth/sign-in", `{"username":"alice","password":"wrong"}`, "", 401)
	listID := idFrom(request("POST", "/api/lists/", `{"title":"Bob's tasks"}`, tokens[1], 201))
	itemPath := fmt.Sprintf("/api/lists/%d/items/", listID)
	firstID := idFrom(request("POST", itemPath, `{"title":"First","done":true}`, tokens[1], 201))
	secondID := idFrom(request("POST", itemPath, `{"title":"Second"}`, tokens[1], 201))
	request("GET", itemPath, "", tokens[0], 404)
	request("POST", itemPath, `{"title":"Forbidden"}`, tokens[0], 404)
	for _, method := range []string{"GET", "PUT", "DELETE"} {
		request(method, fmt.Sprintf("/api/items/%d", secondID), `{"done":true}`, tokens[0], 404)
		request(method, fmt.Sprintf("/api/lists/%d", listID), `{"title":"Forbidden"}`, tokens[0], 404)
	}
	var item struct {
		Done bool `json:"done"`
	}
	if err := json.Unmarshal(request("GET", fmt.Sprintf("/api/items/%d", firstID), "", tokens[1], 200), &item); err != nil || !item.Done {
		t.Fatalf("created done=true: %+v, %v", item, err)
	}
	request("PUT", fmt.Sprintf("/api/items/%d", firstID), `{"done":false}`, tokens[1], 200)
	if err := json.Unmarshal(request("GET", fmt.Sprintf("/api/items/%d", firstID), "", tokens[1], 200), &item); err != nil || item.Done {
		t.Fatalf("updated done=false: %+v, %v", item, err)
	}
	request("DELETE", fmt.Sprintf("/api/lists/%d", listID), "", tokens[1], 200)
	var count int
	if err := db.Get(&count, "SELECT count(*) FROM todo_items"); err != nil || count != 0 {
		t.Fatalf("orphan tasks = %d, %v", count, err)
	}
	request("GET", fmt.Sprintf("/api/items/%d", firstID), "", tokens[1], 404)
	request("DELETE", fmt.Sprintf("/api/lists/%d", listID), "", tokens[1], 404)
}
