package mongostore

import (
	"context"
	"fmt"
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
	if err != nil {
		return fmt.Errorf("upsert edge health: %w", err)
	}
	return nil
}

func (s *Store) List(ctx context.Context) ([]edgehealth.Status, error) {
	cursor, err := s.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("find edge health statuses: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []document
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, fmt.Errorf("decode edge health statuses: %w", err)
	}

	statuses := make([]edgehealth.Status, 0, len(documents))
	for _, item := range documents {
		statuses = append(statuses, edgehealth.Status{
			ID:        item.ID.Hex(),
			Service:   item.Service,
			Instance:  item.Instance,
			Status:    item.Status,
			Timestamp: item.Timestamp,
			Version:   item.Version,
		})
	}
	return statuses, nil
}
