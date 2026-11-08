package doggiepay_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	doggiepay "github.com/doggiepay/doggiepay-go-sdk"
)

func TestNew_RejectsPublishableKey(t *testing.T) {
	_, err := doggiepay.New("dp_live_pk_abcdefghijklmnopqrstuvwxyz")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, doggiepay.ErrInvalidSecretKey) && !strings.Contains(err.Error(), "publishable") {
		t.Fatalf("expected invalid secret, got %v", err)
	}
}

func TestNew_RejectsHTTPWithoutAllow(t *testing.T) {
	_, err := doggiepay.New("dp_live_sk_abcdefghijklmnopqrstuvwxyz", doggiepay.WithBaseURL("http://localhost:8080"))
	if !errors.Is(err, doggiepay.ErrInsecureBaseURL) {
		t.Fatalf("got %v", err)
	}
}

func TestNew_AllowsHTTPWhenOptedIn(t *testing.T) {
	c, err := doggiepay.New(
		"dp_live_sk_abcdefghijklmnopqrstuvwxyz",
		doggiepay.WithBaseURL("http://127.0.0.1:8080"),
		doggiepay.WithAllowInsecureHTTP(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(c.String(), "dp_live_sk_a") || strings.Contains(c.String(), "uvwxyz") {
		t.Fatalf("secret not redacted: %s", c.String())
	}
}

func TestCreateCustomer_RoundTrip(t *testing.T) {
	var sawAuth, sawReqID, sawUA string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		sawReqID = r.Header.Get("X-DoggiePay-Request-Id")
		sawUA = r.Header.Get("User-Agent")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-DoggiePay-Request-Id", sawReqID)
		_, _ = w.Write([]byte(`{"data":{"id":"8e384561-d781-4868-9b66-3fb2d7a83ba5","ref_id":"6ded84e7-5449-4a2b-9c1d-0afcce047b29","name":"John","email":"j@example.com","wallets":[{"blockchain":"ETH","currency":"USDC","address":"0xabc","status":"active"}]},"error":null}`))
	}))
	defer srv.Close()

	client, err := doggiepay.New(
		"dp_live_sk_abcdefghijklmnopqrstuvwxyz",
		doggiepay.WithBaseURL(srv.URL),
		doggiepay.WithAllowInsecureHTTP(),
		doggiepay.WithUserAgent("coinance365/1.0"),
	)
	if err != nil {
		t.Fatal(err)
	}
	reqID := "550e8400-e29b-41d4-a716-446655440000"
	cust, err := client.CreateCustomer(context.Background(), doggiepay.CreateCustomerParams{
		RefID:       "6ded84e7-5449-4a2b-9c1d-0afcce047b29",
		Name:        "John",
		Email:       "j@example.com",
		Blockchains: []string{"ethereum"},
	}, doggiepay.WithRequestID(reqID))
	if err != nil {
		t.Fatal(err)
	}
	if cust.ID == "" || len(cust.Wallets) != 1 {
		t.Fatalf("customer=%+v", cust)
	}
	if sawAuth != "Bearer dp_live_sk_abcdefghijklmnopqrstuvwxyz" {
		t.Fatalf("auth=%q", sawAuth)
	}
	if sawReqID != reqID {
		t.Fatalf("request id=%q", sawReqID)
	}
	if !strings.Contains(sawUA, "doggiepay-go-sdk/") || !strings.Contains(sawUA, "coinance365/1.0") {
		t.Fatalf("ua=%q", sawUA)
	}
	if body["blockchains"].([]any)[0] != "ETH" {
		t.Fatalf("blockchains not normalized: %#v", body["blockchains"])
	}
}

func TestAPIError_Envelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"data":null,"error":{"code":"rate_limited","message":"slow down"}}`))
	}))
	defer srv.Close()

	client, err := doggiepay.New(
		"dp_live_sk_abcdefghijklmnopqrstuvwxyz",
		doggiepay.WithBaseURL(srv.URL),
		doggiepay.WithAllowInsecureHTTP(),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetCustomerByRefID(context.Background(), "6ded84e7-5449-4a2b-9c1d-0afcce047b29")
	apiErr, ok := doggiepay.AsError(err)
	if !ok {
		t.Fatalf("expected API error, got %v", err)
	}
	if apiErr.Code != "rate_limited" || !apiErr.Temporary() {
		t.Fatalf("%+v", apiErr)
	}
	if apiErr.RetryAfter.Seconds() != 60 {
		t.Fatalf("retry after=%v", apiErr.RetryAfter)
	}
}

func TestCreateUserWithdrawal_RequiresIdFields(t *testing.T) {
	client, err := doggiepay.New(
		"dp_live_sk_abcdefghijklmnopqrstuvwxyz",
		doggiepay.WithBaseURL("https://api.example.test"),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CreateUserWithdrawal(context.Background(), doggiepay.CreateUserWithdrawalParams{
		ExternalUserID: "not-a-uuid",
		Amount:         "1",
		Currency:       "USDC",
		Chain:          "ETH",
		ToAddress:      "0x1",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestNormalizeChain(t *testing.T) {
	if got := doggiepay.NormalizeChain("ethereum"); got != "ETH" {
		t.Fatalf("got %q", got)
	}
	folded := doggiepay.FoldCustomerWallets([]doggiepay.CustomerWallet{
		{Blockchain: "ETH", Currency: "USDC", Address: "0xabc"},
		{Blockchain: "ETH", Currency: "USDT", Address: "0xabc"},
		{Blockchain: "BASE", Currency: "USDC", Address: "0xdef"},
	})
	if len(folded) != 2 {
		t.Fatalf("len=%d", len(folded))
	}
}

func TestWithRequestID_MustBeUUID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call server")
	}))
	defer srv.Close()
	client, err := doggiepay.New(
		"dp_live_sk_abcdefghijklmnopqrstuvwxyz",
		doggiepay.WithBaseURL(srv.URL),
		doggiepay.WithAllowInsecureHTTP(),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListWallets(context.Background(), "6ded84e7-5449-4a2b-9c1d-0afcce047b29", doggiepay.WithRequestID("not-uuid"))
	if !errors.Is(err, doggiepay.ErrInvalidRequestID) {
		t.Fatalf("got %v", err)
	}
}
