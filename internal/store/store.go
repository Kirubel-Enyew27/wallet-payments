package store

import (
	"errors"
	"sync"
	"time"

	"wallet-payments-plugin/internal/model"
)

var ErrNotFound = errors.New("payment not found")

type Store struct {
	mu       sync.RWMutex
	payments map[string]*model.Payment
}

func New() *Store {
	return &Store{payments: make(map[string]*model.Payment)}
}

func (s *Store) Create(p *model.Payment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.CreatedAt = time.Now()
	p.UpdatedAt = p.CreatedAt
	s.payments[p.ID] = p
}

func (s *Store) Get(id string) (*model.Payment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.payments[id]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *Store) Update(p *model.Payment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.payments[p.ID]; !ok {
		return ErrNotFound
	}
	p.UpdatedAt = time.Now()
	s.payments[p.ID] = p
	return nil
}
