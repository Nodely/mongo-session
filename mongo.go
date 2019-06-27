package mongo

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/op/go-logging"
	"gopkg.in/session.v3"
)

var (
	_ session.ManagerStore = &managerStore{}
	_ session.Store        = &store{}
)

// NewMongoStore Create an instance of a mongo store
func NewMongoStore(opt *Options) (session.ManagerStore, error) {
	if opt == nil {
		panic("Option cannot be nil")
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(opt.Connection))
	if err != nil {
		return nil, errors.New("Unable to get connection string: " + err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err = client.Connect(ctx)

	if err != nil {
		return nil, errors.New("Unable to connect: " + err.Error())
	}

	return &managerStore{logger: opt.Logger, client: client, col: client.Database(opt.DB).Collection(opt.Collection), ctx: ctx}, nil
}

type managerStore struct {
	client *mongo.Client
	col    *mongo.Collection
	ctx    context.Context
	logger *logging.Logger
}

func (s *managerStore) getValue(sid string) (r record, err error) {
	err = s.col.FindOne(s.ctx, bson.M{"sid": sid}).Decode(&r)
	return
}

func (s *managerStore) parseValue(value string) (map[string]interface{}, error) {
	var values map[string]interface{}

	if len(value) > 0 {
		err := json.Unmarshal([]byte(value), &values)
		if err != nil {
			return nil, err
		}
	}

	if values == nil {
		values = make(map[string]interface{})
	}
	return values, nil
}

func (s *managerStore) Create(ctx context.Context, sid string, expired int64) (session.Store, error) {
	values := make(map[string]interface{})

	_, err := s.col.InsertOne(s.ctx, record{
		Sid:  sid,
		Time: time.Now(),
	})
	if err != nil {
		s.logger.Errorf("Store.Crate: %s", err.Error())
		return nil, err
	}

	return &store{ctx: ctx, sid: sid, cli: s.col, expired: expired, values: values}, nil
}

func (s *managerStore) Update(ctx context.Context, sid string, expired int64) (session.Store, error) {
	r, err := s.getValue(sid)
	if err != nil {
		return nil, err
	}

	r.Time = time.Now().Add(time.Second * time.Duration(expired))

	_, err = s.col.UpdateOne(ctx, bson.M{"_id": r.ID}, r)
	if err != nil {
		return nil, err
	}

	values, err := s.parseValue(r.Values)
	if err != nil {
		return nil, err
	}

	return &store{ctx: ctx, sid: sid, cli: s.col, expired: expired, values: values}, nil
}

func (s *managerStore) Refresh(ctx context.Context, oldsid, sid string, expired int64) (session.Store, error) {
	r, err := s.getValue(sid)
	if err != nil {
		return nil, err
	}

	r.Time = time.Now().Add(time.Second * time.Duration(expired))

	values, err := s.parseValue(r.Values)
	if err != nil {
		return nil, err
	}

	return &store{ctx: ctx, sid: sid, cli: s.col, expired: expired, values: values}, nil
}

func (s *managerStore) Delete(_ context.Context, sid string) error {
	_, err := s.col.DeleteOne(s.ctx, bson.M{"sid": sid})
	return err
}

func (s *managerStore) Check(_ context.Context, sid string) (bool, error) {
	var r record
	err := s.col.FindOne(s.ctx, bson.M{"sid": sid}).Decode(&r)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *managerStore) Close() error {
	defer s.client.Disconnect(s.ctx)
	return nil
}

type store struct {
	sid     string
	cli     *mongo.Collection
	expired int64
	values  map[string]interface{}
	sync.RWMutex
	ctx context.Context
}

func (s *store) Context() context.Context {
	return s.ctx
}

func (s *store) SessionID() string {
	return s.sid
}

func (s *store) Set(key string, value interface{}) {
	s.Lock()
	s.values[key] = value
	s.Unlock()
}

func (s *store) Get(key string) (interface{}, bool) {
	s.RLock()
	defer s.RUnlock()
	val, ok := s.values[key]
	return val, ok
}

func (s *store) Delete(key string) interface{} {
	s.RLock()
	v, ok := s.values[key]
	s.RUnlock()
	if ok {
		s.Lock()
		delete(s.values, key)
		s.Unlock()
	}
	return v
}

func (s *store) Flush() error {
	s.Lock()
	s.values = make(map[string]interface{})
	s.Unlock()
	return s.Save()
}

func (s *store) Save() error {
	var value string

	s.RLock()
	if len(s.values) > 0 {
		buf, _ := json.Marshal(s.values)
		value = string(buf)
	}
	s.RUnlock()

	// find id
	var r record
	if err := s.cli.FindOne(s.ctx, bson.M{"sid": s.sid}).Decode(&r); err != nil {
		return err
	}

	r.Values = value

	res := s.cli.FindOneAndUpdate(s.ctx, bson.M{"_id": r.ID}, r)
	return res.Err()
}
