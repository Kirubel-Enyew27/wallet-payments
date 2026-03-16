package providers

import (
	"context"
	"fmt"
)

type TelebirrProvider struct{}

func (p TelebirrProvider) Name() string { return "TELEBIRR" }

func (p TelebirrProvider) Initiate(_ context.Context, req InitiateRequest) (InitiateResult, error) {
	return InitiateResult{
		Channel:     "USSD",
		ProviderRef: fmt.Sprintf("TB-%s", req.TransactionID),
	}, nil
}
