# SeekStorm for Minimal Product Catalog Search

This guide is for a setup where:

- **ScyllaDB is the source of truth** for products.
- **SeekStorm is only a search index**.
- You want **lexical/keyword search only**, not vector search.
- Each query searches inside **one client/customer partition**.
- The search result should be **product IDs only**.

Recommended model:

```text
ScyllaDB              = product source of truth
SeekStorm index       = product search index
One SeekStorm index   = one client/customer
SeekStorm document ID = product ID, or an internal numeric ID
search_text           = product name + SKU + brand + keywords
```

Example:

```text
Client 123
  SeekStorm index: products_client_123
  Documents:
    doc_id=98765, search_text="Samsung Galaxy S22 black case SM-S901"
    doc_id=98766, search_text="Samsung USB-C fast charger 25W"
```

Query:

```text
products_client_123 + "galaxy case" -> [98765, ...]
```

---

## 1. Why SeekStorm fits this use case

SeekStorm can run as:

1. a **standalone HTTP server**, or
2. an **embedded Rust library**.

For your Go + ScyllaDB backend, the easiest architecture is:

```text
Go backend -> HTTP -> SeekStorm server
Go backend -> ScyllaDB for product details
```

SeekStorm supports lexical search and vector search, but internally these are separate engines. For your case, you should create lexical-only indexes and query using lexical mode only.

Important facts from the official docs:

- SeekStorm server exposes REST endpoints and an embedded Web UI.
- SeekStorm server supports multi-tenancy: multiple users, each with multiple indices.
- SeekStorm supports lexical search using an inverted index.
- SeekStorm can be configured with `Inference::None` in embedded Rust mode.
- REST examples create indexes by defining fields with `index_lexical: true` or `index_lexical: false`.

References are listed at the end of this file.

---

## 2. Recommended index-per-client design

For **200 clients max**, use one SeekStorm index per client:

```text
products_client_001
products_client_002
...
products_client_200
```

Keep a mapping somewhere in your backend:

```sql
client_id -> seekstorm_index_id
```

For example, in ScyllaDB:

```sql
CREATE TABLE search_indexes_by_client (
    client_id bigint PRIMARY KEY,
    seekstorm_index_id int,
    seekstorm_index_name text,
    created_at timestamp
);
```

Why keep this mapping?

SeekStorm REST queries use the numeric `index_id` in endpoints like:

```text
/api/v1/index/{index_id}/query
```

The human-readable name, such as `products_client_123`, is useful, but your backend should store the returned numeric `index_id`.

---

## 3. Install SeekStorm server from source

### 3.1 Install Rust

On Fedora:

```bash
sudo dnf install -y git gcc gcc-c++ make openssl-devel
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source "$HOME/.cargo/env"
rustc --version
cargo --version
```

On Ubuntu/Debian:

```bash
sudo apt update
sudo apt install -y git build-essential pkg-config libssl-dev
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source "$HOME/.cargo/env"
rustc --version
cargo --version
```

### 3.2 Clone and build

```bash
git clone https://github.com/SeekStorm/SeekStorm.git
cd SeekStorm
cargo build --release
```

The server README shows:

```bash
cargo build --release
```

After building, look for the server binary:

```bash
find target/release -maxdepth 1 -type f -executable -printf '%f\n'
```

Expected binary name is likely:

```text
seekstorm_server
```

If your build produces a slightly different binary name, use the one shown by `find`.

---

## 4. Run SeekStorm server

Create a data directory:

```bash
sudo mkdir -p /var/lib/seekstorm
sudo chown "$USER:$USER" /var/lib/seekstorm
```

Set a master secret. This is important because the server README warns that generated API keys can be compromised if `MASTER_KEY_SECRET` is not set.

```bash
export MASTER_KEY_SECRET='change-this-to-a-long-random-secret'
```

Run on localhost first:

```bash
./target/release/seekstorm_server \
  local_ip="127.0.0.1" \
  local_port=8080 \
  index_path="/var/lib/seekstorm"
```

The server prints a **master API key** on startup. Save it temporarily as:

```bash
export SEEKSTORM_MASTER_KEY='PASTE_MASTER_KEY_FROM_SERVER_CONSOLE'
```

Do not expose this server directly to the public internet. Put it behind your backend, or bind it to `127.0.0.1`.

---

## 5. Create an API key for your product-search service

Create one API key for your backend service. Since you expect about 200 clients, set `indices_max` above that, for example `300`.

```bash
curl --request POST 'http://127.0.0.1:8080/api/v1/apikey' \
  --header "apikey: $SEEKSTORM_MASTER_KEY" \
  --header 'content-type: application/json' \
  --data '{
    "indices_max": 300,
    "indices_size_max": 100000000000,
    "documents_max": 100000000,
    "operations_max": 100000000,
    "rate_limit": 100000
  }'
```

The response should include a generated API key. Save it:

```bash
export SEEKSTORM_API_KEY='PASTE_GENERATED_API_KEY'
```

In production, store this key in your secret manager, not in source code.

---

## 6. Create one lexical-only index per client

For your minimal case, use only two fields:

```text
search_text  = indexed lexical text, not returned unless you want debugging
product_id   = stored ID returned by query
```

Create an index for client `123`:

```bash
curl --request POST 'http://127.0.0.1:8080/api/v1/index' \
  --header "apikey: $SEEKSTORM_API_KEY" \
  --header 'content-type: application/json' \
  --data '{
    "index_name": "products_client_123",
    "similarity": "Bm25f",
    "tokenizer": "UnicodeAlphanumeric",
    "schema": [
      {
        "field": "search_text",
        "field_type": "Text",
        "store": false,
        "index_lexical": true
      },
      {
        "field": "product_id",
        "field_type": "U64",
        "store": true,
        "index_lexical": false
      }
    ]
  }'
```

Notes:

- `index_lexical: true` only on `search_text`.
- `product_id` is stored so you can return it.
- No vector fields are defined.
- No inference is configured in this REST schema.
- `store: false` for `search_text` reduces unnecessary stored data because ScyllaDB already has product details.

Store the returned `index_id` in ScyllaDB:

```text
client_id=123 -> index_id=0
```

The exact returned JSON shape can change, so inspect the response and store the numeric ID it returns.

---

## 7. Index products

### 7.1 Build `search_text`

Create one compact searchable string in your Go backend:

```text
{name} {brand} {sku} {keywords}
```

Example:

```text
Samsung Galaxy S22 black case Samsung SM-S901 phone cover protector
```

Keep this string normalized:

- lower-case
- remove repeated spaces
- remove weird punctuation if needed
- include SKU and common aliases
- avoid huge descriptions unless you need them

### 7.2 Insert one product

Suppose client `123` has SeekStorm index ID `0`.

```bash
curl --request POST 'http://127.0.0.1:8080/api/v1/index/0/doc' \
  --header "apikey: $SEEKSTORM_API_KEY" \
  --header 'content-type: application/json' \
  --data '{
    "search_text": "samsung galaxy s22 black case samsung sm-s901 phone cover protector",
    "product_id": 98765
  }'
```

### 7.3 Insert many products at once

Batch writes are better than many single-document HTTP requests.

```bash
curl --request POST 'http://127.0.0.1:8080/api/v1/index/0/doc' \
  --header "apikey: $SEEKSTORM_API_KEY" \
  --header 'content-type: application/json' \
  --data '[
    {
      "search_text": "samsung galaxy s22 black case samsung sm-s901 phone cover protector",
      "product_id": 98765
    },
    {
      "search_text": "samsung usb c fast charger 25w power adapter",
      "product_id": 98766
    },
    {
      "search_text": "iphone 15 transparent case magsafe cover",
      "product_id": 98767
    }
  ]'
```

### 7.4 Commit after bulk indexing

SeekStorm has a commit endpoint:

```bash
curl --request PATCH 'http://127.0.0.1:8080/api/v1/index/0' \
  --header "apikey: $SEEKSTORM_API_KEY" \
  --header 'content-type: application/json'
```

For live product updates, you can also query with `realtime: true`, shown below.

---

## 8. Query products and return IDs

Use POST query:

```bash
curl --request POST 'http://127.0.0.1:8080/api/v1/index/0/query' \
  --header "apikey: $SEEKSTORM_API_KEY" \
  --header 'content-type: application/json' \
  --data '{
    "query": "galaxy case",
    "offset": 0,
    "length": 20,
    "realtime": true,
    "field_filter": ["search_text"]
  }'
```

Then extract `product_id` from the result documents.

If the response contains internal document IDs plus stored fields, use the stored `product_id`. If it returns internal doc IDs only in your version/config, keep a mapping from SeekStorm doc ID to ScyllaDB product ID.

Recommended approach:

```text
Return product_id from SeekStorm -> fetch product details from ScyllaDB
```

Avoid returning product names, descriptions, prices, etc. from SeekStorm. Keep it as an ID-only search index.

---

## 9. Updating products

When a product changes in ScyllaDB:

1. Rebuild `search_text`.
2. Update the corresponding SeekStorm document.

The server README shows update via `PATCH /api/v1/index/{index_id}/doc` using a pair:

```bash
curl --request PATCH 'http://127.0.0.1:8080/api/v1/index/0/doc' \
  --header "apikey: $SEEKSTORM_API_KEY" \
  --header 'content-type: application/json' \
  --data '[98765, {
    "search_text": "samsung galaxy s22 black rugged case shockproof",
    "product_id": 98765
  }]'
```

Important: this assumes you use the product ID as the SeekStorm document ID. If SeekStorm auto-assigns document IDs in your ingestion path, store the mapping:

```text
client_id + product_id -> seekstorm_doc_id
```

A safer production design is:

```text
Scylla product_id      = your real ID
SeekStorm doc_id      = numeric ID used by SeekStorm
Stored product_id     = same product ID returned in search results
```

If your product IDs are already compact numeric IDs, use them as SeekStorm doc IDs.

---

## 10. Deleting products

Delete one document by document ID:

```bash
curl --request DELETE 'http://127.0.0.1:8080/api/v1/index/0/doc/98765' \
  --header "apikey: $SEEKSTORM_API_KEY" \
  --header 'content-type: application/json'
```

Delete multiple documents:

```bash
curl --request DELETE 'http://127.0.0.1:8080/api/v1/index/0/doc' \
  --header "apikey: $SEEKSTORM_API_KEY" \
  --header 'content-type: application/json' \
  --data '[98765, 98766, 98767]'
```

---

## 11. Go integration example

This is a minimal Go helper for querying SeekStorm.

```go
package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTP: &http.Client{
			Timeout: 800 * time.Millisecond,
		},
	}
}

type QueryRequest struct {
	Query       string   `json:"query"`
	Offset      int      `json:"offset"`
	Length      int      `json:"length"`
	Realtime    bool     `json:"realtime"`
	FieldFilter []string `json:"field_filter,omitempty"`
}

// The exact response shape may differ by SeekStorm version.
// Keep this flexible while prototyping.
type QueryResponse struct {
	Results []json.RawMessage `json:"results"`
}

func (c *Client) Query(ctx context.Context, indexID int, q string, limit int) (*QueryResponse, error) {
	body := QueryRequest{
		Query:       q,
		Offset:      0,
		Length:      limit,
		Realtime:    true,
		FieldFilter: []string{"search_text"},
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/index/%d/query", c.BaseURL, indexID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", c.APIKey)
	req.Header.Set("content-type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("seekstorm query failed: status=%d", resp.StatusCode)
	}

	var out QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}
```

During prototyping, print the raw JSON response once:

```go
fmt.Printf("%s\n", rawResponse)
```

Then define the exact Go struct for the response shape from your SeekStorm version.

---

## 12. Product save flow with ScyllaDB

When saving a product:

```text
1. Save product in ScyllaDB.
2. Build search_text.
3. Upsert document in SeekStorm.
4. Optionally commit, or rely on realtime query mode.
```

Pseudo-code:

```go
func SaveProduct(ctx context.Context, product Product) error {
	// 1. Save source of truth.
	if err := scyllaSaveProduct(ctx, product); err != nil {
		return err
	}

	// 2. Build minimal search text.
	searchText := BuildSearchText(product)

	// 3. Find client index ID.
	indexID, err := getSeekStormIndexID(ctx, product.ClientID)
	if err != nil {
		return err
	}

	// 4. Upsert into SeekStorm.
	return seekstormUpsertProduct(ctx, indexID, product.ID, searchText)
}
```

Search flow:

```go
func SearchProducts(ctx context.Context, clientID int64, query string) ([]ProductID, error) {
	indexID, err := getSeekStormIndexID(ctx, clientID)
	if err != nil {
		return nil, err
	}

	ids, err := seekstormQueryIDs(ctx, indexID, query, 20)
	if err != nil {
		return nil, err
	}

	return ids, nil
}
```

Then fetch details from ScyllaDB:

```text
SeekStorm IDs -> ScyllaDB product rows
```

---

## 13. Configuration for minimum CPU usage

Use these choices:

```text
One index per client
One lexical text field
No vector fields
No inference
No spelling correction
No query completion
No facets
No highlighting
Small result length, e.g. 20–50
Batch writes
Query only one client index
```

Schema:

```json
[
  {
    "field": "search_text",
    "field_type": "Text",
    "store": false,
    "index_lexical": true
  },
  {
    "field": "product_id",
    "field_type": "U64",
    "store": true,
    "index_lexical": false
  }
]
```

Query body:

```json
{
  "query": "galaxy case",
  "offset": 0,
  "length": 20,
  "realtime": true,
  "field_filter": ["search_text"]
}
```

Do not request highlights or large result windows.

---

## 14. Lexical-only embedded Rust configuration

If you ever decide to embed SeekStorm in a Rust service instead of using the HTTP server, the docs show the key pieces.

Use:

```rust
inference: Inference::None
```

and:

```rust
let search_mode = SearchMode::Lexical;
```

Example adapted to your product use case:

```rust
use std::path::Path;
use seekstorm::index::{
    IndexMetaObject, Clustering, LexicalSimilarity, TokenizerType,
    StopwordType, FrequentwordType, AccessType, StemmerType,
    NgramSet, DocumentCompression, create_index,
};
use seekstorm::vector::Inference;

let index_path = Path::new("/var/lib/seekstorm/products_client_123");

let schema_json = r#"
[
  {
    "field": "search_text",
    "field_type": "Text",
    "store": false,
    "index_lexical": true
  },
  {
    "field": "product_id",
    "field_type": "U64",
    "store": true,
    "index_lexical": false
  }
]
"#;

let schema = serde_json::from_str(schema_json).unwrap();

let meta = IndexMetaObject {
    id: 0,
    name: "products_client_123".to_string(),
    lexical_similarity: LexicalSimilarity::Bm25f,
    tokenizer: TokenizerType::UnicodeAlphanumeric,
    stemmer: StemmerType::None,
    stop_words: StopwordType::None,
    frequent_words: FrequentwordType::None,
    ngram_indexing: NgramSet::SingleTerm as u8,
    document_compression: DocumentCompression::Snappy,
    access_type: AccessType::Mmap,
    spelling_correction: None,
    query_completion: None,
    clustering: Clustering::None,
    inference: Inference::None,
};

let segment_number_bits = 11;
let index = create_index(
    index_path,
    meta,
    &schema,
    &Vec::new(),
    segment_number_bits,
    false,
    None,
).await.unwrap();
```

For searching:

```rust
use seekstorm::search::{
    Search, SearchMode, QueryType, ResultType, QueryRewriting,
};

let query = "galaxy case".to_string();
let query_vector = None;
let search_mode = SearchMode::Lexical;
let query_type = QueryType::Intersection;
let result_type = ResultType::TopkCount;
let include_uncommitted = true;
let offset = 0;
let length = 20;

let result = index.search(
    query,
    query_vector,
    query_type,
    search_mode,
    false,
    offset,
    length,
    result_type,
    include_uncommitted,
    Vec::new(),
    Vec::new(),
    Vec::new(),
    Vec::new(),
    QueryRewriting::SearchOnly,
).await;
```

---

## 15. Should you use n-grams?

SeekStorm has n-gram indexing options in the Rust library. N-grams can speed up phrase queries, but they increase index size.

For your first version, use:

```text
SingleTerm
```

or the REST server defaults.

Add n-grams later only if profiling shows phrase queries are too slow.

Rule:

```text
Normal keyword search: avoid n-gram indexing at first.
Heavy exact phrase search: test n-gram indexing.
```

---

## 16. Operational notes

### 16.1 200 clients

One index per client is reasonable.

```text
200 clients = 200 indexes
```

This keeps queries isolated and simple.

### 16.2 10,000 clients

Do not create one index per tiny client if you ever grow that high.

Use shared shard indexes:

```text
products_shard_000
products_shard_001
...
products_shard_127
```

Then route:

```text
shard = hash(client_id) % 128
```

But for your stated maximum of around 200 clients, index-per-client is cleaner.

### 16.3 Backups

Back up:

```text
/var/lib/seekstorm
```

Only while server is stopped, or use filesystem snapshots if available.

Also keep ScyllaDB as source of truth, so you can rebuild SeekStorm indexes if needed.

### 16.4 Rebuild strategy

If an index is corrupted or you change schema:

```text
1. Create new index: products_client_123_v2
2. Re-index all products for that client from ScyllaDB
3. Commit
4. Update client_id -> index_id mapping
5. Delete old index
```

This avoids downtime.

---

## 17. Minimal production checklist

- [ ] SeekStorm binds to `127.0.0.1` or a private network only.
- [ ] `MASTER_KEY_SECRET` is set.
- [ ] Master API key is not used by the application.
- [ ] App uses a limited API key.
- [ ] One index per client.
- [ ] ScyllaDB stores `client_id -> seekstorm_index_id`.
- [ ] `search_text` is normalized in Go.
- [ ] `search_text` is indexed but not stored.
- [ ] `product_id` is stored.
- [ ] No vector fields.
- [ ] No spelling correction/query completion unless you explicitly need them.
- [ ] Search limit is small, e.g. 20–50.
- [ ] Product details are fetched from ScyllaDB, not SeekStorm.
- [ ] You have a rebuild process from ScyllaDB.

---

## 18. References

- SeekStorm GitHub README: https://github.com/SeekStorm/SeekStorm
- SeekStorm server README: https://github.com/SeekStorm/SeekStorm/blob/main/src/seekstorm_server/README.md
- SeekStorm Rust crate docs: https://docs.rs/seekstorm/
- SeekStorm REST docs: https://seekstorm.github.io/documentation/
- SeekStorm API docs overview: https://seekstorm.com/docs
- N-gram phrase search article: https://seekstorm.com/blog/n-gram-indexing-for-faster-phrase-search/
