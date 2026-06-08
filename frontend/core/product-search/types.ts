// Product-search public contracts. Names now arrive as plain text from the CDN snapshot + delta
// cache, so the index is word/prefix based — no binary `.idx` structures.

// Minimal product view rendered by search results.
export interface IndexedProduct {
	productID: number;
	productName: string;
	brandID: number;
	brandName: string;
}

// A search result row plus its ranking score (higher = more relevant).
export interface ProductSearchHit {
	product: IndexedProduct;
	rank: number;
}
