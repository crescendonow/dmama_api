package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"dmama_api/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// catalogCollection is the Vallaris registry that maps an alias to its feature collection id.
const catalogCollection = "collections"

// ErrCollectionNotFound is returned when no catalog entry exists for a branch+shape alias.
var ErrCollectionNotFound = errors.New("feature collection not found")

// FeatureRepo reads/writes Vallaris feature documents. Each (pwa_code, shape) pair maps to a
// features_<catalogId> collection resolved via the `collections` catalog and cached per process.
type FeatureRepo struct {
	db    *mongo.Database
	cache sync.Map // alias -> items collection name
}

func NewFeatureRepo(db *mongo.Database) *FeatureRepo {
	return &FeatureRepo{db: db}
}

// Alias builds the catalog alias for a branch + shape, e.g. b5521040_dma_boundary.
func Alias(pwaCode, shape string) string {
	return "b" + pwaCode + "_" + shape
}

// itemsCollection resolves (and caches) the features_<id> collection for a branch+shape.
// Returns ErrCollectionNotFound when the alias has no catalog entry.
func (r *FeatureRepo) itemsCollection(ctx context.Context, pwaCode, shape string) (*mongo.Collection, error) {
	alias := Alias(pwaCode, shape)
	if name, ok := r.cache.Load(alias); ok {
		return r.db.Collection(name.(string)), nil
	}

	var doc struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	err := r.db.Collection(catalogCollection).FindOne(ctx, bson.M{"alias": alias}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, fmt.Errorf("%w: alias %s", ErrCollectionNotFound, alias)
	}
	if err != nil {
		return nil, err
	}

	name := "features_" + doc.ID.Hex()
	r.cache.Store(alias, name)
	return r.db.Collection(name), nil
}

// Insert stores a new feature (with its pre-generated _id).
func (r *FeatureRepo) Insert(ctx context.Context, pwaCode, shape string, f *model.Feature) error {
	col, err := r.itemsCollection(ctx, pwaCode, shape)
	if err != nil {
		return err
	}
	_, err = col.InsertOne(ctx, f)
	return err
}

// GetByID returns the feature, or (nil, nil) when not found.
func (r *FeatureRepo) GetByID(ctx context.Context, pwaCode, shape string, id primitive.ObjectID) (*model.Feature, error) {
	col, err := r.itemsCollection(ctx, pwaCode, shape)
	if err != nil {
		return nil, err
	}
	var f model.Feature
	err = col.FindOne(ctx, bson.M{"_id": id}).Decode(&f)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// List returns features in a branch+shape, optionally narrowed by a bbox (geoIntersects).
func (r *FeatureRepo) List(ctx context.Context, pwaCode, shape string, bbox []float64) ([]model.Feature, error) {
	col, err := r.itemsCollection(ctx, pwaCode, shape)
	if err != nil {
		return nil, err
	}

	q := bson.M{}
	if len(bbox) == 4 {
		q["geometry"] = bson.M{"$geoIntersects": bson.M{"$geometry": bboxPolygon(bbox)}}
	}

	cur, err := col.Find(ctx, q)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	features := make([]model.Feature, 0)
	if err := cur.All(ctx, &features); err != nil {
		return nil, err
	}
	return features, nil
}

// Update replaces geometry + properties of an existing feature. found is false when id is missing.
func (r *FeatureRepo) Update(ctx context.Context, pwaCode, shape string, id primitive.ObjectID, f *model.Feature) (found bool, err error) {
	col, err := r.itemsCollection(ctx, pwaCode, shape)
	if err != nil {
		return false, err
	}
	res, err := col.UpdateByID(ctx, id, bson.M{"$set": bson.M{
		"type":       "Feature",
		"geometry":   f.Geometry,
		"properties": f.Properties,
	}})
	if err != nil {
		return false, err
	}
	return res.MatchedCount > 0, nil
}

// Delete removes a feature. found is false when id is missing.
func (r *FeatureRepo) Delete(ctx context.Context, pwaCode, shape string, id primitive.ObjectID) (found bool, err error) {
	col, err := r.itemsCollection(ctx, pwaCode, shape)
	if err != nil {
		return false, err
	}
	res, err := col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return false, err
	}
	return res.DeletedCount > 0, nil
}

// MaxDmaID returns the largest properties.dmaId in the branch's dma_boundary collection (0 if none).
func (r *FeatureRepo) MaxDmaID(ctx context.Context, pwaCode string) (int, error) {
	col, err := r.itemsCollection(ctx, pwaCode, model.ShapeDmaBoundary)
	if err != nil {
		return 0, err
	}

	cur, err := col.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "max", Value: bson.D{{Key: "$max", Value: "$properties.dmaId"}}},
		}}},
	})
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)

	if cur.Next(ctx) {
		var res bson.M
		if err := cur.Decode(&res); err != nil {
			return 0, err
		}
		return model.AsInt(res["max"]), nil
	}
	return 0, cur.Err()
}

// DmaIDExists reports whether a dmaId already exists in the branch, optionally excluding one doc.
func (r *FeatureRepo) DmaIDExists(ctx context.Context, pwaCode string, dmaID int, exclude primitive.ObjectID) (bool, error) {
	col, err := r.itemsCollection(ctx, pwaCode, model.ShapeDmaBoundary)
	if err != nil {
		return false, err
	}

	q := bson.M{"properties.dmaId": dmaID}
	if !exclude.IsZero() {
		q["_id"] = bson.M{"$ne": exclude}
	}
	err = col.FindOne(ctx, q).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// bboxPolygon builds a GeoJSON polygon from [minLng, minLat, maxLng, maxLat].
func bboxPolygon(b []float64) bson.M {
	minLng, minLat, maxLng, maxLat := b[0], b[1], b[2], b[3]
	return bson.M{
		"type": "Polygon",
		"coordinates": [][][]float64{{
			{minLng, minLat},
			{maxLng, minLat},
			{maxLng, maxLat},
			{minLng, maxLat},
			{minLng, minLat},
		}},
	}
}
