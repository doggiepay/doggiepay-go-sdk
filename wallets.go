package doggiepay

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

// DepositWallet is a long-lived per-chain deposit address.
type DepositWallet struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	ExternalUserID string `json:"external_user_id"`
	Blockchain     string `json:"blockchain"`
	Address        string `json:"address"`
	CircleWalletID string `json:"circle_wallet_id"`
	Status         string `json:"status"`
}

// EnsureWalletsParams is the body for POST /v1/wallets/ensure.
type EnsureWalletsParams struct {
	// ExternalUserID must equal the customer's ref_id (UUID).
	ExternalUserID string   `json:"external_user_id"`
	Blockchains    []string `json:"blockchains,omitempty"`
}

// EnsureWallets creates missing long-lived deposit addresses for a customer.
func (c *Client) EnsureWallets(ctx context.Context, params EnsureWalletsParams, opts ...CallOption) ([]DepositWallet, error) {
	ext := strings.TrimSpace(params.ExternalUserID)
	if _, err := uuid.Parse(ext); err != nil {
		return nil, fmt.Errorf("doggiepay: external_user_id must be a UUID (customer ref_id)")
	}
	body := EnsureWalletsParams{
		ExternalUserID: ext,
		Blockchains:    NormalizeChains(params.Blockchains),
	}
	var out []DepositWallet
	if err := c.do(ctx, http.MethodPost, "/v1/wallets/ensure", body, &out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

// ListWallets returns existing deposit wallets for external_user_id (= ref_id).
func (c *Client) ListWallets(ctx context.Context, externalUserID string, opts ...CallOption) ([]DepositWallet, error) {
	ext := strings.TrimSpace(externalUserID)
	if _, err := uuid.Parse(ext); err != nil {
		return nil, fmt.Errorf("doggiepay: external_user_id must be a UUID (customer ref_id)")
	}
	path := "/v1/wallets/" + url.PathEscape(ext)
	var out []DepositWallet
	if err := c.do(ctx, http.MethodGet, path, nil, &out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

// FoldCustomerWallets collapses currency-expanded wallets to one row per blockchain
// (address is shared per chain). Useful for UIs that show one address per network.
func FoldCustomerWallets(wallets []CustomerWallet) []DepositWallet {
	seen := map[string]struct{}{}
	out := make([]DepositWallet, 0, len(wallets))
	for _, w := range wallets {
		chain := strings.TrimSpace(w.Blockchain)
		if chain == "" {
			continue
		}
		key := strings.ToUpper(chain)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, DepositWallet{
			Blockchain: chain,
			Address:    w.Address,
			Status:     w.Status,
		})
	}
	return out
}
