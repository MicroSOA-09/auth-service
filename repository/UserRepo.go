package repository

import (
	"context"
	"errors"
	"fmt"
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

func (repo *UserRepo) GetUser(ctx context.Context, id primitive.ObjectID) (*model.User, error) {
	user := model.User{}
	err := repo.users.FindOne(ctx, bson.M{"is_active": true, "_id": id}).Decode(&user)
	if err != nil {
		repo.logger.Printf("No user with that id %v", id)
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func (repo *UserRepo) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	user := model.User{}
	err := repo.users.FindOne(ctx, bson.M{"is_active": true, "username": username}).Decode(&user)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func (repo *UserRepo) GetUserByIds(ctx context.Context, ids []primitive.ObjectID) ([]model.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Ukloni duplikate
	uniqueIDs := make(map[primitive.ObjectID]struct{})
	for _, id := range ids {
		uniqueIDs[id] = struct{}{}
	}
	idList := make([]primitive.ObjectID, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		idList = append(idList, id)
	}
	fmt.Printf("Unique IDs: %v\n", idList)

	// Upit
	cursor, err := repo.users.Find(ctx, bson.M{
		"_id": bson.M{"$in": idList},
	})
	if err != nil {
		fmt.Printf("Find error: %v\n", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	// Dohvati korisnike
	var users []model.User
	if err := cursor.All(ctx, &users); err != nil {
		fmt.Printf("Cursor error: %v\n", err)
		return nil, err
	}

	// Detaljan ispis
	fmt.Printf("Found %d users:\n", len(users))
	for i, user := range users {
		fmt.Printf("User %d: ID=%s, Username=%s\n", i+1, user.ID.Hex(), user.Username)
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
