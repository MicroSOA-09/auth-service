package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/MicroSOA-09/auth-service/model"
	"github.com/MicroSOA-09/auth-service/repository"
	"github.com/MicroSOA-09/auth-service/service"
)

type AuthHandler struct {
	logger      *log.Logger
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService, logger *log.Logger) *AuthHandler {
	return &AuthHandler{authService: authService, logger: logger}
}

func (h *AuthHandler) Register(rw http.ResponseWriter, r *http.Request) {
	var input struct {
		Username     string `json:"username"`
		Password     string `json:"password"`
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		Email        string `json:"email"`
		ProfileImage string `json:"profile_image"`
		Role         string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(rw, "Invalid input", http.StatusBadRequest)
		return
	}

	user := &model.User{
		Username: input.Username,
	}
	if input.Role == "Author" || input.Role == "author" {
		user.Role = model.RoleAuthor
	} else if input.Role == "Tourist" || input.Role == "tourist" {
		user.Role = model.RoleTourist
	} else {
		http.Error(rw, "Invalid role input", http.StatusBadRequest)
		return
	}

	person := &model.Person{
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		Email:        input.Email,
		ProfileImage: input.ProfileImage,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	token, err := h.authService.Register(ctx, user, person, input.Password)

	if err == repository.ErrDuplicateUser {
		http.Error(rw, "Username or email already exists", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(rw, "Failed to register", http.StatusInternalServerError)
		return
	}

	// Send email
	go func() {
		emailCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := h.authService.EmailClient.SendVerificationEmail(emailCtx, person.Email, user.ID.Hex(), token); err != nil {
			h.logger.Printf("Async email send failed to %s: %v", person.Email, err)
		}
	}()

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusCreated)
	json.NewEncoder(rw).Encode(map[string]string{"message": "User registered, please verify email"})

}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	token, err := h.authService.Login(ctx, input.Username, input.Password)
	if err == repository.ErrUserNotFound || err == repository.ErrInvalidCredentials {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	if err == repository.ErrUserNotActive {
		http.Error(w, "Account not verified", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, "Failed to login", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (a *AuthHandler) MiddlewareContentTypeSet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		a.logger.Println("Method [", h.Method, "] - Hit path :", h.URL.Path)

		rw.Header().Add("Content-Type", "application/json")

		next.ServeHTTP(rw, h)
	})
}
