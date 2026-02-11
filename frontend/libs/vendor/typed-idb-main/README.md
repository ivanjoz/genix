# Typed-IDB

Transform IndexedDB into a powerful, type-safe storage solution for modern web applications. Built with enterprise-grade architecture, Typed-IDB eliminates IndexedDB's complexity while maintaining full flexibility.

## ✨ Why Typed-IDB?

| Problem with Raw IndexedDB | Typed-IDB Solution |
|---------------------------|-------------------|
| Callback hell | Clean async/await API |
| No TypeScript support | Full type safety with generics |
| Complex transaction management | Automatic transaction handling |
| Manual error handling | Typed errors with recovery patterns |
| No bulk operations | Automatic chunking for large datasets |
| Verbose boilerplate | 70% less code |

### Single vs Batch Operations

```typescript
// ❌ Inefficient: Multiple transactions
for (const user of users) {
  await repo.add("users", user);  // Each creates a new transaction!
}

// ✅ Efficient: Single operation
await repo.addMany("users", users);  // One optimized transaction

// ❌ Inefficient: Multiple queries
const user1 = await repo.getById("users", "id1");
const user2 = await repo.getById("users", "id2");
const user3 = await repo.getById("users", "id3");

// ✅ Efficient: Batch fetch
const users = await repo.getByIds("users", ["id1", "id2", "id3"]);
```

## 🚀 Quick Start

```bash
npm install typed-idb
```

```typescript
import { IndexedDBDatabase, IndexedDBStorageAdapter, UniquelyIdentifiable } from 'typed-idb';

// 1. Define your data
interface User extends UniquelyIdentifiable {
  id: string;
  name: string;
  email: string;
}

// 2. Setup database (one-time)
class AppDB extends IndexedDBDatabase {
  protected handleUpgrade(event: IDBVersionChangeEvent): void {
    const db = (event.target as IDBOpenDBRequest).result;
    const store = db.createObjectStore("users", { keyPath: "id" });
    store.createIndex("email", "email", { unique: true });
  }
}

// 3. Use it!
const adapter = new IndexedDBStorageAdapter<User>(new AppDB("myapp", 1));
const repo = adapter.getRepository();

// Simple operations with full type safety
await repo.add("users", { id: "1", name: "Alice", email: "alice@example.com" });

// Powerful batch operations
await repo.addMany("users", [
  { id: "2", name: "Bob", email: "bob@example.com" },
  { id: "3", name: "Carol", email: "carol@example.com" }
]);

// Efficient batch queries
const users = await repo.getByIds("users", ["1", "2", "3"]);
```

## 🎯 Core Features

### Complete CRUD Operations
```typescript
const repo = adapter.getRepository();

// Bulk operations - perfect for real-world use cases
await repo.addMany("users", newUsers);                    // Add multiple
await repo.updateMany("users", updatedUsers);             // Update multiple  
await repo.deleteMany("users", ["id1", "id2", "id3"]);  // Delete multiple
await repo.clear("users");                               // Clear entire store

// Efficient batch fetching
const specificUsers = await repo.getByIds("users", ["id1", "id2", "id3"]);
```

### Powerful Queries with Indexes
```typescript
const indexRepo = adapter.getIndexRepository();

// Find by exact match
const users = await indexRepo.getAllByIndex(
  "users", "role", IDBKeyRange.only("admin")
);

// Range queries
const recentUsers = await indexRepo.getAllByIndex(
  "users", "createdAt", IDBKeyRange.lowerBound(lastWeek)
);
```

### Smart Bulk Processing
```typescript
// Process thousands of records efficiently
await adapter.getBulkRepository().bulkOperation(
  "users", "readwrite", thousandsOfUsers,
  async (tx, chunk) => {
    // Auto-chunked to prevent memory issues
    const store = tx.objectStore("users");
    return Promise.all(chunk.map(u => IndexedDBUtils.wrapRequest(store.put(u))));
  },
  100 // Process 100 at a time
);
```

### Production-Ready Error Handling
```typescript
try {
  await repo.addMany("users", users);
} catch (error) {
  if (error instanceof ConstraintError) {
    // Duplicate key - maybe switch to updateMany
    await repo.updateMany("users", users);
  } else if (error instanceof QuotaExceededError) {
    // Storage full - clear old data
    await this.clearOldData();
  }
}
```

## 📚 Documentation
### API Quick Reference

```typescript
const repo = adapter.getRepository();

// Read operations
await repo.getAll("store");                    // Get all items
await repo.getById("store", "id");            // Get single item
await repo.getByIds("store", ["id1", "id2"]); // Get multiple items

// Write operations  
await repo.add("store", item);                 // Add single item
await repo.addMany("store", items);            // Add multiple items
await repo.update("store", item);              // Update single item
await repo.updateMany("store", items);         // Update multiple items

// Delete operations
await repo.delete("store", "id");              // Delete single item
await repo.deleteMany("store", ids);           // Delete multiple items
await repo.clear("store");                     // Delete all items
```

#### Index Methods (Queries)

```typescript
const indexRepo = adapter.getIndexRepository();

// Query by index
await indexRepo.getAllByIndex("store", "indexName", IDBKeyRange.only(value));
await indexRepo.getAllByIndex("store", "indexName", IDBKeyRange.bound(min, max));
await indexRepo.getAllByIndex("store", "indexName", IDBKeyRange.lowerBound(min));
await indexRepo.getAllByIndex("store", "indexName", IDBKeyRange.upperBound(max));

// Cursor operations
const cursor = await indexRepo.keyCursor("store", "indexName");
```

#### Bulk Methods (Large Datasets)

```typescript
const bulkRepo = adapter.getBulkRepository();

// Process large arrays in chunks
await bulkRepo.bulkOperation(
  "store",
  "readwrite",
  largeArray,
  async (transaction, chunk) => {
    // Process each chunk in its own transaction
    const store = transaction.objectStore("store");
    return Promise.all(chunk.map(item => 
      IndexedDBUtils.wrapRequest(store.put(item))
    ));
  },
  100 // chunk size
);
```

## 🔄 Migration Guides

### Before (Raw IndexedDB)
```typescript
// ❌ Complex, error-prone, no type safety
const request = indexedDB.open("myDB", 1);
request.onsuccess = (event) => {
  const db = event.target.result;
  const transaction = db.transaction(["users"], "readwrite");
  const store = transaction.objectStore("users");
  const addRequest = store.add(user);
  addRequest.onsuccess = () => console.log("Success!");
};
```

### After (Typed-IDB)
```typescript
// ✅ Simple, type-safe, maintainable
await adapter.getRepository().add("users", user);
```

### Before (localStorage)
```typescript
// ❌ 5MB limit, synchronous, no batch operations
const users = JSON.parse(localStorage.getItem("users") || "[]");
users.push(newUser);
localStorage.setItem("users", JSON.stringify(users));

// Inefficient filtering through all data
const activeUsers = users.filter(u => u.status === "active");
```

### After (Typed-IDB)
```typescript
// ✅ Unlimited storage, async, powerful batch operations
await repo.addMany("users", newUsers);         // Add multiple efficiently
const active = await indexRepo.getAllByIndex(  // Indexed query (fast!)
  "users", "status", IDBKeyRange.only("active")
);
```

## 🏗️ Architecture

Typed-IDB uses a modular, composable architecture:

```typescript
const adapter = new IndexedDBStorageAdapter<T>(database);

adapter.getRepository()       // Basic CRUD operations
adapter.getIndexRepository()  // Index-based queries
adapter.getBulkRepository()   // Bulk operations with chunking
```

### Common Operation Patterns

```typescript
const repo = adapter.getRepository();

// 📥 Import data
await repo.addMany("items", importedItems);

// 🔄 Sync changes
const modified = await repo.getByIds("items", modifiedIds);
await repo.updateMany("items", modified);

// 🗑️ Batch cleanup
await repo.deleteMany("items", expiredIds);

// 🔍 Efficient queries
const results = await adapter.getIndexRepository()
  .getAllByIndex("items", "status", IDBKeyRange.only("active"));

// 💾 Bulk processing
await adapter.getBulkRepository().bulkOperation(
  "items", "readwrite", hugeDataset,
  async (tx, chunk) => processChunk(tx, chunk),
  500 // Optimal chunk size
);
```

**No Lock-in**: You can always drop down to raw IndexedDB when needed:

```typescript
// Use high-level API for most operations
await adapter.getRepository().add("users", user);

// Drop down when you need full control
await database.executeTransaction(["users"], "readwrite", async (tx) => {
  // Direct access to IDBTransaction
  const store = tx.objectStore("users");
  // Custom IndexedDB operations...
});
```

## 🛡️ Error Handling

Typed-IDB provides specific error types for robust error handling:

| Error Type | When It Occurs | Recovery Strategy |
|------------|---------------|-------------------|
| `ConstraintError` | Duplicate keys | Update instead of add |
| `QuotaExceededError` | Storage full | Clear old data |
| `TransactionError` | Transaction failed | Retry with backoff |
| `ConnectionError` | Can't open database | Fallback storage |

## 📦 Installation & Setup

### Requirements
- TypeScript 4.0+
- Modern browser with IndexedDB support

### Package Managers
```bash
# npm
npm install typed-idb

# yarn
yarn add typed-idb

# pnpm
pnpm add typed-idb
```

### Browser Support
- ✅ Chrome 24+
- ✅ Firefox 16+
- ✅ Safari 8+
- ✅ Edge (all versions)

## 💡 Real-World Examples

```typescript
class ShoppingCartService {
  constructor(private adapter: IndexedDBStorageAdapter<CartItem>) {}

  // Add multiple items at once
  async addToCart(items: CartItem[]): Promise<void> {
    await this.adapter.getRepository().addMany("cart", items);
  }

  // Update quantities for multiple items
  async updateQuantities(updates: CartItem[]): Promise<void> {
    await this.adapter.getRepository().updateMany("cart", updates);
  }

  // Remove multiple items
  async removeItems(itemIds: string[]): Promise<void> {
    await this.adapter.getRepository().deleteMany("cart", itemIds);
  }

  // Get specific items for checkout
  async getCheckoutItems(itemIds: string[]): Promise<CartItem[]> {
    return this.adapter.getRepository().getByIds("cart", itemIds);
  }

  // Clear cart after purchase
  async clearCart(): Promise<void> {
    await this.adapter.getRepository().clear("cart");
  }
}
```


```typescript
class SessionManager {
  constructor(private adapter: IndexedDBStorageAdapter<Session>) {}

  // Get all active sessions
  async getActiveSessions(): Promise<Session[]> {
    return this.adapter.getIndexRepository().getAllByIndex(
      "sessions", "status", IDBKeyRange.only("active")
    );
  }

  // Batch logout - invalidate multiple sessions
  async invalidateSessions(userIds: string[]): Promise<void> {
    const sessions = await this.adapter.getRepository().getByIds("sessions", userIds);
    const invalidated = sessions.map(s => ({ ...s, status: "expired" }));
    await this.adapter.getRepository().updateMany("sessions", invalidated);
  }

  // Clean up old sessions
  async cleanupExpiredSessions(): Promise<void> {
    const expired = await this.adapter.getIndexRepository().getAllByIndex(
      "sessions", "expiryDate", IDBKeyRange.upperBound(new Date())
    );
    const ids = expired.map(s => s.id);
    await this.adapter.getRepository().deleteMany("sessions", ids);
  }
}
```

```typescript
class OfflineSyncService {
  constructor(private adapter: IndexedDBStorageAdapter<SyncItem>) {}

  // Queue multiple items for sync
  async queueForSync(items: SyncItem[]): Promise<void> {
    const timestamped = items.map(item => ({
      ...item,
      queuedAt: new Date(),
      syncStatus: 'pending'
    }));
    await this.adapter.getRepository().addMany("syncQueue", timestamped);
  }

  // Process sync batch
  async processSyncBatch(batchSize = 50): Promise<void> {
    const pending = await this.adapter.getIndexRepository().getAllByIndex(
      "syncQueue", "syncStatus", IDBKeyRange.only("pending")
    );
    
    const batch = pending.slice(0, batchSize);
    const batchIds = batch.map(item => item.id);
    
    try {
      // Send to server
      await this.sendToServer(batch);
      
      // Remove successfully synced items
      await this.adapter.getRepository().deleteMany("syncQueue", batchIds);
    } catch (error) {
      // Mark as failed
      const failed = batch.map(item => ({ ...item, syncStatus: 'failed' }));
      await this.adapter.getRepository().updateMany("syncQueue", failed);
    }
  }

  // Clear all synced data
  async clearSyncedData(): Promise<void> {
    await this.adapter.getRepository().clear("syncQueue");
  }
}
```

## 🧪 Testing

The architecture makes testing straightforward:

```typescript
// Mock the database for unit tests
const mockDb = new MockDatabase();
const adapter = new IndexedDBStorageAdapter(mockDb);

// Test your services with mocked storage
const service = new UserService(adapter);
await service.createUser(testUser);
```

[See testing guide →](./docs/testing.md)

## 📈 Performance Tips

### Chunking Strategy
- Simple records: 500 items per chunk
- Complex records: 50-100 items per chunk
- Adjust based on your data size

### Index Design
- Only index fields you query
- Use compound indexes for multi-field queries
- Consider storage overhead vs query speed

### Transaction Scope
- Keep transactions small and focused
- Use readonly when possible
- Batch related operations


# 📚 Additional Documentation

## Getting Started Guide

### Basic Setup

Create your database schema by extending `IndexedDBDatabase`:

```typescript
import { IndexedDBDatabase, UniquelyIdentifiable } from 'typed-idb';

interface Product implements UniquelyIdentifiable {
  id: string;
  name: string;
  price: number;
  category: string;
}

class StoreDatabase extends IndexedDBDatabase {
  protected handleUpgrade(event: IDBVersionChangeEvent): void {
    const db = (event.target as IDBOpenDBRequest).result;
    
    if (!db.objectStoreNames.contains("products")) {
      const store = db.createObjectStore("products", { keyPath: "id" });
      store.createIndex("category", "category");
      store.createIndex("price", "price");
    }
  }
}
```

### Repository Pattern

Use specialized repositories for different operations:

```typescript
const db = new StoreDatabase("store", 1);
const adapter = new IndexedDBStorageAdapter<Product>(db);
const repo = adapter.getRepository();

// Single item operations
await repo.add("products", product);
const product = await repo.getById("products", "prod-123");
await repo.update("products", updatedProduct);
await repo.delete("products", "prod-123");

// Batch operations - much more efficient for multiple items
await repo.addMany("products", newProducts);
const products = await repo.getByIds("products", ["prod-1", "prod-2", "prod-3"]);
await repo.updateMany("products", updatedProducts);
await repo.deleteMany("products", ["prod-1", "prod-2", "prod-3"]);

// Manage entire store
const allProducts = await repo.getAll("products");
await repo.clear("products"); // Remove all products

// Index queries for filtering
const indexRepo = adapter.getIndexRepository();
const electronics = await indexRepo.getAllByIndex(
  "products", 
  "category", 
  IDBKeyRange.only("electronics")
);

// Bulk operations for large datasets
const bulkRepo = adapter.getBulkRepository();
await bulkRepo.bulkOperation(
  "products",
  "readwrite", 
  thousandsOfProducts,
  async (tx, chunk) => {
    const store = tx.objectStore("products");
    return Promise.all(
      chunk.map(p => IndexedDBUtils.wrapRequest(store.add(p)))
    );
  },
  100 // Process 100 at a time
);
```

## Error Handling Examples

### Comprehensive Error Recovery

```typescript
import { 
  ConstraintError, 
  QuotaExceededError, 
  TransactionError 
} from 'typed-idb';

class ResilientService {
  async saveWithRecovery<T>(storeName: string, item: T): Promise<void> {
    try {
      await this.adapter.getRepository().add(storeName, item);
    } catch (error) {
      if (error instanceof ConstraintError) {
        // Item exists, update instead
        await this.adapter.getRepository().update(storeName, item);
      } else if (error instanceof QuotaExceededError) {
        // Clear cache and retry
        await this.clearOldData(storeName);
        await this.adapter.getRepository().add(storeName, item);
      } else if (error instanceof TransactionError) {
        // Retry with exponential backoff
        await this.retryWithBackoff(
          () => this.adapter.getRepository().add(storeName, item)
        );
      } else {
        throw error; // Unknown error, propagate
      }
    }
  }

  private async retryWithBackoff<T>(
    operation: () => Promise<T>,
    maxRetries = 3
  ): Promise<T> {
    for (let i = 0; i < maxRetries; i++) {
      try {
        return await operation();
      } catch (error) {
        if (i === maxRetries - 1) throw error;
        await new Promise(r => setTimeout(r, 100 * Math.pow(2, i)));
      }
    }
    throw new Error("Max retries exceeded");
  }
}
```

## Migration Examples

### Complete localStorage Migration

```typescript
class MigrationService {
  async migrateFromLocalStorage(): Promise<void> {
    // 1. Setup new IndexedDB database
    const db = new AppDatabase("app", 1);
    const adapter = new IndexedDBStorageAdapter<User>(db);
    const repo = adapter.getRepository();
    
    // 2. Read existing localStorage data
    const keys = Object.keys(localStorage).filter(k => k.startsWith("user_"));
    const users: User[] = keys.map(key => {
      const data = localStorage.getItem(key);
      return data ? JSON.parse(data) : null;
    }).filter(Boolean);
    
    // 3. Bulk import to IndexedDB (much faster than adding one by one!)
    if (users.length > 0) {
      await repo.addMany("users", users);  // Single efficient operation
      
      // 4. Verify migration
      const imported = await repo.getByIds("users", users.map(u => u.id));
      console.log(`✅ Migrated ${imported.length} users to IndexedDB`);
      
      // 5. Clean up localStorage after successful migration
      keys.forEach(key => localStorage.removeItem(key));
    }
  }
  
  // Bonus: Now you can do things localStorage never could
  async demonstrateNewCapabilities(): Promise<void> {
    const repo = adapter.getRepository();
    const indexRepo = adapter.getIndexRepository();
    
    // Batch operations
    await repo.updateMany("users", updatedUsers);
    await repo.deleteMany("users", inactiveUserIds);
    
    // Indexed queries (impossible with localStorage!)
    const admins = await indexRepo.getAllByIndex(
      "users", "role", IDBKeyRange.only("admin")
    );
    
    // Clear specific data without affecting other stores
    await repo.clear("temporaryData");
  }
}
```

## Advanced Patterns

### Multi-Store Transactions

```typescript
async function transferFunds(
  fromAccountId: string,
  toAccountId: string,
  amount: number
): Promise<void> {
  await db.executeTransaction(
    ["accounts", "transactions"],
    "readwrite",
    async (tx) => {
      const accountStore = tx.objectStore("accounts");
      const transactionStore = tx.objectStore("transactions");
      
      // Get accounts
      const fromAccount = await IndexedDBUtils.wrapRequest<Account>(
        accountStore.get(fromAccountId)
      );
      const toAccount = await IndexedDBUtils.wrapRequest<Account>(
        accountStore.get(toAccountId)
      );
      
      // Validate
      if (!fromAccount || !toAccount) {
        throw new Error("Account not found");
      }
      if (fromAccount.balance < amount) {
        throw new Error("Insufficient funds");
      }
      
      // Update balances
      fromAccount.balance -= amount;
      toAccount.balance += amount;
      
      // Save updates
      await IndexedDBUtils.wrapRequest(accountStore.put(fromAccount));
      await IndexedDBUtils.wrapRequest(accountStore.put(toAccount));
      
      // Record transaction
      const transaction = {
        id: crypto.randomUUID(),
        from: fromAccountId,
        to: toAccountId,
        amount,
        timestamp: new Date()
      };
      await IndexedDBUtils.wrapRequest(transactionStore.add(transaction));
    }
  );
}
```

### Custom Query Builder

```typescript
class QueryBuilder<T extends UniquelyIdentifiable> {
  constructor(
    private adapter: IndexedDBStorageAdapter<T>,
    private storeName: string
  ) {}

  where(indexName: string, value: any): QueryFilter<T> {
    return new QueryFilter(this.adapter, this.storeName, indexName, value);
  }
}

class QueryFilter<T extends UniquelyIdentifiable> {
  constructor(
    private adapter: IndexedDBStorageAdapter<T>,
    private storeName: string,
    private indexName: string,
    private value: any
  ) {}

  async equals(): Promise<T[]> {
    return this.adapter.getIndexRepository().getAllByIndex(
      this.storeName,
      this.indexName,
      IDBKeyRange.only(this.value)
    );
  }

  async between(upper: any): Promise<T[]> {
    return this.adapter.getIndexRepository().getAllByIndex(
      this.storeName,
      this.indexName,
      IDBKeyRange.bound(this.value, upper)
    );
  }

  async greaterThan(): Promise<T[]> {
    return this.adapter.getIndexRepository().getAllByIndex(
      this.storeName,
      this.indexName,
      IDBKeyRange.lowerBound(this.value, true)
    );
  }
}

// Usage
const query = new QueryBuilder(adapter, "products");
const expensiveProducts = await query.where("price", 100).greaterThan();
const midRangeProducts = await query.where("price", 50).between(150);
```