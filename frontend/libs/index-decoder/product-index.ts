import { decodeBinary } from "./decoder";
import {
	encodeQueryWordToDictionarySyllables,
	normalizeAndFilterQueryTokens,
	normalizeSearchToken,
	splitNormalizedWords
} from "./encoder";
import { Env } from "$core/env";
import type { IndexedProduct, ProductSearchHit } from "./types";
import { ProductosDeltaService } from "$services/services/productos.svelte";

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

const productsDeltaService = new ProductosDeltaService()

export class ProductIndex {
	// Keep scoring constants explicit so ranking behavior is predictable and easy to tune.
	private static readonly SYLLABLE_MATCH_POINTS = 4;
	private static readonly COMPLETE_WORD_MATCH_POINTS = 5;
	private static readonly BRAND_PREFIX_MATCH_POINTS = 2;
	private static readonly FIRST_WORD_POSITIONAL_POINTS = 2;
	private static readonly SECOND_WORD_POSITIONAL_POINTS = 1;

	private readonly productNameBytesByProductID = new Map<number, Uint8Array>();
	private readonly brandDictionaryIndexByProductID = new Map<number, number>();
	private readonly brandIDMapByProductID = new Map<number, number>();
	private readonly categoryIDsMapByProductID = new Map<number, number[]>();
	private readonly productIDsSorted: number[] = [];
	private readonly dictionaryTokens: string[] = [];
	private readonly dictionaryTokenIDByNormalizedToken = new Map<string, number>();
	private readonly aliasDictionaryIDByNormalizedToken = new Map<string, number>();
	private readonly brandNameByID = new Map<number, string>();
	private readonly categoryNameByID = new Map<number, string>();
	private readonly normalizedBrandWordsByBrandID = new Map<number, string[]>();
	private readonly updatedInt32: number;
	private latestSearchDebugSnapshot: ProductSearchDebugSnapshot | null = null;

	constructor(indexBytesInput: Uint8Array | ArrayBuffer) {
		const decodedPayload = decodeBinary(indexBytesInput);
		// Keep build/update watermark from the text header for delta sync workflows.
		this.updatedInt32 = decodedPayload.stats.updated;
		this.dictionaryTokens = decodedPayload.dictionaryTokens;
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
			const productID = decodedRecord.productID;
			if (this.productNameBytesByProductID.has(productID)) {
				throw new Error(`duplicated product id=${productID}`);
			}
			this.productIDsSorted.push(productID);
			this.productNameBytesByProductID.set(productID, separatedWordBytes);
			this.brandDictionaryIndexByProductID.set(productID, decodedRecord.brandDictionaryIndex);
			this.brandIDMapByProductID.set(productID, decodedRecord.brandID);
			this.categoryIDsMapByProductID.set(productID, [...decodedRecord.categoryIDs]);
		}

		// Keep tiny taxonomy dictionaries for on-demand name resolution during get(productID).
		if (decodedPayload.taxonomy) {
			for (let brandIndex = 0; brandIndex < decodedPayload.taxonomy.brandIDs.length; brandIndex++) {
				const brandName = decodedPayload.taxonomy.brandNames[brandIndex];
				const brandID = decodedPayload.taxonomy.brandIDs[brandIndex];
				this.brandNameByID.set(
					brandID,
					brandName
				);
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
		this.loadDeltas()
	}

	async loadDeltas() {
		await productsDeltaService.fetchOnline()
		console.log("productsDeltaService", productsDeltaService, productsDeltaService.records.length)
	}
	
	static fromBinary(indexBytesInput: Uint8Array | ArrayBuffer): ProductIndex {
		return new ProductIndex(indexBytesInput);
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
		const debugBrandScoring = fullDebugEnabled
			? this.computeBrandScoringByBrandID(encodedQueryTokens)
			: null;
		const brandPointsByBrandID =
			debugBrandScoring?.brandPointsByBrandID ??
			this.computeBrandPointsByBrandID(encodedQueryTokens);
		const brandMatchesByBrandID =
			debugBrandScoring?.brandMatchesByBrandID ?? new Map<number, ProductBrandMatchDebugInfo[]>();
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
				firstWordPoints = ProductIndex.FIRST_WORD_POSITIONAL_POINTS;
			}
			let secondWordPoints = 0;
			if (
				productWordSyllables.length > 1 &&
				this.hasAnyPrefixMatchInWordBySyllables(encodedQueryWords, productWordSyllables[1])
			) {
				secondWordPoints = ProductIndex.SECOND_WORD_POSITIONAL_POINTS;
			}
			const totalRankPoints = brandPoints + productNamePoints + firstWordPoints + secondWordPoints;

			if (totalRankPoints > 0) {
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
					return rightRank - leftRank;
				}
				const leftTieBreakInfo = rankingTieBreakByProductID.get(leftProductID);
				const rightTieBreakInfo = rankingTieBreakByProductID.get(rightProductID);
				const leftExactWordMatchCount = leftTieBreakInfo?.exactWordMatchCount ?? 0;
				const rightExactWordMatchCount = rightTieBreakInfo?.exactWordMatchCount ?? 0;
				if (rightExactWordMatchCount !== leftExactWordMatchCount) {
					return rightExactWordMatchCount - leftExactWordMatchCount;
				}
				const leftBestSyllablePrefixLength = leftTieBreakInfo?.bestSyllablePrefixLength ?? 0;
				const rightBestSyllablePrefixLength = rightTieBreakInfo?.bestSyllablePrefixLength ?? 0;
				if (rightBestSyllablePrefixLength !== leftBestSyllablePrefixLength) {
					return rightBestSyllablePrefixLength - leftBestSyllablePrefixLength;
				}
				// Keep deterministic order as final fallback.
				return leftProductID - rightProductID;
			}
		);

		const topSearchHits: ProductSearchHit[] = [];
		for (const [productID, rank] of sortedRankedProductIDs.slice(0, 20)) {
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
		const minimumCharacterPrefixLength = Math.min(2, queryWord.length);
		let bestMatchedSyllablePrefixLength = 0;
		let bestMatchedCharacterPrefixLength = 0;
		let hasExactWordMatch = false;
		let matchedProductWord = "";
		let matchedBy: "syllable_prefix" | "character_prefix" | "none" = "none";

		for (const productWord of productWordSyllables) {
			const comparableLength = Math.min(queryWordSyllables.length, productWord.length);
			let matchedSyllablePrefixLength = 0;
			for (let offset = 0; offset < comparableLength; offset++) {
				if (queryWordSyllables[offset] !== productWord[offset]) {
					break;
				}
				matchedSyllablePrefixLength += 1;
			}

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
				matchedSyllablePrefixLength > bestMatchedSyllablePrefixLength;
			if (hasBetterCharacterPrefix || hasSameCharacterButBetterSyllable) {
				bestMatchedCharacterPrefixLength = matchedCharacterPrefixLength;
				bestMatchedSyllablePrefixLength = matchedSyllablePrefixLength;
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

		let pointsAwarded = 0;
		// Core relevance is awarded per matched syllable (uint8 token IDs).
		pointsAwarded += bestMatchedSyllablePrefixLength * ProductIndex.SYLLABLE_MATCH_POINTS;
		if (hasExactWordMatch) {
			pointsAwarded += ProductIndex.COMPLETE_WORD_MATCH_POINTS;
		}

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

	private hasAnyPrefixMatchInWord(
		encodedQueryWords: Array<{ word: string; syllables: Uint8Array }>,
		productWordSyllables: Uint8Array
	): boolean {
		const decodedProductWord = this.decodeWordFromSyllables(productWordSyllables);
		for (const encodedQueryWord of encodedQueryWords) {
			const minimumCharacterPrefixLength = Math.min(2, encodedQueryWord.word.length);
			const querySyllables = encodedQueryWord.syllables;
			const comparableLength = Math.min(querySyllables.length, productWordSyllables.length);
			const matchedCharacterPrefixLength = this.countSharedPrefixCharacters(
				encodedQueryWord.word,
				decodedProductWord
			);
			if (matchedCharacterPrefixLength >= minimumCharacterPrefixLength) {
				return true;
			}
			for (let offset = 0; offset < comparableLength; offset++) {
				if (querySyllables[offset] !== productWordSyllables[offset]) {
					break;
				}
				// Partial or full prefix match both qualify for positional bonus.
				return true;
			}
		}
		return false;
	}

	private hasAnyPrefixMatchInWordBySyllables(
		encodedQueryWords: Array<{ word: string; syllables: Uint8Array }>,
		productWordSyllables: Uint8Array
	): boolean {
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
		let bestMatchedSyllablePrefixLength = 0;
		let hasExactWordMatch = false;
		for (const productWord of productWordSyllables) {
			const comparableLength = Math.min(queryWordSyllables.length, productWord.length);
			let matchedSyllablePrefixLength = 0;
			for (let offset = 0; offset < comparableLength; offset++) {
				if (queryWordSyllables[offset] !== productWord[offset]) {
					break;
				}
				matchedSyllablePrefixLength++;
			}
			if (matchedSyllablePrefixLength > bestMatchedSyllablePrefixLength) {
				bestMatchedSyllablePrefixLength = matchedSyllablePrefixLength;
			}
			if (
				matchedSyllablePrefixLength === queryWordSyllables.length &&
				productWord.length === queryWordSyllables.length
			) {
				hasExactWordMatch = true;
			}
		}

		let pointsAwarded =
			bestMatchedSyllablePrefixLength * ProductIndex.SYLLABLE_MATCH_POINTS;
		if (hasExactWordMatch) {
			pointsAwarded += ProductIndex.COMPLETE_WORD_MATCH_POINTS;
		}
		return {
			pointsAwarded,
			exactWordMatch: hasExactWordMatch,
			bestSyllablePrefixLength: bestMatchedSyllablePrefixLength
		};
	}

	private computeBrandScoringByBrandID(normalizedQueryWords: string[]): {
		brandPointsByBrandID: Map<number, number>;
		brandMatchesByBrandID: Map<number, ProductBrandMatchDebugInfo[]>;
	} {
		const brandPointsByBrandID = new Map<number, number>();
		const brandMatchesByBrandID = new Map<number, ProductBrandMatchDebugInfo[]>();
		for (const [brandID, brandWords] of this.normalizedBrandWordsByBrandID.entries()) {
			let totalBonusPoints = 0;
			const brandMatches: ProductBrandMatchDebugInfo[] = [];
			for (const queryWord of normalizedQueryWords) {
				for (const brandWord of brandWords) {
					// Brand influence is intentionally lower than name relevance: +2 fixed per matched query token.
					if (brandWord === queryWord || brandWord.startsWith(queryWord)) {
						totalBonusPoints += ProductIndex.BRAND_PREFIX_MATCH_POINTS;
						brandMatches.push({
							queryWord,
							brandWord,
							pointsAwarded: ProductIndex.BRAND_PREFIX_MATCH_POINTS
						});
						break;
					}
				}
			}
			if (totalBonusPoints > 0) {
				brandPointsByBrandID.set(brandID, totalBonusPoints);
				brandMatchesByBrandID.set(brandID, brandMatches);
			}
		}
		return { brandPointsByBrandID, brandMatchesByBrandID };
	}

	private computeBrandPointsByBrandID(normalizedQueryWords: string[]): Map<number, number> {
		const brandPointsByBrandID = new Map<number, number>();
		for (const [brandID, brandWords] of this.normalizedBrandWordsByBrandID.entries()) {
			let totalBonusPoints = 0;
			for (const queryWord of normalizedQueryWords) {
				for (const brandWord of brandWords) {
					if (brandWord === queryWord || brandWord.startsWith(queryWord)) {
						totalBonusPoints += ProductIndex.BRAND_PREFIX_MATCH_POINTS;
						break;
					}
				}
			}
			if (totalBonusPoints > 0) {
				brandPointsByBrandID.set(brandID, totalBonusPoints);
			}
		}
		return brandPointsByBrandID;
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
}
