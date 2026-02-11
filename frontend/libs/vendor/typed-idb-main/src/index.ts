/**
 * @fileoverview IndexedDB Storage Adapter Module
 * 
 * @module IndexedDBStorage
 * 
 * @description
 * A comprehensive storage adapter implementation for working with IndexedDB in browser
 * environments. This module provides a well-structured, SOLID-compliant approach to
 * IndexedDB interactions with strong TypeScript typing.
 * 
 * The module follows these design principles:
 * - Single Responsibility Principle (SRP): Each class has a clear, focused purpose
 * - Interface Segregation Principle (ISP): Specific interfaces for different operations
 * - Dependency Inversion Principle (DIP): Dependencies on abstractions, not implementations
 * - Open/Closed Principle (OCP): Extensible design without requiring modification
 * - Composition over Inheritance: Building complex functionality through component composition
 * - DRY (Don't Repeat Yourself): Common patterns extracted into reusable components
 * 
 * @example
 * // Setting up and using the IndexedDB adapter
 * 
 * // 1. Create a database implementation by extending IndexedDBDatabase
 * class AppDatabase extends IndexedDBDatabase {
 *   protected handleUpgrade(event: IDBVersionChangeEvent): void {
 *     const db = (event.target as IDBOpenDBRequest).result;
 *     
 *     // Create object stores and indexes
 *     if (!db.objectStoreNames.contains("users")) {
 *       const userStore = db.createObjectStore("users", { keyPath: "id" });
 *       userStore.createIndex("email", "email", { unique: true });
 *     }
 *   }
 * }
 * 
 * // 2. Create the adapter with your database implementation
 * const db = new AppDatabase("my-app-db", 1);
 * const adapter = new IndexedDBStorageAdapter<User>(db);
 * 
 * // 3. Perform operations using the appropriate repository
 * async function saveUser(user: User) {
 *   return adapter.getRepository().add("users", user);
 * }
 * 
 * async function getUsersByEmail(email: string) {
 *   return adapter.getIndexRepository().getAllByIndex(
 *     "users", 
 *     "email", 
 *     IDBKeyRange.only(email)
 *   );
 * }
 * 
 * async function bulkUpdateUsers(users: User[]) {
 *   return adapter.getBulkRepository().bulkOperation(
 *     "users",
 *     "readwrite",
 *     users,
 *     async (tx, chunk) => {
 *       const store = tx.objectStore("users");
 *       return Promise.all(chunk.map(user => {
 *         return IndexedDBUtils.wrapRequest<IDBValidKey>(store.put(user));
 *       }));
 *     }
 *   );
 * }
 * 
 * @example
 * // Error handling example
 * async function safeOperation() {
 *   try {
 *     const users = await adapter.getRepository().getAll("users");
 *     return users;
 *   } catch (error) {
 *     console.error("Database operation failed:", error);
 *     // Implement fallback strategy or error reporting
 *     return [];
 *   }
 * }
 * 
 * @example
 * // Using transactions across multiple stores
 * async function transferData(sourceId: string, targetId: string) {
 *   return db.executeTransaction(
 *     ["sourceStore", "targetStore"],
 *     "readwrite",
 *     async (tx) => {
 *       const sourceStore = tx.objectStore("sourceStore");
 *       const targetStore = tx.objectStore("targetStore");
 *       
 *       // Read from one store
 *       const sourceData = await IndexedDBUtils.wrapRequest<SourceData>(
 *         sourceStore.get(sourceId)
 *       );
 *       
 *       if (!sourceData) {
 *         throw new Error("Source data not found");
 *       }
 *       
 *       // Transform data if needed
 *       const targetData = {
 *         id: targetId,
 *         content: sourceData.content,
 *         timestamp: new Date()
 *       };
 *       
 *       // Write to another store in the same transaction
 *       await IndexedDBUtils.wrapRequest<IDBValidKey>(
 *         targetStore.add(targetData)
 *       );
 *       
 *       return targetData;
 *     }
 *   );
 * }
 * 
 * @see {@link https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API|MDN IndexedDB Documentation}
 * 
 * @author Doug Fenstermacher
 * 
 * @version 1.0.0
 * 
 * @license MIT
 */
"use strict";

export type UniqueIdentifier = string | number;

export interface UniquelyIdentifiable {
    id: UniqueIdentifier;
}

export interface DatabaseInformation {
  name: string;
  version: number;
  objectStores: string[];
}

/**
 * @interface IDatabase
 * 
 * @description
 * Defines the core database connection and transaction management operations.
 * This interface abstracts the details of establishing connections and managing
 * transactions with the underlying database system.
 * 
 * The interface follows the Repository pattern, providing a higher-level abstraction
 * over the low-level IndexedDB API. It handles connection management, transaction
 * creation, and error handling in a consistent way.
 * 
 * @example
 * // Using the database interface for a transaction
 * async function complexOperation(db: IDatabase) {
 *   return db.executeTransaction(
 *     ["store1", "store2"], // Object stores involved
 *     "readwrite",          // Transaction mode
 *     async (tx) => {
 *       // Perform multiple operations in a single transaction
 *       const store1 = tx.objectStore("store1");
 *       const store2 = tx.objectStore("store2");
 *       
 *       // Read from store1
 *       const data = await someAsyncOperation(store1);
 *       
 *       // Write to store2 based on results
 *       await anotherAsyncOperation(store2, data);
 *       
 *       return "Transaction completed successfully";
 *     }
 *   );
 * }
 * 
 * @remarks
 * The implementation handles all the complexity of:
 * - Database connection management (creation, opening, error handling)
 * - Transaction lifecycle management
 * - Promise-based API wrapping around the callback-based IndexedDB API
 * 
 * Implementations should ensure proper cleanup of resources and handle edge cases
 * like connection errors or transaction aborts.
 */
export interface IDatabase {
  openConnection(): Promise<IDBDatabase>;
  executeTransaction<T>(
    storeNames: string[],
    mode: IDBTransactionMode,
    operation: (tx: IDBTransaction) => Promise<T>
  ): Promise<T>;
  getInfo(): Promise<DatabaseInformation>;
}

/**
 * @interface IStorageRepository
 * 
 * @description
 * Defines the core CRUD (Create, Read, Update, Delete) operations for storage repositories.
 * This interface provides a consistent pattern for working with stored entities regardless
 * of the underlying storage mechanism.
 * 
 * All methods are designed to be asynchronous, returning Promises to accommodate the
 * asynchronous nature of IndexedDB operations.
 * 
 * @template T - The type of entity being stored, must have a unique identifier
 * 
 * @example
 * // Basic usage example
 * async function userExample(repo: IStorageRepository<User>) {
 *   // Get all users
 *   const allUsers = await repo.getAll("users");
 *   
 *   // Create a new user
 *   const newUser = { id: "user123", name: "Alice", email: "alice@example.com" };
 *   await repo.add("users", newUser);
 *   
 *   // Get a specific user
 *   const user = await repo.getById("users", "user123");
 *   
 *   // Update a user
 *   if (user) {
 *     user.name = "Alice Smith";
 *     await repo.update("users", user);
 *   }
 *   
 *   // Delete a user
 *   await repo.delete("users", "user123");
 * }
 * 
 * @remarks
 * All methods operate within transactions appropriate to their operation (readonly for
 * queries, readwrite for modifications). For operations that need to span multiple
 * methods or stores, consider using the executeTransaction method from IDatabase.
 */
export interface IStorageRepository<T extends UniquelyIdentifiable> {
  getAll(storeName: string): Promise<T[]>;
  getById(storeName: string, id: string | number): Promise<T | null>;
  getByIds(storeName: string, ids: (string | number)[]): Promise<T[]>;
  add(storeName: string, item: T): Promise<IDBValidKey>;
  addMany(storeName: string, items: T[]): Promise<IDBValidKey[]>;
  update(storeName: string, item: T): Promise<IDBValidKey>;
  updateMany(storeName: string, items: T[]): Promise<IDBValidKey[]>;
  delete(storeName: string, id: string | number): Promise<void>;
  deleteMany(storeName: string, ids: (string | number)[]): Promise<void>;
  clear(storeName: string): Promise<void>;
}

/**
 * @interface IIndexOperations
 * 
 * @description
 * Defines operations that leverage IndexedDB's indexing capabilities for efficient
 * data retrieval and filtering. This interface provides methods for querying data
 * based on secondary indexes rather than primary keys.
 * 
 * IndexedDB indexes are similar to database indexes in traditional relational databases,
 * allowing for quick lookups on non-primary-key fields. This interface encapsulates
 * the patterns for working with these indexes in a type-safe manner.
 * 
 * @template T - The type of entity being queried, must have a unique identifier
 * 
 * @example
 * // Querying users by email using an index
 * async function findUsersByEmail(repo: IIndexOperations<User>, email: string) {
 *   // Get users with exact email match
 *   const users = await repo.getAllByIndex(
 *     "users",           // Object store name
 *     "emailIndex",      // Index name
 *     IDBKeyRange.only(email) // Key range for exact match
 *   );
 *   
 *   return users;
 * }
 * 
 * @example
 * // Finding users within an age range
 * async function findUsersInAgeRange(
 *   repo: IIndexOperations<User>, 
 *   minAge: number, 
 *   maxAge: number
 * ) {
 *   const users = await repo.getAllByIndex(
 *     "users",
 *     "ageIndex",
 *     IDBKeyRange.bound(minAge, maxAge)
 *   );
 *   
 *   return users;
 * }
 * 
 * @example
 * // Using a key cursor to iterate through keys efficiently
 * async function getAllUserEmails(repo: IIndexOperations<User>) {
 *   const emails: string[] = [];
 *   let cursor = await repo.keyCursor("users", "emailIndex");
 *   
 *   while (cursor) {
 *     emails.push(cursor.key as string);
 *     await cursor.continue();
 *     // The cursor will be updated automatically
 *   }
 *   
 *   return emails;
 * }
 * 
 * @remarks
 * Index operations are particularly useful for:
 * - Filtering data by non-primary-key attributes
 * - Implementing search functionality
 * - Sorting data in a specific order
 * - Creating distinct value lists
 * 
 * When designing indexes, consider:
 * - Only index attributes that will be frequently queried
 * - Use compound indexes for queries that filter on multiple fields
 * - Set appropriate uniqueness constraints on indexes
 * 
 * Indexes must be defined during the database schema setup in the
 * handleUpgrade method of your IndexedDBDatabase implementation.
 */
export interface IIndexOperations<T extends UniquelyIdentifiable> {
  getAllByIndex(storeName: string, indexName: string, range?: IDBKeyRange): Promise<T[]>;
  keyCursor(storeName: string, indexName: string): Promise<IDBCursor>;
}

/**
 * @interface IBulkOperations
 * 
 * @description
 * Provides specialized operations for handling large datasets in IndexedDB.
 * This interface focuses on efficient processing of multiple entities in
 * batches to overcome browser limitations and optimize performance.
 * 
 * Bulk operations are crucial for handling large datasets in browser
 * environments, where transaction timeouts and memory limitations can
 * affect application reliability. This interface implements the chunking
 * pattern to process large datasets in manageable pieces.
 * 
 * @template T - The type of entity being processed, must have a unique identifier
 * 
 * @example
 * // Updating many records with a status change
 * async function markUsersAsInactive(
 *   repo: IBulkOperations<User>,
 *   userIds: string[]
 * ) {
 *   // First, get all users
 *   const db = await getDatabase();
 *   const users = await db.executeTransaction(
 *     ["users"], "readonly",
 *     async (tx) => {
 *       const store = tx.objectStore("users");
 *       const results: User[] = [];
 *       
 *       for (const id of userIds) {
 *         const request = store.get(id);
 *         const user = await IndexedDBUtils.wrapRequest<User>(request);
 *         if (user) {
 *           results.push(user);
 *         }
 *       }
 *       
 *       return results;
 *     }
 *   );
 *   
 *   // Update status and save in chunks
 *   const updatedUsers = users.map(user => ({
 *     ...user,
 *     status: 'inactive',
 *     lastModified: new Date()
 *   }));
 *   
 *   return repo.bulkOperation(
 *     "users",
 *     "readwrite",
 *     updatedUsers,
 *     async (tx, chunk) => {
 *       const store = tx.objectStore("users");
 *       return Promise.all(chunk.map(user => {
 *         return IndexedDBUtils.wrapRequest<IDBValidKey>(store.put(user));
 *       }));
 *     },
 *     50 // Process in chunks of 50 users
 *   );
 * }
 * 
 * @example
 * // Importing data from an external source
 * async function importUserData(
 *   repo: IBulkOperations<User>,
 *   userData: User[]
 * ) {
 *   return repo.bulkOperation(
 *     "users",
 *     "readwrite",
 *     userData,
 *     async (tx, chunk) => {
 *       const store = tx.objectStore("users");
 *       
 *       // Process each item with validation
 *       const results: IDBValidKey[] = [];
 *       for (const user of chunk) {
 *         // Perform validation or transformation
 *         if (!user.email) {
 *           continue; // Skip invalid records
 *         }
 *         
 *         // Add valid records
 *         const key = await IndexedDBUtils.wrapRequest<IDBValidKey>(
 *           store.add(user)
 *         );
 *         results.push(key);
 *       }
 *       
 *       return results;
 *     }
 *   );
 * }
 * 
 * @remarks
 * The bulkOperation method is the core of this interface, providing:
 * 
 * - Automatic chunking: Splits large arrays into manageable pieces
 * - Transaction management: Each chunk gets its own transaction
 * - Error handling: Failed chunks don't affect other chunks
 * - Progress tracking: Results accumulate as chunks complete
 * 
 * When using bulk operations, consider:
 * - Appropriate chunk sizes (default is 100)
 * - Transaction timeout limits in the browser
 * - Memory usage when processing large datasets
 * - Error recovery strategies for partial failures
 * 
 * This interface is particularly useful for:
 * - Data import/export functionality
 * - Batch updates to many records
 * - Initial data population of IndexedDB stores
 * - Synchronization with server-side databases
 */
export interface IBulkOperations<T extends UniquelyIdentifiable> {
  bulkOperation<R>(
    storeName: string,
    mode: IDBTransactionMode,
    items: T[],
    operation: (tx: IDBTransaction, chunk: T[]) => Promise<R[]>,
    chunkSize?: number
  ): Promise<R[]>;
}

/**
 * @class IndexedDBDatabase
 * 
 * @description
 * Abstract base class that manages the connection to an IndexedDB database and
 * provides utilities for transaction management. This class implements the IDatabase
 * interface and handles the low-level details of working with the IndexedDB API.
 * 
 * This class serves as the foundation of the storage adapter system, providing:
 * - Database connection management
 * - Schema versioning and upgrades
 * - Transaction execution with Promise-based API
 * 
 * The abstract design requires concrete implementations to define the database schema
 * through the handleUpgrade method, while providing common functionality for all
 * database operations.
 * 
 * @implements {IDatabase}
 * 
 * @example
 * // Creating a concrete database implementation
 * class TodoDatabase extends IndexedDBDatabase {
 *   protected handleUpgrade(event: IDBVersionChangeEvent): void {
 *     const db = (event.target as IDBOpenDBRequest).result;
 *     const oldVersion = event.oldVersion;
 *     
 *     // Create and update schema based on version
 *     if (oldVersion < 1) {
 *       // Version 1: Initial schema
 *       const todoStore = db.createObjectStore("todos", { keyPath: "id" });
 *       todoStore.createIndex("status", "status", { unique: false });
 *       todoStore.createIndex("dueDate", "dueDate", { unique: false });
 *     }
 *     
 *     if (oldVersion < 2) {
 *       // Version 2: Add tags support
 *       const todoStore = event.transaction.objectStore("todos");
 *       todoStore.createIndex("tags", "tags", { unique: false, multiEntry: true });
 *     }
 *   }
 * }
 * 
 * const db = new TodoDatabase("todo-app", 2);
 * 
 * @example
 * // Using the executeTransaction method for complex operations
 * async function moveTodoToList(
 *   db: IndexedDBDatabase,
 *   todoId: string,
 *   fromListId: string,
 *   toListId: string
 * ) {
 *   return db.executeTransaction(
 *     ["todos", "lists"], // Multiple stores in one transaction
 *     "readwrite",
 *     async (tx) => {
 *       const todoStore = tx.objectStore("todos");
 *       const listStore = tx.objectStore("lists");
 *       
 *       // Get the todo
 *       const todo = await IndexedDBUtils.wrapRequest<Todo>(
 *         todoStore.get(todoId)
 *       );
 *       
 *       if (!todo) {
 *         throw new Error("Todo not found");
 *       }
 *       
 *       // Update the todo's list reference
 *       todo.listId = toListId;
 *       await IndexedDBUtils.wrapRequest(todoStore.put(todo));
 *       
 *       // Update counts on both lists
 *       const fromList = await IndexedDBUtils.wrapRequest<TodoList>(
 *         listStore.get(fromListId)
 *       );
 *       if (fromList) {
 *         fromList.todoCount--;
 *         await IndexedDBUtils.wrapRequest(listStore.put(fromList));
 *       }
 *       
 *       const toList = await IndexedDBUtils.wrapRequest<TodoList>(
 *         listStore.get(toListId)
 *       );
 *       if (toList) {
 *         toList.todoCount++;
 *         await IndexedDBUtils.wrapRequest(listStore.put(toList));
 *       }
 *       
 *       return todo;
 *     }
 *   );
 * }
 * 
 * @remarks
 * This class is designed as abstract to force implementations to define their
 * schema upgrade logic. This ensures that each application has full control over
 * its database schema while benefiting from shared connection management logic.
 * 
 * Key design considerations:
 * - Lazily creates and caches the database connection
 * - Abstracts away the complex Promise/callback handling of IndexedDB
 * - Provides transaction management with automatic error handling
 * - Supports schema versioning and migrations
 * 
 * For browser compatibility, the static isSupported method should be called
 * before attempting to use this class, as some browsers or environments may
 * not support IndexedDB.
 * 
 * @abstract
 */
export abstract class IndexedDBDatabase implements IDatabase {
  private dbName: string;
  private dbVersion: number;
  private dbPromise: Promise<IDBDatabase> | null = null;

  constructor(dbName: string, dbVersion: number) {
    this.dbName = dbName;
    this.dbVersion = dbVersion;
  }

  /**
   * Opens a connection to the database
   */
  public openConnection(): Promise<IDBDatabase> {
    if (!this.dbPromise) {
      this.dbPromise = new Promise<IDBDatabase>((resolve, reject) => {
        const request = indexedDB.open(this.dbName, this.dbVersion);
        request.onupgradeneeded = this.handleUpgrade.bind(this);
        request.onsuccess = (event: Event) => {
          const db = (event.target as IDBOpenDBRequest).result;
          resolve(db);
        };
        request.onerror = (e: Event) => {
          console.error("Database error:", (e as any).error);
          reject(e);
        };
      });
    }
    return this.dbPromise;
  }

  /**
   * Executes a transaction on the database
   */
  public async executeTransaction<T>(
    storeNames: string[],
    mode: IDBTransactionMode,
    operation: (tx: IDBTransaction) => Promise<T>
  ): Promise<T> {
    const db = await this.openConnection();
    const transaction = db.transaction(storeNames, mode);
    
    return new Promise<T>((resolve, reject) => {
      try {
        const result = operation(transaction);
        result.then(resolve).catch(reject);
        
        transaction.onerror = (event) => {
          reject(event);
        };
      } catch (error) {
        reject(error);
      }
    });
  }

  /**
   * Gets information about the database
   */
  public async getInfo(): Promise<DatabaseInformation> {
    const db = await this.openConnection();
    return {
      name: db.name,
      version: db.version,
      objectStores: Array.from(db.objectStoreNames)
    };
  }

  /**
   * Abstract method for handling database schema upgrades
   */
  protected abstract handleUpgrade(event: IDBVersionChangeEvent): void;

  /**
   * Checks if IndexedDB is supported
   */
  public static isSupported(): boolean {
    return !!window.indexedDB;
  }
}


// Custom error classes with proper inheritance
export class DatabaseError extends Error {
  constructor(message: string, public readonly cause?: Error | DOMException) {
    super(message);
    this.name = this.constructor.name;
  }
}

export class ConstraintError extends DatabaseError {}
export class QuotaExceededError extends DatabaseError {}
export class TransactionError extends DatabaseError {}
export class ConnectionError extends DatabaseError {}

/**
 * @class IndexedDBUtils
 * 
 * @description
 * Provides utility functions for common IndexedDB operations.
 * This class contains static methods that simplify working with the
 * callback-based IndexedDB API by wrapping operations in Promises.
 * 
 * These utilities help maintain the DRY principle across the codebase
 * by centralizing common patterns like request handling.
 * 
 * @example
 * // Convert an IDBRequest to a Promise
 * const store = transaction.objectStore("users");
 * const request = store.get(userId);
 * 
 * try {
 *   const user = await IndexedDBUtils.wrapRequest<User>(request);
 *   console.log("Found user:", user);
 * } catch (error) {
 *   console.error("Failed to get user:", error);
 * }
 * 
 * @static
 * 
 * @remarks
 * All methods in this class are static and can be used without instantiation.
 * This class follows the Utility pattern, providing stateless helper functions
 * that can be used anywhere in the codebase.
 */
export class IndexedDBUtils {
  /**
   * Creates a promise-based wrapper for an IDBRequest
   */
  public static wrapRequest<T>(request: IDBRequest): Promise<T> {
    return new Promise<T>((resolve, reject) => {
      request.onsuccess = (event: Event) => {
        resolve((event.target as IDBRequest).result);
      };
      request.onerror = (_event: Event) => {
        const idbError = request.error;

        if (!idbError) {
            reject(new DatabaseError('Unknown database error occurred'));
            return;
        }

        switch (idbError?.name) {
          case 'ConstraintError':
            reject(new ConstraintError('Constraint violation occurred', idbError));
            break;
          case 'QuotaExceededError':
            reject(new QuotaExceededError('Storage quota exceeded', idbError));
            break;
          case 'TransactionInactiveError':
            reject(new TransactionError('Transaction is no longer active', idbError));
            break;
          default:
            reject(new DatabaseError(`Database operation failed: ${idbError?.message}`, idbError));
          }
        };
    });
  }
}

/**
 * Base repository for IndexedDB CRUD operations
 */
export class IndexedDBRepository<T extends UniquelyIdentifiable> implements IStorageRepository<T> {
  protected database: IDatabase;

  constructor(database: IDatabase) {
    this.database = database;
  }

  /**
   * Retrieves all items from a store
   */
  public async getAll(storeName: string): Promise<T[]> {
    return this.database.executeTransaction<T[]>(
      [storeName], 
      "readonly",
      (tx) => {
        const store = tx.objectStore(storeName);
        const request = store.getAll();
        return IndexedDBUtils.wrapRequest<T[]>(request);
      }
    );
  }

  /**
   * Retrieves a single item by ID
   */
  public async getById(storeName: string, id: string | number): Promise<T | null> {
    return this.database.executeTransaction<T | null>(
      [storeName], 
      "readonly",
      async (tx) => {
        const store = tx.objectStore(storeName);
        const request = store.get(id);
        const result = await IndexedDBUtils.wrapRequest<T | undefined>(request);
        return result || null;
      }
    );
  }

  /**
   * Retrieves multiple items by their IDs
   */
  public async getByIds(storeName: string, ids: (string | number)[]): Promise<T[]> {
    return this.database.executeTransaction<T[]>(
      [storeName], 
      "readonly",
      async (tx) => {
        const store = tx.objectStore(storeName);
        const results: T[] = [];
        
        for (const id of ids) {
          const request = store.get(id);
          const result = await IndexedDBUtils.wrapRequest<T | undefined>(request);
          if (result) {
            results.push(result);
          }
        }
        
        return results;
      }
    );
  }

  /**
   * Adds a new item to the store
   */
  public async add(storeName: string, item: T): Promise<IDBValidKey> {
    return this.database.executeTransaction<IDBValidKey>(
      [storeName], 
      "readwrite",
      (tx) => {
        const store = tx.objectStore(storeName);
        const request = store.add(item);
        return IndexedDBUtils.wrapRequest<IDBValidKey>(request);
      }
    );
  }

  /**
   * Adds multiple items to the store
   */
  public async addMany(storeName: string, items: T[]): Promise<IDBValidKey[]> {
    return this.database.executeTransaction<IDBValidKey[]>(
      [storeName], 
      "readwrite",
      async (tx) => {
        const store = tx.objectStore(storeName);
        return Promise.all(
            items.map(item => {
                const request = store.add(item);
                return IndexedDBUtils.wrapRequest<IDBValidKey>(request);
            })
        );
      }
    );
  }

  /**
   * Updates an existing item in the store
   */
  public async update(storeName: string, item: T): Promise<IDBValidKey> {
    return this.database.executeTransaction<IDBValidKey>(
      [storeName], 
      "readwrite",
      (tx) => {
        const store = tx.objectStore(storeName);
        const request = store.put(item);
        return IndexedDBUtils.wrapRequest<IDBValidKey>(request);
      }
    );
  }

  /**
   * Updates multiple items in the store
   */
  public async updateMany(storeName: string, items: T[]): Promise<IDBValidKey[]> {
    return this.database.executeTransaction<IDBValidKey[]>(
      [storeName], 
      "readwrite",
      async (tx) => {
        const store = tx.objectStore(storeName);

        return Promise.all(
            items.map(item => {
                const request = store.put(item);
                return IndexedDBUtils.wrapRequest<IDBValidKey>(request);
            })
        );
      }
    );
  }

  /**
   * Deletes an item from the store
   */
  public async delete(storeName: string, id: string | number): Promise<void> {
    return this.database.executeTransaction<void>(
      [storeName], 
      "readwrite",
      async (tx) => {
        const store = tx.objectStore(storeName);
        const request = store.delete(id);
        await IndexedDBUtils.wrapRequest<any>(request);
      }
    );
  }

  /**
   * Deletes multiple items from the store
   */
  public async deleteMany(storeName: string, ids: (string | number)[]): Promise<void> {
    return this.database.executeTransaction<void>(
      [storeName], 
      "readwrite",
      async (tx) => {
        const store = tx.objectStore(storeName);

        Promise.all(
            ids.map(id => {
                const request = store.delete(id);
                return IndexedDBUtils.wrapRequest<IDBValidKey>(request);
            })
        );
      }
    );
  }

  /**
   * Clears all records from a store
   */
  public async clear(storeName: string): Promise<void> {
    return this.database.executeTransaction<void>(
      [storeName], 
      "readwrite",
      (tx) => {
        const store = tx.objectStore(storeName);
        const request = store.clear();
        return IndexedDBUtils.wrapRequest<void>(request);
      }
    );
  }
}

/**
 * @class IndexedDBIndexRepository
 * 
 * @description
 * Concrete implementation of the IIndexOperations interface for IndexedDB.
 * This class specializes in index-based queries, providing methods for
 * efficiently retrieving data using secondary indexes in IndexedDB object stores.
 * 
 * The repository leverages IndexedDB's native indexing capabilities to enable
 * efficient queries on non-primary-key attributes, similar to database indexes
 * in traditional relational database systems.
 * 
 * @template T - The type of entity being queried, must extend UniquelyIdentifiable
 * 
 * @implements {IIndexOperations<T>}
 * 
 * @example
 * // Creating and using an index repository
 * const db = new AppDatabase("app-db", 1);
 * const userIndexRepo = new IndexedDBIndexRepository<User>(db);
 * 
 * // Find users by role
 * const admins = await userIndexRepo.getAllByIndex(
 *   "users",
 *   "roleIndex",
 *   IDBKeyRange.only("admin")
 * );
 * 
 * // Find users created in the last week
 * const oneWeekAgo = new Date();
 * oneWeekAgo.setDate(oneWeekAgo.getDate() - 7);
 * 
 * const recentUsers = await userIndexRepo.getAllByIndex(
 *   "users",
 *   "createdAtIndex",
 *   IDBKeyRange.lowerBound(oneWeekAgo)
 * );
 * 
 * @example
 * // Using a cursor to process large datasets efficiently
 * async function countUsersByDomain(
 *   repo: IndexedDBIndexRepository<User>
 * ) {
 *   const domains = new Map<string, number>();
 *   let cursor = await repo.keyCursor("users", "emailIndex");
 *   
 *   while (cursor) {
 *     const email = cursor.key as string;
 *     const domain = email.split('@')[1];
 *     
 *     if (domain) {
 *       const count = domains.get(domain) || 0;
 *       domains.set(domain, count + 1);
 *     }
 *     
 *     await cursor.continue();
 *   }
 *   
 *   return domains;
 * }
 * 
 * @remarks
 * This repository implementation:
 * 
 * - Provides index-based query functionality beyond simple primary key lookups
 * - Uses the database's executeTransaction method for all operations
 * - Supports various IDBKeyRange configurations for flexible querying
 * 
 * For optimal use of this repository:
 * 
 * - Ensure appropriate indexes are created during database initialization
 * - Consider the performance implications of indexes (storage overhead vs. query speed)
 * - Use appropriate key ranges for filtering data efficiently
 * - Consider cursor-based operations for handling large result sets with lower memory usage
 * 
 * Indexes must be defined during the database schema setup in the
 * handleUpgrade method of your IndexedDBDatabase implementation.
 */
export class IndexedDBIndexRepository<T extends UniquelyIdentifiable> implements IIndexOperations<T> {
  protected database: IDatabase;

  constructor(database: IDatabase) {
    this.database = database;
  }

  /**
   * Retrieves items filtered by an index
   */
  public async getAllByIndex(
    storeName: string, 
    indexName: string, 
    range?: IDBKeyRange
  ): Promise<T[]> {
    return this.database.executeTransaction<T[]>(
      [storeName], 
      "readonly",
      (tx) => {
        const store = tx.objectStore(storeName);
        const index = store.index(indexName);
        const request = range ? index.getAll(range) : index.getAll();
        return IndexedDBUtils.wrapRequest<T[]>(request);
      }
    );
  }

  /**
   * Opens a key cursor on an index
   */
  public async keyCursor(storeName: string, indexName: string): Promise<IDBCursor> {
    return this.database.executeTransaction<IDBCursor>(
      [storeName], 
      "readonly",
      (tx) => {
        const store = tx.objectStore(storeName);
        const index = store.index(indexName);
        const request = index.openKeyCursor();
        return IndexedDBUtils.wrapRequest<IDBCursor>(request);
      }
    );
  }
}

/**
 * Repository for bulk operations
 */
export class IndexedDBBulkRepository<T extends UniquelyIdentifiable> implements IBulkOperations<T> {
  protected database: IDatabase;

  constructor(database: IDatabase) {
    this.database = database;
  }

  /**
   * Performs a bulk operation on items in chunks
   */
  public async bulkOperation<R>(
    storeName: string,
    mode: IDBTransactionMode,
    items: T[],
    operation: (tx: IDBTransaction, chunk: T[]) => Promise<R[]>,
    chunkSize = 100
  ): Promise<R[]> {
    // Create chunks of items
    const chunks: T[][] = [];
    for (let i = 0; i < items.length; i += chunkSize) {
      chunks.push(items.slice(i, i + chunkSize));
    }
    
    // Process each chunk in its own transaction
    const chunkResults = await Promise.all(chunks.map(chunk => {
      return this.database.executeTransaction<R[]>(
        [storeName], 
        mode,
        (tx) => operation(tx, chunk)
      )
    }));
    
    return chunkResults.reduce((acc, chunk) => [...acc, ...chunk], []);
  }
}

/**
 * @class IndexedDBStorageAdapter
 * 
 * @description
 * The main storage adapter for IndexedDB operations, combining multiple specialized
 * repositories to provide a comprehensive API for database interactions.
 * 
 * This class serves as a facade over the detailed implementation of IndexedDB operations,
 * providing a simple, consistent interface for clients while maintaining separation of
 * concerns internally.
 * 
 * It uses composition to combine specialized repositories:
 * - Basic CRUD operations (IStorageRepository)
 * - Index-based operations (IIndexOperations)
 * - Bulk operations (IBulkOperations)
 * 
 * @template T - The type of entity being stored, must implement UniquelyIdentifiable
 * 
 * @example
 * // Creating and using the adapter
 * const db = new MyAppDatabase("app-database", 1);
 * const adapter = new IndexedDBStorageAdapter<User>(db);
 * 
 * // Perform basic CRUD operations
 * const userRepo = adapter.getRepository();
 * await userRepo.add("users", newUser);
 * 
 * // Perform index-based operations
 * const indexRepo = adapter.getIndexRepository();
 * const activeUsers = await indexRepo.getAllByIndex(
 *   "users", "statusIndex", IDBKeyRange.only("active")
 * );
 * 
 * // Perform bulk operations
 * const bulkRepo = adapter.getBulkRepository();
 * await bulkRepo.bulkOperation(
 *   "users", "readwrite", usersToUpdate,
 *   async (tx, chunk) => {
 *     // Implementation of bulk update
 *   }
 * );
 * 
 * @remarks
 * This adapter uses the Facade design pattern to provide a simplified interface
 * to a complex subsystem (IndexedDB). It also applies the Composition design pattern
 * by building functionality from specialized components rather than through inheritance.
 * 
 * The design prioritizes:
 * - Flexibility: Each component can be extended independently
 * - Testability: Dependencies can be mocked for testing
 * - Maintainability: Clear separation of concerns
 */
export class IndexedDBStorageAdapter<T extends UniquelyIdentifiable> {
  private database: IndexedDBDatabase;
  private repository: IndexedDBRepository<T>;
  private indexRepository: IndexedDBIndexRepository<T>;
  private bulkRepository: IndexedDBBulkRepository<T>;

  constructor(database: IndexedDBDatabase) {
    this.database = database;
    this.repository = new IndexedDBRepository<T>(database);
    this.indexRepository = new IndexedDBIndexRepository<T>(database);
    this.bulkRepository = new IndexedDBBulkRepository<T>(database);
  }

  /**
   * Gets the basic CRUD repository
   */
  public getRepository(): IStorageRepository<T> {
    return this.repository;
  }

  /**
   * Gets the index operations repository
   */
  public getIndexRepository(): IIndexOperations<T> {
    return this.indexRepository;
  }

  /**
   * Gets the bulk operations repository
   */
  public getBulkRepository(): IBulkOperations<T> {
    return this.bulkRepository;
  }

  /**
   * Gets database information
   */
  public async getDatabaseInfo(): Promise<{name: string; version: number; objectStores: string[]}> {
    return this.database.getInfo();
  }

  /**
   * Checks if IndexedDB is supported
   */
  public static isSupported(): boolean {
    return IndexedDBDatabase.isSupported();
  }
}