package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
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
		h.logger.Printf("Register endpoint - invalid input")
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
		h.logger.Printf("Register endpoint - invalid ROLE input")
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
		h.logger.Printf("Register endpoint - username/email already exists")
		http.Error(rw, "Username or email already exists", http.StatusConflict)
		return
	}
	if err != nil {
		h.logger.Printf("Register endpoint - Failed to register")
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
		h.logger.Printf("Login endpoint - invalid input")
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	token, userId, err := h.authService.Login(ctx, input.Username, input.Password)
	if err == repository.ErrUserNotFound || err == repository.ErrInvalidCredentials {
		h.logger.Printf("Login endpoint - invalid credentials")
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	if err == repository.ErrUserNotActive {
		h.logger.Printf("Login endpoint - account not verified")
		http.Error(w, "Account not verified", http.StatusForbidden)
		return
	}
	if err != nil {
		h.logger.Printf("Login endpoint - failed to login %v", err)
		http.Error(w, "Failed to login", http.StatusInternalServerError)
		return
	}

	response := struct {
		ID          string `json:"id"`
		AccessToken string `json:"accessToken"`
	}{
		ID:          userId,
		AccessToken: token,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) ValidateJWT(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.logger.Printf("missing Authorization header")
		h.writeResponse(w, http.StatusUnauthorized, map[string]string{"error": "missing Authorization header"})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		h.logger.Printf("invalid Authorization format")
		h.writeResponse(w, http.StatusUnauthorized, map[string]string{"error": "invalid Authorization format"})
		return
	}

	userID, username, role, err := h.authService.ValidateJWT(parts[1])
	if err != nil {
		h.logger.Printf("JWT validation failed: %v", err)
		h.writeResponse(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	response := map[string]string{
		"userID":   userID,
		"username": username,
		"role":     role,
	}
	h.writeResponse(w, http.StatusOK, response)
}

func (a *AuthHandler) MiddlewareContentTypeSet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		a.logger.Println("Method [", h.Method, "] - Hit path :", h.URL.Path)

		rw.Header().Add("Content-Type", "application/json")

		next.ServeHTTP(rw, h)
	})
}

func (h *AuthHandler) writeResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Printf("Failed to write response: %v", err)
	}
}
