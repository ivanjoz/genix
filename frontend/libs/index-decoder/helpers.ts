import { BRAND_INDEX_ENCODING_UINT12, BRAND_INDEX_ENCODING_UINT16 } from "./types";
import type { DecodedRecord } from "./types";

// Maps the taxonomy brand index encoding flag to a readable mode name.
export const brandEncodingName = (flag: number): string => {
	if (flag === BRAND_INDEX_ENCODING_UINT12) {
		return "uint12";
	}
	if (flag === BRAND_INDEX_ENCODING_UINT16) {
		return "uint16";
	}
	return "unknown";
};

// Mirrors backend sampling behavior to inspect decoded rows in predictable order.
export const sampleDecodedRecords = (
	records: DecodedRecord[],
	sampleCount: number,
	seed = 0
): DecodedRecord[] => {
	if (sampleCount <= 0 || records.length === 0) {
		return [];
	}
	const safeSampleCount = Math.min(sampleCount, records.length);
	const randomSeed = seed || Date.now();
	const randomGenerator = mulberry32(randomSeed);
	const selectedIndexes = fisherYatesPermutation(records.length, randomGenerator).slice(0, safeSampleCount);
	selectedIndexes.sort((leftIndex, rightIndex) => leftIndex - rightIndex);
	return selectedIndexes.map((recordIndex) => records[recordIndex]);
};

// Preserves record context in low-level decode errors for easier debugging.
export const readWithRecordContext = <T>(
	recordIndex: number,
	reader: () => T,
	errorAction: string
): T => {
	try {
		return reader();
	} catch (error) {
		throw new Error(`${errorAction} at record=${recordIndex}: ${(error as Error).message}`);
	}
};

// Converts binary magic byte ranges to strings for header validation.
export const decodeAscii = (bytes: Uint8Array): string => {
	let decodedValue = "";
	for (let byteIndex = 0; byteIndex < bytes.length; byteIndex++) {
		decodedValue += String.fromCharCode(bytes[byteIndex]);
	}
	return decodedValue;
};

// Reads an unsigned uint16 value in little-endian order.
export const readUint16LE = (bytes: Uint8Array, offset: number): number =>
	(bytes[offset] | (bytes[offset + 1] << 8)) >>> 0;

// Reads an unsigned uint32 value in little-endian order.
export const readUint32LE = (bytes: Uint8Array, offset: number): number =>
	(bytes[offset] |
		(bytes[offset + 1] << 8) |
		(bytes[offset + 2] << 16) |
		(bytes[offset + 3] << 24)) >>>
	0;

// Computes expected byte count for packed uint12 payloads.
export const expectedUint12PackedBytes = (valueCount: number): number =>
	Math.floor(((valueCount + 1) / 2) * 3);

// CRC32 is used by section tables to ensure payload integrity.
export const computeCrc32 = (inputBytes: Uint8Array): number => {
	let crcValue = 0xffffffff;
	for (let byteIndex = 0; byteIndex < inputBytes.length; byteIndex++) {
		const tableIndex = (crcValue ^ inputBytes[byteIndex]) & 0xff;
		crcValue = (crcValue >>> 8) ^ CRC32_TABLE[tableIndex];
	}
	return (crcValue ^ 0xffffffff) >>> 0;
};

// Table is precomputed once to avoid repeated CRC polynomial expansion.
const CRC32_TABLE: Uint32Array = (() => {
	const generatedTable = new Uint32Array(256);
	for (let tableIndex = 0; tableIndex < 256; tableIndex++) {
		let tableValue = tableIndex;
		for (let bitOffset = 0; bitOffset < 8; bitOffset++) {
			tableValue =
				(tableValue & 1) !== 0 ? (0xedb88320 ^ (tableValue >>> 1)) >>> 0 : tableValue >>> 1;
		}
		generatedTable[tableIndex] = tableValue >>> 0;
	}
	return generatedTable;
})();

// Small deterministic PRNG used for sampling helper.
const mulberry32 = (seed: number): (() => number) => {
	let state = seed >>> 0;
	return () => {
		state = (state + 0x6d2b79f5) >>> 0;
		let mixedState = Math.imul(state ^ (state >>> 15), state | 1);
		mixedState ^= mixedState + Math.imul(mixedState ^ (mixedState >>> 7), mixedState | 61);
		return ((mixedState ^ (mixedState >>> 14)) >>> 0) / 4294967296;
	};
};

// Generates an in-place-shuffled index array for uniform random sampling.
const fisherYatesPermutation = (size: number, randomGenerator: () => number): number[] => {
	const indexes = Array.from({ length: size }, (_, index) => index);
	for (let rightIndex = size - 1; rightIndex > 0; rightIndex--) {
		const leftIndex = Math.floor(randomGenerator() * (rightIndex + 1));
		const temporaryValue = indexes[rightIndex];
		indexes[rightIndex] = indexes[leftIndex];
		indexes[leftIndex] = temporaryValue;
	}
	return indexes;
};
