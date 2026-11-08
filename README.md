# doggiepay-go-sdk

Official **Go SDK** for the [DoggiePay](https://doggiepay.com) **Integration Open API**.

Use it from your backend to:

- create **customers** and long-lived **deposit addresses**
- submit **user withdrawals** from your merchant crypto pool
- **verify merchant webhooks** (`deposit.credited`, `withdrawal.*`)

> This is a **server-side** SDK. Secret Keys must never ship in apps or browsers.

## Install

```bash
go get github.com/doggiepay/doggiepay-go-sdk@latest
```

Requires Go 1.22+.

## Quick start

```go
package main

import (
	"context"
	"log"
	"os"

	doggiepay "github.com/doggiepay/doggiepay-go-sdk"
)

func main() {
	client, err := doggiepay.New(os.Getenv("DOGGIEPAY_SECRET_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	cust, err := client.CreateCustomer(context.Background(), doggiepay.CreateCustomerParams{
		RefID: "6ded84e7-5449-4a2b-9c1d-0afcce047b29", // your user UUID
		Name:  "Jane Doe",
		Email: "jane@example.com",
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, w := range doggiepay.FoldCustomerWallets(cust.Wallets) {
		log.Printf("%s → %s", w.Blockchain, w.Address)
	}
}
```

### Options

```go
client, err := doggiepay.New(
	os.Getenv("DOGGIEPAY_SECRET_KEY"),
	doggiepay.WithBaseURL("https://api.doggiepay.com"), // default
	doggiepay.WithHTTPClient(myHTTPClient),             // timeouts / mTLS / tracing
	doggiepay.WithUserAgent("my-product/1.2.3"),
)
```

`http://` base URLs are rejected unless you pass `WithAllowInsecureHTTP()` (local testing only).

## API surface

| Method | Open API |
|--------|----------|
| `CreateCustomer` | `POST /v1/customers` |
| `GetCustomer` | `GET /v1/customers/{id}` |
| `GetCustomerByRefID` | `GET /v1/customers?ref_id=` |
| `ListCustomers` | `GET /v1/customers?page=&per_page=` |
| `UpdateCustomer` | `PATCH /v1/customers/{id}` |
| `EnsureWallets` | `POST /v1/wallets/ensure` |
| `ListWallets` | `GET /v1/wallets/{external_user_id}` |
| `CreateUserWithdrawal` | `POST /v1/user-withdrawals` |
| `GetUserWithdrawal` | `GET /v1/user-withdrawals/{id}` |
| `ListUserWithdrawals` | `GET /v1/user-withdrawals` |
| `webhook.ConstructEvent` | verify `X-DoggiePay-Signature` |

Auth on every call:

```http
Authorization: Bearer dp_live_sk_...
X-DoggiePay-Request-Id: <uuid>
```

### Idempotent retries

On write timeouts, **reuse** the same request id and body:

```go
reqID := "550e8400-e29b-41d4-a716-446655440000"
cust, err := client.CreateCustomer(ctx, params, doggiepay.WithRequestID(reqID))
```

For withdrawals, also pass `WithIdempotencyKey` (preferably a UUID aligned with your local freeze id).

### Errors

```go
cust, err := client.CreateCustomer(ctx, params)
if err != nil {
	if apiErr, ok := doggiepay.AsError(err); ok {
		log.Printf("code=%s status=%d request_id=%s", apiErr.Code, apiErr.Status, apiErr.RequestID)
		if apiErr.Temporary() {
			// backoff using apiErr.RetryAfter when set
		}
	}
}
```

## Webhooks

```go
import "github.com/doggiepay/doggiepay-go-sdk/webhook"

raw, _ := io.ReadAll(r.Body) // raw bytes only
evt, err := webhook.ConstructEvent(raw, r.Header.Get(webhook.HeaderSignature), webhookSecret)
if err != nil {
	http.Error(w, "unauthorized", http.StatusUnauthorized)
	return
}

switch evt.Type {
case webhook.EventDepositCredited:
	var data webhook.DepositCreditedData
	_ = webhook.UnmarshalData(evt, &data)
	// credit your user ledger (idempotent on data.DepositID / evt.ID)
case webhook.EventWithdrawalComplete:
	// capture local freeze
case webhook.EventWithdrawalFailed:
	// unfreeze local balance
}
```

Signature scheme: `v1 = HMAC-SHA256(secret, t + "." + rawBody)` with default ±5 minute skew.

## Money flow (your responsibilities)

```text
CreateCustomer → show deposit addresses
user pays on-chain → webhook deposit.credited → YOU credit user balance
user withdraws → YOU freeze balance → CreateUserWithdrawal
  → withdrawal.complete → capture freeze
  → withdrawal.failed   → unfreeze
```

DoggiePay maintains the **merchant crypto pool**, not per-user balances.

## Examples

```bash
export DOGGIEPAY_SECRET_KEY=dp_live_sk_...
go run ./examples/create_customer

export DOGGIEPAY_WEBHOOK_SECRET=...
go run ./examples/webhook_handler
```

## Security

See [SECURITY.md](./SECURITY.md). Highlights:

- Rejects publishable keys (`dp_live_pk_*`) and JWT-shaped credentials
- HTTPS-only base URL by default
- Secret redacted in `Client.String()`
- Constant-time webhook HMAC compare + timestamp tolerance
- Response body size limit (default 4 MiB)

## Compatibility

Extracted from production Integration usage (customers / wallets / user-withdrawals / merchant webhooks) and aligned with the public Integration guide.

Module path: `github.com/doggiepay/doggiepay-go-sdk`  
License: MIT
