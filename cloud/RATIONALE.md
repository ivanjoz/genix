## Publishing backend code also pushes the Lambda `CONFIG`

**Context** — `accion=1` shipped only the binary and `accion=2` only the environment, so a config
key that a code change had renamed stayed unpushed until somebody remembered the second action. The
`[server_utils]` → `[fareward]` rename hit exactly that: the deployed `CONFIG` still carried the old
section name, the new binary read an absent `[fareward]`, took it as `public = false` and dialed
`127.0.0.1:14013` from inside Lambda, where every lock and credit call was refused.

**Decision** — `accion=1` now ends with `UpdateEnviromentVariables` for both backend Lambdas.
`UpdateFunctionCode` leaves the function in `InProgress` and AWS rejects a second update while it
lasts, so `WaitForLambdaReady` (the SDK's `FunctionUpdatedV2Waiter`, 5-minute cap) runs first, and a
failed `UpdateFunctionConfiguration` panics instead of printing and returning.

**Rationale** — Coupling the two is what makes a config-shape change undeployable-by-halves; it
lives in `cloud` rather than in the deploy TUI so every caller of `accion=1` gets it, not only the
button. The cost is that publishing code now overwrites the deployed environment from the local
`config.toml`, which is the intended coupling but means a stale local file can now un-deploy a good
config. The swallowed error had to go with it: a silent env failure recreates the exact bug.

## The `CONFIG` that travels to Lambda has its comments stripped

**Context** — AWS caps a function's whole environment at 4 KB, keys and values together. `config.toml`
is documented line by line; at 10 KB of source it compressed to a 4836-byte base64 `CONFIG` and AWS
refused the update outright. Nothing could be deployed until the payload shrank.

**Decision** — Both writers (`cloud/helpers.go`, `scripts/deployer/lambda_env.go`, duplicated because
they are separate Go modules) drop whole-line comments and blank lines before compressing: 2912 of
4096 bytes for the full environment. Each one also measures the environment and fails with the byte
count before calling AWS.

**Rationale** — The backend parses TOML, where a comment carries nothing, so the cheapest 55% came
from content the runtime never reads — no key had to be dropped and the source file keeps its
documentation. It is line-based, so it is only safe while `config.toml` has no multi-line `"""`
strings, where a leading `#` would be content; inline comments after a value are left alone for the
same reason. The remaining 1.1 KB of headroom is why the size check reports on success too.

## The renderer zip URL comes from `frontend.app_url`

**Context** — This module carried its own hardcoded copy of the webpage-renderer artifact URL,
duplicated across three Go modules that cannot import each other.

**Decision** — The literal is gone; the URL is `frontend.webpage_renderer_url`, or
`<frontend.app_url>/webpage-renderer.zip` when that key is empty.

**Rationale** — See the full entry in `backend/RATIONALE.md`.

