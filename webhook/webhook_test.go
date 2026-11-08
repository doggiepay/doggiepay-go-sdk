package webhook_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/doggiepay/doggiepay-go-sdk/webhook"
)

func TestConstructEvent_OK(t *testing.T) {
	secret := "whsec_test_secret"
	payload, _ := json.Marshal(map[string]any{
		"id":   "evt_1",
		"type": webhook.EventDepositCredited,
		"data": map[string]any{
			"deposit_id":       "dep_1",
			"amount":           "10.00",
			"external_user_id": "6ded84e7-5449-4a2b-9c1d-0afcce047b29",
		},
	})
	now := time.Unix(1_700_000_000, 0).UTC()
	header := webhook.Sign(secret, now, payload)

	evt, err := webhook.ConstructEventWithOptions(payload, header, secret, webhook.ConstructOptions{
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("ConstructEvent: %v", err)
	}
	if evt.Type != webhook.EventDepositCredited {
		t.Fatalf("type=%q", evt.Type)
	}
	var data webhook.DepositCreditedData
	if err := webhook.UnmarshalData(evt, &data); err != nil {
		t.Fatal(err)
	}
	if data.DepositID != "dep_1" || data.Amount != "10.00" {
		t.Fatalf("data=%+v", data)
	}
}

func TestVerifySignature_RejectsTamper(t *testing.T) {
	secret := "whsec_test_secret"
	payload := []byte(`{"id":"evt_1","type":"deposit.credited","data":{}}`)
	now := time.Now().UTC()
	header := webhook.Sign(secret, now, payload)
	tampered := append(append([]byte{}, payload...), ' ')
	err := webhook.VerifySignature(tampered, header, secret)
	if !errors.Is(err, webhook.ErrInvalidSignature) {
		t.Fatalf("got %v", err)
	}
}

func TestVerifySignature_RejectsSkew(t *testing.T) {
	secret := "whsec_test_secret"
	payload := []byte(`{"id":"evt_1","type":"deposit.credited","data":{}}`)
	old := time.Now().UTC().Add(-1 * time.Hour)
	header := webhook.Sign(secret, old, payload)
	err := webhook.VerifySignature(payload, header, secret)
	if !errors.Is(err, webhook.ErrTimestampSkew) {
		t.Fatalf("got %v", err)
	}
}

func TestVerifySignature_RejectsEmptySecret(t *testing.T) {
	err := webhook.VerifySignature([]byte(`{}`), "t=1,v1=ab", "")
	if !errors.Is(err, webhook.ErrEmptySecret) {
		t.Fatalf("got %v", err)
	}
}
