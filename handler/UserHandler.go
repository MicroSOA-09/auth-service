package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MicroSOA-09/auth-service/model"
	"github.com/MicroSOA-09/auth-service/service"
	"github.com/gorilla/mux"
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

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok || id == "" {
		w.WriteHeader(http.StatusNotFound)
		h.Logger.Printf("No id provided")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := h.UserService.GetUser(ctx, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) GetUsernames(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ids, ok := vars["ids"]
	if !ok || ids == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	idList := strings.Split(ids, ",")
	if len(idList) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	users, err := h.UserService.GetUsernames(ctx, idList)
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
