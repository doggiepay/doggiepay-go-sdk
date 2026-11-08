package doggiepay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type callConfig struct {
	requestID      string
	idempotencyKey string
}

// CallOption customizes a single API call (idempotency / request id reuse).
type CallOption func(*callConfig)

// WithRequestID sets X-DoggiePay-Request-Id. Reuse the same UUID + identical
// body when retrying timed-out writes (CreateCustomer, PatchCustomer, …).
func WithRequestID(id string) CallOption {
	return func(c *callConfig) {
		c.requestID = strings.TrimSpace(id)
	}
}

// WithIdempotencyKey sets Idempotency-Key (strongly recommended for withdrawals).
// If the key is a UUID, the API may use it as withdrawal_id.
func WithIdempotencyKey(key string) CallOption {
	return func(c *callConfig) {
		c.idempotencyKey = strings.TrimSpace(key)
	}
}

func applyCallOptions(opts []CallOption) (callConfig, error) {
	var cfg callConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.requestID == "" {
		cfg.requestID = uuid.NewString()
	} else if _, err := uuid.Parse(cfg.requestID); err != nil {
		return cfg, ErrInvalidRequestID
	}
	return cfg, nil
}

type apiEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) do(ctx context.Context, method, path string, body any, dest any, opts ...CallOption) error {
	if c == nil || c.secretKey == "" || c.baseURL == "" {
		return ErrNotConfigured
	}
	if ctx == nil {
		return fmt.Errorf("doggiepay: context must not be nil")
	}
	cfg, err := applyCallOptions(opts)
	if err != nil {
		return err
	}

	var bodyReader io.Reader
	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("doggiepay: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set(headerAuthorization, "Bearer "+c.secretKey)
	req.Header.Set(headerAccept, "application/json")
	req.Header.Set(headerRequestID, cfg.requestID)
	req.Header.Set(headerUserAgent, c.userAgent)
	if body != nil {
		req.Header.Set(headerContentType, "application/json")
	}
	if cfg.idempotencyKey != "" {
		req.Header.Set(headerIdempotency, cfg.idempotencyKey)
	}

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, c.maxBodyBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	if int64(len(raw)) > c.maxBodyBytes {
		return ErrResponseTooLarge
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidJSON, truncate(string(raw), 200))
	}

	requestID := resp.Header.Get(headerRequestID)
	if requestID == "" {
		requestID = cfg.requestID
	}

	if envelope.Error != nil && envelope.Error.Code != "" {
		return &Error{
			Status:     resp.StatusCode,
			Code:       envelope.Error.Code,
			Message:    envelope.Error.Message,
			RequestID:  requestID,
			RetryAfter: parseRetryAfter(resp.Header.Get(headerRetryAfter)),
		}
	}
	if resp.StatusCode >= 400 {
		return &Error{
			Status:     resp.StatusCode,
			Code:       "http_error",
			Message:    truncate(string(raw), 200),
			RequestID:  requestID,
			RetryAfter: parseRetryAfter(resp.Header.Get(headerRetryAfter)),
		}
	}
	if dest == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, dest); err != nil {
		return fmt.Errorf("doggiepay: decode data: %w", err)
	}
	return nil
}

func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
		return time.Duration(sec) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
