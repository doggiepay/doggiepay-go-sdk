// Package doggiepay is the official Go client for the DoggiePay Integration Open API.
//
// It is intended for server-side use only. Keep Secret Keys (dp_live_sk_…) out of
// mobile apps, SPAs, and version control.
//
// Typical flow:
//
//  1. CreateCustomer for each end-user (ref_id = your user UUID)
//  2. Show FoldCustomerWallets(customer.Wallets) as deposit addresses
//  3. On webhook deposit.credited, credit your local user ledger
//  4. Freeze local balance, then CreateUserWithdrawal
//  5. On withdrawal.complete / withdrawal.failed, capture or unfreeze
//
// Webhook verification lives in the sibling package:
//
//	github.com/doggiepay/doggiepay-go-sdk/webhook
package doggiepay
