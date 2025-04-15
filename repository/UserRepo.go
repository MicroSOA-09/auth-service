package repository

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/MicroSOA-09/auth-service/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateUser      = errors.New("username or email already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotActive      = errors.New("user not active")
)

type UserRepo struct {
	cli     *mongo.Client
	logger  *log.Logger
	users   *mongo.Collection
	persons *mongo.Collection
}

func (repo *UserRepo) GetAll(ctx context.Context) ([]model.User, error) {
	users := []model.User{}
	cursor, err := repo.users.Find(ctx, bson.M{"is_active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func New(ctx context.Context, logger *log.Logger) (*UserRepo, error) {
	dbURI := os.Getenv("MONGO_DB_URI")
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbURI))
	if err != nil {
		return nil, err
	}

	db := client.Database("auth")
	users := db.Collection("users")
	persons := db.Collection("persons")

	_, err = users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"username": 1}, Options: options.Index().SetUnique(true),
	})
	if err != nil {
		client.Disconnect(ctx)
		return nil, err
	}

	_, err = persons.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.M{"user_id": 1}, Options: options.Index().SetUnique(true)},
		{Keys: bson.M{"email": 1}, Options: options.Index().SetUnique(true)},
	})
	if err != nil {
		client.Disconnect(ctx)
		return nil, err
	}

	return &UserRepo{
		cli:     client,
		logger:  logger,
		users:   users,
		persons: persons,
	}, nil
}

func (userRepo *UserRepo) Disconnect(ctx context.Context) error {
	err := userRepo.cli.Disconnect(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (r *UserRepo) CreateUser(ctx context.Context, user *model.User, person *model.Person, password string) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		r.logger.Printf("Failed to hash password: %v", err)
		return err
	}

	user.PasswordHash = string(passwordHash)
	user.ID = primitive.NewObjectID()
	user.IsActive = false

	person.ID = primitive.NewObjectID()
	person.UserID = user.ID

	session, err := r.cli.StartSession()
	if err != nil {
		r.logger.Printf("Failed to start session: %v", err)
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		count, err := r.users.CountDocuments(sc, bson.M{"$or": []bson.M{
			{"username": user.Username},
			{"email": person.Email},
		}})
		if err != nil {
			r.logger.Printf("Error checking uniqueness: %%v", err)
			return nil, err
		}
		if count > 0 {
			r.logger.Print("Username %s or email %s already exists", user.Username, person.Email)
			return nil, ErrDuplicateUser
		}

		_, err = r.users.InsertOne(sc, user)
		if err != nil {
			r.logger.Printf("Failed to insert user: %v", err)
			return nil, err
		}

		_, err = r.persons.InsertOne(sc, person)
		if err != nil {
			r.logger.Printf("Failed to insert person: %v", err)
			return nil, err
		}

		return nil, nil
	})

	return err
}

func (r *UserRepo) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	user := &model.User{}
	err := r.users.FindOne(ctx, bson.M{"username": username}).Decode(user)
	if err == mongo.ErrNoDocuments {
		r.logger.Printf("User not found: %s", username)
		return nil, ErrUserNotFound
	}
	if err != nil {
		r.logger.Printf("Error finding user: %v", err)
		return nil, err
	}
	return user, nil
}
