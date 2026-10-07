package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	todo "github.com/bikojii/todo-app"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type authMemory struct{ users map[string]todo.User }

func (r *authMemory) CreateUser(user todo.User) (int, error) {
	if _, exists := r.users[user.Username]; exists {
		return 0, todo.ErrConflict
	}
	user.Id = len(r.users) + 1
	r.users[user.Username] = user
	return user.Id, nil
}

func (r *authMemory) GetUser(username string) (todo.User, error) {
	user, ok := r.users[username]
	if !ok {
		return user, todo.ErrNotFound
	}
	return user, nil
}

func (r *authMemory) UpdatePassword(id int, hash string) error {
	for name, user := range r.users {
		if user.Id == id {
			user.Password = hash
			r.users[name] = user
			return nil
		}
	}
	return todo.ErrNotFound
}

func TestAuthentication(t *testing.T) {
	repo := &authMemory{users: make(map[string]todo.User)}
	secret := strings.Repeat("a", 32)
	auth, err := NewAuthService(repo, secret)
	if err != nil {
		t.Fatal(err)
	}
	id, err := auth.CreateUser(todo.User{Name: "Alice", Username: "alice", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.users["alice"].Password), []byte("password123")); err != nil {
		t.Fatal("password is not bcrypt")
	}
	_, err = auth.CreateUser(todo.User{Name: "Alice", Username: "alice", Password: "password123"})
	if !errors.Is(err, todo.ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}
	token, err := auth.GenerateToken("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	got, err := auth.ParseToken(token)
	if err != nil || got != id {
		t.Fatalf("token user = %d, error = %v", got, err)
	}
	for _, input := range [][2]string{{"alice", "wrong"}, {"missing", "password123"}, {"alice", ""}} {
		if _, err := auth.GenerateToken(input[0], input[1]); !errors.Is(err, todo.ErrCredentials) {
			t.Fatalf("login accepted: %v", err)
		}
	}
	for _, tc := range []struct {
		name, key string
		method    jwt.SigningMethod
		claims    jwt.MapClaims
	}{
		{"public old key", "jkfireun9843klskdmfer0eankvf", jwt.SigningMethodHS256, jwt.MapClaims{"user_id": 1, "exp": time.Now().Add(time.Hour).Unix()}},
		{"wrong algorithm", secret, jwt.SigningMethodHS512, jwt.MapClaims{"user_id": 1, "exp": time.Now().Add(time.Hour).Unix()}},
		{"missing expiration", secret, jwt.SigningMethodHS256, jwt.MapClaims{"user_id": 1}},
		{"expired", secret, jwt.SigningMethodHS256, jwt.MapClaims{"user_id": 1, "exp": time.Now().Add(-time.Hour).Unix()}},
		{"invalid user", secret, jwt.SigningMethodHS256, jwt.MapClaims{"user_id": 0, "exp": time.Now().Add(time.Hour).Unix()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token, err := jwt.NewWithClaims(tc.method, tc.claims).SignedString([]byte(tc.key))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := auth.ParseToken(token); !errors.Is(err, todo.ErrToken) {
				t.Fatalf("token accepted: %v", err)
			}
		})
	}
}

func TestLegacyPasswordUpgrade(t *testing.T) {
	repo := &authMemory{users: map[string]todo.User{"legacy": {Id: 1, Username: "legacy", Password: "6a6b66697265756e393834336b6c736b646d6665723065616e6b7666cbfdac6008f9cab4083784cbd1874f76618d2a97"}}}
	auth, err := NewAuthService(repo, strings.Repeat("a", 32))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.GenerateToken("legacy", "wrong"); !errors.Is(err, todo.ErrCredentials) {
		t.Fatalf("wrong password: %v", err)
	}
	if _, err := auth.GenerateToken("legacy", "password123"); err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.users["legacy"].Password), []byte("password123")); err != nil {
		t.Fatal("legacy password was not upgraded")
	}
	if _, err := auth.GenerateToken("legacy", "password123"); err != nil {
		t.Fatal(err)
	}
}

func TestAuthenticationValidation(t *testing.T) {
	repo := &authMemory{users: make(map[string]todo.User)}
	if _, err := NewAuthService(repo, "short"); err == nil {
		t.Fatal("accepted short key")
	}
	auth, err := NewAuthService(repo, strings.Repeat("a", 32))
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range []todo.User{
		{Name: " ", Username: "alice", Password: "password123"},
		{Name: "Alice", Username: "", Password: "password123"},
		{Name: "Alice", Username: "alice", Password: "short"},
		{Name: "Alice", Username: "alice", Password: strings.Repeat("a", 73)},
	} {
		if _, err := auth.CreateUser(user); !errors.Is(err, todo.ErrInvalidInput) {
			t.Fatalf("accepted invalid user: %v", err)
		}
	}
}
