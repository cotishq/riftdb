package storage

import (
	"context"
	"strings"
	"sync"
)

// Memory is an in-process object store for tests and local discovery.
type Memory struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func NewMemory() *Memory {
	return &Memory{objects: make(map[string][]byte)}
}

var _ Storage = (*Memory)(nil)

func (m *Memory) Put(_ context.Context, key string, data []byte) error {
	key = strings.TrimSpace(key)
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	m.objects[key] = cp
	return nil
}

func (m *Memory) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.objects[key]
	if !ok {
		return nil, ErrNotFound
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, key)
	return nil
}

func (m *Memory) List(_ context.Context, prefix string) ([]Object, error) {
	prefix = strings.TrimSpace(prefix)
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]Object, 0)
	for key := range m.objects {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			out = append(out, Object{Key: key})
		}
	}
	return out, nil
}
