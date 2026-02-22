// Product-search binary decoder: validates idx/taxonomy payloads and reconstructs searchable records.
import {
	BRAND_INDEX_ENCODING_UINT12,
	BRAND_INDEX_ENCODING_UINT16,
	TAXONOMY_BINARY_MAGIC,
	TAXONOMY_BINARY_VERSION,
	TAXONOMY_SECTION_BRAND_IDS,
	TAXONOMY_SECTION_BRAND_INDEXES,
	TAXONOMY_SECTION_BRAND_NAMES,
	TAXONOMY_SECTION_CATEGORY_COUNTS,
	TAXONOMY_SECTION_CATEGORY_IDS,
	TAXONOMY_SECTION_CATEGORY_INDEX,
	TAXONOMY_SECTION_CATEGORY_NAMES,
	PRODUCT_ID_UINT32_SENTINEL,
	TEXT_BINARY_MAGIC,
	TEXT_BINARY_VERSION,
	TEXT_SECTION_ALIASES,
	TEXT_SECTION_CONTENT,
	TEXT_SECTION_DICTIONARY,
	TEXT_SECTION_PRODUCT_IDS,
	TEXT_SECTION_SHAPES
} from "./types";
import type { DecodeResult, DecodedRecord, DecodedTaxonomy } from "./types";

import {
	brandEncodingName,
	computeCrc32,
	decodeAscii,
	expectedUint12PackedBytes,
	readUint16LE,
	readUint32LE,
	readWithRecordContext
} from "./helpers";

interface BinarySectionDescriptor {
	sectionID: number;
	offset: number;
	length: number;
	itemCount: number;
	checksum: number;
}

interface DecodedTextHeader {
	recordCount: number;
	buildSunixTime: number;
	dictionaryCount: number;
	sectionsByID: Map<number, BinarySectionDescriptor>;
	payloadEnd: number;
}

// Decoder supports both ArrayBuffer and Uint8Array inputs from fetch/file APIs.
export const decodeBinary = (indexBytesInput: Uint8Array | ArrayBuffer): DecodeResult => {
	// Normalize input once so the rest of the decoder only works with Uint8Array views.
	const indexBytes =
		indexBytesInput instanceof Uint8Array ? indexBytesInput : new Uint8Array(indexBytesInput);
	// Decode and validate the text header + section table before touching payload data.
	const textHeader = decodeTextHeaderV2(indexBytes);

	// Resolve each mandatory text section by id (dictionary, shapes, content, aliases, product IDs).
	const dictionarySection = requiredSection(indexBytes, textHeader.sectionsByID, TEXT_SECTION_DICTIONARY);
	const shapeSection = requiredSection(indexBytes, textHeader.sectionsByID, TEXT_SECTION_SHAPES);
	const contentSection = requiredSection(indexBytes, textHeader.sectionsByID, TEXT_SECTION_CONTENT);
	const aliasSection = requiredSection(indexBytes, textHeader.sectionsByID, TEXT_SECTION_ALIASES);
	const productIDsSection = requiredSection(indexBytes, textHeader.sectionsByID, TEXT_SECTION_PRODUCT_IDS);

	// Decode text payload columns and rebuild record rows in the exact sorted index order.
	const dictionaryTokens = decodeDictionarySectionRaw(dictionarySection, textHeader.dictionaryCount);
	const decodedProductIDs = decodeProductIDsSection(productIDsSection, textHeader.recordCount);
	const decodedShapeDelta = decodeShapeDeltaStream(shapeSection, textHeader.recordCount);
	const decodedShapes = decodedShapeDelta.shapeValues.map((shapeValue) => decodeShapeValue(shapeValue));
	const decodedRecords = decodeContentRecords(
		contentSection,
		decodedShapes,
		dictionaryTokens,
		decodedProductIDs
	);
	const aliasDescriptor = textHeader.sectionsByID.get(TEXT_SECTION_ALIASES)!;
	const dictionaryAliases = decodeAliasSection(
		aliasSection,
		dictionaryTokens.length,
		aliasDescriptor.itemCount
	);

	if (decodedRecords.length !== textHeader.recordCount) {
		throw new Error(
			`decoded record count mismatch expected=${textHeader.recordCount} got=${decodedRecords.length}`
		);
	}

	const decodeResult: DecodeResult = {
		dictionaryTokens,
		dictionaryAliases,
		records: decodedRecords,
		taxonomy: null,
		stats: {
			recordCount: textHeader.recordCount,
			buildSunixTime: textHeader.buildSunixTime,
			updated: textHeader.buildSunixTime,
			dictionaryCount: textHeader.dictionaryCount,
			dictionaryBytes: dictionarySection.length,
			aliasBytes: aliasSection.length,
			shapesBytes: shapeSection.length,
			contentBytes: contentSection.length,
			productIDsBytes: productIDsSection.length,
			shapeDelta8Count: decodedShapeDelta.delta8Count,
			shapeDelta16Count: decodedShapeDelta.delta16Count,
			shapeDelta24Count: decodedShapeDelta.delta24Count,
			taxonomyBytes: 0,
			taxonomyBrandCount: 0,
			taxonomyCategoryCount: 0,
			taxonomyBrandIndexMode: ""
		}
	};

	if (textHeader.payloadEnd === indexBytes.length) {
		// Some payloads only contain the text block; taxonomy is optional by design.
		return decodeResult;
	}

	// Any trailing bytes after text payload are interpreted as taxonomy block.
	const taxonomySlice = indexBytes.subarray(textHeader.payloadEnd);
	const decodedTaxonomyBlock = decodeTaxonomyBlockV2(taxonomySlice, textHeader.recordCount);
	// Mutate decoded records in-place to attach brand/category information by row index.
	attachTaxonomyToRecords(decodeResult.records, decodedTaxonomyBlock.taxonomy);

	decodeResult.taxonomy = decodedTaxonomyBlock.taxonomy;
	decodeResult.stats.taxonomyBytes = decodedTaxonomyBlock.payloadEnd;
	decodeResult.stats.taxonomyBrandCount = decodedTaxonomyBlock.taxonomy.brandIDs.length;
	decodeResult.stats.taxonomyCategoryCount = decodedTaxonomyBlock.taxonomy.categoryIDs.length;
	decodeResult.stats.taxonomyBrandIndexMode = brandEncodingName(
		decodedTaxonomyBlock.taxonomy.brandIndexEncodingFlag
	);
	return decodeResult;
};

const utf8Decoder = new TextDecoder();

export const readBuildSunixTimeFromHeader = (
	indexBytesInput: Uint8Array | ArrayBuffer
): number => {
	// Fast-path reader used when only the build watermark is needed (no full decode).
	const indexBytes =
		indexBytesInput instanceof Uint8Array ? indexBytesInput : new Uint8Array(indexBytesInput);
	const headerPrefixSize = TEXT_BINARY_MAGIC.length + 1 + 1 + 4 + 4;
	if (indexBytes.length < headerPrefixSize) {
		throw new Error(`index too small bytes=${indexBytes.length}`);
	}
	if (decodeAscii(indexBytes.subarray(0, TEXT_BINARY_MAGIC.length)) !== TEXT_BINARY_MAGIC) {
		throw new Error("invalid text magic");
	}
	if (indexBytes[TEXT_BINARY_MAGIC.length] !== TEXT_BINARY_VERSION) {
		throw new Error(
			`unsupported text version=${indexBytes[TEXT_BINARY_MAGIC.length]} (expected=${TEXT_BINARY_VERSION})`
		);
	}
	const buildSunixOffset = TEXT_BINARY_MAGIC.length + 1 + 1 + 4;
	return readUint32LE(indexBytes, buildSunixOffset);
};

const decodeTextHeaderV2 = (indexBytes: Uint8Array): DecodedTextHeader => {
	// Base header matches backend v1: magic + version + flags + record_count + build_sunix_time + counters + section table size.
	const baseHeaderSize = TEXT_BINARY_MAGIC.length + 1 + 1 + 4 + 4 + 1 + 1 + 2;
	if (indexBytes.length < baseHeaderSize) {
		throw new Error(`index too small bytes=${indexBytes.length}`);
	}
	if (decodeAscii(indexBytes.subarray(0, TEXT_BINARY_MAGIC.length)) !== TEXT_BINARY_MAGIC) {
		throw new Error("invalid text magic");
	}
	if (indexBytes[TEXT_BINARY_MAGIC.length] !== TEXT_BINARY_VERSION) {
		throw new Error(
			`unsupported text version=${indexBytes[TEXT_BINARY_MAGIC.length]} (expected=${TEXT_BINARY_VERSION})`
		);
	}

	let cursor = TEXT_BINARY_MAGIC.length + 1;
	cursor += 1; // Reserved flags.
	const recordCount = readUint32LE(indexBytes, cursor);
	cursor += 4;
	const buildSunixTime = readUint32LE(indexBytes, cursor);
	cursor += 4;
	const dictionaryCount = indexBytes[cursor];
	cursor += 1;
	const sectionCount = indexBytes[cursor];
	cursor += 1;
	const headerSize = readUint16LE(indexBytes, cursor);
	cursor += 2;

	if (recordCount <= 0) {
		throw new Error(`invalid record count=${recordCount}`);
	}
	if (dictionaryCount <= 0 || dictionaryCount > 255) {
		throw new Error(`invalid dictionary count=${dictionaryCount}`);
	}
	if (sectionCount !== 5) {
		throw new Error(`invalid text section count=${sectionCount}`);
	}

	const decodedSectionTable = decodeSectionTable(indexBytes, cursor, headerSize, sectionCount);
	// Keep section ids explicit so unexpected/partial payloads fail early.
	const requiredTextSections = [
		TEXT_SECTION_DICTIONARY,
		TEXT_SECTION_SHAPES,
		TEXT_SECTION_CONTENT,
		TEXT_SECTION_ALIASES,
		TEXT_SECTION_PRODUCT_IDS
	];
	for (const sectionID of requiredTextSections) {
		if (!decodedSectionTable.sectionsByID.has(sectionID)) {
			throw new Error(`missing text section=${sectionID}`);
		}
	}

	if (decodedSectionTable.sectionsByID.get(TEXT_SECTION_DICTIONARY)!.itemCount !== dictionaryCount) {
		throw new Error(
			`text dictionary item count mismatch expected=${dictionaryCount} got=${decodedSectionTable.sectionsByID.get(TEXT_SECTION_DICTIONARY)!.itemCount}`
		);
	}
	if (decodedSectionTable.sectionsByID.get(TEXT_SECTION_SHAPES)!.itemCount !== recordCount) {
		throw new Error(
			`text shapes item count mismatch expected=${recordCount} got=${decodedSectionTable.sectionsByID.get(TEXT_SECTION_SHAPES)!.itemCount}`
		);
	}
	if (
		decodedSectionTable.sectionsByID.get(TEXT_SECTION_CONTENT)!.itemCount !==
		decodedSectionTable.sectionsByID.get(TEXT_SECTION_CONTENT)!.length
	) {
		throw new Error(
			`text content item count mismatch expected=${decodedSectionTable.sectionsByID.get(TEXT_SECTION_CONTENT)!.length} got=${decodedSectionTable.sectionsByID.get(TEXT_SECTION_CONTENT)!.itemCount}`
		);
	}
	if (decodedSectionTable.sectionsByID.get(TEXT_SECTION_PRODUCT_IDS)!.itemCount !== recordCount) {
		throw new Error(
			`text product_ids item count mismatch expected=${recordCount} got=${decodedSectionTable.sectionsByID.get(TEXT_SECTION_PRODUCT_IDS)!.itemCount}`
		);
	}

	return {
		recordCount,
		buildSunixTime,
		dictionaryCount,
		sectionsByID: decodedSectionTable.sectionsByID,
		payloadEnd: decodedSectionTable.maxPayloadEnd
	};
};

const requiredSection = (
	payload: Uint8Array,
	sectionsByID: Map<number, BinarySectionDescriptor>,
	sectionID: number
): Uint8Array => {
	// Centralized section lookup keeps all "missing required section" errors consistent.
	const sectionDescriptor = sectionsByID.get(sectionID);
	if (!sectionDescriptor) {
		throw new Error(`missing required section=${sectionID}`);
	}
	return payload.subarray(sectionDescriptor.offset, sectionDescriptor.offset + sectionDescriptor.length);
};

const decodeSectionTable = (
	payload: Uint8Array,
	tableStart: number,
	headerSize: number,
	sectionCount: number
): { sectionsByID: Map<number, BinarySectionDescriptor>; maxPayloadEnd: number } => {
	// Each entry has fixed-size metadata and CRC32 used for integrity checks.
	const sectionEntrySize = 1 + 4 + 4 + 4 + 4;
	if (sectionCount <= 0) {
		throw new Error(`invalid section count=${sectionCount}`);
	}
	if (headerSize < tableStart) {
		throw new Error(`invalid header size=${headerSize} table_start=${tableStart}`);
	}
	if (tableStart + sectionCount * sectionEntrySize > payload.length) {
		throw new Error("truncated section table");
	}

	const sectionsByID = new Map<number, BinarySectionDescriptor>();
	let cursor = tableStart;
	let maxPayloadEnd = headerSize;

	for (let sectionIndex = 0; sectionIndex < sectionCount; sectionIndex++) {
		// Read one fixed-size section row from the table.
		const sectionID = payload[cursor];
		cursor += 1;
		const offset = readUint32LE(payload, cursor);
		cursor += 4;
		const length = readUint32LE(payload, cursor);
		cursor += 4;
		const itemCount = readUint32LE(payload, cursor);
		cursor += 4;
		const checksum = readUint32LE(payload, cursor);
		cursor += 4;

		if (sectionsByID.has(sectionID)) {
			throw new Error(`duplicated section id=${sectionID}`);
		}
		if (offset < headerSize || offset + length > payload.length) {
			throw new Error(`invalid section bounds id=${sectionID} offset=${offset} length=${length}`);
		}

		const sectionPayload = payload.subarray(offset, offset + length);
		// Verify integrity exactly as backend wrote it (CRC32 per section).
		if (computeCrc32(sectionPayload) !== checksum) {
			throw new Error(`checksum mismatch id=${sectionID}`);
		}

		sectionsByID.set(sectionID, { sectionID, offset, length, itemCount, checksum });
		if (offset + length > maxPayloadEnd) {
			maxPayloadEnd = offset + length;
		}
	}

	if (cursor !== headerSize) {
		// Header size must match table parsing cursor to detect malformed section table metadata.
		throw new Error(`header size mismatch expected=${headerSize} got=${cursor}`);
	}

	return { sectionsByID, maxPayloadEnd };
};

const decodeDictionarySectionRaw = (section: Uint8Array, dictionaryCount: number): string[] => {
	const tokens: string[] = [];
	let cursor = 0;
	for (let tokenIndex = 0; tokenIndex < dictionaryCount; tokenIndex++) {
		if (cursor >= section.length) {
			throw new Error(`dictionary truncated at token=${tokenIndex}`);
		}
		const tokenLength = section[cursor];
		cursor += 1;
		if (tokenLength <= 0 || cursor + tokenLength > section.length) {
			throw new Error(`invalid dictionary token len at token=${tokenIndex}`);
		}
		// Dictionary tokens are stored as [len][utf8 bytes] with no delimiters.
		tokens.push(utf8Decoder.decode(section.subarray(cursor, cursor + tokenLength)));
		cursor += tokenLength;
	}
	if (cursor !== section.length) {
		throw new Error(`dictionary trailing bytes=${section.length - cursor}`);
	}
	return tokens;
};

const decodeAliasSection = (
	section: Uint8Array,
	dictionaryTokenCount: number,
	expectedAliases: number
): Map<string, number> => {
	const decodedAliases = new Map<string, number>();
	let cursor = 0;
	for (let aliasIndex = 0; aliasIndex < expectedAliases; aliasIndex++) {
		if (cursor >= section.length) {
			throw new Error(`alias section truncated at alias=${aliasIndex}`);
		}
		const aliasLength = section[cursor];
		cursor += 1;
		if (aliasLength <= 0 || cursor + aliasLength > section.length) {
			throw new Error(`invalid alias length at alias=${aliasIndex}`);
		}
		const aliasToken = utf8Decoder.decode(section.subarray(cursor, cursor + aliasLength));
		cursor += aliasLength;
		if (cursor >= section.length) {
			throw new Error(`alias section missing dictionary id at alias=${aliasIndex}`);
		}
		// Alias payload points to 1-based dictionary IDs used in content bytes.
		const dictionaryID = section[cursor];
		cursor += 1;
		if (dictionaryID <= 0 || dictionaryID > dictionaryTokenCount) {
			throw new Error(
				`alias dictionary id out of range alias=${aliasToken} id=${dictionaryID} dict=${dictionaryTokenCount}`
			);
		}
		decodedAliases.set(aliasToken, dictionaryID);
	}
	if (cursor !== section.length) {
		throw new Error(`alias section trailing bytes=${section.length - cursor}`);
	}
	return decodedAliases;
};

const decodeProductIDsSection = (section: Uint8Array, expectedRecordCount: number): number[] => {
	const decodedProductIDs: number[] = [];
	let cursor = 0;

	const readWord = (): number => {
		if (cursor + 2 > section.length) {
			throw new Error(`product_ids section truncated at byte=${cursor}`);
		}
		const wordValue = readUint16LE(section, cursor);
		cursor += 2;
		return wordValue;
	};

	for (let recordIndex = 0; recordIndex < expectedRecordCount; recordIndex++) {
		const firstWord = readWord();
		let decodedProductID = 0;
		if (firstWord !== PRODUCT_ID_UINT32_SENTINEL) {
			// Compact path: 16-bit id directly embedded.
			decodedProductID = firstWord;
		} else {
			// Extended path: sentinel + two uint16 words (low/high) reconstruct one uint32 id.
			const lowWord = readWord();
			const highWord = readWord();
			decodedProductID = (lowWord | (highWord << 16)) >>> 0;
		}

		if (decodedProductID <= 0) {
			throw new Error(`invalid decoded product id=${decodedProductID} at record=${recordIndex}`);
		}
		decodedProductIDs.push(decodedProductID);
	}

	if (cursor !== section.length) {
		throw new Error(`product_ids section trailing bytes=${section.length - cursor}`);
	}
	return decodedProductIDs;
};

class BitReader {
	private readonly bytes: Uint8Array;
	private offset = 0;

	constructor(bytes: Uint8Array) {
		this.bytes = bytes;
	}

	// Shapes use a packed bit stream; bits are consumed MSB-first.
	readBit(): number {
		if (this.offset >= this.bytes.length * 8) {
			throw new Error("bitstream exhausted");
		}
		const byteIndex = Math.floor(this.offset / 8);
		const bitIndex = this.offset % 8;
		const bit = (this.bytes[byteIndex] >> (7 - bitIndex)) & 1;
		this.offset += 1;
		return bit;
	}

	readBits(count: number): number {
		let value = 0;
		for (let bitOffset = 0; bitOffset < count; bitOffset++) {
			value = (value << 1) | this.readBit();
		}
		return value >>> 0;
	}
}

const decodeShapeDeltaStream = (
	shapeBytes: Uint8Array,
	recordCount: number
): { shapeValues: number[]; delta8Count: number; delta16Count: number; delta24Count: number } => {
	const bitReader = new BitReader(shapeBytes);
	const shapeValues: number[] = [];
	let previousShapeValue = 0;
	let delta8Count = 0;
	let delta16Count = 0;
	let delta24Count = 0;

	for (let recordIndex = 0; recordIndex < recordCount; recordIndex++) {
		// Flag 0 => 8-bit delta, flag 1 => 16-bit delta or escaped 24-bit delta.
		const flagBit = readWithRecordContext(recordIndex, () => bitReader.readBit(), "read shape flag");
		let deltaValue = 0;
		if (flagBit === 0) {
			deltaValue = readWithRecordContext(recordIndex, () => bitReader.readBits(8), "read 8-bit delta");
			delta8Count += 1;
		} else {
			const decodedDelta16 = readWithRecordContext(
				recordIndex,
				() => bitReader.readBits(16),
				"read 16-bit delta"
			);
			if (decodedDelta16 === 0xffff) {
				// 0xffff is the sentinel that upgrades delta width to 24 bits.
				deltaValue = readWithRecordContext(
					recordIndex,
					() => bitReader.readBits(24),
					"read 24-bit delta"
				);
				delta24Count += 1;
			} else {
				deltaValue = decodedDelta16;
				delta16Count += 1;
			}
		}
		const currentShapeValue = (previousShapeValue + deltaValue) >>> 0;
		// Delta stream is cumulative; each shape value is relative to the previous one.
		shapeValues.push(currentShapeValue);
		previousShapeValue = currentShapeValue;
	}

	return { shapeValues, delta8Count, delta16Count, delta24Count };
};

const decodeShapeValue = (shapeValue: number): number[] => {
	const decodedWordSizes: number[] = [];
	for (let wordIndex = 0; wordIndex < 8; wordIndex++) {
		const shift = 21 - wordIndex * 3;
		// Shape packs up to 8 words, each one encoded in 3 bits as syllable count.
		const wordSyllableCount = (shapeValue >>> shift) & 0x07;
		if (wordSyllableCount === 0) {
			break;
		}
		decodedWordSizes.push(wordSyllableCount);
	}
	return decodedWordSizes;
};

const decodeContentRecords = (
	content: Uint8Array,
	shapes: number[][],
	dictionaryTokens: string[],
	productIDs: number[]
): DecodedRecord[] => {
	if (shapes.length !== productIDs.length) {
		throw new Error(`shape/product id count mismatch shapes=${shapes.length} ids=${productIDs.length}`);
	}

	let cursor = 0;
	const decodedRecords: DecodedRecord[] = [];

	for (let recordIndex = 0; recordIndex < shapes.length; recordIndex++) {
		const shape = shapes[recordIndex];
		const recordContentStart = cursor;
		const decodedWords: string[] = [];
		for (const wordSyllableCount of shape) {
			if (cursor + wordSyllableCount > content.length) {
				throw new Error(`content overflow at record=${recordIndex}`);
			}
			let decodedWord = "";
			for (let syllableIndex = 0; syllableIndex < wordSyllableCount; syllableIndex++) {
				const dictionaryID = content[cursor];
				cursor += 1;
				if (dictionaryID <= 0 || dictionaryID > dictionaryTokens.length) {
					throw new Error(`invalid dictionary id=${dictionaryID} at record=${recordIndex}`);
				}
				decodedWord += dictionaryTokens[dictionaryID - 1];
			}
			decodedWords.push(decodedWord);
		}

		// Taxonomy fields are attached later in a second pass when taxonomy block is present.
		decodedRecords.push({
			recordIndex,
			productID: productIDs[recordIndex],
			shape: [...shape],
			// Preserve raw syllable-token IDs for direct, fast product reconstruction.
			productBytes: content.slice(recordContentStart, cursor),
			text: decodedWords.join(" "),
			brandDictionaryIndex: 0,
			brandID: 0,
			brandName: "",
			categoryDictionaryIndexes: [],
			categoryIDs: [],
			categoryNames: []
		});
	}

	if (cursor !== content.length) {
		throw new Error(`unused content bytes=${content.length - cursor}`);
	}
	return decodedRecords;
};

const decodeTaxonomyBlockV2 = (
	taxonomyBytes: Uint8Array,
	expectedProductCount: number
): { taxonomy: DecodedTaxonomy; payloadEnd: number } => {
	// Base header matches backend taxonomy binary serializer.
	const baseTaxonomyHeaderSize = TAXONOMY_BINARY_MAGIC.length + 1 + 1 + 4 + 1 + 2;
	if (taxonomyBytes.length < baseTaxonomyHeaderSize) {
		throw new Error(`taxonomy section too small bytes=${taxonomyBytes.length}`);
	}
	if (decodeAscii(taxonomyBytes.subarray(0, TAXONOMY_BINARY_MAGIC.length)) !== TAXONOMY_BINARY_MAGIC) {
		throw new Error("invalid taxonomy magic");
	}
	if (taxonomyBytes[TAXONOMY_BINARY_MAGIC.length] !== TAXONOMY_BINARY_VERSION) {
		throw new Error(
			`unsupported taxonomy version=${taxonomyBytes[TAXONOMY_BINARY_MAGIC.length]} (expected=${TAXONOMY_BINARY_VERSION})`
		);
	}

	let cursor = TAXONOMY_BINARY_MAGIC.length + 1;
	const brandEncodingFlag = taxonomyBytes[cursor];
	cursor += 1;
	const sortedProductCount = readUint32LE(taxonomyBytes, cursor);
	cursor += 4;
	const sectionCount = taxonomyBytes[cursor];
	cursor += 1;
	const headerSize = readUint16LE(taxonomyBytes, cursor);
	cursor += 2;

	if (sortedProductCount !== expectedProductCount) {
		throw new Error(
			`taxonomy/text row mismatch taxonomy=${sortedProductCount} text=${expectedProductCount}`
		);
	}
	if (sectionCount !== 7) {
		throw new Error(`invalid taxonomy section count=${sectionCount}`);
	}

	const decodedTaxonomySectionTable = decodeSectionTable(taxonomyBytes, cursor, headerSize, sectionCount);
	// Taxonomy has seven fixed sections and must remain schema-stable.
	const requiredTaxonomySections = [
		TAXONOMY_SECTION_BRAND_IDS,
		TAXONOMY_SECTION_BRAND_NAMES,
		TAXONOMY_SECTION_CATEGORY_IDS,
		TAXONOMY_SECTION_CATEGORY_NAMES,
		TAXONOMY_SECTION_BRAND_INDEXES,
		TAXONOMY_SECTION_CATEGORY_COUNTS,
		TAXONOMY_SECTION_CATEGORY_INDEX
	];
	for (const sectionID of requiredTaxonomySections) {
		if (!decodedTaxonomySectionTable.sectionsByID.has(sectionID)) {
			throw new Error(`missing taxonomy section=${sectionID}`);
		}
	}

	const brandIDsSection = requiredSection(
		taxonomyBytes,
		decodedTaxonomySectionTable.sectionsByID,
		TAXONOMY_SECTION_BRAND_IDS
	);
	const brandNamesSection = requiredSection(
		taxonomyBytes,
		decodedTaxonomySectionTable.sectionsByID,
		TAXONOMY_SECTION_BRAND_NAMES
	);
	const categoryIDsSection = requiredSection(
		taxonomyBytes,
		decodedTaxonomySectionTable.sectionsByID,
		TAXONOMY_SECTION_CATEGORY_IDS
	);
	const categoryNamesSection = requiredSection(
		taxonomyBytes,
		decodedTaxonomySectionTable.sectionsByID,
		TAXONOMY_SECTION_CATEGORY_NAMES
	);
	const brandIndexesSection = requiredSection(
		taxonomyBytes,
		decodedTaxonomySectionTable.sectionsByID,
		TAXONOMY_SECTION_BRAND_INDEXES
	);
	const categoryCountsSection = requiredSection(
		taxonomyBytes,
		decodedTaxonomySectionTable.sectionsByID,
		TAXONOMY_SECTION_CATEGORY_COUNTS
	);
	const categoryIndexesSection = requiredSection(
		taxonomyBytes,
		decodedTaxonomySectionTable.sectionsByID,
		TAXONOMY_SECTION_CATEGORY_INDEX
	);

	const brandIDs = decodeUint16Section(brandIDsSection, "brand_ids");
	const brandNames = decodeStringColumnSection(brandNamesSection, "brand_names");
	if (brandIDs.length !== brandNames.length) {
		throw new Error(`brand dictionary mismatch ids=${brandIDs.length} names=${brandNames.length}`);
	}
	if (
		decodedTaxonomySectionTable.sectionsByID.get(TAXONOMY_SECTION_BRAND_IDS)!.itemCount !== brandIDs.length
	) {
		throw new Error(
			`brand_ids item count mismatch expected=${brandIDs.length} got=${decodedTaxonomySectionTable.sectionsByID.get(TAXONOMY_SECTION_BRAND_IDS)!.itemCount}`
		);
	}
	if (
		decodedTaxonomySectionTable.sectionsByID.get(TAXONOMY_SECTION_BRAND_NAMES)!.itemCount !==
		brandNames.length
	) {
		throw new Error(
			`brand_names item count mismatch expected=${brandNames.length} got=${decodedTaxonomySectionTable.sectionsByID.get(TAXONOMY_SECTION_BRAND_NAMES)!.itemCount}`
		);
	}

	const categoryIDs = decodeUint16Section(categoryIDsSection, "category_ids");
	const categoryNames = decodeStringColumnSection(categoryNamesSection, "category_names");
	if (categoryIDs.length !== categoryNames.length) {
		throw new Error(`category dictionary mismatch ids=${categoryIDs.length} names=${categoryNames.length}`);
	}
	if (
		decodedTaxonomySectionTable.sectionsByID.get(TAXONOMY_SECTION_CATEGORY_IDS)!.itemCount !==
		categoryIDs.length
	) {
		throw new Error(
			`category_ids item count mismatch expected=${categoryIDs.length} got=${decodedTaxonomySectionTable.sectionsByID.get(TAXONOMY_SECTION_CATEGORY_IDS)!.itemCount}`
		);
	}
	if (
		decodedTaxonomySectionTable.sectionsByID.get(TAXONOMY_SECTION_CATEGORY_NAMES)!.itemCount !==
		categoryNames.length
	) {
		throw new Error(
			`category_names item count mismatch expected=${categoryNames.length} got=${decodedTaxonomySectionTable.sectionsByID.get(TAXONOMY_SECTION_CATEGORY_NAMES)!.itemCount}`
		);
	}

	if (
		decodedTaxonomySectionTable.sectionsByID.get(TAXONOMY_SECTION_BRAND_INDEXES)!.itemCount !==
		sortedProductCount
	) {
		throw new Error(
			`brand_indexes item count mismatch expected=${sortedProductCount} got=${decodedTaxonomySectionTable.sectionsByID.get(TAXONOMY_SECTION_BRAND_INDEXES)!.itemCount}`
		);
	}
	const productBrandIndexes = decodeBrandIndexes(
		brandIndexesSection,
		brandEncodingFlag,
		sortedProductCount
	);
	// Validate every brand reference before attaching taxonomy to text records.
	for (let rowIndex = 0; rowIndex < productBrandIndexes.length; rowIndex++) {
		const brandDictionaryIndex = productBrandIndexes[rowIndex];
		if (brandDictionaryIndex < 0 || brandDictionaryIndex >= brandIDs.length) {
			throw new Error(
				`brand index out of range row=${rowIndex} index=${brandDictionaryIndex} dict=${brandIDs.length}`
			);
		}
	}

	if (
		decodedTaxonomySectionTable.sectionsByID.get(TAXONOMY_SECTION_CATEGORY_COUNTS)!.itemCount !==
		sortedProductCount
	) {
		throw new Error(
			`category_count item count mismatch expected=${sortedProductCount} got=${decodedTaxonomySectionTable.sectionsByID.get(TAXONOMY_SECTION_CATEGORY_COUNTS)!.itemCount}`
		);
	}
	const productCategoryCounts = decodePackedCategoryCounts(categoryCountsSection, sortedProductCount);

	// Category index payload is flat; per-record boundaries come from packed 2-bit counts.
	const totalCategoryReferences = productCategoryCounts.reduce((sum, categoryCount) => sum + categoryCount, 0);
	if (totalCategoryReferences !== categoryIndexesSection.length) {
		throw new Error(
			`category indexes payload mismatch expected=${totalCategoryReferences} got=${categoryIndexesSection.length}`
		);
	}
	if (
		decodedTaxonomySectionTable.sectionsByID.get(TAXONOMY_SECTION_CATEGORY_INDEX)!.itemCount !==
		categoryIndexesSection.length
	) {
		throw new Error(
			`category_indexes item count mismatch expected=${categoryIndexesSection.length} got=${decodedTaxonomySectionTable.sectionsByID.get(TAXONOMY_SECTION_CATEGORY_INDEX)!.itemCount}`
		);
	}
	for (let payloadIndex = 0; payloadIndex < categoryIndexesSection.length; payloadIndex++) {
		const categoryDictionaryIndex = categoryIndexesSection[payloadIndex];
		if (categoryDictionaryIndex >= categoryIDs.length) {
			throw new Error(
				`category index out of range at payload index=${payloadIndex} index=${categoryDictionaryIndex} dict=${categoryIDs.length}`
			);
		}
	}

	return {
		taxonomy: {
			brandIDs,
			brandNames,
			categoryIDs,
			categoryNames,
			productBrandDictionaryIndex: productBrandIndexes,
			productCategoryCounts,
			productCategoryIndexes: Array.from(categoryIndexesSection),
			brandIndexEncodingFlag: brandEncodingFlag
		},
		payloadEnd: decodedTaxonomySectionTable.maxPayloadEnd
	};
};

const decodeUint16Section = (section: Uint8Array, sectionName: string): number[] => {
	if (section.length % 2 !== 0) {
		throw new Error(`${sectionName} bytes must be even got=${section.length}`);
	}
	const values: number[] = [];
	for (let cursor = 0; cursor < section.length; cursor += 2) {
		// All uint16 columns are little-endian to mirror backend serializer.
		values.push(readUint16LE(section, cursor));
	}
	return values;
};

const decodeStringColumnSection = (section: Uint8Array, sectionName: string): string[] => {
	const values: string[] = [];
	let cursor = 0;
	while (cursor < section.length) {
		// String columns are compact [len][bytes] tuples.
		const valueLength = section[cursor];
		cursor += 1;
		if (cursor + valueLength > section.length) {
			throw new Error(`${sectionName} truncated string at offset=${cursor - 1}`);
		}
		values.push(utf8Decoder.decode(section.subarray(cursor, cursor + valueLength)));
		cursor += valueLength;
	}
	return values;
};

const decodeBrandIndexes = (
	brandIndexesSection: Uint8Array,
	brandEncodingFlag: number,
	sortedProductCount: number
): number[] => {
	if (brandEncodingFlag === BRAND_INDEX_ENCODING_UINT12) {
		// Packed uint12 path keeps brand indexes dense for small dictionaries.
		return unpackUint12Values(brandIndexesSection, sortedProductCount);
	}
	if (brandEncodingFlag === BRAND_INDEX_ENCODING_UINT16) {
		if (brandIndexesSection.length !== sortedProductCount * 2) {
			throw new Error(
				`uint16 brand index payload mismatch expected=${sortedProductCount * 2} got=${brandIndexesSection.length}`
			);
		}
		const indexes: number[] = [];
		for (let cursor = 0; cursor < brandIndexesSection.length; cursor += 2) {
			// Wider uint16 path is used when brand dictionary does not fit uint12.
			indexes.push(readUint16LE(brandIndexesSection, cursor));
		}
		return indexes;
	}
	throw new Error(`unsupported brand index encoding flag=${brandEncodingFlag}`);
};

const unpackUint12Values = (packed: Uint8Array, expectedValues: number): number[] => {
	const expectedBytes = expectedUint12PackedBytes(expectedValues);
	if (packed.length !== expectedBytes) {
		throw new Error(`uint12 payload mismatch expected=${expectedBytes} got=${packed.length}`);
	}

	const values: number[] = [];
	for (let cursor = 0; cursor < packed.length; cursor += 3) {
		// Two 12-bit values are packed into 3 bytes: [LLLLLLLL][LLLLRRRR][RRRRRRRR].
		const leftValue = (packed[cursor] << 4) | ((packed[cursor + 1] >> 4) & 0x0f);
		values.push(leftValue);
		if (values.length === expectedValues) {
			break;
		}
		const rightValue = ((packed[cursor + 1] & 0x0f) << 8) | packed[cursor + 2];
		values.push(rightValue);
	}
	return values;
};

const decodePackedCategoryCounts = (packedCounts: Uint8Array, sortedProductCount: number): number[] => {
	const expectedCountBytes = Math.floor((sortedProductCount + 3) / 4);
	if (packedCounts.length !== expectedCountBytes) {
		throw new Error(
			`category count payload mismatch expected=${expectedCountBytes} got=${packedCounts.length}`
		);
	}

	const counts: number[] = [];
	for (let productIndex = 0; productIndex < sortedProductCount; productIndex++) {
		// Four 2-bit counters per byte, stored as (count-1) to represent range [1..4].
		const packedByte = packedCounts[Math.floor(productIndex / 4)];
		const shift = 6 - (productIndex % 4) * 2;
		const countMinusOne = (packedByte >> shift) & 0x03;
		counts.push(countMinusOne + 1);
	}
	return counts;
};

const attachTaxonomyToRecords = (records: DecodedRecord[], taxonomy: DecodedTaxonomy): void => {
	if (records.length !== taxonomy.productBrandDictionaryIndex.length) {
		throw new Error(
			`record/brand rows mismatch records=${records.length} brands=${taxonomy.productBrandDictionaryIndex.length}`
		);
	}
	if (records.length !== taxonomy.productCategoryCounts.length) {
		throw new Error(
			`record/category rows mismatch records=${records.length} counts=${taxonomy.productCategoryCounts.length}`
		);
	}

	let categoryPayloadCursor = 0;
	for (let recordIndex = 0; recordIndex < records.length; recordIndex++) {
		// Brand payload is row-aligned: one dictionary index per record.
		const brandDictionaryIndex = taxonomy.productBrandDictionaryIndex[recordIndex];
		records[recordIndex].brandDictionaryIndex = brandDictionaryIndex;
		records[recordIndex].brandID = taxonomy.brandIDs[brandDictionaryIndex];
		records[recordIndex].brandName = taxonomy.brandNames[brandDictionaryIndex];

		const categoryCount = taxonomy.productCategoryCounts[recordIndex];
		if (categoryPayloadCursor + categoryCount > taxonomy.productCategoryIndexes.length) {
			throw new Error(`category payload overflow at record=${recordIndex}`);
		}

		// Category payload is a shared flat stream; consume exactly categoryCount entries for this row.
		const categoryDictionaryIndexes: number[] = [];
		const categoryIDs: number[] = [];
		const categoryNames: string[] = [];
		for (let categoryOffset = 0; categoryOffset < categoryCount; categoryOffset++) {
			const categoryDictionaryIndex = taxonomy.productCategoryIndexes[categoryPayloadCursor];
			categoryPayloadCursor += 1;
			categoryDictionaryIndexes.push(categoryDictionaryIndex);
			categoryIDs.push(taxonomy.categoryIDs[categoryDictionaryIndex]);
			categoryNames.push(taxonomy.categoryNames[categoryDictionaryIndex]);
		}
		records[recordIndex].categoryDictionaryIndexes = categoryDictionaryIndexes;
		records[recordIndex].categoryIDs = categoryIDs;
		records[recordIndex].categoryNames = categoryNames;
	}

	if (categoryPayloadCursor !== taxonomy.productCategoryIndexes.length) {
		throw new Error(
			`unused category payload bytes=${taxonomy.productCategoryIndexes.length - categoryPayloadCursor}`
		);
	}
};
