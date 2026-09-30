package pagination

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Handle represents a stored chunk of deferred data
type Handle struct {
	ID        string          `json:"id"`
	Data      json.RawMessage `json:"-"`
	CreatedAt time.Time       `json:"created_at"`
	ExpiresAt time.Time       `json:"expires_at"`
	TotalSize int             `json:"total_size"`
}

// Store manages deferred data handles with automatic TTL cleanup
type Store struct {
	mu      sync.RWMutex
	handles map[string]*Handle
	ttl     time.Duration
}

// NewStore creates a handle store with the given TTL (default 10 minutes)
func NewStore(ttl time.Duration) *Store {
	if ttl == 0 {
		ttl = 10 * time.Minute
	}
	return &Store{
		handles: make(map[string]*Handle),
		ttl:     ttl,
	}
}

func (s *Store) cleanup() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, handle := range s.handles {
		if now.After(handle.ExpiresAt) {
			delete(s.handles, id)
		}
	}
}

// Put stores data and returns a handle ID
func (s *Store) Put(data json.RawMessage) string {
	s.cleanup()

	id := uuid.New().String()
	now := time.Now()

	handle := &Handle{
		ID:        id,
		Data:      data,
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl),
		TotalSize: len(data),
	}

	s.mu.Lock()
	s.handles[id] = handle
	s.mu.Unlock()

	return id
}

// Fetch retrieves a slice of the stored data by handle ID.
// offset and limit are byte offsets into the raw JSON.
// Returns the slice, total size, and whether the handle exists.
func (s *Store) Fetch(handleID string, offset, limit int) (json.RawMessage, int, bool) {
	s.cleanup()

	s.mu.RLock()
	handle, exists := s.handles[handleID]
	s.mu.RUnlock()

	if !exists {
		return nil, 0, false
	}

	totalSize := handle.TotalSize

	if offset >= totalSize {
		return json.RawMessage(""), totalSize, true
	}

	end := offset + limit
	if end > totalSize {
		end = totalSize
	}

	slice := handle.Data[offset:end]

	return slice, totalSize, true
}

// Count returns number of active handles
func (s *Store) Count() int {
	s.cleanup()

	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.handles)
}
