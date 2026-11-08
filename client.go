package doggiepay

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the production Integration API origin.
const DefaultBaseURL = "https://api.doggiepay.com"

const (
	headerAuthorization = "Authorization"
	headerRequestID     = "X-DoggiePay-Request-Id"
	headerIdempotency   = "Idempotency-Key"
	headerUserAgent     = "User-Agent"
	headerAccept        = "Accept"
	headerContentType   = "Content-Type"
	headerRetryAfter    = "Retry-After"

	defaultHTTPTimeout   = 60 * time.Second
	defaultMaxBodyBytes  = 4 << 20 // 4 MiB
	secretKeyLivePrefix  = "dp_live_sk_"
	publishableKeyPrefix = "dp_live_pk_"
)

// Client is a thread-safe Integration Open API client.
//
// Secret keys must only live on your server. Never embed them in mobile apps
// or browser code. Prefer loading them from a secret manager / env var.
type Client struct {
	baseURL           string
	secretKey         string
	httpClient        *http.Client
	userAgent         string
	maxBodyBytes      int64
	allowInsecureHTTP bool
}

// Option configures a Client.
type Option func(*Client) error

// WithBaseURL overrides the API origin (no trailing slash). Must be https unless
// WithAllowInsecureHTTP is also set.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) error {
		c.baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		return nil
	}
}

// WithHTTPClient injects a custom *http.Client (timeouts, mTLS, tracing, etc.).
// The SDK never mutates http.DefaultClient.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		if hc == nil {
			return fmt.Errorf("doggiepay: http client must not be nil")
		}
		c.httpClient = hc
		return nil
	}
}

// WithUserAgent appends a product identifier to the default User-Agent.
func WithUserAgent(ua string) Option {
	return func(c *Client) error {
		ua = strings.TrimSpace(ua)
		if ua == "" {
			return nil
		}
		c.userAgent = defaultUserAgent + " " + ua
		return nil
	}
}

// WithMaxBodyBytes caps decoded response bodies (default 4 MiB).
func WithMaxBodyBytes(n int64) Option {
	return func(c *Client) error {
		if n <= 0 {
			return fmt.Errorf("doggiepay: max body bytes must be positive")
		}
		c.maxBodyBytes = n
		return nil
	}
}

// WithAllowInsecureHTTP permits http:// base URLs for local development only.
func WithAllowInsecureHTTP() Option {
	return func(c *Client) error {
		c.allowInsecureHTTP = true
		return nil
	}
}

// New constructs a Client. secretKey must be an Integration Secret Key
// (dp_live_sk_…). Publishable keys and Portal JWTs are rejected.
func New(secretKey string, opts ...Option) (*Client, error) {
	key := strings.TrimSpace(secretKey)
	if key == "" {
		return nil, ErrInvalidSecretKey
	}
	if strings.HasPrefix(key, publishableKeyPrefix) || strings.Contains(key, "_pk_") {
		return nil, fmt.Errorf("%w: publishable keys are not allowed for Integration API", ErrInvalidSecretKey)
	}
	if !strings.HasPrefix(key, secretKeyLivePrefix) && !strings.HasPrefix(key, "dp_test_sk_") {
		return nil, ErrInvalidSecretKey
	}
	if len(key) < len(secretKeyLivePrefix)+8 {
		return nil, ErrInvalidSecretKey
	}

	c := &Client{
		baseURL:      DefaultBaseURL,
		secretKey:    key,
		userAgent:    defaultUserAgent,
		maxBodyBytes: defaultMaxBodyBytes,
		httpClient:   &http.Client{Timeout: defaultHTTPTimeout},
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	if err := c.validateBaseURL(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) validateBaseURL() error {
	if c.baseURL == "" {
		return fmt.Errorf("doggiepay: base URL is required")
	}
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("doggiepay: invalid base URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("doggiepay: invalid base URL: missing scheme or host")
	}
	if u.Scheme != "https" && !c.allowInsecureHTTP {
		return ErrInsecureBaseURL
	}
	if u.User != nil {
		return fmt.Errorf("doggiepay: base URL must not embed credentials")
	}
	return nil
}

// BaseURL returns the configured API origin.
func (c *Client) BaseURL() string {
	if c == nil {
		return ""
	}
	return c.baseURL
}

// String redacts the secret key.
func (c *Client) String() string {
	if c == nil {
		return "doggiepay.Client<nil>"
	}
	return fmt.Sprintf("doggiepay.Client{baseURL:%q, secretKey:%q}", c.baseURL, redactSecret(c.secretKey))
}

func redactSecret(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 12 {
		return "***"
	}
	return key[:12] + "…"
}
