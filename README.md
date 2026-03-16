# Wallet Payments Plugin (Gin)

Standalone payment integration service for wallet payments: Telebirr, M-Pesa, Yaya, Kacha, and Awash.

## Features
- Initiate a payment with `payment_method`, `amount`, and `phone_number`.
- Returns a response compatible with the request/response style in this project.
- Stores payment state in-memory.
- Webhook-style callback and manual completion endpoints.

## Run

```bash
cd /home/k-i-r-a/Projects/cash-flow/wallet-payments-plugin
go mod tidy
go run .
```

## Environment Variables

- `BASE_URL` (default `http://localhost:8080`)
- `YAYA_BASE_URL` (optional, used to compose `payment_url`)
- `SIMULATE_ONLY` (default `true`)

## API

### Initiate Payment

`POST /api/v1/payments`

Request:
```json
{
  "payment_method": "TELEBIRR",
  "amount": "150.00",
  "phone_number": "251912345678"
}
```

Response (USSD wallets):
```json
{
  "status": "success",
  "message": "Payment initiated. Please complete the USSD prompt.",
  "data": {
    "status": "pending",
    "payment_channel": "USSD",
    "payment_url": "",
    "redirect_url": "http://localhost:8080/api/v1/payments/TRANSACTION_ID"
  }
}
```

Response (Yaya):
```json
{
  "status": "success",
  "message": "Payment link generated successfully, please complete your payment.",
  "data": {
    "status": "pending",
    "payment_channel": "WEB",
    "payment_url": "http://localhost:8080/pay/TRANSACTION_ID",
    "redirect_url": "http://localhost:8080/api/v1/payments/TRANSACTION_ID"
  }
}
```

### Get Payment Status

`GET /api/v1/payments/:id`

### Complete Payment Manually (for testing)

`POST /api/v1/payments/:id/complete`

Request:
```json
{
  "status": "success"
}
```

### Provider Callback

`POST /api/v1/callbacks/:provider`

Request:
```json
{
  "transaction_id": "TRANSACTION_ID",
  "status": "success"
}
```

Supported `provider` values: `telebirr`, `mpesa`, `yaya`, `kacha`, `awash`.
