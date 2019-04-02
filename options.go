package mongo

import (
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
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
	ID     bson.ObjectId `bson:"_id"`
	Sid    string        `bson:"sid"`
	Time   time.Time     `bson:"time"`
	Values string        `bson:"values"`
}

func (o *Options) mongoOptions() *mgo.DialInfo {
	di, _ := mgo.ParseURL(o.Connection)
	if o.Collection == "" {
		o.Collection = "sessions"
	}
	o.DB = di.Database
	return di
}
