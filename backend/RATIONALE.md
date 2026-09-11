## A delta watermark is read through `GetUpVersion` / `GetUpdated`, never `GetQueryInt`

**Context** — the client now sends both watermarks of a response key in one param, `"<upv>.<upd>"`,
so the handler picks the half its table is keyed on instead of the client guessing which one to
send. Every handler read that param with `req.GetQueryInt("upv")` or `req.GetQueryInt("Frames")`,
which returns 0 for a value with a dot in it — silently, as a first sync.

**Decision** — Two readers on `HandlerArgs`, `GetUpVersion(responseKey ...string)` and
`GetUpdated(responseKey ...string)`, the variadic key naming the param on a multi-table route and
its absence meaning the single-array param `up`. Every watermark call site moved to them.

**Rationale** — Naming the two halves in the reader is what makes a handler's choice greppable:
`GetUpVersion()` says this table has a delta index, `GetUpdated()` says it is keyed on the
timestamp, and neither can be confused with an ordinary query param again. A single
`GetWatermark() (int32, int32)` would have been one function instead of two, but every call site
would then discard one half at the call site, which reads as if the handler had a choice to make.
A value with no dot is read as a bare `upv`, so a URL typed by hand still works.

## The local API queue lives in `LocalHandler` and excludes by path suffix

**Context** — `disable_api_concurrency_local` serializes the standalone server so one request's logs
are never cut in half by another's. Two shapes must stay outside that queue or the flag turns into a
hang: SSE routes, which hold the connection until the client leaves, and the agent turn, which waits
on an LLM.

**Decision** — A single `sync.Mutex` taken at the top of `LocalHandler`, skipped when the path ends
in `-stream` or `agent-turn`.

**Rationale** — `LocalHandler` is mounted at `/`, and `/agent/stream`, `/agent/in` and `/agent` have
their own mux entries, so the browser's permanent event stream is already outside the queue without
naming it. The suffix test is used instead of a list of route names because the naming convention is
what the exclusion is really about — a future `*-stream` route is excluded the day it is written,
and a list would have to be remembered. It costs the ability to queue a route that happens to end in
those words, which is precisely the route that should not be queued.

## The dev handler line carries the query string, and `GET.products` prints its watermark

**Context** — A delta sync that arrives without its `upv` watermark and a client that has nothing
cached both produce the same terminal output: `Ejecutando Handler:: GET.products` followed by
`Rows Scanned 10010`. There was no way to tell, from the backend alone, whether the frontend had
sent a watermark at all — which is exactly the question when a delta route looks like it is
re-sending everything.

**Decision** — `formatDevQueryParams` appends the sorted query string to the existing
`Ejecutando Handler::` line, gated on `core.Env.IS_DEV_ARG`. `GetProducts` additionally logs the
`upv` it received next to the row count it returned.

**Rationale** — The handler line is printed for every request already, so the diagnostic costs one
string join and no new line. It is dev-only because in serverless every line is billed and query
params carry client-supplied values that do not belong in CloudWatch. The `GET.products` line
duplicates part of that on purpose: it pairs cause with effect in one line, so the ORM flag is only
needed when the answer is "the watermark did arrive and the query still scanned everything".

## ResetCounter does nothing now, and warns instead

**Context** — `ResetCounter` realigned a table's sequence with the rows that exist, and ran after
every restore. It was wrong in three ways, and the id rework made the third total.

**Decision** — `resetCounterForTable` in genix-orm is a no-op returning nil. `exec/restore.go`'s
`ResetCounters` prints one warning naming what was skipped, and both carry a TODO describing what a
correct version needs. `applyCounterReset` and opcode `0x08` are untouched.

**Rationale** — It was only ever right for a key that is a bare `Autoincrement(0)`. The counter name
hardcoded the autoincrement part as `0`, so a table declaring `AutoincrementPart` had its real
per-part counters missed and a phantom one written instead; and the target was `max(key)`, which packs
the counter with random digits and any `KeyIntPacking` columns, so the figure was orders of magnitude
too large. Since sale and document ids became caller-built, it also skips both tables anyone would
actually want to realign. A function that confidently writes a wrong value into a sequence the
allocator owns is worse than one that admits it does not work.

The failure this leaves is the mild direction: counters keep climbing, so an id is skipped, never
reused. The case it does not survive is a restore into a *fresh* keyspace, where every counter starts
at zero while the restored rows carry high ids — the next insert then collides with one of them. That
is what the warning says.

A real implementation has to reach further than the old one did. Sale ids and invoice correlativos are
minted by this project under their own counter names through `db.GetAutoincrementID`, so no
table-driven walk can discover them; they have to be reset by name. `applyCounterReset` is the
primitive, and it already routes through the allocator so a live reservation is dropped in the same
critical section as the move — which is the reason `0x08 SetSequence` exists at all.

## `db/autoincrement.go` wires the ORM to fareward in package init

**Context** — The daemon has to be the *only* writer of the `sequences` counters: it claims ranges
in advance, so anything still on the ORM's read-then-increment would hand out ids from inside a
range the daemon already owns. That is a silently overwritten record. The backend reaches the ORM
from several places — `main.go`'s `SetScyllaConnection` and a handful of `exec/*.go`
`MakeScyllaConnection` calls — and `exec/init.go` even calls `db.GetAutoincrementID` directly.

**Decision** — A new `db/autoincrement.go` sets `scylla.ReserveCounterRange` and
`scylla.SetCounterValue` in package `init()`, alongside `db/driver.go` which already declares the
project's database choice. No configuration switch anywhere: in Genix the ORM always reserves
through fareward. The keyspace argument is dropped, since the daemon writes the one it is configured
against. Both hooks go in together — the second is what routes `ResetCounter` (reached from
`RestoreBackup`, a live handler) through the daemon, so it can drop the block it derived from the
value being erased instead of being left serving ids from a range that no longer exists.

**Rationale** — Wiring it beside each `SetScyllaConnection` call would have made "forgetting a line"
produce duplicate primary keys, which is exactly the failure this feature exists to remove; an
`init()` in the one package every module already imports makes the unwired state unreachable. It
buys that at the cost of implicit setup — `init()` is not top-to-bottom readable, so the file is
named for what it does and carries the reasoning. Verified that `backend` is the only binary that
writes through the ORM: `scripts/validation` names the ORM path only as a string in AST analysis,
and `cloud`/`db-backup` do not link it. The remaining consequence is that a write attempted before
`ConfigureFareward` fails with "not configured" rather than falling back — deliberate, and loud.

## A tolerated credit refusal retries as an authorize-only frame

**Context** — The operator company (ID 1) now sends real credit charges, so the daemon can refuse
it, and that refusal must not become the HTTP response. Swallowing the error alone is not enough:
the daemon returns `CreditViolation` *before* it returns the access verdict, so a tolerated refusal
leaves `accessGrant` nil and `args.User.SubAccesos` empty — the operator's sub-accesses would turn
off precisely when its budget ran out, which is unreadable as a symptom.

**Decision** — In `enforceAccessAndCredits`, a refusal that `core.TolerateCreditRefusal` accepts is
followed by a second `core.ChargeAPIAccessOnly` frame, and only its error can produce a response.
`chargeGetResponseTopUp` tolerates without retrying — the settlement carries no accesses, so there
is no grant to recover. An `AccessDenied` is never tolerated in either place.

**Rationale** — The retry is safe rather than merely likely to work: a zero-credit frame cannot be
refused on quota, because the gate compares `current + requested > limit` and a refusal charges
nothing, so accumulated usage never gets past the ceiling. Alternative was widening the daemon's
reply so `CreditViolation` also carries the verdict — a protocol change on both ends to save a
round trip that only happens when the operator is already over budget. What this costs: two frames
in that degraded case, and the tolerated request itself goes uncounted.

## A dev backend binds loopback and the tailnet, not every interface

**Context** — The dev backend listened on `*:14010`. It holds the production database credentials
and has no nginx and no firewall rule in front of it, so every interface meant the LAN and anything
forwarded to this host as well. It showed up as bursts of rejected requests for routes the app does
not serve — `/dana-na`, `/global-protect/prelogin.esp`, `/remote/login`, `/sslvpnclient`, the
standard VPN-appliance scanner sweep — each one writing a row into the request log, which for a dev
run is the *production* Scylla.

**Decision** — `resolveDevListenAddresses` returns `127.0.0.1:<port>` plus every interface address
inside 100.64.0.0/10, and a dev launch opens one `net.Listen` per address sharing a single
`http.Server`. A deployed binary is untouched and still calls `ListenAndServe` on `:<port>`.

**Rationale** — Those two addresses are exactly the reachability `serve_tailscale` needs: the machine
itself, and a browser on another tailnet node. Matching on the CGNAT block rather than on an
interface named `tailscale0` keeps it correct on macOS `utun` and on the userspace daemon. Narrowing
the *deployed* listener is a deploy decision — nginx, the health check and the Function URL each
arrive by a different address — so it is gated on `IS_DEV_ARG` rather than applied to both. A
listener that fails to open is logged by address and skipped instead of being fatal: losing the
tailnet address still leaves a working loopback backend. Verified live: `ss` shows `127.0.0.1:14010`
and `100.64.0.2:14010` and no longer `*:14010`.

## The route log line carries the caller's address

**Context** — `core.Log("Route:", args.Route)` printed the route alone, so a burst of rejected
requests read as an application error with no way to tell where it came from. `args.ClientIP` was
already resolved three lines earlier and simply not used.

**Decision** — The line is now `Route: <route> clientIP: <ip>`.

**Rationale** — One field, already computed, that turns "the app is failing" into "someone is
scanning this port". The cost is a slightly longer line on every request.

## `is_local` is gone: the dev launch argument is the only "this is a development machine" signal

**Context** — `is_local` was a root key in `config.toml`, and it gated things that must never be
live in a deployment: verbose logs, the agent prompt log, `DevLogin`'s password-less session mint,
and (as of this change) the plaintext `UserInfo`. A config key is exactly the wrong shape for that
guarantee — the value ships with the deploy, and one stale `is_local = true` in a server's config
silently turns every one of those on. The trigger was the plaintext `UserInfo`: gating a new hole on
`is_local` would have widened what a single wrong config line costs.

**Decision** — Removed `is_local` from `config.toml`, `config.example.toml` and `fileConfig`, and
deleted `Env.IS_LOCAL`. `core.ReadDevArgument()` scans `os.Args` for `dev` and `PopulateVariables`
assigns the result to `Env.IS_DEV_ARG` *before* `applyToEnv`, so nothing in the file can set or clear
it. Every former `IS_LOCAL` reader now reads `IS_DEV_ARG`: `main.go` (LOGS_FULL, the cron seed),
`main-handlers.go` (local usage accounting), `agent/prompt_log.go`, `agent/pagebuilder/loop_log.go`,
`makeFarewardAddress` and `DevLogin`.

**Rationale** — `start.js` is the only launcher that passes the argument
(`BACKEND_GO_SCRIPT = "go run . dev"`); the systemd unit is `ExecStart={SERVICE_BINARY_PATH}` with no
arguments and Lambda passes none, so a compiled binary cannot be talked into a dev relaxation by its
configuration no matter what it was deployed with. The name keeps the `_ARG` suffix at every call
site on purpose — the provenance *is* the security property. `DevLogin` keeps its second, independent
loopback check: `serve_tailscale` makes a dev backend reachable from other machines, so the argument
alone is not enough for a password-less session mint. Cost: the flag is invisible to `config.toml`,
so someone who runs the backend by hand (`go run .`) gets production behaviour and has to know to add
`dev`; the startup log line `[core.config] config_parsed is_dev_arg=%t` is there to make that legible.

## `MakeCipherKey` hung the process on an empty `secret_phrase`

**Context** — Found while unit-testing `MakeUsuarioResponse`: the test froze until the Go test
timeout fired. `MakeCipherKey` built its key with `for len(key) < 32 { key += Env.SECRET_PHRASE }`,
which never terminates when the phrase is empty. `Encrypt` called it unconditionally, *before*
honouring an explicit key argument, so a backend with no `secret_phrase` hung on its first encrypt
with no error and no log. Two neighbouring defects: `Encrypt` did `cypherKey_[0][:32]`, which panics
on a key shorter than 32 — and the keys come from clients — while `Decrypt` sliced
`MakeCipherKey()`'s result instead of the argument, silently ignoring any explicit key it was given.

**Decision** — `MakeCipherKey` returns `""` for an empty phrase. Both entry points now go through
`resolveCipherKey`, which picks the explicit key over the default and returns an error for anything
under 32 characters rather than slicing it.

**Rationale** — A hang is the worst of the available failure modes: no stack, no log, and a stuck
goroutine that looks like a slow database. An error naming the length is diagnosable. The `Decrypt`
fix is behaviour-changing in principle, but no caller in the repo passes it an explicit key — every
one relies on the `SECRET_PHRASE` default — so no stored ciphertext (the invoicing company secrets)
changes meaning.

## An insecure origin has no WebCrypto, so an empty CipherKey asks for the UserInfo in clear

**Context** — `serve_tailscale` hands the dev app out at `http://100.x.y.z:3572`. That origin is not
a secure context (only https and loopback are), so the browser never defines `crypto.subtle`, and
`parseLogin` died on `Cannot read properties of undefined (reading 'importKey')` — the login POST
had already succeeded, so the token and the access blobs were in hand and only the AES-GCM decrypt
of `UserInfo` failed. Serving the dev app over real HTTPS would need `tailscale serve`, tailnet
certs, and the Go API moved behind the same origin to dodge mixed content: a lot of machinery for a
dev-only convenience.

**Decision** — The client decides. `makeCipherKey` (`frontend/services/login.ts`) returns `''` when
`crypto.subtle` is absent, and `MakeUsuarioResponse` reads an empty key as "this browser cannot
decrypt": on a dev launch it answers `UserInfoPlain` (the same JSON, unciphered) instead of
`UserInfo`, and otherwise errors exactly as before. The per-caller `CipherKey` checks in `PostLogin`,
`PostSignUpCompany` and `DevLogin` were removed or relaxed so the rule lives in the one function all
four login paths already funnel through.

**Rationale** — Keying the fallback on the *client's* capability rather than on the environment alone
keeps localhost dev exercising the real encrypt/decrypt path, so a break in it still surfaces before
production. `UserInfo` was never a trust boundary anyway: the client generates `CipherKey` and sends
it in the request body in clear, so anyone who can read the response can read the key — the real
session credential is `UserToken`, which is unaffected. The cost is a second response shape that only
a dev backend can emit. `makeCipherKey` also replaced the hardcoded `"12341234..."` key the login had
been sending, and merged the duplicate copy in `RegistrationModal.svelte`.

## A partial write to `users` must name `status`, because the delta view keys on it

**Context** — `recompute_user_accesos` and `PostPerfiles` both rewrite only the two grant blobs on a
user row, naming exactly those columns in `db.Update` so a concurrent edit of an unrelated field
cannot be lost. Both panicked on the first real run: `Table "users": A composit index/view requires
the columns "status", "updated" be updated together. Not Included: status`. `UserTable.GetSchema`
declares `{Type: db.TypeView, Keys: db.Cols(Status, Updated.DecimalSize(10))}` and the ORM assigns
`updated` on every write, so a write that touches `updated` without `status` cannot maintain that
view's key.

**Decision** — Both call sites name `usuarioQuery.Status` alongside the two blob columns. The value
written is the one the read returned, so nothing about the row's meaning changes.

**Rationale** — The alternative readings were both worse. Dropping the view is not on the table: it
is the delta read the whole user cache depends on. Widening the update to the full row is what the
narrow column list exists to avoid, and would reintroduce the lost-update it was written to prevent.
Naming `status` is the minimum that satisfies the view, and it is not a new coupling -- the ORM has
always required a composite view's key columns to move together. The cost is that any future partial
write to `users` has to remember the same thing; the error message says so explicitly, which is why
this is a note rather than a helper. Worth recording that `PostPerfiles` carried this bug in the
live profile-save path, not only in the one-off migration script: no test caught it because both are
integration paths against a real view, and the migration had never been run.

## A user's grants live in two big-endian byte columns, split by whether they carry sub-accesses

**Context** — Sub-accesses are up to 13 flags an access may declare, and a user's grants had to
carry them. `users.accesos_computed` was a `[]uint16` of `accesoID<<2 | nivel-1`, one word per
access, with no room for a mask. Widening the word to `uint32` was the obvious move; the format is
read by three separate processes (this backend, the Rust daemon, and the browser), so whatever it
became had to be identical in all three.

**Decision** — Two `[]byte` columns, both big-endian, both sorted ascending by accesoID, sharing one
16-bit grant word `[14 bits accesoID][2 bits nivel-1]`:

- `accesos_computed` — accesses with **no** granted sub-access. Grant words only, fixed 2-byte
  stride, binary searchable.
- `accesos_sub_computed` — accesses with **at least one**. Every grant word is followed by sub bytes
  of `[1 bit MORE][7 bits flags]`.

`core/accesos-blob.go` is the only encoder in Go; nothing else may write these bytes.

**Rationale** — `[]byte` over a wider integer slice because `genix-orm/scylla/converter.go` takes a
`reflect.Copy` fast path for `[]uint8` in both directions while `[]uint16` pays a per-element
`binary.LittleEndian` loop, and because the Rust side reads the column into a `Vec<u8>` straight out
of Scylla — with `[]byte` both processes hold the identical bytes and the conversion disappears.
Neither CQL type is new (both map to `blob`), so `accesos_computed` needed no `ALTER TABLE`, only a
data rebuild.

**Two columns rather than one self-delimiting blob, because the column name carries the bit for
free.** The alternative spent one bit of the grant word on a `SUB_FOLLOWS` flag, dropping the id
ceiling from 16383 to 8191 and making the whole array variable-width. Here, a sub byte always
follows in `accesos_sub_computed` — that is the *definition* of the column — so nothing encodes it,
both arrays keep 14-bit ids, and the common array keeps a fixed stride and a single-modulo length
check. The variable half is the small one by construction: an access appears there only when a
profile actually granted it a sub-access.

The accepted cost is that **an access lives in exactly one column**, so every check that misses the
first must scan the second. It fails closed — a missed second lookup denies a user something they
hold — but it has to be right in Go, Rust and TypeScript. Each side has a test for exactly that
lookup. The performance question that prompted the split is empty either way and is written down so
it is not reopened: a user's grants are ~110 bytes total, and the daemon's check already sits behind
a per-shard mutex and a `HashMap` lookup, each costing more than scanning every byte of both arrays.

**Big-endian** because `access.rs` carried a standing warning that this column was the one
little-endian integer in a daemon whose entire wire protocol is big-endian, and that reading it
backwards would not fail — it would silently authorize the wrong things. The bytes are ours to
choose and the ORM only copies them, so one endianness everywhere deletes the exception, and sorting
by raw `u16` becomes sorting by accesoID.

**What replaced the defensive re-sort.** `decode_grants` used to sort and dedup on load, so an
out-of-order blob degraded into a wrong answer for one user rather than a broken binary search. That
cannot survive on a variable-width array where position is load-bearing. Instead every reader
validates while it walks — ascending ids, whole words, terminated sub runs, no empty mask — and
rejects the blob loudly. Free, since the parser is already walking; same intent, better failure
mode; and it also catches the corruptions the old defense could not name.

## The access catalog is TOML, and sub-accesses are two parallel arrays

**Context** — `backend/access_list.yml` was the single source of truth for authorization, embedded
by the backend and imported as *text* by the frontend, which parses it with a hand-written parser
because dragging a YAML library into the bundle for one file is not worth it. That parser had to
track indentation and strip `- ` prefixes to know where a record began.

**Decision** — `backend/access.toml`, sections renamed `access_list` → `access` and `access_groups`
→ `groups`. `github.com/pelletier/go-toml/v2` was already a direct dependency, so nothing new was
added. Sub-accesses are declared as two parallel single-line arrays, `sub_accesses_ids` and
`sub_accesses_names`. The file was generated from the YAML and asserted equal to it field by field —
9 groups, 36 access entries — before the old one was deleted.

**Rationale** — The win is the frontend parser: `[[access]]` is unambiguous, so indentation stops
mattering and records self-delimit. It gets shorter, and its remaining failure modes (a multi-line
array, a trailing comma) are rejected by name rather than surfacing as a `SyntaxError`.

Parallel arrays have one hazard — two lists drifting out of step mis-name every entry after the gap —
so **load refuses rather than repairs**: a length mismatch, an id outside 2..13, a duplicate, or a
blank name aborts the load naming the offending access. The catalog is embedded at build time, so
a malformed declaration is a build-time mistake and must surface as one. Id 1 is reserved for
"Todos": never declared, satisfies every sub-access check on its access, and costs one byte in the
blob because it is bit 0.

## A profile stores sub-accesses readably; the binary packing happens once

**Context** — Sub-accesses had to be editable, and the thing being edited is a profile. Storing the
grant blob on the profile would have put the binary format in a second place, and one a human reads.

**Decision** — `profiles.sub_accesos` is a `[]int32` of `accesoID*100 + subID`. The frontend sends
that shape, `PostPerfiles` validates every entry against the embedded catalog, and the blob is packed
exactly once — in `core/accesos-blob.go`, when a *user's* grants are computed from their profiles.

**Rationale** — The profile is the human-facing record: `1002` is legible in a database console and
in a log line, and a binary column there would need a decoder to answer "what does this profile
grant". `buildAccesosComputedFromPerfiles` merges in **two passes** over the profiles, every access
first and then every sub-access, because a sub-access granted by one profile may qualify an access
granted by another and a single interleaved pass would drop those. `PostPerfiles` compares **both**
grant columns before deciding an affected user is unchanged; comparing only the first would skip the
write for an edit that adds nothing but a sub-access, leaving every affected user stale.

Validation rejects rather than drops: a sub-access the catalog never declared, one on an access that
does not exist, and one granted without its parent access. A profile silently saved without what
the operator just ticked is worse than an error.

**Scope, stated so it is a decision and not a gap.** Sub-accesses are a *profile* concept. A user
may also be granted an access directly through `AccessLevelIDs`, and that path produces an empty
sub-mask — so an access whose sub-accesses matter must be granted through a profile. Extending
direct grants is deliberately deferred, not overlooked.

## The gate translates fareward's slots into access ids, and only for the current request

**Context** — The daemon answers authorization in *slots* over the `requiredAccess` list the gate
sent it, because it holds no copy of `access.toml` and cannot know which access a slot stands for.
Something had to turn slots back into ids, and handlers had to be able to ask.

**Decision** — `mapGrantedSubAccesos` does the translation in `main-handlers.go` and sets the result
on the **per-request** `args.User`. `core.UsuarioToken.SubAccesos` is tagged `cb:"-"`, so it never
rides in the browser-held session token. Handlers call
`req.User.HasSubAcceso(accesoID, subAccesoID)`.

**Rationale** — The translation belongs on the side that embeds the catalog; that is the same reason
the daemon returns a raw mask and "id 1 means all" is expanded here. It goes on `args.User` and never
on the `core.User` global, which is only assigned under `IS_SERVERLESS` precisely because local and
VPS mode are concurrent. `cb:"-"` because the map is a statement about one request, not an identity:
serializing it would both bloat the token and let a stale copy answer a later request.

Two consequences worth naming. A granted access with no sub-accesses still enters the map with an
empty mask, because "holds none of them" and "that access did not authorize this route at all" are
different answers and only the presence of the key states it. And the **scope limit**: a handler can
read the sub-accesses of the accesses that gated *its own route*, and nothing else — that is all the
reply carries. Reading an unrelated access's flags would need a different transport.

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

