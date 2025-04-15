package service

import (
	"context"
	"fmt"

	"github.com/MicroSOA-09/auth-service/model"
	"github.com/MicroSOA-09/auth-service/repository"
)

type UserService struct {
	UserRepo *repository.UserRepo
}

func NewUserService(repo *repository.UserRepo) *UserService {
	return &UserService{
		UserRepo: repo,
	}
}

func (service *UserService) GetAll(ctx context.Context) ([]model.User, error) {
	blogs, err := service.UserRepo.GetAll(ctx)
	// HTTP REQ to ASP.NET application to get author info
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("there are no blogs"))
	}
	return blogs, nil
}
