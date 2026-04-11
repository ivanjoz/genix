// Categorizes numbers into UInt8, UInt16, and UInt32 arrays, converts each to URL-safe Base64, and joins them with dots (u8.u16.u32). Position is preserved even if a category is empty.
export const concatenateInts = (values: number[], sorted?: boolean): string => {
	const vals = sorted ? [...values].sort((a, b) => a - b) : values;

	const u8: number[] = [];
	const u16: number[] = [];
	const u32: number[] = [];

	for (const v of vals) {
		if (v >= 0 && v <= 255) {
			u8.push(v);
		} else if (v >= 0 && v <= 65535) {
			u16.push(v);
		} else if (v >= 0 && v <= 4294967295) {
			u32.push(v);
		}
	}

	const toBase64 = (arr: Uint8Array | Uint16Array | Uint32Array) => {
		if (arr.length === 0) return "";
		const bytes = new Uint8Array(arr.buffer, arr.byteOffset, arr.byteLength);
		let binary = "";
		for (let i = 0; i < bytes.byteLength; i++) {
			binary += String.fromCharCode(bytes[i]);
		}
		return btoa(binary);
	};

	const b64u8 = toBase64(new Uint8Array(u8));
	const b64u16 = toBase64(new Uint16Array(u16));
	const b64u32 = toBase64(new Uint32Array(u32));

	return [b64u8, b64u16, b64u32]
		.map((s) => s.replaceAll("=", "").replaceAll("+", "-").replaceAll("/", "_")).join(".");
};

export const checksum = (content: string): string => {
	let rollingSeed = 888888

	for (let index = 0; index < content.length; index++) {
		const contentOffset = index % 1000
		const contentCharacter = content[index]
		const charCode = contentCharacter.charCodeAt(0)
		const levelDelta = Math.abs(rollingSeed - charCode + contentOffset) % 10

		if (levelDelta > 6) {
			rollingSeed += ((charCode + contentOffset) * levelDelta) + levelDelta
		} else if (levelDelta > 3) {
			rollingSeed -= ((charCode - contentOffset) * levelDelta) - contentOffset
		} else {
			rollingSeed += Math.abs((charCode - contentOffset) * (levelDelta + 1)) - (levelDelta * (index % 10))
			if (rollingSeed >= 1000000) {
				rollingSeed = Math.abs(rollingSeed) % 100000
			}
		}
	}

	const seedSuffix = String(rollingSeed % 1000)
	for (let index = 0; index < seedSuffix.length; index++) {
		rollingSeed += Math.pow(parseInt(seedSuffix[0]), (6 - index))
	}

	let packedHash = rollingSeed.toString(32).split('').reverse().join('')
	if (packedHash.length > 4) {
		packedHash = packedHash.substring(0, 4)
	} else if (packedHash.length < 4) {
		for (let index = packedHash.length; index < 4; index++) {
			packedHash += String(4 - index)
		}
	}

	return packedHash
}

export const checksumBase64_6 = (content: string): string => {
	let hashAccumulator = 2166136261

	for (let index = 0; index < content.length; index++) {
		hashAccumulator ^= content.charCodeAt(index)
		hashAccumulator = Math.imul(hashAccumulator, 16777619) >>> 0
	}

	// Encode the 32-bit hash as URL-safe base64 and keep a fixed 6-char key for localStorage payloads.
	const hashBytes = new Uint8Array([
		(hashAccumulator >>> 24) & 255,
		(hashAccumulator >>> 16) & 255,
		(hashAccumulator >>> 8) & 255,
		hashAccumulator & 255
	])

	let hashBinary = ""
	for (const hashByte of hashBytes) {
		hashBinary += String.fromCharCode(hashByte)
	}

	return btoa(hashBinary)
		.replaceAll("=", "")
		.replaceAll("+", "-")
		.replaceAll("/", "_")
		.padEnd(6, "0")
		.slice(0, 6)
}

export const base64ToUInt16 = (packedValuesBase64: string): Uint16Array => {
	if (!packedValuesBase64) {
		return new Uint16Array()
	}

	// Match the backend custom URL-safe base64 variant: "-" => "+", "_" => "/", "~" => "=".
	const normalizedBase64 = packedValuesBase64
		.trim()
		.replaceAll("-", "+")
		.replaceAll("_", "/")
		.replaceAll("~", "=")
	const paddedBase64 = normalizedBase64 + "=".repeat((4 - (normalizedBase64.length % 4)) % 4)

	let packedValuesBinary = ""
	try {
		packedValuesBinary = atob(paddedBase64)
	} catch (decodeError) {
		console.warn("[parsers] invalid uint16 base64 payload", decodeError)
		return new Uint16Array()
	}

	const packedValueBytes = Uint8Array.from(packedValuesBinary, (character) => character.charCodeAt(0))

	if (packedValueBytes.length % 2 !== 0) {
		console.warn("[parsers] invalid uint16 base64 payload length", packedValueBytes.length)
		return new Uint16Array()
	}

	const packedUInt16Values = new Uint16Array(packedValueBytes.length / 2)
	for (let byteIndex = 0; byteIndex < packedValueBytes.length; byteIndex += 2) {
		// Read little-endian uint16 values to match the backend encoder.
		packedUInt16Values[byteIndex / 2] = packedValueBytes[byteIndex] | (packedValueBytes[byteIndex + 1] << 8)
	}

	return packedUInt16Values
}
