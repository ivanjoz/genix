# The credit limiter, end to end

How one HTTP request becomes a charge, who decides what, and where the numbers end up. Describes
**what is implemented today** — read this before `PLAN.md`, which does not cover authorization,
multiplexing or the extra pool.

The reference material is split across `README.md` — "Rate limiter behavior", "TCP contract",
"Authorization behavior", "Extra credits", "Go charging rules". This document is the flow those
sections are pieces of.

---

## 1. The two questions

Every authenticated request has to answer two things before it may run:

1. **May this user do this?** — authorization against `access_list.yml`.
2. **Can this tenant afford it?** — quota against the company's entitlement.

**One frame answers both.** The router asks once, before the handler, and gets one reply carrying
both verdicts:

```
   ┌──────────────────────────────┐
   │ backend (Lambda or VPS)      │
   │                              │       one frame        ┌─────────────────────┐
   │   no ScyllaDB read here      │──────────────────────▶ │ daemon, always      │
   │                              │ ◀───────────────────── │ resident, in-memory │
   │   access verdict + quota     │      one reply         └─────────────────────┘
   └──────────────────────────────┘
```

The daemon is where the grants are cached because it is the only process that is always resident.
The backend cannot cache them usefully: on Lambda every new execution environment starts empty, and
at any scale a large share of requests are somebody's first — so an in-process cache there would pay
a full ScyllaDB round trip on the authorization path, before the handler does anything.

**Only the answer lives here, not the policy.** The daemon holds no copy of `access_list.yml`, never
sees an access *name*, and does not know which route maps to which access or what level a method
implies. It answers "does this user hold any of these four packed grants".

---

## 2. Who talks to whom

```
browser              backend (Go, Lambda or VPS)            server-utils (Rust daemon)
   │                          │                                       │
   │── GET api/productos ────▶│                                       │
   │                          │  CheckUser: token only, no DB         │
   │                          │  resolveRouteAccess: the catalogue     │
   │                          │  chargedMethodFor: the tariff          │
   │                          │                                        │
   │                          │──── 0x01, 29 bytes ──────────────────▶│  admit_at
   │                          │                                        │   ├ authorize
   │                          │◀─── 5-byte reply ─────────────────────│   ├ burst gates
   │                          │                                        │   ├ entitlement
   │                          │  handler runs                          │   ├ extra pool
   │                          │                                        │   └ charge
   │                          │──── 0x01 top-up (only if > 8 KiB) ───▶│
   │◀── 200 ──────────────────│                                        │
                                                                        │
                                                        every 15s: flush_dirty
                                                                        │
                                                                        ▼
                                              credit_usage_company · credit_usage_user
                                                     company_credit_budget
```

Both processes must be deployed together: the frame HMAC is domain-separated by version
(`genix-server-utils:v6`), so a mismatched pair fails every frame.

---

## 3. What Go decides before the frame

All of this is `enforceAccessAndCredits` in `backend/main-handlers.go`, which runs **before the
handler is even looked up**. That is deliberate: a request to a route that does not exist is charged
too, so scanning for endpoints stops being free. The daemon attributes it to route 0.

### 3.1 Which accesses to require — `resolveRouteAccess`

The catalogue is `access_list.yml`, embedded in the Go binary. Every rule that means *"do not ask
the daemon"* produces an **empty slot list**:

| Case | Required access | Why |
|---|---|---|
| Unmapped `GET` | *(empty)* | Reads close one at a time; an unmapped one is free to any session |
| Mapped `GET` | one per access, nivel 1 | A read in the catalogue behaves like everything else |
| `POST`/`PUT`, mapped | one per access, nivel 2 | A write always needs a level-2 grant |
| `POST`/`PUT`, **unmapped** | *(refused in Go, 403)* | The catalogue denies by default — see the trap below |
| `selfServiceRoutes` | *(empty)* | Needs a session, no access (`POST.user-self`) |
| **User 1** | *(empty)* | `login.go` synthesizes its grants in the login response and never persists them, so its stored blob is empty and the daemon would deny it |
| More than 4 accesses | *(refused in Go, 500)* | The frame holds four slots; truncating would authorize against fewer accesses than declared |

> **The trap.** An empty slot list is precisely a frame that asks for *no* authorization. So an
> unmapped write cannot be delegated to the daemon — that would make it an open route. It is refused
> in Go, with no frame, and `TestResolveRouteAccess` pins it.

A grant is packed into one `u16`: `acceso_id << 2 | (nivel - 1)`. The level occupies the low two
bits and nothing else, which is what makes `required | 0b11` the bucket ceiling on the Rust side.

### 3.2 What it costs — `chargedMethodFor` and `APICPUCredits`

```
GET        2 credits for the first 8 KiB, then 1 per started 16 KiB
POST/PUT   5 credits for the first 8 KiB, then 1 per started  8 KiB
inference  1 per started 8 KiB of provider input, 2 per started 8 KiB of output
```

`POST` and `PUT` are **one behaviour**, declared in a single `case "POST", "PUT"` so the two cannot
drift apart: same tariff, same required level. `TestPostAndPutAreIndistinguishable` asserts it at
every payload size, because a method the tariff does not recognize returns an error and the router
fails closed on one.

`chargedMethodFor` returns `""` for two situations, and both still send a frame:

- **A method with no tariff** (`DELETE`, `PATCH`…) — authorized, not charged. Better than a 503.
- **`creditControlRoutes`** — the credit panel's own reads, so a tenant out of credit can still see
  *why*. They skip the **credits**, never the **frame**: three of them are access-mapped and two are
  SaaS-only, so skipping the frame would leave them open to any session.

A frame with neither credits nor a required access is refused by both sides. Everything else is
valid, including an authorize-only frame (zero credits, slots filled).

### 3.3 Why a GET is charged twice

A GET's byte count only exists *after* the handler runs, but its authorization verdict is needed
*before*. So it is split:

```
   ┌─ pre-handler frame ─┐                        ┌─ top-up frame ─┐
   │ access check        │                        │ credits only   │
   │ base: 2 credits     │──▶ handler ──▶ 40 KiB ─│ +2 credits     │
   └─────────────────────┘                        └────────────────┘
                                    only when the response passed 8 KiB
```

Most GETs never pass 8 KiB, so most send one frame and no top-up. Two consequences worth knowing
when reading the usage tables: **a GET that ends in an error still costs its two base credits**, and
**a streamed response is charged its base and never topped up** — those bytes never went out as a
measured response.

> **Known wart.** The top-up can be *refused*, and then a 429 replaces a body that was already
> generated. The work was done either way, so counting it would be strictly better than denying it
> and losing the number. Not fixed; see the end of `docs/EXTRA_CREDITS_PLAN.md`.

---

## 4. The bytes

### Handshake

On accept, the daemon writes **eight random bytes**. Every later frame is
`[opcode:1][payload][hmac:8]`, big-endian, the tag bound to the nonce *and* to the frame's sequence
number.

### Request frame — 29 bytes

```
 offset  0        3        6      8      10       12                    20
        ┌────────┬────────┬──────┬──────┬──────┬──────┬──────┬──────┬──────┐
opcode  │company │  user  │route │ cpu  │infer │ acc0 │ acc1 │ acc2 │ acc3 │  hmac
 0x01   │  u24   │  u24   │ u16  │ u16  │ u16  │ u16  │ u16  │ u16  │ u16  │  8 bytes
        └────────┴────────┴──────┴──────┴──────┴──────┴──────┴──────┴──────┘
```

The real pinned vector, from `TestChargeFrameMatchesTheRustAuthVector` and its Rust twin
`matches_the_go_client_vectors`:

```
01 12 34 56 00 00 2A 00 67 01 2C 00 19 01 39 00 8B 00 00 00 00 0F 13 FA B1 DE F3 CA A8
│  └──┬───┘ └──┬───┘ └─┬─┘ └─┬─┘ └─┬─┘ └─┬─┘ └─┬─┘ └─┬─┘ └─┬─┘ └────────┬─────────┘
│   company    user   route  cpu  infer  acc0  acc1  acc2  acc3       hmac
│   0x123456    42     103   300    25   0x139 0x8B    —     —
└─ CHARGE_CREDITS
```

Two things about this layout are worth stating out loud:

**The slots fill from index 0 and zero terminates.** A gap would truncate the list at the gap and
authorize against fewer accesses than the caller named, so a sparse list is a malformed frame
(`SparseRequiredAccess`), not a short one. A zero can therefore never be a grant — it would mean
acceso 0, which does not exist.

**The high bit of the route field is not part of the route number.** `MAX_ROUTE_ID` is fourteen
bits, so the top two of those sixteen are free. Bit 15 is `EXTRA_CREDIT_FLAG` — see §6. Bit 14 is
unassigned, and the range check both sides already run is what refuses a frame carrying it: the flag
is stripped before the check, so anything left above fourteen bits is an error.

### Reply frame — 5 bytes

```
[correlation:u16][status:u8][detail:u16]
```

`correlation` is the low 16 bits of the request's sequence, echoed back — nothing extra travels to
carry it, and it is what lets one connection serve many callers at once.

**The two refusals arrive in different fields**, and they can never both be set, because
authorization is resolved first and returns without charging:

```
status  0        allowed
        else     scope | window<<1 | inference<<3 | cpu<<4      ← a 429
        0xFF     the daemon could not answer                    ← fail closed

detail  0        nothing was asked                              ← treated as unavailable if it WAS
        1        granted
        2        the user holds none of the required accesses    ← a 403
        3        no such user in this company                    ← a 401
        4        the user is not active                          ← a 401
```

Go reads `status` for a 429 and `detail` for a 403 or 401. A daemon that ignored the slots would
answer `detail = 0`; that is treated as unavailability rather than as a grant, because failing open
would silently unauthorize every gated route the moment the two binaries drifted apart.

---

## 5. What the daemon decides — `admit_at`

One shard lock for the whole decision. The shard is `company_id % shard_count`, and the
authorization cache lives in the same map, on the same key, behind the same mutex as the quota
state — so a request that both authorizes and charges takes **one** lock.

```
  ┌─ 1. AUTHORIZE ───────────────────────────────────────────────────────┐
  │  ensure_access → cached grants, or one ScyllaDB point read           │
  │  verdict: identity first (found? active?), then the grants           │
  │  DENIED ──▶ return. No usage row, no quota state, no budget load.    │
  └──────────────────────────────────────────────────────────────────────┘
                                  │ granted
  ┌─ 2. BURST GATES ─────────────────────────────────────────────────────┐
  │  company 10s → user 10s → company 1h → user 1h                      │
  │  DENIED ──▶ return. Never relaxed by anything.                      │
  └──────────────────────────────────────────────────────────────────────┘
                                  │
  ┌─ 3. ENTITLEMENT GATES ───────────────────────────────────────────────┐
  │  company daily (company_credit_budget.daily_cpu)                     │
  │  user daily  (= company daily / 2)                                   │
  │  monthly ceiling (inactive month = everything refused)               │
  └──────────────────────────────────────────────────────────────────────┘
                    │ all pass                     │ any refuses
                    ▼                              ▼
             charge normally            ┌─ 4. EXTRA POOL ────────────────┐
                                        │  marked as a read?             │
                                        │  inference == 0?               │
                                        │  fits in what is left today?   │
                                        │  no ──▶ return the violation   │
                                        └────────────────────────────────┘
```

**Refusal precedes charging.** A 403 costs the tenant nothing: the work given away is one binary
search over a cached list. Deliberate, and pinned by `a_refusal_charges_nothing_at_all`.

**The authorization cache.** Per `(company, user)`: the sorted packed grants, `users.status`, and
whether the row exists at all — negative results are cached too. Two bytes per grant, so a user
holding every access in today's catalogue costs 68 bytes, less than the `HashMap` entry around it.
`rate_limit.access_cache_seconds` (default 600) is a **backstop, not the mechanism**: the backend
sends `INVALIDATE_USER_ACCESS` (`0x06`) right after rewriting the column, so a revoked access stops
working immediately. The TTL only covers a lost frame or a restarted backend.

> **`accesos_computed` is little-endian.** It is a `blob` of `u16`s written by
> `backend/genix-orm/scylla/converter.go` with `binary.LittleEndian.PutUint16`, while every integer
> in this protocol is big-endian. Reading it the wrong way round does not fail — it authorizes the
> wrong things. A real blob, `[31,35,39,43,47,51,55,56,60]`, is accesos 7–13 at nivel 4 and 14–15 at
> nivel 1; read big-endian it becomes accesos 1984–3840, all outside the catalogue, which is a
> silent total denial. `decodes_a_real_blob_from_the_database` pins it against bytes from production.

**Identity is checked before grants**, because they are different HTTP answers: a client that has
lost its identity must re-authenticate (401), one that merely lacks a permission must not (403).

**A grant covers the levels above it.** `holds` is a binary search plus one bounded lookahead:
`required | 0b11` is the ceiling of the bucket the required access lives in, so any grant for the
same acceso at a *higher* level satisfies a lower requirement.

---

## 6. The extra pool

`rate_limit.company_extra_credits_24h` is CPU a company may spend per local business day **after**
its normal quota has already refused, and only on a frame marked as a read. It is the difference
between a tenant out of credit seeing a 429 everywhere and one that can still look at its data.
Zero — the default — removes the feature entirely.

**Reads only, and the daemon does not decide which.** Eligibility rides in bit 15 of the route
field, set on the Go side inside `ChargeAPIUsage`, derived from the very string that chose the
tariff. It is not a parameter: a caller cannot mark a write by disagreeing with itself, because
there is only one value to disagree with. A marked frame that also asks for inference credits is
not relaxed in any dimension — the pool is a single CPU figure and has nothing that could
authorize one.

**The burst gates are never relaxed**, and a charge paid from the pool still consumes burst tokens
and still counts against `hour_used`. Skipping them would hand a company in read-only mode
unlimited burst. What the pool bypasses is the *entitlement*, not the rate.

**The pool is consulted last.** A read that fits inside the entitlement is charged against it; only
once one of the three entitlement gates refuses is the pool asked. If it cannot cover the charge the
original violation is returned unchanged, so the client sees exactly the 429 it would have seen.

**No per-user share.** The pool belongs to the company and one user can drain it. Halving it the way
the daily user gate is halved would leave a single-user company — most of them — unable to reach it
at all.

> That is not a hypothetical. With `daily_cpu = 4`, the **user** gate is 2, so a single user is
> refused at half the company's allowance and never reaches a company-level pool. Three of the
> tests in `quota.rs` are built on exactly that fixture.

**Counted apart.** Extra spending never touches `day_used` or `month_used`, so `daily - day_used`
keeps meaning what a *write* is judged against, and the monthly ceiling that was paid for never
moves. It lands in `company_credit_budget.day_extra_cpu_used`.

`month_extra_cpu_used` is **not** a second ceiling — there is no monthly extra limit. It is a
correction term, and the reason it has to exist is subtle:

```
   cold start:  month_used = Σ(this month's daily usage rows)
                                    │
                                    └─ those rows include what the pool paid for,
                                       because a request served from the pool was
                                       still served and still lands in them
                                    │
                month_used = Σ(rows) − month_extra_cpu_used
                                       └─ without this, every restart quietly
                                          shrinks the entitlement
```

Nothing about the reply changes: a request served from the pool is answered exactly like one served
from quota, and the client cannot tell. The daemon logs it at `info` — that line is the only outward
sign a tenant is in read-only mode.

---

## 7. Where the numbers end up

An accepted charge increments **four** in-memory usage records, then the platform aggregate:

```
increment_usage:   {user, company-aggregate} × {five-minute frame, daily frame}
increment_platform_usage:  company 0, five-minute frame
```

Time frames are decimal-prefixed so one `int32` column carries both series, and the daily one is
bucketed by the **Lima business day (UTC-5)**, not the UTC day:

```
FIVE_MINUTE_PREFIX  100_000_000 + unix_seconds / 300
DAILY_PREFIX        200_000_000 + local_unix_day
```

That offset is load-bearing. On the raw UTC division the daily quota window and the daily usage row
it is recovered from disagree for the last five hours of every day, and the cap would reset at 19:00
local time. A fixed offset rather than a zone lookup, because Peru has no DST and both processes
must agree on the boundary from any host — including a Lambda that is always UTC.

Per-route totals are stored as a canonical compact blob:

```
header:u16 = (route_id << 2) | width_code      route 0..16383, width = width_code + 1 (1..4 bytes)
[cpu:width][inference:width]
```

Entries ascend by route, an all-zero route is omitted, and every value uses the narrowest width
that holds it. Those three rules make one set of totals have exactly one representation, so `decode`
can refuse anything else as corruption rather than guess.

**The flush is lossless, not periodic-overwrite.** Every record carries a mutation version and the
version last written. `flush_dirty` (every `flush_seconds`, default 15) writes only records whose two
versions differ, and a mutation that lands *during* a write leaves the record dirty rather than
being hidden by an older completed write. The same pass publishes each charged company's daily and
monthly counters into `company_credit_budget`.

> **The daemon must be a single active process.** Two instances would hold independent in-memory
> quota state and write the same absolute rows.

### The tables

| Table | Written by | Holds |
|---|---|---|
| `credit_usage_company` | daemon flush | per-company totals, per-route blob, both frames |
| `credit_usage_user` | daemon flush | per-user totals, both frames, no route split |
| `company_credit_budget` | **both** | entitlement (SaaS panel) + usage counters (daemon) |
| `credit_history` | backend | who granted what, which the balance alone cannot say |

`company_credit_budget` is the one row two writers touch, which is why there are **two** statements
for it: `upsert_budget` writes only the entitlement columns and `upsert_budget_usage` only the usage
ones. The two writes race — a flush and a panel mutation can land in either order — and each must
leave the other's columns untouched.

Its period columns (`usage_day_period`, `usage_month_start_day`) say which window each counter
belongs to. Nothing rewrites the row when a day or month rolls over with no traffic, so a reader
whose current window differs must read the counters as **zero**: a window the daemon has not touched
is unused.

### What the panel shows

`backend/config/company_credit_usage.go` subtracts the daemon's own flushed counters from the
ceilings rather than re-summing the usage rows. Re-summing would be a second implementation of the
same arithmetic, and the moment the two diverged the panel would advertise credit the limiter does
not grant. The cost is a lag of at most one flush interval.

One deliberate exception: the extra pool's figures are reported **even when there is no budget for
the current month**, skipping the early return the other counters take. That company is precisely
the one living on the pool, so hiding it would make read-only mode invisible exactly when it is in
use.

---

## 8. When things go wrong

| Situation | What happens | Why |
|---|---|---|
| Daemon unreachable | Go fails **closed** — no charge, no request | An unauthenticated decision cannot authorize API work |
| `status = 0xFF`, or malformed | Treated as unavailability, not as a verdict | A verdict nobody computed is not a verdict |
| `detail = 0` when access *was* asked | Treated as unavailability | Failing open would unauthorize every gated route on a version drift |
| ScyllaDB read fails mid-admission | Admission fails cleanly | The platform row is loaded *before* any counter is mutated, so nothing is charged half-way |
| Daemon restart | Counters recovered by summing usage rows | Up to one flush interval of usage is lost — the same window for every counter |
| Backend/daemon version mismatch | Every frame fails the HMAC | Domain separation (`:v6`) makes it loud instead of subtly wrong |
| No budget row for the company | **Everything refused** | `StoredBudget::default()` is `daily = 0`, and `exceeds(0, 2, 0)` is true. Nothing seeds this row at company creation — only the SaaS panel writes it. This is the most common "out of credit" state there is, and the extra pool is what makes it survivable |

---

## 9. Where the code is

| Concern | File |
|---|---|
| The gate, the catalogue, the tariff | `backend/main-handlers.go` — `enforceAccessAndCredits`, `resolveRouteAccess`, `chargedMethodFor`, `chargeGetResponseTopUp` |
| Frame encoding, tariff arithmetic, reply decoding | `backend/core/server_utils/credits.go` |
| Access invalidation (`0x06`) | `backend/core/server_utils/access_invalidation.go`, called from `backend/security/shared.go` |
| Packed grant construction | `backend/core/responses.go` — `MakeAccesoNivelPacked` |
| Wire codec | `server_utils/src/limiter/protocol.rs` |
| Grant cache, codec, verdict | `server_utils/src/limiter/access.rs` |
| The decision and every counter | `server_utils/src/limiter/quota.rs` — `admit_at` |
| Blob encoding | `server_utils/src/limiter/credits_blob.rs` |
| Time frames and the UTC-5 offset | `server_utils/src/limiter/time_frame.rs` |
| ScyllaDB statements | `server_utils/src/limiter/storage.rs` |
| Opcode dispatch, reply construction | `server_utils/src/service/server.rs`, `protocol.rs`, `auth.rs` |
| Panel reporting | `backend/config/company_credit_usage.go`, `company_credit_budget.go` |

### The tests that hold the contracts

Layout and arithmetic are written twice, once per language, so the pairs are what actually hold
them:

| Contract | Go | Rust |
|---|---|---|
| Frame byte offsets | `TestChargePayloadMatchesTheWireOffsets` | `parses_the_exact_wire_offsets` |
| Signed frame bytes | `TestChargeFrameMatchesTheRustAuthVector` | `matches_the_go_client_vectors` |
| Grant packing | `TestAccesoNivelPacking` | `packed` unit tests in `access.rs` |
| The extra-credit flag | `TestTheExtraCreditFlagRidesInTheRouteField` | `the_extra_credit_flag_is_stripped_from_the_route_number` |
| Only reads are marked | `TestOnlyReadsAreMarkedForTheExtraPool` | — (the daemon deliberately trusts the frame) |
| Frame width per opcode | `TestChargePayloadMatchesTheWireOffsets` (length) | `every_opcode_keeps_its_documented_width` |
| Four access slots suffice | `TestEveryRouteFitsTheRequiredAccessSlots` | — (the daemon has no catalogue to check against) |

`TestEveryRouteFitsTheRequiredAccessSlots` is the one that earns its keep the least often and matters
the most: adding a fifth access to one route's `backend_apis` fails the build instead of failing one
endpoint in production with a 500.

---

## 10. Design records

- [PLAN.md](PLAN.md) — the limiter design. Does not cover authorization, multiplexing or the extra
  pool; this document is the current shape.
- [PLAN_MULTIPLEXING.md](PLAN_MULTIPLEXING.md) — why replies carry a correlation id.
- [PLAN_OPTIMIZATION.md](PLAN_OPTIMIZATION.md) — memory and hot-path work: the three maps that
  never empty, the container costs, and what each change must not disturb.
- `docs/EXTRA_CREDITS_PLAN.md` — the extra pool: the alternatives weighed, what was rejected, and
  the three things that came out differently from the plan. **Written in Spanish.**
