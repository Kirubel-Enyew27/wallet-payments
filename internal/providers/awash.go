package providers

import (
	"context"
	"fmt"
)

type AwashProvider struct{}

func (p AwashProvider) Name() string { return "AWASH" }

func (p AwashProvider) Initiate(_ context.Context, req InitiateRequest) (InitiateResult, error) {
	return InitiateResult{
		Channel:     "USSD",
		ProviderRef: fmt.Sprintf("AW-%s", req.TransactionID),
	}, nil
}
