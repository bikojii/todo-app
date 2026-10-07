package service

import (
	"crypto/sha1"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	todo "github.com/bikojii/todo-app"
	"github.com/bikojii/todo-app/pkg/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type tokenClaims struct {
	jwt.RegisteredClaims
	UserId int `json:"user_id"`
}

type AuthService struct {
	repo       repository.Authorization
	signingKey []byte
	dummyHash  []byte
}

func NewAuthService(repo repository.Authorization, signingKey string) (*AuthService, error) {
	if len(signingKey) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	dummy, err := bcrypt.GenerateFromPassword([]byte("invalid-login-placeholder"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &AuthService{repo: repo, signingKey: []byte(signingKey), dummyHash: dummy}, nil
}

func (s *AuthService) CreateUser(user todo.User) (int, error) {
	if err := user.Validate(); err != nil {
		return 0, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	user.Password = string(hash)
	return s.repo.CreateUser(user)
}

func (s *AuthService) GenerateToken(username, password string) (string, error) {
	if err := todo.ValidateText(username, "username", true); err != nil {
		return "", todo.ErrCredentials
	}
	if len(password) == 0 || len(password) > 72 {
		return "", todo.ErrCredentials
	}
	user, err := s.repo.GetUser(username)
	if errors.Is(err, todo.ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return "", todo.ErrCredentials
	}
	if err != nil {
		return "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		legacy := fmt.Sprintf("%x", append([]byte("jkfireun9843klskdmfer0eankvf"), sha1Digest(password)...))
		if subtle.ConstantTimeCompare([]byte(user.Password), []byte(legacy)) != 1 {
			return "", todo.ErrCredentials
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return "", err
		}
		if err := s.repo.UpdatePassword(user.Id, string(hash)); err != nil {
			return "", err
		}
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserId: user.Id,
	})
	return token.SignedString(s.signingKey)
}

func sha1Digest(password string) []byte {
	digest := sha1.Sum([]byte(password))
	return digest[:]
}

func (s *AuthService) ParseToken(accessToken string) (int, error) {
	token, err := jwt.ParseWithClaims(accessToken, &tokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.signingKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || token == nil || !token.Valid {
		return 0, todo.ErrToken
	}
	claims, ok := token.Claims.(*tokenClaims)
	if !ok || claims.UserId <= 0 {
		return 0, todo.ErrToken
	}
	return claims.UserId, nil
}
