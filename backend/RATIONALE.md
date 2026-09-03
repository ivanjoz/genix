## A user is valid with direct accesses and no profile

**Context** — `PostUsuarios` rejected any user whose `ProfileIDs` was empty ("El user debe tener al
menos 1 permiso"), but the users form grants permissions two ways: through profiles (`ProfileIDs`)
and directly, per access+level (`AccessLevelIDs`). Assigning only direct accesses — the common case
for a one-off user who does not warrant a profile — was rejected even though the form showed the
granted cards.

**Decision** — the check passes when either list is non-empty, and the message names both sources:
"El user debe tener al menos 1 perfil o 1 acceso directo".

**Rationale** — `AccesosComputed` is already built from the union of both lists further down the
same handler, so the validation was the only place that treated profiles as the sole source of
permissions. The invariant that matters is "the saved user ends up with at least one access", and
both inputs feed it.

## The session token's hash is a 128-bit keyed BLAKE2s tag

**Context** — `ComputeUsuarioTokenHash` produced a `uint64` by taking the first eight bytes of an
HMAC-SHA256, and fareward's SSE bridge recomputes it to verify a browser identity without a round
trip. Two problems in one function: a 256-bit digest paid for and mostly discarded, and — the one
that matters — only 64 bits of tag on a bearer credential. The token carries no random component,
so the tag is not integrity protection over a secret, it *is* the credential; 64 bits sat exactly on
the floor NIST sets for session secrets, below what every mainstream signed-token format uses.

**Decision** — Keyed BLAKE2s-128 under the `usrToken:v3` domain, keyed by the full
`SHA-256(SECRET_PHRASE)` — 32 bytes, exactly BLAKE2s' maximum key length, so a configuration string
of any length reaches the key whole. `UsuarioToken.Hash` widened from `uint64` to `[]byte`, and
`CheckUser` compares with `subtle.ConstantTimeCompare`. `golang.org/x/crypto` becomes a direct
dependency. `agent/bridge.go` moved in the same change but to a different primitive: `X-Bridge-Auth`
is SipHash-2-4 under `sse-bridge:v2`, 16 hex characters, because it is an internal
service-to-service header with a 300 s window rather than something a user holds.

**Rationale** — Keyed BLAKE2s-128 rather than HMAC-SHA256 truncated to 128 bits: RFC 7693 defines
the keyed mode as a MAC, `x/crypto` refuses to construct `New128` without a key precisely because a
128-bit digest is only safe as one, and it is a purpose-built 128-bit tag instead of a truncation.
The Rust mirror in `fareward/src/bridge/auth.rs` uses RustCrypto's `Blake2sMac<U16>`, pinned against
`x/crypto`'s own `hashes128` vectors — BLAKE2 folds digest and key length into its parameter block,
so BLAKE2s-128 keyed is not BLAKE2s-256 truncated, and a mismatch would only surface as every
browser being rejected. The cost is breaking and taken deliberately in pre-alpha: tokens under
`usrToken:v1`/`v2` no longer validate, so every session logs in again, the token grew 8 bytes, and
the backend and the daemon must deploy together. See `fareward/RATIONALE.md` for the full decision.

## A dev backend dials its own fareward daemon unless `use_remote_dev_host` says otherwise

**Context** — `config.toml` carried `[fareward] public = true, host = <server IP>`, so a developer
running the backend locally dialed the deployed daemon while a freshly built one listened, unused,
on `127.0.0.1:14013`. The two builds were on different sides of the `fareward:v6` → `fareward:v7`
domain rename, so every frame's HMAC failed and the daemon closed the connection. There is no
diagnosis in that: the client reports `connection closed while waiting for a reply`, the daemon
logs a rejected frame at `debug!` on another machine, and only the operations that wait for a reply
surface it at all — a fire-and-forget log frame or an exempt company hides it completely.

**Decision** — `makeFarewardAddress` takes `is_local` and a new `[fareward] use_remote_dev_host`.
With `is_local = true` the address is `127.0.0.1:<port>` regardless of `host` and `public`, unless
`use_remote_dev_host = true`. `public = false` still forces loopback on its own, so the opt-in only
ever chooses between two hosts and can never invent a route to a daemon bound to loopback elsewhere.

**Rationale** — The wire protocol is versioned precisely because backend and daemon have to cross a
format change together, and a checkout already contains the daemon that matches it: on a dev
machine the local one is the only correct default. Making it a default rather than advice means the
mismatch cannot happen by leaving a deployment value in `config.toml`. The cost is that pointing a
dev backend at the shared daemon — to reproduce a limiter state that only exists there — now needs
a config key instead of just working; that is the rarer case, and it fails loudly (a stale build
closes the connection) rather than silently.

## The webpage-renderer zip URL is derived from `frontend.app_url`, not hardcoded per module

**Context** — The renderer artifact URL was declared as a literal in three separate Go modules —
`backend/core/security.go` (`DefaultWebpageRendererURL`), `cloud/webpage-renderer.go`
(`defaultRendererZipUrl`) and `scripts/deployer/lambda_env.go` (`defaultRendererZipURL`). They are
separate modules and cannot import each other, so nothing but a comment kept the three equal, and
the comments only named two of the three copies. Moving the app off `genix-dev.un.pe` meant editing
the same domain in three places that the compiler could not cross-check.

**Decision** — The three constants are deleted. `frontend.webpage_renderer_url` still wins when
set; otherwise every module derives `<frontend.app_url>/webpage-renderer.zip` from config.toml.
`app_url` was missing from `config.example.toml` and was added. Where the derivation cannot
produce a URL (both keys empty) each module reports it in its local idiom: the backend leaves
`Env.WEBPAGE_RENDERER_URL` empty and `backend/cloud/webpage_renderer.go` names it in the
required-variable check before running the renderer locally, `cloud/webpage-renderer.go` panics
like its neighbouring `cdn_url` check, and the deployer prints a warning and skips the renderer
Lambda, matching the existing `cdn_url`-empty branch.

**Rationale** — CI publishes `webpage-renderer.zip` next to the frontend, so the artifact URL is
not independent information: it is `app_url` plus a filename. Deriving it makes the domain a single
declaration in config.toml and removes a class of silent drift, at the cost of one implicit
coupling — if CI ever publishes the zip somewhere other than `app_url`, every deployment must set
`webpage_renderer_url` explicitly. The derivation is duplicated rather than shared because the
three modules cannot import one another; that duplication is one string concatenation each instead
of three literal domains.

## The fareward client is no longer part of this module

**Context** — The client lived in `core/fareward/`, importing only the standard library. That made
it portable by accident, and its location cost two things: a wire change to the daemon's HMAC
domain meant editing this repository and the daemon's in lockstep with nothing enforcing it, and
the cross-language vectors that pin the two implementations byte for byte sat on opposite sides of
a repository boundary.

**Decision** — The package moved to the daemon's repository as `github.com/ivanjoz/fareward/go`,
reached through `replace github.com/ivanjoz/fareward/go => ../fareward/go`, the pattern
`genix-orm` and `facturago` already use. `core/fareward_api.go` stays: it is the seam, and
everything in it is a Genix concern — the aliases that keep call sites saying `core.X`, the
`LockAction` enum, and the three `HandlerArgs`/`HandlerResponse` mappings. `main.go` still pushes
`core.Log` in through `fareward.SetLogger`. No signature changed on either side.

**Rationale** — The module boundary now enforces what a convention used to. `core/fareward` was
forbidden from importing `core` because `CreditLimitExceeded` would have made a cycle; that rule
was a comment and a code review. It is now a different Go module, so the compiler states it.

In `MODULE_BOUNDARIES.md` the client therefore drops from **L1 Core** to **L0 Foundation**,
alongside `genix-orm` and `facturago` — external, importing nothing from `app/`.

The cost is that a wire-protocol fix is now a submodule commit before it is a backend commit, which
is the same cost `genix-orm` already imposes and the same one that keeps the daemon and its client
from drifting.

## The fareward client takes the name all the way down, including the wire and the schema

**Context** — The submodule folder became `fareward/` to match its repository, and the daemon
renamed its crate, TOML section and env vars to suit. The Go client still lived in
`core/server_utils/`, re-exported through `core/server_utils_api.go`, so the backend named the
service one thing and the service named itself another. An earlier pass fixed the Go package but
deliberately stopped at two contracts — the HMAC domain string and the `server_metrics` columns —
on the grounds that neither is really a name. That left `server_metrics` reading `server_utils_*`
for columns owned by `fareward`, and a wire whose identity nothing in either repository spelled.

**Decision** — `core/server_utils/` → `core/fareward/` (package `fareward`),
`core/server_utils_api.go` → `core/fareward_api.go`, and every identifier renamed with it:
`ServerUtilsClient` → `FarewardClient`, `ErrServerUtilsUnavailable` → `ErrFarewardUnavailable`,
`Env.SERVER_UTILS_ADDRESS` → `Env.FAREWARD_ADDRESS`, and the `toml:"server_utils"` tag on the
config struct → `toml:"fareward"`. The two contracts followed:

- `farewardAuthDomain` in `core/fareward/connection.go` is now `"fareward:v7"`, mirrored byte for
  byte by `DOMAIN` in `fareward/src/service/auth.rs`. Renaming it is not a frame-format change, but
  it invalidates every tag a peer still signing `genix-server-utils:v6` produces, so it spends a
  version bump rather than leaving two incompatible protocols both answering to `:v6`. The
  cross-language vectors in `credits_test.go` and `locks_test.go` were regenerated against the new
  domain and re-pinned on the Rust side, so the pair still proves the two implementations agree.
  **Backend and daemon must cross this boundary in a single deploy.**
- `ServerMetricRecord.FarewardMemMb` / `.FarewardCpuPercent` in `core/types/server_metrics.go`,
  their mirrors in `config/server_metrics.go`, and the frontend's `ServerMetricField` union. These
  carry no column tag, so the ORM derives `fareward_mem_mb` / `fareward_cpu_percent` from the field
  name — the exact columns the Rust writer's INSERT names and the JSON keys the Server Panel reads.

**Rationale** — Grepping `fareward` now finds the whole path from handler to daemon to column, and
the two exceptions were precisely the half that made grepping unreliable: names you could only find
by already knowing them. The costs are real and accepted. The domain change is a lockstep deploy.
The column change is a schema change on a table that already holds rows: the ORM adds missing
columns and never drops them, so a deployed `server_metrics` gains the new pair and keeps
`server_utils_*` until those rows expire under the table's TTL. Nothing back-fills, so the Server
Panel reads not-measured for windows sampled before the deploy. Ordering is not a constraint in
either direction — the daemon's `ensure_prepared` retries on a one-minute interval, so a daemon that
starts before the schema deploy heals on its own rather than needing a choreographed rollout.

## fn-init seeds a company that can already operate, and the bootstrap page is gone

**Context** — A company needs a Site, a Warehouse and a CashBank before it can do anything. The
sign-up wizard (`/welcome`) collects them as step 3, reusing `InitialDataForm`. But `fn-init`
writes companies 1 and 2 by hand and created none of those rows, so login had to report
`InitialDataPending` and redirect to a standalone `/initial-data` page — a second entry point
existing only for the companies the wizard never touched.

**Decision** — `seedCompanyOperatingRecords` in `exec/init.go` gives every seeded company the same
three records `PostInitialData` creates. With that, `InitialDataPending` is always false, so the
flag, `hasPendingInitialData` and the `/initial-data` route are all deleted. The wizard is the only
path, and `InitialDataForm.svelte` stays because it is what the wizard renders.

**Rationale** — the standalone page existed to compensate for an incomplete seed. Completing the
seed removes the reason for it rather than leaving two ways to reach the same three inserts, one of
which almost never ran and was therefore the one that rotted.

Deleting `hasPendingInitialData` also takes two ScyllaDB queries off every login. It was the one
part of login that touched ScyllaDB at all — users live in the cloud store — which is why it needed
a `recover()` and a "degrade to false" path that no longer has to exist.

The seeded rows use autoincremented IDs rather than literal ones, so unlike the companies and users
above they need no `reserveSeededAutoincrementIDs`: an autoincremented insert advances its own
counter.

**The accepted cost:** a company whose warehouse or cash bank is later deleted has no guided way
back. It has to be fixed from the Sites & Warehouses and Cash Banks pages, which is where those
records are managed anyway.

## Company 1 is exempt from credit budgets, not from permissions

**Context** — The credit limiter meters every request through `chargeConfiguredCredits`. It also
meters the platform operator's own company, so an exhausted budget locked company 1 out of its own
software — including the "Datos Iniciales" bootstrap, which is exactly when someone needs to get in.

**Decision** — `fareward.CreditExemptCompanyID = 1` (the company `fn-init` seeds as
"Principal"). In `chargeConfiguredCredits`, that company's CPU and inference credits are zeroed. If
the frame then carries no required access there is nothing left to ask and the call returns nil;
if it does carry one, the frame still goes to the daemon and the access is still enforced.

**Rationale** — the seam is the single funnel every charge already passes through: API usage, the
GET top-up and agent inference all reach the daemon through it, so one guard covers all three
rather than three guards that can drift apart.

Zeroing the credits rather than skipping the frame is what keeps the exemption narrow. It mirrors
the split `ChargeAPIAccessOnly` already makes for the credit-exempt routes: no budget, same
authorization. Returning early on an empty frame is required, not an optimization —
`encodeCharge` rejects a frame carrying neither credits nor an access, so a blanket short-circuit
would have turned every exempt read into a limiter error.

**The accepted cost:** company 1 is unmetered, so its usage does not appear in the credit
accounting and cannot be capped. That is the intent — it is the operator, not a tenant — but it
does mean a runaway loop under company 1 has no budget backstop.

## Product quantities are a (whole units, sub-units) pair, not a scaled integer

**Context** — Sale orders could not express a fraction of a product. The `Sbu*` sub-unit columns
on `Product` existed but no backend code read them: the sale page pushed a sub-unit count straight
into `DetailQuantities`, so selling 3 candies deducted 3 boxes. Any fix has to serve two different
needs at once — divisible goods (rice by the gram) want fractions, packaging (a box of 6) wants an
integer factor, and 1/6 is not representable in any decimal scale.

**Decision** — `core.Quantity{Units, Sub int32}` plus a divisor, in `core/quantity.go`. The value is
`Units + Sub/Divisor`, an exact rational. `Units` is never scaled; all fractional information lives
in `Sub`/`Divisor`, so continuous goods are simply divisor 1000 and discrete ones divisor 6. `Sub` is
deliberately **not** normalized in storage: it may exceed the divisor and may be negative.

It is stored two ways, split by what the row is for. **Aggregates keep two flat `int32` columns** —
stock, the movement ledger and the sale summaries, everything that is accumulated or `SUM()`-ed.
**Documents pack** both halves into one `int32` as `Units*1000 + Sub`, so a sale-order line reads
1001 at divisor 6 as one box plus one candy. `core.Quantity` is a computation type assembled from
either form, never a column type itself.

**Rationale** — leaving `Sub` unnormalized is what makes addition closed without a carry:
`{a1,b1} + {a2,b2} = {a1+a2, b1+b2}`, order-independent. That buys two things a packed
`whole*1000+sub` integer cannot. The stock engine keeps accumulating with a plain `+` — no
decompose/normalize/recompose with signed borrow in the two hottest write paths — and Scylla can
still `SUM()` both columns independently, so `product-supply-management.go`'s pushdown survives.
Normalization is confined to validation and display, where it is isolated and unit-testable.

A packed integer breaks on the first subtraction: `stock.Quantity = prev + mov.Quantity` computes
`1000 - 4 = 996` (0 units + 996 sub-units, invalid and ~498x too large, and it passes the existing
`< 0` guard) where the answer is `2`. That argument only binds where rows are accumulated, which is
why documents pack and aggregates do not: nothing ever adds two sale-order lines together or asks
the database to sum them, so a line pays one column instead of two. `PostSaleOrder` is the single
conversion point.

The packing slot is 3 digits, and both ceilings it imposes are per line, reaching no aggregate:
`MaxQuantityDivisor` is 1000, because a normalized `Sub` has to fit 0..999 — a kilogram sold by the
gram fills it exactly, and cannot be refined further afterwards. `MaxQuantityLineUnits` is 2,147,483
whole units on one line.

**Costs accepted:** two columns per quantity instead of one, and a divisor that has to travel with
them. `int64` at facturago's `QuantityScale = 1_000_000` would have removed every headroom concern
and made the SUNAT bridge an identity cast, but with `Units` unscaled the `int32` ceiling is already
~2.1 billion whole units — the same magnitude as today — so the extra width bought nothing. Note the
ORM only *reports* a column type mismatch (`genix-orm/scylla/deploy.go:952`) and Scylla will not
convert `int` to `bigint` in place, so that choice would have meant recreating tables.

Full analysis and the rejected alternatives: `docs/QUANTITY_SCALE_PLAN.md`.

## The boundary rule is enforced by a script, not by good intentions

**Context** — Go cannot catch a module-to-module import: `accounting` importing `finance`
compiles cleanly. The six violations this refactor removed had accumulated exactly that way,
and two of them had already produced copy-pasted code rather than a compile error. Left
unchecked the rule would decay again.

**Decision** — `scripts/boundaries/check_module_imports.go`, dispatched as
`go run . check_module_imports` and registered in the `deploy.sh` TUI beside `check_tables`.
It reads `go list -json ./...` from `backend/` and classifies every import edge against the
layer table. The rule and its consequences are documented in
`backend/docs/MODULE_BOUNDARIES.md`, with pointers from `AGENTS.md`.

**Rationale** — the layer membership is written as **literal path lists**, not inferred from
directory shape. A new top-level package then fails loudly ("not classified") instead of being
silently guessed into the wrong layer, and `rg moduleBodies` shows the whole policy. Test
imports are included, since a `_test.go` in `accounting` importing `finance` is the same
violation.

Writing the test first paid for itself: it caught that the checker allowed
`finance/types -> cloud`, because the infrastructure allowance was evaluated before the
`*/types`-must-stay-a-leaf rule. That edge does not exist today, so nothing would have
surfaced it until someone added it — which is exactly the case a checker is for. The order is
now leaf-rule first, and `check_module_imports_test.go` pins all six original violations plus
the leaf property.

The checker was also verified end-to-end by injecting `accounting -> finance` and confirming a
non-zero exit and a message naming the fix, then reverting.

