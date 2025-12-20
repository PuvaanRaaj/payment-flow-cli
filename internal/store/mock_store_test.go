package store

import (
	"payment-sim/internal/domain"

	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of Repository for testing.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Save(payment *domain.Payment) error {
	args := m.Called(payment)
	return args.Error(0)
}

func (m *MockRepository) Get(id string) (*domain.Payment, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *MockRepository) List() ([]*domain.Payment, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Payment), args.Error(1)
}

func (m *MockRepository) Exists(id string) bool {
	args := m.Called(id)
	return args.Bool(0)
}

func (m *MockRepository) RecordBatchID(batchID string) {
	m.Called(batchID)
}

func (m *MockRepository) GetBatchIDs() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockRepository) BatchIDExists(batchID string) bool {
	args := m.Called(batchID)
	return args.Bool(0)
}
