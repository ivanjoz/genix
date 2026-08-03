# Plan — Cut peak memory in the response path

Status: **stages 0-4 complete.** Brotli dropped from the repo; zstd is now the primary codec with
gzip for compatibility. Measured results at the bottom.

## Context

`core/responses.go:488` `MakeResponse` is now a single path: `serialize.Marshal(respStruct)` → in-memory
compress → base64 → API Gateway. The `/tmp` disk branch is gone; it never avoided the allocation it was
written for, because the compressed bytes were read back and base64'd into a string anyway.

The remaining cost is real and it is concentrated in the JSON conversion, exactly as suspected.

## Where the memory actually goes

For a response of N records × F fields, one call to `serialize.Marshal` (`serialize/marshal.go:34`) does:

| # | Step | Allocates |
|---|------|-----------|
| 1 | `ResetUsedFlags()` (`marshal.go:36`) | mutates the process-global `globalRegistry` |
| 2 | **Pass 1** `marshalContent(v)` (`marshal.go:40`) | full `[]any` tree — **built, then discarded** |
| 3 | `ComputeOptimizedOrder()` (`marshal.go:46`) | per-type sort |
| 4 | **Pass 2** `marshalContent(v)` (`marshal.go:52`) | the `[]any` tree again — this one kept |
| 5 | `sonic.Marshal(result)` (`marshal.go:60`) | reflects over the tree → final `[]byte` |

Then `MakeResponseFinal` (`core/responses.go:104`) adds:

| # | Step | Allocates |
|---|------|-----------|
| 6 | brotli/gzip into `bytes.Buffer` | full compressed copy |
| 7 | `base64.StdEncoding.EncodeToString` | **string copy, +33%** |
| 8 | `aws-lambda-go` marshals the response struct | copies the base64 string again, with JSON escaping |

Two observations that drive the whole plan:

- **The `[]any` tree is built twice and is pure garbage.** Every scalar is boxed into an `any` (a heap
  allocation for most types), every record is a `[]any`. Steps 2 and 4 are the dominant allocator, and
  step 2's output is thrown away entirely.
- **Steps 5–8 copy the whole payload four more times.** Peak RSS is roughly
  `tree + json + compressed + base64 + lambda buffer` all live at once.

## Staged plan

Each stage is independently shippable and independently revertible. Stage 0 gates everything else.

### Stage 0 — Measure before changing anything

`serialize/` has **no benchmarks today** (`marshal_test.go`, `skipblock_test.go` are correctness only).
Every number above is read off the code, not measured. Optimising without a baseline is how you spend a
week on step 3 and miss that step 7 dominates.

- Add `serialize/marshal_bench_test.go` with two realistic payloads captured from real routes — one wide
  (products: many fields, many zero) and one deep (sale orders with detail arrays).
- Report `ns/op`, `B/op`, `allocs/op` for `Marshal`, and separately for pass 1 vs pass 2 vs `sonic.Marshal`.
- Add an end-to-end bench over `MakeResponse` → `MakeResponseFinal` so steps 6–8 are in the same picture.

**Gate: do not start Stage 1 until we can see which step actually dominates.** The stage ordering below is
my prediction; the numbers may reorder it.

### Stage 1 — Delete pass 1 (cache the field order per type)

Pass 1 exists only to count field usage so `ComputeOptimizedOrder` can sort fields most-used-first. But the
resulting order is *transmitted* in the `keys` header, so the client uses whatever order we send — the
order does not need to be recomputed per response.

Replace per-response analysis with a per-type order cached on `TypeInfo`, computed once from the first
response that carries that type and reused thereafter.

- Expected: **~50% of marshal CPU and allocations**, since an entire full-graph walk disappears.
- Bonus: this is also the fix for the `globalRegistry` data race flagged in `docs/LAMBDA.md` §3 — once the
  order is computed once and read-only afterwards, `ResetUsedFlags`/`ComputeOptimizedOrder` stop mutating
  shared state on every request.
- **Open question to settle with Stage 0 numbers:** is usage-based ordering earning its keep at all versus
  plain declaration order? It only helps by clustering skip indices. If the payload delta is small,
  delete the mechanism instead of caching it — strictly less code.
- Risk: low. Wire format is unchanged; the `keys` header still describes whatever order we emit.

### Stage 2 — Delete the `[]any` tree (encode straight to bytes)

Replace `marshalContent`'s `[]any` return with an append-style encoder writing into a single `[]byte`:
`func (e *Encoder) appendValue(dst []byte, v reflect.Value) []byte`. This removes steps 2, 4 **and** 5 —
`sonic.Marshal` is no longer needed because we emit the JSON text directly.

- Expected: the largest single win. Kills per-scalar boxing and the per-record `[]any`, and drops one full
  reflection pass over the tree.
- Pair with a `sync.Pool` of output buffers sized from the previous response for the same route, so steady
  state stops re-growing the buffer.
- Risk: **highest of the plan.** This rewrites the encoder core. Mitigation: `marshal_test.go` and
  `skipblock_test.go` already pin the wire format; add a differential test that runs old and new encoders
  over the Stage 0 payloads and asserts byte-identical output before deleting the old path.

### Stage 3 — Stop copying the payload in the tail

- Compress directly into a pooled buffer and base64 into a **pre-sized** `[]byte` via
  `base64.StdEncoding.Encode` (not `EncodeToString`), then a single `unsafe`-free string conversion —
  removes one full copy at step 7.
- Check whether API Gateway is configured to compress. If it is, sending uncompressed and letting the edge
  handle it deletes steps 6–8 almost entirely. **Needs an infra answer, not a code answer** — flagged as a
  question below.

### Stage 4 — Optional: Lambda response streaming

`InvokeWithResponseStream` lets the handler write the body incrementally instead of returning it whole.
That removes the "entire payload in RAM" requirement outright and lifts the 6 MB response cap (note
`core/responses.go:115` already has an `isMaxLen` check at 5 MB, so we are near it).

Only worth doing if Stage 0 shows large responses are common and Stages 1–3 do not get us far enough. It
requires a Function URL or ALB rather than plain API Gateway REST, so it is an infra change too.

## Expected outcome

Stages 1–3 should take peak transient allocation per response from roughly
`2×tree + json + compressed + base64` down to `compressed + base64`, with the JSON text written once
directly into a pooled buffer. That is the bulk of the win without touching the wire format or the
frontend.

## Questions before I start

1. **Stage 0 payloads** — which two routes should I capture as the benchmark fixtures? I would guess
   products list and sale orders, but you know which responses are actually the fat ones in production.
2. **Stage 3 / API Gateway compression** — is content encoding handled at the gateway/CDN today, or does
   the Lambda have to ship compressed bytes itself? This decides whether Stage 3 is a micro-optimisation
   or deletes three steps.
3. **Scope** — approve all stages up front, or Stage 0 + 1 first and re-decide with numbers in hand? I
   recommend the latter.


---

# Results

All figures: 1000 products, 292 KB compact payload, `go test -benchmem`, i7-1355U.

## Encoder (`serialize.Marshal`)

| | ns/op | B/op | allocs/op | throughput |
|---|---|---|---|---|
| Baseline (two-pass tree + sonic) | 15,004,622 | 15,316,371 | 201,315 | 19.5 MB/s |
| Stage 1 (single pass, frozen order) | 7,134,707 | 7,620,624 | 93,965 | 41.0 MB/s |
| Stage 2 (direct-to-bytes) | 1,411,381 | 312,879 | **17** | 207 MB/s |
| **Total** | **−91%** | **−98%** | **−99.99%** | **10.6×** |

Allocation went from 52× the payload size to roughly 1×: 17 allocations to emit 292 KB.

## Full response pipeline (`MakeResponse` → `MakeResponseFinal`)

| | ns/op | B/op | allocs/op |
|---|---|---|---|
| Baseline (brotli) | 20,163,776 | 19,186,978 | 201,402 |
| Baseline (gzip) | 20,764,594 | 16,378,980 | 201,368 |
| **Final (zstd)** | **2,001,467** | **581,938** | **33** |
| Final (gzip) | 4,561,101 | 533,708 | 36 |

zstd is also 2.3× faster than gzip and compresses this payload **30.7% smaller**
(26,679 vs 38,513 bytes, 7.6% vs 11.0% of raw).

## What changed

- **Stage 1** — `Marshal` ran the object graph twice: an analysis pass to count field usage, then
  an emit pass. The resulting order is transmitted in the keys header, so it never needed
  recomputing per response; it is now frozen from the first payload carrying a type and reused.
  Field usage moved from the process-global registry onto the `Encoder`, which removed a global
  mutex lock *per field per record* and fixed the concurrent-Marshal data race in `docs/LAMBDA.md` §3.
- **Decoder correctness** — `Unmarshal` was discarding the payload's keys header and decoding
  against shared registry state, which only worked because the old `Marshal` happened to leave
  the registry holding the order it had just emitted. It now reads the order from each payload,
  so payloads are genuinely self-describing (matching `unmarshall.ts`).
- **Stage 2** — the `[]any` tree is gone. `append.go` writes the compact form straight into a
  pooled byte buffer, so no scalar is boxed and sonic is no longer involved in rendering. Also
  dropped a `reflect.New`+copy that ran for every non-addressable struct, i.e. every record.
- **Stage 3** — brotli removed from the repo entirely (including four dead `CompressBrotli*`
  helpers and the `andybalholm/brotli` dependency). zstd is primary, gzip the fallback.
  Compressors and their buffers are pooled and reset per use instead of being rebuilt per
  response; the VPS path writes compressed bytes straight to the socket with no base64 step.

## Guarantees

The wire format is unchanged and pinned by tests, not by inspection:

- `TestDirectEncoderMatchesTreeEncoder` diffs the new encoder against the retained original
  (`marshal_tree_oracle_test.go`, compiled only into the test binary) byte-for-byte across
  scalars, nesting, maps, pointers, leading arrays, ignored fields and bare values.
- `TestProductsDirectEncoderMatchesTree` does the same over 200 real `GET.products` records.
- `TestAppendJSONStringMatchesSonic` / `...Quick` and `TestAppendScalarMatchesSonic` / `...Quick`
  pin the hand-written scalar encoders against sonic, including 20,000-case fuzz runs. These
  caught two wrong assumptions: sonic does not HTML-escape, and it passes invalid UTF-8 through
  verbatim rather than replacing it with U+FFFD.
- `TestFreezeBoundaryRoundTrip` covers the pre-freeze/post-freeze layout change.
- `TestConcurrentMarshalIsolation` and `TestPooledCompressorsConcurrent` guard the shared-state
  and buffer-reuse paths; both are meaningful under `-race`.

## Stage 4 — Lambda response streaming

Implemented, **off by default**. `aws-lambda-go` bumped 1.32.0 → 1.52.0 for
`events.LambdaFunctionURLStreamingResponse` (the runtime is already `provided.al2`, which is what
that type requires).

| 1000 products, zstd | ns/op | B/op | allocs/op |
|---|---|---|---|
| Buffered (`MakeResponseFinal`) | 2,006,004 | 601,838 | 33 |
| **Streaming (`MakeStreamingResponseFinal`)** | 1,922,182 | **458,525** | 39 |

Same for gzip: 538,378 → 386,559 B/op. About **−24% transient bytes** and −4% wall clock, from
dropping the base64 expansion (+33% of the compressed body) and the runtime re-marshalling that
string into a JSON envelope. Allocation count rises slightly (33 → 39) for the read-closer
wrapper, which is a good trade for 143 KB.

**It streams the transport, not the generation.** Two things prevent a genuinely incremental,
constant-memory response, and both are outside this plan:

1. The compact format is `[keys, content]` and the keys header is derived from which fields the
   content actually used — nothing can be emitted until the whole payload has been walked.
2. The ORM materialises the full `[]Product` before `MakeResponse` is ever called, so the record
   slice, not the response buffer, is the real memory peak.

Fixing (1) would mean emitting every non-ignored field name in the header instead of only the
used ones — cheap on the wire, since the header is written once per payload, but it is a format
change. Fixing (2) means a streaming/iterator API in the ORM. Worth doing together if payload
sizes ever justify it; neither is worth doing alone.

### Enabling it

Streaming is now the deployed default, and it is no longer an environment switch. The Function
URL's `InvokeMode` and the Go handler shape must agree or every request fails, so both are
pinned in `cloud/template.yml`: the URLs declare `InvokeMode: RESPONSE_STREAM` and the function
environments declare `LAMBDA_RESPONSE_STREAMING=1`, which `core.PopulateVariables` reads.

Deploy action `2` rewrites the whole function environment, so it re-sends the same flag from
`lambdaResponseStreamingFlag` in `cloud/main.go`. To go back to buffered, change the template
and that constant together — never one alone.

**Not yet validated against a real deployment.** The tests read the response exactly as the
runtime does — JSON prelude, eight NUL bytes, then raw body — but that is a model of the runtime,
not the runtime. Deploy to a non-production stack and confirm before flipping prod.

## Not done

Nothing outstanding from this plan. The two follow-ups worth tracking are the format and ORM
changes described under Stage 4, which together would make responses genuinely constant-memory.
