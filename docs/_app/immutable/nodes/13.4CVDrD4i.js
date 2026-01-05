import { G as user_effect, L as delegate, M as append, P as from_html, Q as sibling, S as set_class, W as template_effect, X as child, _t as reset, at as state, j as set_text, lt as pop, m as bind_value, q as remove_textarea_child, rt as set, ut as push, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/2yAUi6s0.js";
var t = "undefined" != typeof self ? self : {};
function e(e$1, n$1) {
	t: {
		for (var r$1 = ["CLOSURE_FLAGS"], i$1 = t, o$1 = 0; o$1 < r$1.length; o$1++) if (null == (i$1 = i$1[r$1[o$1]])) {
			r$1 = null;
			break t;
		}
		r$1 = i$1;
	}
	return null != (e$1 = r$1 && r$1[e$1]) ? e$1 : n$1;
}
var n;
var r = "undefined" != typeof TextEncoder;
function i(t$1) {
	if (r) t$1 = (n ||= new TextEncoder()).encode(t$1);
	else {
		let n$1 = 0;
		const r$1 = new Uint8Array(3 * t$1.length);
		for (let i$1 = 0; i$1 < t$1.length; i$1++) {
			var e$1 = t$1.charCodeAt(i$1);
			if (e$1 < 128) r$1[n$1++] = e$1;
			else {
				if (e$1 < 2048) r$1[n$1++] = e$1 >> 6 | 192;
				else {
					if (e$1 >= 55296 && e$1 <= 57343) {
						if (e$1 <= 56319 && i$1 < t$1.length) {
							const o$1 = t$1.charCodeAt(++i$1);
							if (o$1 >= 56320 && o$1 <= 57343) {
								e$1 = 1024 * (e$1 - 55296) + o$1 - 56320 + 65536, r$1[n$1++] = e$1 >> 18 | 240, r$1[n$1++] = e$1 >> 12 & 63 | 128, r$1[n$1++] = e$1 >> 6 & 63 | 128, r$1[n$1++] = 63 & e$1 | 128;
								continue;
							}
							i$1--;
						}
						e$1 = 65533;
					}
					r$1[n$1++] = e$1 >> 12 | 224, r$1[n$1++] = e$1 >> 6 & 63 | 128;
				}
				r$1[n$1++] = 63 & e$1 | 128;
			}
		}
		t$1 = n$1 === r$1.length ? r$1 : r$1.subarray(0, n$1);
	}
	return t$1;
}
var o, s = e(610401301, !1), a = e(748402147, e(1, !0));
function u() {
	var e$1 = t.navigator;
	return e$1 && (e$1 = e$1.userAgent) ? e$1 : "";
}
var c = t.navigator;
o = c && c.userAgentData || null;
var l = {}, h = null;
function f(t$1) {
	const e$1 = t$1.length;
	let n$1 = 3 * e$1 / 4;
	n$1 % 3 ? n$1 = Math.floor(n$1) : -1 != "=.".indexOf(t$1[e$1 - 1]) && (n$1 = -1 != "=.".indexOf(t$1[e$1 - 2]) ? n$1 - 2 : n$1 - 1);
	const r$1 = new Uint8Array(n$1);
	let i$1 = 0;
	return function(t$2, e$2) {
		function n$2(e$3) {
			for (; r$2 < t$2.length;) {
				const e$4 = t$2.charAt(r$2++), n$3 = h[e$4];
				if (null != n$3) return n$3;
				if (!/^[\s\xa0]*$/.test(e$4)) throw Error("Unknown base64 encoding at char: " + e$4);
			}
			return e$3;
		}
		d();
		let r$2 = 0;
		for (;;) {
			const t$3 = n$2(-1), r$3 = n$2(0), i$2 = n$2(64), o$1 = n$2(64);
			if (64 === o$1 && -1 === t$3) break;
			e$2(t$3 << 2 | r$3 >> 4), 64 != i$2 && (e$2(r$3 << 4 & 240 | i$2 >> 2), 64 != o$1 && e$2(i$2 << 6 & 192 | o$1));
		}
	}(t$1, (function(t$2) {
		r$1[i$1++] = t$2;
	})), i$1 !== n$1 ? r$1.subarray(0, i$1) : r$1;
}
function d() {
	if (!h) {
		h = {};
		var t$1 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789".split(""), e$1 = [
			"+/=",
			"+/",
			"-_=",
			"-_.",
			"-_"
		];
		for (let n$1 = 0; n$1 < 5; n$1++) {
			const r$1 = t$1.concat(e$1[n$1].split(""));
			l[n$1] = r$1;
			for (let t$2 = 0; t$2 < r$1.length; t$2++) {
				const e$2 = r$1[t$2];
				void 0 === h[e$2] && (h[e$2] = t$2);
			}
		}
	}
}
var p = "undefined" != typeof Uint8Array, m = !(!(s && o && o.brands.length > 0) && (-1 != u().indexOf("Trident") || -1 != u().indexOf("MSIE"))) && "function" == typeof btoa;
var g = /[-_.]/g, _ = {
	"-": "+",
	_: "/",
	".": "="
};
function y(t$1) {
	return _[t$1] || "";
}
function b(t$1) {
	if (!m) return f(t$1);
	t$1 = g.test(t$1) ? t$1.replace(g, y) : t$1, t$1 = atob(t$1);
	const e$1 = new Uint8Array(t$1.length);
	for (let n$1 = 0; n$1 < t$1.length; n$1++) e$1[n$1] = t$1.charCodeAt(n$1);
	return e$1;
}
function w(t$1) {
	return p && null != t$1 && t$1 instanceof Uint8Array;
}
var v = {};
function S() {
	return A ||= new E(null, v);
}
var E = class {
	constructor(t$1, e$1) {
		if (T(e$1), this.i = t$1, null != t$1 && 0 === t$1.length) throw Error("ByteString should be constructed with non-empty values");
	}
};
var A, I;
function T(t$1) {
	if (t$1 !== v) throw Error("illegal external caller");
}
function P(t$1, e$1) {
	t$1.__closure__error__context__984382 || (t$1.__closure__error__context__984382 = {}), t$1.__closure__error__context__984382.severity = e$1;
}
function L(t$1) {
	return P(t$1 = Error(t$1), "warning"), t$1;
}
function O(e$1, n$1) {
	if (null != e$1) {
		var r$1 = I ??= {}, i$1 = r$1[e$1] || 0;
		i$1 >= n$1 || (r$1[e$1] = i$1 + 1, P(e$1 = Error(), "incident"), function(e$2) {
			t.setTimeout((() => {
				throw e$2;
			}), 0);
		}(e$1));
	}
}
var j = "function" == typeof Symbol && "symbol" == typeof Symbol();
function k(t$1, e$1, n$1 = !1) {
	return "function" == typeof Symbol && "symbol" == typeof Symbol() ? n$1 && Symbol.for && t$1 ? Symbol.for(t$1) : null != t$1 ? Symbol(t$1) : Symbol() : e$1;
}
var x = k("jas", void 0, !0), U = k(void 0, "1oa"), B = k(void 0, "0ubsb"), N = k(void 0, "0actk"), F = k("m_m", "oa", !0);
var R = { ga: {
	value: 0,
	configurable: !0,
	writable: !0,
	enumerable: !1
} }, D = Object.defineProperties, M = j ? x : "ga";
var V;
var C = [];
function G(t$1, e$1) {
	j || M in t$1 || D(t$1, R), t$1[M] |= e$1;
}
function z(t$1, e$1) {
	j || M in t$1 || D(t$1, R), t$1[M] = e$1;
}
function W() {
	return "function" == typeof BigInt;
}
z(C, 7), V = Object.freeze(C);
var H = {};
function $(t$1, e$1) {
	return void 0 === e$1 ? t$1.i !== q && !!(2 & (0 | t$1.m[M])) : !!(2 & e$1) && t$1.i !== q;
}
var q = {};
var K = Object.freeze({});
function Y(t$1) {
	return t$1.na = !0, t$1;
}
var J = Y(((t$1) => "number" == typeof t$1)), X = Y(((t$1) => "string" == typeof t$1)), Q = Y(((t$1) => "boolean" == typeof t$1)), Z = "function" == typeof t.BigInt && "bigint" == typeof t.BigInt(0), tt = Y(((t$1) => Z ? t$1 >= nt && t$1 <= it : "-" === t$1[0] ? ot(t$1, et) : ot(t$1, rt)));
var et = Number.MIN_SAFE_INTEGER.toString(), nt = Z ? BigInt(Number.MIN_SAFE_INTEGER) : void 0, rt = Number.MAX_SAFE_INTEGER.toString(), it = Z ? BigInt(Number.MAX_SAFE_INTEGER) : void 0;
function ot(t$1, e$1) {
	if (t$1.length > e$1.length) return !1;
	if (t$1.length < e$1.length || t$1 === e$1) return !0;
	for (let n$1 = 0; n$1 < t$1.length; n$1++) {
		const r$1 = t$1[n$1], i$1 = e$1[n$1];
		if (r$1 > i$1) return !1;
		if (r$1 < i$1) return !0;
	}
}
var st, at = 0, ut = 0;
function ct(t$1) {
	const e$1 = t$1 >>> 0;
	at = e$1, ut = (t$1 - e$1) / 4294967296 >>> 0;
}
function lt(t$1) {
	if (t$1 < 0) {
		ct(-t$1);
		const [e$1, n$1] = mt(at, ut);
		at = e$1 >>> 0, ut = n$1 >>> 0;
	} else ct(t$1);
}
function ht(t$1, e$1) {
	const n$1 = 4294967296 * e$1 + (t$1 >>> 0);
	return Number.isSafeInteger(n$1) ? n$1 : ft(t$1, e$1);
}
function ft(t$1, e$1) {
	if (t$1 >>>= 0, (e$1 >>>= 0) <= 2097151) var n$1 = "" + (4294967296 * e$1 + t$1);
	else W() ? n$1 = "" + (BigInt(e$1) << BigInt(32) | BigInt(t$1)) : (t$1 = (16777215 & t$1) + 6777216 * (n$1 = 16777215 & (t$1 >>> 24 | e$1 << 8)) + 6710656 * (e$1 = e$1 >> 16 & 65535), n$1 += 8147497 * e$1, e$1 *= 2, t$1 >= 1e7 && (n$1 += t$1 / 1e7 >>> 0, t$1 %= 1e7), n$1 >= 1e7 && (e$1 += n$1 / 1e7 >>> 0, n$1 %= 1e7), n$1 = e$1 + dt(n$1) + dt(t$1));
	return n$1;
}
function dt(t$1) {
	return t$1 = String(t$1), "0000000".slice(t$1.length) + t$1;
}
function pt(t$1) {
	if (t$1.length < 16) lt(Number(t$1));
	else if (W()) t$1 = BigInt(t$1), at = Number(t$1 & BigInt(4294967295)) >>> 0, ut = Number(t$1 >> BigInt(32) & BigInt(4294967295));
	else {
		const e$1 = +("-" === t$1[0]);
		ut = at = 0;
		const n$1 = t$1.length;
		for (let r$1 = e$1, i$1 = (n$1 - e$1) % 6 + e$1; i$1 <= n$1; r$1 = i$1, i$1 += 6) {
			const e$2 = Number(t$1.slice(r$1, i$1));
			ut *= 1e6, at = 1e6 * at + e$2, at >= 4294967296 && (ut += Math.trunc(at / 4294967296), ut >>>= 0, at >>>= 0);
		}
		if (e$1) {
			const [t$2, e$2] = mt(at, ut);
			at = t$2, ut = e$2;
		}
	}
}
function mt(t$1, e$1) {
	return e$1 = ~e$1, t$1 ? t$1 = 1 + ~t$1 : e$1 += 1, [t$1, e$1];
}
function gt(t$1) {
	return Array.prototype.slice.call(t$1);
}
var _t = "function" == typeof BigInt ? BigInt.asIntN : void 0, yt = "function" == typeof BigInt ? BigInt.asUintN : void 0, bt = Number.isSafeInteger, wt = Number.isFinite, vt = Math.trunc;
function St(t$1) {
	if (null != t$1 && "number" != typeof t$1) throw Error(`Value of float/double field must be a number, found ${typeof t$1}: ${t$1}`);
	return t$1;
}
function Et(t$1) {
	return null == t$1 || "number" == typeof t$1 ? t$1 : "NaN" === t$1 || "Infinity" === t$1 || "-Infinity" === t$1 ? Number(t$1) : void 0;
}
function At(t$1) {
	if (null != t$1 && "boolean" != typeof t$1) {
		var e$1 = typeof t$1;
		throw Error(`Expected boolean but got ${"object" != e$1 ? e$1 : t$1 ? Array.isArray(t$1) ? "array" : e$1 : "null"}: ${t$1}`);
	}
	return t$1;
}
function It(t$1) {
	return null == t$1 || "boolean" == typeof t$1 ? t$1 : "number" == typeof t$1 ? !!t$1 : void 0;
}
var Tt = /^-?([1-9][0-9]*|0)(\.[0-9]+)?$/;
function Pt(t$1) {
	switch (typeof t$1) {
		case "bigint": return !0;
		case "number": return wt(t$1);
		case "string": return Tt.test(t$1);
		default: return !1;
	}
}
function Lt(t$1) {
	if ("number" != typeof t$1) throw L("int32");
	if (!wt(t$1)) throw L("int32");
	return 0 | t$1;
}
function Ot(t$1) {
	return null == t$1 ? t$1 : Lt(t$1);
}
function jt(t$1) {
	if (null == t$1) return t$1;
	if ("string" == typeof t$1 && t$1) t$1 = +t$1;
	else if ("number" != typeof t$1) return;
	return wt(t$1) ? 0 | t$1 : void 0;
}
function kt(t$1) {
	if (null == t$1) return t$1;
	if ("string" == typeof t$1 && t$1) t$1 = +t$1;
	else if ("number" != typeof t$1) return;
	return wt(t$1) ? t$1 >>> 0 : void 0;
}
function xt(t$1) {
	if ("-" === t$1[0]) return !1;
	const e$1 = t$1.length;
	return e$1 < 20 || 20 === e$1 && Number(t$1.substring(0, 6)) < 184467;
}
function Ut(t$1) {
	if (null == t$1) return t$1;
	var e$1 = typeof t$1;
	if ("bigint" === e$1) return String(yt(64, t$1));
	if (Pt(t$1)) {
		if ("string" === e$1) return e$1 = vt(Number(t$1)), bt(e$1) && e$1 >= 0 ? t$1 = String(e$1) : (-1 !== (e$1 = t$1.indexOf(".")) && (t$1 = t$1.substring(0, e$1)), xt(t$1) || (pt(t$1), t$1 = ft(at, ut))), t$1;
		if ("number" === e$1) return (t$1 = vt(t$1)) >= 0 && bt(t$1) ? t$1 : function(t$2) {
			if (t$2 < 0) {
				lt(t$2);
				var e$2 = ft(at, ut);
				return t$2 = Number(e$2), bt(t$2) ? t$2 : e$2;
			}
			return xt(e$2 = String(t$2)) ? e$2 : (lt(t$2), ht(at, ut));
		}(t$1);
	}
}
function Bt(t$1) {
	return null == t$1 || "string" == typeof t$1 ? t$1 : void 0;
}
function Nt(t$1, e$1, n$1) {
	if (null != t$1 && t$1[F] === H) return t$1;
	if (Array.isArray(t$1)) {
		var r$1 = 0 | t$1[M];
		return (n$1 = r$1 | 32 & n$1 | 2 & n$1) !== r$1 && z(t$1, n$1), new e$1(t$1);
	}
}
function Ft(t$1, e$1, n$1, r$1) {
	var i$1 = void 0 !== r$1;
	r$1 = !!r$1;
	const o$1 = [];
	var s$1 = t$1.length;
	let a$1, u$1 = 4294967295, c$1 = !1;
	const l$1 = !!(64 & e$1), h$1 = l$1 ? 128 & e$1 ? 0 : -1 : void 0;
	for (1 & e$1 || (a$1 = s$1 && t$1[s$1 - 1], null != a$1 && "object" == typeof a$1 && a$1.constructor === Object ? u$1 = --s$1 : a$1 = void 0, !l$1 || 128 & e$1 || i$1 || (c$1 = !0, u$1 = u$1 - h$1 + h$1)), e$1 = void 0, i$1 = 0; i$1 < s$1; i$1++) {
		let s$2 = t$1[i$1];
		if (null != s$2 && null != (s$2 = n$1(s$2, r$1))) if (l$1 && i$1 >= u$1) {
			const t$2 = i$1 - h$1;
			(e$1 ??= {})[t$2] = s$2;
		} else o$1[i$1] = s$2;
	}
	if (a$1) for (let i$2 in a$1) {
		if (null == (t$1 = a$1[i$2]) || null == (t$1 = n$1(t$1, r$1))) continue;
		let c$2;
		s$1 = +i$2, l$1 && !Number.isNaN(s$1) && (c$2 = s$1 + h$1) < u$1 ? o$1[c$2] = t$1 : (e$1 ??= {})[i$2] = t$1;
	}
	return e$1 && (c$1 ? o$1.push(e$1) : o$1[u$1] = e$1), o$1;
}
function Rt(t$1) {
	switch (typeof t$1) {
		case "number": return Number.isFinite(t$1) ? t$1 : "" + t$1;
		case "bigint": return tt(t$1) ? Number(t$1) : "" + t$1;
		case "boolean": return t$1 ? 1 : 0;
		case "object":
			if (Array.isArray(t$1)) {
				var e$1 = 0 | t$1[M];
				return 0 === t$1.length && 1 & e$1 ? void 0 : Ft(t$1, e$1, Rt);
			}
			if (null != t$1 && t$1[F] === H) return Dt(t$1);
			if (t$1 instanceof E) {
				if (null == (e$1 = t$1.i)) t$1 = "";
				else if ("string" == typeof e$1) t$1 = e$1;
				else {
					if (m) {
						for (var n$1 = "", r$1 = 0, i$1 = e$1.length - 10240; r$1 < i$1;) n$1 += String.fromCharCode.apply(null, e$1.subarray(r$1, r$1 += 10240));
						n$1 += String.fromCharCode.apply(null, r$1 ? e$1.subarray(r$1) : e$1), e$1 = btoa(n$1);
					} else {
						void 0 === n$1 && (n$1 = 0), d(), n$1 = l[n$1], r$1 = Array(Math.floor(e$1.length / 3)), i$1 = n$1[64] || "";
						let t$2 = 0, c$1 = 0;
						for (; t$2 < e$1.length - 2; t$2 += 3) {
							var o$1 = e$1[t$2], s$1 = e$1[t$2 + 1], a$1 = e$1[t$2 + 2], u$1 = n$1[o$1 >> 2];
							o$1 = n$1[(3 & o$1) << 4 | s$1 >> 4], s$1 = n$1[(15 & s$1) << 2 | a$1 >> 6], a$1 = n$1[63 & a$1], r$1[c$1++] = u$1 + o$1 + s$1 + a$1;
						}
						switch (u$1 = 0, a$1 = i$1, e$1.length - t$2) {
							case 2: a$1 = n$1[(15 & (u$1 = e$1[t$2 + 1])) << 2] || i$1;
							case 1: e$1 = e$1[t$2], r$1[c$1] = n$1[e$1 >> 2] + n$1[(3 & e$1) << 4 | u$1 >> 4] + a$1 + i$1;
						}
						e$1 = r$1.join("");
					}
					t$1 = t$1.i = e$1;
				}
				return t$1;
			}
			return;
	}
	return t$1;
}
function Dt(t$1) {
	return Ft(t$1 = t$1.m, 0 | t$1[M], Rt);
}
var Mt, Vt;
function Ct(t$1, e$1, n$1) {
	return Gt(t$1, e$1[0], e$1[1], n$1 ? 1 : 2);
}
function Gt(t$1, e$1, n$1, r$1 = 0) {
	if (null == t$1) {
		var i$1 = 32;
		n$1 ? (t$1 = [n$1], i$1 |= 128) : t$1 = [], e$1 && (i$1 = -8380417 & i$1 | (1023 & e$1) << 13);
	} else {
		if (!Array.isArray(t$1)) throw Error("narr");
		if (i$1 = 0 | t$1[M], a && 1 & i$1) throw Error("rfarr");
		if (2048 & i$1 && !(2 & i$1) && function() {
			if (a) throw Error("carr");
			O(N, 5);
		}(), 256 & i$1) throw Error("farr");
		if (64 & i$1) return 0 !== r$1 || 2048 & i$1 || z(t$1, 2048 | i$1), t$1;
		if (n$1 && (i$1 |= 128, n$1 !== t$1[0])) throw Error("mid");
		t: {
			i$1 |= 64;
			var o$1 = (n$1 = t$1).length;
			if (o$1) {
				var s$1 = o$1 - 1;
				const t$2 = n$1[s$1];
				if (null != t$2 && "object" == typeof t$2 && t$2.constructor === Object) {
					if ((s$1 -= e$1 = 128 & i$1 ? 0 : -1) >= 1024) throw Error("pvtlmt");
					for (var u$1 in t$2) (o$1 = +u$1) < s$1 && (n$1[o$1 + e$1] = t$2[u$1], delete t$2[u$1]);
					i$1 = -8380417 & i$1 | (1023 & s$1) << 13;
					break t;
				}
			}
			if (e$1) {
				if ((u$1 = Math.max(e$1, o$1 - (128 & i$1 ? 0 : -1))) > 1024) throw Error("spvt");
				i$1 = -8380417 & i$1 | (1023 & u$1) << 13;
			}
		}
	}
	return i$1 |= 64, 0 === r$1 && (i$1 |= 2048), z(t$1, i$1), t$1;
}
function zt(t$1, e$1) {
	if ("object" != typeof t$1) return t$1;
	if (Array.isArray(t$1)) {
		var n$1 = 0 | t$1[M];
		return 0 === t$1.length && 1 & n$1 ? t$1 = void 0 : 2 & n$1 || (!e$1 || 4096 & n$1 || 16 & n$1 ? t$1 = Ht(t$1, n$1, !1, e$1 && !(16 & n$1)) : (G(t$1, 34), 4 & n$1 && Object.freeze(t$1))), t$1;
	}
	return null != t$1 && t$1[F] === H ? $(t$1, n$1 = 0 | (e$1 = t$1.m)[M]) ? t$1 : Yt(t$1, e$1, n$1) ? Wt(t$1, e$1) : Ht(e$1, n$1) : t$1 instanceof E ? t$1 : void 0;
}
function Wt(t$1, e$1, n$1) {
	return t$1 = new t$1.constructor(e$1), n$1 && (t$1.i = q), t$1.o = q, t$1;
}
function Ht(t$1, e$1, n$1, r$1) {
	return r$1 ??= !!(34 & e$1), t$1 = Ft(t$1, e$1, zt, r$1), r$1 = 32, n$1 && (r$1 |= 2), z(t$1, e$1 = 8380609 & e$1 | r$1), t$1;
}
function $t(t$1) {
	if (t$1.i !== q) return !1;
	var e$1 = t$1.m;
	return G(e$1 = Ht(e$1, 0 | e$1[M]), 2048), t$1.m = e$1, t$1.i = void 0, t$1.o = void 0, !0;
}
function qt(t$1) {
	if (!$t(t$1) && $(t$1, 0 | t$1.m[M])) throw Error();
}
function Kt(t$1, e$1) {
	void 0 === e$1 && (e$1 = 0 | t$1[M]), 32 & e$1 && !(4096 & e$1) && z(t$1, 4096 | e$1);
}
function Yt(t$1, e$1, n$1) {
	return !!(2 & n$1) || !(!(32 & n$1) || 4096 & n$1) && (z(e$1, 2 | n$1), t$1.i = q, !0);
}
function Jt(t$1, e$1, n$1) {
	if (null !== (t$1 = Xt(t$1.m, e$1, void 0, n$1))) return t$1;
}
function Xt(t$1, e$1, n$1, r$1) {
	if (-1 === e$1) return null;
	const i$1 = e$1 + (n$1 ? 0 : -1), o$1 = t$1.length - 1;
	let s$1, a$1;
	if (!(o$1 < 1 + (n$1 ? 0 : -1))) {
		if (i$1 >= o$1) if (s$1 = t$1[o$1], null != s$1 && "object" == typeof s$1 && s$1.constructor === Object) n$1 = s$1[e$1], a$1 = !0;
		else {
			if (i$1 !== o$1) return;
			n$1 = s$1;
		}
		else n$1 = t$1[i$1];
		if (r$1 && null != n$1) {
			if (null == (r$1 = r$1(n$1))) return r$1;
			if (!Object.is(r$1, n$1)) return a$1 ? s$1[e$1] = r$1 : t$1[i$1] = r$1, r$1;
		}
		return n$1;
	}
}
function Qt(t$1, e$1, n$1) {
	qt(t$1), Zt(t$1 = t$1.m, 0 | t$1[M], e$1, n$1);
}
function Zt(t$1, e$1, n$1, r$1, i$1) {
	const o$1 = n$1 + (i$1 ? 0 : -1);
	var s$1 = t$1.length - 1;
	if (s$1 >= 1 + (i$1 ? 0 : -1) && o$1 >= s$1) {
		const i$2 = t$1[s$1];
		if (null != i$2 && "object" == typeof i$2 && i$2.constructor === Object) return i$2[n$1] = r$1, e$1;
	}
	return o$1 <= s$1 ? (t$1[o$1] = r$1, e$1) : (void 0 !== r$1 && (n$1 >= (s$1 = (e$1 ??= 0 | t$1[M]) >> 13 & 1023 || 536870912) ? null != r$1 && (t$1[s$1 + (i$1 ? 0 : -1)] = { [n$1]: r$1 }) : t$1[o$1] = r$1), e$1);
}
function te(t$1, e$1, n$1, r$1, i$1) {
	let o$1 = t$1.m, s$1 = 0 | o$1[M];
	r$1 = $(t$1, s$1) ? 1 : r$1, i$1 = !!i$1 || 3 === r$1, 2 === r$1 && $t(t$1) && (o$1 = t$1.m, s$1 = 0 | o$1[M]);
	let a$1 = (t$1 = ne(o$1, e$1)) === V ? 7 : 0 | t$1[M], u$1 = re(a$1, s$1);
	var c$1 = !(4 & u$1);
	if (c$1) {
		4 & u$1 && (t$1 = gt(t$1), a$1 = 0, u$1 = fe(u$1, s$1), s$1 = Zt(o$1, s$1, e$1, t$1));
		let r$2 = 0, i$2 = 0;
		for (; r$2 < t$1.length; r$2++) {
			const e$2 = n$1(t$1[r$2]);
			null != e$2 && (t$1[i$2++] = e$2);
		}
		i$2 < r$2 && (t$1.length = i$2), n$1 = -513 & (4 | u$1), u$1 = n$1 &= -1025, u$1 &= -4097;
	}
	return u$1 !== a$1 && (z(t$1, u$1), 2 & u$1 && Object.freeze(t$1)), ee(t$1, u$1, o$1, s$1, e$1, r$1, c$1, i$1);
}
function ee(t$1, e$1, n$1, r$1, i$1, o$1, s$1, a$1) {
	let u$1 = e$1;
	return 1 === o$1 || 4 === o$1 && (2 & e$1 || !(16 & e$1) && 32 & r$1) ? ie(e$1) || ((e$1 |= !t$1.length || s$1 && !(4096 & e$1) || 32 & r$1 && !(4096 & e$1 || 16 & e$1) ? 2 : 256) !== u$1 && z(t$1, e$1), Object.freeze(t$1)) : (2 === o$1 && ie(e$1) && (t$1 = gt(t$1), u$1 = 0, e$1 = fe(e$1, r$1), r$1 = Zt(n$1, r$1, i$1, t$1)), ie(e$1) || (a$1 || (e$1 |= 16), e$1 !== u$1 && z(t$1, e$1))), 2 & e$1 || !(4096 & e$1 || 16 & e$1) || Kt(n$1, r$1), t$1;
}
function ne(t$1, e$1, n$1) {
	return t$1 = Xt(t$1, e$1, n$1), Array.isArray(t$1) ? t$1 : V;
}
function re(t$1, e$1) {
	return 2 & e$1 && (t$1 |= 2), 1 | t$1;
}
function ie(t$1) {
	return !!(2 & t$1) && !!(4 & t$1) || !!(256 & t$1);
}
function oe(t$1, e$1, n$1) {
	qt(t$1);
	let r$1 = 0 | (t$1 = t$1.m)[M];
	if (null == n$1) Zt(t$1, r$1, e$1);
	else {
		var i$1 = n$1 === V ? 7 : 0 | n$1[M], o$1 = i$1, s$1 = ie(i$1), a$1 = s$1 || Object.isFrozen(n$1);
		for (s$1 || (i$1 = 0), a$1 || (n$1 = gt(n$1), o$1 = 0, i$1 = fe(i$1, r$1), a$1 = !1), i$1 |= 5, s$1 = 0; s$1 < n$1.length; s$1++) {
			const t$2 = n$1[s$1], e$2 = Lt(t$2);
			Object.is(t$2, e$2) || (a$1 && (n$1 = gt(n$1), o$1 = 0, i$1 = fe(i$1, r$1), a$1 = !1), n$1[s$1] = e$2);
		}
		i$1 !== o$1 && (a$1 && (n$1 = gt(n$1), i$1 = fe(i$1, r$1)), z(n$1, i$1)), Zt(t$1, r$1, e$1, n$1);
	}
}
function se(t$1, e$1, n$1, r$1) {
	qt(t$1), Zt(t$1 = t$1.m, 0 | t$1[M], e$1, ("0" === r$1 ? 0 === Number(n$1) : n$1 === r$1) ? void 0 : n$1);
}
function ae(t$1) {
	if (j) return t$1[U] ?? (t$1[U] = /* @__PURE__ */ new Map());
	if (U in t$1) return t$1[U];
	const e$1 = /* @__PURE__ */ new Map();
	return Object.defineProperty(t$1, U, { value: e$1 }), e$1;
}
function ue(t$1, e$1, n$1) {
	var r$1 = qr;
	let i$1 = t$1.get(r$1);
	if (null != i$1) return i$1;
	i$1 = 0;
	for (let t$2 = 0; t$2 < r$1.length; t$2++) {
		const o$1 = r$1[t$2];
		null != Xt(e$1, o$1) && (0 !== i$1 && (n$1 = Zt(e$1, n$1, i$1)), i$1 = o$1);
	}
	return t$1.set(r$1, i$1), i$1;
}
function ce(t$1, e$1, n$1) {
	let r$1 = t$1.m, i$1 = 0 | r$1[M];
	if (e$1 = function(t$2, e$2, n$2, r$2) {
		let i$2 = !1;
		if (null != (r$2 = Xt(t$2, r$2, void 0, ((t$3) => {
			const r$3 = Nt(t$3, n$2, e$2);
			return i$2 = r$3 !== t$3 && null != r$3, r$3;
		})))) return i$2 && !$(r$2) && Kt(t$2, e$2), r$2;
	}(r$1, i$1, e$1, n$1), null == e$1) return e$1;
	if (i$1 = 0 | r$1[M], !$(t$1, i$1)) {
		var o$1, s$1 = e$1;
		const a$1 = s$1.m, u$1 = 0 | a$1[M];
		(o$1 = $(s$1, u$1) ? Yt(s$1, a$1, u$1) ? Wt(s$1, a$1, !0) : new s$1.constructor(Ht(a$1, u$1, !1)) : s$1) !== e$1 && ($t(t$1) && (r$1 = t$1.m, i$1 = 0 | r$1[M]), i$1 = Zt(r$1, i$1, n$1, e$1 = o$1), Kt(r$1, i$1));
	}
	return e$1;
}
function le(t$1) {
	return t$1 ??= void 0, t$1;
}
function he(t$1, e$1, n$1) {
	return Qt(t$1, e$1, n$1 = le(n$1)), n$1 && !$(n$1) && Kt(t$1.m), t$1;
}
function fe(t$1, e$1) {
	return -273 & (2 & e$1 ? 2 | t$1 : -3 & t$1);
}
function de(t$1, e$1, n$1, r$1) {
	var i$1 = r$1;
	qt(t$1);
	var o$1 = r$1 = t$1.m, s$1 = 0 | r$1[M];
	const a$1 = $(t$1, s$1) ? 1 : 2;
	2 === a$1 && $t(t$1) && (s$1 = 0 | (o$1 = t$1.m)[M]);
	let u$1 = (t$1 = ne(o$1, e$1)) === V ? 7 : 0 | t$1[M];
	var c$1 = re(u$1, s$1);
	const l$1 = !(4 & c$1);
	if (l$1) {
		var h$1 = t$1, f$1 = s$1;
		const e$2 = !!(2 & c$1);
		e$2 && (f$1 |= 2);
		let r$2 = !e$2, i$2 = !0, o$2 = 0, a$2 = 0;
		for (; o$2 < h$1.length; o$2++) {
			const t$2 = Nt(h$1[o$2], n$1, f$1);
			if (t$2 instanceof n$1) {
				if (!e$2) {
					const e$3 = $(t$2);
					r$2 &&= !e$3, i$2 &&= e$3;
				}
				h$1[a$2++] = t$2;
			}
		}
		a$2 < o$2 && (h$1.length = a$2), c$1 |= 4, c$1 = i$2 ? -4097 & c$1 : 4096 | c$1, c$1 = r$2 ? 8 | c$1 : -9 & c$1;
	}
	c$1 !== u$1 && (z(t$1, c$1), 2 & c$1 && Object.freeze(t$1)), e$1 = t$1 = ee(t$1, c$1, o$1, s$1, e$1, a$1, l$1, !0), i$1 = null != i$1 ? i$1 : new n$1(), e$1.push(i$1), o$1 = n$1 = e$1 === V ? 7 : 0 | e$1[M], (i$1 = $(i$1)) ? (n$1 &= -9, 1 === e$1.length && (n$1 &= -4097)) : n$1 |= 4096, n$1 !== o$1 && z(e$1, n$1), i$1 || Kt(r$1);
}
function pe(t$1, e$1) {
	return kt(Jt(t$1, e$1)) ?? 0;
}
function me(t$1, e$1, n$1) {
	se(t$1, e$1, Ot(n$1), 0);
}
function ge(t$1, e$1, n$1) {
	if (null != n$1) {
		if ("number" != typeof n$1) throw L("uint32");
		if (!wt(n$1)) throw L("uint32");
		n$1 >>>= 0;
	}
	Qt(t$1, e$1, n$1);
}
function _e(t$1, e$1, n$1) {
	if (null != n$1 && "string" != typeof n$1) throw Error();
	se(t$1, e$1, n$1, "");
}
function ye(t$1, e$1, n$1) {
	if (qt(t$1), e$1 = (t$1 = te(t$1, e$1, Bt, 2, !0)).push, "string" != typeof n$1) throw Error();
	e$1.call(t$1, n$1);
}
var be = class {
	constructor(t$1, e$1, n$1) {
		if (this.buffer = t$1, n$1 && !e$1) throw Error();
	}
};
function we(t$1) {
	if ("string" == typeof t$1) return new be(b(t$1), !0);
	if (Array.isArray(t$1)) return new be(new Uint8Array(t$1), !0);
	if (t$1.constructor === Uint8Array) return new be(t$1, !1);
	if (t$1.constructor === ArrayBuffer) return t$1 = new Uint8Array(t$1), new be(t$1, !1);
	if (t$1.constructor === E) {
		T(v);
		var e$1 = t$1.i;
		return e$1 = (null == (e$1 = null == e$1 || w(e$1) ? e$1 : "string" == typeof e$1 ? b(e$1) : null) ? e$1 : t$1.i = e$1) || new Uint8Array(0), new be(e$1, !0, t$1);
	}
	if (t$1 instanceof Uint8Array) return t$1 = t$1.constructor === Uint8Array ? t$1 : new Uint8Array(t$1.buffer, t$1.byteOffset, t$1.byteLength), new be(t$1, !1);
	throw Error();
}
function ve(t$1) {
	return t$1 ? /^\d+$/.test(t$1) ? (pt(t$1), new Se(at, ut)) : null : Ee ||= new Se(0, 0);
}
var Se = class {
	constructor(t$1, e$1) {
		this.j = t$1 >>> 0, this.i = e$1 >>> 0;
	}
};
var Ee;
function Ae(t$1) {
	return t$1 ? /^-?\d+$/.test(t$1) ? (pt(t$1), new Ie(at, ut)) : null : Te ||= new Ie(0, 0);
}
var Ie = class {
	constructor(t$1, e$1) {
		this.j = t$1 >>> 0, this.i = e$1 >>> 0;
	}
};
var Te;
function Pe(t$1, e$1, n$1) {
	for (; n$1 > 0 || e$1 > 127;) t$1.i.push(127 & e$1 | 128), e$1 = (e$1 >>> 7 | n$1 << 25) >>> 0, n$1 >>>= 7;
	t$1.i.push(e$1);
}
function Le(t$1, e$1) {
	for (; e$1 > 127;) t$1.i.push(127 & e$1 | 128), e$1 >>>= 7;
	t$1.i.push(e$1);
}
function Oe(t$1, e$1) {
	if (e$1 >= 0) Le(t$1, e$1);
	else {
		for (let n$1 = 0; n$1 < 9; n$1++) t$1.i.push(127 & e$1 | 128), e$1 >>= 7;
		t$1.i.push(1);
	}
}
function je(t$1, e$1) {
	0 !== e$1.length && (t$1.l.push(e$1), t$1.j += e$1.length);
}
function ke(t$1, e$1, n$1) {
	Le(t$1.i, 8 * e$1 + n$1);
}
function xe(t$1, e$1) {
	return ke(t$1, e$1, 2), e$1 = t$1.i.end(), je(t$1, e$1), e$1.push(t$1.j), e$1;
}
function Ue(t$1, e$1) {
	var n$1 = e$1.pop();
	for (n$1 = t$1.j + t$1.i.length() - n$1; n$1 > 127;) e$1.push(127 & n$1 | 128), n$1 >>>= 7, t$1.j++;
	e$1.push(n$1), t$1.j++;
}
function Be(t$1, e$1, n$1) {
	ke(t$1, e$1, 2), Le(t$1.i, n$1.length), je(t$1, t$1.i.end()), je(t$1, n$1);
}
function Ne() {
	const t$1 = class {
		constructor() {
			throw Error();
		}
	};
	return Object.setPrototypeOf(t$1, t$1.prototype), t$1;
}
var Fe = Ne(), Re = Ne(), De = Ne(), Me = Ne(), Ve = Ne(), Ce = Ne(), Ge = Ne(), ze = Ne(), We = Ne(), He = Ne(), $e = class {
	constructor(t$1, e$1) {
		this.m = Gt(t$1, e$1);
	}
	toJSON() {
		return Dt(this);
	}
};
$e.prototype[F] = H, $e.prototype.toString = function() {
	return this.m.toString();
};
var qe = class {
	constructor(t$1, e$1) {
		this.i = t$1, t$1 = Fe, this.j = !!t$1 && e$1 === t$1 || !1;
	}
};
function Ke(t$1, e$1, n$1, r$1, i$1) {
	null != (e$1 = en(e$1, r$1)) && (n$1 = xe(t$1, n$1), i$1(e$1, t$1), Ue(t$1, n$1));
}
var Ye = new qe(Ke, Fe), Je = new qe(Ke, Fe);
var Xe = Symbol(), Qe = Symbol();
var Ze;
function tn(t$1) {
	var e$1 = nn, n$1 = rn, r$1 = t$1[Xe];
	if (r$1) return r$1;
	(r$1 = {}).ma = t$1, r$1.W = function(t$2) {
		switch (typeof t$2) {
			case "boolean": return Mt ||= [
				0,
				void 0,
				!0
			];
			case "number": return t$2 > 0 ? void 0 : 0 === t$2 ? Vt ||= [0, void 0] : [-t$2, void 0];
			case "string": return [0, t$2];
			case "object": return t$2;
		}
	}(t$1[0]);
	var i$1 = t$1[1];
	let o$1 = 1;
	i$1 && i$1.constructor === Object && (r$1.ba = i$1, "function" == typeof (i$1 = t$1[++o$1]) && (r$1.ha = !0, Ze ??= t$1[o$1 + 1], i$1 = t$1[o$1 += 2]));
	const s$1 = {};
	for (; i$1 && Array.isArray(i$1) && i$1.length && "number" == typeof i$1[0] && i$1[0] > 0;) {
		for (var a$1 = 0; a$1 < i$1.length; a$1++) s$1[i$1[a$1]] = i$1;
		i$1 = t$1[++o$1];
	}
	for (a$1 = 1; void 0 !== i$1;) {
		let s$2;
		"number" == typeof i$1 && (a$1 += i$1, i$1 = t$1[++o$1]);
		var u$1 = void 0;
		if (i$1 instanceof qe ? s$2 = i$1 : (s$2 = Ye, o$1--), s$2?.j) {
			i$1 = t$1[++o$1], u$1 = t$1;
			var c$1 = o$1;
			"function" == typeof i$1 && (i$1 = i$1(), u$1[c$1] = i$1), u$1 = i$1;
		}
		for (c$1 = a$1 + 1, "number" == typeof (i$1 = t$1[++o$1]) && i$1 < 0 && (c$1 -= i$1, i$1 = t$1[++o$1]); a$1 < c$1; a$1++) u$1 ? n$1(r$1, a$1, s$2, u$1) : e$1(r$1, a$1, s$2);
	}
	return t$1[Xe] = r$1;
}
function en(t$1, e$1) {
	return t$1 instanceof $e ? t$1.m : Array.isArray(t$1) ? Ct(t$1, e$1, !1) : void 0;
}
function nn(t$1, e$1, n$1) {
	t$1[e$1] = n$1.i;
}
function rn(t$1, e$1, n$1, r$1) {
	let i$1, o$1;
	const s$1 = n$1.i;
	t$1[e$1] = (t$2, e$2, n$2) => s$1(t$2, e$2, n$2, o$1 ||= tn(r$1).W, i$1 ||= on(r$1));
}
function on(t$1) {
	let e$1 = t$1[Qe];
	if (!e$1) {
		const n$1 = tn(t$1);
		e$1 = (t$2, e$2) => sn(t$2, e$2, n$1), t$1[Qe] = e$1;
	}
	return e$1;
}
function sn(t$1, e$1, n$1) {
	(function(t$2, e$2, n$2) {
		const r$1 = 128 & e$2 ? 0 : -1, i$1 = t$2.length;
		var o$1;
		(o$1 = !!i$1) && (o$1 = null != (o$1 = t$2[i$1 - 1]) && "object" == typeof o$1 && o$1.constructor === Object);
		const s$1 = i$1 + (o$1 ? -1 : 0);
		for (e$2 = 128 & e$2 ? 1 : 0; e$2 < s$1; e$2++) n$2(e$2 - r$1, t$2[e$2]);
		if (o$1) {
			t$2 = t$2[i$1 - 1];
			for (const e$3 in t$2) !isNaN(e$3) && n$2(+e$3, t$2[e$3]);
		}
	})(t$1, 0 | t$1[M], ((t$2, r$1) => {
		if (null != r$1) {
			var i$1 = function(t$3, e$2) {
				var n$2 = t$3[e$2];
				if (n$2) return n$2;
				if ((n$2 = t$3.ba) && (n$2 = n$2[e$2])) {
					var r$2 = (n$2 = Array.isArray(n$2) ? n$2[0] instanceof qe ? n$2 : [Je, n$2] : [n$2, void 0])[0].i;
					if (n$2 = n$2[1]) {
						const e$3 = on(n$2), i$2 = tn(n$2).W;
						n$2 = t$3.ha ? Ze(i$2, e$3) : (t$4, n$3, o$1) => r$2(t$4, n$3, o$1, i$2, e$3);
					} else n$2 = r$2;
					return t$3[e$2] = n$2;
				}
			}(n$1, t$2);
			i$1 ? i$1(e$1, r$1, t$2) : t$2 < 500 || O(B, 3);
		}
	}));
}
var an, un = 0, cn = un;
if (X(cn)) {
	if (!/^\s*(?:-?[1-9]\d*|0)?\s*$/.test(cn)) throw Error(String(cn));
} else if ((an = J(cn)) && (an = !Number.isSafeInteger(cn)), an) throw Error(String(cn));
function ln(t$1, e$1) {
	if (Array.isArray(e$1)) {
		var n$1 = 0 | e$1[M];
		if (4 & n$1) return e$1;
		for (var r$1 = 0, i$1 = 0; r$1 < e$1.length; r$1++) {
			const n$2 = t$1(e$1[r$1]);
			null != n$2 && (e$1[i$1++] = n$2);
		}
		return i$1 < r$1 && (e$1.length = i$1), z(e$1, -1537 & (5 | n$1)), 2 & n$1 && Object.freeze(e$1), e$1;
	}
}
function hn(t$1, e$1) {
	return new qe(t$1, e$1);
}
function fn(t$1, e$1, n$1) {
	null != (e$1 = Et(e$1)) && (ke(t$1, n$1, 5), t$1 = t$1.i, (n$1 = st ||= /* @__PURE__ */ new DataView(/* @__PURE__ */ new ArrayBuffer(8))).setFloat32(0, +e$1, !0), ut = 0, e$1 = at = n$1.getUint32(0, !0), t$1.i.push(e$1 >>> 0 & 255), t$1.i.push(e$1 >>> 8 & 255), t$1.i.push(e$1 >>> 16 & 255), t$1.i.push(e$1 >>> 24 & 255));
}
function dn(t$1, e$1, n$1) {
	null != (e$1 = jt(e$1)) && null != e$1 && (ke(t$1, n$1, 0), Oe(t$1.i, e$1));
}
function pn(t$1, e$1, n$1) {
	null != (e$1 = It(e$1)) && (ke(t$1, n$1, 0), t$1.i.i.push(e$1 ? 1 : 0));
}
function mn(t$1, e$1, n$1) {
	null != (e$1 = Bt(e$1)) && Be(t$1, n$1, i(e$1));
}
function gn(t$1, e$1, n$1, r$1, i$1) {
	null != (e$1 = en(e$1, r$1)) && (n$1 = xe(t$1, n$1), i$1(e$1, t$1), Ue(t$1, n$1));
}
function _n(t$1, e$1, n$1) {
	null != (e$1 = jt(e$1)) && (e$1 = parseInt(e$1, 10), ke(t$1, n$1, 0), Oe(t$1.i, e$1));
}
Z || (un = Q(un) ? un ? "1" : "0" : X(un) ? un.trim() || "0" : String(un));
var yn, bn = hn(fn, ze), wn = hn(fn, ze), vn = hn((function(t$1, e$1, n$1) {
	if (e$1 = function(t$2) {
		if (null == t$2) return t$2;
		var e$2 = typeof t$2;
		if ("bigint" === e$2) return String(_t(64, t$2));
		if (Pt(t$2)) {
			if ("string" === e$2) {
				if (e$2 = vt(Number(t$2)), bt(e$2)) t$2 = String(e$2);
				else if (-1 !== (e$2 = t$2.indexOf(".")) && (t$2 = t$2.substring(0, e$2)), e$2 = t$2.length, !("-" === t$2[0] ? e$2 < 20 || 20 === e$2 && Number(t$2.substring(0, 7)) > -922337 : e$2 < 19 || 19 === e$2 && Number(t$2.substring(0, 6)) < 922337)) if (pt(t$2), t$2 = at, 2147483648 & (e$2 = ut)) if (W()) t$2 = "" + (BigInt(0 | e$2) << BigInt(32) | BigInt(t$2 >>> 0));
				else {
					const [n$3, r$1] = mt(t$2, e$2);
					t$2 = "-" + ft(n$3, r$1);
				}
				else t$2 = ft(t$2, e$2);
				return t$2;
			}
			if ("number" === e$2) {
				if (t$2 = vt(t$2), !bt(t$2)) {
					lt(t$2), e$2 = at;
					var n$2 = ut;
					(t$2 = 2147483648 & n$2) && (n$2 = ~n$2 >>> 0, 0 == (e$2 = 1 + ~e$2 >>> 0) && (n$2 = n$2 + 1 >>> 0)), t$2 = "number" == typeof (e$2 = ht(e$2, n$2)) ? t$2 ? -e$2 : e$2 : t$2 ? "-" + e$2 : e$2;
				}
				return t$2;
			}
		}
	}(e$1), null != e$1) {
		if ("string" == typeof e$1) Ae(e$1);
		if (null != e$1) switch (ke(t$1, n$1, 0), typeof e$1) {
			case "number":
				t$1 = t$1.i, lt(e$1), Pe(t$1, at, ut);
				break;
			case "bigint":
				n$1 = BigInt.asUintN(64, e$1), n$1 = new Ie(Number(n$1 & BigInt(4294967295)), Number(n$1 >> BigInt(32))), Pe(t$1.i, n$1.j, n$1.i);
				break;
			default: n$1 = Ae(e$1), Pe(t$1.i, n$1.j, n$1.i);
		}
	}
}), Ce), Sn = hn((function(t$1, e$1, n$1) {
	if (null != (e$1 = Ut(e$1))) {
		if ("string" == typeof e$1) ve(e$1);
		if (null != e$1) switch (ke(t$1, n$1, 0), typeof e$1) {
			case "number":
				t$1 = t$1.i, lt(e$1), Pe(t$1, at, ut);
				break;
			case "bigint":
				n$1 = BigInt.asUintN(64, e$1), n$1 = new Se(Number(n$1 & BigInt(4294967295)), Number(n$1 >> BigInt(32))), Pe(t$1.i, n$1.j, n$1.i);
				break;
			default: n$1 = ve(e$1), Pe(t$1.i, n$1.j, n$1.i);
		}
	}
}), Ge), En = hn(dn, Me);
yn = new qe((function(t$1, e$1, n$1) {
	if (null != (e$1 = ln(jt, e$1)) && e$1.length) {
		n$1 = xe(t$1, n$1);
		for (let n$2 = 0; n$2 < e$1.length; n$2++) Oe(t$1.i, e$1[n$2]);
		Ue(t$1, n$1);
	}
}), Me);
var An, In = hn(dn, Me), Tn = hn(dn, Me), Pn = hn(pn, Re), Ln = hn(pn, Re), On = hn(mn, De);
An = new qe((function(t$1, e$1, n$1) {
	if (null != (e$1 = ln(Bt, e$1))) for (let a$1 = 0; a$1 < e$1.length; a$1++) {
		var r$1 = t$1, o$1 = n$1, s$1 = e$1[a$1];
		null != s$1 && Be(r$1, o$1, i(s$1));
	}
}), De);
var jn, kn = hn(mn, De), xn = hn(mn, De), Un = function(t$1, e$1, n$1 = Fe) {
	return new qe(e$1, n$1);
}(0, (function(t$1, e$1, n$1, r$1, i$1) {
	if (Array.isArray(e$1)) for (let o$1 = 0; o$1 < e$1.length; o$1++) gn(t$1, e$1[o$1], n$1, r$1, i$1);
})), Bn = new qe(gn, Fe), Nn = hn((function(t$1, e$1, n$1) {
	null != (e$1 = kt(e$1)) && null != e$1 && (ke(t$1, n$1, 0), Le(t$1.i, e$1));
}), Ve), Fn = hn(_n, He);
jn = new qe((function(t$1, e$1, n$1) {
	if (null != (e$1 = ln(jt, e$1)) && e$1.length) {
		n$1 = xe(t$1, n$1);
		for (let n$2 = 0; n$2 < e$1.length; n$2++) Oe(t$1.i, e$1[n$2]);
		Ue(t$1, n$1);
	}
}), He);
var Rn = hn(_n, He);
function Dn(t$1) {
	return function() {
		const e$1 = new class {
			constructor() {
				this.l = [], this.j = 0, this.i = new class {
					constructor() {
						this.i = [];
					}
					length() {
						return this.i.length;
					}
					end() {
						const t$2 = this.i;
						return this.i = [], t$2;
					}
				}();
			}
		}();
		sn(this.m, e$1, tn(t$1)), je(e$1, e$1.i.end());
		const n$1 = new Uint8Array(e$1.j), r$1 = e$1.l, i$1 = r$1.length;
		let o$1 = 0;
		for (let t$2 = 0; t$2 < i$1; t$2++) {
			const e$2 = r$1[t$2];
			n$1.set(e$2, o$1), o$1 += e$2.length;
		}
		return e$1.l = [n$1], n$1;
	};
}
function Mn(t$1, e$1) {
	if (null != e$1) if (Array.isArray(e$1)) Qt(t$1, 2, Ft(e$1, 0, Rt));
	else {
		if (!("string" == typeof e$1 || e$1 instanceof E || w(e$1))) throw Error("invalid value in Any.value field: " + e$1 + " expected a ByteString, a base64 encoded string, a Uint8Array or a jspb array");
		if (null != e$1) {
			if ("string" == typeof e$1) e$1 = e$1 ? new E(e$1, v) : S();
			else if (e$1.constructor !== E) {
				if (!w(e$1)) throw Error();
				e$1 = e$1.length ? new E(new Uint8Array(e$1), v) : S();
			}
		}
		se(t$1, 2, e$1, S());
	}
}
var Vn = class extends $e {
	constructor(t$1) {
		super(t$1);
	}
}, Cn = [
	0,
	kn,
	hn((function(t$1, e$1, n$1) {
		if (null != e$1) {
			if (e$1 instanceof $e) {
				const r$1 = e$1.pa;
				r$1 ? (e$1 = r$1(e$1), null != e$1 && Be(t$1, n$1, we(e$1).buffer)) : O(B, 3);
				return;
			}
			if (Array.isArray(e$1)) return void O(B, 3);
		}
		null != (e$1 = null == e$1 || "string" == typeof e$1 || e$1 instanceof E ? e$1 : void 0) && Be(t$1, n$1, we(e$1).buffer);
	}), We)
];
var Gn, zn = globalThis.trustedTypes;
function Wn(t$1) {
	var e$1;
	return void 0 === Gn && (Gn = function() {
		let t$2 = null;
		if (!zn) return t$2;
		try {
			const e$2 = (t$3) => t$3;
			t$2 = zn.createPolicy("goog#html", {
				createHTML: e$2,
				createScript: e$2,
				createScriptURL: e$2
			});
		} catch (t$3) {}
		return t$2;
	}()), t$1 = (e$1 = Gn) ? e$1.createScriptURL(t$1) : t$1, new class {
		constructor(t$2) {
			this.i = t$2;
		}
		toString() {
			return this.i + "";
		}
	}(t$1);
}
function Hn(t$1, ...e$1) {
	if (0 === e$1.length) return Wn(t$1[0]);
	let n$1 = t$1[0];
	for (let r$1 = 0; r$1 < e$1.length; r$1++) n$1 += encodeURIComponent(e$1[r$1]) + t$1[r$1 + 1];
	return Wn(n$1);
}
var $n = {};
$n[336783863] = [
	0,
	On,
	Pn,
	-1,
	En,
	[
		0,
		[
			1,
			2,
			3,
			4,
			5,
			6,
			7,
			8,
			9
		],
		Bn,
		[0],
		Bn,
		[
			0,
			Pn,
			On,
			Pn,
			Fn,
			-1,
			jn,
			On,
			-1,
			[
				0,
				Pn,
				-1
			],
			Fn,
			Pn,
			-1
		],
		Bn,
		[
			0,
			On,
			-2
		],
		Bn,
		[
			0,
			En,
			Pn,
			1,
			Pn,
			-3
		],
		Bn,
		[
			0,
			En,
			Fn,
			Pn,
			-1,
			yn,
			Fn,
			-1,
			Pn
		],
		Bn,
		[
			0,
			On,
			-2
		],
		Bn,
		[
			0,
			On,
			Fn
		],
		Bn,
		[
			0,
			Fn,
			On,
			-1,
			Pn
		],
		Bn,
		[
			0,
			Fn,
			-1,
			Pn
		]
	],
	[0, On],
	Pn,
	[
		0,
		[1, 3],
		[2, 4],
		Bn,
		[0, yn],
		-1,
		Bn,
		[0, An],
		-1,
		Un,
		[
			0,
			On,
			-1
		]
	],
	On
];
var qn = class extends $e {
	constructor(t$1) {
		super(t$1);
	}
}, Kn = [
	0,
	vn,
	-1,
	Ln,
	-3,
	vn,
	yn,
	kn,
	In,
	vn,
	-1,
	Ln,
	In,
	Ln,
	-2,
	kn
], Yn = class extends $e {
	constructor(t$1) {
		super(t$1, 500);
	}
	N(t$1) {
		return he(this, 7, t$1);
	}
}, Jn = [-1, {}], Xn = [
	0,
	On,
	1,
	Jn
], Qn = [
	0,
	On,
	An,
	Jn
];
function Zn(t$1, e$1) {
	de(t$1, 1, Yn, e$1);
}
var tr = class extends $e {
	constructor(t$1) {
		super(t$1, 500);
	}
	N(t$1) {
		return he(this, 1001, t$1);
	}
};
tr.prototype.j = Dn([
	-500,
	Un,
	[
		-500,
		kn,
		-1,
		An,
		-3,
		[
			-2,
			$n,
			Pn
		],
		Un,
		Cn,
		In,
		-1,
		Xn,
		Qn,
		Un,
		[
			0,
			kn,
			Ln
		],
		kn,
		Kn,
		In,
		An,
		987,
		An
	],
	4,
	Un,
	[
		-500,
		On,
		-1,
		[-1, {}],
		998,
		On
	],
	Un,
	[
		-500,
		On,
		An,
		-1,
		[
			-2,
			{},
			Pn
		],
		997,
		An,
		-1
	],
	In,
	Un,
	[
		-500,
		On,
		An,
		Jn,
		998,
		An
	],
	An,
	In,
	Xn,
	Qn,
	Un,
	[
		0,
		kn,
		-1,
		Jn
	],
	An,
	-2,
	Kn,
	kn,
	-1,
	Ln,
	[
		0,
		Ln,
		Nn
	],
	978,
	Jn,
	Un,
	Cn
]);
var er = class extends $e {
	constructor(t$1) {
		super(t$1);
	}
};
var nr;
var rr = new Uint8Array([
	0,
	97,
	115,
	109,
	1,
	0,
	0,
	0,
	1,
	5,
	1,
	96,
	0,
	1,
	123,
	3,
	2,
	1,
	0,
	10,
	10,
	1,
	8,
	0,
	65,
	0,
	253,
	15,
	253,
	98,
	11
]);
async function ir() {
	if (void 0 === nr) try {
		await WebAssembly.instantiate(rr), nr = !0;
	} catch {
		nr = !1;
	}
	return nr;
}
async function or(t$1, e$1 = Hn``) {
	const n$1 = await ir() ? "wasm_internal" : "wasm_nosimd_internal";
	return {
		wasmLoaderPath: `${e$1}/${t$1}_${n$1}.js`,
		wasmBinaryPath: `${e$1}/${t$1}_${n$1}.wasm`
	};
}
var sr = class {};
function ar(t$1) {
	function e$1(e$2, n$2) {
		return new ReadableStream({
			start() {},
			async pull(r$2) {
				i$1 = i$1.then((async () => {
					if (e$2.cache.length > 0) r$2.enqueue(e$2.cache.shift());
					else {
						var { value: i$2, done: o$2 } = await t$1.read();
						i$2 && (n$2.active && n$2.cache.push(i$2), e$2.active && r$2.enqueue(i$2)), o$2 && r$2.close();
					}
				})), await i$1;
			},
			cancel() {
				e$2.active = !1, e$2.cache.length = 0, n$2.active || t$1.cancel();
			}
		});
	}
	var n$1 = {
		cache: [],
		active: !0
	};
	const r$1 = {
		cache: [],
		active: !0
	};
	let i$1 = Promise.resolve();
	const o$1 = e$1(n$1, r$1);
	return n$1 = e$1(r$1, n$1), [o$1.getReader(), n$1.getReader()];
}
async function ur(t$1, e$1) {
	const n$1 = new Uint8Array(e$1);
	let r$1 = 0;
	for (; r$1 < e$1;) {
		const { value: i$1, done: o$1 } = await t$1.read();
		if (i$1) {
			const t$2 = i$1.subarray(0, e$1 - r$1);
			n$1.set(t$2, r$1), r$1 += t$2.length;
		}
		if (o$1) throw Error(`Expected ${e$1} bytes, but stream ended after reading ${r$1} bytes.`);
	}
	return await t$1.cancel(), n$1;
}
sr.forVisionTasks = function(t$1) {
	return or("vision", t$1);
}, sr.forTextTasks = function(t$1) {
	return or("text", t$1);
}, sr.forGenAiExperimentalTasks = function(t$1) {
	return or("genai_experimental", t$1);
}, sr.forGenAiTasks = function(t$1) {
	return or("genai", t$1);
}, sr.forAudioTasks = function(t$1) {
	return or("audio", t$1);
}, sr.isSimdSupported = function() {
	return ir();
};
var cr = [[0, async (t$1) => {
	const e$1 = new TextEncoder().encode("TFL3").length;
	return t$1 = await ur(t$1, e$1 + 4), "TFL3" === new TextDecoder("utf-8").decode(t$1.subarray(4, e$1 + 4));
}], [1, async (t$1) => 80 === (t$1 = await ur(t$1, 6))[4] && 75 === t$1[5]]];
function lr() {
	var t$1 = navigator;
	return "undefined" != typeof OffscreenCanvas && (!function(t$2 = navigator) {
		return (t$2 = t$2.userAgent).includes("Safari") && !t$2.includes("Chrome");
	}(t$1) || !!((t$1 = t$1.userAgent.match(/Version\/([\d]+).*Safari/)) && t$1.length >= 1 && Number(t$1[1]) >= 17));
}
async function hr(t$1) {
	if ("function" != typeof importScripts) {
		const e$1 = document.createElement("script");
		return e$1.src = t$1.toString(), e$1.crossOrigin = "anonymous", new Promise(((t$2, n$1) => {
			e$1.addEventListener("load", (() => {
				t$2();
			}), !1), e$1.addEventListener("error", ((t$3) => {
				n$1(t$3);
			}), !1), document.body.appendChild(e$1);
		}));
	}
	importScripts(t$1.toString());
}
function fr(t$1, e$1, n$1) {
	t$1.o || console.error("No wasm multistream support detected: ensure dependency inclusion of :gl_graph_runner_internal_multi_input target"), n$1(e$1 = t$1.h.stringToNewUTF8(e$1)), t$1.h._free(e$1);
}
function dr(t$1, e$1, n$1) {
	t$1.o || console.error("No wasm multistream support detected: ensure dependency inclusion of :gl_graph_runner_internal_multi_input target");
	const r$1 = new Uint32Array(e$1.length);
	for (let n$2 = 0; n$2 < e$1.length; n$2++) r$1[n$2] = t$1.h.stringToNewUTF8(e$1[n$2]);
	e$1 = t$1.h._malloc(4 * r$1.length), t$1.h.HEAPU32.set(r$1, e$1 >> 2), n$1(e$1);
	for (const e$2 of r$1) t$1.h._free(e$2);
	t$1.h._free(e$1);
}
function pr(t$1, e$1, n$1) {
	t$1.h.simpleListeners = t$1.h.simpleListeners || {}, t$1.h.simpleListeners[e$1] = n$1;
}
function mr(t$1, e$1, n$1) {
	let r$1 = [];
	t$1.h.simpleListeners = t$1.h.simpleListeners || {}, t$1.h.simpleListeners[e$1] = (t$2, e$2, i$1) => {
		e$2 ? (n$1(r$1, i$1), r$1 = []) : r$1.push(t$2);
	};
}
var gr = (_r = class {
	constructor(t$1, e$1) {
		this.l = !0, this.h = t$1, this.i = null, this.j = 0, this.o = "function" == typeof this.h._addIntToInputStream, void 0 !== e$1 ? this.h.canvas = e$1 : lr() ? this.h.canvas = new OffscreenCanvas(1, 1) : (console.warn("OffscreenCanvas not supported and GraphRunner constructor glCanvas parameter is undefined. Creating backup canvas."), this.h.canvas = document.createElement("canvas"));
	}
	async initializeGraph(t$1) {
		const e$1 = await (await fetch(t$1)).arrayBuffer();
		t$1 = !(t$1.endsWith(".pbtxt") || t$1.endsWith(".textproto")), this.setGraph(new Uint8Array(e$1), t$1);
	}
	setGraphFromString(t$1) {
		this.setGraph(new TextEncoder().encode(t$1), !1);
	}
	setGraph(t$1, e$1) {
		const n$1 = t$1.length, r$1 = this.h._malloc(n$1);
		this.h.HEAPU8.set(t$1, r$1), e$1 ? this.h._changeBinaryGraph(n$1, r$1) : this.h._changeTextGraph(n$1, r$1), this.h._free(r$1);
	}
	configureAudio(t$1, e$1, n$1, r$1, i$1) {
		this.h._configureAudio || console.warn("Attempting to use configureAudio without support for input audio. Is build dep \":gl_graph_runner_audio\" missing?"), fr(this, r$1 || "input_audio", ((r$2) => {
			fr(this, i$1 = i$1 || "audio_header", ((i$2) => {
				this.h._configureAudio(r$2, i$2, t$1, e$1 ?? 0, n$1);
			}));
		}));
	}
	setAutoResizeCanvas(t$1) {
		this.l = t$1;
	}
	setAutoRenderToScreen(t$1) {
		this.h._setAutoRenderToScreen(t$1);
	}
	setGpuBufferVerticalFlip(t$1) {
		this.h.gpuOriginForWebTexturesIsBottomLeft = t$1;
	}
	attachErrorListener(t$1) {
		this.h.errorListener = t$1;
	}
	attachEmptyPacketListener(t$1, e$1) {
		this.h.emptyPacketListeners = this.h.emptyPacketListeners || {}, this.h.emptyPacketListeners[t$1] = e$1;
	}
	addAudioToStream(t$1, e$1, n$1) {
		this.addAudioToStreamWithShape(t$1, 0, 0, e$1, n$1);
	}
	addAudioToStreamWithShape(t$1, e$1, n$1, r$1, i$1) {
		const o$1 = 4 * t$1.length;
		this.j !== o$1 && (this.i && this.h._free(this.i), this.i = this.h._malloc(o$1), this.j = o$1), this.h.HEAPF32.set(t$1, this.i / 4), fr(this, r$1, ((t$2) => {
			this.h._addAudioToInputStream(this.i, e$1, n$1, t$2, i$1);
		}));
	}
	addGpuBufferToStream(t$1, e$1, n$1) {
		fr(this, e$1, ((e$2) => {
			if (!this.h.canvas) throw Error("No OpenGL canvas configured.");
			e$2 ? this.h._bindTextureToStream(e$2) : this.h._bindTextureToCanvas();
			const r$1 = this.h.canvas.getContext("webgl2") || this.h.canvas.getContext("webgl");
			if (!r$1) throw Error("Failed to obtain WebGL context from the provided canvas. `getContext()` should only be invoked with `webgl` or `webgl2`.");
			this.h.gpuOriginForWebTexturesIsBottomLeft && r$1.pixelStorei(r$1.UNPACK_FLIP_Y_WEBGL, !0), r$1.texImage2D(r$1.TEXTURE_2D, 0, r$1.RGBA, r$1.RGBA, r$1.UNSIGNED_BYTE, t$1), this.h.gpuOriginForWebTexturesIsBottomLeft && r$1.pixelStorei(r$1.UNPACK_FLIP_Y_WEBGL, !1);
			const [i$1, o$1] = void 0 !== t$1.videoWidth ? [t$1.videoWidth, t$1.videoHeight] : void 0 !== t$1.naturalWidth ? [t$1.naturalWidth, t$1.naturalHeight] : void 0 !== t$1.displayWidth ? [t$1.displayWidth, t$1.displayHeight] : [t$1.width, t$1.height];
			!this.l || i$1 === this.h.canvas.width && o$1 === this.h.canvas.height || (this.h.canvas.width = i$1, this.h.canvas.height = o$1);
			const [s$1, a$1] = [i$1, o$1];
			this.h._addBoundTextureToStream(e$2, s$1, a$1, n$1);
		}));
	}
	addBoolToStream(t$1, e$1, n$1) {
		fr(this, e$1, ((e$2) => {
			this.h._addBoolToInputStream(t$1, e$2, n$1);
		}));
	}
	addDoubleToStream(t$1, e$1, n$1) {
		fr(this, e$1, ((e$2) => {
			this.h._addDoubleToInputStream(t$1, e$2, n$1);
		}));
	}
	addFloatToStream(t$1, e$1, n$1) {
		fr(this, e$1, ((e$2) => {
			this.h._addFloatToInputStream(t$1, e$2, n$1);
		}));
	}
	addIntToStream(t$1, e$1, n$1) {
		fr(this, e$1, ((e$2) => {
			this.h._addIntToInputStream(t$1, e$2, n$1);
		}));
	}
	addUintToStream(t$1, e$1, n$1) {
		fr(this, e$1, ((e$2) => {
			this.h._addUintToInputStream(t$1, e$2, n$1);
		}));
	}
	addStringToStream(t$1, e$1, n$1) {
		fr(this, e$1, ((e$2) => {
			fr(this, t$1, ((t$2) => {
				this.h._addStringToInputStream(t$2, e$2, n$1);
			}));
		}));
	}
	addStringRecordToStream(t$1, e$1, n$1) {
		fr(this, e$1, ((e$2) => {
			dr(this, Object.keys(t$1), ((r$1) => {
				dr(this, Object.values(t$1), ((i$1) => {
					this.h._addFlatHashMapToInputStream(r$1, i$1, Object.keys(t$1).length, e$2, n$1);
				}));
			}));
		}));
	}
	addProtoToStream(t$1, e$1, n$1, r$1) {
		fr(this, n$1, ((n$2) => {
			fr(this, e$1, ((e$2) => {
				const i$1 = this.h._malloc(t$1.length);
				this.h.HEAPU8.set(t$1, i$1), this.h._addProtoToInputStream(i$1, t$1.length, e$2, n$2, r$1), this.h._free(i$1);
			}));
		}));
	}
	addEmptyPacketToStream(t$1, e$1) {
		fr(this, t$1, ((t$2) => {
			this.h._addEmptyPacketToInputStream(t$2, e$1);
		}));
	}
	addBoolVectorToStream(t$1, e$1, n$1) {
		fr(this, e$1, ((e$2) => {
			const r$1 = this.h._allocateBoolVector(t$1.length);
			if (!r$1) throw Error("Unable to allocate new bool vector on heap.");
			for (const e$3 of t$1) this.h._addBoolVectorEntry(r$1, e$3);
			this.h._addBoolVectorToInputStream(r$1, e$2, n$1);
		}));
	}
	addDoubleVectorToStream(t$1, e$1, n$1) {
		fr(this, e$1, ((e$2) => {
			const r$1 = this.h._allocateDoubleVector(t$1.length);
			if (!r$1) throw Error("Unable to allocate new double vector on heap.");
			for (const e$3 of t$1) this.h._addDoubleVectorEntry(r$1, e$3);
			this.h._addDoubleVectorToInputStream(r$1, e$2, n$1);
		}));
	}
	addFloatVectorToStream(t$1, e$1, n$1) {
		fr(this, e$1, ((e$2) => {
			const r$1 = this.h._allocateFloatVector(t$1.length);
			if (!r$1) throw Error("Unable to allocate new float vector on heap.");
			for (const e$3 of t$1) this.h._addFloatVectorEntry(r$1, e$3);
			this.h._addFloatVectorToInputStream(r$1, e$2, n$1);
		}));
	}
	addIntVectorToStream(t$1, e$1, n$1) {
		fr(this, e$1, ((e$2) => {
			const r$1 = this.h._allocateIntVector(t$1.length);
			if (!r$1) throw Error("Unable to allocate new int vector on heap.");
			for (const e$3 of t$1) this.h._addIntVectorEntry(r$1, e$3);
			this.h._addIntVectorToInputStream(r$1, e$2, n$1);
		}));
	}
	addUintVectorToStream(t$1, e$1, n$1) {
		fr(this, e$1, ((e$2) => {
			const r$1 = this.h._allocateUintVector(t$1.length);
			if (!r$1) throw Error("Unable to allocate new unsigned int vector on heap.");
			for (const e$3 of t$1) this.h._addUintVectorEntry(r$1, e$3);
			this.h._addUintVectorToInputStream(r$1, e$2, n$1);
		}));
	}
	addStringVectorToStream(t$1, e$1, n$1) {
		fr(this, e$1, ((e$2) => {
			const r$1 = this.h._allocateStringVector(t$1.length);
			if (!r$1) throw Error("Unable to allocate new string vector on heap.");
			for (const e$3 of t$1) fr(this, e$3, ((t$2) => {
				this.h._addStringVectorEntry(r$1, t$2);
			}));
			this.h._addStringVectorToInputStream(r$1, e$2, n$1);
		}));
	}
	addBoolToInputSidePacket(t$1, e$1) {
		fr(this, e$1, ((e$2) => {
			this.h._addBoolToInputSidePacket(t$1, e$2);
		}));
	}
	addDoubleToInputSidePacket(t$1, e$1) {
		fr(this, e$1, ((e$2) => {
			this.h._addDoubleToInputSidePacket(t$1, e$2);
		}));
	}
	addFloatToInputSidePacket(t$1, e$1) {
		fr(this, e$1, ((e$2) => {
			this.h._addFloatToInputSidePacket(t$1, e$2);
		}));
	}
	addIntToInputSidePacket(t$1, e$1) {
		fr(this, e$1, ((e$2) => {
			this.h._addIntToInputSidePacket(t$1, e$2);
		}));
	}
	addUintToInputSidePacket(t$1, e$1) {
		fr(this, e$1, ((e$2) => {
			this.h._addUintToInputSidePacket(t$1, e$2);
		}));
	}
	addStringToInputSidePacket(t$1, e$1) {
		fr(this, e$1, ((e$2) => {
			fr(this, t$1, ((t$2) => {
				this.h._addStringToInputSidePacket(t$2, e$2);
			}));
		}));
	}
	addProtoToInputSidePacket(t$1, e$1, n$1) {
		fr(this, n$1, ((n$2) => {
			fr(this, e$1, ((e$2) => {
				const r$1 = this.h._malloc(t$1.length);
				this.h.HEAPU8.set(t$1, r$1), this.h._addProtoToInputSidePacket(r$1, t$1.length, e$2, n$2), this.h._free(r$1);
			}));
		}));
	}
	addBoolVectorToInputSidePacket(t$1, e$1) {
		fr(this, e$1, ((e$2) => {
			const n$1 = this.h._allocateBoolVector(t$1.length);
			if (!n$1) throw Error("Unable to allocate new bool vector on heap.");
			for (const e$3 of t$1) this.h._addBoolVectorEntry(n$1, e$3);
			this.h._addBoolVectorToInputSidePacket(n$1, e$2);
		}));
	}
	addDoubleVectorToInputSidePacket(t$1, e$1) {
		fr(this, e$1, ((e$2) => {
			const n$1 = this.h._allocateDoubleVector(t$1.length);
			if (!n$1) throw Error("Unable to allocate new double vector on heap.");
			for (const e$3 of t$1) this.h._addDoubleVectorEntry(n$1, e$3);
			this.h._addDoubleVectorToInputSidePacket(n$1, e$2);
		}));
	}
	addFloatVectorToInputSidePacket(t$1, e$1) {
		fr(this, e$1, ((e$2) => {
			const n$1 = this.h._allocateFloatVector(t$1.length);
			if (!n$1) throw Error("Unable to allocate new float vector on heap.");
			for (const e$3 of t$1) this.h._addFloatVectorEntry(n$1, e$3);
			this.h._addFloatVectorToInputSidePacket(n$1, e$2);
		}));
	}
	addIntVectorToInputSidePacket(t$1, e$1) {
		fr(this, e$1, ((e$2) => {
			const n$1 = this.h._allocateIntVector(t$1.length);
			if (!n$1) throw Error("Unable to allocate new int vector on heap.");
			for (const e$3 of t$1) this.h._addIntVectorEntry(n$1, e$3);
			this.h._addIntVectorToInputSidePacket(n$1, e$2);
		}));
	}
	addUintVectorToInputSidePacket(t$1, e$1) {
		fr(this, e$1, ((e$2) => {
			const n$1 = this.h._allocateUintVector(t$1.length);
			if (!n$1) throw Error("Unable to allocate new unsigned int vector on heap.");
			for (const e$3 of t$1) this.h._addUintVectorEntry(n$1, e$3);
			this.h._addUintVectorToInputSidePacket(n$1, e$2);
		}));
	}
	addStringVectorToInputSidePacket(t$1, e$1) {
		fr(this, e$1, ((e$2) => {
			const n$1 = this.h._allocateStringVector(t$1.length);
			if (!n$1) throw Error("Unable to allocate new string vector on heap.");
			for (const e$3 of t$1) fr(this, e$3, ((t$2) => {
				this.h._addStringVectorEntry(n$1, t$2);
			}));
			this.h._addStringVectorToInputSidePacket(n$1, e$2);
		}));
	}
	attachBoolListener(t$1, e$1) {
		pr(this, t$1, e$1), fr(this, t$1, ((t$2) => {
			this.h._attachBoolListener(t$2);
		}));
	}
	attachBoolVectorListener(t$1, e$1) {
		mr(this, t$1, e$1), fr(this, t$1, ((t$2) => {
			this.h._attachBoolVectorListener(t$2);
		}));
	}
	attachIntListener(t$1, e$1) {
		pr(this, t$1, e$1), fr(this, t$1, ((t$2) => {
			this.h._attachIntListener(t$2);
		}));
	}
	attachIntVectorListener(t$1, e$1) {
		mr(this, t$1, e$1), fr(this, t$1, ((t$2) => {
			this.h._attachIntVectorListener(t$2);
		}));
	}
	attachUintListener(t$1, e$1) {
		pr(this, t$1, e$1), fr(this, t$1, ((t$2) => {
			this.h._attachUintListener(t$2);
		}));
	}
	attachUintVectorListener(t$1, e$1) {
		mr(this, t$1, e$1), fr(this, t$1, ((t$2) => {
			this.h._attachUintVectorListener(t$2);
		}));
	}
	attachDoubleListener(t$1, e$1) {
		pr(this, t$1, e$1), fr(this, t$1, ((t$2) => {
			this.h._attachDoubleListener(t$2);
		}));
	}
	attachDoubleVectorListener(t$1, e$1) {
		mr(this, t$1, e$1), fr(this, t$1, ((t$2) => {
			this.h._attachDoubleVectorListener(t$2);
		}));
	}
	attachFloatListener(t$1, e$1) {
		pr(this, t$1, e$1), fr(this, t$1, ((t$2) => {
			this.h._attachFloatListener(t$2);
		}));
	}
	attachFloatVectorListener(t$1, e$1) {
		mr(this, t$1, e$1), fr(this, t$1, ((t$2) => {
			this.h._attachFloatVectorListener(t$2);
		}));
	}
	attachStringListener(t$1, e$1) {
		pr(this, t$1, e$1), fr(this, t$1, ((t$2) => {
			this.h._attachStringListener(t$2);
		}));
	}
	attachStringVectorListener(t$1, e$1) {
		mr(this, t$1, e$1), fr(this, t$1, ((t$2) => {
			this.h._attachStringVectorListener(t$2);
		}));
	}
	attachProtoListener(t$1, e$1, n$1) {
		pr(this, t$1, e$1), fr(this, t$1, ((t$2) => {
			this.h._attachProtoListener(t$2, n$1 || !1);
		}));
	}
	attachProtoVectorListener(t$1, e$1, n$1) {
		mr(this, t$1, e$1), fr(this, t$1, ((t$2) => {
			this.h._attachProtoVectorListener(t$2, n$1 || !1);
		}));
	}
	attachAudioListener(t$1, e$1, n$1) {
		this.h._attachAudioListener || console.warn("Attempting to use attachAudioListener without support for output audio. Is build dep \":gl_graph_runner_audio_out\" missing?"), pr(this, t$1, ((t$2, n$2) => {
			t$2 = new Float32Array(t$2.buffer, t$2.byteOffset, t$2.length / 4), e$1(t$2, n$2);
		})), fr(this, t$1, ((t$2) => {
			this.h._attachAudioListener(t$2, n$1 || !1);
		}));
	}
	finishProcessing() {
		this.h._waitUntilIdle();
	}
	closeGraph() {
		this.h._closeGraph(), this.h.simpleListeners = void 0, this.h.emptyPacketListeners = void 0;
	}
}, class extends _r {
	ja() {
		this.h._registerModelResourcesGraphService();
	}
});
var _r;
async function yr(t$1, e$1) {
	const n$1 = await (async (t$2, e$2, n$2) => {
		var r$1 = ii;
		if (t$2 && await hr(t$2), !self.ModuleFactory) throw Error("ModuleFactory not set.");
		if (e$2 && (await hr(e$2), !self.ModuleFactory)) throw Error("ModuleFactory not set.");
		return self.Module && n$2 && ((t$2 = self.Module).locateFile = n$2.locateFile, n$2.mainScriptUrlOrBlob && (t$2.mainScriptUrlOrBlob = n$2.mainScriptUrlOrBlob)), n$2 = await self.ModuleFactory(self.Module || n$2), self.ModuleFactory = self.Module = void 0, new r$1(n$2, null);
	})(t$1.wasmLoaderPath, t$1.assetLoaderPath, { locateFile: (e$2) => e$2.endsWith(".wasm") ? t$1.wasmBinaryPath.toString() : t$1.assetBinaryPath && e$2.endsWith(".data") ? t$1.assetBinaryPath.toString() : e$2 });
	return await n$1.N(e$1), n$1;
}
async function br(t$1, e$1) {
	return yr(t$1, e$1);
}
function wr(t$1) {
	try {
		const e$1 = t$1.J.length;
		if (1 === e$1) throw Error(t$1.J[0].message);
		if (e$1 > 1) throw Error("Encountered multiple errors: " + t$1.J.map(((t$2) => t$2.message)).join(", "));
	} finally {
		t$1.J = [];
	}
}
function vr(t$1, e$1) {
	t$1.I = Math.max(t$1.I, e$1);
}
var Sr = class {
	constructor(t$1) {
		this.j = t$1, this.J = [], this.I = 0, this.j.setAutoRenderToScreen(!1);
	}
	setGraph(t$1, e$1) {
		this.j.attachErrorListener(((t$2, e$2) => {
			this.J.push(Error(e$2));
		})), this.j.ja(), this.j.setGraph(t$1, e$1), wr(this);
	}
	finishProcessing() {
		this.j.finishProcessing(), wr(this);
	}
	close() {
		this.j.closeGraph();
	}
};
Sr.prototype.close = Sr.prototype.close;
var Er = class extends $e {
	constructor(t$1) {
		super(t$1);
	}
	j() {
		return jt(Jt(this, 2)) ?? 0;
	}
};
function Ar(t$1, e$1) {
	he(t$1, 1, e$1);
}
var Ir = class extends $e {
	constructor(t$1) {
		super(t$1);
	}
}, Tr = [
	0,
	Rn,
	In,
	wn,
	-1,
	En
];
function Pr(t$1, e$1, n$1, r$1) {
	if (void 0 !== t$1.data) {
		var i$1 = new Uint8Array(t$1.data.buffer, e$1, n$1);
		return 1 === r$1 && function(t$2, e$2, n$2) {
			t$2.i.push([e$2, n$2]), t$2.i.sort(((t$3, e$3) => t$3[0] - e$3[0])), e$2 = 0;
			for (const [r$2, i$2] of t$2.i) {
				const t$3 = i$2;
				(n$2 = r$2) <= e$2 && (e$2 = Math.max(e$2, n$2 + t$3));
			}
			e$2 === t$2.length && (t$2.data = void 0);
		}(t$1, e$1, n$1), i$1;
	}
}
Er.prototype.l = Dn(Tr);
var Lr = class {
	constructor(t$1) {
		this.i = [], this.data = t$1, this.length = t$1.length;
	}
};
function Or(t$1, e$1) {
	return new kr((async () => {
		const { value: e$2, done: n$1 } = await t$1.read();
		return n$1 ? void 0 : e$2;
	}), e$1);
}
async function jr(t$1, e$1, n$1, r$1, i$1) {
	if (2 === i$1) return t$1.i = [], t$1.j = () => Promise.resolve(void 0), setTimeout((() => {
		t$1.l();
	}), 0), Promise.resolve(0);
	for (; t$1.size < n$1 + r$1;) {
		var o$1 = await t$1.j();
		if (void 0 === o$1) break;
		t$1.i.push(new Lr(o$1));
	}
	if (t$1.size < n$1 + r$1) throw Error(`Data size is too small: ${t$1.size}, expected at least ${n$1 + r$1}.`);
	o$1 = e$1._malloc(r$1) >>> 0;
	let s$1 = 0;
	for (let a$1 = 0; a$1 < t$1.i.length; a$1++) {
		const u$1 = t$1.i[a$1];
		if (n$1 >= u$1.length) {
			n$1 -= u$1.length;
			continue;
		}
		const c$1 = Math.min(r$1, u$1.length - n$1);
		if (void 0 === (n$1 = Pr(u$1, n$1, c$1, i$1))) throw Error("Data has already been released.");
		if (e$1.HEAPU8.set(n$1, o$1 + s$1), n$1 = 0, s$1 += c$1, 0 === (r$1 -= c$1)) break;
	}
	if (0 !== r$1) throw Error("Data not found.");
	return Promise.resolve(o$1);
}
var kr = class {
	constructor(t$1, e$1) {
		this.i = [], this.j = t$1, this.l = e$1;
	}
	get size() {
		let t$1 = 0;
		for (let e$1 = 0; e$1 < this.i.length; e$1++) t$1 += this.i[e$1].length;
		return t$1;
	}
};
function xr(t$1) {
	return "object" == typeof t$1 && null != t$1 && "imageSource" in t$1;
}
function Ur(t$1) {
	return "object" == typeof t$1 && null != t$1 && "audioSource" in t$1;
}
async function Br(t$1, e$1, n$1) {
	t$1 = new Fr(t$1, n$1);
	let r$1 = 0;
	for (e$1 = e$1.getReader();;) {
		const { value: n$2, done: i$1 } = await e$1.read();
		if (i$1) break;
		t$1.set(n$2, r$1), r$1 += n$2.byteLength;
	}
	if (n$1 !== r$1) throw Nr(t$1), Error(`File could not be fully loaded to memory, so was not retained. Loaded ${r$1}/${n$1} bytes before failure`);
	return t$1;
}
function Nr(t$1) {
	if (t$1.i) try {
		t$1.h._free(t$1.j);
	} catch {} finally {
		t$1.i = !1;
	}
}
var Fr = class {
	constructor(t$1, e$1) {
		this.h = t$1, this.l = e$1, this.j = this.h._malloc(e$1) >>> 0, this.o = this.h.HEAPU8, this.i = !!this.j;
	}
	get offset() {
		if (!this.i) throw Error("WasmFileReference has been freed.");
		return this.j;
	}
	get size() {
		if (!this.i) throw Error("WasmFileReference has been freed.");
		return this.l;
	}
	set(t$1, e$1) {
		this.o.set(t$1, this.j + (e$1 ?? 0));
	}
}, Rr = class extends $e {
	constructor(t$1) {
		super(t$1);
	}
};
Rr.prototype.j = Dn([
	0,
	kn,
	2,
	An,
	In,
	Ln
]);
var Dr = class extends $e {
	constructor(t$1) {
		super(t$1);
	}
}, Mr = class extends $e {
	constructor(t$1) {
		super(t$1);
	}
}, Vr = class extends $e {
	constructor(t$1) {
		super(t$1);
	}
}, Cr = class extends $e {
	constructor(t$1) {
		super(t$1);
	}
}, Gr = [
	0,
	In,
	-6,
	1,
	In,
	1,
	[
		0,
		Ln,
		Rn,
		-2
	],
	[
		0,
		Ln,
		wn
	],
	Rn,
	-2,
	[
		0,
		Ln,
		-1,
		Rn,
		wn,
		Fn,
		En,
		Pn,
		-1
	],
	1,
	Ln,
	In,
	En,
	-1,
	[
		0,
		Rn,
		In
	],
	Ln,
	-1,
	bn,
	In,
	-5,
	bn,
	-1,
	[
		0,
		En,
		bn
	],
	En,
	Pn,
	[
		0,
		En,
		-2
	],
	bn,
	[0, In],
	[
		0,
		In,
		-4
	],
	Pn,
	En,
	-2,
	Pn
], zr = [
	0,
	kn,
	-2
], Wr = [
	0,
	[4, 6],
	Gr,
	In,
	1,
	Tn,
	An,
	xn,
	jn,
	zr,
	En,
	[
		0,
		[
			0,
			In,
			-1,
			Un,
			[
				0,
				In,
				[
					0,
					In,
					-1
				],
				-1,
				[
					0,
					Rn,
					-1
				],
				Ln
			],
			Ln,
			-2,
			In,
			-1
		],
		[
			0,
			In,
			-1,
			Ln
		],
		Gr,
		Ln,
		In,
		[0, In]
	],
	On,
	-3,
	[
		0,
		In,
		Ln
	],
	Gr,
	[
		0,
		zr,
		-2
	],
	yn
];
Cr.prototype.j = Dn([
	0,
	kn,
	8,
	[
		0,
		Ln,
		-6
	],
	1,
	In,
	1,
	In,
	[
		0,
		Un,
		[
			0,
			kn,
			Sn,
			-1,
			Rn
		],
		Wr,
		In
	],
	[
		0,
		In,
		Ln,
		-3
	],
	1,
	Rn,
	1,
	Wr,
	1,
	In,
	5,
	Rn,
	yn,
	1,
	Tr,
	Ln,
	In
]);
var Hr = class extends $e {
	constructor(t$1) {
		super(t$1);
	}
}, $r = class extends $e {
	constructor(t$1) {
		super(t$1);
	}
}, qr = [2, 4];
$r.prototype.j = Dn([
	0,
	qr,
	In,
	xn,
	In,
	Bn,
	[
		0,
		1,
		kn
	]
]);
var Kr = function(t$1) {
	return class extends t$1 {
		constructor() {
			super(...arguments), this.P = !1, this.F = this.H = 0;
		}
		M() {
			if (this.P) throw Error("Cannot process because LLM inference engine is currently loading or processing.");
			this.P = !0;
		}
		L() {
			this.P = !1;
		}
		async createLlmInferenceEngine(t$2, e$1) {
			this.M();
			try {
				const n$1 = Or(t$2, (() => {}));
				await this.h.createLlmInferenceEngine(pe(e$1, 2) ?? 512, ce(e$1, Er, 3)?.j() ?? 40, It(Jt(e$1, 6)) ?? !1 ?? !1, pe(e$1, 7) ?? 0, It(Jt(e$1, 8)) ?? !1 ?? !1, ((t$3, e$2, r$1) => jr(n$1, this.h, t$3, e$2, r$1)));
			} finally {
				this.L();
			}
		}
		async aa(t$2, e$1) {
			this.M();
			try {
				await this.la(t$2), await this.h.ccall("CreateLlmInferenceEngineConverted", "void", [
					"number",
					"number",
					"boolean"
				], [
					pe(e$1, 2) ?? 512,
					ce(e$1, Er, 3)?.j() ?? 40,
					It(Jt(e$1, 6)) ?? !1 ?? !1
				], { async: !0 });
			} finally {
				this.L();
			}
		}
		V() {
			this.M();
			try {
				const t$2 = this.h;
				t$2.ccall("DeleteLlmInferenceEngine", "void", [], [], { async: !1 }), this.H && (t$2._FreeSession(this.H), this.F === this.H && (this.F = 0), this.H = 0), this.F && (t$2._FreeSession(this.F), this.F = 0);
			} finally {
				this.L();
			}
		}
		async R(t$2, e$1, n$1) {
			this.M();
			try {
				const r$1 = [], i$1 = this.h;
				i$1._userProgressListener = (t$3, e$2) => {
					t$3 && r$1.push(t$3), n$1 && n$1(t$3, e$2);
				};
				const o$1 = e$1.l(), s$1 = o$1.length, a$1 = this.h._malloc(s$1);
				this.h.HEAPU8.set(o$1, a$1);
				const u$1 = t$2.some(Ur), c$1 = t$2.some(xr);
				i$1.ccallNum = i$1.ccall;
				const l$1 = await i$1.ccallNum("MakeSessionForPredict", "number", [
					"number",
					"number",
					"boolean",
					"boolean"
				], [
					a$1,
					s$1,
					c$1,
					u$1
				], { async: !0 });
				e$1 = [];
				for (const n$2 of t$2) if ("string" == typeof n$2) fr(this, n$2, ((t$3) => {
					i$1._AddTextQueryChunk(l$1, t$3);
				}));
				else if (xr(n$2)) {
					const { image: t$3, width: r$2, height: o$2 } = await this.ea(n$2.imageSource), s$2 = "undefined" != typeof OffscreenCanvas ? new OffscreenCanvas(r$2, o$2) : document.createElement("canvas");
					s$2.width = r$2, s$2.height = o$2;
					const a$2 = s$2.getContext("2d");
					a$2.drawImage(t$3, 0, 0);
					const u$2 = a$2.getImageData(0, 0, r$2, o$2), c$2 = this.h._malloc(u$2.width * u$2.height * 4);
					this.h.HEAPU8.set(u$2.data, c$2), i$1._AddImageQueryChunk(l$1, c$2, u$2.width, u$2.height), e$1.push(c$2);
				} else {
					if (!Ur(n$2)) throw Error("Unsupported PromptPart type in query.");
					{
						const t$3 = await this.da(n$2.audioSource), r$2 = this.h._malloc(t$3.audioSamples.byteLength);
						this.h.HEAPF32.set(t$3.audioSamples, r$2 / 4), i$1._AddAudioQueryChunk(l$1, t$3.audioSampleRateHz, r$2, t$3.audioSamples.length), e$1.push(r$2);
					}
				}
				await i$1.ccall("PredictSession", "void", ["number"], [l$1], { async: !0 }), t$2 = !0, c$1 && 0 === this.F && (this.F = l$1, t$2 = !1), u$1 && 0 === this.H && (this.H = l$1, t$2 = !1), t$2 && i$1._FreeSession(l$1);
				for (const t$3 of e$1) this.h._free(t$3);
				return e$1.length = 0, n$1 && n$1("", !0), this.h._free(a$1), i$1._userProgressListener = void 0, r$1.join("");
			} finally {
				this.L();
			}
		}
		S(t$2) {
			this.M();
			let e$1 = 0, n$1 = "";
			for (const r$1 of t$2) "string" == typeof r$1 ? n$1 += r$1 : xr(r$1) ? e$1 += 260 : Ur(r$1) && console.warn("sizeInTokens is not yet implemented for audio; audio tokens will not be counted");
			try {
				let t$3;
				return fr(this, n$1, ((e$2) => {
					t$3 = this.h._GetSizeInTokens(e$2);
				})), e$1 + t$3;
			} finally {
				this.L();
			}
		}
		async la(t$2) {
			t$2 = await async function(t$3) {
				const e$1 = [];
				for (var n$1 = 0;;) {
					const { done: r$1, value: i$1 } = await t$3.read();
					if (r$1) break;
					e$1.push(i$1), n$1 += i$1.length;
				}
				if (0 === e$1.length) return new Uint8Array(0);
				if (1 === e$1.length) return e$1[0];
				t$3 = new Uint8Array(n$1), n$1 = 0;
				for (const r$1 of e$1) t$3.set(r$1, n$1), n$1 += r$1.length;
				return t$3;
			}(t$2);
			try {
				this.h.FS_unlink("llm.task");
			} catch {}
			this.h.FS_createDataFile("/", "llm.task", t$2, !0, !1, !1);
		}
		async ea(t$2) {
			if ("string" == typeof t$2) {
				const e$1 = new Image();
				e$1.src = t$2, e$1.crossOrigin = "Anonymous";
				try {
					await e$1.decode();
				} catch {
					throw Error(`Image from URL ${t$2} failed to load`);
				}
				return {
					image: e$1,
					width: e$1.naturalWidth,
					height: e$1.naturalHeight
				};
			}
			if (t$2 instanceof HTMLImageElement) {
				try {
					await t$2.decode();
				} catch {
					throw Error("Image from HTMLImageElement failed to load");
				}
				return {
					image: t$2,
					width: t$2.naturalWidth,
					height: t$2.naturalHeight
				};
			}
			return t$2 instanceof HTMLVideoElement ? {
				image: t$2,
				width: t$2.videoWidth,
				height: t$2.videoHeight
			} : t$2 instanceof VideoFrame ? {
				image: t$2,
				width: t$2.displayWidth,
				height: t$2.displayHeight
			} : {
				image: t$2,
				width: t$2.width,
				height: t$2.height
			};
		}
		async da(t$2) {
			if ("string" == typeof t$2) {
				const e$1 = await fetch(t$2);
				if (!e$1.ok) throw Error(`Audio fetch for ${t$2} had error: ${e$1.status}`);
				return t$2 = await e$1.arrayBuffer(), {
					audioSamples: (t$2 = await new AudioContext({ sampleRate: 16e3 }).decodeAudioData(t$2)).getChannelData(0),
					audioSampleRateHz: t$2.sampleRate
				};
			}
			return "object" == typeof t$2 && null != t$2 && "audioSamples" in t$2 && "audioSampleRateHz" in t$2 ? t$2 : {
				audioSamples: t$2.getChannelData(0),
				audioSampleRateHz: t$2.sampleRate
			};
		}
	};
}(function(t$1) {
	return class e$1 extends t$1 {
		static async ka(t$2, n$1) {
			let r$1;
			n$1 ||= await e$1.X();
			const i$1 = [];
			for (const e$2 of t$2?.requiredFeatures ?? []) n$1.features.has(e$2) ? i$1.push(e$2) : console.warn(`WebGPU feature ${e$2} is not supported.`);
			t$2 = {
				...t$2,
				requiredFeatures: i$1
			};
			try {
				r$1 = await n$1.requestDevice(t$2);
			} catch (t$3) {
				throw console.error("Unable to initialize WebGPU with the requested features."), t$3;
			}
			return (t$2 = r$1).adapterInfo || (t$2.adapterInfo = n$1.info), r$1;
		}
		static async X(t$2) {
			if (!(t$2 = await navigator.gpu.requestAdapter(t$2))) throw Error("Unable to request adapter from navigator.gpu; Ensure WebGPU is enabled.");
			return t$2;
		}
		fa(t$2) {
			if (e$2) "undefined" != typeof HTMLCanvasElement && e$2 instanceof HTMLCanvasElement && (e$2.id = "canvas_webgpu");
			else var e$2 = new OffscreenCanvas(1, 1);
			e$2.getContext("webgpu").configure({
				device: t$2,
				format: navigator.gpu.getPreferredCanvasFormat()
			}), this.h.preinitializedWebGPUDevice = t$2;
		}
		Z() {
			return this.h.ccall("closeGraph", "void", [], [], { async: !0 });
		}
	};
}(function(t$1) {
	return class extends t$1 {
		addStreamingReaderToInputSidePacket(t$2, e$1) {
			this.h.addStreamingReaderToInputSidePacket(((e$2, n$1, r$1) => jr(t$2, this.h, e$2, n$1, r$1)), e$1);
		}
	};
}(function(t$1) {
	return class extends t$1 {
		Y(t$2, e$1) {
			fr(this, "lora_model_ref_in", ((n$1) => {
				this.h._addRawDataSpanToInputStream(t$2.offset, t$2.size, n$1, e$1);
			}));
		}
	};
}(class extends gr {}))));
var Yr = class extends Kr {};
var Jr = class {
	constructor(t$1) {
		this.j = t$1, this.i = Xr, Xr++;
	}
}, Xr = 1;
var Qr = class {
	constructor() {
		let t$1, e$1;
		this.promise = new Promise(((n$1, r$1) => {
			t$1 = n$1, e$1 = r$1;
		})), this.resolve = t$1, this.reject = e$1;
	}
};
function Zr(t$1) {
	return 1 === t$1 ? 1 : t$1 + t$1 % 2;
}
async function ti() {
	const t$1 = await Yr.X({ powerPreference: "high-performance" });
	var e$1 = t$1.limits.maxBufferSize, n$1 = t$1.limits.maxStorageBufferBindingSize;
	return e$1 < 524550144 && console.warn(`This WebGPU device is unable to execute most LLM tasks, because the required maxBufferSize is usually at least 524550144, but your device only supports maxBufferSize of ${e$1}`), n$1 < 524550144 && console.warn(`The WebGPU device is unable to execute LLM tasks, because the required maxStorageBufferBindingSize is usually at least 524550144, but your device only supports maxStorageBufferBindingSize of ${n$1}`), e$1 = {
		requiredFeatures: ["shader-f16"],
		requiredLimits: {
			maxStorageBufferBindingSize: n$1,
			maxBufferSize: e$1,
			maxStorageBuffersPerShaderStage: t$1.limits.maxStorageBuffersPerShaderStage
		}
	}, t$1.features.has("subgroups") && (console.warn("Experimental Chromium WGSL subgroup support detected. Enabling this feature in the inference engine."), n$1 = ["shader-f16", "subgroups"], t$1.features.has("subgroups-f16") && n$1.push("subgroups-f16"), e$1.requiredFeatures = n$1), Yr.ka(e$1, t$1);
}
function ei(t$1) {
	if (t$1.D.length > 0) {
		const e$1 = [...t$1.D];
		if (t$1.D.length = 0, !t$1.o) throw e$1;
		t$1.o.reject(e$1), t$1.o = void 0;
	}
}
function ni(t$1) {
	const e$1 = function(t$2) {
		const e$2 = new tr();
		ye(e$2, 10, "text_in"), ye(e$2, 10, "token_cost_in"), ye(e$2, 10, "lora_model_id_to_apply_in"), ye(e$2, 10, "lora_model_ref_in"), ye(e$2, 10, "lora_model_id_to_load_in"), ye(e$2, 16, "streaming_reader"), ye(e$2, 15, "text_out"), ye(e$2, 15, "text_end"), ye(e$2, 15, "token_cost_out");
		var n$2 = new Yn();
		_e(n$2, 2, "TokenizerInputBuildCalculator"), ye(n$2, 3, "PROMPT:text_in"), ye(n$2, 3, "LORA_ID:lora_model_id_to_apply_in"), ye(n$2, 4, "prompt"), Zn(e$2, n$2), _e(n$2 = new Yn(), 2, "ModelDataCalculator"), ye(n$2, 6, "MODEL_DATA:__side_packet_1"), ye(n$2, 6, "MODEL_TYPE:model_type"), ye(n$2, 5, "READ_DATA_FN:streaming_reader"), ye(n$2, 3, "LORA_MODEL_SPAN:lora_model_ref_in"), ye(n$2, 3, "LORA_MODEL_ID:lora_model_id_to_load_in"), ye(n$2, 4, "LORA_DATA:lora_model_data"), Zn(e$2, n$2), _e(n$2 = new Yn(), 2, "Gpt2UnicodeMappingCalculator"), ye(n$2, 5, "MODEL_TYPE:model_type"), ye(n$2, 6, "BYTES_TO_UNICODE_MAPPING:tokenizer_mapping"), Zn(e$2, n$2), _e(n$2 = new Vn(), 1, "type.googleapis.com/odml.infra.proto.TokenizerCalculatorOptions");
		var r$1 = new $r(), i$1 = pe(t$2.i, 2);
		me(r$1, 1, i$1), _e(i$1 = new Hr(), 2, "spm_vocab_model"), i$1 = le(i$1);
		t: {
			qt(r$1);
			var o$1 = r$1.m, s$1 = 0 | o$1[M];
			if (null == i$1) {
				var a$1 = ae(o$1);
				if (4 !== ue(a$1, o$1, s$1)) break t;
				a$1.set(qr, 0);
			} else {
				const t$3 = ae(a$1 = o$1), e$3 = ue(t$3, a$1, s$1);
				4 !== e$3 && (e$3 && (s$1 = Zt(a$1, s$1, e$3)), t$3.set(qr, 4));
			}
			Zt(o$1, s$1, 4, i$1);
		}
		return i$1 && !$(i$1) && Kt(r$1.m), me(r$1, 3, 2), Mn(n$2, r$1.j()), _e(r$1 = new Yn(), 2, "TokenizerCalculator"), de(r$1, 8, Vn, n$2), ye(r$1, 5, "MODEL_DATA:__side_packet_1"), ye(r$1, 3, "PROMPT_AND_INPUT_OPTIONS:prompt"), ye(r$1, 5, "BYTES_TO_UNICODE_MAPPING:tokenizer_mapping"), ye(r$1, 6, "PROCESSOR_GETTER:__input_side_1"), ye(r$1, 4, "IDS_AND_INPUT_OPTIONS:__stream_0"), Zn(e$2, r$1), _e(n$2 = new Vn(), 1, "type.googleapis.com/odml.infra.proto.LlmGpuCalculatorOptions"), me(r$1 = new Cr(), 12, 3), _e(r$1, 1, "llm.tflite"), me(r$1, 14, 0), i$1 = Zr(pe(t$2.i, 5)), me(r$1, 22, i$1), i$1 = ce(t$2.i, Er, 3), he(r$1, 31, i$1), se(i$1 = new Dr(), 1, At(!0), !1), null != It(Jt(t$2.i, 6)) && (It(Jt(t$2.i, 6)) ?? !1) && se(i$1, 1, At(!1), !1), se(i$1, 2, At(!0), !1), se(i$1, 5, At(!0), !1), he(r$1, 10, i$1), i$1 = te(t$2.i, 4, jt, void 0 === K ? 2 : 4), oe(r$1, 29, i$1), i$1 = new Vr(), me(o$1 = new Mr(), 1, 1), a$1 = pe(t$2.i, 2), me(o$1, 2, a$1), he(i$1, 1, o$1), he(r$1, 20, i$1), Mn(n$2, r$1.j()), _e(r$1 = new Yn(), 2, "LlmGpuCalculator"), de(r$1, 8, Vn, n$2), ye(r$1, 3, "IDS_AND_INPUT_OPTIONS:__stream_0"), ye(r$1, 3, "FINISH:finish"), ye(r$1, 3, "LORA_DATA:lora_model_data"), ye(r$1, 5, "MODEL_DATA:__side_packet_1"), ye(r$1, 4, "DECODED_IDS:__stream_3"), ye(r$1, 4, "OUTPUT_END:__stream_4"), _e(n$2 = new qn(), 1, "FINISH"), se(n$2, 2, At(!0), !1), de(r$1, 13, qn, n$2), Zn(e$2, r$1), _e(n$2 = new Yn(), 2, "IsPacketPresentCalculator"), ye(n$2, 3, "__stream_4"), ye(n$2, 4, "text_end"), Zn(e$2, n$2), _e(n$2 = new Vn(), 1, "type.googleapis.com/odml.infra.proto.DetokenizerCalculatorOptions"), r$1 = new Rr(), t$2 = Zr(pe(t$2.i, 5)), me(r$1, 5, t$2), ye(r$1, 4, "<eos>"), ye(r$1, 4, "<|endoftext|>"), Mn(n$2, r$1.j()), _e(t$2 = new Yn(), 2, "DetokenizerCalculator"), de(t$2, 8, Vn, n$2), ye(t$2, 3, "IDS_AND_INPUT_OPTIONS:__stream_3"), ye(t$2, 5, "PROCESSOR_GETTER:__input_side_1"), ye(t$2, 5, "BYTES_TO_UNICODE_MAPPING:tokenizer_mapping"), ye(t$2, 5, "MODEL_DATA:__side_packet_1"), ye(t$2, 4, "FINISH_AND_INPUT_OPTIONS:finish"), ye(t$2, 4, "WORDS:text_out"), Zn(e$2, t$2), _e(t$2 = new Yn(), 2, "TokenCostCalculator"), ye(t$2, 3, "PROMPT:token_cost_in"), ye(t$2, 5, "PROCESSOR_GETTER:__input_side_1"), ye(t$2, 5, "BYTES_TO_UNICODE_MAPPING:tokenizer_mapping"), ye(t$2, 4, "NUM_TOKENS:token_cost_out"), Zn(e$2, t$2), e$2;
	}(t$1);
	t$1.j.attachStringVectorListener("text_out", ((e$2, n$2) => {
		e$2 = function(t$2, e$3) {
			return null == t$2 || 0 === t$2.length ? [] : t$2.map(((t$3) => (t$3 = (t$3 = t$3.replaceAll("▁", " ")).replaceAll("<0x0A>", "\n"), e$3 && (t$3 = t$3.trimStart()), t$3.split("\\[eod\\]", 1)[0])));
		}(e$2, 0 === t$1.G.length), e$2.forEach(((e$3, n$3) => {
			n$3 < pe(t$1.i, 5) && t$1.G[n$3].push(e$3);
		})), t$1.v && 0 === t$1.D.length && (t$1.A ? (e$2.length > pe(t$1.i, 5) && e$2.pop(), t$1.v(e$2, !1)) : t$1.v(e$2[0], !1)), vr(t$1, n$2);
	})), t$1.j.attachEmptyPacketListener("text_out", ((e$2) => {
		vr(t$1, e$2);
	})), t$1.j.attachBoolListener("text_end", ((e$2, n$2) => {
		vr(t$1, n$2);
		try {
			ei(t$1);
		} catch (e$3) {
			throw t$1.l = !1, e$3;
		}
		if (t$1.o && (t$1.o.resolve(t$1.G.map(((t$2) => t$2.join("")))), t$1.o = void 0), t$1.v) if (t$1.A) {
			for (e$2 = [], n$2 = 0; n$2 < pe(t$1.i, 5); n$2++) e$2.push("");
			t$1.v(e$2, !0);
		} else t$1.v("", !0);
		t$1.l = !1, t$1.A = void 0;
	})), t$1.j.attachEmptyPacketListener("text_end", ((e$2) => {
		t$1.l = !1, t$1.A = void 0, vr(t$1, e$2), ei(t$1), t$1.o && (t$1.o.resolve(t$1.G.map(((t$2) => t$2.join("")))), t$1.o = void 0);
	})), t$1.j.attachIntListener("token_cost_out", ((e$2, n$2) => {
		t$1.T = e$2, vr(t$1, n$2);
	})), t$1.U && t$1.j.addStreamingReaderToInputSidePacket(t$1.U, "streaming_reader");
	const n$1 = e$1.j();
	return t$1.C?.removeEventListener("uncapturederror", t$1.K), t$1.j.Z().then((() => {
		t$1.C?.addEventListener("uncapturederror", t$1.K), t$1.D.length = 0, t$1.setGraph(new Uint8Array(n$1), !0), t$1.finishProcessing();
	}));
}
function ri(t$1, e$1, n$1, r$1) {
	if (t$1.v = "function" == typeof n$1 ? n$1 : r$1, (r$1 = (e$1 = Array.isArray(e$1) ? e$1 : [e$1]).filter(((t$2) => xr(t$2))).length) > 0 && (null == kt(Jt(t$1.i, 7)) || pe(t$1.i, 7) < r$1)) throw Error(`maxNumImages is set to ${null != kt(Jt(t$1.i, 7)) ? pe(t$1.i, 7) : 0}, but the query included ${r$1} images.`);
	if ((r$1 = e$1.filter(((t$2) => Ur(t$2))).length) > 0 && (null == It(Jt(t$1.i, 8)) || !It(Jt(t$1.i, 8)))) throw Error(`supportAudio was not enabled, but the query included ${r$1} audio chunks.`);
	if (t$1.B) {
		if (t$1.A && pe(t$1.i, 5) > 1) throw Error("Multi-response generation is not supported for converted LLM models (.task format) yet, nor is it supported for multimodality. Please use the .bin format without multimodality or request only one response.");
		if (n$1 instanceof Jr) throw Error("LoRA is not supported for converted LLM models (.task format) yet, nor is it supported for multimodality. Please use the .bin format without multimodality to use LoRA.");
		return t$1.j.R(e$1, t$1.u, ((e$2, n$2) => {
			0 === t$1.D.length && t$1.v && (t$1.A ? t$1.v([e$2], n$2) : t$1.v(e$2, n$2));
		})).then(((e$2) => (ei(t$1), [e$2])));
	}
	if (t$1.l) throw Error("Previous invocation or loading is still ongoing.");
	for (t$1.l = !0, t$1.G.length = 0, r$1 = 0; r$1 < pe(t$1.i, 5); r$1++) t$1.G[r$1] = [];
	if (r$1 = t$1.I + 1, t$1.j.addStringToStream(e$1.join(""), "text_in", r$1), n$1 instanceof Jr) {
		if (n$1.j !== t$1) throw t$1.l = !1, t$1.A = void 0, Error("The LoRA model was not loaded by this LLM Inference task.");
		t$1.j.addUintToStream(n$1.i, "lora_model_id_to_apply_in", r$1);
	} else t$1.j.addEmptyPacketToStream("lora_model_id_to_apply_in", r$1);
	return t$1.finishProcessing(), t$1.o = new Qr(), t$1.o.promise;
}
var ii = class extends Sr {
	constructor(t$1, e$1) {
		if (super(new Yr(t$1, e$1)), this.G = [], this.O = this.B = this.l = !1, this.D = [], this.K = (t$2) => {
			if ((t$2 = t$2.error).message.match(/exceeds the max buffer size limit/)) throw Error(`Failed to run this LLM model because it requires a buffer size that exceeds the maximum size your device supports, but you could try a smaller LLM model or different device.\nWebGPU throws: "${t$2.message}"`);
			if (t$2.message.match(/is larger than the maximum storage buffer binding size/)) throw Error(`Failed to run this LLM model because it requires a storage buffer binding size that exceeds the maximum size your device supports, but you could try a smaller LLM model or different device.\nWebGPU throws: "${t$2.message}"`);
			this.D.push(t$2);
		}, this.i = new Ir(), Ar(this.i, new er()), this.u = new Er(), he(this.i, 3, this.u), ge(this.i, 2, 512), t$1 = this.u, !wt(2)) throw L("enum");
		se(t$1, 1, 2, 0), me(this.u, 2, 40), se(this.u, 3, St(1), 0), Qt(this.u, 5, Ot(0)), se(this.u, 4, St(.8), 0), ge(this.i, 5, 1);
	}
	async N(t$1) {
		if (this.l) throw Error("Cannot set options while loading or processing.");
		if (t$1.baseOptions?.modelAssetPath && t$1.baseOptions?.modelAssetBuffer) throw Error("Cannot set both baseOptions.modelAssetPath and baseOptions.modelAssetBuffer");
		let e$1;
		const n$1 = new Promise(((t$2) => {
			e$1 = t$2;
		}));
		if (t$1.baseOptions?.modelAssetPath) {
			var r$1 = await fetch(t$1.baseOptions.modelAssetPath.toString());
			if (!r$1.ok) throw Error(`Failed to fetch model: ${t$1.baseOptions.modelAssetPath} (${r$1.status})`);
			if (!r$1.body) throw Error(`Failed to fetch model: ${t$1.baseOptions.modelAssetPath} (no body)`);
			r$1 = r$1.body.getReader();
		} else t$1.baseOptions?.modelAssetBuffer instanceof Uint8Array ? r$1 = function(t$2) {
			return new ReadableStream({
				start() {},
				async pull(e$2) {
					e$2.enqueue(t$2), e$2.close();
				}
			});
		}(t$1.baseOptions.modelAssetBuffer).getReader() : t$1.baseOptions?.modelAssetBuffer instanceof ReadableStreamDefaultReader ? (r$1 = t$1.baseOptions.modelAssetBuffer, t$1.baseOptions.modelAssetBuffer = void 0) : e$1();
		if (!r$1) throw Error("No model asset provided.");
		{
			const [n$2, s$1] = ar(r$1);
			this.O = 1 === await async function(t$2) {
				const e$2 = [];
				let n$3;
				for (const [i$2, o$2] of cr) {
					const s$2 = i$2;
					var r$2 = o$2;
					[t$2, n$3] = ar(t$2), r$2 = await r$2(n$3), await n$3.cancel(), r$2 && e$2.push(s$2);
				}
				if (await t$2.cancel(), 0 === e$2.length) throw Error("No model format matched.");
				if (1 === e$2.length) return e$2[0];
				throw Error(`Multiple model formats matched: ${e$2}`);
			}(s$1);
			var i$1 = "maxNumImages" in t$1 && t$1.maxNumImages ? t$1.maxNumImages : 0;
			ge(this.i, 7, i$1);
			var o$1 = "supportAudio" in t$1 && !!t$1.supportAudio;
			Qt(this.i, 8, At(o$1)), this.O || i$1 > 0 || o$1 ? (this.B = !0, r$1 = n$2) : (this.B = !1, this.U = Or(n$2, e$1));
		}
		if (t$1.baseOptions?.gpuOptions?.device && (this.C && this.C.removeEventListener("uncapturederror", this.K), this.C = t$1.baseOptions.gpuOptions.device, this.j.fa(this.C), this.C.addEventListener("uncapturederror", this.K)), "maxTokens" in t$1 && ge(this.i, 2, t$1.maxTokens ?? 512), "topK" in t$1 && me(this.u, 2, t$1.topK ?? 40), "temperature" in t$1 && se(this.u, 4, St(t$1.temperature ?? .8), 0), "randomSeed" in t$1 && Qt(this.u, 5, Ot(t$1.randomSeed ?? 0)), "loraRanks" in t$1 && function(t$2, e$2) {
			oe(t$2, 4, e$2);
		}(this.i, t$1.loraRanks ?? []), "numResponses" in t$1) {
			if ((i$1 = t$1.numResponses ?? 1) < 1) throw Error("'numResponses' must be at least 1.");
			if (this.B && i$1 > 1) throw Error("'numResponses > 1' is not supported for converted LLM models yet, and is also not supported with multimodality.");
			ge(this.i, 5, i$1), o$1 = ce(this.i, Er, 3), i$1 > 1 && o$1 && (o$1.j() <= 1 || (Jt(o$1, 4, Et) ?? 0) <= 0) && console.warn("To generate multiple responses, it is expected topK > 1 and temperature > 0; otherwise, all the generated responses may be the same.");
		}
		return "forceF32" in t$1 && void 0 !== t$1.forceF32 && Qt(this.i, 6, At(t$1.forceF32)), this.B ? (this.j.V(), this.O ? this.j.aa(r$1, this.i).then((() => {
			ei(this);
		})) : this.j.createLlmInferenceEngine(r$1, this.i).then((() => {
			ei(this);
		}))) : (this.l = !0, t$1 = ni(this).then((() => {})), Promise.all([n$1, t$1]).then((() => {
			this.l = !1, ei(this);
		})));
	}
	get baseOptions() {
		return ce(this.i, er, 1);
	}
	set baseOptions(t$1) {
		Ar(this.i, t$1);
	}
	get isIdle() {
		return !this.l && !this.o;
	}
	R(t$1, e$1, n$1) {
		return pe(this.i, 5) > 1 && console.warn("'numResponses' is set larger than 1 and this function only returns the first response, so we recommend either using 'generateResponses()' to obtain multiple responses, or else setting 'numResponses' to 1 for better performance."), this.A = !1, ri(this, t$1, e$1, n$1).then(((t$2) => t$2[0]));
	}
	ca(t$1, e$1, n$1) {
		return this.A = !0, ri(this, t$1, e$1, n$1);
	}
	S(t$1) {
		if (t$1 = Array.isArray(t$1) ? t$1 : [t$1], this.B) return this.j.S(t$1);
		if (this.l) throw Error("Previous invocation or loading is still ongoing.");
		if (t$1.some(xr)) throw Error("sizeInTokens requires maxNumImages > 0 for images.");
		if (t$1.some(Ur)) throw Error("sizeInTokens requires supportAudio for audio.");
		return t$1 = t$1.join(""), this.l = !0, this.T = void 0, this.j.addStringToStream(t$1, "token_cost_in", this.I + 1), this.finishProcessing(), this.l = !1, this.T;
	}
	async ia(t$1) {
		if (this.B) throw Error("LoRA is not supported for converted LLM models (.task format) yet, nor is it supported for multimodality. Please use the old format (.bin) without multimodality to use LoRA.");
		if (this.l) throw Error("Cannot load LoRA model while loading or processing.");
		if (this.l = !0, t$1 instanceof Uint8Array) {
			var e$1 = new Fr(this.j.h, t$1.length);
			e$1.set(t$1), t$1 = e$1;
		} else t$1 = t$1 instanceof Blob ? await async function(t$2, e$2) {
			return Br(t$2, e$2.stream(), e$2.size);
		}(this.j.h, t$1) : await async function(t$2, e$2) {
			e$2 = await fetch(e$2.toString());
			const n$2 = Number(e$2.headers.get("content-length"));
			if (!e$2.body) throw Error("Response body is not available.");
			if (!n$2) throw Error("File size is 0.");
			return Br(t$2, e$2.body, n$2);
		}(this.j.h, t$1);
		e$1 = new Jr(this);
		const n$1 = this.I + 1;
		return this.j.Y(t$1, n$1), this.j.addUintToStream(e$1.i, "lora_model_id_to_load_in", n$1), this.finishProcessing(), Nr(t$1), vr(this, n$1), this.l = !1, e$1;
	}
	close() {
		this.B && this.j.V(), this.C?.removeEventListener("uncapturederror", this.K), super.close();
	}
};
ii.prototype.loadLoraModel = ii.prototype.ia, ii.prototype.sizeInTokens = ii.prototype.S, ii.prototype.generateResponses = ii.prototype.ca, ii.prototype.generateResponse = ii.prototype.R, ii.prototype.setOptions = ii.prototype.N, ii.createWebGpuDevice = ti, ii.createFromModelPath = async function(t$1, e$1) {
	return br(t$1, e$1 = { baseOptions: {
		gpuOptions: { device: await ti() },
		modelAssetPath: e$1
	} });
}, ii.createFromModelBuffer = async function(t$1, e$1) {
	return br(t$1, e$1 = { baseOptions: {
		gpuOptions: { device: await ti() },
		modelAssetBuffer: e$1
	} });
}, ii.createFromOptions = async function(t$1, e$1) {
	if (!e$1.baseOptions?.gpuOptions?.device) {
		const t$2 = await ti();
		e$1.baseOptions = e$1.baseOptions ?? {}, e$1.baseOptions.gpuOptions = e$1?.baseOptions?.gpuOptions ?? {}, e$1.baseOptions.gpuOptions.device = t$2;
	}
	return br(t$1, e$1);
};
var root = from_html(`<div class="aL2 aMd"><h1>Gemma 3 Web Explorer</h1> <div> </div> <div class="chat-interface"><textarea placeholder="Ask your fine-tuned model anything..." class="aMd"></textarea> <button class="aMd"> </button></div> <div class="aLH aMd"><strong>Output:</strong> <p class="response aMd"> </p></div></div>`);
function _page($$anchor, $$props) {
	push($$props, true);
	let llm = state(null);
	let prompt = state("");
	let responseText = state("System ready. Waiting for model initialization...");
	let isLoading = state(false);
	let isModelLoaded = state(false);
	const MODEL_PATH = "/models/functiongemma_finetuned_fp16_ekv1024.bin";
	user_effect(() => {
		async function initModel() {
			try {
				const genai = await sr.forGenAiTasks("https://cdn.jsdelivr.net/npm/@mediapipe/tasks-genai/wasm");
				set(llm, await ii.createFromOptions(genai, {
					baseOptions: { modelAssetPath: MODEL_PATH },
					maxTokens: 1024,
					temperature: .4,
					topK: 40
				}));
				set(isModelLoaded, true);
				set(responseText, "FunctionGemma is ready!");
			} catch (e$1) {
				console.error("Model load error:", e$1);
				set(responseText, `Error: ${e$1 instanceof Error ? e$1.message : "Failed to load model"}`);
			}
		}
		initModel();
		return () => {
			if (get(llm)) get(llm).close();
		};
	});
	async function generate() {
		if (!get(prompt) || !get(llm) || get(isLoading)) {
			console.log("promp:", get(prompt), get(llm), get(isLoading));
			return;
		}
		set(isLoading, true);
		set(responseText, "");
		try {
			await get(llm).generateResponse(get(prompt), (partialText, done) => {
				set(responseText, get(responseText) + partialText);
			});
		} catch (e$1) {
			set(responseText, "Inference failed. check console.");
			console.error(e$1);
		} finally {
			set(isLoading, false);
		}
	}
	var div = root();
	var div_1 = sibling(child(div), 2);
	let classes;
	var text = child(div_1, true);
	reset(div_1);
	var div_2 = sibling(div_1, 2);
	var textarea = child(div_2);
	remove_textarea_child(textarea);
	var button = sibling(textarea, 2);
	button.__click = generate;
	var text_1 = child(button, true);
	reset(button);
	reset(div_2);
	var div_3 = sibling(div_2, 2);
	var p$1 = sibling(child(div_3), 2);
	var text_2 = child(p$1, true);
	reset(p$1);
	reset(div_3);
	reset(div);
	template_effect(() => {
		classes = set_class(div_1, 1, "aLF aMd", null, classes, { aLG: get(isModelLoaded) });
		set_text(text, get(isModelLoaded) ? "● Local Model Active" : "○ Downloading Model...");
		textarea.disabled = !get(isModelLoaded) || get(isLoading);
		button.disabled = !get(isModelLoaded) || get(isLoading);
		set_text(text_1, get(isLoading) ? "Thinking..." : "Run Inference");
		set_text(text_2, get(responseText));
	});
	bind_value(textarea, () => get(prompt), ($$value) => set(prompt, $$value));
	append($$anchor, div);
	pop();
}
delegate(["click"]);
export { _page as component };
