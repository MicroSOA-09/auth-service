package service

import (
	"context"
	"errors"
	"time"

	"github.com/MicroSOA-09/auth-service/model"
	"github.com/MicroSOA-09/auth-service/repository"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	Repo        *repository.UserRepo
	jwtSecret   string
	EmailClient *EmailClient
}

func NewAuthService(repo *repository.UserRepo, jwtSecret string, emailClient *EmailClient) *AuthService {
	return &AuthService{
		Repo:        repo,
		jwtSecret:   jwtSecret,
		EmailClient: emailClient,
	}
}

func (s *AuthService) Register(ctx context.Context, user *model.User, person *model.Person, password string) (string, error) {
	err := s.Repo.CreateUser(ctx, user, person, password)
	if err != nil {
		return "", err
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":    user.ID.Hex(),
		"action": "verify_email",
		"iat":    time.Now().Unix(),
		"exp":    time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(s.jwtSecret))

	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (string, string, error) {
	user, err := s.Repo.GetUserByUsername(ctx, username)
	if err != nil {
		return "", "", err
	}

	if !user.IsActive {
		return "", "", repository.ErrUserNotActive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", repository.ErrInvalidCredentials
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      user.ID.Hex(),
		"username": user.Username,
		"role":     user.Role,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}).SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", err
	}

	return token, user.ID.String(), nil
}

func (s *AuthService) ValidateJWT(tokenString string) (string, string, string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return "", "", "", err
	}

	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok || !token.Valid {
		return "", "", "", errors.New("invalid token claims")
	}

	userID, ok := (*claims)["sub"].(string)
	if !ok {
		return "", "", "", errors.New("missing or invalid sub claim")
	}

	username, ok := (*claims)["username"].(string)
	if !ok {
		return "", "", "", errors.New("missing or invalid username claim")
	}

	role, ok := (*claims)["role"].(string)
	if !ok {
		return "", "", "", errors.New("missing or invalid role claim")
	}

	return userID, username, role, nil
}
