package repository

import (
	"context"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type HealthRepositoryInterface interface {
	Upsert(ctx context.Context, status domain.HealthStatus) error
}

type healthRepository struct {
	db *mongo.Database
}

func NewHealthRepository(client *mongo.Client, dbName string) HealthRepositoryInterface {
	return &healthRepository{
		db: client.Database(dbName),
	}
}

func (h *healthRepository) Upsert(ctx context.Context, status domain.HealthStatus) error {
	filter := bson.M{
		"service":  status.Service,
		"instance": status.Instance,
	}

	update := bson.M{
		"$set": bson.M{
			"status":    status.Status,
			"timestamp": status.Timestamp,
			"version":   status.Version,
		},
	}

	_, err := h.db.Collection("health_status").UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}
