// Example: create a DoggiePay customer and print deposit addresses.
//
//	export DOGGIEPAY_SECRET_KEY=dp_live_sk_...
//	export DOGGIEPAY_API_BASE=https://api.doggiepay.com   # optional
//	go run ./examples/create_customer
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
	if secret == "" {
		log.Fatal("DOGGIEPAY_SECRET_KEY is required")
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

	// Use a stable UUID for your end-user. Persist it; it is external_user_id later.
	refID := uuid.NewString()
	requestID := uuid.NewString() // reuse on timeout retry with the same body

	cust, err := client.CreateCustomer(ctx, doggiepay.CreateCustomerParams{
		RefID: refID,
		Name:  "Jane Doe",
		Email: "jane@example.com",
	}, doggiepay.WithRequestID(requestID))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("customer_id=%s ref_id=%s\n", cust.ID, cust.RefID)
	for _, w := range doggiepay.FoldCustomerWallets(cust.Wallets) {
		fmt.Printf("  %s %s (%s)\n", w.Blockchain, w.Address, w.Status)
	}
}
