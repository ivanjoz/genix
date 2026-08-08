# Rust Credit Rate Limiter Plan

Status: implemented, including Go-side size-to-credit formulas and TCP client integration.

Target: `server_utils/`

## 1. Goal

Build a standalone Rust service that accepts authenticated fixed-size messages over raw TCP,
enforces CPU and inference-credit limits in memory, and periodically persists usage summaries to
one ScyllaDB table.

The service must enforce separate limits for:

- The whole company.
- The individual user.
- Ten seconds.
- One hour.
- Twenty-four hours.
- CPU credits.
- Inference credits.

Accepted usage is aggregated by API group into five-minute and daily records. ScyllaDB is only the
persistence destination; CQL counter columns are not used.

## 2. Confirmed decisions

- Rust with Tokio and raw TCP.
- One TCP connection may carry multiple fixed-size request frames.
- The server sends a fresh eight-byte nonce when each connection opens.
- Incoming `company_id` and `user_id` are unsigned 24-bit positive IDs.
- `user_id = -1` never appears on the TCP wire. It exists only in memory and ScyllaDB as the
  company-wide aggregate.
- The request carries CPU and inference credits as separate unsigned 16-bit values.
- API groups occupy six bits in the persisted usage encoding.
- Usage counting happens in memory.
- Dirty usage records are flushed every 15 seconds.
- Usage is grouped into five-minute records and daily records.
- There is one ScyllaDB table. Limit policies are loaded from environment variables or
  `config.toml`.
- Persisted usage is written as absolute totals, not database-side increments.
- Time-frame type and period are packed into one positive decimal `int32`.

## 3. Service boundary

```text
# Purpose: Show the only runtime data path and keep policy/storage responsibilities explicit.
Genix Go backend
    │ persistent authenticated TCP
    ▼
Rust rate limiter
    ├── in-memory company/user quota state
    ├── in-memory five-minute/daily usage aggregation
    └── dirty absolute snapshots every 15 seconds
            │
            ▼
       ScyllaDB credit_usage
```

Version one supports one active rate-limiter process. Multiple active processes would have
independent quota state and could overwrite the same absolute database rows. Horizontal scaling
would require routing each company to one owner or adding a writer dimension to the schema; neither
is part of this plan.

## 4. TCP protocol

### 4.1 Connection handshake

Immediately after accepting a TCP connection, the server writes an unpredictable eight-byte nonce.
The client must read the complete nonce before sending its first frame.

The connection owns an implicit unsigned 64-bit frame sequence:

- It starts at zero.
- It is encoded big-endian when included in the HMAC input.
- It advances after every authenticated frame for which the server returns a response.
- An authentication failure closes the connection, so the peers cannot lose sequence alignment.

The nonce prevents replay across connections. The sequence prevents replay within a connection.

### 4.2 Request frame

Each request is exactly 19 bytes in network byte order:

| Offset | Size | Field | Rust representation |
|---:|---:|---|---|
| 0 | 3 | `company_id` | decoded into `u32` |
| 3 | 3 | `user_id` | decoded into `u32` |
| 6 | 1 | `api_group` | `u8` |
| 7 | 2 | `cpu_credits_used` | `u16` |
| 9 | 2 | `inference_credits_used` | `u16` |
| 11 | 8 | `auth_hash` | `u64` |

Tokio must use an exact 19-byte read. A clean EOF before the next frame ends the connection; a
partial frame, timeout, or invalid frame closes it.

Version-one validation:

- `company_id > 0`.
- `user_id > 0`.
- `api_group` is one of the configured six groups; the storage codec reserves values `0..63`.
- At least one of CPU or inference credits is greater than zero.
- Converting either ID to the internal `int32` representation must not overflow.

Recommended API group assignment:

| Value | Meaning |
|---:|---|
| 0 | GET, small response |
| 1 | GET, medium response |
| 2 | GET, large response |
| 3 | POST, small payload |
| 4 | POST, medium payload |
| 5 | POST, large payload |

The Go backend uses uncompressed binary KiB (`1 KiB = 1024 bytes`) and assigns groups as follows:

- Small: less than 32 KiB.
- Medium: 32 KiB through 256 KiB, inclusive.
- Large: more than 256 KiB.

GET uses the serialized response size and POST uses the request-body size. GET costs two CPU
credits for the first 8 KiB plus one credit per started 16 KiB above it. POST costs five CPU credits
for the first 8 KiB plus one credit per started 8 KiB above it. Successful inference calls cost one
credit per started 8 KiB of serialized provider input plus two credits per started 8 KiB of raw
provider output. The Rust service trusts these authenticated quantities and does not receive sizes.

### 4.3 Frame authentication

The 64-bit hash is the first eight bytes of a domain-separated HMAC-SHA256:

```text
# Purpose: Bind every frame to this protocol, connection, position, identity, and credit charge.
auth_hash = HMAC-SHA256(
    SECRET_PHRASE,
    "genix-rate-limiter:v1" || nonce || sequence_be || request_bytes[0..11]
)[0..8]
```

The client and server interpret the truncated eight bytes as a big-endian `u64`. The server compares
the received and expected hashes in constant time. Authentication failures are logged without the
secret or expected hash and close the connection without a detailed error response.

### 4.4 Response byte

The successful response is one byte:

- `0x00`: accepted and charged.
- Nonzero low five bits: rejected because a quota would be exceeded.

Quota rejection uses this bit layout:

| Bits | Meaning |
|---|---|
| 0 | Scope: `0` company, `1` user |
| 1-2 | Window: `00` ten seconds, `01` one hour, `10` twenty-four hours |
| 3 | Inference credits exhausted |
| 4 | CPU credits exhausted |
| 5-7 | Reserved and zero in version one |

At least one of bits 3 or 4 is set in every quota rejection. If both credit types exceed the same
selected constraint, both bits are set.

Malformed input, failed authentication, a read timeout, or an internal error closes the connection.
The Go client maps a disconnect/timeout to an unavailable rate-limiter error rather than confusing
it with a quota response.

One byte can report only one scope/window. Version one uses this deterministic priority:

1. Ten seconds, then one hour, then twenty-four hours.
2. Company before user within the same window.

## 5. Limit policy configuration

There is no policy table. Version one has a company policy applied to every company aggregate and a
user policy applied to every individual user.

Each policy has independent CPU and inference limits for the three windows, for 12 required values:

| TOML key under `[rate_limit]` | Environment override | Meaning |
|---|---|---|
| `company_cpu_10s` | `RATE_LIMIT_COMPANY_CPU_10S` | Company CPU capacity per ten seconds |
| `company_inference_10s` | `RATE_LIMIT_COMPANY_INFERENCE_10S` | Company inference capacity per ten seconds |
| `company_cpu_1h` | `RATE_LIMIT_COMPANY_CPU_1H` | Company CPU capacity per hour |
| `company_inference_1h` | `RATE_LIMIT_COMPANY_INFERENCE_1H` | Company inference capacity per hour |
| `company_cpu_24h` | `RATE_LIMIT_COMPANY_CPU_24H` | Company CPU capacity per 24 hours |
| `company_inference_24h` | `RATE_LIMIT_COMPANY_INFERENCE_24H` | Company inference capacity per 24 hours |
| `user_cpu_10s` | `RATE_LIMIT_USER_CPU_10S` | User CPU capacity per ten seconds |
| `user_inference_10s` | `RATE_LIMIT_USER_INFERENCE_10S` | User inference capacity per ten seconds |
| `user_cpu_1h` | `RATE_LIMIT_USER_CPU_1H` | User CPU capacity per hour |
| `user_inference_1h` | `RATE_LIMIT_USER_INFERENCE_1H` | User inference capacity per hour |
| `user_cpu_24h` | `RATE_LIMIT_USER_CPU_24H` | User CPU capacity per 24 hours |
| `user_inference_24h` | `RATE_LIMIT_USER_INFERENCE_24H` | User inference capacity per 24 hours |

Configuration precedence is:

1. Environment variable.
2. Matching lowercase key in `[rate_limit]` from `config.toml`, selected by `GENIX_CONFIG_FILE`.
3. Startup failure if a required value is absent or invalid.

The service also reads the existing `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, and
`SECRET_PHRASE` settings. It must never log their values.

If per-company or per-user-ID overrides are later required, they must be represented explicitly in
configuration. They must not introduce a second ScyllaDB table.

## 6. In-memory rate limiting

### 6.1 Ownership and concurrency

Requests are routed by `company_id` to one of a fixed number of mutex-protected shards. Holding one
shard lock therefore checks and commits the company and user portions of a request atomically
without a process-wide lock.

The shard computes prospective totals for both scopes and both credit types. A request is accepted
only when every configured constraint permits it. Equality is allowed: the request is rejected when
`current + requested > limit`.

If rejected, none of its rate-window or persistent-usage counters are changed.

### 6.2 Window semantics

Version one uses these semantics:

- Ten seconds: token bucket with capacity equal to the configured limit and continuous refill at
  `limit / 10 seconds`.
- One hour: fixed UTC hour identified by `unix_seconds / 3_600`.
- Twenty-four hours: fixed UTC Unix day identified by `unix_seconds / 86_400`.

CPU and inference maintain independent token/counter values. Company and user maintain independent
state but are evaluated in the same admission operation.

All arithmetic uses checked `u64`. Time comes from a monotonic clock for token refill and wall-clock
Unix time for hour/day boundaries.

## 7. Usage aggregation

Rate-window state and persisted usage aggregation are separate concerns. Every accepted request
increments these four logical usage records:

| Scope | Period |
|---|---|
| `company_id`, actual `user_id` | Current five-minute bucket |
| `company_id`, actual `user_id` | Current Unix day |
| `company_id`, `user_id = -1` | Current five-minute bucket |
| `company_id`, `user_id = -1` | Current Unix day |

Each record contains CPU and inference totals per API group. In-memory totals use `u64`, even though
the current wire charge is `u16` and the persisted codec is limited to `u32`.

The aggregation key is `(company_id: i32, user_id: i32, time_frame: i32)`. Each record also has a
monotonic in-memory mutation version and a dirty flag.

## 8. Packed `time_frame`

The time-frame type is a decimal prefix in a single positive `int32`:

| Range | Type | Decode |
|---|---|---|
| `100_000_000..199_999_999` | Five minutes | subtract `100_000_000` |
| `200_000_000..299_999_999` | Unix day | subtract `200_000_000` |

Encoding formulas:

```rust
// Purpose: Place five-minute and daily records in disjoint, human-readable int32 ranges.
const FIVE_MINUTE_PREFIX: i64 = 100_000_000;
const DAILY_PREFIX: i64 = 200_000_000;

let five_minute_time_frame = FIVE_MINUTE_PREFIX + unix_seconds / 60 / 5;
let daily_time_frame = DAILY_PREFIX + unix_seconds / 86_400;
```

Examples:

- Five-minute index `5_954_061` becomes `105_954_061`.
- Unix day `54_123` becomes `200_054_123`.

The implementation calculates in `i64`, validates the expected range, and only then converts to
`i32`. Buckets use UTC because they derive directly from Unix time.

## 9. `used_credits` blob

The blob is a concatenation of variable-width API-group entries with no entry count. Every entry is:

| Part | Size | Meaning |
|---|---:|---|
| Header | 1 byte | Six-bit API group plus two-bit value-width code |
| CPU credits | 1, 2, 3, or 4 bytes | Unsigned big-endian absolute total |
| Inference credits | Same width | Unsigned big-endian absolute total |

Header encoding:

```rust
// Purpose: Keep the API group in the high six bits and the shared integer width in the low two.
let header = (api_group << 2) | width_code;
```

Width codes:

| Low bits | Bytes per credit | Maximum value |
|---|---:|---:|
| `00` | 1 | `255` |
| `01` | 2 | `65_535` |
| `10` | 3 | `16_777_215` |
| `11` | 4 | `4_294_967_295` |

Canonical encoding rules:

- Pick the smallest width that can represent `max(cpu, inference)`.
- Encode both values with that same width.
- Use big-endian byte order.
- Emit API groups in ascending order.
- Emit at most one entry per API group.
- Omit a group only when both totals are zero.
- Reject duplicate groups, truncated values, or a decoded value that cannot fit the internal type.
- Refuse to flush and emit a critical log if either total exceeds `u32::MAX`; never wrap or
  silently saturate persisted usage.

Example for API group 3, CPU 300, inference 25:

```text
# Purpose: Demonstrate a two-byte shared width; header = (3 << 2) | 1 = 0x0D.
0D 01 2C 00 19
```

## 10. ScyllaDB table

The only table is `credit_usage`:

```sql
-- Purpose: Store one absolute compact usage snapshot per company, subject, and period.
CREATE TABLE credit_usage (
    company_id int,
    user_id int,
    time_frame int,
    used_credits blob,
    PRIMARY KEY ((company_id), user_id, time_frame)
);
```

The Genix schema declaration contains paired `CreditUsage` and `CreditUsageTable` structs:

| Go field | Go type | ORM table field |
|---|---|---|
| `CompanyID` | `int32` | `db.Col[CreditUsageTable, int32]` |
| `UserID` | `int32` | `db.Col[CreditUsageTable, int32]` |
| `TimeFrame` | `int32` | `db.Col[CreditUsageTable, int32]` |
| `UsedCredits` | `[]byte` | `db.Col[CreditUsageTable, []byte]` |

`GetSchema()` uses `CompanyID` as the partition and `db.Cols(UserID, TimeFrame)` as clustering keys.
It uses stable `TableSchema.ID` 42. It does not enable delta
views, indexes, autoincrement, or updated-version tracking.

The generated registry and static schema checks include this table.

### Partition growth

The requested schema keeps all historical rows for a company in one partition. Five-minute data
adds up to 288 rows per user per day, plus the `user_id = -1` aggregate. Version one applies no TTL;
retention remains an operational decision. Different TTLs can be applied per write while keeping
one table.

## 11. Dirty-only flush

A single flush coordinator runs every 15 seconds:

1. Lock each shard briefly and copy snapshots of dirty aggregation records.
2. Encode each record's complete absolute per-group totals.
3. Upsert only those keys with a mutation since their last successful flush.
4. Use one prepared statement initialized once and reused.
5. Write snapshots sequentially and avoid cross-partition batches.
6. Mark a record clean only when the write succeeded and its mutation version still equals the
   snapshotted version. If it changed during the write, keep it dirty for the next cycle.
7. On failure, retain the dirty state and retry a later cycle with structured error logging.

Flush cycles are serialized. An older snapshot is never allowed to complete after a newer snapshot
for the same row. Absolute upserts are therefore safe to retry and do not have the ambiguous retry
behavior of database counter increments.

Graceful shutdown stops accepting connections, waits for connection tasks, and attempts one final
flush.

## 12. Startup and recovery

Because writes replace an absolute blob, the first access to an existing current period must load
its persisted value before incrementing it. A zero-valued process must never overwrite a nonzero
database row after restart.

Implemented lazy recovery:

- On first company/user activity, load the current daily rows for the user and company aggregate.
- Load the five-minute rows required to rebuild the current hour if fixed-hour enforcement is
  confirmed.
- Decode and validate every blob before admitting the first request for that state owner.
- Cache the initialized state in its mutex-protected shard.
- Deduplicate simultaneous initialization so only one load occurs per company/user.

The exact ten-second token-bucket position cannot be reconstructed from five-minute summaries. On
restart, version one starts that bucket full unless a durable local journal is added.

Without a local journal, a process or machine crash may lose accepted increments since the last
successful 15-second flush. Normal shutdown attempts a final flush, but it cannot protect against
power loss. Version one documents and accepts this tradeoff.

Initial state loading fails closed when ScyllaDB is unavailable or contains a corrupt usage blob.

## 13. Logging and observability

Use structured `tracing` logs with concise debug events for:

- Connection accepted/closed and reason, without secrets or full hashes.
- Frame validation failure.
- Authentication failure.
- Admission decision with company/user IDs, API group, credit amounts, selected limit, and response
  code.
- State initialization and decoded row count.
- Flush start/end, dirty row count, duration, and failure count.
- Blob overflow or corruption.
- Graceful shutdown progress.

Per-request success logging is debug-level so production can disable its volume. Authentication
secrets, nonces, expected hashes, and database passwords must never be logged.

## 14. Project layout

```text
# Purpose: Keep protocol, business rules, persistence, and process wiring independently testable.
server_utils/
├── Cargo.toml
├── README.md
├── PLAN.md
├── src/
│   ├── main.rs
│   ├── config.rs
│   ├── protocol.rs
│   ├── auth.rs
│   ├── limiter.rs
│   ├── aggregation.rs
│   ├── credits_blob.rs
│   ├── time_frame.rs
│   └── storage.rs
```

The Go backend still needs a matching client package with the same frame, HMAC, sequence,
response-bit, and test-vector definitions.

## 15. Implementation phases

### Phase 1: Behavior decisions

- [x] Use token-bucket ten-second limits and fixed UTC hour/day limits.
- [x] Prioritize shorter windows, then company scope.
- [x] Use no TTL in version one and document the crash-loss window.
- [x] Fail closed during database recovery.
- [x] Define GET/POST size thresholds and credit formulas in the Go backend.

### Phase 2: Protocol and configuration

- [x] Scaffold the Rust crate.
- [x] Load and validate environment/`config.toml` settings.
- [x] Implement the 24-bit codec, nonce handshake, sequence-bound HMAC, fixed-frame parser, and response
  codec.
- [x] Add Go protocol-frame/HMAC coverage alongside the Rust protocol tests.

### Phase 3: In-memory limiter

- [x] Implement company-owned mutex shards.
- [x] Implement CPU/inference and company/user checks.
- [x] Implement the three windows with test-injectable clocks.
- [x] Make admission and accounting one atomic shard operation.

### Phase 4: Aggregation and persistence

- [x] Implement decimal time-frame keys.
- [x] Implement the canonical blob codec.
- [x] Implement the four-record increment per accepted request.
- [x] Add Scylla connection setup, lazy state recovery, prepared absolute upserts, dirty versions, and
  graceful final flush.

### Phase 5: Genix integration

- [x] Add the paired Go ORM table declaration with a unique schema ID.
- [x] Run static table validation.
- [x] Add the persistent Go TCP client and backend call sites.
- [x] Map exhaustion bits to HTTP 429 and expose the raw byte in `X-Rate-Limit-Code`.

### Phase 6: Deployment and load validation

- [x] Add README configuration and operations documentation.
- [ ] Add deployment-script integration; the README currently provides the hardened systemd unit.
- [ ] Measure throughput, p50/p95/p99 admission latency, memory per active user, shard contention, and
  Scylla flush load.

## 16. Test matrix

The crate currently has unit coverage for the wire offsets, response encoding, nonce/sequence HMAC
binding, time-frame examples, blob width boundaries/canonical validation, dirty-version safety,
exact-limit rejection, four-row aggregation, and dirty-only absolute flushing. The remaining items
below are integration and load-validation work.

### Protocol

- Every integer boundary for unsigned 24-, 16-, and 64-bit fields.
- Partial frames, EOF, timeout, invalid IDs, invalid API groups, and zero-credit requests.
- Nonce and sequence replay rejection.
- HMAC vectors shared byte-for-byte with Go.
- Every response-bit combination used by version one.

### Rate limiting

- Exact-limit acceptance and one-credit-over rejection.
- Independent CPU/inference exhaustion.
- Company and user interactions.
- Window boundary behavior with a deterministic clock.
- Concurrent requests cannot exceed a quota through races.
- Rejected requests do not mutate any counter.

### Time-frame and blob codecs

- `5_954_061 -> 105_954_061`.
- `54_123 -> 200_054_123`.
- Prefix/range rejection and checked `int32` conversion.
- Width transitions at 255, 65,535, and 16,777,215.
- Maximum `u32`, overflow, truncation, duplicate group, ordering, and round-trip tests.

### Persistence

- Only dirty rows are written.
- One accepted request dirties exactly four logical rows.
- A mutation during a flush remains dirty afterward.
- A failed absolute upsert can be retried without double counting.
- Restart loads existing absolute totals before incrementing.
- Corrupt blobs fail initialization without being overwritten.
- Graceful shutdown attempts the final flush.

### Load and resource safety

- Many connections and many frames per connection.
- Hot company with many concurrent users.
- Slow/partial TCP clients cannot hold unlimited resources.
- Scylla outage does not lose dirty in-memory records while the process remains alive.
- Bounded actor and storage queues apply backpressure rather than growing without limit.

## 17. Acceptance criteria

- A valid authenticated frame gets exactly one one-byte decision.
- Company and user checks are atomic for every accepted frame.
- No accepted request can race past an in-memory limit.
- Every accepted request is represented in both user/company and five-minute/daily in-memory totals.
- Every successful Scylla write is an absolute, canonical, decodable snapshot.
- The 15-second flush writes only records changed since their last successful write.
- Restart never overwrites an existing current-period total with an uninitialized zero value.
- Rust and Go agree on all protocol, HMAC, response, time-frame, and blob test vectors.
- The single table passes Genix static schema validation.

## 18. Optional operational follow-ups

1. Choose retention if five-minute rows should expire; version one writes no TTL.
2. Add a local journal if losing usage since the last 15-second flush on an abrupt machine failure
   is unacceptable.
