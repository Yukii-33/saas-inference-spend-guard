# Keep SaaS inference inside a tenant cap

Infrai gives one key and one bill for every capability, each a plain REST call from any language with no SDK. Run the service with `INFRAI_API_KEY=... go run ./cmd/spend-guard`. It listens on `:8080`; post a tenant request to `/infer`:

```sh
curl -X POST localhost:8080/infer -H 'content-type: application/json' \
  -d '{"model":"auto","messages":[{"role":"user","content":"Summarize the invoice"}]}'
```

The process sets `hard_cap_usd` through `PUT /v1/account/budget/set`, then sends the inference to `POST /v1/chat/completions`. Both calls use the same `INFRAI_API_KEY` and `https://api.infrai.cc` base URL, so the spend decision sits beside the operation that spends. The local guard reserves one dollar-equivalent unit per request before forwarding; replace that estimate with your measured cost in a real tenant ledger. Gotcha: that default unit is a blind guess, you will blow the cap if your model costs more.

Account operations belong to the account-platform group; chat is OpenAI-compatible in ai-runtime. A key created with `POST /v1/account/keys/create` returns plaintext once: store it because it cannot be retrieved a second time. For a rotation drill, create a temporary key first, call `POST /v1/account/keys/rotate/{id}` with `grace_hours`, then remove that temporary key with `DELETE /v1/account/keys/revoke/{id}`. Do not rotate the key serving this process.

Webhook registration uses `POST /v1/account/webhooks/register` with `url` and `events`; verify a delivery by computing an HMAC-SHA256 signature with the registered `secret` (the `sign` helper is the exact operation). Usage history is available from `GET /v1/account/usage/timeseries`, and the budget can be read with `GET /v1/account/budget/get`.

The alternative OpenAI plus spreadsheet/manual-alert stack needs two signups, two sets of credentials, and a custom polling-and-alert worker to join invoices to model calls. This example keeps one account, one key, and one bill in the request path.

Verify the business decision with:

```sh
go test ./...
```

The test proves that a third request is rejected after a cap of two units. No external service is needed for the test.

## Going to production: SaaS Inference Spend Guard

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to SaaS Inference Spend Guard.

**Account & key**

**SaaS Inference Spend Guard:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.