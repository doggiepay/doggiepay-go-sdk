package doggiepay

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

// CustomerWallet is a currency-expanded deposit address from Create/Get Customer.
// Same-chain currencies share one on-chain address.
type CustomerWallet struct {
	Blockchain string `json:"blockchain"`
	Currency   string `json:"currency"`
	Address    string `json:"address"`
	Status     string `json:"status"`
}

// Customer is the Integration customer view.
type Customer struct {
	ID          string           `json:"id"`
	RefID       string           `json:"ref_id"`
	Name        string           `json:"name"`
	Email       string           `json:"email"`
	Description string           `json:"description,omitempty"`
	Wallets     []CustomerWallet `json:"wallets,omitempty"`
	CreatedAt   string           `json:"created_at,omitempty"`
	UpdatedAt   string           `json:"updated_at,omitempty"`
}

// CreateCustomerParams is the body for POST /v1/customers.
type CreateCustomerParams struct {
	// RefID is your stable user id and MUST be a UUID. It becomes external_user_id
	// for wallets and withdrawals. Immutable after create.
	RefID string `json:"ref_id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	// Description is an optional note.
	Description string `json:"description,omitempty"`
	// Blockchains limits which chains to provision; omit for all platform-enabled chains.
	Blockchains []string `json:"blockchains,omitempty"`
}

// UpdateCustomerParams is the body for PATCH /v1/customers/{id}.
// RefID cannot be changed.
type UpdateCustomerParams struct {
	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	Description string `json:"description,omitempty"`
}

// CustomerList is a paginated customer list (items typically omit full wallets).
type CustomerList struct {
	Items []Customer      `json:"items"`
	Meta  PaginationMeta  `json:"meta"`
}

// PaginationMeta describes list pagination.
type PaginationMeta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

// ListCustomersParams filters GET /v1/customers (list mode).
type ListCustomersParams struct {
	Page    int
	PerPage int // max 100
}

// CreateCustomer opens a customer and provisions long-lived deposit addresses.
// On timeout, retry with the same WithRequestID and identical params.
func (c *Client) CreateCustomer(ctx context.Context, params CreateCustomerParams, opts ...CallOption) (*Customer, error) {
	params.RefID = strings.TrimSpace(params.RefID)
	params.Name = strings.TrimSpace(params.Name)
	params.Email = strings.TrimSpace(params.Email)
	params.Description = strings.TrimSpace(params.Description)
	if err := validateCreateCustomer(params); err != nil {
		return nil, err
	}
	params.Blockchains = NormalizeChains(params.Blockchains)
	var out Customer
	if err := c.do(ctx, http.MethodPost, "/v1/customers", params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCustomer fetches a customer by DoggiePay customer id (does not auto-ensure new chains).
func (c *Client) GetCustomer(ctx context.Context, customerID string, opts ...CallOption) (*Customer, error) {
	id := strings.TrimSpace(customerID)
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("doggiepay: customer id must be a UUID")
	}
	var out Customer
	path := "/v1/customers/" + url.PathEscape(id)
	if err := c.do(ctx, http.MethodGet, path, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCustomerByRefID looks up a customer by your ref_id UUID.
func (c *Client) GetCustomerByRefID(ctx context.Context, refID string, opts ...CallOption) (*Customer, error) {
	refID = strings.TrimSpace(refID)
	if _, err := uuid.Parse(refID); err != nil {
		return nil, fmt.Errorf("doggiepay: ref_id must be a UUID")
	}
	path := "/v1/customers?ref_id=" + url.QueryEscape(refID)
	var out Customer
	if err := c.do(ctx, http.MethodGet, path, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListCustomers returns a paginated customer list.
func (c *Client) ListCustomers(ctx context.Context, params ListCustomersParams, opts ...CallOption) (*CustomerList, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	perPage := params.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	path := fmt.Sprintf("/v1/customers?page=%d&per_page=%d", page, perPage)
	var out CustomerList
	if err := c.do(ctx, http.MethodGet, path, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateCustomer patches name/email/description.
func (c *Client) UpdateCustomer(ctx context.Context, customerID string, params UpdateCustomerParams, opts ...CallOption) (*Customer, error) {
	id := strings.TrimSpace(customerID)
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("doggiepay: customer id must be a UUID")
	}
	path := "/v1/customers/" + url.PathEscape(id)
	var out Customer
	if err := c.do(ctx, http.MethodPatch, path, params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

func validateCreateCustomer(p CreateCustomerParams) error {
	refID := strings.TrimSpace(p.RefID)
	if refID == "" {
		return fmt.Errorf("doggiepay: ref_id is required")
	}
	if _, err := uuid.Parse(refID); err != nil {
		return fmt.Errorf("doggiepay: ref_id must be a UUID")
	}
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("doggiepay: name is required")
	}
	if strings.TrimSpace(p.Email) == "" {
		return fmt.Errorf("doggiepay: email is required")
	}
	return nil
}
