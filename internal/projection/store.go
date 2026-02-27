package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Store is the interface for projection persistence (Redis-backed in production).
type Store interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// InMemoryStore is a simple in-memory projection store for development/testing.
type InMemoryStore struct {
	data map[string]entry
}

type entry struct {
	value     []byte
	expiresAt time.Time
}

// NewInMemoryStore creates a new in-memory projection store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{data: make(map[string]entry)}
}

func (s *InMemoryStore) Get(_ context.Context, key string) ([]byte, error) {
	e, ok := s.data[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		delete(s.data, key)
		return nil, fmt.Errorf("key expired: %s", key)
	}
	return e.value, nil
}

func (s *InMemoryStore) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	s.data[key] = entry{value: value, expiresAt: exp}
	return nil
}

func (s *InMemoryStore) Delete(_ context.Context, key string) error {
	delete(s.data, key)
	return nil
}

// SetJSON is a convenience helper to serialize and store a value.
func SetJSON(ctx context.Context, store Store, key string, v interface{}, ttl time.Duration) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal projection: %w", err)
	}
	return store.Set(ctx, key, data, ttl)
}

// GetJSON is a convenience helper to retrieve and deserialize a value.
func GetJSON(ctx context.Context, store Store, key string, dest interface{}) error {
	data, err := store.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}
