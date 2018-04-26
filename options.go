package mongo

import (
	"crypto/tls"
	"net"
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

	// The network type, either tcp or unix.
	// Default is tcp.
	Network string
	// host:port address.
	Addr string

	// Dialer creates new network connection and has priority over
	// Network and Addr options.
	Dialer func() (net.Conn, error)

	// Optional password. Must match the password specified in the
	// requirepass server configuration option.
	Password string

	// Database to be selected after connecting to the server.
	DB string

	// Maximum number of retries before giving up.
	// Default is to not retry failed commands.
	MaxRetries int
	// Minimum backoff between each retry.
	// Default is 8 milliseconds; -1 disables backoff.
	MinRetryBackoff time.Duration
	// Maximum backoff between each retry.
	// Default is 512 milliseconds; -1 disables backoff.
	MaxRetryBackoff time.Duration

	// Dial timeout for establishing new connections.
	// Default is 5 seconds.
	DialTimeout time.Duration
	// Timeout for socket reads. If reached, commands will fail
	// with a timeout instead of blocking.
	// Default is 3 seconds.
	ReadTimeout time.Duration
	// Timeout for socket writes. If reached, commands will fail
	// with a timeout instead of blocking.
	// Default is ReadTimeout.
	WriteTimeout time.Duration

	// Maximum number of socket connections.
	// Default is 10 connections per every CPU as reported by runtime.NumCPU.
	PoolSize int
	// Amount of time client waits for connection if all connections
	// are busy before returning an error.
	// Default is ReadTimeout + 1 second.
	PoolTimeout time.Duration
	// Amount of time after which client closes idle connections.
	// Should be less than server's timeout.
	// Default is 5 minutes.
	IdleTimeout time.Duration
	// Frequency of idle checks.
	// Default is 1 minute.
	// When minus value is set, then idle check is disabled.
	IdleCheckFrequency time.Duration

	// TLS Config to use. When set TLS will be negotiated.
	TLSConfig *tls.Config
}

type record struct {
	Id     bson.ObjectId `bson:"_id"`
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
