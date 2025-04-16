package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/MicroSOA-09/auth-service/model"
	"github.com/MicroSOA-09/auth-service/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	users, err := service.UserRepo.GetAll(ctx)
	// HTTP REQ to ASP.NET application to get author info
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("there are no users"))
	}
	return users, nil
}

func (service *UserService) GetUser(ctx context.Context, id string) (*model.User, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid user ID format")
	}
	user, err := service.UserRepo.GetUser(ctx, oid)
	// HTTP REQ to ASP.NET application to get author info
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("there are no user"))
	}
	return user, nil
}

func (service *UserService) GetUsernames(ctx context.Context, ids []string) ([]model.User, error) {
	// Validacija ID-ova
	var objectIds []primitive.ObjectID
	for _, id := range ids {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return nil, errors.New("invalid user ID format")
		}
		objectIds = append(objectIds, oid)
	}

	// Poziv repozitorijuma
	users, err := service.UserRepo.GetUserByIds(ctx, objectIds)
	if err != nil {
		return nil, err
	}

	return users, nil
}
