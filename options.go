package mongo

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Options Mongo parameter options
type Options struct {
	// connection string
	Connection string

	// collection name
	Collection string

	// database name
	DB string
}

type record struct {
	ID     primitive.ObjectID `bson:"_id"`
	Sid    string             `bson:"sid"`
	Time   time.Time          `bson:"time"`
	Values string             `bson:"values"`
}
