package providers

import (
	"context"
	"fmt"
)

type MpesaProvider struct{}

func (p MpesaProvider) Name() string { return "M-PESA" }

func (p MpesaProvider) Initiate(_ context.Context, req InitiateRequest) (InitiateResult, error) {
	return InitiateResult{
		Channel:     "USSD",
		ProviderRef: fmt.Sprintf("MP-%s", req.TransactionID),
	}, nil
}
