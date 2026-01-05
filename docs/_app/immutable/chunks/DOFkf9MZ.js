import { Tt as __toESM } from "./CTB4HzdN.js";
import { y as Env } from "./BwrZ3UQO.js";
import { o as require_notiflix_aio_3_2_8_min } from "./DwKmqXYU.js";
const { Notify, Loading, Confirm } = (/* @__PURE__ */ __toESM(require_notiflix_aio_3_2_8_min(), 1)).default;
var throttleTimer;
if (typeof window !== "undefined") Loading.init({ zindex: 400 });
const ConfirmWarn = (a, b, c, d, e, f) => {
	Confirm.init({
		fontFamily: "main",
		messageFontSize: "15px",
		titleColor: "#db3030",
		titleFontSize: "18px",
		messageColor: "#1e1e1e",
		okButtonColor: "#f8f8f8",
		okButtonBackground: "#f35c5c"
	});
	Confirm.show(a, b, c, d, e, f);
};
const throttle = (func, delay) => {
	if (throttleTimer) clearTimeout(throttleTimer);
	throttleTimer = setTimeout(() => {
		func();
		throttleTimer = null;
	}, delay);
};
const highlString = (phrase, words) => {
	if (typeof phrase !== "string") {
		console.error("no es string");
		console.log(phrase);
		return [{ text: "!" }];
	}
	const arr = [{ text: phrase }];
	if (!words || words.length === 0) return arr;
	for (let word of words) {
		if (word.length < 2) continue;
		for (let i = 0; i < arr.length; i++) {
			const str = arr[i].text;
			if (typeof str !== "string") continue;
			const idx = str.toLowerCase().indexOf(word);
			if (idx !== -1) {
				const ini = str.slice(0, idx);
				const middle = str.slice(idx, idx + word.length);
				const fin = str.slice(idx + word.length);
				const splited = [
					{ text: ini },
					{
						text: middle,
						highl: true
					},
					{ text: fin }
				].filter((x) => x.text);
				arr.splice(i, 1, ...splited);
				if (arr.length > 40) return arr.filter((x) => x);
				continue;
			}
		}
	}
	console.log("words 111:", arr.filter((x) => x), "|", phrase, words);
	return arr.filter((x) => x);
};
const parseSVG = (svgContent) => {
	return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svgContent)}`;
};
function include(e, h) {
	if (h && typeof h === "string") h = h.split(" ").filter((x) => x.length > 0);
	if (!h || h === "undefined" || h.length === 0) return true;
	else if (h.length === 1) return e.includes(h[0]);
	else if (h.length === 2) return e.includes(h[0]) && e.includes(h[1]);
	else if (h.length === 3) return e.includes(h[0]) && e.includes(h[1]) && e.includes(h[2]);
	else if (h.length === 4) return e.includes(h[0]) && e.includes(h[1]) && e.includes(h[2]) && e.includes(h[3]);
	else if (h.length === 5) return e.includes(h[0]) && e.includes(h[1]) && e.includes(h[2]) && e.includes(h[3]) && e.includes(h[4]);
	else return e.includes(h[0]) && e.includes(h[1]) && e.includes(h[2]) && e.includes(h[3]) && e.includes(h[4]) && e.includes(h[5]);
}
var pendingWorkerRequests = /* @__PURE__ */ new Map();
var workerPool = /* @__PURE__ */ new Map();
var MAX_WORKERS = 4;
var setWorkerCommunication = (wi) => {
	wi.worker.onmessage = ({ data }) => {
		wi.tasks = Math.max(0, wi.tasks - 1);
		const { id, dataUrl, error } = data;
		const request = pendingWorkerRequests.get(id);
		if (request) {
			clearTimeout(request.timeout);
			pendingWorkerRequests.delete(id);
			if (error) {
				console.error(`❌ Worker error for request ${id}:`, error);
				request.reject(error);
			} else {
				console.log(`📨 Received message from worker for request ${id} | len: ${dataUrl.length}`);
				request.resolve(dataUrl || "");
			}
		}
	};
	wi.worker.onerror = (error) => {
		wi.tasks = Math.max(0, wi.tasks - 1);
		console.error("❌ Worker error:", error);
	};
};
var getBestWorker = () => {
	if (Env.imageWorker && !workerPool.has("preloaded")) {
		const wi = {
			worker: Env.imageWorker,
			tasks: 0
		};
		setWorkerCommunication(wi);
		workerPool.set("preloaded", wi);
	}
	let best = null;
	for (const wi of workerPool.values()) {
		if (wi.tasks === 0) {
			best = wi;
			break;
		}
		if (!best || wi.tasks < best.tasks) best = wi;
	}
	if ((!best || best.tasks > 0) && workerPool.size < MAX_WORKERS && Env.ImageWorkerClass) {
		console.log(`🚀 Spawning new worker (Pool size: ${workerPool.size + 1})`);
		const wi = {
			worker: new Env.ImageWorkerClass(),
			tasks: 0
		};
		setWorkerCommunication(wi);
		const id = Math.random();
		workerPool.set(id, wi);
		best = wi;
	}
	if (best) {
		best.tasks++;
		return best.worker;
	}
	return Env.imageWorker;
};
const fileToImage = (blob, resolution, fileType) => {
	if (resolution > 2e3) {
		Notify.failure("2mpx is max resolution for image conversion.");
		return Promise.resolve("");
	}
	const useJpeg = fileType === "jpg";
	const useAvif = fileType === "avif";
	const worker = getBestWorker();
	console.log("📸 fileToImage called with:", {
		blobType: blob.type,
		blobSize: blob.size,
		resolution,
		useJpeg,
		useAvif,
		poolSize: workerPool.size,
		workerExists: !!worker
	});
	if (!worker) {
		console.error("❌ No image worker available!");
		return Promise.reject("Image worker not available");
	}
	return new Promise((resolve, reject) => {
		const id = Math.floor(Math.random() * 1e6);
		const timeout = setTimeout(() => {
			console.error(`⏱️ Worker timeout - no response after 8 seconds (id=${id}, r=${resolution})`);
			pendingWorkerRequests.delete(id);
			reject("Error al procesar la imagen. (superó los 8 segundos.)");
		}, 8e3);
		pendingWorkerRequests.set(id, {
			resolve,
			reject,
			timeout
		});
		console.log(`📤 Posting message to worker (id=${id})...`);
		worker.postMessage({
			id,
			blob,
			resolution,
			useJpeg,
			useAvif
		});
	});
};
var mesesMap = new Map([
	["01", {
		es: "ENE",
		en: "JAN"
	}],
	["02", {
		es: "FEB",
		en: "FEB"
	}],
	["03", {
		es: "MAR",
		en: "MAR"
	}],
	["04", {
		es: "ABR",
		en: "APR"
	}],
	["05", {
		es: "MAY",
		en: "MAY"
	}],
	["06", {
		es: "JUN",
		en: "JUN"
	}],
	["07", {
		es: "JUL",
		en: "JUL"
	}],
	["08", {
		es: "AGO",
		en: "AUG"
	}],
	["09", {
		es: "SEP",
		en: "SEP"
	}],
	["10", {
		es: "OCT",
		en: "OCT"
	}],
	["11", {
		es: "NOV",
		en: "NOV"
	}],
	["12", {
		es: "DIC",
		en: "DEC"
	}]
]);
const formatTime = (date, layout) => {
	let d;
	if (!date) d = /* @__PURE__ */ new Date();
	else if (typeof date === "number") {
		if (date < 3e4) date = date * 1e3 * 86400 + 36e6;
		else if (date < 8e8) date = 10 ** 9 + date * 2;
		if (date < 18e10) date = date * 1e3;
		d = new Date(date);
	} else if (typeof date === "object" && date.constructor === Date) d = date;
	else if (typeof date === "string" && date.length === 8) {
		const year$1 = parseInt(date.substring(0, 4));
		const month = parseInt(date.substring(4, 6)) - 1;
		const day = parseInt(date.substring(6, 8));
		d = new Date(year$1, month, day);
	} else if (typeof date === "string") {
		if (date.includes("T")) date = date.replace("T", " ");
		if (date.includes("Z") && date.includes(".")) {
			const idx1 = date.lastIndexOf(".");
			date = date.substring(0, idx1);
		}
		const portions = date.split(" ");
		let day = portions[0];
		const regex1 = /[0-9]{1,2}(\.|-|\/)[0-9]{1,2}(\.|-|\/)[0-9]{4}/g;
		const regex2 = /[0-9]{4}(\.|-|\/)[0-9]{1,2}(\.|-|\/)[0-9]{1,2}/g;
		const r1 = regex1.test(day);
		const r2 = r1 ? void 0 : regex2.test(day);
		if (r1 || r2) {
			for (const s of [
				"/",
				"-",
				"."
			]) if (day.includes(s)) {
				let parsed = day.split(s);
				if (r1) parsed.reverse();
				const parsedStr = parsed.join("-") + "T" + (portions[1] || "12:00:00");
				d = new Date(parsedStr);
				if (!d.getTime) return null;
			}
		} else return null;
	}
	if (!d || !(d instanceof Date) || !d.getTime) return layout ? "" : null;
	const _dia = d.getDate();
	if (isNaN(_dia)) return !layout ? null : "";
	const dia = _dia < 10 ? "0" + _dia : String(_dia);
	const _mes = d.getMonth() + 1;
	const mes = _mes < 10 ? "0" + _mes : String(_mes);
	const year = String(d.getFullYear());
	if (!layout) return d;
	let fechaStr = "";
	for (const sec of layout) switch (sec) {
		case "y":
			fechaStr += year.substring(2, 4);
			break;
		case "Y":
			fechaStr += year;
			break;
		case "m":
			fechaStr += mes;
			break;
		case "M":
			fechaStr += mesesMap.get(mes)?.es || "?";
			break;
		case "d":
			fechaStr += dia;
			break;
		case "h": {
			let hora = d.getHours();
			if (hora < 10) hora = "0" + hora;
			fechaStr += hora;
			break;
		}
		case "n": {
			let min = d.getMinutes();
			if (min < 10) min = "0" + min;
			fechaStr += min;
			break;
		}
		default: fechaStr += sec;
	}
	return fechaStr;
};
export { formatTime as a, parseSVG as c, fileToImage as i, throttle as l, Loading as n, highlString as o, Notify as r, include as s, ConfirmWarn as t };
