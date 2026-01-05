import { $ as proxy, H as update_version, St as __exportAll, at as state, bt as false_default, et as increment, ft as label, it as source, pt as tag, rt as set, z as get } from "./CTB4HzdN.js";
import { t as goto } from "./D25ese0K.js";
import { t as browser } from "./BemFyt0E.js";
import { a as throttle$1, t as Notify } from "./DwKmqXYU.js";
const makeRamdomString = (length) => {
	let str = "";
	while (str.length < length) str += (Math.random() + 1).toString(36).substring(2);
	return str.substring(0, length);
};
const decrypt = async (encryptedString, key) => {
	key = key.substring(0, 32);
	const encryptedData = Uint8Array.from(atob(encryptedString), (c) => c.charCodeAt(0));
	console.log("encrypted len::", encryptedData.length);
	const keyBuffer = await crypto.subtle.importKey("raw", new TextEncoder().encode(key), { name: "AES-GCM" }, false, ["decrypt"]);
	if (encryptedData.length < 12) throw new Error("Invalid encrypted data");
	const nonce = encryptedData.slice(0, 12);
	console.log("nonce:: ", new TextDecoder().decode(nonce));
	const ciphertext = encryptedData.slice(12);
	console.log("desencriptando:: ", nonce.length, key.length, ciphertext.length);
	let decryptedData;
	try {
		decryptedData = await crypto.subtle.decrypt({
			name: "AES-GCM",
			iv: nonce
		}, keyBuffer, ciphertext);
	} catch (error) {
		console.log("Error desencriptando:: ", error);
		return "";
	}
	console.log("decripted data:: ", decryptedData);
	return new TextDecoder().decode(decryptedData);
};
const formatN = (x, decimal, fixedLen, charF) => {
	decimal = decimal || 0;
	if (typeof x !== "number") return x ? "-" : "";
	if (decimal === -1) {
		if (x < 1) x = Math.round(x * 1e4) / 1e4;
		else if (x < 10) x = Math.round(x * 1e3) / 1e3;
		else if (x >= 10) x = Math.round(x * 100) / 100;
	}
	let xString;
	if (typeof decimal === "number" && decimal >= 0) if (decimal === 0) xString = Math.round(x).toString();
	else {
		const pow = Math.pow(10, decimal);
		xString = (Math.round(x * pow) / pow).toFixed(decimal);
	}
	else xString = x.toString();
	if (x >= 100) xString = xString.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
	if (fixedLen) {
		charF = charF || " ";
		while (xString.length < fixedLen) xString = charF + xString;
	}
	return xString;
};
function bind(fn, thisArg) {
	return function wrap() {
		return fn.apply(thisArg, arguments);
	};
}
var { toString } = Object.prototype;
var { getPrototypeOf } = Object;
var { iterator, toStringTag } = Symbol;
var kindOf = ((cache) => (thing) => {
	const str = toString.call(thing);
	return cache[str] || (cache[str] = str.slice(8, -1).toLowerCase());
})(Object.create(null));
var kindOfTest = (type) => {
	type = type.toLowerCase();
	return (thing) => kindOf(thing) === type;
};
var typeOfTest = (type) => (thing) => typeof thing === type;
var { isArray } = Array;
var isUndefined = typeOfTest("undefined");
function isBuffer(val) {
	return val !== null && !isUndefined(val) && val.constructor !== null && !isUndefined(val.constructor) && isFunction$1(val.constructor.isBuffer) && val.constructor.isBuffer(val);
}
var isArrayBuffer = kindOfTest("ArrayBuffer");
function isArrayBufferView(val) {
	let result;
	if (typeof ArrayBuffer !== "undefined" && ArrayBuffer.isView) result = ArrayBuffer.isView(val);
	else result = val && val.buffer && isArrayBuffer(val.buffer);
	return result;
}
var isString = typeOfTest("string");
var isFunction$1 = typeOfTest("function");
var isNumber = typeOfTest("number");
var isObject = (thing) => thing !== null && typeof thing === "object";
var isBoolean = (thing) => thing === true || thing === false;
var isPlainObject = (val) => {
	if (kindOf(val) !== "object") return false;
	const prototype$2 = getPrototypeOf(val);
	return (prototype$2 === null || prototype$2 === Object.prototype || Object.getPrototypeOf(prototype$2) === null) && !(toStringTag in val) && !(iterator in val);
};
var isEmptyObject = (val) => {
	if (!isObject(val) || isBuffer(val)) return false;
	try {
		return Object.keys(val).length === 0 && Object.getPrototypeOf(val) === Object.prototype;
	} catch (e) {
		return false;
	}
};
var isDate = kindOfTest("Date");
var isFile = kindOfTest("File");
var isBlob = kindOfTest("Blob");
var isFileList = kindOfTest("FileList");
var isStream = (val) => isObject(val) && isFunction$1(val.pipe);
var isFormData = (thing) => {
	let kind;
	return thing && (typeof FormData === "function" && thing instanceof FormData || isFunction$1(thing.append) && ((kind = kindOf(thing)) === "formdata" || kind === "object" && isFunction$1(thing.toString) && thing.toString() === "[object FormData]"));
};
var isURLSearchParams = kindOfTest("URLSearchParams");
var [isReadableStream, isRequest, isResponse, isHeaders] = [
	"ReadableStream",
	"Request",
	"Response",
	"Headers"
].map(kindOfTest);
var trim = (str) => str.trim ? str.trim() : str.replace(/^[\s\uFEFF\xA0]+|[\s\uFEFF\xA0]+$/g, "");
function forEach(obj, fn, { allOwnKeys = false } = {}) {
	if (obj === null || typeof obj === "undefined") return;
	let i;
	let l;
	if (typeof obj !== "object") obj = [obj];
	if (isArray(obj)) for (i = 0, l = obj.length; i < l; i++) fn.call(null, obj[i], i, obj);
	else {
		if (isBuffer(obj)) return;
		const keys = allOwnKeys ? Object.getOwnPropertyNames(obj) : Object.keys(obj);
		const len = keys.length;
		let key;
		for (i = 0; i < len; i++) {
			key = keys[i];
			fn.call(null, obj[key], key, obj);
		}
	}
}
function findKey(obj, key) {
	if (isBuffer(obj)) return null;
	key = key.toLowerCase();
	const keys = Object.keys(obj);
	let i = keys.length;
	let _key;
	while (i-- > 0) {
		_key = keys[i];
		if (key === _key.toLowerCase()) return _key;
	}
	return null;
}
var _global = (() => {
	if (typeof globalThis !== "undefined") return globalThis;
	return typeof self !== "undefined" ? self : typeof window !== "undefined" ? window : global;
})();
var isContextDefined = (context) => !isUndefined(context) && context !== _global;
function merge() {
	const { caseless, skipUndefined } = isContextDefined(this) && this || {};
	const result = {};
	const assignValue = (val, key) => {
		const targetKey = caseless && findKey(result, key) || key;
		if (isPlainObject(result[targetKey]) && isPlainObject(val)) result[targetKey] = merge(result[targetKey], val);
		else if (isPlainObject(val)) result[targetKey] = merge({}, val);
		else if (isArray(val)) result[targetKey] = val.slice();
		else if (!skipUndefined || !isUndefined(val)) result[targetKey] = val;
	};
	for (let i = 0, l = arguments.length; i < l; i++) arguments[i] && forEach(arguments[i], assignValue);
	return result;
}
var extend = (a, b, thisArg, { allOwnKeys } = {}) => {
	forEach(b, (val, key) => {
		if (thisArg && isFunction$1(val)) a[key] = bind(val, thisArg);
		else a[key] = val;
	}, { allOwnKeys });
	return a;
};
var stripBOM = (content) => {
	if (content.charCodeAt(0) === 65279) content = content.slice(1);
	return content;
};
var inherits = (constructor, superConstructor, props, descriptors$1) => {
	constructor.prototype = Object.create(superConstructor.prototype, descriptors$1);
	constructor.prototype.constructor = constructor;
	Object.defineProperty(constructor, "super", { value: superConstructor.prototype });
	props && Object.assign(constructor.prototype, props);
};
var toFlatObject = (sourceObj, destObj, filter, propFilter) => {
	let props;
	let i;
	let prop;
	const merged = {};
	destObj = destObj || {};
	if (sourceObj == null) return destObj;
	do {
		props = Object.getOwnPropertyNames(sourceObj);
		i = props.length;
		while (i-- > 0) {
			prop = props[i];
			if ((!propFilter || propFilter(prop, sourceObj, destObj)) && !merged[prop]) {
				destObj[prop] = sourceObj[prop];
				merged[prop] = true;
			}
		}
		sourceObj = filter !== false && getPrototypeOf(sourceObj);
	} while (sourceObj && (!filter || filter(sourceObj, destObj)) && sourceObj !== Object.prototype);
	return destObj;
};
var endsWith = (str, searchString, position) => {
	str = String(str);
	if (position === void 0 || position > str.length) position = str.length;
	position -= searchString.length;
	const lastIndex = str.indexOf(searchString, position);
	return lastIndex !== -1 && lastIndex === position;
};
var toArray = (thing) => {
	if (!thing) return null;
	if (isArray(thing)) return thing;
	let i = thing.length;
	if (!isNumber(i)) return null;
	const arr = new Array(i);
	while (i-- > 0) arr[i] = thing[i];
	return arr;
};
var isTypedArray = ((TypedArray) => {
	return (thing) => {
		return TypedArray && thing instanceof TypedArray;
	};
})(typeof Uint8Array !== "undefined" && getPrototypeOf(Uint8Array));
var forEachEntry = (obj, fn) => {
	const _iterator = (obj && obj[iterator]).call(obj);
	let result;
	while ((result = _iterator.next()) && !result.done) {
		const pair = result.value;
		fn.call(obj, pair[0], pair[1]);
	}
};
var matchAll = (regExp, str) => {
	let matches;
	const arr = [];
	while ((matches = regExp.exec(str)) !== null) arr.push(matches);
	return arr;
};
var isHTMLForm = kindOfTest("HTMLFormElement");
var toCamelCase = (str) => {
	return str.toLowerCase().replace(/[-_\s]([a-z\d])(\w*)/g, function replacer(m, p1, p2) {
		return p1.toUpperCase() + p2;
	});
};
var hasOwnProperty = (({ hasOwnProperty: hasOwnProperty$1 }) => (obj, prop) => hasOwnProperty$1.call(obj, prop))(Object.prototype);
var isRegExp = kindOfTest("RegExp");
var reduceDescriptors = (obj, reducer) => {
	const descriptors$1 = Object.getOwnPropertyDescriptors(obj);
	const reducedDescriptors = {};
	forEach(descriptors$1, (descriptor, name) => {
		let ret;
		if ((ret = reducer(descriptor, name, obj)) !== false) reducedDescriptors[name] = ret || descriptor;
	});
	Object.defineProperties(obj, reducedDescriptors);
};
var freezeMethods = (obj) => {
	reduceDescriptors(obj, (descriptor, name) => {
		if (isFunction$1(obj) && [
			"arguments",
			"caller",
			"callee"
		].indexOf(name) !== -1) return false;
		const value = obj[name];
		if (!isFunction$1(value)) return;
		descriptor.enumerable = false;
		if ("writable" in descriptor) {
			descriptor.writable = false;
			return;
		}
		if (!descriptor.set) descriptor.set = () => {
			throw Error("Can not rewrite read-only method '" + name + "'");
		};
	});
};
var toObjectSet = (arrayOrString, delimiter) => {
	const obj = {};
	const define = (arr) => {
		arr.forEach((value) => {
			obj[value] = true;
		});
	};
	isArray(arrayOrString) ? define(arrayOrString) : define(String(arrayOrString).split(delimiter));
	return obj;
};
var noop = () => {};
var toFiniteNumber = (value, defaultValue) => {
	return value != null && Number.isFinite(value = +value) ? value : defaultValue;
};
function isSpecCompliantForm(thing) {
	return !!(thing && isFunction$1(thing.append) && thing[toStringTag] === "FormData" && thing[iterator]);
}
var toJSONObject = (obj) => {
	const stack = new Array(10);
	const visit = (source$1, i) => {
		if (isObject(source$1)) {
			if (stack.indexOf(source$1) >= 0) return;
			if (isBuffer(source$1)) return source$1;
			if (!("toJSON" in source$1)) {
				stack[i] = source$1;
				const target = isArray(source$1) ? [] : {};
				forEach(source$1, (value, key) => {
					const reducedValue = visit(value, i + 1);
					!isUndefined(reducedValue) && (target[key] = reducedValue);
				});
				stack[i] = void 0;
				return target;
			}
		}
		return source$1;
	};
	return visit(obj, 0);
};
var isAsyncFn = kindOfTest("AsyncFunction");
var isThenable = (thing) => thing && (isObject(thing) || isFunction$1(thing)) && isFunction$1(thing.then) && isFunction$1(thing.catch);
var _setImmediate = ((setImmediateSupported, postMessageSupported) => {
	if (setImmediateSupported) return setImmediate;
	return postMessageSupported ? ((token, callbacks) => {
		_global.addEventListener("message", ({ source: source$1, data }) => {
			if (source$1 === _global && data === token) callbacks.length && callbacks.shift()();
		}, false);
		return (cb) => {
			callbacks.push(cb);
			_global.postMessage(token, "*");
		};
	})(`axios@${Math.random()}`, []) : (cb) => setTimeout(cb);
})(typeof setImmediate === "function", isFunction$1(_global.postMessage));
var asap = typeof queueMicrotask !== "undefined" ? queueMicrotask.bind(_global) : typeof process !== "undefined" && process.nextTick || _setImmediate;
var isIterable = (thing) => thing != null && isFunction$1(thing[iterator]);
var utils_default = {
	isArray,
	isArrayBuffer,
	isBuffer,
	isFormData,
	isArrayBufferView,
	isString,
	isNumber,
	isBoolean,
	isObject,
	isPlainObject,
	isEmptyObject,
	isReadableStream,
	isRequest,
	isResponse,
	isHeaders,
	isUndefined,
	isDate,
	isFile,
	isBlob,
	isRegExp,
	isFunction: isFunction$1,
	isStream,
	isURLSearchParams,
	isTypedArray,
	isFileList,
	forEach,
	merge,
	extend,
	trim,
	stripBOM,
	inherits,
	toFlatObject,
	kindOf,
	kindOfTest,
	endsWith,
	toArray,
	forEachEntry,
	matchAll,
	isHTMLForm,
	hasOwnProperty,
	hasOwnProp: hasOwnProperty,
	reduceDescriptors,
	freezeMethods,
	toObjectSet,
	toCamelCase,
	noop,
	toFiniteNumber,
	findKey,
	global: _global,
	isContextDefined,
	isSpecCompliantForm,
	toJSONObject,
	isAsyncFn,
	isThenable,
	setImmediate: _setImmediate,
	asap,
	isIterable
};
function AxiosError(message, code, config, request, response) {
	Error.call(this);
	if (Error.captureStackTrace) Error.captureStackTrace(this, this.constructor);
	else this.stack = (/* @__PURE__ */ new Error()).stack;
	this.message = message;
	this.name = "AxiosError";
	code && (this.code = code);
	config && (this.config = config);
	request && (this.request = request);
	if (response) {
		this.response = response;
		this.status = response.status ? response.status : null;
	}
}
utils_default.inherits(AxiosError, Error, { toJSON: function toJSON() {
	return {
		message: this.message,
		name: this.name,
		description: this.description,
		number: this.number,
		fileName: this.fileName,
		lineNumber: this.lineNumber,
		columnNumber: this.columnNumber,
		stack: this.stack,
		config: utils_default.toJSONObject(this.config),
		code: this.code,
		status: this.status
	};
} });
var prototype$1 = AxiosError.prototype;
var descriptors = {};
[
	"ERR_BAD_OPTION_VALUE",
	"ERR_BAD_OPTION",
	"ECONNABORTED",
	"ETIMEDOUT",
	"ERR_NETWORK",
	"ERR_FR_TOO_MANY_REDIRECTS",
	"ERR_DEPRECATED",
	"ERR_BAD_RESPONSE",
	"ERR_BAD_REQUEST",
	"ERR_CANCELED",
	"ERR_NOT_SUPPORT",
	"ERR_INVALID_URL"
].forEach((code) => {
	descriptors[code] = { value: code };
});
Object.defineProperties(AxiosError, descriptors);
Object.defineProperty(prototype$1, "isAxiosError", { value: true });
AxiosError.from = (error, code, config, request, response, customProps) => {
	const axiosError = Object.create(prototype$1);
	utils_default.toFlatObject(error, axiosError, function filter(obj) {
		return obj !== Error.prototype;
	}, (prop) => {
		return prop !== "isAxiosError";
	});
	const msg = error && error.message ? error.message : "Error";
	const errCode = code == null && error ? error.code : code;
	AxiosError.call(axiosError, msg, errCode, config, request, response);
	if (error && axiosError.cause == null) Object.defineProperty(axiosError, "cause", {
		value: error,
		configurable: true
	});
	axiosError.name = error && error.name || "Error";
	customProps && Object.assign(axiosError, customProps);
	return axiosError;
};
var AxiosError_default = AxiosError;
function isVisitable(thing) {
	return utils_default.isPlainObject(thing) || utils_default.isArray(thing);
}
function removeBrackets(key) {
	return utils_default.endsWith(key, "[]") ? key.slice(0, -2) : key;
}
function renderKey(path, key, dots) {
	if (!path) return key;
	return path.concat(key).map(function each(token, i) {
		token = removeBrackets(token);
		return !dots && i ? "[" + token + "]" : token;
	}).join(dots ? "." : "");
}
function isFlatArray(arr) {
	return utils_default.isArray(arr) && !arr.some(isVisitable);
}
var predicates = utils_default.toFlatObject(utils_default, {}, null, function filter(prop) {
	return /^is[A-Z]/.test(prop);
});
function toFormData(obj, formData, options) {
	if (!utils_default.isObject(obj)) throw new TypeError("target must be an object");
	formData = formData || new FormData();
	options = utils_default.toFlatObject(options, {
		metaTokens: true,
		dots: false,
		indexes: false
	}, false, function defined(option, source$1) {
		return !utils_default.isUndefined(source$1[option]);
	});
	const metaTokens = options.metaTokens;
	const visitor = options.visitor || defaultVisitor;
	const dots = options.dots;
	const indexes = options.indexes;
	const useBlob = (options.Blob || typeof Blob !== "undefined" && Blob) && utils_default.isSpecCompliantForm(formData);
	if (!utils_default.isFunction(visitor)) throw new TypeError("visitor must be a function");
	function convertValue(value) {
		if (value === null) return "";
		if (utils_default.isDate(value)) return value.toISOString();
		if (utils_default.isBoolean(value)) return value.toString();
		if (!useBlob && utils_default.isBlob(value)) throw new AxiosError_default("Blob is not supported. Use a Buffer instead.");
		if (utils_default.isArrayBuffer(value) || utils_default.isTypedArray(value)) return useBlob && typeof Blob === "function" ? new Blob([value]) : Buffer.from(value);
		return value;
	}
	function defaultVisitor(value, key, path) {
		let arr = value;
		if (value && !path && typeof value === "object") {
			if (utils_default.endsWith(key, "{}")) {
				key = metaTokens ? key : key.slice(0, -2);
				value = JSON.stringify(value);
			} else if (utils_default.isArray(value) && isFlatArray(value) || (utils_default.isFileList(value) || utils_default.endsWith(key, "[]")) && (arr = utils_default.toArray(value))) {
				key = removeBrackets(key);
				arr.forEach(function each(el, index) {
					!(utils_default.isUndefined(el) || el === null) && formData.append(indexes === true ? renderKey([key], index, dots) : indexes === null ? key : key + "[]", convertValue(el));
				});
				return false;
			}
		}
		if (isVisitable(value)) return true;
		formData.append(renderKey(path, key, dots), convertValue(value));
		return false;
	}
	const stack = [];
	const exposedHelpers = Object.assign(predicates, {
		defaultVisitor,
		convertValue,
		isVisitable
	});
	function build(value, path) {
		if (utils_default.isUndefined(value)) return;
		if (stack.indexOf(value) !== -1) throw Error("Circular reference detected in " + path.join("."));
		stack.push(value);
		utils_default.forEach(value, function each(el, key) {
			if ((!(utils_default.isUndefined(el) || el === null) && visitor.call(formData, el, utils_default.isString(key) ? key.trim() : key, path, exposedHelpers)) === true) build(el, path ? path.concat(key) : [key]);
		});
		stack.pop();
	}
	if (!utils_default.isObject(obj)) throw new TypeError("data must be an object");
	build(obj);
	return formData;
}
var toFormData_default = toFormData;
function encode$1(str) {
	const charMap = {
		"!": "%21",
		"'": "%27",
		"(": "%28",
		")": "%29",
		"~": "%7E",
		"%20": "+",
		"%00": "\0"
	};
	return encodeURIComponent(str).replace(/[!'()~]|%20|%00/g, function replacer(match) {
		return charMap[match];
	});
}
function AxiosURLSearchParams(params, options) {
	this._pairs = [];
	params && toFormData_default(params, this, options);
}
var prototype = AxiosURLSearchParams.prototype;
prototype.append = function append(name, value) {
	this._pairs.push([name, value]);
};
prototype.toString = function toString$1(encoder) {
	const _encode = encoder ? function(value) {
		return encoder.call(this, value, encode$1);
	} : encode$1;
	return this._pairs.map(function each(pair) {
		return _encode(pair[0]) + "=" + _encode(pair[1]);
	}, "").join("&");
};
var AxiosURLSearchParams_default = AxiosURLSearchParams;
function encode(val) {
	return encodeURIComponent(val).replace(/%3A/gi, ":").replace(/%24/g, "$").replace(/%2C/gi, ",").replace(/%20/g, "+");
}
function buildURL(url, params, options) {
	if (!params) return url;
	const _encode = options && options.encode || encode;
	if (utils_default.isFunction(options)) options = { serialize: options };
	const serializeFn = options && options.serialize;
	let serializedParams;
	if (serializeFn) serializedParams = serializeFn(params, options);
	else serializedParams = utils_default.isURLSearchParams(params) ? params.toString() : new AxiosURLSearchParams_default(params, options).toString(_encode);
	if (serializedParams) {
		const hashmarkIndex = url.indexOf("#");
		if (hashmarkIndex !== -1) url = url.slice(0, hashmarkIndex);
		url += (url.indexOf("?") === -1 ? "?" : "&") + serializedParams;
	}
	return url;
}
var InterceptorManager = class {
	constructor() {
		this.handlers = [];
	}
	use(fulfilled, rejected, options) {
		this.handlers.push({
			fulfilled,
			rejected,
			synchronous: options ? options.synchronous : false,
			runWhen: options ? options.runWhen : null
		});
		return this.handlers.length - 1;
	}
	eject(id) {
		if (this.handlers[id]) this.handlers[id] = null;
	}
	clear() {
		if (this.handlers) this.handlers = [];
	}
	forEach(fn) {
		utils_default.forEach(this.handlers, function forEachHandler(h) {
			if (h !== null) fn(h);
		});
	}
};
var InterceptorManager_default = InterceptorManager;
var transitional_default = {
	silentJSONParsing: true,
	forcedJSONParsing: true,
	clarifyTimeoutError: false
};
var browser_default = {
	isBrowser: true,
	classes: {
		URLSearchParams: typeof URLSearchParams !== "undefined" ? URLSearchParams : AxiosURLSearchParams_default,
		FormData: typeof FormData !== "undefined" ? FormData : null,
		Blob: typeof Blob !== "undefined" ? Blob : null
	},
	protocols: [
		"http",
		"https",
		"file",
		"blob",
		"url",
		"data"
	]
};
var utils_exports = /* @__PURE__ */ __exportAll({
	hasBrowserEnv: () => hasBrowserEnv,
	hasStandardBrowserEnv: () => hasStandardBrowserEnv,
	hasStandardBrowserWebWorkerEnv: () => hasStandardBrowserWebWorkerEnv,
	navigator: () => _navigator,
	origin: () => origin
}, 1);
var hasBrowserEnv = typeof window !== "undefined" && typeof document !== "undefined";
var _navigator = typeof navigator === "object" && navigator || void 0;
var hasStandardBrowserEnv = hasBrowserEnv && (!_navigator || [
	"ReactNative",
	"NativeScript",
	"NS"
].indexOf(_navigator.product) < 0);
var hasStandardBrowserWebWorkerEnv = typeof WorkerGlobalScope !== "undefined" && self instanceof WorkerGlobalScope && typeof self.importScripts === "function";
var origin = hasBrowserEnv && window.location.href || "http://localhost";
var platform_default = {
	...utils_exports,
	...browser_default
};
function toURLEncodedForm(data, options) {
	return toFormData_default(data, new platform_default.classes.URLSearchParams(), {
		visitor: function(value, key, path, helpers) {
			if (platform_default.isNode && utils_default.isBuffer(value)) {
				this.append(key, value.toString("base64"));
				return false;
			}
			return helpers.defaultVisitor.apply(this, arguments);
		},
		...options
	});
}
function parsePropPath(name) {
	return utils_default.matchAll(/\w+|\[(\w*)]/g, name).map((match) => {
		return match[0] === "[]" ? "" : match[1] || match[0];
	});
}
function arrayToObject(arr) {
	const obj = {};
	const keys = Object.keys(arr);
	let i;
	const len = keys.length;
	let key;
	for (i = 0; i < len; i++) {
		key = keys[i];
		obj[key] = arr[key];
	}
	return obj;
}
function formDataToJSON(formData) {
	function buildPath(path, value, target, index) {
		let name = path[index++];
		if (name === "__proto__") return true;
		const isNumericKey = Number.isFinite(+name);
		const isLast = index >= path.length;
		name = !name && utils_default.isArray(target) ? target.length : name;
		if (isLast) {
			if (utils_default.hasOwnProp(target, name)) target[name] = [target[name], value];
			else target[name] = value;
			return !isNumericKey;
		}
		if (!target[name] || !utils_default.isObject(target[name])) target[name] = [];
		if (buildPath(path, value, target[name], index) && utils_default.isArray(target[name])) target[name] = arrayToObject(target[name]);
		return !isNumericKey;
	}
	if (utils_default.isFormData(formData) && utils_default.isFunction(formData.entries)) {
		const obj = {};
		utils_default.forEachEntry(formData, (name, value) => {
			buildPath(parsePropPath(name), value, obj, 0);
		});
		return obj;
	}
	return null;
}
var formDataToJSON_default = formDataToJSON;
function stringifySafely(rawValue, parser, encoder) {
	if (utils_default.isString(rawValue)) try {
		(parser || JSON.parse)(rawValue);
		return utils_default.trim(rawValue);
	} catch (e) {
		if (e.name !== "SyntaxError") throw e;
	}
	return (encoder || JSON.stringify)(rawValue);
}
var defaults = {
	transitional: transitional_default,
	adapter: [
		"xhr",
		"http",
		"fetch"
	],
	transformRequest: [function transformRequest(data, headers) {
		const contentType = headers.getContentType() || "";
		const hasJSONContentType = contentType.indexOf("application/json") > -1;
		const isObjectPayload = utils_default.isObject(data);
		if (isObjectPayload && utils_default.isHTMLForm(data)) data = new FormData(data);
		if (utils_default.isFormData(data)) return hasJSONContentType ? JSON.stringify(formDataToJSON_default(data)) : data;
		if (utils_default.isArrayBuffer(data) || utils_default.isBuffer(data) || utils_default.isStream(data) || utils_default.isFile(data) || utils_default.isBlob(data) || utils_default.isReadableStream(data)) return data;
		if (utils_default.isArrayBufferView(data)) return data.buffer;
		if (utils_default.isURLSearchParams(data)) {
			headers.setContentType("application/x-www-form-urlencoded;charset=utf-8", false);
			return data.toString();
		}
		let isFileList$1;
		if (isObjectPayload) {
			if (contentType.indexOf("application/x-www-form-urlencoded") > -1) return toURLEncodedForm(data, this.formSerializer).toString();
			if ((isFileList$1 = utils_default.isFileList(data)) || contentType.indexOf("multipart/form-data") > -1) {
				const _FormData = this.env && this.env.FormData;
				return toFormData_default(isFileList$1 ? { "files[]": data } : data, _FormData && new _FormData(), this.formSerializer);
			}
		}
		if (isObjectPayload || hasJSONContentType) {
			headers.setContentType("application/json", false);
			return stringifySafely(data);
		}
		return data;
	}],
	transformResponse: [function transformResponse(data) {
		const transitional = this.transitional || defaults.transitional;
		const forcedJSONParsing = transitional && transitional.forcedJSONParsing;
		const JSONRequested = this.responseType === "json";
		if (utils_default.isResponse(data) || utils_default.isReadableStream(data)) return data;
		if (data && utils_default.isString(data) && (forcedJSONParsing && !this.responseType || JSONRequested)) {
			const strictJSONParsing = !(transitional && transitional.silentJSONParsing) && JSONRequested;
			try {
				return JSON.parse(data, this.parseReviver);
			} catch (e) {
				if (strictJSONParsing) {
					if (e.name === "SyntaxError") throw AxiosError_default.from(e, AxiosError_default.ERR_BAD_RESPONSE, this, null, this.response);
					throw e;
				}
			}
		}
		return data;
	}],
	timeout: 0,
	xsrfCookieName: "XSRF-TOKEN",
	xsrfHeaderName: "X-XSRF-TOKEN",
	maxContentLength: -1,
	maxBodyLength: -1,
	env: {
		FormData: platform_default.classes.FormData,
		Blob: platform_default.classes.Blob
	},
	validateStatus: function validateStatus(status) {
		return status >= 200 && status < 300;
	},
	headers: { common: {
		"Accept": "application/json, text/plain, */*",
		"Content-Type": void 0
	} }
};
utils_default.forEach([
	"delete",
	"get",
	"head",
	"post",
	"put",
	"patch"
], (method) => {
	defaults.headers[method] = {};
});
var defaults_default = defaults;
var ignoreDuplicateOf = utils_default.toObjectSet([
	"age",
	"authorization",
	"content-length",
	"content-type",
	"etag",
	"expires",
	"from",
	"host",
	"if-modified-since",
	"if-unmodified-since",
	"last-modified",
	"location",
	"max-forwards",
	"proxy-authorization",
	"referer",
	"retry-after",
	"user-agent"
]);
var parseHeaders_default = (rawHeaders) => {
	const parsed = {};
	let key;
	let val;
	let i;
	rawHeaders && rawHeaders.split("\n").forEach(function parser(line) {
		i = line.indexOf(":");
		key = line.substring(0, i).trim().toLowerCase();
		val = line.substring(i + 1).trim();
		if (!key || parsed[key] && ignoreDuplicateOf[key]) return;
		if (key === "set-cookie") if (parsed[key]) parsed[key].push(val);
		else parsed[key] = [val];
		else parsed[key] = parsed[key] ? parsed[key] + ", " + val : val;
	});
	return parsed;
};
var $internals = Symbol("internals");
function normalizeHeader(header) {
	return header && String(header).trim().toLowerCase();
}
function normalizeValue(value) {
	if (value === false || value == null) return value;
	return utils_default.isArray(value) ? value.map(normalizeValue) : String(value);
}
function parseTokens(str) {
	const tokens = Object.create(null);
	const tokensRE = /([^\s,;=]+)\s*(?:=\s*([^,;]+))?/g;
	let match;
	while (match = tokensRE.exec(str)) tokens[match[1]] = match[2];
	return tokens;
}
var isValidHeaderName = (str) => /^[-_a-zA-Z0-9^`|~,!#$%&'*+.]+$/.test(str.trim());
function matchHeaderValue(context, value, header, filter, isHeaderNameFilter) {
	if (utils_default.isFunction(filter)) return filter.call(this, value, header);
	if (isHeaderNameFilter) value = header;
	if (!utils_default.isString(value)) return;
	if (utils_default.isString(filter)) return value.indexOf(filter) !== -1;
	if (utils_default.isRegExp(filter)) return filter.test(value);
}
function formatHeader(header) {
	return header.trim().toLowerCase().replace(/([a-z\d])(\w*)/g, (w, char, str) => {
		return char.toUpperCase() + str;
	});
}
function buildAccessors(obj, header) {
	const accessorName = utils_default.toCamelCase(" " + header);
	[
		"get",
		"set",
		"has"
	].forEach((methodName) => {
		Object.defineProperty(obj, methodName + accessorName, {
			value: function(arg1, arg2, arg3) {
				return this[methodName].call(this, header, arg1, arg2, arg3);
			},
			configurable: true
		});
	});
}
var AxiosHeaders = class {
	constructor(headers) {
		headers && this.set(headers);
	}
	set(header, valueOrRewrite, rewrite) {
		const self$1 = this;
		function setHeader(_value, _header, _rewrite) {
			const lHeader = normalizeHeader(_header);
			if (!lHeader) throw new Error("header name must be a non-empty string");
			const key = utils_default.findKey(self$1, lHeader);
			if (!key || self$1[key] === void 0 || _rewrite === true || _rewrite === void 0 && self$1[key] !== false) self$1[key || _header] = normalizeValue(_value);
		}
		const setHeaders = (headers, _rewrite) => utils_default.forEach(headers, (_value, _header) => setHeader(_value, _header, _rewrite));
		if (utils_default.isPlainObject(header) || header instanceof this.constructor) setHeaders(header, valueOrRewrite);
		else if (utils_default.isString(header) && (header = header.trim()) && !isValidHeaderName(header)) setHeaders(parseHeaders_default(header), valueOrRewrite);
		else if (utils_default.isObject(header) && utils_default.isIterable(header)) {
			let obj = {}, dest, key;
			for (const entry of header) {
				if (!utils_default.isArray(entry)) throw TypeError("Object iterator must return a key-value pair");
				obj[key = entry[0]] = (dest = obj[key]) ? utils_default.isArray(dest) ? [...dest, entry[1]] : [dest, entry[1]] : entry[1];
			}
			setHeaders(obj, valueOrRewrite);
		} else header != null && setHeader(valueOrRewrite, header, rewrite);
		return this;
	}
	get(header, parser) {
		header = normalizeHeader(header);
		if (header) {
			const key = utils_default.findKey(this, header);
			if (key) {
				const value = this[key];
				if (!parser) return value;
				if (parser === true) return parseTokens(value);
				if (utils_default.isFunction(parser)) return parser.call(this, value, key);
				if (utils_default.isRegExp(parser)) return parser.exec(value);
				throw new TypeError("parser must be boolean|regexp|function");
			}
		}
	}
	has(header, matcher) {
		header = normalizeHeader(header);
		if (header) {
			const key = utils_default.findKey(this, header);
			return !!(key && this[key] !== void 0 && (!matcher || matchHeaderValue(this, this[key], key, matcher)));
		}
		return false;
	}
	delete(header, matcher) {
		const self$1 = this;
		let deleted = false;
		function deleteHeader(_header) {
			_header = normalizeHeader(_header);
			if (_header) {
				const key = utils_default.findKey(self$1, _header);
				if (key && (!matcher || matchHeaderValue(self$1, self$1[key], key, matcher))) {
					delete self$1[key];
					deleted = true;
				}
			}
		}
		if (utils_default.isArray(header)) header.forEach(deleteHeader);
		else deleteHeader(header);
		return deleted;
	}
	clear(matcher) {
		const keys = Object.keys(this);
		let i = keys.length;
		let deleted = false;
		while (i--) {
			const key = keys[i];
			if (!matcher || matchHeaderValue(this, this[key], key, matcher, true)) {
				delete this[key];
				deleted = true;
			}
		}
		return deleted;
	}
	normalize(format) {
		const self$1 = this;
		const headers = {};
		utils_default.forEach(this, (value, header) => {
			const key = utils_default.findKey(headers, header);
			if (key) {
				self$1[key] = normalizeValue(value);
				delete self$1[header];
				return;
			}
			const normalized = format ? formatHeader(header) : String(header).trim();
			if (normalized !== header) delete self$1[header];
			self$1[normalized] = normalizeValue(value);
			headers[normalized] = true;
		});
		return this;
	}
	concat(...targets) {
		return this.constructor.concat(this, ...targets);
	}
	toJSON(asStrings) {
		const obj = Object.create(null);
		utils_default.forEach(this, (value, header) => {
			value != null && value !== false && (obj[header] = asStrings && utils_default.isArray(value) ? value.join(", ") : value);
		});
		return obj;
	}
	[Symbol.iterator]() {
		return Object.entries(this.toJSON())[Symbol.iterator]();
	}
	toString() {
		return Object.entries(this.toJSON()).map(([header, value]) => header + ": " + value).join("\n");
	}
	getSetCookie() {
		return this.get("set-cookie") || [];
	}
	get [Symbol.toStringTag]() {
		return "AxiosHeaders";
	}
	static from(thing) {
		return thing instanceof this ? thing : new this(thing);
	}
	static concat(first, ...targets) {
		const computed = new this(first);
		targets.forEach((target) => computed.set(target));
		return computed;
	}
	static accessor(header) {
		const accessors = (this[$internals] = this[$internals] = { accessors: {} }).accessors;
		const prototype$2 = this.prototype;
		function defineAccessor(_header) {
			const lHeader = normalizeHeader(_header);
			if (!accessors[lHeader]) {
				buildAccessors(prototype$2, _header);
				accessors[lHeader] = true;
			}
		}
		utils_default.isArray(header) ? header.forEach(defineAccessor) : defineAccessor(header);
		return this;
	}
};
AxiosHeaders.accessor([
	"Content-Type",
	"Content-Length",
	"Accept",
	"Accept-Encoding",
	"User-Agent",
	"Authorization"
]);
utils_default.reduceDescriptors(AxiosHeaders.prototype, ({ value }, key) => {
	let mapped = key[0].toUpperCase() + key.slice(1);
	return {
		get: () => value,
		set(headerValue) {
			this[mapped] = headerValue;
		}
	};
});
utils_default.freezeMethods(AxiosHeaders);
var AxiosHeaders_default = AxiosHeaders;
function transformData(fns, response) {
	const config = this || defaults_default;
	const context = response || config;
	const headers = AxiosHeaders_default.from(context.headers);
	let data = context.data;
	utils_default.forEach(fns, function transform(fn) {
		data = fn.call(config, data, headers.normalize(), response ? response.status : void 0);
	});
	headers.normalize();
	return data;
}
function isCancel(value) {
	return !!(value && value.__CANCEL__);
}
function CanceledError(message, config, request) {
	AxiosError_default.call(this, message == null ? "canceled" : message, AxiosError_default.ERR_CANCELED, config, request);
	this.name = "CanceledError";
}
utils_default.inherits(CanceledError, AxiosError_default, { __CANCEL__: true });
var CanceledError_default = CanceledError;
function settle(resolve, reject, response) {
	const validateStatus = response.config.validateStatus;
	if (!response.status || !validateStatus || validateStatus(response.status)) resolve(response);
	else reject(new AxiosError_default("Request failed with status code " + response.status, [AxiosError_default.ERR_BAD_REQUEST, AxiosError_default.ERR_BAD_RESPONSE][Math.floor(response.status / 100) - 4], response.config, response.request, response));
}
function parseProtocol(url) {
	const match = /^([-+\w]{1,25})(:?\/\/|:)/.exec(url);
	return match && match[1] || "";
}
function speedometer(samplesCount, min) {
	samplesCount = samplesCount || 10;
	const bytes = new Array(samplesCount);
	const timestamps = new Array(samplesCount);
	let head = 0;
	let tail = 0;
	let firstSampleTS;
	min = min !== void 0 ? min : 1e3;
	return function push(chunkLength) {
		const now = Date.now();
		const startedAt = timestamps[tail];
		if (!firstSampleTS) firstSampleTS = now;
		bytes[head] = chunkLength;
		timestamps[head] = now;
		let i = tail;
		let bytesCount = 0;
		while (i !== head) {
			bytesCount += bytes[i++];
			i = i % samplesCount;
		}
		head = (head + 1) % samplesCount;
		if (head === tail) tail = (tail + 1) % samplesCount;
		if (now - firstSampleTS < min) return;
		const passed = startedAt && now - startedAt;
		return passed ? Math.round(bytesCount * 1e3 / passed) : void 0;
	};
}
var speedometer_default = speedometer;
function throttle(fn, freq) {
	let timestamp = 0;
	let threshold = 1e3 / freq;
	let lastArgs;
	let timer;
	const invoke = (args, now = Date.now()) => {
		timestamp = now;
		lastArgs = null;
		if (timer) {
			clearTimeout(timer);
			timer = null;
		}
		fn(...args);
	};
	const throttled = (...args) => {
		const now = Date.now();
		const passed = now - timestamp;
		if (passed >= threshold) invoke(args, now);
		else {
			lastArgs = args;
			if (!timer) timer = setTimeout(() => {
				timer = null;
				invoke(lastArgs);
			}, threshold - passed);
		}
	};
	const flush = () => lastArgs && invoke(lastArgs);
	return [throttled, flush];
}
var throttle_default = throttle;
const progressEventReducer = (listener, isDownloadStream, freq = 3) => {
	let bytesNotified = 0;
	const _speedometer = speedometer_default(50, 250);
	return throttle_default((e) => {
		const loaded = e.loaded;
		const total = e.lengthComputable ? e.total : void 0;
		const progressBytes$1 = loaded - bytesNotified;
		const rate = _speedometer(progressBytes$1);
		const inRange = loaded <= total;
		bytesNotified = loaded;
		listener({
			loaded,
			total,
			progress: total ? loaded / total : void 0,
			bytes: progressBytes$1,
			rate: rate ? rate : void 0,
			estimated: rate && total && inRange ? (total - loaded) / rate : void 0,
			event: e,
			lengthComputable: total != null,
			[isDownloadStream ? "download" : "upload"]: true
		});
	}, freq);
};
const progressEventDecorator = (total, throttled) => {
	const lengthComputable = total != null;
	return [(loaded) => throttled[0]({
		lengthComputable,
		total,
		loaded
	}), throttled[1]];
};
const asyncDecorator = (fn) => (...args) => utils_default.asap(() => fn(...args));
var isURLSameOrigin_default = platform_default.hasStandardBrowserEnv ? ((origin$1, isMSIE) => (url) => {
	url = new URL(url, platform_default.origin);
	return origin$1.protocol === url.protocol && origin$1.host === url.host && (isMSIE || origin$1.port === url.port);
})(new URL(platform_default.origin), platform_default.navigator && /(msie|trident)/i.test(platform_default.navigator.userAgent)) : () => true;
var cookies_default = platform_default.hasStandardBrowserEnv ? {
	write(name, value, expires, path, domain, secure, sameSite) {
		if (typeof document === "undefined") return;
		const cookie = [`${name}=${encodeURIComponent(value)}`];
		if (utils_default.isNumber(expires)) cookie.push(`expires=${new Date(expires).toUTCString()}`);
		if (utils_default.isString(path)) cookie.push(`path=${path}`);
		if (utils_default.isString(domain)) cookie.push(`domain=${domain}`);
		if (secure === true) cookie.push("secure");
		if (utils_default.isString(sameSite)) cookie.push(`SameSite=${sameSite}`);
		document.cookie = cookie.join("; ");
	},
	read(name) {
		if (typeof document === "undefined") return null;
		const match = document.cookie.match(/* @__PURE__ */ new RegExp("(?:^|; )" + name + "=([^;]*)"));
		return match ? decodeURIComponent(match[1]) : null;
	},
	remove(name) {
		this.write(name, "", Date.now() - 864e5, "/");
	}
} : {
	write() {},
	read() {
		return null;
	},
	remove() {}
};
function isAbsoluteURL(url) {
	return /^([a-z][a-z\d+\-.]*:)?\/\//i.test(url);
}
function combineURLs(baseURL, relativeURL) {
	return relativeURL ? baseURL.replace(/\/?\/$/, "") + "/" + relativeURL.replace(/^\/+/, "") : baseURL;
}
function buildFullPath(baseURL, requestedURL, allowAbsoluteUrls) {
	let isRelativeUrl = !isAbsoluteURL(requestedURL);
	if (baseURL && (isRelativeUrl || allowAbsoluteUrls == false)) return combineURLs(baseURL, requestedURL);
	return requestedURL;
}
var headersToObject = (thing) => thing instanceof AxiosHeaders_default ? { ...thing } : thing;
function mergeConfig(config1, config2) {
	config2 = config2 || {};
	const config = {};
	function getMergedValue(target, source$1, prop, caseless) {
		if (utils_default.isPlainObject(target) && utils_default.isPlainObject(source$1)) return utils_default.merge.call({ caseless }, target, source$1);
		else if (utils_default.isPlainObject(source$1)) return utils_default.merge({}, source$1);
		else if (utils_default.isArray(source$1)) return source$1.slice();
		return source$1;
	}
	function mergeDeepProperties(a, b, prop, caseless) {
		if (!utils_default.isUndefined(b)) return getMergedValue(a, b, prop, caseless);
		else if (!utils_default.isUndefined(a)) return getMergedValue(void 0, a, prop, caseless);
	}
	function valueFromConfig2(a, b) {
		if (!utils_default.isUndefined(b)) return getMergedValue(void 0, b);
	}
	function defaultToConfig2(a, b) {
		if (!utils_default.isUndefined(b)) return getMergedValue(void 0, b);
		else if (!utils_default.isUndefined(a)) return getMergedValue(void 0, a);
	}
	function mergeDirectKeys(a, b, prop) {
		if (prop in config2) return getMergedValue(a, b);
		else if (prop in config1) return getMergedValue(void 0, a);
	}
	const mergeMap = {
		url: valueFromConfig2,
		method: valueFromConfig2,
		data: valueFromConfig2,
		baseURL: defaultToConfig2,
		transformRequest: defaultToConfig2,
		transformResponse: defaultToConfig2,
		paramsSerializer: defaultToConfig2,
		timeout: defaultToConfig2,
		timeoutMessage: defaultToConfig2,
		withCredentials: defaultToConfig2,
		withXSRFToken: defaultToConfig2,
		adapter: defaultToConfig2,
		responseType: defaultToConfig2,
		xsrfCookieName: defaultToConfig2,
		xsrfHeaderName: defaultToConfig2,
		onUploadProgress: defaultToConfig2,
		onDownloadProgress: defaultToConfig2,
		decompress: defaultToConfig2,
		maxContentLength: defaultToConfig2,
		maxBodyLength: defaultToConfig2,
		beforeRedirect: defaultToConfig2,
		transport: defaultToConfig2,
		httpAgent: defaultToConfig2,
		httpsAgent: defaultToConfig2,
		cancelToken: defaultToConfig2,
		socketPath: defaultToConfig2,
		responseEncoding: defaultToConfig2,
		validateStatus: mergeDirectKeys,
		headers: (a, b, prop) => mergeDeepProperties(headersToObject(a), headersToObject(b), prop, true)
	};
	utils_default.forEach(Object.keys({
		...config1,
		...config2
	}), function computeConfigValue(prop) {
		const merge$1 = mergeMap[prop] || mergeDeepProperties;
		const configValue = merge$1(config1[prop], config2[prop], prop);
		utils_default.isUndefined(configValue) && merge$1 !== mergeDirectKeys || (config[prop] = configValue);
	});
	return config;
}
var resolveConfig_default = (config) => {
	const newConfig = mergeConfig({}, config);
	let { data, withXSRFToken, xsrfHeaderName, xsrfCookieName, headers, auth } = newConfig;
	newConfig.headers = headers = AxiosHeaders_default.from(headers);
	newConfig.url = buildURL(buildFullPath(newConfig.baseURL, newConfig.url, newConfig.allowAbsoluteUrls), config.params, config.paramsSerializer);
	if (auth) headers.set("Authorization", "Basic " + btoa((auth.username || "") + ":" + (auth.password ? unescape(encodeURIComponent(auth.password)) : "")));
	if (utils_default.isFormData(data)) {
		if (platform_default.hasStandardBrowserEnv || platform_default.hasStandardBrowserWebWorkerEnv) headers.setContentType(void 0);
		else if (utils_default.isFunction(data.getHeaders)) {
			const formHeaders = data.getHeaders();
			const allowedHeaders = ["content-type", "content-length"];
			Object.entries(formHeaders).forEach(([key, val]) => {
				if (allowedHeaders.includes(key.toLowerCase())) headers.set(key, val);
			});
		}
	}
	if (platform_default.hasStandardBrowserEnv) {
		withXSRFToken && utils_default.isFunction(withXSRFToken) && (withXSRFToken = withXSRFToken(newConfig));
		if (withXSRFToken || withXSRFToken !== false && isURLSameOrigin_default(newConfig.url)) {
			const xsrfValue = xsrfHeaderName && xsrfCookieName && cookies_default.read(xsrfCookieName);
			if (xsrfValue) headers.set(xsrfHeaderName, xsrfValue);
		}
	}
	return newConfig;
};
var xhr_default = typeof XMLHttpRequest !== "undefined" && function(config) {
	return new Promise(function dispatchXhrRequest(resolve, reject) {
		const _config = resolveConfig_default(config);
		let requestData = _config.data;
		const requestHeaders = AxiosHeaders_default.from(_config.headers).normalize();
		let { responseType, onUploadProgress, onDownloadProgress } = _config;
		let onCanceled;
		let uploadThrottled, downloadThrottled;
		let flushUpload, flushDownload;
		function done() {
			flushUpload && flushUpload();
			flushDownload && flushDownload();
			_config.cancelToken && _config.cancelToken.unsubscribe(onCanceled);
			_config.signal && _config.signal.removeEventListener("abort", onCanceled);
		}
		let request = new XMLHttpRequest();
		request.open(_config.method.toUpperCase(), _config.url, true);
		request.timeout = _config.timeout;
		function onloadend() {
			if (!request) return;
			const responseHeaders = AxiosHeaders_default.from("getAllResponseHeaders" in request && request.getAllResponseHeaders());
			settle(function _resolve(value) {
				resolve(value);
				done();
			}, function _reject(err) {
				reject(err);
				done();
			}, {
				data: !responseType || responseType === "text" || responseType === "json" ? request.responseText : request.response,
				status: request.status,
				statusText: request.statusText,
				headers: responseHeaders,
				config,
				request
			});
			request = null;
		}
		if ("onloadend" in request) request.onloadend = onloadend;
		else request.onreadystatechange = function handleLoad() {
			if (!request || request.readyState !== 4) return;
			if (request.status === 0 && !(request.responseURL && request.responseURL.indexOf("file:") === 0)) return;
			setTimeout(onloadend);
		};
		request.onabort = function handleAbort() {
			if (!request) return;
			reject(new AxiosError_default("Request aborted", AxiosError_default.ECONNABORTED, config, request));
			request = null;
		};
		request.onerror = function handleError(event) {
			const err = new AxiosError_default(event && event.message ? event.message : "Network Error", AxiosError_default.ERR_NETWORK, config, request);
			err.event = event || null;
			reject(err);
			request = null;
		};
		request.ontimeout = function handleTimeout() {
			let timeoutErrorMessage = _config.timeout ? "timeout of " + _config.timeout + "ms exceeded" : "timeout exceeded";
			const transitional = _config.transitional || transitional_default;
			if (_config.timeoutErrorMessage) timeoutErrorMessage = _config.timeoutErrorMessage;
			reject(new AxiosError_default(timeoutErrorMessage, transitional.clarifyTimeoutError ? AxiosError_default.ETIMEDOUT : AxiosError_default.ECONNABORTED, config, request));
			request = null;
		};
		requestData === void 0 && requestHeaders.setContentType(null);
		if ("setRequestHeader" in request) utils_default.forEach(requestHeaders.toJSON(), function setRequestHeader(val, key) {
			request.setRequestHeader(key, val);
		});
		if (!utils_default.isUndefined(_config.withCredentials)) request.withCredentials = !!_config.withCredentials;
		if (responseType && responseType !== "json") request.responseType = _config.responseType;
		if (onDownloadProgress) {
			[downloadThrottled, flushDownload] = progressEventReducer(onDownloadProgress, true);
			request.addEventListener("progress", downloadThrottled);
		}
		if (onUploadProgress && request.upload) {
			[uploadThrottled, flushUpload] = progressEventReducer(onUploadProgress);
			request.upload.addEventListener("progress", uploadThrottled);
			request.upload.addEventListener("loadend", flushUpload);
		}
		if (_config.cancelToken || _config.signal) {
			onCanceled = (cancel) => {
				if (!request) return;
				reject(!cancel || cancel.type ? new CanceledError_default(null, config, request) : cancel);
				request.abort();
				request = null;
			};
			_config.cancelToken && _config.cancelToken.subscribe(onCanceled);
			if (_config.signal) _config.signal.aborted ? onCanceled() : _config.signal.addEventListener("abort", onCanceled);
		}
		const protocol = parseProtocol(_config.url);
		if (protocol && platform_default.protocols.indexOf(protocol) === -1) {
			reject(new AxiosError_default("Unsupported protocol " + protocol + ":", AxiosError_default.ERR_BAD_REQUEST, config));
			return;
		}
		request.send(requestData || null);
	});
};
var composeSignals = (signals, timeout) => {
	const { length } = signals = signals ? signals.filter(Boolean) : [];
	if (timeout || length) {
		let controller = new AbortController();
		let aborted;
		const onabort = function(reason) {
			if (!aborted) {
				aborted = true;
				unsubscribe();
				const err = reason instanceof Error ? reason : this.reason;
				controller.abort(err instanceof AxiosError_default ? err : new CanceledError_default(err instanceof Error ? err.message : err));
			}
		};
		let timer = timeout && setTimeout(() => {
			timer = null;
			onabort(new AxiosError_default(`timeout ${timeout} of ms exceeded`, AxiosError_default.ETIMEDOUT));
		}, timeout);
		const unsubscribe = () => {
			if (signals) {
				timer && clearTimeout(timer);
				timer = null;
				signals.forEach((signal$1) => {
					signal$1.unsubscribe ? signal$1.unsubscribe(onabort) : signal$1.removeEventListener("abort", onabort);
				});
				signals = null;
			}
		};
		signals.forEach((signal$1) => signal$1.addEventListener("abort", onabort));
		const { signal } = controller;
		signal.unsubscribe = () => utils_default.asap(unsubscribe);
		return signal;
	}
};
var composeSignals_default = composeSignals;
const streamChunk = function* (chunk, chunkSize) {
	let len = chunk.byteLength;
	if (!chunkSize || len < chunkSize) {
		yield chunk;
		return;
	}
	let pos = 0;
	let end;
	while (pos < len) {
		end = pos + chunkSize;
		yield chunk.slice(pos, end);
		pos = end;
	}
};
const readBytes = async function* (iterable, chunkSize) {
	for await (const chunk of readStream(iterable)) yield* streamChunk(chunk, chunkSize);
};
var readStream = async function* (stream) {
	if (stream[Symbol.asyncIterator]) {
		yield* stream;
		return;
	}
	const reader = stream.getReader();
	try {
		for (;;) {
			const { done, value } = await reader.read();
			if (done) break;
			yield value;
		}
	} finally {
		await reader.cancel();
	}
};
const trackStream = (stream, chunkSize, onProgress, onFinish) => {
	const iterator$1 = readBytes(stream, chunkSize);
	let bytes = 0;
	let done;
	let _onFinish = (e) => {
		if (!done) {
			done = true;
			onFinish && onFinish(e);
		}
	};
	return new ReadableStream({
		async pull(controller) {
			try {
				const { done: done$1, value } = await iterator$1.next();
				if (done$1) {
					_onFinish();
					controller.close();
					return;
				}
				let len = value.byteLength;
				if (onProgress) onProgress(bytes += len);
				controller.enqueue(new Uint8Array(value));
			} catch (err) {
				_onFinish(err);
				throw err;
			}
		},
		cancel(reason) {
			_onFinish(reason);
			return iterator$1.return();
		}
	}, { highWaterMark: 2 });
};
var DEFAULT_CHUNK_SIZE = 64 * 1024;
var { isFunction } = utils_default;
var globalFetchAPI = (({ Request, Response }) => ({
	Request,
	Response
}))(utils_default.global);
var { ReadableStream: ReadableStream$1, TextEncoder: TextEncoder$1 } = utils_default.global;
var test = (fn, ...args) => {
	try {
		return !!fn(...args);
	} catch (e) {
		return false;
	}
};
var factory = (env) => {
	env = utils_default.merge.call({ skipUndefined: true }, globalFetchAPI, env);
	const { fetch: envFetch, Request, Response } = env;
	const isFetchSupported = envFetch ? isFunction(envFetch) : typeof fetch === "function";
	const isRequestSupported = isFunction(Request);
	const isResponseSupported = isFunction(Response);
	if (!isFetchSupported) return false;
	const isReadableStreamSupported = isFetchSupported && isFunction(ReadableStream$1);
	const encodeText = isFetchSupported && (typeof TextEncoder$1 === "function" ? ((encoder) => (str) => encoder.encode(str))(new TextEncoder$1()) : async (str) => new Uint8Array(await new Request(str).arrayBuffer()));
	const supportsRequestStream = isRequestSupported && isReadableStreamSupported && test(() => {
		let duplexAccessed = false;
		const hasContentType = new Request(platform_default.origin, {
			body: new ReadableStream$1(),
			method: "POST",
			get duplex() {
				duplexAccessed = true;
				return "half";
			}
		}).headers.has("Content-Type");
		return duplexAccessed && !hasContentType;
	});
	const supportsResponseStream = isResponseSupported && isReadableStreamSupported && test(() => utils_default.isReadableStream(new Response("").body));
	const resolvers = { stream: supportsResponseStream && ((res) => res.body) };
	isFetchSupported && [
		"text",
		"arrayBuffer",
		"blob",
		"formData",
		"stream"
	].forEach((type) => {
		!resolvers[type] && (resolvers[type] = (res, config) => {
			let method = res && res[type];
			if (method) return method.call(res);
			throw new AxiosError_default(`Response type '${type}' is not supported`, AxiosError_default.ERR_NOT_SUPPORT, config);
		});
	});
	const getBodyLength = async (body) => {
		if (body == null) return 0;
		if (utils_default.isBlob(body)) return body.size;
		if (utils_default.isSpecCompliantForm(body)) return (await new Request(platform_default.origin, {
			method: "POST",
			body
		}).arrayBuffer()).byteLength;
		if (utils_default.isArrayBufferView(body) || utils_default.isArrayBuffer(body)) return body.byteLength;
		if (utils_default.isURLSearchParams(body)) body = body + "";
		if (utils_default.isString(body)) return (await encodeText(body)).byteLength;
	};
	const resolveBodyLength = async (headers, body) => {
		const length = utils_default.toFiniteNumber(headers.getContentLength());
		return length == null ? getBodyLength(body) : length;
	};
	return async (config) => {
		let { url, method, data, signal, cancelToken, timeout, onDownloadProgress, onUploadProgress, responseType, headers, withCredentials = "same-origin", fetchOptions } = resolveConfig_default(config);
		let _fetch = envFetch || fetch;
		responseType = responseType ? (responseType + "").toLowerCase() : "text";
		let composedSignal = composeSignals_default([signal, cancelToken && cancelToken.toAbortSignal()], timeout);
		let request = null;
		const unsubscribe = composedSignal && composedSignal.unsubscribe && (() => {
			composedSignal.unsubscribe();
		});
		let requestContentLength;
		try {
			if (onUploadProgress && supportsRequestStream && method !== "get" && method !== "head" && (requestContentLength = await resolveBodyLength(headers, data)) !== 0) {
				let _request = new Request(url, {
					method: "POST",
					body: data,
					duplex: "half"
				});
				let contentTypeHeader;
				if (utils_default.isFormData(data) && (contentTypeHeader = _request.headers.get("content-type"))) headers.setContentType(contentTypeHeader);
				if (_request.body) {
					const [onProgress, flush] = progressEventDecorator(requestContentLength, progressEventReducer(asyncDecorator(onUploadProgress)));
					data = trackStream(_request.body, DEFAULT_CHUNK_SIZE, onProgress, flush);
				}
			}
			if (!utils_default.isString(withCredentials)) withCredentials = withCredentials ? "include" : "omit";
			const isCredentialsSupported = isRequestSupported && "credentials" in Request.prototype;
			const resolvedOptions = {
				...fetchOptions,
				signal: composedSignal,
				method: method.toUpperCase(),
				headers: headers.normalize().toJSON(),
				body: data,
				duplex: "half",
				credentials: isCredentialsSupported ? withCredentials : void 0
			};
			request = isRequestSupported && new Request(url, resolvedOptions);
			let response = await (isRequestSupported ? _fetch(request, fetchOptions) : _fetch(url, resolvedOptions));
			const isStreamResponse = supportsResponseStream && (responseType === "stream" || responseType === "response");
			if (supportsResponseStream && (onDownloadProgress || isStreamResponse && unsubscribe)) {
				const options = {};
				[
					"status",
					"statusText",
					"headers"
				].forEach((prop) => {
					options[prop] = response[prop];
				});
				const responseContentLength = utils_default.toFiniteNumber(response.headers.get("content-length"));
				const [onProgress, flush] = onDownloadProgress && progressEventDecorator(responseContentLength, progressEventReducer(asyncDecorator(onDownloadProgress), true)) || [];
				response = new Response(trackStream(response.body, DEFAULT_CHUNK_SIZE, onProgress, () => {
					flush && flush();
					unsubscribe && unsubscribe();
				}), options);
			}
			responseType = responseType || "text";
			let responseData = await resolvers[utils_default.findKey(resolvers, responseType) || "text"](response, config);
			!isStreamResponse && unsubscribe && unsubscribe();
			return await new Promise((resolve, reject) => {
				settle(resolve, reject, {
					data: responseData,
					headers: AxiosHeaders_default.from(response.headers),
					status: response.status,
					statusText: response.statusText,
					config,
					request
				});
			});
		} catch (err) {
			unsubscribe && unsubscribe();
			if (err && err.name === "TypeError" && /Load failed|fetch/i.test(err.message)) throw Object.assign(new AxiosError_default("Network Error", AxiosError_default.ERR_NETWORK, config, request), { cause: err.cause || err });
			throw AxiosError_default.from(err, err && err.code, config, request);
		}
	};
};
var seedCache = /* @__PURE__ */ new Map();
const getFetch = (config) => {
	let env = config && config.env || {};
	const { fetch: fetch$1, Request, Response } = env;
	const seeds = [
		Request,
		Response,
		fetch$1
	];
	let i = seeds.length, seed, target, map = seedCache;
	while (i--) {
		seed = seeds[i];
		target = map.get(seed);
		target === void 0 && map.set(seed, target = i ? /* @__PURE__ */ new Map() : factory(env));
		map = target;
	}
	return target;
};
getFetch();
var knownAdapters = {
	http: null,
	xhr: xhr_default,
	fetch: { get: getFetch }
};
utils_default.forEach(knownAdapters, (fn, value) => {
	if (fn) {
		try {
			Object.defineProperty(fn, "name", { value });
		} catch (e) {}
		Object.defineProperty(fn, "adapterName", { value });
	}
});
var renderReason = (reason) => `- ${reason}`;
var isResolvedHandle = (adapter$1) => utils_default.isFunction(adapter$1) || adapter$1 === null || adapter$1 === false;
function getAdapter(adapters, config) {
	adapters = utils_default.isArray(adapters) ? adapters : [adapters];
	const { length } = adapters;
	let nameOrAdapter;
	let adapter$1;
	const rejectedReasons = {};
	for (let i = 0; i < length; i++) {
		nameOrAdapter = adapters[i];
		let id;
		adapter$1 = nameOrAdapter;
		if (!isResolvedHandle(nameOrAdapter)) {
			adapter$1 = knownAdapters[(id = String(nameOrAdapter)).toLowerCase()];
			if (adapter$1 === void 0) throw new AxiosError_default(`Unknown adapter '${id}'`);
		}
		if (adapter$1 && (utils_default.isFunction(adapter$1) || (adapter$1 = adapter$1.get(config)))) break;
		rejectedReasons[id || "#" + i] = adapter$1;
	}
	if (!adapter$1) {
		const reasons = Object.entries(rejectedReasons).map(([id, state$1]) => `adapter ${id} ` + (state$1 === false ? "is not supported by the environment" : "is not available in the build"));
		throw new AxiosError_default(`There is no suitable adapter to dispatch the request ` + (length ? reasons.length > 1 ? "since :\n" + reasons.map(renderReason).join("\n") : " " + renderReason(reasons[0]) : "as no adapter specified"), "ERR_NOT_SUPPORT");
	}
	return adapter$1;
}
var adapters_default = {
	getAdapter,
	adapters: knownAdapters
};
function throwIfCancellationRequested(config) {
	if (config.cancelToken) config.cancelToken.throwIfRequested();
	if (config.signal && config.signal.aborted) throw new CanceledError_default(null, config);
}
function dispatchRequest(config) {
	throwIfCancellationRequested(config);
	config.headers = AxiosHeaders_default.from(config.headers);
	config.data = transformData.call(config, config.transformRequest);
	if ([
		"post",
		"put",
		"patch"
	].indexOf(config.method) !== -1) config.headers.setContentType("application/x-www-form-urlencoded", false);
	return adapters_default.getAdapter(config.adapter || defaults_default.adapter, config)(config).then(function onAdapterResolution(response) {
		throwIfCancellationRequested(config);
		response.data = transformData.call(config, config.transformResponse, response);
		response.headers = AxiosHeaders_default.from(response.headers);
		return response;
	}, function onAdapterRejection(reason) {
		if (!isCancel(reason)) {
			throwIfCancellationRequested(config);
			if (reason && reason.response) {
				reason.response.data = transformData.call(config, config.transformResponse, reason.response);
				reason.response.headers = AxiosHeaders_default.from(reason.response.headers);
			}
		}
		return Promise.reject(reason);
	});
}
const VERSION = "1.13.2";
var validators$1 = {};
[
	"object",
	"boolean",
	"number",
	"function",
	"string",
	"symbol"
].forEach((type, i) => {
	validators$1[type] = function validator(thing) {
		return typeof thing === type || "a" + (i < 1 ? "n " : " ") + type;
	};
});
var deprecatedWarnings = {};
validators$1.transitional = function transitional(validator, version, message) {
	function formatMessage(opt, desc) {
		return "[Axios v" + VERSION + "] Transitional option '" + opt + "'" + desc + (message ? ". " + message : "");
	}
	return (value, opt, opts) => {
		if (validator === false) throw new AxiosError_default(formatMessage(opt, " has been removed" + (version ? " in " + version : "")), AxiosError_default.ERR_DEPRECATED);
		if (version && !deprecatedWarnings[opt]) {
			deprecatedWarnings[opt] = true;
			console.warn(formatMessage(opt, " has been deprecated since v" + version + " and will be removed in the near future"));
		}
		return validator ? validator(value, opt, opts) : true;
	};
};
validators$1.spelling = function spelling(correctSpelling) {
	return (value, opt) => {
		console.warn(`${opt} is likely a misspelling of ${correctSpelling}`);
		return true;
	};
};
function assertOptions(options, schema, allowUnknown) {
	if (typeof options !== "object") throw new AxiosError_default("options must be an object", AxiosError_default.ERR_BAD_OPTION_VALUE);
	const keys = Object.keys(options);
	let i = keys.length;
	while (i-- > 0) {
		const opt = keys[i];
		const validator = schema[opt];
		if (validator) {
			const value = options[opt];
			const result = value === void 0 || validator(value, opt, options);
			if (result !== true) throw new AxiosError_default("option " + opt + " must be " + result, AxiosError_default.ERR_BAD_OPTION_VALUE);
			continue;
		}
		if (allowUnknown !== true) throw new AxiosError_default("Unknown option " + opt, AxiosError_default.ERR_BAD_OPTION);
	}
}
var validator_default = {
	assertOptions,
	validators: validators$1
};
var validators = validator_default.validators;
var Axios = class {
	constructor(instanceConfig) {
		this.defaults = instanceConfig || {};
		this.interceptors = {
			request: new InterceptorManager_default(),
			response: new InterceptorManager_default()
		};
	}
	async request(configOrUrl, config) {
		try {
			return await this._request(configOrUrl, config);
		} catch (err) {
			if (err instanceof Error) {
				let dummy = {};
				Error.captureStackTrace ? Error.captureStackTrace(dummy) : dummy = /* @__PURE__ */ new Error();
				const stack = dummy.stack ? dummy.stack.replace(/^.+\n/, "") : "";
				try {
					if (!err.stack) err.stack = stack;
					else if (stack && !String(err.stack).endsWith(stack.replace(/^.+\n.+\n/, ""))) err.stack += "\n" + stack;
				} catch (e) {}
			}
			throw err;
		}
	}
	_request(configOrUrl, config) {
		if (typeof configOrUrl === "string") {
			config = config || {};
			config.url = configOrUrl;
		} else config = configOrUrl || {};
		config = mergeConfig(this.defaults, config);
		const { transitional, paramsSerializer, headers } = config;
		if (transitional !== void 0) validator_default.assertOptions(transitional, {
			silentJSONParsing: validators.transitional(validators.boolean),
			forcedJSONParsing: validators.transitional(validators.boolean),
			clarifyTimeoutError: validators.transitional(validators.boolean)
		}, false);
		if (paramsSerializer != null) if (utils_default.isFunction(paramsSerializer)) config.paramsSerializer = { serialize: paramsSerializer };
		else validator_default.assertOptions(paramsSerializer, {
			encode: validators.function,
			serialize: validators.function
		}, true);
		if (config.allowAbsoluteUrls !== void 0) {} else if (this.defaults.allowAbsoluteUrls !== void 0) config.allowAbsoluteUrls = this.defaults.allowAbsoluteUrls;
		else config.allowAbsoluteUrls = true;
		validator_default.assertOptions(config, {
			baseUrl: validators.spelling("baseURL"),
			withXsrfToken: validators.spelling("withXSRFToken")
		}, true);
		config.method = (config.method || this.defaults.method || "get").toLowerCase();
		let contextHeaders = headers && utils_default.merge(headers.common, headers[config.method]);
		headers && utils_default.forEach([
			"delete",
			"get",
			"head",
			"post",
			"put",
			"patch",
			"common"
		], (method) => {
			delete headers[method];
		});
		config.headers = AxiosHeaders_default.concat(contextHeaders, headers);
		const requestInterceptorChain = [];
		let synchronousRequestInterceptors = true;
		this.interceptors.request.forEach(function unshiftRequestInterceptors(interceptor) {
			if (typeof interceptor.runWhen === "function" && interceptor.runWhen(config) === false) return;
			synchronousRequestInterceptors = synchronousRequestInterceptors && interceptor.synchronous;
			requestInterceptorChain.unshift(interceptor.fulfilled, interceptor.rejected);
		});
		const responseInterceptorChain = [];
		this.interceptors.response.forEach(function pushResponseInterceptors(interceptor) {
			responseInterceptorChain.push(interceptor.fulfilled, interceptor.rejected);
		});
		let promise;
		let i = 0;
		let len;
		if (!synchronousRequestInterceptors) {
			const chain = [dispatchRequest.bind(this), void 0];
			chain.unshift(...requestInterceptorChain);
			chain.push(...responseInterceptorChain);
			len = chain.length;
			promise = Promise.resolve(config);
			while (i < len) promise = promise.then(chain[i++], chain[i++]);
			return promise;
		}
		len = requestInterceptorChain.length;
		let newConfig = config;
		while (i < len) {
			const onFulfilled = requestInterceptorChain[i++];
			const onRejected = requestInterceptorChain[i++];
			try {
				newConfig = onFulfilled(newConfig);
			} catch (error) {
				onRejected.call(this, error);
				break;
			}
		}
		try {
			promise = dispatchRequest.call(this, newConfig);
		} catch (error) {
			return Promise.reject(error);
		}
		i = 0;
		len = responseInterceptorChain.length;
		while (i < len) promise = promise.then(responseInterceptorChain[i++], responseInterceptorChain[i++]);
		return promise;
	}
	getUri(config) {
		config = mergeConfig(this.defaults, config);
		return buildURL(buildFullPath(config.baseURL, config.url, config.allowAbsoluteUrls), config.params, config.paramsSerializer);
	}
};
utils_default.forEach([
	"delete",
	"get",
	"head",
	"options"
], function forEachMethodNoData(method) {
	Axios.prototype[method] = function(url, config) {
		return this.request(mergeConfig(config || {}, {
			method,
			url,
			data: (config || {}).data
		}));
	};
});
utils_default.forEach([
	"post",
	"put",
	"patch"
], function forEachMethodWithData(method) {
	function generateHTTPMethod(isForm) {
		return function httpMethod(url, data, config) {
			return this.request(mergeConfig(config || {}, {
				method,
				headers: isForm ? { "Content-Type": "multipart/form-data" } : {},
				url,
				data
			}));
		};
	}
	Axios.prototype[method] = generateHTTPMethod();
	Axios.prototype[method + "Form"] = generateHTTPMethod(true);
});
var Axios_default = Axios;
var CancelToken_default = class CancelToken {
	constructor(executor) {
		if (typeof executor !== "function") throw new TypeError("executor must be a function.");
		let resolvePromise;
		this.promise = new Promise(function promiseExecutor(resolve) {
			resolvePromise = resolve;
		});
		const token = this;
		this.promise.then((cancel) => {
			if (!token._listeners) return;
			let i = token._listeners.length;
			while (i-- > 0) token._listeners[i](cancel);
			token._listeners = null;
		});
		this.promise.then = (onfulfilled) => {
			let _resolve;
			const promise = new Promise((resolve) => {
				token.subscribe(resolve);
				_resolve = resolve;
			}).then(onfulfilled);
			promise.cancel = function reject() {
				token.unsubscribe(_resolve);
			};
			return promise;
		};
		executor(function cancel(message, config, request) {
			if (token.reason) return;
			token.reason = new CanceledError_default(message, config, request);
			resolvePromise(token.reason);
		});
	}
	throwIfRequested() {
		if (this.reason) throw this.reason;
	}
	subscribe(listener) {
		if (this.reason) {
			listener(this.reason);
			return;
		}
		if (this._listeners) this._listeners.push(listener);
		else this._listeners = [listener];
	}
	unsubscribe(listener) {
		if (!this._listeners) return;
		const index = this._listeners.indexOf(listener);
		if (index !== -1) this._listeners.splice(index, 1);
	}
	toAbortSignal() {
		const controller = new AbortController();
		const abort = (err) => {
			controller.abort(err);
		};
		this.subscribe(abort);
		controller.signal.unsubscribe = () => this.unsubscribe(abort);
		return controller.signal;
	}
	static source() {
		let cancel;
		return {
			token: new CancelToken(function executor(c) {
				cancel = c;
			}),
			cancel
		};
	}
};
function spread(callback) {
	return function wrap(arr) {
		return callback.apply(null, arr);
	};
}
function isAxiosError(payload) {
	return utils_default.isObject(payload) && payload.isAxiosError === true;
}
var HttpStatusCode = {
	Continue: 100,
	SwitchingProtocols: 101,
	Processing: 102,
	EarlyHints: 103,
	Ok: 200,
	Created: 201,
	Accepted: 202,
	NonAuthoritativeInformation: 203,
	NoContent: 204,
	ResetContent: 205,
	PartialContent: 206,
	MultiStatus: 207,
	AlreadyReported: 208,
	ImUsed: 226,
	MultipleChoices: 300,
	MovedPermanently: 301,
	Found: 302,
	SeeOther: 303,
	NotModified: 304,
	UseProxy: 305,
	Unused: 306,
	TemporaryRedirect: 307,
	PermanentRedirect: 308,
	BadRequest: 400,
	Unauthorized: 401,
	PaymentRequired: 402,
	Forbidden: 403,
	NotFound: 404,
	MethodNotAllowed: 405,
	NotAcceptable: 406,
	ProxyAuthenticationRequired: 407,
	RequestTimeout: 408,
	Conflict: 409,
	Gone: 410,
	LengthRequired: 411,
	PreconditionFailed: 412,
	PayloadTooLarge: 413,
	UriTooLong: 414,
	UnsupportedMediaType: 415,
	RangeNotSatisfiable: 416,
	ExpectationFailed: 417,
	ImATeapot: 418,
	MisdirectedRequest: 421,
	UnprocessableEntity: 422,
	Locked: 423,
	FailedDependency: 424,
	TooEarly: 425,
	UpgradeRequired: 426,
	PreconditionRequired: 428,
	TooManyRequests: 429,
	RequestHeaderFieldsTooLarge: 431,
	UnavailableForLegalReasons: 451,
	InternalServerError: 500,
	NotImplemented: 501,
	BadGateway: 502,
	ServiceUnavailable: 503,
	GatewayTimeout: 504,
	HttpVersionNotSupported: 505,
	VariantAlsoNegotiates: 506,
	InsufficientStorage: 507,
	LoopDetected: 508,
	NotExtended: 510,
	NetworkAuthenticationRequired: 511,
	WebServerIsDown: 521,
	ConnectionTimedOut: 522,
	OriginIsUnreachable: 523,
	TimeoutOccurred: 524,
	SslHandshakeFailed: 525,
	InvalidSslCertificate: 526
};
Object.entries(HttpStatusCode).forEach(([key, value]) => {
	HttpStatusCode[value] = key;
});
var HttpStatusCode_default = HttpStatusCode;
function createInstance(defaultConfig) {
	const context = new Axios_default(defaultConfig);
	const instance = bind(Axios_default.prototype.request, context);
	utils_default.extend(instance, Axios_default.prototype, context, { allOwnKeys: true });
	utils_default.extend(instance, context, null, { allOwnKeys: true });
	instance.create = function create(instanceConfig) {
		return createInstance(mergeConfig(defaultConfig, instanceConfig));
	};
	return instance;
}
var axios = createInstance(defaults_default);
axios.Axios = Axios_default;
axios.CanceledError = CanceledError_default;
axios.CancelToken = CancelToken_default;
axios.isCancel = isCancel;
axios.VERSION = VERSION;
axios.toFormData = toFormData_default;
axios.AxiosError = AxiosError_default;
axios.Cancel = axios.CanceledError;
axios.all = function all(promises) {
	return Promise.all(promises);
};
axios.spread = spread;
axios.isAxiosError = isAxiosError;
axios.mergeConfig = mergeConfig;
axios.AxiosHeaders = AxiosHeaders_default;
axios.formToJSON = (thing) => formDataToJSON_default(utils_default.isHTMLForm(thing) ? new FormData(thing) : thing);
axios.getAdapter = adapters_default.getAdapter;
axios.HttpStatusCode = HttpStatusCode_default;
axios.default = axios;
var axios_default = axios;
const IsClient = () => {
	return browser;
};
var api = "https://genix-dev-api-1.un.pe/";
if (browser) {
	const host = window.location.host;
	if ((host.includes("localhost") || host.includes("127.0.0.1")) && host !== "localhost:8000") api = "http://localhost:3589/";
}
const Env = {
	appId: "genix",
	S3_URL: "https://d16qwm950j0pjf.cloudfront.net/",
	serviceWorker: "/sw.js",
	apiRoute: api,
	enviroment: "dev",
	counterID: 1,
	sideLayerSize: 0,
	fetchID: 1e3,
	imageWorker: null,
	ImageWorkerClass: null,
	zoneOffset: (/* @__PURE__ */ new Date()).getTimezoneOffset() * 60,
	dexieVersion: 1,
	cache: {},
	params: {
		fetchID: 1001,
		fetchProcesses: /* @__PURE__ */ new Map()
	},
	pendingRequests: [],
	API_ROUTES: { MAIN: api },
	screen: browser ? window.screen : {
		height: -1,
		width: -1
	},
	language: browser ? window.navigator?.language || "" : "-",
	deviceMemory: browser ? window.navigator?.deviceMemory || 0 : 0,
	throttleTimer: null,
	hostname: "",
	pathname: "",
	empresaID: 0,
	empresa: {},
	imageCounter: 1e4,
	clearAccesos: null,
	navigate: goto,
	history: { pushState: (data, unused, url) => {
		console.log("Es server!!", data, unused, url);
	} },
	getPathname: () => {
		if (browser) return window.location.pathname;
		return Env.pathname || "";
	},
	getEmpresaID: () => {
		if (!Env.empresaID) {
			const localEmpresaID = localStorage.getItem(Env.appId + "EmpresaID");
			if (localEmpresaID) {
				Env.empresaID = parseInt(localEmpresaID);
				return Env.empresaID;
			}
			let pathname = "";
			if (browser) pathname = document.head.querySelector(`meta[name="loc"]`)?.getAttribute("content") || "";
			if (!pathname) pathname = Env.getPathname();
			pathname = pathname.replace(".html", "");
			const paths = pathname.split("/").filter((x) => x);
			if (paths[1] && paths[1].includes("-")) {
				let empresaID = paths[1].split("-")[0];
				if (!isNaN(empresaID)) Env.empresaID = parseInt(empresaID);
			} else if (!isNaN(paths[1])) return Env.empresaID = parseInt(paths[1]);
		}
		return Env.empresaID;
	},
	loadEmpresaConfig: () => {
		if (Env.empresa.id) return;
		const empresaID = Env.getEmpresaID();
		if (empresaID) fetch(Env.S3_URL + `empresas/e-${empresaID}.json`).then((res) => res.json()).then((res) => {
			Env.empresa = res;
		});
		else console.warn("No se encontró la empresa-id:", empresaID);
	}
};
const LocalStorage = typeof window !== "undefined" ? window.localStorage : {
	getItem: (k) => {
		return "";
	},
	setItem: (k, v) => {
		return "";
	},
	removeItem: (k) => {
		return "";
	}
};
var tempID = parseInt(String(Math.floor(Date.now() / 1e3)).substring(4));
var serviceWorkerResolverMap = /* @__PURE__ */ new Map();
var successfulResponses = /* @__PURE__ */ new Set();
var nowTime = Date.now();
const doInitServiceWorker = () => {
	if (typeof navigator.serviceWorker === "undefined") {
		console.log("serviceWorker es undefined");
		return Promise.resolve(0);
	}
	return new Promise((resolve) => {
		navigator.serviceWorker.register(Env.serviceWorker, {
			scope: "/",
			type: "module"
		}).then(() => {
			console.log("Service Worker registrado!");
			return navigator.serviceWorker.ready;
		}).then(() => {
			console.log("Service Worker iniciado en: ", Date.now() - nowTime, "ms");
			navigator.serviceWorker.addEventListener("message", ({ data }) => {
				console.log("Mensaje del service worker::", data);
				if (data.__response__ === 5) setFetchProgress(data.bytes);
				else if (data.__response__ > 0 && data.__req__ > 0) {
					if (data.__response__ === 3) fetchEvent(data.__req__, 0);
					successfulResponses.add(data.__req__);
					if (serviceWorkerResolverMap.get(data.__req__)) {
						serviceWorkerResolverMap.get(data.__req__)?.(data);
						serviceWorkerResolverMap.delete(data.__req__);
					} else console.log("No se envió el request ID!");
				} else console.log("Mensaje del service worker no reconocido:", data);
			});
			resolve(1);
		});
	});
};
const sendServiceMessage = async (accion, content) => {
	const reqID = tempID;
	let info = "";
	if (accion === 3) {
		info = [
			content.route,
			content.cacheMode,
			reqID
		].join(" | ");
		if (content.cacheMode !== "offline") fetchEvent(reqID, { url: content.route });
	}
	tempID++;
	return new Promise((resolve) => {
		serviceWorkerResolverMap.set(reqID, resolve);
		const status = {
			id: 0,
			updated: 0,
			tryCount: 0
		};
		const doFetch = () => {
			status.id = 0;
			status.updated = 0;
			if (status.tryCount > 0) console.log(`Intentando fech por ${status.tryCount}º vez...`);
			let route = `${window.location.origin}/_sw_?accion=${accion}&req=${reqID}&env=${Env.enviroment}`;
			if (content.route) route += "&r=" + content.route.replaceAll("?", "_").replaceAll("&", "_");
			status.tryCount++;
			fetch(route, {
				method: "POST",
				body: JSON.stringify(content),
				headers: { "Content-Type": "application/json" }
			}).then((res) => res.json()).then((res) => {
				console.log(`respuesta del service worker (${info}):`, res);
				status.id = 2;
				status.updated = Date.now();
			}).catch((err) => {
				console.log(`Error en la respuesta, intentando (${info}):`, err);
				status.id = 3;
				status.updated = Date.now();
			});
		};
		const interval = setInterval(() => {
			if (successfulResponses.has(reqID)) {
				clearInterval(interval);
				return;
			}
			if (!status.updated) return;
			if (status.id === 3) doFetch();
			else if (Date.now() - status.updated >= 2e3) doFetch();
		}, 1e3);
		doFetch();
	});
};
var makeApiRoute = (route) => {
	const apiUrl = Env.apiRoute;
	return route.includes("://") ? route : apiUrl + route;
};
const fetchCache = async (args) => {
	args.routeParsed = makeApiRoute(args.route);
	args.__version__ = args.useCache?.ver || 1;
	console.log("fetching cache...", args);
	const response = await sendServiceMessage(3, args);
	console.log("cache response::", response);
	return response;
};
const fetchCacheParsed = async (args) => {
	const response = await fetchCache(args);
	if (response.error) {
		let errMessage = response.error;
		if (typeof response.error === "string") try {
			let errorJson = JSON.parse(response.error);
			errMessage = errorJson.error || JSON.stringify(errorJson);
		} catch (_) {}
		else errMessage = response.error.error || response.error;
		console.log("errMessage", errMessage);
		Notify.failure(errMessage);
		return null;
	}
	let content = response.content;
	if (args.cacheMode === "offline") {
		if (!content || response.isEmpty) return null;
	} else if (args.cacheMode === "updateOnly") {
		if (response.notUpdated) return null;
	}
	if (content?._default) return content._default;
	return content;
};
const unmarshall = (encoded) => {
	if (!Array.isArray(encoded) || encoded.length !== 2) return encoded;
	const [keysDef, content] = encoded;
	if (!Array.isArray(keysDef)) return encoded;
	const keysMap = {};
	for (const k of keysDef) {
		if (!Array.isArray(k)) continue;
		const typeId = k[0];
		const fields = {};
		let maxIndex = -1;
		for (let i = 1; i < k.length; i += 2) {
			const idx = k[i];
			fields[idx] = k[i + 1];
			if (idx > maxIndex) maxIndex = idx;
		}
		keysMap[typeId] = {
			fields,
			maxIndex
		};
	}
	let lastTypeId = null;
	const decode = (val) => {
		if (!Array.isArray(val) || val.length === 0) return val;
		const header = val[0];
		if (header === 1) {
			if (val.length < 2) return val;
			const refBlock = val[1];
			if (!Array.isArray(refBlock)) return val;
			const typeId = refBlock[0];
			const skipIndices = /* @__PURE__ */ new Set();
			for (let i = 1; i < refBlock.length; i++) skipIndices.add(refBlock[i]);
			lastTypeId = typeId;
			return populate(typeId, val.slice(2), skipIndices);
		}
		if (header === 0) {
			if (lastTypeId === null) return val;
			let skipIndices = /* @__PURE__ */ new Set();
			let valueStartIdx = 1;
			if (Array.isArray(val[1])) {
				const sub = val[1];
				let isSkipBlock = false;
				if (sub.length > 0) {
					const h = sub[0];
					if (typeof h === "number" && h !== 0 && h !== 1 && h !== 2 && h !== 3) isSkipBlock = true;
					else isSkipBlock = typeof h === "number" && h !== 0 && h !== 1 && h !== 2 && h !== 3;
				}
				if (isSkipBlock) {
					for (const s of sub) if (typeof s === "number") skipIndices.add(s);
					valueStartIdx = 2;
				}
			}
			return populate(lastTypeId, val.slice(valueStartIdx), skipIndices);
		}
		if (header === 2) {
			const result = [];
			for (let i = 1; i < val.length; i++) result.push(decode(val[i]));
			return result;
		}
		if (header === 3) {
			const result = {};
			for (let i = 1; i < val.length; i += 2) if (i + 1 < val.length) {
				const key = String(val[i]);
				result[key] = decode(val[i + 1]);
			}
			return result;
		}
		return val.map(decode);
	};
	const populate = (typeId, values, skipIndices) => {
		const typeDef = keysMap[typeId];
		if (!typeDef) return values;
		const { fields, maxIndex } = typeDef;
		const obj = {};
		let valIdx = 0;
		for (let i = 0; i <= maxIndex; i++) {
			if (skipIndices.has(i)) continue;
			if (valIdx >= values.length) break;
			const fieldName = fields[i];
			if (fieldName) obj[fieldName] = decode(values[valIdx]);
			valIdx++;
		}
		return obj;
	};
	return decode(content);
};
const makeRoute = (route) => {
	const apiUrl = Env.apiRoute;
	return route.includes("://") ? route : apiUrl + route;
};
const buildHeaders = (contentType) => {
	const cTs = { "json": "application/json" };
	const headers = {};
	if (contentType && cTs[contentType]) headers["Content-Type"] = cTs[contentType];
	headers["Authorization"] = `Bearer ${getToken()}`;
	return headers;
};
var extractError = (result) => {
	let errorJson;
	let errorString = "";
	if (typeof result === "string") {
		errorString = result.trim();
		if (errorString[0] === "{" || errorString[0] === "[") try {
			errorJson = JSON.parse(errorString);
		} catch {}
	} else errorJson = result;
	if (errorJson) {
		if (Array.isArray(errorJson)) errorJson = errorJson[0];
		if (errorJson.message || errorJson.error || errorJson.errorMessage) errorJson = errorJson.message || errorJson.error || errorJson.errorMessage;
		errorString = typeof errorJson === "string" ? errorJson : JSON.stringify(errorJson);
	}
	return errorString;
};
var checkErrorResponse = (result, status) => {
	if (!status.code || status.code !== 200 || result.errorMessage) {
		console.warn(result);
		Notify.failure(extractError(result));
		return false;
	} else return true;
};
var parsePreResponse = (res, status) => {
	const contentType = res.headers.get("content-type");
	if (res.status) {
		status.code = res.status;
		status.message = res.statusText;
	}
	if (res.status === 200) return res.json();
	else if (res.status === 401) {
		accessHelper.clearAccesos?.();
		console.warn("Error 401, la sesión ha expirado.");
		Notify.failure("La sesión ha expirado, vuelva a iniciar sesión.");
	} else if (res.status !== 200) if (!contentType || contentType.indexOf("/json") === -1) return res.text();
	else return res.json();
};
function parseResponseBody(res, props, status) {
	if (!res) res = "Hubo un error desconocido en el servidor";
	else if (typeof res === "string") try {
		res = JSON.parse(res);
	} catch {}
	if (!checkErrorResponse(res, status)) return false;
	if (props.successMessage) Notify.success(props.successMessage);
	return true;
}
var POST_PUT = (props, method) => {
	const data = props.data;
	if (typeof data !== "object") {
		const err = "The data provided is not a JSON";
		console.error(err);
		return Promise.reject(err);
	}
	const status = {
		code: 200,
		message: ""
	};
	const apiRoute = makeRoute(props.route);
	if ((props.refreshRoutes || []).length > 0) sendServiceMessage(24, { routes: props.refreshRoutes });
	return new Promise((resolve, reject) => {
		console.log(`Fetching ${method} : ` + props.route);
		fetch(apiRoute, {
			method,
			headers: buildHeaders("json"),
			body: JSON.stringify(data)
		}).then((res) => parsePreResponse(res, status)).then((res) => {
			res = unmarshall(res);
			parseResponseBody(res, props, status) ? resolve(res) : reject(res);
		}).catch((error) => {
			console.log("error::", error);
			if (props.errorMessage) Notify.failure(props.errorMessage);
			else Notify.failure(String(error));
			reject(error);
		});
	});
};
function POST(props) {
	return POST_PUT(props, "POST");
}
const POST_XMLHR = (props) => {
	const data = props.data;
	if (typeof data !== "object") {
		const err = "The data provided is not a JSON";
		console.error(err);
		return Promise.reject(err);
	}
	props.status = {
		code: 200,
		message: ""
	};
	const apiRoute = makeRoute(props.route);
	return new Promise((resolve, reject) => {
		axios_default.post(apiRoute, data, {
			onUploadProgress: props.onUploadProgress,
			headers: { "authorization": `Bearer ${getToken()}` }
		}).then((result) => {
			const data$1 = unmarshall(result.data);
			if (result.status !== 200) {
				let message = data$1.message || data$1.error || data$1.errorMessage;
				if (!message) message = String(data$1);
				Notify.failure(data$1);
				reject(data$1);
			} else resolve(data$1);
		}).catch((error) => {
			if (error.response && error.response.data) error = error.response.data;
			const message = error.message || error.error || error.errorMessage;
			Notify.failure(String(message || error));
			reject(error);
		});
	});
};
var progressTimeStart = 0;
var progressBytes = 0;
const setFetchProgress = (bytesLen) => {
	const nowTime$1 = Date.now();
	if (!progressBytes) progressTimeStart = nowTime$1;
	progressBytes += bytesLen;
	let mbps = 0;
	const kb = progressBytes / 1e3;
	const elapsed = nowTime$1 - progressTimeStart;
	if (elapsed > 50) mbps = kb / elapsed;
	let msg = `Descargando... ${formatN(kb)} kb`;
	if (mbps) {
		if (mbps > 10) mbps = 10;
		msg += ` (${formatN(mbps, 2)} MB/s)`;
	}
	const loadingMsgDiv = document.getElementById("NotiflixLoadingMessage");
	if (loadingMsgDiv) {
		let nextElement = loadingMsgDiv.nextElementSibling;
		if (!nextElement) {
			nextElement = document.createElement("div");
			nextElement.setAttribute("id", "NotifyProgressMessage");
			loadingMsgDiv.parentNode.insertBefore(nextElement, loadingMsgDiv.nextSibling);
		}
		nextElement.innerHTML = msg;
	}
};
function GET(props) {
	const status = {
		code: 200,
		message: ""
	};
	const routeParsed = makeRoute(props.route);
	if (props.useCache) {
		const args = {
			routeParsed,
			route: props.route,
			useCache: props.useCache,
			module: props.module || "a",
			headers: buildHeaders("json"),
			cacheMode: props.cacheMode
		};
		return new Promise((resolve, reject) => {
			fetchCacheParsed(args).then((cachedResponse) => {
				resolve(cachedResponse);
			}).catch((err) => {
				reject(err);
			});
		});
	} else return new Promise((resolve, reject) => {
		console.log("realizando fetch::", props);
		fetch(routeParsed, { headers: buildHeaders() }).then((res) => parsePreResponse(res, status)).then((res) => {
			res = unmarshall(res);
			return parseResponseBody(res, props, status) ? resolve(res) : reject(res);
		}).catch((error) => {
			console.warn(error);
			if (props.errorMessage) Notify.failure(props.errorMessage);
			reject(error);
		});
	});
}
var GetHandler = class {
	route = "";
	routeParsed = "";
	module = "a";
	keyID = "";
	keysIDs = {};
	useCache = void 0;
	headers = void 0;
	handler(e) {}
	isTest = false;
	Test() {
		setTimeout(() => {
			this.handler({ message: "Message 1" });
			setTimeout(() => {
				this.handler({ message: "Message 2" });
			}, 1e3);
		}, 1e3);
	}
	makeProps(cacheMode) {
		return {
			routeParsed: makeRoute(this.route),
			route: this.route,
			useCache: this.useCache,
			module: this.module,
			headers: buildHeaders("json"),
			cacheMode,
			keyID: this.keyID,
			keysIDs: this.keysIDs
		};
	}
	fetch() {
		if (!browser) return;
		if (this.route.length === 0) {
			Notify.failure("No se especificó el route en productos.");
			return;
		}
		fetchCacheParsed(this.makeProps("offline")).then((cachedResponse) => {
			if (cachedResponse) {
				delete cachedResponse.__version__;
				this.handler(cachedResponse);
			}
			return fetchCacheParsed(this.makeProps("refresh"));
		}).then((fetchedResponse) => {
			if (fetchedResponse) {
				delete fetchedResponse.__version__;
				this.handler(fetchedResponse);
			}
		});
	}
};
const sendUserLogin = async (data) => {
	let loginInfo;
	data.CipherKey = makeRamdomString(32);
	try {
		loginInfo = await POST({
			data,
			route: `p-user-login`,
			apiName: "MAIN",
			headers: { "Content-Type": "application/json" }
		});
	} catch (error) {
		console.log(error);
		return { error };
	}
	let userInfo = "";
	try {
		await accessHelper.parseAccesos(loginInfo, data.CipherKey);
		if (!accessHelper.checkAcceso(1)) Env.clearAccesos?.();
		else Env.navigate("/");
	} catch (error) {
		console.log("error encriptando::");
		console.log(error);
	}
	console.log(userInfo);
	return { result: loginInfo };
};
const reloadLogin = async () => {
	let loginInfo;
	const CipherKey = makeRamdomString(32);
	try {
		loginInfo = await GET({
			route: `reload-login?cipher-key=${CipherKey}`,
			headers: { "Content-Type": "application/json" }
		});
	} catch (error) {
		console.log(error);
		return { error };
	}
	let userInfo = "";
	try {
		await accessHelper.parseAccesos(loginInfo, CipherKey);
		if (!accessHelper.checkAcceso(1)) Env.clearAccesos?.();
	} catch (error) {
		console.log("error encriptando::");
		console.log(error);
	}
	console.log(userInfo);
	return { result: loginInfo };
};
var TOKEN_REFRESH_THRESHOLD = 2400;
var TOKEN_CHECK_INTERVAL = 240;
var REFRESH_LOCK_DURATION = 30;
var REFRESH_LOCK_KEY = Env.appId + "TokenRefreshLock";
const getToken = (noError) => {
	const userToken = LocalStorage.getItem(Env.appId + "UserToken");
	const expTime = parseInt(LocalStorage.getItem(Env.appId + "TokenExpTime") || "0");
	const nowTime$1 = Math.floor(Date.now() / 1e3);
	if (!userToken) {
		console.error("No se encontró la data del usuario. ¿Está logeado?:", Env.appId);
		return "";
	} else if (!expTime || nowTime$1 > expTime) {
		if (!noError) {
			Notify.failure("La sesión ha expirado, vuelva a iniciar sesión.");
			Env.clearAccesos?.();
		}
		return "";
	}
	if (expTime - nowTime$1 < 900) throttle$1(() => {
		Notify.warning(`La sesión expirará en 15 minutos`);
	}, 20);
	else if (expTime - nowTime$1 < 300) throttle$1(() => {
		Notify.warning(`La sesión expirará en 5 minutos`);
	}, 20);
	return userToken || "";
};
const checkIsLogin = () => {
	if (!IsClient()) return 0;
	else if (IsClient() && !!getToken(true)) return 2;
	else return 3;
};
var acquireRefreshLock = () => {
	if (!IsClient()) return false;
	const lockTime = parseInt(LocalStorage.getItem(REFRESH_LOCK_KEY) || "0");
	const nowUnix = Math.floor(Date.now() / 1e3);
	if (lockTime && nowUnix - lockTime < REFRESH_LOCK_DURATION) {
		console.log("Token refresh already in progress in another tab");
		return false;
	}
	LocalStorage.setItem(REFRESH_LOCK_KEY, String(nowUnix));
	return true;
};
var shouldRefreshToken = () => {
	const tokenCreated = parseInt(LocalStorage.getItem(Env.appId + "TokenCreated") || "0");
	const tokenAge = Math.floor(Date.now() / 1e3) - tokenCreated;
	return tokenCreated > 0 && tokenAge >= TOKEN_REFRESH_THRESHOLD;
};
var tokenRefreshInterval = null;
var checkAndRefreshToken = async () => {
	if (!IsClient()) return;
	if (!getToken(true)) {
		stopTokenRefreshCheck();
		return;
	}
	if (!shouldRefreshToken() || !acquireRefreshLock()) return;
	console.log("Token refresh initiated - token is older than 40 minutes");
	try {
		await reloadLogin();
		console.log("Token refreshed successfully");
	} catch (error) {
		console.error("Error refreshing token:", error);
	} finally {
		LocalStorage.removeItem(REFRESH_LOCK_KEY);
	}
};
const startTokenRefreshCheck = () => {
	if (!IsClient()) return;
	if (tokenRefreshInterval !== null) clearInterval(tokenRefreshInterval);
	tokenRefreshInterval = window.setInterval(checkAndRefreshToken, TOKEN_CHECK_INTERVAL * 1e3);
	console.log("Token refresh check started - will check every 4 minutes");
};
const stopTokenRefreshCheck = () => {
	if (tokenRefreshInterval !== null) {
		clearInterval(tokenRefreshInterval);
		tokenRefreshInterval = null;
		console.log("Token refresh check stopped");
	}
};
Env.clearAccesos = () => {
	if (!IsClient) return;
	stopTokenRefreshCheck();
	LocalStorage.removeItem(Env.appId + "Accesos");
	LocalStorage.removeItem(Env.appId + "UserInfo");
	LocalStorage.removeItem(Env.appId + "UserToken");
	LocalStorage.removeItem(Env.appId + "TokenExpTime");
	LocalStorage.removeItem(Env.appId + "TokenCreated");
	LocalStorage.removeItem(REFRESH_LOCK_KEY);
	Env.navigate("/login");
};
const initTokenRefreshCheck = () => {
	if (!IsClient()) return;
	const hasToken = getToken(true);
	const tokenCreated = LocalStorage.getItem(Env.appId + "TokenCreated");
	if (hasToken && tokenCreated) startTokenRefreshCheck();
};
if (IsClient()) setTimeout(initTokenRefreshCheck, 1 * 1e3);
var AccessHelper = class {
	constructor() {
		const b32l = [];
		for (let i = 32; i < 36; i++) b32l.push(i.toString(36));
		this.#b32l = b32l;
		this.#b32ls = b32l.join(",");
		this.#setUserInfo();
	}
	#b32l = [];
	#b32ls = "";
	#avoidCheckSum = false;
	#accesos = "";
	#cachedResults = /* @__PURE__ */ new Map();
	#userInfo = null;
	#setUserInfo() {
		const userInfoJson = LocalStorage?.getItem(Env.appId + "UserInfo");
		if (!userInfoJson) return;
		this.#userInfo = JSON.parse(userInfoJson);
	}
	clearAccesos = Env.clearAccesos;
	getUserInfo() {
		return this.#userInfo;
	}
	setUserInfo(userInfo) {
		this.#userInfo = userInfo;
		LocalStorage.setItem(Env.appId + "UserInfo", JSON.stringify(userInfo));
	}
	async parseAccesos(login, cipherKey) {
		debugger;
		const userInfoStr = await decrypt(login.UserInfo, cipherKey);
		const userInfo = JSON.parse(userInfoStr);
		const rolesIDsParsed = (userInfo.rolesIDs || []).map((x) => x * 10 + 8);
		const accesosIDs = (userInfo.accesosIDs || []).concat(rolesIDsParsed);
		const UnixTime = Math.floor(Date.now() / 1e3);
		LocalStorage.setItem(Env.appId + "TokenCreated", String(UnixTime));
		LocalStorage.setItem(Env.appId + "UserInfo", JSON.stringify(userInfo));
		LocalStorage.setItem(Env.appId + "UserToken", login.UserToken);
		LocalStorage.setItem(Env.appId + "TokenExpTime", String(login.TokenExpTime));
		LocalStorage.setItem(Env.appId + "EmpresaID", String(login.EmpresaID));
		this.#setUserInfo();
		const b32l = this.#b32l;
		let parsedAccesos = b32l[b32l.length - 1];
		let i = 0;
		for (let e of accesosIDs.map((x) => x.toString(32))) {
			if (i > 3) i = 0;
			parsedAccesos += e;
			parsedAccesos += b32l[i];
			i++;
		}
		const hash = checksum(parsedAccesos);
		const hashParsed = `${hash.substring(0, 2)}${parsedAccesos}${hash.substring(2, 4)}`;
		LocalStorage.setItem(Env.appId + "Accesos", hashParsed);
		debugger;
		startTokenRefreshCheck();
	}
	checkAcceso(accesoID, nivel) {
		nivel = nivel || 1;
		const b32ls = this.#b32ls;
		if (this.#accesos === "") this.#accesos = LocalStorage.getItem(Env.appId + "Accesos") || "";
		if (!this.#accesos) return false;
		if (!this.#avoidCheckSum) {
			if (!this.#checkAccesosCheckSum()) {
				console.warn("Los accesos han sido modificados.");
				return false;
			}
			this.#avoidCheckSum = true;
			setTimeout(() => {
				this.#avoidCheckSum = false;
			}, 50);
		}
		const check = (niveles) => {
			for (let nivel$1 of niveles) {
				const code = (accesoID * 10 + nivel$1).toString(32);
				if (!this.#cachedResults.has(code)) {
					const hasAccess$1 = (/* @__PURE__ */ new RegExp(`[${b32ls}]${code}[${b32ls}]`)).test(this.#accesos);
					this.#cachedResults.set(code, hasAccess$1);
				}
				const hasAccess = this.#cachedResults.get(code);
				if (hasAccess) return hasAccess;
			}
			return false;
		};
		if (!nivel || nivel === 1) return check([1, 7]);
		else if (nivel === 2) return check([2, 7]);
		else if (nivel === 3) return check([3, 7]);
		else if (nivel === 4) return check([4, 7]);
		else if (nivel === 8) return check([8]);
		else return check([7]);
	}
	checkRol(roleID) {
		return this.checkAcceso(roleID, 8);
	}
	#checkAccesosCheckSum() {
		const accesos = this.#accesos;
		const accesosInternal = accesos.substring(2, accesos.length - 2);
		return accesos.substring(0, 2) + accesos.substring(accesos.length - 2) === checksum(accesosInternal);
	}
};
const checksum = (string) => {
	let seed = 888888;
	for (let i = 0; i < string.length; i++) {
		const in3 = i % 1e3;
		const code = string[i].charCodeAt(0);
		const ld = Math.abs(seed - code + in3) % 10;
		if (ld > 6) seed += (code + in3) * ld + ld;
		else if (ld > 3) seed -= (code - in3) * ld - in3;
		else {
			seed += Math.abs((code - in3) * (ld + 1)) - ld * (i % 10);
			if (seed >= 1e6) seed = Math.abs(seed) % 1e5;
		}
	}
	const rs = String(seed % 1e3);
	for (let i = 0; i < rs.length; i++) seed += Math.pow(parseInt(rs[0]), 6 - i);
	let seedT = seed.toString(32).split("").reverse().join("");
	if (seedT.length > 4) seedT = seedT.substring(0, 4);
	else if (seedT.length < 4) for (let i = seedT.length; i < 4; i++) seedT += String(4 - i);
	return seedT;
};
const accessHelper = new AccessHelper();
var SvelteMap = class extends Map {
	#sources = /* @__PURE__ */ new Map();
	#version = /* @__PURE__ */ state(0);
	#size = /* @__PURE__ */ state(0);
	#update_version = update_version || -1;
	constructor(value) {
		super();
		if (value) {
			for (var [key, v] of value) super.set(key, v);
			this.#size.v = super.size;
		}
	}
	#source(value) {
		return update_version === this.#update_version ? /* @__PURE__ */ state(value) : source(value);
	}
	has(key) {
		var sources = this.#sources;
		var s = sources.get(key);
		if (s === void 0) if (super.get(key) !== void 0) {
			s = this.#source(0);
			sources.set(key, s);
		} else {
			get(this.#version);
			return false;
		}
		get(s);
		return true;
	}
	forEach(callbackfn, this_arg) {
		this.#read_all();
		super.forEach(callbackfn, this_arg);
	}
	get(key) {
		var sources = this.#sources;
		var s = sources.get(key);
		if (s === void 0) if (super.get(key) !== void 0) {
			s = this.#source(0);
			sources.set(key, s);
		} else {
			get(this.#version);
			return;
		}
		get(s);
		return super.get(key);
	}
	set(key, value) {
		var sources = this.#sources;
		var s = sources.get(key);
		var prev_res = super.get(key);
		var res = super.set(key, value);
		var version = this.#version;
		if (s === void 0) {
			s = this.#source(0);
			sources.set(key, s);
			set(this.#size, super.size);
			increment(version);
		} else if (prev_res !== value) {
			increment(s);
			var v_reactions = version.reactions === null ? null : new Set(version.reactions);
			if (v_reactions === null || !s.reactions?.every((r) => v_reactions.has(r))) increment(version);
		}
		return res;
	}
	delete(key) {
		var sources = this.#sources;
		var s = sources.get(key);
		var res = super.delete(key);
		if (s !== void 0) {
			sources.delete(key);
			set(this.#size, super.size);
			set(s, -1);
			increment(this.#version);
		}
		return res;
	}
	clear() {
		if (super.size === 0) return;
		super.clear();
		var sources = this.#sources;
		set(this.#size, 0);
		for (var s of sources.values()) set(s, -1);
		increment(this.#version);
		sources.clear();
	}
	#read_all() {
		get(this.#version);
		var sources = this.#sources;
		if (this.#size.v !== sources.size) {
			for (var key of super.keys()) if (!sources.has(key)) {
				var s = this.#source(0);
				sources.set(key, s);
			}
		}
		for ([, s] of this.#sources) get(s);
	}
	keys() {
		get(this.#version);
		return super.keys();
	}
	values() {
		this.#read_all();
		return super.values();
	}
	entries() {
		this.#read_all();
		return super.entries();
	}
	[Symbol.iterator]() {
		return this.entries();
	}
	get size() {
		get(this.#size);
		return super.size;
	}
};
URLSearchParams, Symbol.iterator;
const getDeviceType = () => {
	let view = 1;
	if (!browser) return view;
	if (window.innerWidth < 740) view = 3;
	else if (window.innerWidth < 1140) view = 2;
	return view;
};
const Core = proxy({
	module: { menus: [] },
	openSearchLayer: 0,
	deviceType: getDeviceType(),
	mobileMenuOpen: 0,
	popoverShowID: 0,
	showSideLayer: 0,
	isLoading: 1,
	pageTitle: "",
	pageOptions: [],
	pageOptionSelected: 1,
	showMobileSearchLayer: null,
	toggleMobileMenu: () => {},
	openSideLayer: (layerId) => {
		Core.showSideLayer = layerId;
	},
	hideSideLayer: () => {
		Core.showSideLayer = 0;
	},
	openModal: (id) => {
		if (!openModals.includes(id)) openModals.push(id);
	},
	closeModal: (id) => {
		const index = openModals.indexOf(id);
		if (index > -1) openModals.splice(index, 1);
	}
});
const WeakSearchRef = /* @__PURE__ */ new WeakMap();
const fetchOnCourse = proxy(new SvelteMap());
const fetchEvent = (fetchID, props) => {
	if (fetchID === 0) {
		Env.fetchID++;
		return Env.fetchID;
	}
	if (props === 0) fetchOnCourse.delete(fetchID);
	else fetchOnCourse.set(fetchID, props);
};
const openModals = proxy([]);
const openModal = (id) => {
	if (!openModals.includes(id)) openModals.push(id);
};
const closeModal = (id) => {
	const index = openModals.indexOf(id);
	if (index > -1) openModals.splice(index, 1);
};
const closeAllModals = () => {
	openModals.length = 0;
};
const imagesToUpload = /* @__PURE__ */ new Map();
export { doInitServiceWorker as _, fetchOnCourse as a, formatN as b, openModal as c, checkIsLogin as d, sendUserLogin as f, POST_XMLHR as g, POST as h, closeModal as i, openModals as l, GetHandler as m, WeakSearchRef as n, getDeviceType as o, GET as p, closeAllModals as r, imagesToUpload as s, Core as t, accessHelper as u, sendServiceMessage as v, Env as y };
