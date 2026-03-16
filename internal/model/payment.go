package model

import "time"

type PaymentStatus string

type PaymentChannel string

const (
	StatusPending PaymentStatus = "pending"
	StatusSuccess PaymentStatus = "success"
	StatusFailed  PaymentStatus = "failed"

	ChannelUSSD PaymentChannel = "USSD"
	ChannelWeb  PaymentChannel = "WEB"
)

type Payment struct {
	ID            string
	PaymentMethod string
	Amount        string
	PhoneNumber   string
	Status        PaymentStatus
	Channel       PaymentChannel
	PaymentURL    string
	ProviderRef   string
	FailedReason  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
