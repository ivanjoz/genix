const defaultOptions = {
	quality: 50,
	qualityAlpha: -1,
	denoiseLevel: 0,
	tileColsLog2: 0,
	tileRowsLog2: 0,
	speed: 6,
	subsample: 1,
	chromaDeltaQ: false,
	sharpness: 0,
	tune: 0,
	enableSharpYUV: false,
	bitDepth: 8,
	lossless: false
};
function initEmscriptenModule(moduleFactory, wasmModule, moduleOptionOverrides = {}) {
	let instantiateWasm;
	if (wasmModule) instantiateWasm = (imports, callback) => {
		const instance = new WebAssembly.Instance(wasmModule, imports);
		callback(instance);
		return instance.exports;
	};
	return moduleFactory({
		noInitialRun: true,
		instantiateWasm,
		...moduleOptionOverrides
	});
}
const threads = () => (async (e) => {
	try {
		return "undefined" != typeof MessageChannel && new MessageChannel().port1.postMessage(new SharedArrayBuffer(1)), WebAssembly.validate(e);
	} catch (e) {
		return !1;
	}
})(new Uint8Array([
	0,
	97,
	115,
	109,
	1,
	0,
	0,
	0,
	1,
	4,
	1,
	96,
	0,
	0,
	3,
	2,
	1,
	0,
	5,
	4,
	1,
	3,
	1,
	1,
	10,
	11,
	1,
	9,
	0,
	65,
	0,
	254,
	16,
	2,
	0,
	26,
	11
]));
let emscriptenModule;
const isRunningInNode = () => typeof process !== "undefined" && process.release && process.release.name === "node";
const isRunningInCloudflareWorker = () => {
	var _a;
	return ((_a = globalThis.caches) === null || _a === void 0 ? void 0 : _a.default) !== void 0;
};
async function init(module, moduleOptionOverrides) {
	let actualModule = module;
	let actualOptions = moduleOptionOverrides;
	if (arguments.length === 1 && !(module instanceof WebAssembly.Module)) {
		actualModule = void 0;
		actualOptions = module;
	}
	if (!isRunningInNode() && !isRunningInCloudflareWorker() && await threads()) {
		emscriptenModule = initEmscriptenModule((await import("./chunks/DAALLAvo.js")).default, actualModule, actualOptions);
		return emscriptenModule;
	}
	emscriptenModule = initEmscriptenModule((await import("./chunks/Bw1us3O2.js")).default, actualModule, actualOptions);
	return emscriptenModule;
}
async function encode(data, options = {}) {
	if (!emscriptenModule) emscriptenModule = init();
	const _options = {
		...defaultOptions,
		...options
	};
	if (_options.bitDepth !== 8 && _options.bitDepth !== 10 && _options.bitDepth !== 12) throw new Error("Invalid bit depth. Supported values are 8, 10, or 12.");
	if (!(data.data instanceof Uint16Array) && _options.bitDepth !== 8) throw new Error("Invalid image data for bit depth. Must use Uint16Array for bit depths greater than 8.");
	if (_options.lossless) {
		if (options.quality !== void 0 && options.quality !== 100) console.warn("AVIF lossless: Quality setting is ignored when lossless is enabled (quality must be 100).");
		if (options.qualityAlpha !== void 0 && options.qualityAlpha !== 100 && options.qualityAlpha !== -1) console.warn("AVIF lossless: QualityAlpha setting is ignored when lossless is enabled (qualityAlpha must be 100 or -1).");
		if (options.subsample !== void 0 && options.subsample !== 3) console.warn("AVIF lossless: Subsample setting is ignored when lossless is enabled (subsample must be 3 for YUV444).");
		_options.quality = 100;
		_options.qualityAlpha = -1;
		_options.subsample = 3;
	}
	const output = (await emscriptenModule).encode(new Uint8Array(data.data.buffer), data.width, data.height, _options);
	if (!output) throw new Error("Encoding error.");
	return output.buffer;
}
console.log("🔧 Image worker loaded and ready");
let nativeAvifSupport = null;
const checkNativeAvifSupport = async () => {
	if (nativeAvifSupport !== null) return nativeAvifSupport;
	try {
		nativeAvifSupport = (await new OffscreenCanvas(1, 1).convertToBlob({ type: "image/avif" })).type === "image/avif";
	} catch (e) {
		nativeAvifSupport = false;
	}
	return nativeAvifSupport;
};
self.onmessage = async (event) => {
	const { id, bitmap: inputBitmap, blob: inputBlob, resolution, useJpeg, useAvif } = event.data;
	console.log("📨 Worker received message:", {
		id,
		hasBitmap: !!inputBitmap,
		hasBlob: !!inputBlob,
		resolution,
		useJpeg,
		useAvif
	});
	try {
		let bitmap;
		if (inputBitmap) {
			console.log("🖼️ Processing bitmap...");
			bitmap = inputBitmap;
		} else if (inputBlob) {
			console.log("🖼️ Processing blob...");
			const blob = inputBlob;
			console.log("🔄 Creating ImageBitmap from blob...");
			bitmap = await createImageBitmap(blob);
			console.log("✅ ImageBitmap created:", bitmap.width, "x", bitmap.height);
		} else {
			console.error("❌ No bitmap or blob provided in message");
			self.postMessage({
				id,
				error: "No bitmap or blob provided"
			});
			return;
		}
		let targetResolution = resolution || 1e3;
		if (targetResolution > 2500) targetResolution = 2500;
		const { width, height } = calculateDimensions(bitmap.width, bitmap.height, targetResolution);
		console.log("📏 Calculated dimensions:", {
			width,
			height
		});
		const canvas = new OffscreenCanvas(width, height);
		const ctx = canvas.getContext("2d");
		ctx.drawImage(bitmap, 0, 0, width, height);
		let blob;
		if (useAvif) {
			console.log("🥑 Attempting AVIF conversion...");
			if (await checkNativeAvifSupport()) {
				console.log("🚀 Using native AVIF support");
				blob = await canvas.convertToBlob({
					type: "image/avif",
					quality: .8
				});
			} else {
				console.log("📦 Using @jsquash/avif fallback");
				const avifBuffer = await encode(ctx.getImageData(0, 0, width, height));
				blob = new Blob([avifBuffer], { type: "image/avif" });
			}
		} else {
			const fileType = useJpeg ? "image/jpeg" : "image/webp";
			console.log(`🎨 Converting to ${fileType}...`);
			blob = await canvas.convertToBlob({
				type: fileType,
				quality: .8
			});
		}
		console.log("🎨 Blob created:", blob.size, "bytes", blob.type);
		const dataUrl = await blobToBase64_(blob);
		console.log("✅ Data URL created, length:", dataUrl.length);
		self.postMessage({
			id,
			dataUrl
		});
		console.log("📤 Response sent to main thread");
		bitmap.close();
	} catch (error) {
		console.error("❌ Error in worker:", error);
		self.postMessage({
			id,
			error: String(error)
		});
	}
};
const calculateDimensions = (originalWidth, originalHeight, targetMP) => {
	const aspectRatio = originalWidth / originalHeight;
	const originalPixels = originalWidth * originalHeight;
	const targetPixels = targetMP * targetMP;
	if (targetPixels > originalPixels) return {
		width: originalWidth,
		height: originalHeight
	};
	let width = Math.sqrt(targetPixels * aspectRatio);
	let height = width / aspectRatio;
	if (width > originalWidth) {
		width = originalWidth;
		height = originalHeight;
	}
	return {
		width: Math.round(width),
		height: Math.round(height)
	};
};
function blobToBase64_(blob) {
	return new Promise((resolve) => {
		const reader = new FileReader();
		reader.onloadend = () => resolve(reader.result);
		reader.readAsDataURL(blob);
	});
}
