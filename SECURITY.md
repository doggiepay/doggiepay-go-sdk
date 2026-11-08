# Security Policy

## Reporting a vulnerability

Please email **security@doggiepay.com** (or open a private security advisory on GitHub).
Do not file public issues for secrets exposure, auth bypass, or payment integrity bugs.

## Integrating safely

1. **Secret Key is server-only.** Never ship `dp_live_sk_*` in mobile apps, SPAs, or public repos.
2. **Prefer a secret manager** (AWS Secrets Manager, GCP Secret Manager, Vault, KMS) over plain env files in production.
3. **Always verify webhooks** with `webhook.ConstructEvent` using the raw request body and `DOGGIEPAY_WEBHOOK_SECRET`. Reject on failure with HTTP 401.
4. **Enforce HTTPS** for API base URLs. The SDK rejects `http://` unless you explicitly call `WithAllowInsecureHTTP()` (local only).
5. **Reuse Request-Id on retries** for write APIs; never invent a new id when retrying the same CreateCustomer / Patch payload.
6. **Use Idempotency-Key** on withdrawals and align it with your local freeze ledger key.
7. **Treat end-user balances as your responsibility.** Credit only after verified `deposit.credited`; unfreeze only after verified `withdrawal.failed`.
8. **Do not log secrets.** `Client.String()` redacts the key; avoid dumping `Authorization` headers in access logs.
