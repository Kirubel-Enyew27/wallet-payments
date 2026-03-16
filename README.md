# Wallet Payments Plugin (Gin)

Standalone payment integration service for wallet payments: Telebirr, M-Pesa, Yaya, Kacha, and Awash.

## Features
- Initiate a payment with `payment_method`, `amount`, and `phone_number`.
- Returns a response compatible with the request/response style in this project.
- Stores payment state in Postgres.
- Webhook-style callback and manual completion endpoints.
- Idempotency with `Idempotency-Key` header.
- Payments listing with filters and pagination.
- Health check endpoint.

## Run

```bash
cd wallet-payments
go mod tidy
go run .
```

## Environment Variables

- `BASE_URL` (default `http://localhost:8080`)
- `YAYA_BASE_URL` (optional, used to compose `payment_url`)
- `SIMULATE_ONLY` (default `true`)
- `DATABASE_URL` (default `postgres://postgres:postgres@localhost:5432/wallet_payments?sslmode=disable`)
- `DB_MAX_OPEN_CONNS` (default `25`)
- `DB_MAX_IDLE_CONNS` (default `10`)
- `DB_CONN_MAX_LIFETIME_MINUTES` (default `30`)

## API

### Initiate Payment

`POST /api/v1/payments`

Optional header:
`Idempotency-Key: <unique-key>`

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

### List Payments

`GET /api/v1/payments`

Query params:
- `status` (`pending`, `success`, `failed`)
- `method` (`TELEBIRR`, `M-PESA`, `YAYA`, `KACHA`, `AWASH`)
- `created_from` (RFC3339)
- `created_to` (RFC3339)
- `limit` (default `50`, max `200`)
- `offset` (default `0`)

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

### Health

`GET /api/v1/health`
