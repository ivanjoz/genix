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
	
	console.log("sizes:", u8.length, u16.length, u32.length)

	return [b64u8, b64u16, b64u32]
		.map((s) => s.replaceAll("=", "").replaceAll("+", "-").replaceAll("/", "_")).join(".");
};


export const ConcatenateIntsTest = () => {
	
	const values = [3,6,13,142,333,432,543534,32,4444,34242525,3241,23,23,51,3245543,32,3256,43452,2222,11,111]
	
	console.log("concatenateInts",concatenateInts(values))
	
}
