// Package cache provides a simple in-memory response cache for API handlers.
// The indexer's realtime broadcaster already computes the same payloads for
// SSE subscribers; this cache stores them so that REST handlers can return
// instantly without touching SQLite — eliminating reader/writer contention
// that causes the dashboard to freeze during indexing.
package cache

import (
	"sync"
	"time"
)

// entry holds a pre-serialized JSON response and its expiry time.
type entry struct {
	data    any
	expires time.Time
}

// Store is a thread-safe key→response cache with per-key TTL.
type Store struct {
	mu      sync.RWMutex
	entries map[string]entry
}

// New creates an empty cache store.
func New() *Store {
	return &Store{entries: make(map[string]entry)}
}

// Set stores a response under the given key with the specified TTL.
func (s *Store) Set(key string, data any, ttl time.Duration) {
	s.mu.Lock()
	s.entries[key] = entry{data: data, expires: time.Now().Add(ttl)}
	s.mu.Unlock()
}

// Get returns the cached response and true if it exists and hasn't expired.
func (s *Store) Get(key string) (any, bool) {
	s.mu.RLock()
	e, ok := s.entries[key]
	s.mu.RUnlock()
	if !ok || time.Now().After(e.expires) {
		return nil, false
	}
	return e.data, true
}

// Default is the global cache instance used by API handlers.
// Populated by the realtime broadcaster after each indexer batch.
var Default = New()
