export type CacheConversionType = 'uint8_packed';
export type CacheConversions = Record<string, CacheConversionType>;

const decodeBase64 = (encodedValue: string): Uint8Array => {
	const binaryValue = atob(encodedValue);
	return Uint8Array.from(binaryValue, (character) => character.charCodeAt(0));
};

export const unpackWordBigrams = (encodedValue: string): Uint8Array => {
	const packedBigrams = decodeBase64(encodedValue);
	if (packedBigrams.length === 0) return packedBigrams;

	const wordCount = packedBigrams[0];
	const headerLength = Math.ceil(wordCount / 4);
	const unitsOffset = 1 + headerLength;
	if (wordCount === 0 || unitsOffset > packedBigrams.length) {
		throw new Error(`Invalid packed bigrams header: words=${wordCount} bytes=${packedBigrams.length}`);
	}

	const wordLengths: number[] = [];
	let unitsLength = 0;
	for (let wordIndex = 0; wordIndex < wordCount; wordIndex++) {
		const headerByte = packedBigrams[1 + Math.floor(wordIndex / 4)];
		const shift = 6 - 2 * (wordIndex % 4);
		const wordLength = ((headerByte >> shift) & 0b11) + 1;
		wordLengths.push(wordLength);
		unitsLength += wordLength;
	}
	if (unitsOffset + unitsLength !== packedBigrams.length) {
		throw new Error(`Invalid packed bigrams body: expected=${unitsLength} actual=${packedBigrams.length - unitsOffset}`);
	}

	// Zero separates words without adding a trailing marker.
	const unpackedBigrams = new Uint8Array(unitsLength + wordCount - 1);
	let packedOffset = unitsOffset;
	let unpackedOffset = 0;
	for (let wordIndex = 0; wordIndex < wordLengths.length; wordIndex++) {
		const wordLength = wordLengths[wordIndex];
		unpackedBigrams.set(packedBigrams.subarray(packedOffset, packedOffset + wordLength), unpackedOffset);
		packedOffset += wordLength;
		unpackedOffset += wordLength;
		if (wordIndex < wordLengths.length - 1) unpackedBigrams[unpackedOffset++] = 0;
	}
	return unpackedBigrams;
};

export const applyCacheConversions = (
	response: Record<string, unknown>,
	conversions: CacheConversions | undefined,
	route: string,
): void => {
	if (!conversions) return;

	let convertedFields = 0;
	for (const [responseKey, responseValue] of Object.entries(response)) {
		if (!Array.isArray(responseValue)) continue;
		for (const record of responseValue) {
			if (!record || typeof record !== 'object') continue;
			for (const [fieldName, conversionType] of Object.entries(conversions)) {
				const fieldValue = (record as Record<string, unknown>)[fieldName];
				if (fieldValue === undefined || fieldValue === null) continue;
				if (fieldValue instanceof Uint8Array) continue;
				if (typeof fieldValue !== 'string') {
					throw new Error(`Invalid ${conversionType} field: route=${route} key=${responseKey} field=${fieldName}`);
				}
				(record as Record<string, unknown>)[fieldName] = unpackWordBigrams(fieldValue);
				convertedFields++;
			}
		}
	}
	console.debug(`[DeltaCache] conversions applied route=${route} fields=${convertedFields}`);
};
