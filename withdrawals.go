package doggiepay

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// CreateUserWithdrawalParams is the body for POST /v1/user-withdrawals.
// Freeze the end-user balance in your ledger before calling this API.
type CreateUserWithdrawalParams struct {
	// ExternalUserID must equal the customer's ref_id (UUID).
	ExternalUserID string `json:"external_user_id"`
	Amount         string `json:"amount"`
	Currency       string `json:"currency"`
	Chain          string `json:"chain"`
	ToAddress      string `json:"to_address"`
	AccountType    string `json:"account_type,omitempty"`
}

// UserWithdrawalResult is returned by CreateUserWithdrawal.
type UserWithdrawalResult struct {
	WithdrawalID string `json:"withdrawal_id"`
	Status       string `json:"status"`
}

// UserWithdrawal is a withdrawal detail / list item.
type UserWithdrawal struct {
	WithdrawalID   string `json:"withdrawal_id"`
	Status         string `json:"status"`
	ExternalUserID string `json:"external_user_id"`
	Amount         string `json:"amount"`
	Currency       string `json:"currency"`
	Chain          string `json:"chain,omitempty"`
	ToAddress      string `json:"to_address,omitempty"`
	AccountType    string `json:"account_type,omitempty"`
	NotifyEmail    string `json:"notify_email,omitempty"`
	TxHash         string `json:"tx_hash,omitempty"`
	FailureReason  string `json:"failure_reason,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
}

// ListUserWithdrawalsParams filters GET /v1/user-withdrawals.
type ListUserWithdrawalsParams struct {
	ExternalUserID string
	Status         string
	Page           int
	Limit          int
	Offset         int
}

// UserWithdrawalList is a withdrawal list response.
type UserWithdrawalList struct {
	Withdrawals []UserWithdrawal `json:"withdrawals"`
	Total       int              `json:"total,omitempty"`
	Limit       int              `json:"limit,omitempty"`
	Offset      int              `json:"offset,omitempty"`
	Page        int              `json:"page,omitempty"`
}

// CreateUserWithdrawal debits the merchant crypto pool and queues an on-chain payout.
// Pass WithIdempotencyKey (preferably a UUID aligned with your local freeze id).
func (c *Client) CreateUserWithdrawal(ctx context.Context, params CreateUserWithdrawalParams, opts ...CallOption) (*UserWithdrawalResult, error) {
	if err := validateWithdrawalParams(params); err != nil {
		return nil, err
	}
	params.Chain = NormalizeChain(params.Chain)
	params.Currency = strings.ToUpper(strings.TrimSpace(params.Currency))
	var out UserWithdrawalResult
	if err := c.do(ctx, http.MethodPost, "/v1/user-withdrawals", params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUserWithdrawal fetches a withdrawal by id.
func (c *Client) GetUserWithdrawal(ctx context.Context, withdrawalID string, opts ...CallOption) (*UserWithdrawal, error) {
	id := strings.TrimSpace(withdrawalID)
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("doggiepay: withdrawal id must be a UUID")
	}
	path := "/v1/user-withdrawals/" + url.PathEscape(id)
	var out UserWithdrawal
	if err := c.do(ctx, http.MethodGet, path, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListUserWithdrawals lists withdrawals, optionally filtered by external_user_id / status.
func (c *Client) ListUserWithdrawals(ctx context.Context, params ListUserWithdrawalsParams, opts ...CallOption) (*UserWithdrawalList, error) {
	q := url.Values{}
	if ext := strings.TrimSpace(params.ExternalUserID); ext != "" {
		if _, err := uuid.Parse(ext); err != nil {
			return nil, fmt.Errorf("doggiepay: external_user_id must be a UUID")
		}
		q.Set("external_user_id", ext)
	}
	if st := strings.TrimSpace(params.Status); st != "" {
		q.Set("status", st)
	}
	if params.Page > 0 {
		q.Set("page", strconv.Itoa(params.Page))
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		q.Set("offset", strconv.Itoa(params.Offset))
	}
	path := "/v1/user-withdrawals"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	var out UserWithdrawalList
	if err := c.do(ctx, http.MethodGet, path, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

func validateWithdrawalParams(p CreateUserWithdrawalParams) error {
	ext := strings.TrimSpace(p.ExternalUserID)
	if _, err := uuid.Parse(ext); err != nil {
		return fmt.Errorf("doggiepay: external_user_id must be a UUID (customer ref_id)")
	}
	if strings.TrimSpace(p.Amount) == "" {
		return fmt.Errorf("doggiepay: amount is required")
	}
	if strings.TrimSpace(p.Currency) == "" {
		return fmt.Errorf("doggiepay: currency is required")
	}
	if strings.TrimSpace(p.Chain) == "" {
		return fmt.Errorf("doggiepay: chain is required")
	}
	if strings.TrimSpace(p.ToAddress) == "" {
		return fmt.Errorf("doggiepay: to_address is required")
	}
	return nil
}
