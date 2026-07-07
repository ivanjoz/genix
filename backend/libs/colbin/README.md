# colbin

A columnar, delta-encoded binary serializer for **slices of structs** — sized and
tuned for numeric, DB-row-shaped data. It is a sibling to `app/libs/cbor` with the
same `Marshal`/`Unmarshal` surface, but a fundamentally different layout: instead
of encoding record-by-record (row / AoS, like CBOR or JSON), colbin transposes the
data and encodes it **column-by-column** (SoA), applying frame-of-reference (FOR)
delta encoding plus bit-packing to each numeric column.

On typical ERP row batches this yields payloads **~3× smaller than CBOR**, decode
**~3× faster**, and encode modestly faster — see [Benchmarks](#benchmarks).

## When to use it

Use colbin when you serialize **arrays of homogeneous structs** (every record has
the same fields) where numbers dominate — cache payloads, API list responses, DB
row batches. The wins come from encoding many similar values of one field together.

Do **not** reach for it to serialize a single small object where CBOR/JSON is fine,
or for heterogeneous documents. colbin must buffer all records before emitting (it
is not a record-at-a-time stream).

## Usage

```go
import "app/libs/colbin"

type Product struct {
    ID        int64   `cb:"id"`
    CompanyID int32   `cb:"company_id"`
    Price     int32   `cb:"price"`
    Active    bool    `cb:"active"`
    Name      string  `cb:"name"`
    Weight    float32 `cb:"weight"`
}

rows := []Product{ /* ... */ }

data, err := colbin.Marshal(rows)   // []Product -> []byte

var out []Product
err = colbin.Unmarshal(data, &out)  // []byte -> *[]Product
```

`Marshal` accepts a slice of structs, a pointer to one, or a single struct (encoded
as `N = 1`). `Unmarshal` needs a pointer to a slice, or a pointer to a struct for an
`N == 1` message. Decoding requires the **same Go type** — the format is not
self-describing about field names (see [Field ids](#field-ids)).

## Field ids and the `cb` tag

Every field is identified on the wire by a single `uint8`. By default it is derived
from a **FNV-1a-32 hash of the field name, xor-folded to 8 bits**, with collisions
resolved by linear probing (id, id+1, …, wrapping). Id `255` is reserved as a
terminator, so a struct may have at most **254 encodable fields**.

The `cb` struct tag controls this. Its value is a comma-separated list; an **integer
token sets the id explicitly** (no hashing), and the first **non-integer token
overrides the name** used for hashing:

| Tag | Effect |
|---|---|
| `cb:"name"` | hash `name` instead of the Go field name |
| `cb:"5"` | fixed id `5`, no hashing |
| `cb:"id,5"` | name `id`, fixed id `5` |
| `cb:"-"` | skip the field entirely |
| *(no tag)* | hash the Go field name |

Explicit ids are reserved first, then hashed fields probe around them. Two fields
with the same explicit id, or an id above 254, are rejected. Because ids come from
the type on both sides, encoder and decoder derive the same mapping without
transmitting field names.

## Supported types

| Go type | Encoding |
|---|---|
| `int8..int64`, `uint8..uint32`, `int`, `uint`, `bool` | integer column (FOR + bit-pack) |
| `float32`, `float64` | raw IEEE-754 (32/64-bit) |
| `string` | length sub-column + concatenated UTF-8 |
| `[]byte` | length sub-column + concatenated bytes |
| `[]T` (T scalar or struct) | length sub-column + flattened element column |
| nested `struct` | recursive sub-table of columns |
| `[][]T`, deeper nesting | recursion |
| `*T` (incl. `*struct`, `[]*T`) | nullable column (null bitmap, see below) |
| `map[K]V` (K scalar/string, V any) | length column + flattened keys + values |
| `interface{}` / `any` (incl. `map[string]any`, `[]any`) | self-describing tagged values (see below) |

**Not yet supported** (error on encode): `time.Time`, pointer-to-pointer
(`**T`), non-scalar map keys, a struct/`chan`/`func` held inside an `interface{}`.
`uint64` values above `math.MaxInt64` are not
representable (the internal column type is `int64`). `nil` and empty slices/maps
are indistinguishable — both decode to `nil`.

### Nullability

Any pointer type is nullable and distinguishes all three states — `nil`, a pointer
to the zero value (`&0`, `&""`), and a pointer to any other value. A nullable column
writes a 1-byte `nullFlags`; if it has any nulls, a `ceil(N/8)` presence bitmap
follows (1 = present) and only the non-null values are stored densely. A column with
no nulls costs just the 1 flag byte. This is how `[]*int32` stores a null *slot*
(distinct from `0`) and how `map[K]*V` stores null values.

## How the integer encoding works

This is the core of the format. For each integer column (all N values of one field),
colbin picks the smallest bit-width that fits the data and stores every value as a
small delta from a per-column base:

- **Unsigned mode** (no negative values): `0` is a reserved *sentinel* meaning
  "empty/absent" and decodes back to `0`. The base is `min_nonzero - 1`, so every
  real value maps to `>= 1` and never collides with the sentinel.
  `enc = (v == 0) ? 0 : v - base`.
- **Signed mode** (column contains a negative): `0` can no longer be a sentinel, so
  the base is the true minimum (possibly negative) and `enc = v - base`. The wider
  span typically costs ~1 more bit.

The largest `enc` picks the packed width from `{8, 12, 16, 24, 32, 48, 64}` bits
(true bit-level packing — 12/24/48-bit values straddle byte boundaries). Columns
that are entirely zero are flagged empty and store nothing.

## Wire format

```
message := [version:1] [recordCount:uvarint] subTable

subTable := [colCount:1] column*                 // one column per field

column   := [field_id:1] [flags:1] payload

flags    := field_type(bits 0-2) | is_signed(bit 3) | precision(bits 4-6) | empty(bit 7)
```

Payloads by `field_type`:

- **int**: `[base : nativeWidth bits] [enc : N × precisionWidth bits]`, byte-aligned
  at column end (empty column: no payload).
- **float**: `N × (32|64) raw IEEE-754 bits` (empty column: no payload).
- **string / bytes**: an embedded int length-column, then the concatenated bytes.
- **array**: an embedded int length-column (element count per record), then one
  flattened element column (recursively `[flags] payload`).
- **struct**: a nested `subTable`.
- **map**: an embedded int length-column (entry count per record), then a flattened
  keys column and a flattened values column.
- **nullable** (pointer types): `[nullFlags:1] [presence bitmap IF has_nulls]` in
  front of the (dense) inner column.
- **any** (`interface{}`): `N` self-describing tagged values. Unlike every other
  column, `any` values can't be columnarized (concrete type is unknown at build
  time and varies per value), so each is written row-style as `[tag:1] payload` —
  a compact escape hatch inside the columnar frame. `nil`/`bool`/`int64`/`uint64`/
  `float64`/`string`/`[]byte`/`[]any`/`map[string]any` (recursive). Decode
  normalizes numbers to `int64`/`uint64`/`float64`, matching CBOR's dynamic decode.

Each column is byte-aligned, and its byte span is deterministically recomputable
from N and the flags, so the decoder advances column-to-column with a simple cursor.

## Performance

Two design choices keep it fast:

- **Field access via `github.com/viant/xunsafe`** — cached, typed, unsafe struct
  field get/set on the hot numeric path (array elements use direct pointer casts).
  Type layout (`typeInfo`, field ids, accessors) is built once per type and cached.
- **Encode writes bit-packed data straight into the output buffer** (no per-column
  temp buffer or copy), scans each column once for min/max, and reuses scratch
  slices from `sync.Pool`.

### Benchmarks

1000 records, `bench_test.go`, i7-1355U:

| | colbin | CBOR | ratio |
|---|---|---|---|
| Size (scalar) | 27.7 KB | 89.8 KB | **3.24× smaller** |
| Size (nested) | 43.4 KB | 121.8 KB | **2.81× smaller** |
| Encode (scalar) | 162 µs / 8 allocs | 204 µs / 2 allocs | **1.26× faster** |
| Decode (scalar) | 178 µs | 504 µs | **2.8× faster** |

> The `MB/s` figure printed by `go test -bench` is misleading here: it is
> `bytes ÷ time`, and colbin emits ~3× fewer bytes, so it reports lower MB/s despite
> lower latency. Compare **ns/op**.

Run them:

```sh
go test ./libs/colbin/ -bench . -benchmem
```

## Files

| file | role |
|---|---|
| `bitstream.go` | LSB-first bit packer/reader (32-bit chunked) |
| `format.go` | version, type codes, width table, precision selection, sign-extend |
| `column_int.go` | integer column encode (FOR + bit-pack) and decode |
| `typeinfo.go` | field ids, `cb` tag, recursive type layout, cache |
| `value.go` / `value_elem.go` | typed scalar get/set (struct fields / array elements) |
| `null_map.go` | nullable (pointer) columns via bitmap, and map columns |
| `pool.go` | scratch-slice pools for encoding |
| `encode.go` / `decode.go` | `Marshal` / `Unmarshal` and the column drivers |
| `*_test.go` | round-trip, random, size, and benchmark tests |

## Limitations

- Not a streaming format — all records are buffered before output.
- Trusts the input buffer on decode (internal use); malformed data can panic on
  slice bounds rather than returning an error.
- Self-referential struct types recurse infinitely during type analysis.
- No backwards-compatibility guarantees — the format version byte is bumped on any
  wire change (this project is pre-alpha).
