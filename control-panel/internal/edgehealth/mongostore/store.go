package mongostore

import (
	"context"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/edgehealth"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Store struct {
	collection *mongo.Collection
}

type document struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Service   string             `bson:"service"`
	Instance  string             `bson:"instance"`
	Status    string             `bson:"status"`
	Timestamp time.Time          `bson:"timestamp"`
	Version   string             `bson:"version"`
}

func New(db *mongo.Database) *Store {
	return &Store{collection: db.Collection("health_status")}
}

func (s *Store) Upsert(ctx context.Context, status edgehealth.Status) error {
	filter := bson.M{"service": status.Service, "instance": status.Instance}
	update := bson.M{"$set": bson.M{
		"status": status.Status, "timestamp": status.Timestamp, "version": status.Version,
	}}
	_, err := s.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}
