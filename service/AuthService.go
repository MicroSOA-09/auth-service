package service

import (
	"context"
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

func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.Repo.GetUserByUsername(ctx, username)
	if err != nil {
		return "", err
	}

	if !user.IsActive {
		return "", repository.ErrUserNotActive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", repository.ErrInvalidCredentials
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      user.ID.Hex(),
		"username": user.Username,
		"role":     user.Role,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}).SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return token, nil
}
