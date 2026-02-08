Technical Strategy: Multi-Level Composite Bucketing (Go/ScyllaDB)
1. Objective
To bypass ScyllaDB's lack of range index support for set<int> columns. We will simulate range scans by indexing pre-calculated hashes of (ProductID, BucketID) at varying granularities (1, 4, and 6 weeks) into set<int> virtual columns.

2. Schema Expansion
When CompositeBucketing(1, 4, 6) is detected, the ORM must internally map the logical field DetailProductsIDs to three physical set<int> columns:

idx_products_w1: Stores Hash(ProductID, Week / 1)

idx_products_w4: Stores Hash(ProductID, Week / 4)

idx_products_w6: Stores Hash(ProductID, Week / 6)

The Hash Function
To fit into set<int>, use a 32-bit hash. A simple and fast approach in Go:

3. Write Path (Persistence)
During every Insert or Update, the ORM calculates the hashes for all products across all three bucket levels.

Iterate through DetailProductsIDs.

Calculate BucketID for each size: int16(Week / size).

Hash the (ProductID, BucketID) pair.

Populate the respective hidden set<int> columns in the INSERT statement.

4. Read Path (The Greedy Coverage Algorithm)
When a query is issued for a range of weeks, the ORM must choose the optimal combination of buckets to minimize database round-trips.

Example: Querying Weeks 2521 to 2530 (10 Weeks)
Target Range: [2521, 2530]

Identify Available Buckets:

W6 Buckets: Bucket 420 (2520-2525), Bucket 421 (2526-2531).

W4 Buckets: Bucket 630 (2520-2523), Bucket 631 (2524-2527), Bucket 632 (2528-2531).

W1 Buckets: Individual weeks.

Coverage Selection (Optimization Logic):

The algorithm prioritizes the largest buckets that cover the most target weeks with the least "waste" (over-fetching).

Selected:

Query 1: idx_products_w6 CONTAINS Hash(PID, 420) (Covers 2521-2525, wastes 2520).

Query 2: idx_products_w6 CONTAINS Hash(PID, 421) (Covers 2526-2530, wastes 2531).

Result: 2 high-performance queries instead of 10.

5. Implementation Roadmap for the Agent
A. Core Math Logic
Implement a GetCoverage(start, end int16, sizes []int) []BucketKey function.

It should return a list of bucket sizes and their corresponding IDs.

It should favor over-fetching (reading a few extra weeks) if it reduces the total number of SELECT statements.

B. Query Generator
Modify the SQL builder to:

Intercept WHERE DetailProductsIDs CONTAINS ? AND Week BETWEEN ? AND ?.

Call GetCoverage.

Generate a SELECT statement for each bucket using the CONTAINS operator on the virtual columns.

C. Post-Processing (The "Clean-up")
Because the buckets might return "waste" data (like week 2520 in the example above):

De-duplicate: Use a map[int64]bool on the ID column to ensure unique rows if buckets overlap.

Filter: Perform a final check in Go: if row.Week >= start && row.Week <= end.

6. Key Constraints
Virtual Column Naming: Use a consistent suffix (e.g., _w1, _w4, _w6) so the ORM can auto-discover these during schema migration.

Parallelism: Use golang.org/x/sync/errgroup to execute the bucket queries concurrently to minimize latency.
