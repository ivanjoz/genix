// One value per key in a single-store IndexedDB database, with no library: for callers that keep a
// whole document (read and written whole) rather than records.

const VALUE_STORE_NAME = 'values';

const openValueDatabase = (databaseName: string): Promise<IDBDatabase> =>
	new Promise((resolve, reject) => {
		const openRequest = indexedDB.open(databaseName, 1);
		openRequest.onupgradeneeded = () => openRequest.result.createObjectStore(VALUE_STORE_NAME);
		openRequest.onsuccess = () => resolve(openRequest.result);
		openRequest.onerror = () => reject(openRequest.error);
	});

const runValueRequest = async <T>(
	databaseName: string, mode: IDBTransactionMode, makeRequest: (store: IDBObjectStore) => IDBRequest,
): Promise<T> => {
	const database = await openValueDatabase(databaseName);
	try {
		return await new Promise<T>((resolve, reject) => {
			const request = makeRequest(database.transaction(VALUE_STORE_NAME, mode).objectStore(VALUE_STORE_NAME));
			request.onsuccess = () => resolve(request.result as T);
			request.onerror = () => reject(request.error);
		});
	} finally {
		database.close();
	}
};

/** The value saved under key, or undefined when there is none. */
export const readIndexedDBValue = <T>(databaseName: string, key: string): Promise<T | undefined> =>
	runValueRequest<T | undefined>(databaseName, 'readonly', (store) => store.get(key));

/** Saves value under key, replacing what was there. */
export const writeIndexedDBValue = (databaseName: string, key: string, value: unknown): Promise<void> =>
	runValueRequest<void>(databaseName, 'readwrite', (store) => store.put(value, key));
