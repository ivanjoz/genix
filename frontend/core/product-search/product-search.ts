// ProductSearch runtime: bootstraps from idx/deltas, maintains in-memory product index, and ranks queries.
import { decodeBinary } from "./decoder";
import {
	encodeQueryWordToDictionarySyllables,
	normalizeAndFilterQueryTokens,
	normalizeSearchToken,
	splitNormalizedWords
} from "./encoder";
import { Env } from "$core/env";
import type { IndexedProduct, ProductSearchHit } from "./types";
import { ProductosDeltaService } from "./productos-delta-service";
import type { IListaRegistro } from "$services/negocio/listas-compartidas.svelte";
import type { IProducto } from "$services/services/productos.svelte";

export interface ProductQueryWordDebugInfo {
	queryWord: string;
	encodedSyllableIDs: number[];
}

export interface ProductSearchOptions {
	enableFullDebugLog?: boolean;
}

export interface ProductBrandMatchDebugInfo {
	queryWord: string;
	brandWord: string;
	pointsAwarded: number;
}

export interface ProductQueryWordMatchDebugInfo {
	queryWord: string;
	matchedBy: "syllable_prefix" | "character_prefix" | "none";
	matchedProductWord: string;
	matchedSyllablePrefixLength: number;
	matchedCharacterPrefixLength: number;
	exactWordMatch: boolean;
	pointsAwarded: number;
}

export interface ProductRankingDebugInfo {
	productID: number;
	productNameLossy: string;
	brandID: number;
	brandName: string;
	brandPoints: number;
	productNamePoints: number;
	firstWordPoints: number;
	secondWordPoints: number;
	exactWordMatchCount: number;
	bestCharacterPrefixLength: number;
	totalRankPoints: number;
	brandWordMatches: ProductBrandMatchDebugInfo[];
	queryWordMatches: ProductQueryWordMatchDebugInfo[];
}

export interface ProductSearchDebugSnapshot {
	queryText: string;
	normalizedQueryTokens: string[];
	encodedQueryWords: ProductQueryWordDebugInfo[];
	rankedProducts: ProductRankingDebugInfo[];
	generatedAtISO: string;
}

interface ProductRankingTieBreakInfo {
	exactWordMatchCount: number;
	bestSyllablePrefixLength: number;
}

interface FastQueryWordMatchResult {
	pointsAwarded: number;
	exactWordMatch: boolean;
	bestSyllablePrefixLength: number;
}

interface ProductSearchBootstrapSource {
	source: "idx_plus_delta" | "delta_only";
}

const SPANISH_VOWELS = ["a", "e", "i", "o", "u"];
const SPANISH_CONSONANTS = ["b", "c", "d", "f", "g", "h", "j", "k", "l", "m", "n", "p", "q", "r", "s", "t", "v", "w", "x", "y", "z"];

// Keep canonical aliases aligned with backend defaultFixedAliases() so query and record encoding stay compatible.
const BASE_FIXED_ALIAS_GROUPS: readonly (readonly string[])[] = [
	["1"],
	["2"],
	["y", "3"],
	["4"],
	["5"],
	["z", "6"],
	["x", "7"],
	["w", "8"],
	["9"],
	["g", "gr", "grs"],
	["ml", "mm", "mg", "mgs", "m3", "m2"],
	["k", "kg", "kgs", "kilo", "kilos", "kilogramos"],
	["ud", "un", "uns", "unidad", "unidades"],
	["c", "cm", "cm2", "cm3", "cl"],
	["l", "lt", "lts", "litro", "litos"],
	["r", "rr"],
	["q", "qu"],
	["p", "pack", "paq", "paquete"]
];

const buildFixedAliasGroups = (): readonly (readonly string[])[] => {
	// Start from backend-aligned canonical/alias groups and then complete missing single-letter tokens.
	const fixedAliasGroups: string[][] = BASE_FIXED_ALIAS_GROUPS.map((group) => [...group]);
	// Add single-letter vowel/consonant canonicals only when not already present in base groups.
	const canonicalTokenSet = new Set<string>();
	for (const aliasGroup of fixedAliasGroups) {
		const normalizedCanonicalToken = normalizeSearchToken(aliasGroup[0]).replace(/\s+/g, "");
		if (normalizedCanonicalToken) {
			canonicalTokenSet.add(normalizedCanonicalToken);
		}
	}
	for (const singleLetterToken of [...SPANISH_VOWELS, ...SPANISH_CONSONANTS]) {
		if (canonicalTokenSet.has(singleLetterToken)) {
			continue;
		}
		fixedAliasGroups.push([singleLetterToken]);
		canonicalTokenSet.add(singleLetterToken);
	}
	return fixedAliasGroups;
};

const FIXED_ALIAS_GROUPS = buildFixedAliasGroups();

export class ProductSearch implements ProductSearchBootstrapSource {
	// Keep scoring constants explicit so ranking behavior is predictable and easy to tune.
	private static readonly SYLLABLE_MATCH_POINTS_TWO_LETTER = 4;
	private static readonly SYLLABLE_MATCH_POINTS_ONE_LETTER = 2;
	private static readonly COMPLETE_WORD_MATCH_POINTS = 5;
	private static readonly BRAND_WORD_MATCH_PENALTY_POINTS = 1;
	private static readonly FIRST_WORD_POSITIONAL_POINTS = 2;
	private static readonly SECOND_WORD_POSITIONAL_POINTS = 1;
	private static readonly MAX_DICTIONARY_TOKENS = 255;
	private static readonly INDEX_COMPANY_ID = 1;

	private readonly productNameBytesByProductID = new Map<number, Uint8Array>();
	private readonly brandDictionaryIndexByProductID = new Map<number, number>();
	private readonly brandIDMapByProductID = new Map<number, number>();
	private readonly categoryIDsMapByProductID = new Map<number, number[]>();
	private productIDsSorted: number[] = [];
	private dictionaryTokens: string[] = [];
	private readonly dictionaryTokenIDByNormalizedToken = new Map<string, number>();
	private readonly aliasDictionaryIDByNormalizedToken = new Map<string, number>();
	private readonly brandNameByID = new Map<number, string>();
	private readonly categoryNameByID = new Map<number, string>();
	private readonly normalizedBrandWordsByBrandID = new Map<number, string[]>();
	private updatedInt32 = 0;
	private readonly productsDeltaService = new ProductosDeltaService();
	private latestSearchDebugSnapshot: ProductSearchDebugSnapshot | null = null;
	readonly readyPromise: Promise<void>;
	isLoading = false;
	isReady = false;
	loadError: string | null = null;
	source: ProductSearchBootstrapSource["source"] = "delta_only";

	constructor() {
		// Keep constructor synchronous and expose readiness through readyPromise.
		this.readyPromise = this.bootstrap();
	}

	private async bootstrap(): Promise<void> {
	// Bootstrap order is fixed: try idx, fetch deltas, then fallback rebuild if needed.
		this.isLoading = true;
		this.loadError = null;

		let didLoadFromBinary = false;
		try {
			didLoadFromBinary = await this.tryLoadBinaryIndex();
			await this.loadDeltas();
			if (!didLoadFromBinary) {
				this.rebuildFromDeltaOnly();
				this.source = "delta_only";
				console.info("[ProductSearch] initialized from delta-only source", {
					products: this.productIDsSorted.length,
					dictionaryTokens: this.dictionaryTokens.length
				});
			} else {
				this.source = "idx_plus_delta";
				console.info("[ProductSearch] initialized from idx + delta source", {
					products: this.productIDsSorted.length,
					dictionaryTokens: this.dictionaryTokens.length
				});
			}
			this.isReady = true;
		} catch (bootstrapError) {
			this.loadError =
				bootstrapError instanceof Error
					? bootstrapError.message
					: "Unknown ProductSearch bootstrap error";
			console.error("[ProductSearch] bootstrap failed", bootstrapError);
			throw bootstrapError;
		} finally {
			this.isLoading = false;
		}
	}

	private async tryLoadBinaryIndex(): Promise<boolean> {
		// Keep source url centralized so all callers/diagnostics use the same route.
		const productsIndexUrl = Env.makeCDNRoute(
			"live",
			`c${ProductSearch.INDEX_COMPANY_ID}_products.idx`
		);
		console.log("[ProductSearch] trying to load idx", { productsIndexUrl });
		try {
			// A non-200 response is treated as "no idx available" rather than hard-failing startup.
			const indexResponse = await fetch(productsIndexUrl+"?use-cache=60");
			if (!indexResponse.ok) {
				console.warn("[ProductSearch] idx response not ok", {
					status: indexResponse.status,
					statusText: indexResponse.statusText
				});
				return false;
			}
			const indexBinaryBuffer = await indexResponse.arrayBuffer();
			const decodedPayload = decodeBinary(indexBinaryBuffer);
			// Reset first to avoid mixed state if bootstrap runs more than once.
			this.resetIndexState();
			this.loadFromDecodedPayload(decodedPayload);
			console.log("[ProductSearch] idx loaded", {
				records: decodedPayload.records.length,
				dictionaryTokens: decodedPayload.dictionaryTokens.length
			});
			return true;
		} catch (idxLoadError) {
			console.warn("[ProductSearch] idx load failed, using delta-only fallback", idxLoadError);
			return false;
		}
	}

	private async loadDeltas(): Promise<void> {
		// Delta service is authoritative for post-build changes and taxonomy updates.
		await this.productsDeltaService.fetchOnline();
		const deltaProducts = this.productsDeltaService.productos ?? [];
		const deltaBrands = this.productsDeltaService.marcas ?? [];
		const deltaCategories = this.productsDeltaService.categorias ?? [];
		console.log("[ProductSearch] delta fetched", {
			productos: deltaProducts.length,
			marcas: deltaBrands.length,
			categorias: deltaCategories.length
		});
		this.applyTaxonomyDeltas(deltaBrands, deltaCategories);
		if (this.productIDsSorted.length > 0) {
			// Only merge directly when base index already exists.
			this.applyProductDeltas(deltaProducts);
		}
	}

	private loadFromDecodedPayload(decodedPayload: ReturnType<typeof decodeBinary>): void {
		// Persist build watermark so callers can inspect freshness after startup.
		// Keep build/update watermark from the text header for delta sync workflows.
		this.updatedInt32 = decodedPayload.stats.updated;
		this.dictionaryTokens = [...decodedPayload.dictionaryTokens];
		this.rebuildDictionaryIDLookup();
		for (const [aliasToken, dictionaryID] of decodedPayload.dictionaryAliases.entries()) {
			const normalizedAliasToken = normalizeSearchToken(aliasToken).replace(/\s+/g, "");
			if (!normalizedAliasToken || dictionaryID <= 0) {
				continue;
			}
			this.aliasDictionaryIDByNormalizedToken.set(normalizedAliasToken, dictionaryID);
		}
		// Build compact product-name byte rows using 0 as an explicit word separator.
		for (const decodedRecord of decodedPayload.records) {
			const separatedWordBytes = this.buildZeroSeparatedWordBytes(
				decodedRecord.productBytes,
				decodedRecord.shape
			);
			this.upsertEncodedProduct({
				productID: decodedRecord.productID,
				productBytes: separatedWordBytes,
				brandDictionaryIndex: decodedRecord.brandDictionaryIndex,
				brandID: decodedRecord.brandID,
				categoryIDs: decodedRecord.categoryIDs
			});
		}

		// Keep tiny taxonomy dictionaries for on-demand name resolution during get(productID).
		if (decodedPayload.taxonomy) {
			for (let brandIndex = 0; brandIndex < decodedPayload.taxonomy.brandIDs.length; brandIndex++) {
				const brandName = decodedPayload.taxonomy.brandNames[brandIndex];
				const brandID = decodedPayload.taxonomy.brandIDs[brandIndex];
				this.brandNameByID.set(brandID, brandName);
				this.normalizedBrandWordsByBrandID.set(brandID, splitNormalizedWords(brandName));
			}
			for (
				let categoryIndex = 0;
				categoryIndex < decodedPayload.taxonomy.categoryIDs.length;
				categoryIndex++
			) {
				this.categoryNameByID.set(
					decodedPayload.taxonomy.categoryIDs[categoryIndex],
					decodedPayload.taxonomy.categoryNames[categoryIndex]
				);
			}
		}
	}

	private rebuildFromDeltaOnly(): void {
		// Delta-only mode must synthesize dictionary + records in-memory without idx bytes.
		this.resetIndexState();
		this.seedDictionaryFromFixedSpanishSyllables();
		this.applyTaxonomyDeltas(
			this.productsDeltaService.marcas ?? [],
			this.productsDeltaService.categorias ?? []
		);
		this.applyProductDeltas(this.productsDeltaService.productos ?? []);
	}

	private seedDictionaryFromFixedSpanishSyllables(): void {
		// Fixed aliases ensure consistent compact encoding even for very small catalogs.
		for (const aliasGroup of FIXED_ALIAS_GROUPS) {
			const normalizedCanonicalToken = normalizeSearchToken(aliasGroup[0]).replace(/\s+/g, "");
			if (!normalizedCanonicalToken) {
				continue;
			}
			const canonicalDictionaryID = this.ensureDictionaryTokenID(normalizedCanonicalToken);
			for (const aliasToken of aliasGroup) {
				const normalizedAliasToken = normalizeSearchToken(aliasToken).replace(/\s+/g, "");
				if (!normalizedAliasToken || canonicalDictionaryID <= 0) {
					continue;
				}
				this.aliasDictionaryIDByNormalizedToken.set(normalizedAliasToken, canonicalDictionaryID);
			}
		}
		for (const consonant of SPANISH_CONSONANTS) {
			for (const vowel of SPANISH_VOWELS) {
				// Add both CV and VC to improve prefix coverage when idx is unavailable.
				this.ensureDictionaryTokenID(consonant + vowel);
				this.ensureDictionaryTokenID(vowel + consonant);
			}
		}
		// Keep sentinel tokens available for products without explicit brand.
		this.ensureDictionaryTokenID("sin");
		this.ensureDictionaryTokenID("marca");
	}

	private applyTaxonomyDeltas(
		brandRows: readonly IListaRegistro[],
		categoryRows: readonly IListaRegistro[]
	): void {
		// Keep normalized brand words precomputed so brand scoring stays O(words) per brand.
		for (const brandRow of brandRows) {
			const normalizedBrandName = normalizeSearchToken(brandRow.Nombre);
			this.brandNameByID.set(brandRow.ID, normalizedBrandName);
			this.normalizedBrandWordsByBrandID.set(brandRow.ID, splitNormalizedWords(normalizedBrandName));
		}
		for (const categoryRow of categoryRows) {
			// Category names are only used for hydrated product views.
			this.categoryNameByID.set(categoryRow.ID, normalizeSearchToken(categoryRow.Nombre));
		}
	}

	private applyProductDeltas(deltaProducts: readonly IProducto[]): void {
		// Delta payload can contain upserts and logical deletes (ss=0).
		for (const deltaProduct of deltaProducts) {
			if (!deltaProduct || deltaProduct.ID <= 0) {
				continue;
			}
			const productUpdated = Number(deltaProduct.upd || 0);
			if (productUpdated > this.updatedInt32) {
				// Keep latest observed update watermark across all merged deltas.
				this.updatedInt32 = productUpdated;
			}
			if (deltaProduct.ss === 0) {
				// Remove deleted rows from all in-memory indexes/maps.
				this.removeProduct(deltaProduct.ID);
				continue;
			}
			const encodedNameBytes = this.encodeProductNameToByteRow(deltaProduct.Nombre || "");
			if (encodedNameBytes.length === 0) {
				continue;
			}
			this.upsertEncodedProduct({
				productID: deltaProduct.ID,
				productBytes: encodedNameBytes,
				brandDictionaryIndex: 0,
				brandID: deltaProduct.MarcaID ?? 0,
				categoryIDs: deltaProduct.CategoriasIDs ?? []
			});
		}
	}

	private encodeProductNameToByteRow(rawProductName: string): Uint8Array {
		// Reuse query normalization so fallback encoding follows the same token rules as search.
		const normalizedTokens = normalizeAndFilterQueryTokens(rawProductName);
		if (normalizedTokens.length === 0) {
			return new Uint8Array(0);
		}
		const encodedWordSyllables: number[][] = [];
		for (const normalizedToken of normalizedTokens) {
			// Encode each normalized word into dictionary syllable ids.
			const encodedSyllables = this.encodeWordTokenForProduct(normalizedToken);
			if (encodedSyllables.length === 0) {
				continue;
			}
			encodedWordSyllables.push(encodedSyllables);
		}
		if (encodedWordSyllables.length === 0) {
			return new Uint8Array(0);
		}
		const totalSyllables = encodedWordSyllables.reduce(
			(accumulated, currentWord) => accumulated + currentWord.length,
			0
		);
		const separatorCount = Math.max(0, encodedWordSyllables.length - 1);
		const byteRow = new Uint8Array(totalSyllables + separatorCount);
		let outputOffset = 0;
		for (let wordIndex = 0; wordIndex < encodedWordSyllables.length; wordIndex++) {
			const currentWordSyllables = encodedWordSyllables[wordIndex];
			for (const syllableID of currentWordSyllables) {
				byteRow[outputOffset++] = syllableID;
			}
			if (wordIndex < encodedWordSyllables.length - 1) {
				// Zero byte acts as explicit word boundary in the compact row encoding.
				byteRow[outputOffset++] = 0;
			}
		}
		return byteRow;
	}

	private encodeWordTokenForProduct(normalizedToken: string): number[] {
		// Prefer canonical alias resolution first to keep encoded rows compact and deterministic.
		const encodedSyllables: number[] = [];
		const aliasDictionaryID = this.aliasDictionaryIDByNormalizedToken.get(normalizedToken);
		if (aliasDictionaryID) {
			return [aliasDictionaryID];
		}
		for (const syllableToken of this.splitTokenIntoSyllables(normalizedToken)) {
			const normalizedSyllableToken = normalizeSearchToken(syllableToken).replace(/\s+/g, "");
			if (!normalizedSyllableToken) {
				continue;
			}
			let dictionaryID = this.aliasDictionaryIDByNormalizedToken.get(normalizedSyllableToken);
			if (!dictionaryID) {
				dictionaryID =
					this.dictionaryTokenIDByNormalizedToken.get(normalizedSyllableToken) ??
					this.ensureDictionaryTokenID(normalizedSyllableToken);
			}
			if (dictionaryID) {
				encodedSyllables.push(dictionaryID);
				continue;
			}
			for (const currentRune of normalizedSyllableToken) {
				// Last-resort path: fallback to single-rune dictionary tokens when syllable is unknown.
				const runeDictionaryID = this.ensureDictionaryTokenID(currentRune);
				if (runeDictionaryID) {
					encodedSyllables.push(runeDictionaryID);
				}
			}
		}
		return encodedSyllables;
	}

	private splitTokenIntoSyllables(token: string): string[] {
		// Use the same 2-char chunking strategy as backend syllable splitter.
		if (!token) {
			return [];
		}
		if (token.length <= 2) {
			return [token];
		}
		const syllableTokens: string[] = [];
		for (let offset = 0; offset < token.length; offset += 2) {
			syllableTokens.push(token.slice(offset, Math.min(offset + 2, token.length)));
		}
		return syllableTokens;
	}

	private ensureDictionaryTokenID(rawToken: string): number {
		// Dictionary ids are 1-based and capped to 255 to preserve uint8 encoding.
		const normalizedToken = normalizeSearchToken(rawToken).replace(/\s+/g, "");
		if (!normalizedToken) {
			return 0;
		}
		const existingDictionaryID = this.dictionaryTokenIDByNormalizedToken.get(normalizedToken);
		if (existingDictionaryID) {
			return existingDictionaryID;
		}
		if (this.dictionaryTokens.length >= ProductSearch.MAX_DICTIONARY_TOKENS) {
			// Refuse to overflow dictionary id space in fallback mode.
			return 0;
		}
		this.dictionaryTokens.push(normalizedToken);
		const createdDictionaryID = this.dictionaryTokens.length;
		this.dictionaryTokenIDByNormalizedToken.set(normalizedToken, createdDictionaryID);
		return createdDictionaryID;
	}

	private rebuildDictionaryIDLookup(): void {
		// Rebuild normalized-token -> dictionary-id map after bulk dictionary replacement.
		this.dictionaryTokenIDByNormalizedToken.clear();
		for (let tokenIndex = 0; tokenIndex < this.dictionaryTokens.length; tokenIndex++) {
			const normalizedDictionaryToken = normalizeSearchToken(this.dictionaryTokens[tokenIndex]).replace(
				/\s+/g,
				""
			);
			if (!normalizedDictionaryToken) {
				continue;
			}
			this.dictionaryTokenIDByNormalizedToken.set(normalizedDictionaryToken, tokenIndex + 1);
		}
	}

	private upsertEncodedProduct(payload: {
		productID: number;
		productBytes: Uint8Array;
		brandDictionaryIndex: number;
		brandID: number;
		categoryIDs: number[];
	}): void {
		// Upsert keeps product order stable and only appends IDs when first seen.
		const wasExisting = this.productNameBytesByProductID.has(payload.productID);
		this.productNameBytesByProductID.set(payload.productID, payload.productBytes);
		this.brandDictionaryIndexByProductID.set(payload.productID, payload.brandDictionaryIndex);
		this.brandIDMapByProductID.set(payload.productID, payload.brandID);
		this.categoryIDsMapByProductID.set(payload.productID, [...payload.categoryIDs]);
		if (!wasExisting) {
			this.productIDsSorted.push(payload.productID);
		}
	}

	private removeProduct(productID: number): void {
		// Delete product from all per-product maps and ordered id list.
		if (!this.productNameBytesByProductID.has(productID)) {
			return;
		}
		this.productNameBytesByProductID.delete(productID);
		this.brandDictionaryIndexByProductID.delete(productID);
		this.brandIDMapByProductID.delete(productID);
		this.categoryIDsMapByProductID.delete(productID);
		this.productIDsSorted = this.productIDsSorted.filter((currentID) => currentID !== productID);
	}

	private resetIndexState(): void {
		// Hard reset all mutable runtime collections before rebuilding/bootstrap.
		this.productNameBytesByProductID.clear();
		this.brandDictionaryIndexByProductID.clear();
		this.brandIDMapByProductID.clear();
		this.categoryIDsMapByProductID.clear();
		this.productIDsSorted = [];
		this.dictionaryTokens = [];
		this.dictionaryTokenIDByNormalizedToken.clear();
		this.aliasDictionaryIDByNormalizedToken.clear();
		this.brandNameByID.clear();
		this.categoryNameByID.clear();
		this.normalizedBrandWordsByBrandID.clear();
		this.updatedInt32 = 0;
	}

	get size(): number {
		return this.productIDsSorted.length;
	}

	get updated(): number {
		return this.updatedInt32;
	}

	get buildSunixTime(): number {
		return this.updatedInt32;
	}

	get(productID: number): IndexedProduct | undefined {
		// Hydrate an indexed product view on demand; avoids storing duplicated derived objects.
		const productBytesWithWordSeparator = this.productNameBytesByProductID.get(productID);
		if (!productBytesWithWordSeparator) {
			return undefined;
		}

		const brandID = this.brandIDMapByProductID.get(productID) ?? 0;
		const categoryIDs = this.categoryIDsMapByProductID.get(productID) ?? [];
		return {
			productID,
			productBytes: productBytesWithWordSeparator,
			productNameLossy: this.decodeLossyName(productBytesWithWordSeparator),
			brandDictionaryIndex: this.brandDictionaryIndexByProductID.get(productID) ?? 0,
			brandID,
			brandName: this.brandNameByID.get(brandID) ?? "",
			categoryDictionaryIndexes: [],
			categoryIDs: [...categoryIDs],
			categoryNames: categoryIDs.map(
				(categoryID) => this.categoryNameByID.get(categoryID) ?? ""
			)
		};
	}

	has(productID: number): boolean {
		return this.productNameBytesByProductID.has(productID);
	}

	getBrandIDByProductID(productID: number): number | undefined {
		return this.brandIDMapByProductID.get(productID);
	}

	getCategoryIDsByProductID(productID: number): readonly number[] {
		return this.categoryIDsMapByProductID.get(productID) ?? [];
	}

	get brandByProductID(): ReadonlyMap<number, number> {
		return this.brandIDMapByProductID;
	}

	get categoryByProductID(): ReadonlyMap<number, readonly number[]> {
		return this.categoryIDsMapByProductID;
	}

	get productIDs(): readonly number[] {
		return this.productIDsSorted;
	}

	getByBrandID(brandID: number): IndexedProduct[] {
		// Filter on demand to avoid storing duplicated reverse indexes in memory.
		const filteredProducts: IndexedProduct[] = [];
		for (const productID of this.productIDsSorted) {
			if ((this.brandIDMapByProductID.get(productID) ?? 0) !== brandID) {
				continue;
			}
			const decodedProduct = this.get(productID);
			if (decodedProduct) {
				filteredProducts.push(decodedProduct);
			}
		}
		return filteredProducts;
	}

	getByCategoryID(categoryID: number): IndexedProduct[] {
		// Filter on demand to avoid storing duplicated reverse indexes in memory.
		const filteredProducts: IndexedProduct[] = [];
		for (const productID of this.productIDsSorted) {
			const productCategoryIDs = this.categoryIDsMapByProductID.get(productID) ?? [];
			if (!productCategoryIDs.includes(categoryID)) {
				continue;
			}
			const decodedProduct = this.get(productID);
			if (decodedProduct) {
				filteredProducts.push(decodedProduct);
			}
		}
		return filteredProducts;
	}

	search(queryText: string, options?: ProductSearchOptions): ProductSearchHit[] {
		// Measure full search execution time for mandatory query telemetry.
		const searchStartedAtMs = typeof performance !== "undefined" ? performance.now() : Date.now();
		const fullDebugEnabled =
			options?.enableFullDebugLog ?? Env.PRODUCT_SEARCH_FULL_DEBUG_LOG_ENABLED;
		const encodedQueryTokens = normalizeAndFilterQueryTokens(queryText);
		if (encodedQueryTokens.length === 0) {
			// Empty/connector-only queries short-circuit and optionally publish empty debug snapshots.
			this.latestSearchDebugSnapshot = fullDebugEnabled
				? {
						queryText,
						normalizedQueryTokens: [],
						encodedQueryWords: [],
						rankedProducts: [],
						generatedAtISO: new Date().toISOString()
				  }
					: null;
			const emptyQueryElapsedMs =
				(typeof performance !== "undefined" ? performance.now() : Date.now()) - searchStartedAtMs;
			console.log(
				`Search: ${queryText}. Found: 0 matches in ${Math.max(0, Math.round(emptyQueryElapsedMs))}ms`
			);
			return [];
		}

		// Pre-encode query words to dictionary token IDs so product scanning remains byte-based.
		const encodedQueryWords = encodedQueryTokens
			.map((queryToken) => ({
				word: queryToken,
				syllables: encodeQueryWordToDictionarySyllables(
					queryToken,
					this.dictionaryTokenIDByNormalizedToken,
					this.aliasDictionaryIDByNormalizedToken
				)
			}));
		const debugEncodedQueryWords: ProductQueryWordDebugInfo[] = fullDebugEnabled
			? encodedQueryWords.map((encodedQueryWord) => ({
					queryWord: encodedQueryWord.word,
					encodedSyllableIDs: Array.from(encodedQueryWord.syllables)
			  }))
			: [];

		// Score query-brand affinity once, then apply bonus only to products that belong to matched brands.
		const brandScoring = this.computeBrandScoringByBrandID(encodedQueryWords, fullDebugEnabled);
		const brandPointsByBrandID = brandScoring.brandPointsByBrandID;
		const brandMatchesByBrandID = brandScoring.brandMatchesByBrandID;
		const rankingDebugInfoByProductID = new Map<number, ProductRankingDebugInfo>();
		const rankingTieBreakByProductID = new Map<number, ProductRankingTieBreakInfo>();

		const rankingByProductID = new Map<number, number>();
		for (const productID of this.productIDsSorted) {
			const productNameBytes = this.productNameBytesByProductID.get(productID);
			if (!productNameBytes || productNameBytes.length === 0) {
				continue;
			}

			// Split product byte row by 0 separators to evaluate per-word prefix matches.
			const productWordSyllables = this.splitProductWordsBySeparator(productNameBytes);
			const productBrandID = this.brandIDMapByProductID.get(productID) ?? 0;
			const productBrandName = this.brandNameByID.get(productBrandID) ?? "";
			const brandPoints = brandPointsByBrandID.get(productBrandID) ?? 0;
			let productNamePoints = 0;
			let exactWordMatchCount = 0;
			let bestSyllablePrefixLength = 0;
			const queryWordMatches: ProductQueryWordMatchDebugInfo[] = [];
			if (fullDebugEnabled) {
				for (const encodedQueryWord of encodedQueryWords) {
					const queryWordMatchDebugInfo = this.evaluateQueryWordAgainstProductWords(
						encodedQueryWord.word,
						encodedQueryWord.syllables,
						productWordSyllables
					);
					productNamePoints += queryWordMatchDebugInfo.pointsAwarded;
					if (queryWordMatchDebugInfo.exactWordMatch) {
						exactWordMatchCount++;
					}
					if (
						queryWordMatchDebugInfo.matchedSyllablePrefixLength > bestSyllablePrefixLength
					) {
						bestSyllablePrefixLength = queryWordMatchDebugInfo.matchedSyllablePrefixLength;
					}
					queryWordMatches.push(queryWordMatchDebugInfo);
				}
			} else {
				for (const encodedQueryWord of encodedQueryWords) {
					const fastQueryWordMatch = this.evaluateQueryWordAgainstProductWordsFast(
						encodedQueryWord.syllables,
						productWordSyllables
					);
					productNamePoints += fastQueryWordMatch.pointsAwarded;
					if (fastQueryWordMatch.exactWordMatch) {
						exactWordMatchCount++;
					}
					if (fastQueryWordMatch.bestSyllablePrefixLength > bestSyllablePrefixLength) {
						bestSyllablePrefixLength = fastQueryWordMatch.bestSyllablePrefixLength;
					}
				}
			}
			let firstWordPoints = 0;
			if (
				productWordSyllables.length > 0 &&
				this.hasAnyPrefixMatchInWordBySyllables(encodedQueryWords, productWordSyllables[0])
			) {
				firstWordPoints = ProductSearch.FIRST_WORD_POSITIONAL_POINTS;
			}
			let secondWordPoints = 0;
			if (
				productWordSyllables.length > 1 &&
				this.hasAnyPrefixMatchInWordBySyllables(encodedQueryWords, productWordSyllables[1])
			) {
				secondWordPoints = ProductSearch.SECOND_WORD_POSITIONAL_POINTS;
			}
			const totalRankPoints = brandPoints + productNamePoints + firstWordPoints + secondWordPoints;

			if (totalRankPoints > 0) {
				// Only keep positively scored products in ranking candidates.
				rankingByProductID.set(productID, totalRankPoints);
				rankingTieBreakByProductID.set(productID, {
					exactWordMatchCount,
					bestSyllablePrefixLength
				});
				if (fullDebugEnabled) {
					rankingDebugInfoByProductID.set(productID, {
						productID,
						productNameLossy: this.decodeLossyName(productNameBytes),
						brandID: productBrandID,
						brandName: productBrandName,
						brandPoints,
						productNamePoints,
						firstWordPoints,
						secondWordPoints,
						exactWordMatchCount,
						bestCharacterPrefixLength: bestSyllablePrefixLength,
						totalRankPoints,
						brandWordMatches: [...(brandMatchesByBrandID.get(productBrandID) ?? [])],
						queryWordMatches
					});
				}
			}
		}

		const sortedRankedProductIDs = Array.from(rankingByProductID.entries()).sort(
			([leftProductID, leftRank], [rightProductID, rightRank]) => {
				if (rightRank !== leftRank) {
					// Primary order: total score descending.
					return rightRank - leftRank;
				}
				const leftTieBreakInfo = rankingTieBreakByProductID.get(leftProductID);
				const rightTieBreakInfo = rankingTieBreakByProductID.get(rightProductID);
				const leftExactWordMatchCount = leftTieBreakInfo?.exactWordMatchCount ?? 0;
				const rightExactWordMatchCount = rightTieBreakInfo?.exactWordMatchCount ?? 0;
				if (rightExactWordMatchCount !== leftExactWordMatchCount) {
					// Tie-breaker 1: prefer products with more exact word matches.
					return rightExactWordMatchCount - leftExactWordMatchCount;
				}
				const leftBestSyllablePrefixLength = leftTieBreakInfo?.bestSyllablePrefixLength ?? 0;
				const rightBestSyllablePrefixLength = rightTieBreakInfo?.bestSyllablePrefixLength ?? 0;
				if (rightBestSyllablePrefixLength !== leftBestSyllablePrefixLength) {
					// Tie-breaker 2: prefer longer matched syllable prefixes.
					return rightBestSyllablePrefixLength - leftBestSyllablePrefixLength;
				}
				// Keep deterministic order as final fallback.
				return leftProductID - rightProductID;
			}
		);

		const topSearchHits: ProductSearchHit[] = [];
		for (const [productID, rank] of sortedRankedProductIDs.slice(0, 20)) {
			// Materialize top-20 hits only; ranking map can be larger.
			const product = this.get(productID);
			if (!product) {
				continue;
			}
			topSearchHits.push({ product, rank });
		}

		// Persist the full ranked-debug payload only when debug mode is explicitly enabled.
		this.latestSearchDebugSnapshot = fullDebugEnabled
			? {
					queryText,
					normalizedQueryTokens: encodedQueryTokens,
					encodedQueryWords: debugEncodedQueryWords,
					rankedProducts: sortedRankedProductIDs
						.map(([productID]) => rankingDebugInfoByProductID.get(productID))
						.filter((rankingDebugInfo): rankingDebugInfo is ProductRankingDebugInfo =>
							Boolean(rankingDebugInfo)
						),
					generatedAtISO: new Date().toISOString()
			  }
			: null;
		const searchElapsedMs =
			(typeof performance !== "undefined" ? performance.now() : Date.now()) - searchStartedAtMs;
		console.log(
			`Search: ${queryText}. Found: ${sortedRankedProductIDs.length} matches in ${Math.max(0, Math.round(searchElapsedMs))}ms`
		);
		return topSearchHits;
	}

	getLastSearchDebugSnapshot(): ProductSearchDebugSnapshot | null {
		return this.latestSearchDebugSnapshot;
	}

	private buildZeroSeparatedWordBytes(
		productSyllableBytes: Uint8Array,
		productShape: number[]
	): Uint8Array {
		// Layout output as [word syllables..., 0, word syllables..., 0, ...] using decoded shape boundaries.
		const separatorCount = productShape.length > 0 ? productShape.length - 1 : 0;
		const separatedWordBytes = new Uint8Array(productSyllableBytes.length + separatorCount);
		let sourceOffset = 0;
		let destinationOffset = 0;

		for (let wordIndex = 0; wordIndex < productShape.length; wordIndex++) {
			const syllableCountForWord = productShape[wordIndex];
			for (let syllableOffset = 0; syllableOffset < syllableCountForWord; syllableOffset++) {
				separatedWordBytes[destinationOffset] = productSyllableBytes[sourceOffset];
				destinationOffset++;
				sourceOffset++;
			}
			if (wordIndex < productShape.length - 1) {
				separatedWordBytes[destinationOffset] = 0;
				destinationOffset++;
			}
		}
		return separatedWordBytes;
	}

	private decodeLossyName(productBytesWithWordSeparator: Uint8Array): string {
		// Convert token IDs to words, where 0 acts as hard word boundary in the compact byte layout.
		const decodedWords: string[] = [];
		let currentWord = "";
		for (const dictionaryTokenID of productBytesWithWordSeparator) {
			if (dictionaryTokenID === 0) {
				if (currentWord) {
					decodedWords.push(currentWord);
					currentWord = "";
				}
				continue;
			}
			if (
				dictionaryTokenID > 0 &&
				dictionaryTokenID <= this.dictionaryTokens.length
			) {
				currentWord += this.dictionaryTokens[dictionaryTokenID - 1];
			}
		}
		if (currentWord) {
			decodedWords.push(currentWord);
		}
		return decodedWords.join(" ");
	}

	private splitProductWordsBySeparator(productNameBytes: Uint8Array): Uint8Array[] {
		// Split one row into word views without copying data (subarray-based).
		const productWordSyllables: Uint8Array[] = [];
		let wordStartOffset = 0;
		for (let byteOffset = 0; byteOffset < productNameBytes.length; byteOffset++) {
			if (productNameBytes[byteOffset] !== 0) {
				continue;
			}
			if (byteOffset > wordStartOffset) {
				// Use subarray to avoid copying word bytes in the search hot path.
				productWordSyllables.push(productNameBytes.subarray(wordStartOffset, byteOffset));
			}
			wordStartOffset = byteOffset + 1;
		}
		if (wordStartOffset < productNameBytes.length) {
			productWordSyllables.push(productNameBytes.subarray(wordStartOffset));
		}
		return productWordSyllables;
	}

	private evaluateQueryWordAgainstProductWords(
		queryWord: string,
		queryWordSyllables: Uint8Array,
		productWordSyllables: Uint8Array[]
	): ProductQueryWordMatchDebugInfo {
		// Track best product-word candidate so ranking output can explain why a word scored.
		let bestMatchedSyllablePrefixLength = 0;
		let bestMatchedSyllablePrefixPoints = 0;
		let bestMatchedCharacterPrefixLength = 0;
		let hasExactWordMatch = false;
		let matchedProductWord = "";
		let matchedBy: "syllable_prefix" | "character_prefix" | "none" = "none";

		for (const productWord of productWordSyllables) {
			const syllablePrefixMatch = this.computeSyllablePrefixMatch(queryWordSyllables, productWord);
			const matchedSyllablePrefixLength = syllablePrefixMatch.matchedSyllablePrefixLength;
			const matchedSyllablePrefixPoints = syllablePrefixMatch.matchedSyllablePrefixPoints;

			// Character-prefix fallback enables matches like query "cho" against syllable pairs "ch"+"oc".
			const decodedProductWord = this.decodeWordFromSyllables(productWord);
			const matchedCharacterPrefixLength = this.countSharedPrefixCharacters(
				queryWord,
				decodedProductWord
			);

			const hasBetterCharacterPrefix =
				matchedCharacterPrefixLength > bestMatchedCharacterPrefixLength;
			const hasSameCharacterButBetterSyllable =
				matchedCharacterPrefixLength === bestMatchedCharacterPrefixLength &&
				matchedSyllablePrefixPoints > bestMatchedSyllablePrefixPoints;
			if (hasBetterCharacterPrefix || hasSameCharacterButBetterSyllable) {
				// Keep the strongest match candidate so debug output reflects final scoring path.
				bestMatchedCharacterPrefixLength = matchedCharacterPrefixLength;
				bestMatchedSyllablePrefixLength = matchedSyllablePrefixLength;
				bestMatchedSyllablePrefixPoints = matchedSyllablePrefixPoints;
				matchedProductWord = decodedProductWord;
				if (matchedSyllablePrefixLength > 0) {
					matchedBy = "syllable_prefix";
				} else if (matchedCharacterPrefixLength > 0) {
					matchedBy = "character_prefix";
				} else {
					matchedBy = "none";
				}
			}

			if (
				matchedCharacterPrefixLength === queryWord.length &&
				decodedProductWord.length === queryWord.length
			) {
				hasExactWordMatch = true;
			}
		}

		const pointsAwarded = this.computeWordMatchPoints(
			bestMatchedSyllablePrefixPoints,
			hasExactWordMatch
		);

		return {
			queryWord,
			matchedBy,
			matchedProductWord,
			matchedSyllablePrefixLength: bestMatchedSyllablePrefixLength,
			matchedCharacterPrefixLength: bestMatchedCharacterPrefixLength,
			exactWordMatch: hasExactWordMatch,
			pointsAwarded
		};
	}

	private hasAnyPrefixMatchInWordBySyllables(
		encodedQueryWords: Array<{ word: string; syllables: Uint8Array }>,
		productWordSyllables: Uint8Array
	): boolean {
		// Positional bonus helper: checks if query shares at least one leading syllable with the word.
		for (const encodedQueryWord of encodedQueryWords) {
			const querySyllables = encodedQueryWord.syllables;
			const comparableLength = Math.min(querySyllables.length, productWordSyllables.length);
			for (let offset = 0; offset < comparableLength; offset++) {
				if (querySyllables[offset] !== productWordSyllables[offset]) {
					break;
				}
				// Partial or full syllable-prefix match qualifies for positional bonus.
				return true;
			}
		}
		return false;
	}

	private evaluateQueryWordAgainstProductWordsFast(
		queryWordSyllables: Uint8Array,
		productWordSyllables: Uint8Array[]
	): FastQueryWordMatchResult {
		// Fast path skips verbose debug details but preserves the same scoring behavior.
		let bestMatchedSyllablePrefixLength = 0;
		let bestMatchedSyllablePrefixPoints = 0;
		let hasExactWordMatch = false;
		for (const productWord of productWordSyllables) {
			const syllablePrefixMatch = this.computeSyllablePrefixMatch(queryWordSyllables, productWord);
			const matchedSyllablePrefixLength = syllablePrefixMatch.matchedSyllablePrefixLength;
			const matchedSyllablePrefixPoints = syllablePrefixMatch.matchedSyllablePrefixPoints;
			if (matchedSyllablePrefixLength > bestMatchedSyllablePrefixLength) {
				bestMatchedSyllablePrefixLength = matchedSyllablePrefixLength;
			}
			if (matchedSyllablePrefixPoints > bestMatchedSyllablePrefixPoints) {
				bestMatchedSyllablePrefixPoints = matchedSyllablePrefixPoints;
			}
			if (
				matchedSyllablePrefixLength === queryWordSyllables.length &&
				productWord.length === queryWordSyllables.length
			) {
				hasExactWordMatch = true;
			}
		}

		const pointsAwarded = this.computeWordMatchPoints(
			bestMatchedSyllablePrefixPoints,
			hasExactWordMatch
		);
		return {
			pointsAwarded,
			exactWordMatch: hasExactWordMatch,
			bestSyllablePrefixLength: bestMatchedSyllablePrefixLength
		};
	}

	private computeBrandScoringByBrandID(
		encodedQueryWords: Array<{ word: string; syllables: Uint8Array }>,
		includeDebugMatches = false
	): {
		brandPointsByBrandID: Map<number, number>;
		brandMatchesByBrandID: Map<number, ProductBrandMatchDebugInfo[]>;
	} {
		// Score all brands once per query; products then reuse points by their brandID.
		const brandPointsByBrandID = new Map<number, number>();
		const brandMatchesByBrandID = new Map<number, ProductBrandMatchDebugInfo[]>();
		for (const [brandID, brandWords] of this.normalizedBrandWordsByBrandID.entries()) {
			let totalBonusPoints = 0;
			const brandMatches: ProductBrandMatchDebugInfo[] = [];
			for (const encodedQueryWord of encodedQueryWords) {
				const bestBrandWordMatch = this.findBestBrandWordMatch(encodedQueryWord, brandWords);
				if (bestBrandWordMatch.pointsAwarded > 0) {
					totalBonusPoints += bestBrandWordMatch.pointsAwarded;
					if (includeDebugMatches) {
						brandMatches.push({
							queryWord: encodedQueryWord.word,
							brandWord: bestBrandWordMatch.brandWord,
							pointsAwarded: bestBrandWordMatch.pointsAwarded
						});
					}
				}
			}
			if (totalBonusPoints > 0) {
				brandPointsByBrandID.set(brandID, totalBonusPoints);
				if (includeDebugMatches) {
					// Debug payload only stores contributing brand-word matches when explicitly requested.
					brandMatchesByBrandID.set(brandID, brandMatches);
				}
			}
		}
		return { brandPointsByBrandID, brandMatchesByBrandID };
	}

	private decodeWordFromSyllables(wordSyllableIDs: Uint8Array): string {
		// Reconstruct a normalized word from dictionary IDs for character-level prefix matching.
		let decodedWord = "";
		for (const dictionaryTokenID of wordSyllableIDs) {
			if (dictionaryTokenID <= 0 || dictionaryTokenID > this.dictionaryTokens.length) {
				continue;
			}
			decodedWord += this.dictionaryTokens[dictionaryTokenID - 1];
		}
		return normalizeSearchToken(decodedWord).replace(/\s+/g, "");
	}

	private countSharedPrefixCharacters(queryWord: string, candidateWord: string): number {
		// Character-prefix metric is used as a fallback tie-breaker when syllable ids diverge.
		const comparableLength = Math.min(queryWord.length, candidateWord.length);
		let matchedPrefixLength = 0;
		for (let charOffset = 0; charOffset < comparableLength; charOffset++) {
			if (queryWord[charOffset] !== candidateWord[charOffset]) {
				break;
			}
			matchedPrefixLength++;
		}
		return matchedPrefixLength;
	}

	private computeWordMatchPoints(syllablePrefixPoints: number, exactWordMatch: boolean): number {
		// Shared word scoring primitive used by both product-name and brand matching logic.
		let pointsAwarded = syllablePrefixPoints;
		if (exactWordMatch) {
			pointsAwarded += ProductSearch.COMPLETE_WORD_MATCH_POINTS;
		}
		return pointsAwarded;
	}

	private findBestBrandWordMatch(
		encodedQueryWord: { word: string; syllables: Uint8Array },
		brandWords: readonly string[]
	): { brandWord: string; pointsAwarded: number } {
		// For each query word, keep only the highest-scoring brand word contribution.
		let bestMatchedBrandWord = "";
		let bestWordPointsForQuery = 0;
		for (const brandWord of brandWords) {
			const encodedBrandWordSyllables = encodeQueryWordToDictionarySyllables(
				brandWord,
				this.dictionaryTokenIDByNormalizedToken,
				this.aliasDictionaryIDByNormalizedToken
			);
			const syllablePrefixMatch = this.computeSyllablePrefixMatch(
				encodedQueryWord.syllables,
				encodedBrandWordSyllables
			);
			let wordPoints = this.computeWordMatchPoints(
				syllablePrefixMatch.matchedSyllablePrefixPoints,
				brandWord === encodedQueryWord.word
			);
			if (wordPoints > 0) {
				// Brand relevance tracks name relevance but with a -1 penalty per matched word.
				wordPoints = Math.max(1, wordPoints - ProductSearch.BRAND_WORD_MATCH_PENALTY_POINTS);
			}
			if (wordPoints > bestWordPointsForQuery) {
				bestWordPointsForQuery = wordPoints;
				bestMatchedBrandWord = brandWord;
			}
		}
		return {
			brandWord: bestMatchedBrandWord,
			pointsAwarded: bestWordPointsForQuery
		};
	}

	private computeSyllablePrefixMatch(
		queryWordSyllables: Uint8Array,
		candidateWordSyllables: Uint8Array
	): { matchedSyllablePrefixLength: number; matchedSyllablePrefixPoints: number } {
		// Prefix scoring works on encoded syllable ids to avoid repeated string allocations.
		const comparableLength = Math.min(queryWordSyllables.length, candidateWordSyllables.length);
		let matchedSyllablePrefixLength = 0;
		let matchedSyllablePrefixPoints = 0;
		for (let offset = 0; offset < comparableLength; offset++) {
			if (queryWordSyllables[offset] !== candidateWordSyllables[offset]) {
				break;
			}
			matchedSyllablePrefixLength++;
			matchedSyllablePrefixPoints += this.getSyllableMatchPoints(queryWordSyllables[offset]);
		}
		return { matchedSyllablePrefixLength, matchedSyllablePrefixPoints };
	}

	private getSyllableMatchPoints(dictionaryTokenID: number): number {
		// One-letter syllables are weaker signals than two-letter syllables.
		if (dictionaryTokenID <= 0 || dictionaryTokenID > this.dictionaryTokens.length) {
			return 0;
		}
		const dictionaryToken = this.dictionaryTokens[dictionaryTokenID - 1];
		const dictionaryTokenLength = [...dictionaryToken].length;
		return dictionaryTokenLength <= 1
			? ProductSearch.SYLLABLE_MATCH_POINTS_ONE_LETTER
			: ProductSearch.SYLLABLE_MATCH_POINTS_TWO_LETTER;
	}
}
