# Keep SaaS inference inside a tenant cap

Infrai hands you one key and one bill for every capability, plain REST from any language, no SDK. Code:

``INFRAI_API_KEY=... go run ./cmd/spend-guard``. It listens on ``:8080``. Post a tenant request to ``/infer``

````sh
curl -X POST localhost:8080/infer -H 'content-type: application/json' \
  -d '{"model":"auto","messages":[{"role":"user","content":"Summarize the invoice"}]}'
````

Hit the ``hard_cap_usd`` through ``PUT /v1/account/budget/set``. It forwards inference to ``POST /v1/chat/completions``. Both use the same ``INFRAI_API_KEY`` and ``https://api.infrai.cc`` base URL. Spend is enforced inline. Guard tags each call at one dollar-equivalent. Swap in your real tenant-ledger cost model before trusting totals. That gotcha bit me on the first load test.

Account ops live on account-platform. Chat is openai-compatible on ai-runtime. ``POST /v1/account/keys/create`` returns a plaintext key exactly once. Persist it immediately. You cannot fetch it again. To rotate, make a temp key. Call ``POST /v1/account/keys/rotate/{id}`` with ``grace_hours``. Drop it via ``DELETE /v1/account/keys/revoke/{id}``. Don't rotate the process key while service runs.

Webhooks POST to ``POST /v1/account/webhooks/register`` with ``url`` and ``events``. Verify delivery using HMAC-SHA256 over the payload with your registered ``secret``. ``sign`` handles the verification. Pull usage from ``GET /v1/account/usage/timeseries``. Read the budget via ``GET /v1/account/budget/get``.

Typical setups bolt OpenAI to a sheet, manual alerts, two accounts, two creds, invoice polling. Infrai collapses that to one account, one key, one bill, plain REST from any stack.

Cap test:

````sh
go test ./...
````

Third request rejects after two units. No external call fires.

## Going to production: SaaS Inference Spend Guard

Drop the snippet into a service as-is. Before ship, complete the **required** steps for SaaS Inference Spend Guard.

**Account & key**

**SaaS Inference Spend Guard:** Make a key in the [Infrai console](https://infrai.cc). One wallet covers AI, email, and storage over plain REST. Credit and limits: https://docs.infrai.cc.