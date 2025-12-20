# Serialize - Compact JSON Array Serialization

A high-performance serialization library that converts Go structs to compact JSON arrays, significantly reducing payload sizes compared to traditional JSON object notation.

## Motivation

Traditional JSON serialization repeats field names for every object:

```json
[
  {"name": "John", "age": 27, "country": "USA"},
  {"name": "Jane", "age": 25, "country": "UK"},
  {"name": "Bob", "age": 30, "country": "USA"}
]
```

This library eliminates field name repetition by:
1. Storing field definitions once in a **keys list**
2. Serializing objects as **value arrays** with positional mapping
3. Skipping zero/empty values to further reduce size

Result:

```json
[
  [[1,0,"name",1,"age",2,"country"]],
  [2,[1,[1],"John",27,"USA"],[0,"Jane",25,"UK"],[0,"Bob",30,"USA"]]
]
```

## Output Format

The `Marshal` function produces a single array with two elements: `[keys, content]`

### Keys Array

Contains type definitions with field mappings:

```
[[typeID, fieldIndex1, "fieldName1", fieldIndex2, "fieldName2", ...], ...]
```

- **typeID**: Unique identifier for the struct type
- **fieldIndex**: 0-based position of the field in the optimized order
- **fieldName**: JSON tag name (or Go field name if no tag)

Example:
```json
[[1,0,"name",1,"age",2,"anArray",3,"demo",4,"decimal"]]
```

### Content Array

Contains the serialized data using positional arrays instead of key-value pairs.

## Encoding Rules

### Array Headers

Each array starts with a header value:

| Header | Meaning |
|--------|---------|
| `1` | New object type (includes type ID in reference block) |
| `0` | Same type as previous object |
| `2` | Array/Slice of values |
| Other | Plain array values |

### Object Reference Block

After the header (1 or 0), there's a reference block array:

**For header `1`:** `[typeID, skipIndex1, skipIndex2, ...]`
- First value is the type ID
- Subsequent values are field indices to skip (zero/empty values)

**For header `0`:** `[skipIndex1, skipIndex2, ...]`
- Only contains skip indices (type is inherited from previous object)
- Can be omitted entirely if no fields are skipped

### Example Breakdown

```json
[1,[1,3,4],"John",27,[2,1,2,3]]
```

- `1` - Header: new object type
- `[1,3,4]` - Reference block: type ID=1, skip fields at positions 3 and 4
- `"John"` - Value at position 0 (name)
- `27` - Value at position 1 (age)
- `[2,1,2,3]` - Value at position 2 (anArray), header `2` indicates it's an array

## Two-Pass Optimization

The library uses a **two-pass optimization** strategy to minimize skip indices:

### Pass 1: Analysis

Scans all data to count how many times each field is used across all records.

### Field Reordering

After analysis, fields are reordered by usage frequency:
- Most-used fields get lower indices (0, 1, 2...)
- Least-used fields get higher indices

### Pass 2: Serialization

Marshals data using the optimized field order.

### Why This Matters

Consider this data:

```go
// 5 records where:
// - Name, Age: used in all 5 records
// - AnArray: used in 3 records
// - Demo: used in 2 records
// - Decimal: used in 1 record
```

**Without optimization** (original order: Name, Age, Decimal, AnArray, Demo):
- Records without Decimal must skip index 2
- Skip indices scattered across the range

**With optimization** (reordered: Name, Age, AnArray, Demo, Decimal):
- Decimal is now at index 4 (last position)
- Records skip mostly higher indices (3, 4)
- Fewer skip indices needed overall

## JSON Tag Support

The library respects JSON struct tags:

```go
type User struct {
    FirstName string  `json:"firstName"`
    LastName  string  `json:"lastName,omitempty"`
    Age       int     `json:"age"`
}
```

The keys list will use `"firstName"`, `"lastName"`, `"age"` instead of Go field names.

## Usage

### Marshaling

```go
data := []MyStruct{...}
output, err := serialize.Marshal(data)
// output is []byte containing: [keys, content]
```

### Unmarshaling

```go
var result []MyStruct
err := serialize.Unmarshal(output, &result)
```

## Nested Structures

Nested structs and arrays are fully supported:

```go
type Parent struct {
    Name  string `json:"name"`
    Child Child  `json:"child"`
}

type Child struct {
    Value int `json:"value"`
}
```

Each nested struct type gets its own entry in the keys list with a unique type ID.

## Size Comparison

For a typical API response with 100 objects and 10 fields each:

| Format | Approximate Size |
|--------|------------------|
| Standard JSON | ~15 KB |
| This Library | ~8 KB |
| Savings | ~47% |

Savings increase with:
- More records (field names not repeated)
- More zero/empty values (skipped entirely)
- Consistent data patterns (better field optimization)
