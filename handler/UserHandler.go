package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/MicroSOA-09/auth-service/model"
	"github.com/MicroSOA-09/auth-service/service"
)

type UserHandler struct {
	Logger      *log.Logger
	UserService *service.UserService
}

func NewUserHandler(userService *service.UserService, logger *log.Logger) *UserHandler {
	return &UserHandler{UserService: userService, Logger: logger}
}

func (h *UserHandler) GetAll(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	users, err := h.UserService.GetAll(ctx)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var result model.PagedResult[model.User]
	result.Results = users
	result.TotalCount = len(users)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
