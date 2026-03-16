package providers

import (
	"context"
	"fmt"
)

type KachaProvider struct{}

func (p KachaProvider) Name() string { return "KACHA" }

func (p KachaProvider) Initiate(_ context.Context, req InitiateRequest) (InitiateResult, error) {
	return InitiateResult{
		Channel:     "USSD",
		ProviderRef: fmt.Sprintf("KA-%s", req.TransactionID),
	}, nil
}
