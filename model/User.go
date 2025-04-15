package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserRole string

const (
	RoleAdmin   UserRole = "Administrator"
	RoleAuthor  UserRole = "Author"
	RoleTourist UserRole = "Tourist"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	Username     string             `bson:"username" json:"username"`
	PasswordHash string             `bson:"password_hash" json:"-"`
	Role         UserRole           `bson:"role" json:"role"`
	IsActive     bool               `bson:"is_active" json:"-"`
}

type Person struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	UserID       primitive.ObjectID `bson:"user_id" json:"user_id"`
	FirstName    string             `bson:"first_name" json:"first_name"`
	LastName     string             `bson:"last_name" json:"last_name"`
	Email        string             `bson:"email" json:"email"`
	ProfileImage string             `bson:"profile_image" json:"profile_image"`
}

type PagedResult[T any] struct {
	Results    []T `json:"results"`
	TotalCount int `json:"totalCount"`
}
