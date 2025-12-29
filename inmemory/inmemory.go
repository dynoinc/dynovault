package inmemory

import (
	"context"
	"sync"

	"github.com/dynoinc/dynovault/handler"
)

type entry struct {
	value   string
	version handler.Version
}

type InMemory struct {
	mu    sync.Mutex
	store map[string]entry
}

func New() *InMemory {
	return &InMemory{
		store: make(map[string]entry),
	}
}

func (m *InMemory) Get(_ context.Context, key []byte) (*handler.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.store[string(key)]
	if !ok {
		return nil, handler.ErrNotFound
	}

	return &handler.Result{
		Value:   []byte(e.value),
		Version: e.version,
	}, nil
}

func (m *InMemory) Put(_ context.Context, key []byte, value []byte, opts ...handler.PutOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var cfg handler.PutConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	k := string(key)
	e, exists := m.store[k]

	if cfg.ExpectedVersion != nil {
		currentVersion := handler.Version(0)
		if exists {
			currentVersion = e.version
		}
		if currentVersion != *cfg.ExpectedVersion {
			return handler.ErrConditionFailed
		}
	}

	newVersion := handler.Version(1)
	if exists {
		newVersion = e.version + 1
	}

	m.store[k] = entry{
		value:   string(value),
		version: newVersion,
	}
	return nil
}

func (m *InMemory) Delete(_ context.Context, key []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.store, string(key))
	return nil
}
