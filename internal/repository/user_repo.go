package repository

import (
	"context"
	"errors"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// usersCollection is the user directory in the claystone database.
const usersCollection = "users"

// UserRepo resolves a username (e.g. 10090@pwa.co.th) to its claystone users._id, used to stamp
// _createdBy/_updatedBy on feature documents. Lookups are cached per process.
type UserRepo struct {
	col   *mongo.Collection
	cache sync.Map // username -> primitive.ObjectID
}

func NewUserRepo(db *mongo.Database) *UserRepo {
	return &UserRepo{col: db.Collection(usersCollection)}
}

// Username builds the directory username from a user id and domain,
// e.g. ("10090", "pwa.co.th") -> "10090@pwa.co.th".
func Username(userID, domain string) string {
	return userID + "@" + domain
}

// LookupID returns the users._id for a username. found is false when no such user exists.
func (r *UserRepo) LookupID(ctx context.Context, username string) (id primitive.ObjectID, found bool, err error) {
	if username == "" {
		return primitive.NilObjectID, false, nil
	}
	if v, ok := r.cache.Load(username); ok {
		return v.(primitive.ObjectID), true, nil
	}

	var doc struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	err = r.col.FindOne(ctx, bson.M{"username": username}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return primitive.NilObjectID, false, nil
	}
	if err != nil {
		return primitive.NilObjectID, false, err
	}

	r.cache.Store(username, doc.ID)
	return doc.ID, true, nil
}
