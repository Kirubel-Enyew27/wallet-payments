package providers

import "context"

type InitiateRequest struct {
	TransactionID string
	PaymentMethod string
	Amount        string
	PhoneNumber   string
}

type InitiateResult struct {
	Channel     string
	PaymentURL  string
	ProviderRef string
}

type Provider interface {
	Name() string
	Initiate(ctx context.Context, req InitiateRequest) (InitiateResult, error)
}
