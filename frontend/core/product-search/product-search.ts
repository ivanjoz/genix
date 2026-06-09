// ProductSearch: ranks products against a query over their plain-text names. Names (and brand
// names) come from the CDN snapshot + server delta via ProductEcommerceDataService, so matching is
// word/character-prefix based — no dictionary, syllables or binary encoding.
import { getProductEcommerceData, type ProductEcommerceDataService } from "./productos-delta-service";
import type { IndexedProduct, ProductSearchHit } from "./types";
import type { IProduct } from "$services/services/productos.svelte";

// Spanish connector words carry no search signal and are dropped from both names and queries.
const CONNECTOR_WORDS = new Set(["de", "del", "la", "el", "los", "las", "con", "para", "en", "al", "sin", "y"]);

// Scoring weights. A matched word scores per shared leading character, plus a bonus when the whole
// word matches; matches earlier in the name rank higher, and brand matches count slightly less.
const PREFIX_CHAR_POINTS = 2; // per shared leading character between query and product word
const EXACT_WORD_POINTS = 5; // added when a query word equals a product word
const FIRST_WORD_BONUS = 2; // any query word matching the name's first word
const SECOND_WORD_BONUS = 1; // any query word matching the name's second word
const BRAND_MATCH_PENALTY = 1; // brand relevance tracks name relevance, minus this
const MAX_RESULTS = 20;

// Normalize text for matching: lowercase, strip Spanish accents, keep only alphanumerics + spaces.
const ACCENT_MAP: Record<string, string> = {
	á: "a", à: "a", â: "a", ä: "a", ã: "a", å: "a",
	é: "e", è: "e", ê: "e", ë: "e",
	í: "i", ì: "i", î: "i", ï: "i",
	ó: "o", ò: "o", ô: "o", ö: "o", õ: "o",
	ú: "u", ü: "u",
	ñ: "n"
};

const normalize = (text: string): string => {
	let out = "";
	for (const rune of text.toLowerCase()) {
		const mapped = ACCENT_MAP[rune] ?? rune;
		if ((mapped >= "a" && mapped <= "z") || (mapped >= "0" && mapped <= "9")) {
			out += mapped;
		} else if (/\s/.test(mapped)) {
			out += " ";
		}
	}
	return out.trim().replace(/\s+/g, " ");
};

// Split into searchable words, dropping connectors and single characters (no signal, symmetric for
// queries and names).
const toWords = (text: string): string[] =>
	normalize(text)
		.split(" ")
		.filter((word) => word.length > 1 && !CONNECTOR_WORDS.has(word));

// Count shared leading characters between two normalized words.
const sharedPrefixLength = (a: string, b: string): number => {
	const max = Math.min(a.length, b.length);
	let length = 0;
	while (length < max && a[length] === b[length]) length++;
	return length;
};

interface ProductEntry {
	productID: number;
	productName: string; // raw name, for display
	words: string[]; // normalized words, for matching
	brandID: number;
}

interface ProductRankingDebugInfo {
	productID: number;
	productName: string;
	brandName: string;
	brandPoints: number;
	namePoints: number;
	firstWordPoints: number;
	secondWordPoints: number;
	exactWordMatchCount: number;
	bestPrefixLength: number;
	totalRankPoints: number;
}

interface ProductSearchDebugSnapshot {
	queryText: string;
	queryWords: string[];
	rankedProducts: ProductRankingDebugInfo[];
	generatedAtISO: string;
}

export class ProductSearch {
	private products: ProductEntry[] = [];
	private productByID = new Map<number, IProduct>(); // full record, so cards render without a by-id fetch
	private brandNameByID = new Map<number, string>(); // raw, for display
	private brandWordsByID = new Map<number, string[]>(); // normalized, for scoring
	private updatedInt32 = 0;
	private latestDebugSnapshot: ProductSearchDebugSnapshot | null = null;

	private data: ProductEcommerceDataService | null = null;
	readonly readyPromise: Promise<void>;
	isLoading = false;
	isReady = false;
	loadError: string | null = null;
	source = "file_plus_delta";

	constructor() {
		this.readyPromise = this.bootstrap();
	}

	private async bootstrap(): Promise<void> {
		this.isLoading = true;
		this.loadError = null;
		try {
			// Consume the single shared catalog instance instead of loading our own.
			this.data = await getProductEcommerceData();
			this.buildIndex();
			this.isReady = true;
			console.info("[ProductSearch] ready", { products: this.products.length, updated: this.updatedInt32 });
		} catch (error) {
			this.loadError = error instanceof Error ? error.message : "Unknown ProductSearch bootstrap error";
			console.error("[ProductSearch] bootstrap failed", error);
			throw error;
		} finally {
			this.isLoading = false;
		}
	}

	private buildIndex(): void {
		this.brandNameByID.clear();
		this.brandWordsByID.clear();
		for (const brand of this.data?.marcas ?? []) {
			this.brandNameByID.set(brand.ID, brand.Name);
			this.brandWordsByID.set(brand.ID, toWords(brand.Name));
		}
		this.products = [];
		this.productByID.clear();
		this.updatedInt32 = 0;
		for (const product of this.data?.productos ?? []) {
			if (!product || product.ID <= 0) continue;
			this.updatedInt32 = Math.max(this.updatedInt32, product.upd ?? 0);
			this.productByID.set(product.ID, product);
			this.products.push({
				productID: product.ID,
				productName: product.Name ?? "",
				words: toWords(product.Name ?? ""),
				brandID: product.BrandID ?? 0
			});
		}
	}

	get size(): number {
		return this.products.length;
	}

	get updated(): number {
		return this.updatedInt32;
	}

	getLastSearchDebugSnapshot(): ProductSearchDebugSnapshot | null {
		return this.latestDebugSnapshot;
	}

	// Full record for a hit so the search card can render name/price/image without a by-id fetch.
	getProduct(productID: number): IProduct | undefined {
		return this.productByID.get(productID);
	}

	search(queryText: string, options?: { enableFullDebugLog?: boolean }): ProductSearchHit[] {
		const startedAtMs = typeof performance !== "undefined" ? performance.now() : Date.now();
		const debug = options?.enableFullDebugLog ?? false;
		const queryWords = toWords(queryText);

		if (queryWords.length === 0) {
			this.latestDebugSnapshot = debug
				? { queryText, queryWords: [], rankedProducts: [], generatedAtISO: new Date().toISOString() }
				: null;
			return [];
		}

		const brandPointsByID = this.scoreBrands(queryWords);
		const debugRows: ProductRankingDebugInfo[] = [];
		const scored: Array<{ entry: ProductEntry; rank: number; exactCount: number; bestPrefix: number }> = [];

		for (const entry of this.products) {
			let namePoints = 0;
			let exactCount = 0;
			let bestPrefix = 0;
			for (const queryWord of queryWords) {
				const { prefix, exact } = this.bestWordMatch(queryWord, entry.words);
				if (prefix === 0) continue;
				namePoints += prefix * PREFIX_CHAR_POINTS + (exact ? EXACT_WORD_POINTS : 0);
				if (exact) exactCount++;
				if (prefix > bestPrefix) bestPrefix = prefix;
			}

			const firstWordPoints =
				entry.words.length > 0 && this.anyQueryMatchesWord(queryWords, entry.words[0]) ? FIRST_WORD_BONUS : 0;
			const secondWordPoints =
				entry.words.length > 1 && this.anyQueryMatchesWord(queryWords, entry.words[1]) ? SECOND_WORD_BONUS : 0;
			const brandPoints = brandPointsByID.get(entry.brandID) ?? 0;
			const rank = namePoints + brandPoints + firstWordPoints + secondWordPoints;
			if (rank <= 0) continue;

			scored.push({ entry, rank, exactCount, bestPrefix });
			if (debug) {
				debugRows.push({
					productID: entry.productID,
					productName: entry.productName,
					brandName: this.brandNameByID.get(entry.brandID) ?? "",
					brandPoints,
					namePoints,
					firstWordPoints,
					secondWordPoints,
					exactWordMatchCount: exactCount,
					bestPrefixLength: bestPrefix,
					totalRankPoints: rank
				});
			}
		}

		// Rank by score, then exact-word matches, then longest prefix, then a stable id order.
		scored.sort((left, right) => {
			if (right.rank !== left.rank) return right.rank - left.rank;
			if (right.exactCount !== left.exactCount) return right.exactCount - left.exactCount;
			if (right.bestPrefix !== left.bestPrefix) return right.bestPrefix - left.bestPrefix;
			return left.entry.productID - right.entry.productID;
		});

		const hits: ProductSearchHit[] = scored.slice(0, MAX_RESULTS).map(({ entry, rank }) => ({
			product: this.toIndexedProduct(entry),
			rank
		}));

		this.latestDebugSnapshot = debug
			? {
					queryText,
					queryWords,
					rankedProducts: debugRows.sort((a, b) => b.totalRankPoints - a.totalRankPoints),
					generatedAtISO: new Date().toISOString()
			  }
			: null;

		const elapsedMs = (typeof performance !== "undefined" ? performance.now() : Date.now()) - startedAtMs;
		console.log(`Search: ${queryText}. Found: ${scored.length} matches in ${Math.max(0, Math.round(elapsedMs))}ms`);
		return hits;
	}

	// Best (longest prefix, exact?) of a query word against a product's words.
	private bestWordMatch(queryWord: string, words: readonly string[]): { prefix: number; exact: boolean } {
		let prefix = 0;
		let exact = false;
		for (const word of words) {
			const shared = sharedPrefixLength(queryWord, word);
			if (shared > prefix) prefix = shared;
			if (shared === queryWord.length && word.length === queryWord.length) exact = true;
		}
		return { prefix, exact };
	}

	// Positional bonus helper: does any query word share a leading prefix with this name word?
	private anyQueryMatchesWord(queryWords: readonly string[], word: string): boolean {
		return queryWords.some((queryWord) => sharedPrefixLength(queryWord, word) > 0);
	}

	// Score every brand once; products reuse points via their brandID. Brand relevance mirrors name
	// relevance minus a small penalty so a name hit always outranks the equivalent brand hit.
	private scoreBrands(queryWords: readonly string[]): Map<number, number> {
		const pointsByID = new Map<number, number>();
		for (const [brandID, brandWords] of this.brandWordsByID) {
			let total = 0;
			for (const queryWord of queryWords) {
				const { prefix, exact } = this.bestWordMatch(queryWord, brandWords);
				if (prefix === 0) continue;
				const points = prefix * PREFIX_CHAR_POINTS + (exact ? EXACT_WORD_POINTS : 0);
				total += Math.max(1, points - BRAND_MATCH_PENALTY);
			}
			if (total > 0) pointsByID.set(brandID, total);
		}
		return pointsByID;
	}

	private toIndexedProduct(entry: ProductEntry): IndexedProduct {
		return {
			productID: entry.productID,
			productName: entry.productName,
			brandID: entry.brandID,
			brandName: this.brandNameByID.get(entry.brandID) ?? ""
		};
	}
}
