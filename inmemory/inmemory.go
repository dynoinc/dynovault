package inmemory

import (
	"context"
	"sync"

	"github.com/dynoinc/dynovault/handler"
)

type InMemory struct {
	mu    sync.Mutex
	store map[string]string
}

func New() *InMemory {
	return &InMemory{
		store: make(map[string]string),
	}
}

func (m *InMemory) Get(_ context.Context, key []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, ok := m.store[string(key)]
	if !ok {
		return nil, handler.ErrNotFound
	}

	return []byte(v), nil
}

func (m *InMemory) Put(_ context.Context, key []byte, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store[string(key)] = string(value)
	return nil
}

func (m *InMemory) Delete(_ context.Context, key []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.store, string(key))
	return nil
}
