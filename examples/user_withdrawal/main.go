// Example: create an Integration user withdrawal.
// Freeze the end-user balance in YOUR ledger before calling this API.
//
//	export DOGGIEPAY_SECRET_KEY=dp_live_sk_...
//	export EXTERNAL_USER_ID=<customer ref_id UUID>
//	export TO_ADDRESS=0x...
//	go run ./examples/user_withdrawal
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"

	doggiepay "github.com/doggiepay/doggiepay-go-sdk"
)

func main() {
	secret := os.Getenv("DOGGIEPAY_SECRET_KEY")
	ext := os.Getenv("EXTERNAL_USER_ID")
	to := os.Getenv("TO_ADDRESS")
	if secret == "" || ext == "" || to == "" {
		log.Fatal("DOGGIEPAY_SECRET_KEY, EXTERNAL_USER_ID, TO_ADDRESS are required")
	}

	opts := []doggiepay.Option{}
	if base := os.Getenv("DOGGIEPAY_API_BASE"); base != "" {
		opts = append(opts, doggiepay.WithBaseURL(base))
	}
	client, err := doggiepay.New(secret, opts...)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	idem := uuid.NewString() // align with your local freeze / ledger id
	res, err := client.CreateUserWithdrawal(ctx, doggiepay.CreateUserWithdrawalParams{
		ExternalUserID: ext,
		Amount:         "10.00",
		Currency:       "USDC",
		Chain:          "ETH",
		ToAddress:      to,
	}, doggiepay.WithIdempotencyKey(idem))
	if err != nil {
		if apiErr, ok := doggiepay.AsError(err); ok {
			log.Fatalf("%s (%s) request_id=%s", apiErr.Code, apiErr.Message, apiErr.RequestID)
		}
		log.Fatal(err)
	}
	fmt.Printf("withdrawal_id=%s status=%s idempotency_key=%s\n", res.WithdrawalID, res.Status, idem)
}
