package repository

import (
	"context"
	"control-panel/internal/domain"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CdnRepositoryInterface interface {
	CreateCDN(ctx context.Context, c *domain.CDN) error
	ListCDNs(ctx context.Context) ([]*domain.CDN, error)
	GetCDN(ctx context.Context, id string) (*domain.CDN, error)
	UpdateCDN(ctx context.Context, id string, c *domain.CDN) error
	DeleteCDN(ctx context.Context, id string) error
	GetCDNByOrigin(ctx context.Context, origin string) (*domain.CDN, error)
}

type CdnRepository struct {
	db *mongo.Database
}

func NewCdnRepository(client *mongo.Client, dbName string) CdnRepositoryInterface {
	return &CdnRepository{
		db: client.Database(dbName),
	}
}

func (m *CdnRepository) CreateCDN(ctx context.Context, c *domain.CDN) error {
	_, err := m.db.Collection("cdns").InsertOne(ctx, c)
	return err
}

func (m *CdnRepository) ListCDNs(ctx context.Context) ([]*domain.CDN, error) {
	cur, err := m.db.Collection("cdns").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*domain.CDN
	for cur.Next(ctx) {
		var c domain.CDN
		if err := cur.Decode(&c); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, nil
}

func (m *CdnRepository) GetCDN(ctx context.Context, id string) (*domain.CDN, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var c domain.CDN
	err = m.db.Collection("cdns").FindOne(ctx, bson.M{"_id": oid}).Decode(&c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (m *CdnRepository) UpdateCDN(ctx context.Context, id string, c *domain.CDN) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = m.db.Collection("cdns").UpdateOne(
		ctx,
		bson.M{"_id": oid},
		bson.M{"$set": bson.M{
			"origin":    c.Origin,
			"domain":    c.Domain,
			"is_active": c.IsActive,
		}},
	)
	return err
}

func (m *CdnRepository) DeleteCDN(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	res, err := m.db.Collection("cdns").DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("not found")
	}
	return nil
}

func (m *CdnRepository) GetCDNByOrigin(ctx context.Context, origin string) (*domain.CDN, error) {
	var cdn domain.CDN
	err := m.db.Collection("cdns").FindOne(ctx, bson.M{"origin": origin}).Decode(&cdn)
	if err != nil {
		return nil, err
	}
	return &cdn, nil
}
