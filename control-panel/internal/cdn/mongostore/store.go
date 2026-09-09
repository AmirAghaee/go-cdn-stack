package mongostore

import (
	"context"
	"errors"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/cdn"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Store struct {
	collection *mongo.Collection
}

type document struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Origin   string             `bson:"origin"`
	Domain   string             `bson:"domain"`
	IsActive bool               `bson:"is_active"`
	CacheTTL uint               `bson:"cache_ttl"`
}

func New(db *mongo.Database) *Store {
	return &Store{collection: db.Collection("cdns")}
}

func (s *Store) Create(ctx context.Context, item *cdn.CDN) error {
	_, err := s.collection.InsertOne(ctx, fromCDN(item))
	return err
}

func (s *Store) List(ctx context.Context) ([]*cdn.CDN, error) {
	cursor, err := s.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []*cdn.CDN
	for cursor.Next(ctx) {
		var doc document
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		items = append(items, doc.toCDN())
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) Get(ctx context.Context, id string) (*cdn.CDN, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var doc document
	if err := s.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc); err != nil {
		return nil, err
	}
	return doc.toCDN(), nil
}

func (s *Store) Update(ctx context.Context, id string, item *cdn.CDN) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = s.collection.UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": bson.M{
		"origin":    item.Origin,
		"domain":    item.Domain,
		"is_active": item.IsActive,
		"cache_ttl": item.CacheTTL,
	}})
	return err
}

func (s *Store) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	result, err := s.collection.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return errors.New("not found")
	}
	return nil
}

func (s *Store) FindByOrigin(ctx context.Context, origin string) (*cdn.CDN, error) {
	var doc document
	if err := s.collection.FindOne(ctx, bson.M{"origin": origin}).Decode(&doc); err != nil {
		return nil, err
	}
	return doc.toCDN(), nil
}

func fromCDN(item *cdn.CDN) document {
	doc := document{
		Origin: item.Origin, Domain: item.Domain, IsActive: item.IsActive, CacheTTL: item.CacheTTL,
	}
	if item.ID != "" {
		doc.ID, _ = primitive.ObjectIDFromHex(item.ID)
	}
	return doc
}

func (d document) toCDN() *cdn.CDN {
	return &cdn.CDN{
		ID: d.ID.Hex(), Origin: d.Origin, Domain: d.Domain, IsActive: d.IsActive, CacheTTL: d.CacheTTL,
	}
}
