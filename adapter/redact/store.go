package redact

import (
	"sync"

	"github.com/google/uuid"
	"github.com/scylladb/go-set/strset"
)

var (
	_ StoreReader = (*storeReaderCollection)(nil)
	_ Store       = (*store)(nil)
)

type Store interface {
	StoreReader
	StoreWriter
}

type StoreReader interface {
	Values() []string
	identifiable
}

type StoreWriter interface {
	Add(value ...string)
	identifiable
}

type identifiable interface {
	id() string
}

type storeReaderCollection []StoreReader

func (s storeReaderCollection) id() (val string) {
	for _, r := range s {
		val += r.id()
	}
	return val
}

func newStoreReaderCollection(readers ...StoreReader) StoreReader {
	collection := make(storeReaderCollection, 0, len(readers))
	ids := strset.New()
	addReader := func(rs ...StoreReader) {
		for _, r := range rs {
			if ids.Has(r.id()) {
				continue
			}
			collection = append(collection, r)
			ids.Add(r.id())
		}
	}
	for _, r := range readers {
		if rs, ok := r.(storeReaderCollection); ok {
			addReader(rs...)
		} else {
			addReader(r)
		}
	}
	return collection
}

type store struct {
	redactions *strset.Set
	lock       *sync.RWMutex
	_id        string
}

func NewStore(values ...string) Store {
	return &store{
		redactions: strset.New(values...),
		lock:       &sync.RWMutex{},
		_id:        uuid.New().String(),
	}
}

func (w *store) id() string {
	return w._id
}

func (w *store) Add(values ...string) {
	w.lock.Lock()
	defer w.lock.Unlock()
	for _, value := range values {
		if len(value) <= 1 {
			// smallest possible redaction string is larger than 1 character
			return
		}
		w.redactions.Add(value)
	}
}

func (w *store) Values() []string {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return w.redactions.List()
}

func (s storeReaderCollection) Values() (vals []string) {
	for _, r := range s {
		vals = append(vals, r.Values()...)
	}
	return vals
}
