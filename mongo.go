package mongo

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"gopkg.in/session.v2"
	"github.com/globalsign/mgo"
	"errors"
	"github.com/globalsign/mgo/bson"
)

var (
	_ session.ManagerStore = &managerStore{}
	_ session.Store        = &store{}
)

// NewMongoStore Create an instance of a mongo store
func NewMongoStore(opt *Options) session.ManagerStore {
	if opt == nil {
		panic("Option cannot be nil")
	}
	sess, err := mgo.DialWithInfo(opt.mongoOptions())
	if err != nil {
		errors.New("Unable to connect: " + err.Error())
	}
	sess.SetMode(mgo.Monotonic, true)
	sess = sess.Clone()

	return &managerStore{sess: sess, cli: sess.DB(opt.DB).C(opt.Collection)}
}

type managerStore struct {
	sess *mgo.Session
	cli *mgo.Collection
}

func (s *managerStore) getValue(sid string) (r record, err error) {
	err = s.cli.Find(bson.M{"sid": sid}).One(&r)
	return
}

func (s *managerStore) parseValue(value string) (map[string]string, error) {
	var values map[string]string

	if len(value) > 0 {
		err := json.Unmarshal([]byte(value), &values)
		if err != nil {
			return nil, err
		}
	}

	if values == nil {
		values = make(map[string]string)
	}
	return values, nil
}

func (s *managerStore) Create(ctx context.Context, sid string, expired int64) (session.Store, error) {
	values := make(map[string]string)

	err := s.cli.Insert(record{
		Id: bson.NewObjectId(),
		Sid: sid,
		Time: time.Now(),
	})
	if err != nil {
		return nil, err
	}

	return &store{ctx: ctx, sid: sid, cli: s.cli, expired: expired, values: values}, nil
}

func (s *managerStore) Update(ctx context.Context, sid string, expired int64) (session.Store, error) {
	r, err := s.getValue(sid)
	if err != nil {
		return nil, err
	}

	r.Time = time.Now().Add(time.Second * time.Duration(expired))

	err = s.cli.UpdateId(r.Id, r)
	if err != nil {
		return nil, err
	}

	values, err := s.parseValue(r.Values)
	if err != nil {
		return nil, err
	}

	return &store{ctx: ctx, sid: sid, cli: s.cli, expired: expired, values: values}, nil
}

func (s *managerStore) Delete(_ context.Context, sid string) error {
	err := s.cli.Remove(bson.M{"sid": sid})
	return err
}

func (s *managerStore) Check(_ context.Context, sid string) (bool, error) {
	var r record
	err := s.cli.Find(bson.M{"sid": sid}).One(&r)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *managerStore) Close() error {
	defer s.sess.Close()
	return nil
}

type store struct {
	sid     string
	cli     *mgo.Collection
	sess     *mgo.Session
	expired int64
	values  map[string]string
	sync.RWMutex
	ctx     context.Context
}

func (s *store) Context() context.Context {
	return s.ctx
}

func (s *store) SessionID() string {
	return s.sid
}

func (s *store) Set(key, value string) {
	s.Lock()
	s.values[key] = value
	s.Unlock()
}

func (s *store) Get(key string) (string, bool) {
	s.RLock()
	defer s.RUnlock()
	val, ok := s.values[key]
	return val, ok
}

func (s *store) Delete(key string) string {
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
	s.values = make(map[string]string)
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
	if err := s.cli.Find(bson.M{"sid": s.sid}).One(&r); err != nil {
		return err
	}

	r.Values = value

	return s.cli.UpdateId(r.Id, r)
}
