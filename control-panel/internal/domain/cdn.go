package domain

import "go.mongodb.org/mongo-driver/bson/primitive"

type CDN struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Origin   string             `bson:"origin" json:"origin"`
	Domain   string             `bson:"domain" json:"domain"`
	IsActive bool               `bson:"is_active" json:"is_active"`
	CacheTTL uint               `bson:"cache_ttl" json:"cache_ttl"`
}
