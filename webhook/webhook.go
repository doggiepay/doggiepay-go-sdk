// Package webhook verifies and parses DoggiePay merchant webhook deliveries.
//
// Signature header format:
//
//	X-DoggiePay-Signature: t=<unix>,v1=<hex>
//	v1 = HMAC-SHA256(webhook_secret, t + "." + rawBody)
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	HeaderSignature = "X-DoggiePay-Signature"

	// DefaultTolerance is the maximum allowed clock skew for signature timestamps.
	DefaultTolerance = 5 * time.Minute

	EventDepositCredited      = "deposit.credited"
	EventWithdrawalRequested  = "withdrawal.requested"
	EventWithdrawalSubmitted  = "withdrawal.submitted"
	EventWithdrawalComplete   = "withdrawal.complete"
	EventWithdrawalFailed     = "withdrawal.failed"
	EventWithdrawalPrepared   = "withdrawal.prepared" // portal prepare path only
)

var (
	ErrMissingHeader      = errors.New("doggiepay/webhook: missing signature header")
	ErrInvalidHeader      = errors.New("doggiepay/webhook: invalid signature header")
	ErrTimestampSkew      = errors.New("doggiepay/webhook: timestamp outside tolerance")
	ErrInvalidSignature   = errors.New("doggiepay/webhook: invalid signature")
	ErrEmptySecret        = errors.New("doggiepay/webhook: empty signing secret")
	ErrEmptyPayload       = errors.New("doggiepay/webhook: empty payload")
	ErrInvalidJSON        = errors.New("doggiepay/webhook: invalid JSON payload")
)

// Event is the merchant webhook envelope.
type Event struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// DepositCreditedData is the payload for deposit.credited.
type DepositCreditedData struct {
	DepositID      string `json:"deposit_id"`
	Amount         string `json:"amount"`
	Currency       string `json:"currency"`
	ExternalUserID string `json:"external_user_id"`
	Blockchain     string `json:"blockchain"`
	TxHash         string `json:"tx_hash"`
}

// WithdrawalEventData covers withdrawal.* Integration events.
type WithdrawalEventData struct {
	WithdrawalID   string `json:"withdrawal_id"`
	ExternalUserID string `json:"external_user_id"`
	Amount         string `json:"amount"`
	Currency       string `json:"currency"`
	Chain          string `json:"chain"`
	ToAddress      string `json:"to_address"`
	IdempotencyKey string `json:"idempotency_key"`
	TxHash         string `json:"tx_hash,omitempty"`
	FailureReason  string `json:"failure_reason,omitempty"`
}

// ConstructOptions configures ConstructEvent.
type ConstructOptions struct {
	// Tolerance is the allowed clock skew. Zero means DefaultTolerance unless IgnoreTolerance.
	Tolerance time.Duration
	// IgnoreTolerance skips timestamp checks (tests only — do not use in production).
	IgnoreTolerance bool
	// Now overrides the clock (tests).
	Now func() time.Time
}

// ConstructEvent verifies the signature and unmarshals the event.
// Always pass the raw request body bytes (do not re-serialize JSON).
func ConstructEvent(payload []byte, signatureHeader, secret string) (Event, error) {
	return ConstructEventWithOptions(payload, signatureHeader, secret, ConstructOptions{})
}

// ConstructEventWithOptions is ConstructEvent with explicit tolerance controls.
func ConstructEventWithOptions(payload []byte, signatureHeader, secret string, opts ConstructOptions) (Event, error) {
	var evt Event
	if err := VerifySignatureWithOptions(payload, signatureHeader, secret, opts); err != nil {
		return evt, err
	}
	if err := json.Unmarshal(payload, &evt); err != nil {
		return evt, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}
	if strings.TrimSpace(evt.ID) == "" || strings.TrimSpace(evt.Type) == "" {
		return evt, fmt.Errorf("%w: id and type required", ErrInvalidJSON)
	}
	return evt, nil
}

// VerifySignature validates X-DoggiePay-Signature with DefaultTolerance.
func VerifySignature(payload []byte, signatureHeader, secret string) error {
	return VerifySignatureWithOptions(payload, signatureHeader, secret, ConstructOptions{})
}

// VerifySignatureWithOptions validates the webhook HMAC signature.
func VerifySignatureWithOptions(payload []byte, signatureHeader, secret string, opts ConstructOptions) error {
	if strings.TrimSpace(secret) == "" {
		return ErrEmptySecret
	}
	if len(payload) == 0 {
		return ErrEmptyPayload
	}
	ts, sig, err := parseSignatureHeader(signatureHeader)
	if err != nil {
		return err
	}
	if !opts.IgnoreTolerance {
		tol := opts.Tolerance
		if tol == 0 {
			tol = DefaultTolerance
		}
		nowFn := opts.Now
		if nowFn == nil {
			nowFn = time.Now
		}
		unix, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			return ErrInvalidHeader
		}
		skew := nowFn().UTC().Unix() - unix
		if skew < 0 {
			skew = -skew
		}
		if skew > int64(tol/time.Second) {
			return ErrTimestampSkew
		}
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(ts + "."))
	_, _ = mac.Write(payload)
	want := hex.EncodeToString(mac.Sum(nil))
	got := strings.ToLower(sig)
	// Constant-time compare; prefer crypto/subtle over hmac.Equal for explicitness.
	if subtle.ConstantTimeCompare([]byte(want), []byte(got)) != 1 {
		return ErrInvalidSignature
	}
	return nil
}

// Sign builds an X-DoggiePay-Signature header for tests / local simulators.
func Sign(secret string, ts time.Time, payload []byte) string {
	t := strconv.FormatInt(ts.UTC().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(t + "."))
	_, _ = mac.Write(payload)
	return "t=" + t + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func parseSignatureHeader(header string) (ts, sig string, err error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", "", ErrMissingHeader
	}
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, "t="):
			ts = strings.TrimPrefix(part, "t=")
		case strings.HasPrefix(part, "v1="):
			sig = strings.TrimPrefix(part, "v1=")
		}
	}
	if ts == "" || sig == "" {
		return "", "", ErrInvalidHeader
	}
	return ts, sig, nil
}

// UnmarshalData decodes event.Data into dest.
func UnmarshalData(evt Event, dest any) error {
	if len(evt.Data) == 0 || string(evt.Data) == "null" {
		return fmt.Errorf("%w: empty data", ErrInvalidJSON)
	}
	return json.Unmarshal(evt.Data, dest)
}
