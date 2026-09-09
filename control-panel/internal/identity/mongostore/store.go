package mongostore

import (
	"context"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/identity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Store struct {
	collection *mongo.Collection
}

type document struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Email        string             `bson:"email"`
	PasswordHash string             `bson:"password"`
	CreatedAt    time.Time          `bson:"created_at"`
}

func New(db *mongo.Database) *Store {
	return &Store{collection: db.Collection("users")}
}

func (s *Store) Create(ctx context.Context, user *identity.User) error {
	_, err := s.collection.InsertOne(ctx, fromUser(user))
	return err
}

func (s *Store) List(ctx context.Context) ([]*identity.User, error) {
	cursor, err := s.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []*identity.User
	for cursor.Next(ctx) {
		var doc document
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		users = append(users, doc.toUser())
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (s *Store) FindByEmail(ctx context.Context, email string) (*identity.User, error) {
	var doc document
	if err := s.collection.FindOne(ctx, bson.M{"email": email}).Decode(&doc); err != nil {
		return nil, err
	}
	return doc.toUser(), nil
}

func fromUser(user *identity.User) document {
	doc := document{Email: user.Email, PasswordHash: user.PasswordHash, CreatedAt: user.CreatedAt}
	if user.ID != "" {
		doc.ID, _ = primitive.ObjectIDFromHex(user.ID)
	}
	return doc
}

func (d document) toUser() *identity.User {
	return &identity.User{
		ID: d.ID.Hex(), Email: d.Email, PasswordHash: d.PasswordHash, CreatedAt: d.CreatedAt,
	}
}
