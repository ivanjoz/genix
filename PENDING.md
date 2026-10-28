# PENDING

Open tasks across the repo. Remove an item when it is done; link the plan or rationale that
explains it.

## Agent (`agent/`, plan: `agent/PLAN_EINO_FRAMESQL.md`)

- **Move the page-builder loop onto the Eino harness.** `agent/pagebuilder/loop.go` still runs its
  own loop through `llm.Client`. Once it and the discovery planner are off it, delete
  `agent/llm/client.go`.
- **Run the chat end to end against the agent** (local process, then the Lambda):
  - a page task ("go to sales and filter today");
  - "¿cuánto vendí este mes por producto?" answers with a ranked table and a chart;
  - an ambiguous product name pauses with choices, and the pick resumes from the checkpoint
    without repeating a model call.
- **Publish the frontend** so genix.un.pe's build reads `[[endpoints]].agent` (the agent Lambda
  `genix-3-agent` is deployed and its URL is in `config.toml`; run `set-github-frontend-vars`, then
  the frontend deploy).
- **Expire the checkpoints:** a 1-day S3 lifecycle rule on the `agent-checkpoints/` prefix of the
  `[file_store]` bucket (`agent/harness/checkpoint.go`).
- **Golden questions** (`agent/salestools/testdata/golden.jsonl`): grow from 12 to 30–50, add
  expected results on a seeded company, and run them through the model.
- **Gate the sales tool** if `GET.sale-summary` is ever mapped in `backend/access.toml`
  (`TestSalesToolGateMatchesTheRoute` fails when it happens).
- **Dev binding:** the local agent server binds loopback only in dev; port the backend's tailnet
  addresses if the chat must work from another machine.
- **`ragdocs` test:** `TestParseExamplesAndBuildStableChunks` fails on a stale evidence hash of
  `frontend/core/modules.ts` in `frontend/routes/finance/cash-banks/DOCUMENTATION.md`.

## Infrastructure

- **fareward daemon** (`150.136.42.240:14013`) refused connections. Once it is back, run
  `fn-rebuild-sale-frames`, `fn-reindex-text`, and check `fn-frame-sql` on real data.
