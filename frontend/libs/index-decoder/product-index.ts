import { decodeBinary } from "./decoder";
import {
	encodeQueryWordToDictionarySyllables,
	normalizeAndFilterQueryTokens,
	normalizeSearchToken,
	splitNormalizedWords
} from "./encoder";
import type { IndexedProduct, ProductSearchHit } from "./types";

export class ProductIndex {
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

	search(queryText: string): ProductSearchHit[] {
		const encodedQueryTokens = normalizeAndFilterQueryTokens(queryText);
		if (encodedQueryTokens.length === 0) {
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
			}))
			.filter((encodedWord) => encodedWord.syllables.length > 0);
		if (encodedQueryWords.length === 0) {
			return [];
		}

		// Score query-brand affinity once, then apply bonus only to products that belong to matched brands.
		const brandBonusByBrandID = this.computeBrandBonusByBrandID(encodedQueryTokens);

		const rankingByProductID = new Map<number, number>();
		for (const productID of this.productIDsSorted) {
			const productNameBytes = this.productNameBytesByProductID.get(productID);
			if (!productNameBytes || productNameBytes.length === 0) {
				continue;
			}

			// Split product byte row by 0 separators to evaluate per-word prefix matches.
			const productWordSyllables = this.splitProductWordsBySeparator(productNameBytes);
			let totalRankPoints = 0;
			const productBrandID = this.brandIDMapByProductID.get(productID) ?? 0;
			totalRankPoints += brandBonusByBrandID.get(productBrandID) ?? 0;
			for (const encodedQueryWord of encodedQueryWords) {
				totalRankPoints += this.computeBestWordPrefixSyllableScore(
					encodedQueryWord.syllables,
					productWordSyllables
				);
			}
			// Expand score range so lower/intermediate matches have more ranking headroom.
			totalRankPoints *= 4;
			if (
				productWordSyllables.length > 0 &&
				this.hasAnyPrefixMatchInWord(encodedQueryWords, productWordSyllables[0])
			) {
				totalRankPoints += 2;
			}
			if (
				productWordSyllables.length > 1 &&
				this.hasAnyPrefixMatchInWord(encodedQueryWords, productWordSyllables[1])
			) {
				totalRankPoints += 1;
			}

			if (totalRankPoints > 0) {
				rankingByProductID.set(productID, totalRankPoints);
			}
		}

		const sortedRankedProductIDs = Array.from(rankingByProductID.entries()).sort(
			([leftProductID, leftRank], [rightProductID, rightRank]) =>
				rightRank - leftRank || leftProductID - rightProductID
		);

		const topSearchHits: ProductSearchHit[] = [];
		for (const [productID, rank] of sortedRankedProductIDs.slice(0, 20)) {
			const product = this.get(productID);
			if (!product) {
				continue;
			}
			topSearchHits.push({ product, rank });
		}
		return topSearchHits;
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
				productWordSyllables.push(productNameBytes.slice(wordStartOffset, byteOffset));
			}
			wordStartOffset = byteOffset + 1;
		}
		if (wordStartOffset < productNameBytes.length) {
			productWordSyllables.push(productNameBytes.slice(wordStartOffset));
		}
		return productWordSyllables;
	}

	private computeBestWordPrefixSyllableScore(
		queryWordSyllables: Uint8Array,
		productWordSyllables: Uint8Array[]
	): number {
		// Score by prefix syllable matches from the start of both words and keep the best word hit.
		let bestWordMatchPoints = 0;
		for (const productWord of productWordSyllables) {
			const comparableLength = Math.min(queryWordSyllables.length, productWord.length);
			let matchedPrefixSyllables = 0;
			for (let offset = 0; offset < comparableLength; offset++) {
				if (queryWordSyllables[offset] !== productWord[offset]) {
					break;
				}
				matchedPrefixSyllables += 1;
			}
			let wordMatchScore = matchedPrefixSyllables;
			const isWholeWordMatch =
				matchedPrefixSyllables === queryWordSyllables.length &&
				queryWordSyllables.length === productWord.length;
			if (isWholeWordMatch) {
				wordMatchScore += 1;
				if (productWord.length >= 3) {
					wordMatchScore += 2;
				}
			}
			if (wordMatchScore > bestWordMatchPoints) {
				bestWordMatchPoints = wordMatchScore;
			}
		}
		return bestWordMatchPoints;
	}

	private hasAnyPrefixMatchInWord(
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
				// Partial or full prefix match both qualify for positional bonus.
				return true;
			}
		}
		return false;
	}

	private computeBrandBonusByBrandID(normalizedQueryWords: string[]): Map<number, number> {
		const brandBonusByBrandID = new Map<number, number>();
		for (const [brandID, brandWords] of this.normalizedBrandWordsByBrandID.entries()) {
			let totalBonusPoints = 0;
			for (const queryWord of normalizedQueryWords) {
				for (const brandWord of brandWords) {
					if (brandWord === queryWord) {
						totalBonusPoints += Math.ceil(queryWord.length / 2);
						break;
					}
					let hasPrefixMatch = false;
					for (const prefixLength of [5, 4, 3]) {
						if (queryWord.length < prefixLength) {
							continue;
						}
						const queryPrefix = queryWord.slice(0, prefixLength);
						if (!brandWord.startsWith(queryPrefix)) {
							continue;
						}
						totalBonusPoints += Math.ceil(prefixLength / 2);
						hasPrefixMatch = true;
						break;
					}
					if (hasPrefixMatch) {
						break;
					}
				}
			}
			if (totalBonusPoints > 0) {
				brandBonusByBrandID.set(brandID, totalBonusPoints);
			}
		}
		return brandBonusByBrandID;
	}
}
