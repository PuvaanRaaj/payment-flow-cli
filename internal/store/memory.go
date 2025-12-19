// Package store provides in-memory storage for the payment processing system.
package store

import (
	"sort"
	"sync"

	"payment-sim/internal/domain"
)

// Repository defines the interface for payment storage.
type Repository interface {
	Save(payment *domain.Payment) error
	Get(id string) (*domain.Payment, error)
	List() ([]*domain.Payment, error)
	Exists(id string) bool
	RecordBatchID(batchID string)
	GetBatchIDs() []string
	BatchIDExists(batchID string) bool
}

// MemoryStore is an in-memory implementation of Repository.
type MemoryStore struct {
	payments map[string]*domain.Payment
	batchIDs map[string]bool
	mu       sync.RWMutex
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		payments: make(map[string]*domain.Payment),
		batchIDs: make(map[string]bool),
	}
}

// Save stores a payment. If it already exists, it updates it.
func (s *MemoryStore) Save(payment *domain.Payment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payments[payment.ID] = payment
	return nil
}

// Get retrieves a payment by ID.
func (s *MemoryStore) Get(id string) (*domain.Payment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	payment, exists := s.payments[id]
	if !exists {
		return nil, domain.ErrPaymentNotFound
	}
	return payment, nil
}

// List returns all payments sorted by ID.
func (s *MemoryStore) List() ([]*domain.Payment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Collect all payment IDs and sort them
	ids := make([]string, 0, len(s.payments))
	for id := range s.payments {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// Build sorted result
	result := make([]*domain.Payment, 0, len(s.payments))
	for _, id := range ids {
		result = append(result, s.payments[id])
	}
	return result, nil
}

// Exists checks if a payment exists.
func (s *MemoryStore) Exists(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.payments[id]
	return exists
}

// RecordBatchID records a processed batch ID.
func (s *MemoryStore) RecordBatchID(batchID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batchIDs[batchID] = true
}

// GetBatchIDs returns all recorded batch IDs sorted.
func (s *MemoryStore) GetBatchIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.batchIDs))
	for id := range s.batchIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// BatchIDExists checks if a batch ID has been recorded.
func (s *MemoryStore) BatchIDExists(batchID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.batchIDs[batchID]
}
