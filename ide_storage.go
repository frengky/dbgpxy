package dbgpxy

import (
	"fmt"
	"sync"
)

// IDEStorage define way to manage list of registered IDE
type IDEStorage interface {
	Get(key string) (IDE, error)
	Put(key string, ide IDE)
	Has(key string) bool
	Forget(key string)
}

type simpleIDEStorage struct {
	list map[string]IDE
	lock sync.RWMutex
}

// NewIDEStorage create a default implementation of IDEStorage
func NewIDEStorage() IDEStorage {
	return &simpleIDEStorage{
		list: make(map[string]IDE),
		lock: sync.RWMutex{},
	}
}

func (s *simpleIDEStorage) Get(key string) (IDE, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ide, ok := s.list[key]
	if !ok {
		return nil, fmt.Errorf("ide key %s was not found", key)
	}
	return ide, nil
}

func (s *simpleIDEStorage) Put(key string, ide IDE) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.list[key] = ide
}

func (s *simpleIDEStorage) Has(key string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if _, ok := s.list[key]; ok {
		return true
	}
	return false
}

func (s *simpleIDEStorage) Forget(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.list, key)
}
