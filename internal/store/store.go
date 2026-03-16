package store

import (
	"context"
	"errors"
	"time"

	"wallet-payments-plugin/internal/model"
)

var ErrNotFound = errors.New("payment not found")

type ListFilter struct {
	Status        model.PaymentStatus
	PaymentMethod string
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
	Limit         int
	Offset        int
}

type Store interface {
	Create(ctx context.Context, p *model.Payment) error
	Get(ctx context.Context, id string) (*model.Payment, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*model.Payment, error)
	Update(ctx context.Context, p *model.Payment) error
	List(ctx context.Context, filter ListFilter) ([]*model.Payment, error)
	Health(ctx context.Context) error
}
