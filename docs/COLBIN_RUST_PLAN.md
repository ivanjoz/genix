# Plan — a Rust colbin decoder in the colbin repo, consumed by server_utils over git

## Why

`server_utils/src/bridge/token.rs` hand-writes a colbin decoder in Rust: 577 lines mirroring
`colbin/format.go`, `colbin/bitstream.go`, `colbin/typeinfo.go` and the packed5 string codec, all
pinned by vectors generated from Go. It decodes exactly one struct, `core.UsuarioToken`, and it is
now wrong: colbin v0.1.0 routes every single-record message through **compact mode**, so the
session token arrives as 27 B whose first byte is `0x43` — bit 0 set — where the decoder expects a
`0x01` version byte and a columnar layout. Every browser SSE connection would be rejected.

Rewriting it in place would leave the same problem standing: the format's rules live in one repo
and a second, partial transcription of them lives in another, with nothing but a code review
between them and a silent drift. Two ports already exist (Go, AssemblyScript); this makes the
third a first-class one, in the repo that defines the format and next to the vectors that pin it.

**Answering the question directly: yes, Cargo can do this.** A git dependency does not need the
crate at the repo root — Cargo clones the repo, reads the root manifest, and if it is a workspace
it searches the members for the named package. A Go module and a Cargo workspace coexist in one
repo without either noticing the other.

## Scope

A **decoder only**, and only the value kinds a colbin message can actually contain at the scalar
level. No encoder: nothing in Rust writes colbin, and an unused encoder is a second specification
to keep honest for free.

| in | out |
|---|---|
| standard mode: version `0x02`/`0x06`, record count, columns, the empty-column bit | nested structs, maps, `interface{}` |
| compact mode: header, `ALL_POSITIVE`, shape, `NARROW_KEYS`, 4- and 8-bit keys | anything the encoder writes but compact mode excludes |
| scalars: int, uint, bool, f32, f64, string, bytes | an encoder |
| primitive slices: `[]intN`, `[]string`, `[]f32/64`, `[]bool` | JSON mode (`MarshalJSON` / `DecodeAny`) |
| field ids: FNV-1a-8 + linear probe, and explicit `cb:"N"` ids | |

The caller supplies the schema — a list of `(field id, kind)` — exactly as the Go and
AssemblyScript ports do, because standard mode is not self-describing. That is the same contract
`token.rs` has today, lifted out of it.

## Layout

```
colbin/                         (the existing Go module, untouched)
  Cargo.toml                    NEW — virtual workspace, members = ["rust"]
  rust/
    Cargo.toml                  name = "colbin", edition 2024
    src/lib.rs                  public API: Schema, Kind, Value, decode()
    src/bitstream.rs            LSB-first bit reader
    src/varint.rs               compact selector varint + the columnar array codec
    src/packed5.rs              packed5 frame decode
    src/compact.rs              header, key width, the record loop
    src/standard.rs             columnar: record count, column count, empty-column bit
    src/fieldid.rs              fnv8 + linear probe, mirroring typeinfo.go
    tests/vectors.rs            decodes the Go-generated corpus
  rust/vectors/main.go          NEW — Go generator, modelled on web/vectors/main.go
  .gitignore                    += /rust/target
  .github/workflows/ci.yml      += cargo test (after the Go job that regenerates vectors)
```

`bitstream.rs`, `packed5.rs`, `fieldid.rs` and the columnar int-column reader are ports of code
already written and vector-checked in `token.rs` — they move, they are not invented. `compact.rs`
and the selector varint are the genuinely new work.

## Correctness anchor

The Go codecs are the specification. `rust/vectors/main.go` marshals a fixed corpus with the real
`colbin` package and writes JSON holding, per case, the base64 message plus the decoded values it
must yield. `tests/vectors.rs` reads that file and asserts. The corpus covers, at minimum:

- both version bytes, `0x02` and `0x06`, and an empty column in each
- compact `ShapeStruct` and shapes 1–3
- `ALL_POSITIVE` set and clear (a negative value forces zigzag)
- `Keys4` (a struct tagged `cb:"1"`…) and `Keys8` (hashed ids)
- every scalar kind and every primitive-slice kind
- a string that lands byte-aligned and one that does not, since packed5 shifts in that case
- `UsuarioToken` itself, so the bridge's own shape is a pinned case rather than an assumption

Vectors are committed, so `cargo test` needs no Go toolchain; CI regenerates them and fails on a
diff, which is what catches a Go-side format change that nobody ported.

## Wiring server_utils

```toml
# server_utils/Cargo.toml
colbin = { git = "https://github.com/ivanjoz/colbin", rev = "<sha>" }
```

Pinned by `rev`, not `tag`: the tags in that repo are the Go module's version ladder
(`v0.1.0` = `83d9655`), and a Cargo tag requirement would either entangle the two release cadences
or silently resolve to a tag predating the crate. `rev` is reproducible and says what it means.

`token.rs` then keeps only what is genuinely the bridge's own:

- `UserToken`, the five-field session identity
- `SESSION_FIELD_NAMES` and the schema built from them
- `decode_session_token` / `decode_session_base64`, now ~20 lines over `colbin::decode`
- `decode_channel_token` — untouched, it is a separate custom format, not colbin
- `TokenError`, with the colbin-internal variants (`Version`, `RecordCount`, `ColumnType`,
  `UnknownField`) collapsing into one `#[from] colbin::Error`

Net: ~577 lines down to roughly 150, and none of the removed lines describe a wire format any more.

## Order of work

1. `rust/` crate skeleton + root `Cargo.toml` workspace; confirm `cargo build` and that a git dep
   from server_utils resolves before writing the decoder.
2. Port `bitstream`, `packed5`, `fieldid`, columnar int/string columns from `token.rs`.
3. Write `compact.rs` and the selector varint — the new work.
4. `rust/vectors/main.go` + `tests/vectors.rs`; iterate until the corpus passes.
5. Commit and **push** the colbin repo (it is a separate remote; a genix commit does not publish it).
6. Point `server_utils/Cargo.toml` at the pushed `rev`, gut `token.rs`, `cargo test`.
7. `RATIONALE.md` in both repos; update the colbin README's port list and Limitations.

Steps 1–5 are inside `/run/media/ivanjoz/projects/colbin`, a separate repository from genix.

## What this does not fix

Nothing here restores the colbin blobs already in Scylla — those were written at format `0x01` and
are lost, which you have already decided to handle by reseeding the keyspace.
