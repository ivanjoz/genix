// Product-search shared utilities for binary reads, checksums, sampling, and debug-friendly error context.
import { BRAND_INDEX_ENCODING_UINT12, BRAND_INDEX_ENCODING_UINT16 } from "./types";
import type { DecodedRecord } from "./types";

// Maps the taxonomy brand index encoding flag to a readable mode name.
export const brandEncodingName = (flag: number): string => {
	// Keep mode names aligned with backend constants for readable diagnostics.
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
	// Guard invalid requests and avoid sampling work on empty sets.
	if (sampleCount <= 0 || records.length === 0) {
		return [];
	}
	// Clamp sample size to input length so downstream operations stay bounds-safe.
	const safeSampleCount = Math.min(sampleCount, records.length);
	// Use deterministic seed when provided; fallback keeps ad-hoc debug calls convenient.
	const randomSeed = seed || Date.now();
	const randomGenerator = mulberry32(randomSeed);
	// Shuffle indexes once and then pick the first N entries for uniform sampling.
	const selectedIndexes = fisherYatesPermutation(records.length, randomGenerator).slice(0, safeSampleCount);
	// Sort selected rows so sampled output is stable/predictable for visual inspection.
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
		// Execute caller-provided read logic and preserve its return type.
		return reader();
	} catch (error) {
		// Attach record index/action context so binary decode failures are actionable.
		throw new Error(`${errorAction} at record=${recordIndex}: ${(error as Error).message}`);
	}
};

// Converts binary magic byte ranges to strings for header validation.
export const decodeAscii = (bytes: Uint8Array): string => {
	let decodedValue = "";
	for (let byteIndex = 0; byteIndex < bytes.length; byteIndex++) {
		// Magic/version headers are ASCII-only, so byte-to-char conversion is sufficient.
		decodedValue += String.fromCharCode(bytes[byteIndex]);
	}
	return decodedValue;
};

// Reads an unsigned uint16 value in little-endian order.
export const readUint16LE = (bytes: Uint8Array, offset: number): number =>
	// Combine low/high bytes as little-endian uint16.
	(bytes[offset] | (bytes[offset + 1] << 8)) >>> 0;

// Reads an unsigned uint32 value in little-endian order.
export const readUint32LE = (bytes: Uint8Array, offset: number): number =>
	// Combine 4 bytes as little-endian uint32.
	(bytes[offset] |
		(bytes[offset + 1] << 8) |
		(bytes[offset + 2] << 16) |
		(bytes[offset + 3] << 24)) >>>
	0;

// Computes expected byte count for packed uint12 payloads.
export const expectedUint12PackedBytes = (valueCount: number): number =>
	// Two uint12 values are packed into 3 bytes; odd counts reserve half-pairs.
	Math.floor(((valueCount + 1) / 2) * 3);

// CRC32 is used by section tables to ensure payload integrity.
export const computeCrc32 = (inputBytes: Uint8Array): number => {
	// Initialize with all bits set, matching standard CRC32 implementation.
	let crcValue = 0xffffffff;
	for (let byteIndex = 0; byteIndex < inputBytes.length; byteIndex++) {
		// Fold each byte through lookup-table accelerated polynomial reduction.
		const tableIndex = (crcValue ^ inputBytes[byteIndex]) & 0xff;
		crcValue = (crcValue >>> 8) ^ CRC32_TABLE[tableIndex];
	}
	// Final XOR completes CRC32 post-processing.
	return (crcValue ^ 0xffffffff) >>> 0;
};

// Table is precomputed once to avoid repeated CRC polynomial expansion.
const CRC32_TABLE: Uint32Array = (() => {
	// Precompute all 256 8-bit transitions once for fast checksum loops.
	const generatedTable = new Uint32Array(256);
	for (let tableIndex = 0; tableIndex < 256; tableIndex++) {
		let tableValue = tableIndex;
		for (let bitOffset = 0; bitOffset < 8; bitOffset++) {
			// Apply reflected CRC32 polynomial one bit at a time.
			tableValue =
				(tableValue & 1) !== 0 ? (0xedb88320 ^ (tableValue >>> 1)) >>> 0 : tableValue >>> 1;
		}
		generatedTable[tableIndex] = tableValue >>> 0;
	}
	return generatedTable;
})();

// Small deterministic PRNG used for sampling helper.
const mulberry32 = (seed: number): (() => number) => {
	// Force seed into uint32 range for deterministic cross-runtime behavior.
	let state = seed >>> 0;
	return () => {
		// Advance state and scramble bits (Mulberry32 core).
		state = (state + 0x6d2b79f5) >>> 0;
		let mixedState = Math.imul(state ^ (state >>> 15), state | 1);
		mixedState ^= mixedState + Math.imul(mixedState ^ (mixedState >>> 7), mixedState | 61);
		// Normalize to [0, 1) for consumer APIs.
		return ((mixedState ^ (mixedState >>> 14)) >>> 0) / 4294967296;
	};
};

// Generates an in-place-shuffled index array for uniform random sampling.
const fisherYatesPermutation = (size: number, randomGenerator: () => number): number[] => {
	// Build identity index list and shuffle in place for O(n) unbiased permutation.
	const indexes = Array.from({ length: size }, (_, index) => index);
	for (let rightIndex = size - 1; rightIndex > 0; rightIndex--) {
		// Swap current tail with a random position in [0, rightIndex].
		const leftIndex = Math.floor(randomGenerator() * (rightIndex + 1));
		const temporaryValue = indexes[rightIndex];
		indexes[rightIndex] = indexes[leftIndex];
		indexes[leftIndex] = temporaryValue;
	}
	return indexes;
};
