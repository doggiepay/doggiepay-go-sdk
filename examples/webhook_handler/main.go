// Example: verify DoggiePay merchant webhooks and credit/capture locally.
//
//	export DOGGIEPAY_WEBHOOK_SECRET=...
//	go run ./examples/webhook_handler
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/doggiepay/doggiepay-go-sdk/webhook"
)

func main() {
	secret := os.Getenv("DOGGIEPAY_WEBHOOK_SECRET")
	if secret == "" {
		log.Fatal("DOGGIEPAY_WEBHOOK_SECRET is required")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhooks/doggiepay", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// IMPORTANT: verify against the raw body; do not re-marshal JSON.
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		evt, err := webhook.ConstructEvent(raw, r.Header.Get(webhook.HeaderSignature), secret)
		if err != nil {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		switch evt.Type {
		case webhook.EventDepositCredited:
			var data webhook.DepositCreditedData
			if err := webhook.UnmarshalData(evt, &data); err != nil {
				http.Error(w, "bad data", http.StatusBadRequest)
				return
			}
			// TODO: idempotently credit your user ledger using data.DepositID / evt.ID
			log.Printf("deposit credited user=%s amount=%s tx=%s", data.ExternalUserID, data.Amount, data.TxHash)

		case webhook.EventWithdrawalComplete:
			var data webhook.WithdrawalEventData
			if err := webhook.UnmarshalData(evt, &data); err != nil {
				http.Error(w, "bad data", http.StatusBadRequest)
				return
			}
			// TODO: capture/finalize the local freeze keyed by IdempotencyKey or WithdrawalID
			log.Printf("withdrawal complete id=%s tx=%s", data.WithdrawalID, data.TxHash)

		case webhook.EventWithdrawalFailed:
			var data webhook.WithdrawalEventData
			if err := webhook.UnmarshalData(evt, &data); err != nil {
				http.Error(w, "bad data", http.StatusBadRequest)
				return
			}
			// TODO: unfreeze local user balance
			log.Printf("withdrawal failed id=%s reason=%s", data.WithdrawalID, data.FailureReason)

		case webhook.EventWithdrawalRequested, webhook.EventWithdrawalSubmitted:
			// Intermediate states; local freeze should already exist.
		default:
			log.Printf("ignored event type=%s id=%s", evt.Type, evt.ID)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	addr := ":8080"
	if v := os.Getenv("ADDR"); v != "" {
		addr = v
	}
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
