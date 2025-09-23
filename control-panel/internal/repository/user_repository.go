package repository

import (
	"context"
	"errors"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepositoryInterface interface {
	CreateUser(ctx context.Context, c *domain.User) error
	ListUser(ctx context.Context) ([]*domain.User, error)
	DeleteUser(ctx context.Context, id string) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
}

type UserRepository struct {
	db *mongo.Database
}

func NewUserRepository(client *mongo.Client, dbName string) UserRepositoryInterface {
	return &UserRepository{
		db: client.Database(dbName),
	}
}

func (m *UserRepository) CreateUser(ctx context.Context, c *domain.User) error {
	_, err := m.db.Collection("users").InsertOne(ctx, c)
	return err
}

func (m *UserRepository) ListUser(ctx context.Context) ([]*domain.User, error) {
	cur, err := m.db.Collection("users").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*domain.User
	for cur.Next(ctx) {
		var c domain.User
		if err := cur.Decode(&c); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, nil
}

func (m *UserRepository) DeleteUser(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	res, err := m.db.Collection("users").DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("not found")
	}
	return nil
}

func (m *UserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := m.db.Collection("users").FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
