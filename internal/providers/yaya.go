package providers

import (
	"context"
	"fmt"
)

type YayaProvider struct {
	BaseURL string
}

func (p YayaProvider) Name() string { return "YAYA" }

func (p YayaProvider) Initiate(_ context.Context, req InitiateRequest) (InitiateResult, error) {
	paymentURL := ""
	if p.BaseURL != "" {
		paymentURL = fmt.Sprintf("%s/pay/%s", p.BaseURL, req.TransactionID)
	}
	return InitiateResult{
		Channel:     "WEB",
		PaymentURL:  paymentURL,
		ProviderRef: fmt.Sprintf("YA-%s", req.TransactionID),
	}, nil
}
