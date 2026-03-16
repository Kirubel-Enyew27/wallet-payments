package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"wallet-payments-plugin/internal/config"
	"wallet-payments-plugin/internal/model"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(cfg config.Config) (*SQLStore, error) {
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetimeM) * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	store := &SQLStore{db: db}
	if err := store.ensureSchema(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *SQLStore) ensureSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS payments (
			id TEXT PRIMARY KEY,
			payment_method TEXT NOT NULL,
			amount TEXT NOT NULL,
			phone_number TEXT NOT NULL,
			status TEXT NOT NULL,
			channel TEXT NOT NULL,
			payment_url TEXT NOT NULL DEFAULT '',
			provider_ref TEXT NOT NULL DEFAULT '',
			failed_reason TEXT NOT NULL DEFAULT '',
			idempotency_key TEXT UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS payments_status_idx ON payments(status)`,
		`CREATE INDEX IF NOT EXISTS payments_method_idx ON payments(payment_method)`,
		`CREATE INDEX IF NOT EXISTS payments_created_idx ON payments(created_at)`,
	}

	for _, q := range queries {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLStore) Create(ctx context.Context, p *model.Payment) error {
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO payments (
			id, payment_method, amount, phone_number, status, channel,
			payment_url, provider_ref, failed_reason, idempotency_key
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING created_at, updated_at
	`, p.ID, p.PaymentMethod, p.Amount, p.PhoneNumber, string(p.Status), string(p.Channel),
		p.PaymentURL, p.ProviderRef, p.FailedReason, nullIfEmpty(p.IdempotencyKey),
	)

	return row.Scan(&p.CreatedAt, &p.UpdatedAt)
}

func (s *SQLStore) Get(ctx context.Context, id string) (*model.Payment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, payment_method, amount, phone_number, status, channel,
			payment_url, provider_ref, failed_reason, idempotency_key,
			created_at, updated_at
		FROM payments
		WHERE id = $1
	`, id)
	return scanPayment(row)
}

func (s *SQLStore) GetByIdempotencyKey(ctx context.Context, key string) (*model.Payment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, payment_method, amount, phone_number, status, channel,
			payment_url, provider_ref, failed_reason, idempotency_key,
			created_at, updated_at
		FROM payments
		WHERE idempotency_key = $1
	`, key)
	return scanPayment(row)
}

func (s *SQLStore) Update(ctx context.Context, p *model.Payment) error {
	row := s.db.QueryRowContext(ctx, `
		UPDATE payments
		SET payment_method = $2,
			amount = $3,
			phone_number = $4,
			status = $5,
			channel = $6,
			payment_url = $7,
			provider_ref = $8,
			failed_reason = $9,
			idempotency_key = $10,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`, p.ID, p.PaymentMethod, p.Amount, p.PhoneNumber, string(p.Status), string(p.Channel),
		p.PaymentURL, p.ProviderRef, p.FailedReason, nullIfEmpty(p.IdempotencyKey),
	)

	if err := row.Scan(&p.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *SQLStore) List(ctx context.Context, filter ListFilter) ([]*model.Payment, error) {
	query := `
		SELECT id, payment_method, amount, phone_number, status, channel,
			payment_url, provider_ref, failed_reason, idempotency_key,
			created_at, updated_at
		FROM payments
	`

	conds := make([]string, 0, 4)
	args := make([]interface{}, 0, 6)
	argIdx := 1

	addCond := func(cond string, val interface{}) {
		conds = append(conds, fmt.Sprintf(cond, argIdx))
		args = append(args, val)
		argIdx++
	}

	if filter.Status != "" {
		addCond("status = $%d", string(filter.Status))
	}
	if filter.PaymentMethod != "" {
		addCond("payment_method = $%d", filter.PaymentMethod)
	}
	if filter.CreatedFrom != nil {
		addCond("created_at >= $%d", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		addCond("created_at <= $%d", *filter.CreatedTo)
	}

	if len(conds) > 0 {
		query += " WHERE " + joinConds(conds)
	}
	query += " ORDER BY created_at DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []*model.Payment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return payments, nil
}

func (s *SQLStore) Health(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func scanPayment(row scanner) (*model.Payment, error) {
	var p model.Payment
	var status string
	var channel string
	var paymentURL sql.NullString
	var providerRef sql.NullString
	var failedReason sql.NullString
	var idempotencyKey sql.NullString

	err := row.Scan(
		&p.ID,
		&p.PaymentMethod,
		&p.Amount,
		&p.PhoneNumber,
		&status,
		&channel,
		&paymentURL,
		&providerRef,
		&failedReason,
		&idempotencyKey,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	p.Status = model.PaymentStatus(status)
	p.Channel = model.PaymentChannel(channel)
	p.PaymentURL = paymentURL.String
	p.ProviderRef = providerRef.String
	p.FailedReason = failedReason.String
	p.IdempotencyKey = idempotencyKey.String
	return &p, nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func joinConds(conds []string) string {
	if len(conds) == 0 {
		return ""
	}
	joined := conds[0]
	for i := 1; i < len(conds); i++ {
		joined += " AND " + conds[i]
	}
	return joined
}

func nullIfEmpty(val string) interface{} {
	if val == "" {
		return nil
	}
	return val
}
