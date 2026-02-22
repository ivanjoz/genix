const DEFAULT_CONNECTOR_WORDS = ["de", "la", "con", "del", "para", "en", "al", "con", "sin"];

const normalizedConnectorSet = (() => {
	const connectorSet = new Set<string>();
	for (const connectorWord of DEFAULT_CONNECTOR_WORDS) {
		const normalizedConnectorWord = normalizeSearchToken(connectorWord).replace(/\s+/g, "");
		if (normalizedConnectorWord) {
			connectorSet.add(normalizedConnectorWord);
		}
	}
	return connectorSet;
})();

export const splitNormalizedWords = (rawText: string): string[] =>
	normalizeSearchToken(rawText)
		.split(" ")
		.map((word) => word.trim())
		.filter((word) => word.length > 0);

export const normalizeAndFilterQueryTokens = (rawText: string): string[] => {
	const normalizedText = normalizeSearchToken(rawText);
	if (!normalizedText) {
		return [];
	}
	const rawTokens = normalizedText
		.split(" ")
		.map((token) => token.trim())
		.filter((token) => token.length > 0);
	const filteredTokens: string[] = [];

	for (let tokenIndex = 0; tokenIndex < rawTokens.length; tokenIndex++) {
		let token = rawTokens[tokenIndex];
		if (normalizedConnectorSet.has(token)) {
			continue;
		}

		const numericPrefixParts = splitNumericPrefixToken(token);
		if (numericPrefixParts) {
			const compactNumericPrefix = compactNumericToken(numericPrefixParts.numberPrefix);
			token = compactNumericPrefix + numericPrefixParts.suffix;
		} else if (isNumericToken(token)) {
			token = compactNumericToken(token);
			if (tokenIndex + 1 < rawTokens.length) {
				const nextToken = rawTokens[tokenIndex + 1];
				if (nextToken.length <= 3 && !isNumericToken(nextToken)) {
					token = token + nextToken;
					tokenIndex++;
				}
			}
		}

		if (utf8Length(token) === 1) {
			continue;
		}
		filteredTokens.push(token);
	}
	return filteredTokens;
};

export const encodeQueryWordToDictionarySyllables = (
	queryWord: string,
	dictionaryTokenIDByNormalizedToken: ReadonlyMap<string, number>,
	aliasDictionaryIDByNormalizedToken: ReadonlyMap<string, number>
): Uint8Array => {
	// Mirror backend encodeWordToken: alias lookup, compact numeric path, then syllable split fallback.
	const encodedSyllableIDs: number[] = [];
	const aliasDictionaryID = aliasDictionaryIDByNormalizedToken.get(queryWord);
	if (aliasDictionaryID) {
		return Uint8Array.from([aliasDictionaryID]);
	}

	const compactNumericSuffix = splitCompactNumericToken(queryWord);
	if (compactNumericSuffix !== null) {
		const numericDictionaryID = dictionaryTokenIDByNormalizedToken.get(queryWord.slice(0, 3));
		if (numericDictionaryID) {
			encodedSyllableIDs.push(numericDictionaryID);
			const suffix = compactNumericSuffix;
			if (suffix) {
				const suffixDictionaryID =
					aliasDictionaryIDByNormalizedToken.get(suffix) ??
					dictionaryTokenIDByNormalizedToken.get(suffix);
				if (suffixDictionaryID) {
					encodedSyllableIDs.push(suffixDictionaryID);
				} else {
					for (const suffixSyllable of splitTokenIntoTwoCharSyllables(suffix)) {
						const suffixSyllableDictionaryID =
							aliasDictionaryIDByNormalizedToken.get(suffixSyllable) ??
							dictionaryTokenIDByNormalizedToken.get(suffixSyllable);
						if (suffixSyllableDictionaryID) {
							encodedSyllableIDs.push(suffixSyllableDictionaryID);
						}
					}
				}
			}
			return Uint8Array.from(encodedSyllableIDs);
		}
	}

	for (const syllableToken of splitTokenIntoTwoCharSyllables(queryWord)) {
		const aliasSyllableDictionaryID = aliasDictionaryIDByNormalizedToken.get(syllableToken);
		if (aliasSyllableDictionaryID) {
			encodedSyllableIDs.push(aliasSyllableDictionaryID);
			continue;
		}

		const directSyllableDictionaryID = dictionaryTokenIDByNormalizedToken.get(syllableToken);
		if (directSyllableDictionaryID) {
			encodedSyllableIDs.push(directSyllableDictionaryID);
			continue;
		}

		for (const rune of syllableToken) {
			const singleRuneDictionaryID =
				aliasDictionaryIDByNormalizedToken.get(rune) ??
				dictionaryTokenIDByNormalizedToken.get(rune);
			if (singleRuneDictionaryID) {
				encodedSyllableIDs.push(singleRuneDictionaryID);
			}
		}
	}
	return Uint8Array.from(encodedSyllableIDs);
};

export function normalizeSearchToken(rawText: string): string {
	// Mirror backend normalizeText() to keep query matching consistent with index build normalization.
	const normalizedRunes: string[] = [];
	for (const lowerRune of rawText.toLowerCase()) {
		let mappedRune = lowerRune;
		switch (lowerRune) {
			case "á":
			case "à":
			case "â":
			case "ä":
			case "ã":
			case "å":
			case "ª":
				mappedRune = "a";
				break;
			case "é":
			case "è":
			case "ê":
			case "ë":
				mappedRune = "e";
				break;
			case "í":
			case "ì":
			case "î":
			case "ï":
				mappedRune = "i";
				break;
			case "ó":
			case "ò":
			case "ô":
			case "ö":
			case "õ":
			case "º":
				mappedRune = "o";
				break;
			case "ú":
			case "ü":
				mappedRune = "u";
				break;
			case "ñ":
				mappedRune = "n";
				break;
		}

		const isAlphaNumeric =
			(mappedRune >= "a" && mappedRune <= "z") ||
			(mappedRune >= "0" && mappedRune <= "9");
		const isWhitespace = /\s/.test(mappedRune);
		if (isAlphaNumeric || isWhitespace) {
			normalizedRunes.push(mappedRune);
		}
	}

	return normalizedRunes.join("").trim().replace(/\s+/g, " ");
}

const splitTokenIntoTwoCharSyllables = (token: string): string[] => {
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
};

const splitNumericPrefixToken = (
	token: string
): { numberPrefix: string; suffix: string } | null => {
	if (!token) {
		return null;
	}
	let splitIndex = 0;
	while (splitIndex < token.length) {
		const currentChar = token[splitIndex];
		if (currentChar < "0" || currentChar > "9") {
			break;
		}
		splitIndex++;
	}
	if (splitIndex === 0 || splitIndex >= token.length) {
		return null;
	}
	for (let suffixIndex = splitIndex; suffixIndex < token.length; suffixIndex++) {
		const suffixChar = token[suffixIndex];
		if (suffixChar < "a" || suffixChar > "z") {
			return null;
		}
	}
	return { numberPrefix: token.slice(0, splitIndex), suffix: token.slice(splitIndex) };
};

const compactNumericToken = (rawNumber: string): string => {
	const parsedNumber = Number.parseInt(rawNumber, 10);
	if (!Number.isFinite(parsedNumber) || parsedNumber <= 0) {
		return "001";
	}
	const compactBucket = ((parsedNumber - 1) % 244) + 1;
	return String(compactBucket).padStart(3, "0");
};

const isNumericToken = (token: string): boolean => /^[0-9]+$/.test(token);

const splitCompactNumericToken = (token: string): string | null => {
	// Expected format mirrors backend: 3 digits + optional [a-z] suffix.
	if (token.length < 3) {
		return null;
	}
	for (let digitIndex = 0; digitIndex < 3; digitIndex++) {
		const numericChar = token[digitIndex];
		if (numericChar < "0" || numericChar > "9") {
			return null;
		}
	}
	for (let suffixIndex = 3; suffixIndex < token.length; suffixIndex++) {
		const suffixChar = token[suffixIndex];
		if (suffixChar < "a" || suffixChar > "z") {
			return null;
		}
	}
	return token.slice(3);
};

const utf8Length = (value: string): number => [...value].length;
