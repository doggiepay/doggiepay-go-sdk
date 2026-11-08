package doggiepay

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Sentinel configuration / client-side errors.
var (
	ErrNotConfigured     = errors.New("doggiepay: client not configured")
	ErrInvalidSecretKey  = errors.New("doggiepay: secret key must be a dp_live_sk_* Integration secret")
	ErrInsecureBaseURL   = errors.New("doggiepay: base URL must use https (use WithAllowInsecureHTTP for local testing only)")
	ErrEmptyPayload      = errors.New("doggiepay: empty payload")
	ErrInvalidRequestID  = errors.New("doggiepay: request id must be a UUID")
	ErrResponseTooLarge  = errors.New("doggiepay: response body exceeds size limit")
	ErrInvalidJSON       = errors.New("doggiepay: invalid JSON response")
)

// Error is an API envelope failure ({"error":{"code","message"}}).
type Error struct {
	// Status is the HTTP status code.
	Status int
	// Code is the machine-readable DoggiePay error code (e.g. insufficient_balance).
	Code string
	// Message is the human-readable error message from the API.
	Message string
	// RequestID is the X-DoggiePay-Request-Id used for the call (echoed by the API).
	RequestID string
	// RetryAfter is set when the API returns Retry-After (e.g. rate_limited).
	RetryAfter time.Duration
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != "" {
		if e.RequestID != "" {
			return fmt.Sprintf("doggiepay: %s: %s (request_id=%s)", e.Code, e.Message, e.RequestID)
		}
		return fmt.Sprintf("doggiepay: %s: %s", e.Code, e.Message)
	}
	if e.Message != "" {
		return "doggiepay: " + e.Message
	}
	return fmt.Sprintf("doggiepay: http %d", e.Status)
}

// Temporary reports whether the caller may reasonably retry the request.
func (e *Error) Temporary() bool {
	if e == nil {
		return false
	}
	switch e.Status {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	switch e.Code {
	case "rate_limited", "circle_error", "circle_not_configured":
		return true
	}
	return false
}

// IsConflict is true for idempotency / uniqueness conflicts.
func (e *Error) IsConflict() bool {
	if e == nil {
		return false
	}
	return e.Status == http.StatusConflict ||
		e.Code == "request_id_conflict" ||
		e.Code == "customer_ref_id_conflict"
}

// IsNotFound is true when the resource does not exist.
func (e *Error) IsNotFound() bool {
	if e == nil {
		return false
	}
	return e.Status == http.StatusNotFound ||
		e.Code == "customer_not_found" ||
		e.Code == "withdrawal_not_found"
}

// AsError extracts *Error from err, if present.
func AsError(err error) (*Error, bool) {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}
