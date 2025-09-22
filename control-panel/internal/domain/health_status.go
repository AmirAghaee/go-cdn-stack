package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type HealthStatus struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Service   string             `bson:"service" json:"service"`
	Instance  string             `bson:"instance" json:"instance"`
	Status    string             `bson:"status" json:"status"`
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`
	Version   string             `bson:"version" json:"version"`
}
