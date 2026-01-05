import { B as tick$1, V as untrack$1, at as state, bt as false_default, c as writable, r as onMount$1, rt as set$1, t as index_client_exports, z as get$1 } from "./CTB4HzdN.js";
import { n as true_default, t as version } from "./D9PGjZ6Y.js";
var HttpError = class {
	constructor(status, body) {
		this.status = status;
		if (typeof body === "string") this.body = { message: body };
		else if (body) this.body = body;
		else this.body = { message: `Error: ${status}` };
	}
	toString() {
		return JSON.stringify(this.body);
	}
};
var Redirect = class {
	constructor(status, location$1) {
		this.status = status;
		this.location = location$1;
	}
};
var SvelteKitError = class extends Error {
	constructor(status, text, message) {
		super(message);
		this.status = status;
		this.text = text;
	}
};
new URL("sveltekit-internal://");
function normalize_path(path, trailing_slash) {
	if (path === "/" || trailing_slash === "ignore") return path;
	if (trailing_slash === "never") return path.endsWith("/") ? path.slice(0, -1) : path;
	else if (trailing_slash === "always" && !path.endsWith("/")) return path + "/";
	return path;
}
function decode_pathname(pathname) {
	return pathname.split("%25").map(decodeURI).join("%25");
}
function decode_params(params) {
	for (const key in params) params[key] = decodeURIComponent(params[key]);
	return params;
}
function strip_hash({ href }) {
	return href.split("#")[0];
}
function hash(...values) {
	let hash$1 = 5381;
	for (const value of values) if (typeof value === "string") {
		let i = value.length;
		while (i) hash$1 = hash$1 * 33 ^ value.charCodeAt(--i);
	} else if (ArrayBuffer.isView(value)) {
		const buffer = new Uint8Array(value.buffer, value.byteOffset, value.byteLength);
		let i = buffer.length;
		while (i) hash$1 = hash$1 * 33 ^ buffer[--i];
	} else throw new TypeError("value must be a string or TypedArray");
	return (hash$1 >>> 0).toString(36);
}
new TextEncoder();
new TextDecoder();
function base64_decode(encoded) {
	const binary = atob(encoded);
	const bytes = new Uint8Array(binary.length);
	for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
	return bytes;
}
var native_fetch = window.fetch;
window.fetch = (input, init) => {
	if ((input instanceof Request ? input.method : init?.method || "GET") !== "GET") cache.delete(build_selector(input));
	return native_fetch(input, init);
};
var cache = /* @__PURE__ */ new Map();
function initial_fetch(resource, opts) {
	const selector = build_selector(resource, opts);
	const script = document.querySelector(selector);
	if (script?.textContent) {
		script.remove();
		let { body, ...init } = JSON.parse(script.textContent);
		const ttl = script.getAttribute("data-ttl");
		if (ttl) cache.set(selector, {
			body,
			init,
			ttl: 1e3 * Number(ttl)
		});
		if (script.getAttribute("data-b64") !== null) body = base64_decode(body);
		return Promise.resolve(new Response(body, init));
	}
	return window.fetch(resource, opts);
}
function subsequent_fetch(resource, resolved, opts) {
	if (cache.size > 0) {
		const selector = build_selector(resource, opts);
		const cached = cache.get(selector);
		if (cached) {
			if (performance.now() < cached.ttl && [
				"default",
				"force-cache",
				"only-if-cached",
				void 0
			].includes(opts?.cache)) return new Response(cached.body, cached.init);
			cache.delete(selector);
		}
	}
	return window.fetch(resolved, opts);
}
function build_selector(resource, opts) {
	let selector = `script[data-sveltekit-fetched][data-url=${JSON.stringify(resource instanceof Request ? resource.url : resource)}]`;
	if (opts?.headers || opts?.body) {
		const values = [];
		if (opts.headers) values.push([...new Headers(opts.headers)].join(","));
		if (opts.body && (typeof opts.body === "string" || ArrayBuffer.isView(opts.body))) values.push(opts.body);
		selector += `[data-hash="${hash(...values)}"]`;
	}
	return selector;
}
var param_pattern = /^(\[)?(\.\.\.)?(\w+)(?:=(\w+))?(\])?$/;
function parse_route_id(id) {
	const params = [];
	return {
		pattern: id === "/" ? /^\/$/ : /* @__PURE__ */ new RegExp(`^${get_route_segments(id).map((segment) => {
			const rest_match = /^\[\.\.\.(\w+)(?:=(\w+))?\]$/.exec(segment);
			if (rest_match) {
				params.push({
					name: rest_match[1],
					matcher: rest_match[2],
					optional: false,
					rest: true,
					chained: true
				});
				return "(?:/([^]*))?";
			}
			const optional_match = /^\[\[(\w+)(?:=(\w+))?\]\]$/.exec(segment);
			if (optional_match) {
				params.push({
					name: optional_match[1],
					matcher: optional_match[2],
					optional: true,
					rest: false,
					chained: true
				});
				return "(?:/([^/]+))?";
			}
			if (!segment) return;
			const parts = segment.split(/\[(.+?)\](?!\])/);
			return "/" + parts.map((content, i) => {
				if (i % 2) {
					if (content.startsWith("x+")) return escape(String.fromCharCode(parseInt(content.slice(2), 16)));
					if (content.startsWith("u+")) return escape(String.fromCharCode(...content.slice(2).split("-").map((code) => parseInt(code, 16))));
					const [, is_optional, is_rest, name, matcher] = param_pattern.exec(content);
					params.push({
						name,
						matcher,
						optional: !!is_optional,
						rest: !!is_rest,
						chained: is_rest ? i === 1 && parts[0] === "" : false
					});
					return is_rest ? "([^]*?)" : is_optional ? "([^/]*)?" : "([^/]+?)";
				}
				return escape(content);
			}).join("");
		}).join("")}/?$`),
		params
	};
}
function affects_path(segment) {
	return segment !== "" && !/^\([^)]+\)$/.test(segment);
}
function get_route_segments(route) {
	return route.slice(1).split("/").filter(affects_path);
}
function exec(match, params, matchers) {
	const result = {};
	const values = match.slice(1);
	const values_needing_match = values.filter((value) => value !== void 0);
	let buffered = 0;
	for (let i = 0; i < params.length; i += 1) {
		const param = params[i];
		let value = values[i - buffered];
		if (param.chained && param.rest && buffered) {
			value = values.slice(i - buffered, i + 1).filter((s) => s).join("/");
			buffered = 0;
		}
		if (value === void 0) {
			if (param.rest) result[param.name] = "";
			continue;
		}
		if (!param.matcher || matchers[param.matcher](value)) {
			result[param.name] = value;
			const next_param = params[i + 1];
			const next_value = values[i + 1];
			if (next_param && !next_param.rest && next_param.optional && next_value && param.chained) buffered = 0;
			if (!next_param && !next_value && Object.keys(result).length === values_needing_match.length) buffered = 0;
			continue;
		}
		if (param.optional && param.chained) {
			buffered++;
			continue;
		}
		return;
	}
	if (buffered) return;
	return result;
}
function escape(str) {
	return str.normalize().replace(/[[\]]/g, "\\$&").replace(/%/g, "%25").replace(/\//g, "%2[Ff]").replace(/\?/g, "%3[Ff]").replace(/#/g, "%23").replace(/[.*+?^${}()|\\]/g, "\\$&");
}
function parse({ nodes, server_loads, dictionary, matchers }) {
	const layouts_with_server_load = new Set(server_loads);
	return Object.entries(dictionary).map(([id, [leaf, layouts, errors]]) => {
		const { pattern, params } = parse_route_id(id);
		const route = {
			id,
			exec: (path) => {
				const match = pattern.exec(path);
				if (match) return exec(match, params, matchers);
			},
			errors: [1, ...errors || []].map((n) => nodes[n]),
			layouts: [0, ...layouts || []].map(create_layout_loader),
			leaf: create_leaf_loader(leaf)
		};
		route.errors.length = route.layouts.length = Math.max(route.errors.length, route.layouts.length);
		return route;
	});
	function create_leaf_loader(id) {
		const uses_server_data = id < 0;
		if (uses_server_data) id = ~id;
		return [uses_server_data, nodes[id]];
	}
	function create_layout_loader(id) {
		return id === void 0 ? id : [layouts_with_server_load.has(id), nodes[id]];
	}
}
/* @__NO_SIDE_EFFECTS__ */
function get(key, parse$1 = JSON.parse) {
	try {
		return parse$1(sessionStorage[key]);
	} catch {}
}
function set(key, value, stringify = JSON.stringify) {
	const data = stringify(value);
	try {
		sessionStorage[key] = data;
	} catch {}
}
const base = globalThis.__sveltekit_iqq7mf?.base ?? "";
const assets = globalThis.__sveltekit_iqq7mf?.assets ?? base ?? "";
const SNAPSHOT_KEY = "sveltekit:snapshot";
const SCROLL_KEY = "sveltekit:scroll";
const STATES_KEY = "sveltekit:states";
const HISTORY_INDEX = "sveltekit:history";
const NAVIGATION_INDEX = "sveltekit:navigation";
const PRELOAD_PRIORITIES = {
	tap: 1,
	hover: 2,
	viewport: 3,
	eager: 4,
	off: -1,
	false: -1
};
const origin = location.origin;
function resolve_url(url) {
	if (url instanceof URL) return url;
	let baseURI = document.baseURI;
	if (!baseURI) {
		const baseTags = document.getElementsByTagName("base");
		baseURI = baseTags.length ? baseTags[0].href : document.URL;
	}
	return new URL(url, baseURI);
}
function scroll_state() {
	return {
		x: pageXOffset,
		y: pageYOffset
	};
}
function link_option(element, name) {
	return element.getAttribute(`data-sveltekit-${name}`);
}
var levels = {
	...PRELOAD_PRIORITIES,
	"": PRELOAD_PRIORITIES.hover
};
function parent_element(element) {
	let parent = element.assignedSlot ?? element.parentNode;
	if (parent?.nodeType === 11) parent = parent.host;
	return parent;
}
function find_anchor(element, target$1) {
	while (element && element !== target$1) {
		if (element.nodeName.toUpperCase() === "A" && element.hasAttribute("href")) return element;
		element = parent_element(element);
	}
}
function get_link_info(a, base$1, uses_hash_router) {
	let url;
	try {
		url = new URL(a instanceof SVGAElement ? a.href.baseVal : a.href, document.baseURI);
		if (uses_hash_router && url.hash.match(/^#[^/]/)) {
			const route = location.hash.split("#")[1] || "/";
			url.hash = `#${route}${url.hash}`;
		}
	} catch {}
	const target$1 = a instanceof SVGAElement ? a.target.baseVal : a.target;
	const external = !url || !!target$1 || is_external_url(url, base$1, uses_hash_router) || (a.getAttribute("rel") || "").split(/\s+/).includes("external");
	const download = url?.origin === origin && a.hasAttribute("download");
	return {
		url,
		external,
		target: target$1,
		download
	};
}
function get_router_options(element) {
	let keepfocus = null;
	let noscroll = null;
	let preload_code = null;
	let preload_data = null;
	let reload = null;
	let replace_state = null;
	let el = element;
	while (el && el !== document.documentElement) {
		if (preload_code === null) preload_code = link_option(el, "preload-code");
		if (preload_data === null) preload_data = link_option(el, "preload-data");
		if (keepfocus === null) keepfocus = link_option(el, "keepfocus");
		if (noscroll === null) noscroll = link_option(el, "noscroll");
		if (reload === null) reload = link_option(el, "reload");
		if (replace_state === null) replace_state = link_option(el, "replacestate");
		el = parent_element(el);
	}
	function get_option_state(value) {
		switch (value) {
			case "":
			case "true": return true;
			case "off":
			case "false": return false;
			default: return;
		}
	}
	return {
		preload_code: levels[preload_code ?? "off"],
		preload_data: levels[preload_data ?? "off"],
		keepfocus: get_option_state(keepfocus),
		noscroll: get_option_state(noscroll),
		reload: get_option_state(reload),
		replace_state: get_option_state(replace_state)
	};
}
function notifiable_store(value) {
	const store = writable(value);
	let ready = true;
	function notify() {
		ready = true;
		store.update((val) => val);
	}
	function set$2(new_value) {
		ready = false;
		store.set(new_value);
	}
	function subscribe(run) {
		let old_value;
		return store.subscribe((new_value) => {
			if (old_value === void 0 || ready && new_value !== old_value) run(old_value = new_value);
		});
	}
	return {
		notify,
		set: set$2,
		subscribe
	};
}
const updated_listener = { v: () => {} };
function create_updated_store() {
	const { set: set$2, subscribe } = writable(false);
	let timeout;
	async function check() {
		clearTimeout(timeout);
		try {
			const res = await fetch(`${assets}/_app/version.json`, { headers: {
				pragma: "no-cache",
				"cache-control": "no-cache"
			} });
			if (!res.ok) return false;
			const updated$1 = (await res.json()).version !== version;
			if (updated$1) {
				set$2(true);
				updated_listener.v();
				clearTimeout(timeout);
			}
			return updated$1;
		} catch {
			return false;
		}
	}
	return {
		subscribe,
		check
	};
}
function is_external_url(url, base$1, hash_routing) {
	if (url.origin !== origin || !url.pathname.startsWith(base$1)) return true;
	if (hash_routing) {
		if (url.pathname === base$1 + "/" || url.pathname === base$1 + "/index.html") return false;
		if (url.protocol === "file:" && url.pathname.replace(/\/[^/]+\.html?$/, "") === base$1) return false;
		return true;
	}
	return false;
}
function load_css(deps) {}
function validator(expected) {
	function validate(module, file) {
		if (!module) return;
		for (const key in module) {
			if (key[0] === "_" || expected.has(key)) continue;
			const values = [...expected.values()];
			const hint = hint_for_supported_files(key, file?.slice(file.lastIndexOf("."))) ?? `valid exports are ${values.join(", ")}, or anything with a '_' prefix`;
			throw new Error(`Invalid export '${key}'${file ? ` in ${file}` : ""} (${hint})`);
		}
	}
	return validate;
}
function hint_for_supported_files(key, ext = ".js") {
	const supported_files = [];
	if (valid_layout_exports.has(key)) supported_files.push(`+layout${ext}`);
	if (valid_page_exports.has(key)) supported_files.push(`+page${ext}`);
	if (valid_layout_server_exports.has(key)) supported_files.push(`+layout.server${ext}`);
	if (valid_page_server_exports.has(key)) supported_files.push(`+page.server${ext}`);
	if (valid_server_exports.has(key)) supported_files.push(`+server${ext}`);
	if (supported_files.length > 0) return `'${key}' is a valid export in ${supported_files.slice(0, -1).join(", ")}${supported_files.length > 1 ? " or " : ""}${supported_files.at(-1)}`;
}
var valid_layout_exports = new Set([
	"load",
	"prerender",
	"csr",
	"ssr",
	"trailingSlash",
	"config"
]);
var valid_page_exports = new Set([...valid_layout_exports, "entries"]);
var valid_layout_server_exports = new Set([...valid_layout_exports]);
var valid_page_server_exports = new Set([
	...valid_layout_server_exports,
	"actions",
	"entries"
]);
var valid_server_exports = new Set([
	"GET",
	"POST",
	"PATCH",
	"PUT",
	"DELETE",
	"OPTIONS",
	"HEAD",
	"fallback",
	"prerender",
	"trailingSlash",
	"config",
	"entries"
]);
validator(valid_layout_exports);
validator(valid_page_exports);
validator(valid_layout_server_exports);
validator(valid_page_server_exports);
validator(valid_server_exports);
function compact(arr) {
	return arr.filter((val) => val != null);
}
function get_status(error) {
	return error instanceof HttpError || error instanceof SvelteKitError ? error.status : 500;
}
function get_message(error) {
	return error instanceof SvelteKitError ? error.text : "Internal Error";
}
let page;
let navigating;
let updated;
if (onMount$1.toString().includes("$$") || /function \w+\(\) \{\}/.test(onMount$1.toString())) {
	page = {
		data: {},
		form: null,
		error: null,
		params: {},
		route: { id: null },
		state: {},
		status: -1,
		url: new URL("https://example.com")
	};
	navigating = { current: null };
	updated = { current: false };
} else {
	page = new class Page {
		#data = state({});
		get data() {
			return get$1(this.#data);
		}
		set data(value) {
			set$1(this.#data, value);
		}
		#form = state(null);
		get form() {
			return get$1(this.#form);
		}
		set form(value) {
			set$1(this.#form, value);
		}
		#error = state(null);
		get error() {
			return get$1(this.#error);
		}
		set error(value) {
			set$1(this.#error, value);
		}
		#params = state({});
		get params() {
			return get$1(this.#params);
		}
		set params(value) {
			set$1(this.#params, value);
		}
		#route = state({ id: null });
		get route() {
			return get$1(this.#route);
		}
		set route(value) {
			set$1(this.#route, value);
		}
		#state = state({});
		get state() {
			return get$1(this.#state);
		}
		set state(value) {
			set$1(this.#state, value);
		}
		#status = state(-1);
		get status() {
			return get$1(this.#status);
		}
		set status(value) {
			set$1(this.#status, value);
		}
		#url = state(new URL("https://example.com"));
		get url() {
			return get$1(this.#url);
		}
		set url(value) {
			set$1(this.#url, value);
		}
	}();
	navigating = new class Navigating {
		#current = state(null);
		get current() {
			return get$1(this.#current);
		}
		set current(value) {
			set$1(this.#current, value);
		}
	}();
	updated = new class Updated {
		#current = state(false);
		get current() {
			return get$1(this.#current);
		}
		set current(value) {
			set$1(this.#current, value);
		}
	}();
	updated_listener.v = () => updated.current = true;
}
function update(new_page) {
	Object.assign(page, new_page);
}
var { onMount, tick } = index_client_exports;
var ICON_REL_ATTRIBUTES = new Set([
	"icon",
	"shortcut icon",
	"apple-touch-icon"
]);
var scroll_positions = /* @__PURE__ */ get("sveltekit:scroll") ?? {};
var snapshots = /* @__PURE__ */ get("sveltekit:snapshot") ?? {};
const stores = {
	url: /* @__PURE__ */ notifiable_store({}),
	page: /* @__PURE__ */ notifiable_store({}),
	navigating: /* @__PURE__ */ writable(null),
	updated: /* @__PURE__ */ create_updated_store()
};
function update_scroll_positions(index) {
	scroll_positions[index] = scroll_state();
}
function clear_onward_history(current_history_index$1, current_navigation_index$1) {
	let i = current_history_index$1 + 1;
	while (scroll_positions[i]) {
		delete scroll_positions[i];
		i += 1;
	}
	i = current_navigation_index$1 + 1;
	while (snapshots[i]) {
		delete snapshots[i];
		i += 1;
	}
}
function native_navigation(url, replace = false) {
	if (replace) location.replace(url.href);
	else location.href = url.href;
	return new Promise(() => {});
}
async function update_service_worker() {
	if ("serviceWorker" in navigator) {
		const registration = await navigator.serviceWorker.getRegistration(base || "/");
		if (registration) await registration.update();
	}
}
function noop() {}
var routes;
var default_layout_loader;
var default_error_loader;
var container;
var target;
let app;
var invalidated = [];
var components = [];
var load_cache = null;
var reroute_cache = /* @__PURE__ */ new Map();
var before_navigate_callbacks = /* @__PURE__ */ new Set();
var on_navigate_callbacks = /* @__PURE__ */ new Set();
var after_navigate_callbacks = /* @__PURE__ */ new Set();
var current = {
	branch: [],
	error: null,
	url: null
};
var hydrated = false;
var started = false;
var autoscroll = true;
var is_navigating = false;
var hash_navigating = false;
var has_navigated = false;
var force_invalidation = false;
var root;
var current_history_index;
var current_navigation_index;
var token;
var preload_tokens = /* @__PURE__ */ new Set();
const query_map = /* @__PURE__ */ new Map();
async function start(_app, _target, hydrate) {
	if (globalThis.__sveltekit_iqq7mf?.data) globalThis.__sveltekit_iqq7mf.data;
	if (document.URL !== location.href) location.href = location.href;
	app = _app;
	await _app.hooks.init?.();
	routes = parse(_app);
	container = document.documentElement;
	target = _target;
	default_layout_loader = _app.nodes[0];
	default_error_loader = _app.nodes[1];
	default_layout_loader();
	default_error_loader();
	current_history_index = history.state?.[HISTORY_INDEX];
	current_navigation_index = history.state?.[NAVIGATION_INDEX];
	if (!current_history_index) {
		current_history_index = current_navigation_index = Date.now();
		history.replaceState({
			...history.state,
			[HISTORY_INDEX]: current_history_index,
			[NAVIGATION_INDEX]: current_navigation_index
		}, "");
	}
	const scroll = scroll_positions[current_history_index];
	function restore_scroll() {
		if (scroll) {
			history.scrollRestoration = "manual";
			scrollTo(scroll.x, scroll.y);
		}
	}
	if (hydrate) {
		restore_scroll();
		await _hydrate(target, hydrate);
	} else {
		await navigate({
			type: "enter",
			url: resolve_url(app.hash ? decode_hash(new URL(location.href)) : location.href),
			replace_state: true
		});
		restore_scroll();
	}
	_start_router();
}
function reset_invalidation() {
	invalidated.length = 0;
	force_invalidation = false;
}
function capture_snapshot(index) {
	if (components.some((c) => c?.snapshot)) snapshots[index] = components.map((c) => c?.snapshot?.capture());
}
function restore_snapshot(index) {
	snapshots[index]?.forEach((value, i) => {
		components[i]?.snapshot?.restore(value);
	});
}
function persist_state() {
	update_scroll_positions(current_history_index);
	set(SCROLL_KEY, scroll_positions);
	capture_snapshot(current_navigation_index);
	set(SNAPSHOT_KEY, snapshots);
}
async function _goto(url, options, redirect_count, nav_token) {
	let query_keys;
	if (options.invalidateAll) load_cache = null;
	await navigate({
		type: "goto",
		url: resolve_url(url),
		keepfocus: options.keepFocus,
		noscroll: options.noScroll,
		replace_state: options.replaceState,
		state: options.state,
		redirect_count,
		nav_token,
		accept: () => {
			if (options.invalidateAll) {
				force_invalidation = true;
				query_keys = [...query_map.keys()];
			}
			if (options.invalidate) options.invalidate.forEach(push_invalidated);
		}
	});
	if (options.invalidateAll) tick$1().then(tick$1).then(() => {
		query_map.forEach(({ resource }, key) => {
			if (query_keys?.includes(key)) resource.refresh?.();
		});
	});
}
async function _preload_data(intent) {
	if (intent.id !== load_cache?.id) {
		const preload = {};
		preload_tokens.add(preload);
		load_cache = {
			id: intent.id,
			token: preload,
			promise: load_route({
				...intent,
				preload
			}).then((result) => {
				preload_tokens.delete(preload);
				if (result.type === "loaded" && result.state.error) load_cache = null;
				return result;
			})
		};
	}
	return load_cache.promise;
}
async function _preload_code(url) {
	const route = (await get_navigation_intent(url, false))?.route;
	if (route) await Promise.all([...route.layouts, route.leaf].map((load) => load?.[1]()));
}
function initialize(result, target$1, hydrate) {
	current = result.state;
	const style = document.querySelector("style[data-sveltekit]");
	if (style) style.remove();
	Object.assign(page, result.props.page);
	root = new app.root({
		target: target$1,
		props: {
			...result.props,
			stores,
			components
		},
		hydrate,
		sync: false
	});
	restore_snapshot(current_navigation_index);
	if (hydrate) {
		const navigation = {
			from: null,
			to: {
				params: current.params,
				route: { id: current.route?.id ?? null },
				url: new URL(location.href)
			},
			willUnload: false,
			type: "enter",
			complete: Promise.resolve()
		};
		after_navigate_callbacks.forEach((fn) => fn(navigation));
	}
	started = true;
}
function get_navigation_result_from_branch({ url, params, branch, status, error, route, form }) {
	let slash = "never";
	if (base && (url.pathname === base || url.pathname === base + "/")) slash = "always";
	else for (const node of branch) if (node?.slash !== void 0) slash = node.slash;
	url.pathname = normalize_path(url.pathname, slash);
	url.search = url.search;
	const result = {
		type: "loaded",
		state: {
			url,
			params,
			branch,
			error,
			route
		},
		props: {
			constructors: compact(branch).map((branch_node) => branch_node.node.component),
			page: clone_page(page)
		}
	};
	if (form !== void 0) result.props.form = form;
	let data = {};
	let data_changed = !page;
	let p = 0;
	for (let i = 0; i < Math.max(branch.length, current.branch.length); i += 1) {
		const node = branch[i];
		const prev = current.branch[i];
		if (node?.data !== prev?.data) data_changed = true;
		if (!node) continue;
		data = {
			...data,
			...node.data
		};
		if (data_changed) result.props[`data_${p}`] = data;
		p += 1;
	}
	if (!current.url || url.href !== current.url.href || current.error !== error || form !== void 0 && form !== page.form || data_changed) result.props.page = {
		error,
		params,
		route: { id: route?.id ?? null },
		state: {},
		status,
		url: new URL(url),
		form: form ?? null,
		data: data_changed ? data : page.data
	};
	return result;
}
async function load_node({ loader, parent, url, params, route, server_data_node }) {
	let data = null;
	const uses = {
		dependencies: /* @__PURE__ */ new Set(),
		params: /* @__PURE__ */ new Set(),
		parent: false,
		route: false,
		url: false,
		search_params: /* @__PURE__ */ new Set()
	};
	const node = await loader();
	return {
		node,
		loader,
		server: server_data_node,
		universal: node.universal?.load ? {
			type: "data",
			data,
			uses
		} : null,
		data: data ?? server_data_node?.data ?? null,
		slash: node.universal?.trailingSlash ?? server_data_node?.slash
	};
}
function resolve_fetch_url(input, init, url) {
	let requested = input instanceof Request ? input.url : input;
	const resolved = new URL(requested, url);
	if (resolved.origin === url.origin) requested = resolved.href.slice(url.origin.length);
	return {
		resolved,
		promise: started ? subsequent_fetch(requested, resolved.href, init) : initial_fetch(requested, init)
	};
}
function has_changed(parent_changed, route_changed, url_changed, search_params_changed, uses, params) {
	if (force_invalidation) return true;
	if (!uses) return false;
	if (uses.parent && parent_changed) return true;
	if (uses.route && route_changed) return true;
	if (uses.url && url_changed) return true;
	for (const tracked_params of uses.search_params) if (search_params_changed.has(tracked_params)) return true;
	for (const param of uses.params) if (params[param] !== current.params[param]) return true;
	for (const href of uses.dependencies) if (invalidated.some((fn) => fn(new URL(href)))) return true;
	return false;
}
function create_data_node(node, previous) {
	if (node?.type === "data") return node;
	if (node?.type === "skip") return previous ?? null;
	return null;
}
function diff_search_params(old_url, new_url) {
	if (!old_url) return new Set(new_url.searchParams.keys());
	const changed = new Set([...old_url.searchParams.keys(), ...new_url.searchParams.keys()]);
	for (const key of changed) {
		const old_values = old_url.searchParams.getAll(key);
		const new_values = new_url.searchParams.getAll(key);
		if (old_values.every((value) => new_values.includes(value)) && new_values.every((value) => old_values.includes(value))) changed.delete(key);
	}
	return changed;
}
function preload_error({ error, url, route, params }) {
	return {
		type: "loaded",
		state: {
			error,
			url,
			route,
			params,
			branch: []
		},
		props: {
			page: clone_page(page),
			constructors: []
		}
	};
}
async function load_route({ id, invalidating, url, params, route, preload }) {
	if (load_cache?.id === id) {
		preload_tokens.delete(load_cache.token);
		return load_cache.promise;
	}
	const { errors, layouts, leaf } = route;
	const loaders = [...layouts, leaf];
	errors.forEach((loader) => loader?.().catch(() => {}));
	loaders.forEach((loader) => loader?.[1]().catch(() => {}));
	let server_data = null;
	const url_changed = current.url ? id !== get_page_key(current.url) : false;
	const route_changed = current.route ? route.id !== current.route.id : false;
	const search_params_changed = diff_search_params(current.url, url);
	const server_data_nodes = server_data?.nodes;
	let parent_changed = false;
	const branch_promises = loaders.map(async (loader, i) => {
		if (!loader) return;
		const previous = current.branch[i];
		const server_data_node = server_data_nodes?.[i];
		if ((!server_data_node || server_data_node.type === "skip") && loader[1] === previous?.loader && !has_changed(parent_changed, route_changed, url_changed, search_params_changed, previous.universal?.uses, params)) return previous;
		parent_changed = true;
		if (server_data_node?.type === "error") throw server_data_node;
		return load_node({
			loader: loader[1],
			url,
			params,
			route,
			parent: async () => {
				const data = {};
				for (let j = 0; j < i; j += 1) Object.assign(data, (await branch_promises[j])?.data);
				return data;
			},
			server_data_node: create_data_node(server_data_node === void 0 && loader[0] ? { type: "skip" } : server_data_node ?? null, loader[0] ? previous?.server : void 0)
		});
	});
	for (const p of branch_promises) p.catch(() => {});
	const branch = [];
	for (let i = 0; i < loaders.length; i += 1) if (loaders[i]) try {
		branch.push(await branch_promises[i]);
	} catch (err) {
		if (err instanceof Redirect) return {
			type: "redirect",
			location: err.location
		};
		if (preload_tokens.has(preload)) return preload_error({
			error: await handle_error(err, {
				params,
				url,
				route: { id: route.id }
			}),
			url,
			params,
			route
		});
		let status = get_status(err);
		let error;
		if (server_data_nodes?.includes(err)) {
			status = err.status ?? status;
			error = err.error;
		} else if (err instanceof HttpError) error = err.body;
		else {
			if (await stores.updated.check()) {
				await update_service_worker();
				return await native_navigation(url);
			}
			error = await handle_error(err, {
				params,
				url,
				route: { id: route.id }
			});
		}
		const error_load = await load_nearest_error_page(i, branch, errors);
		if (error_load) return get_navigation_result_from_branch({
			url,
			params,
			branch: branch.slice(0, error_load.idx).concat(error_load.node),
			status,
			error,
			route
		});
		else return await server_fallback(url, { id: route.id }, error, status);
	}
	else branch.push(void 0);
	return get_navigation_result_from_branch({
		url,
		params,
		branch,
		status: 200,
		error: null,
		route,
		form: invalidating ? void 0 : null
	});
}
async function load_nearest_error_page(i, branch, errors) {
	while (i--) if (errors[i]) {
		let j = i;
		while (!branch[j]) j -= 1;
		try {
			return {
				idx: j + 1,
				node: {
					node: await errors[i](),
					loader: errors[i],
					data: {},
					server: null,
					universal: null
				}
			};
		} catch {
			continue;
		}
	}
}
async function load_root_error_page({ status, error, url, route }) {
	const params = {};
	let server_data_node = null;
	try {
		return get_navigation_result_from_branch({
			url,
			params,
			branch: [await load_node({
				loader: default_layout_loader,
				url,
				params,
				route,
				parent: () => Promise.resolve({}),
				server_data_node: create_data_node(server_data_node)
			}), {
				node: await default_error_loader(),
				loader: default_error_loader,
				universal: null,
				server: null,
				data: null
			}],
			status,
			error,
			route: null
		});
	} catch (error$1) {
		if (error$1 instanceof Redirect) return _goto(new URL(error$1.location, location.href), {}, 0);
		throw error$1;
	}
}
async function get_rerouted_url(url) {
	const href = url.href;
	if (reroute_cache.has(href)) return reroute_cache.get(href);
	let rerouted;
	try {
		const promise = (async () => {
			let rerouted$1 = await app.hooks.reroute({
				url: new URL(url),
				fetch: async (input, init) => {
					return resolve_fetch_url(input, init, url).promise;
				}
			}) ?? url;
			if (typeof rerouted$1 === "string") {
				const tmp = new URL(url);
				if (app.hash) tmp.hash = rerouted$1;
				else tmp.pathname = rerouted$1;
				rerouted$1 = tmp;
			}
			return rerouted$1;
		})();
		reroute_cache.set(href, promise);
		rerouted = await promise;
	} catch (e) {
		reroute_cache.delete(href);
		return;
	}
	return rerouted;
}
async function get_navigation_intent(url, invalidating) {
	if (!url) return;
	if (is_external_url(url, base, app.hash)) return;
	{
		const rerouted = await get_rerouted_url(url);
		if (!rerouted) return;
		const path = get_url_path(rerouted);
		for (const route of routes) {
			const params = route.exec(path);
			if (params) return {
				id: get_page_key(url),
				invalidating,
				route,
				params: decode_params(params),
				url
			};
		}
	}
}
function get_url_path(url) {
	return decode_pathname(app.hash ? url.hash.replace(/^#/, "").replace(/[?#].+/, "") : url.pathname.slice(base.length)) || "/";
}
function get_page_key(url) {
	return (app.hash ? url.hash.replace(/^#/, "") : url.pathname) + url.search;
}
function _before_navigate({ url, type, intent, delta, event }) {
	let should_block = false;
	const nav = create_navigation(current, intent, url, type);
	if (delta !== void 0) nav.navigation.delta = delta;
	if (event !== void 0) nav.navigation.event = event;
	const cancellable = {
		...nav.navigation,
		cancel: () => {
			should_block = true;
			nav.reject(/* @__PURE__ */ new Error("navigation cancelled"));
		}
	};
	if (!is_navigating) before_navigate_callbacks.forEach((fn) => fn(cancellable));
	return should_block ? null : nav;
}
async function navigate({ type, url, popped, keepfocus, noscroll, replace_state, state: state$1 = {}, redirect_count = 0, nav_token = {}, accept = noop, block = noop, event }) {
	const prev_token = token;
	token = nav_token;
	const intent = await get_navigation_intent(url, false);
	const nav = type === "enter" ? create_navigation(current, intent, url, type) : _before_navigate({
		url,
		type,
		delta: popped?.delta,
		intent,
		event
	});
	if (!nav) {
		block();
		if (token === nav_token) token = prev_token;
		return;
	}
	const previous_history_index = current_history_index;
	const previous_navigation_index = current_navigation_index;
	accept();
	is_navigating = true;
	if (started && nav.navigation.type !== "enter") stores.navigating.set(navigating.current = nav.navigation);
	let navigation_result = intent && await load_route(intent);
	if (!navigation_result) if (is_external_url(url, base, app.hash)) return await native_navigation(url, replace_state);
	else navigation_result = await server_fallback(url, { id: null }, await handle_error(new SvelteKitError(404, "Not Found", `Not found: ${url.pathname}`), {
		url,
		params: {},
		route: { id: null }
	}), 404, replace_state);
	url = intent?.url || url;
	if (token !== nav_token) {
		nav.reject(/* @__PURE__ */ new Error("navigation aborted"));
		return false;
	}
	if (navigation_result.type === "redirect") {
		if (redirect_count < 20) {
			await navigate({
				type,
				url: new URL(navigation_result.location, url),
				popped,
				keepfocus,
				noscroll,
				replace_state,
				state: state$1,
				redirect_count: redirect_count + 1,
				nav_token
			});
			nav.fulfil(void 0);
			return;
		}
		navigation_result = await load_root_error_page({
			status: 500,
			error: await handle_error(/* @__PURE__ */ new Error("Redirect loop"), {
				url,
				params: {},
				route: { id: null }
			}),
			url,
			route: { id: null }
		});
	} else if (navigation_result.props.page.status >= 400) {
		if (await stores.updated.check()) {
			await update_service_worker();
			await native_navigation(url, replace_state);
		}
	}
	reset_invalidation();
	update_scroll_positions(previous_history_index);
	capture_snapshot(previous_navigation_index);
	if (navigation_result.props.page.url.pathname !== url.pathname) url.pathname = navigation_result.props.page.url.pathname;
	state$1 = popped ? popped.state : state$1;
	if (!popped) {
		const change = replace_state ? 0 : 1;
		const entry = {
			[HISTORY_INDEX]: current_history_index += change,
			[NAVIGATION_INDEX]: current_navigation_index += change,
			[STATES_KEY]: state$1
		};
		(replace_state ? history.replaceState : history.pushState).call(history, entry, "", url);
		if (!replace_state) clear_onward_history(current_history_index, current_navigation_index);
	}
	load_cache = null;
	navigation_result.props.page.state = state$1;
	if (started) {
		const after_navigate = (await Promise.all(Array.from(on_navigate_callbacks, (fn) => fn(nav.navigation)))).filter((value) => typeof value === "function");
		if (after_navigate.length > 0) {
			function cleanup() {
				after_navigate.forEach((fn) => {
					after_navigate_callbacks.delete(fn);
				});
			}
			after_navigate.push(cleanup);
			after_navigate.forEach((fn) => {
				after_navigate_callbacks.add(fn);
			});
		}
		current = navigation_result.state;
		if (navigation_result.props.page) navigation_result.props.page.url = url;
		root.$set(navigation_result.props);
		update(navigation_result.props.page);
		has_navigated = true;
	} else initialize(navigation_result, target, false);
	const { activeElement } = document;
	await tick();
	let scroll = popped ? popped.scroll : noscroll ? scroll_state() : null;
	if (autoscroll) {
		const deep_linked = url.hash && document.getElementById(get_id(url));
		if (scroll) scrollTo(scroll.x, scroll.y);
		else if (deep_linked) {
			deep_linked.scrollIntoView();
			const { top, left } = deep_linked.getBoundingClientRect();
			scroll = {
				x: pageXOffset + left,
				y: pageYOffset + top
			};
		} else scrollTo(0, 0);
	}
	const changed_focus = document.activeElement !== activeElement && document.activeElement !== document.body;
	if (!keepfocus && !changed_focus) reset_focus(url, scroll);
	autoscroll = true;
	if (navigation_result.props.page) Object.assign(page, navigation_result.props.page);
	is_navigating = false;
	if (type === "popstate") restore_snapshot(current_navigation_index);
	nav.fulfil(void 0);
	after_navigate_callbacks.forEach((fn) => fn(nav.navigation));
	stores.navigating.set(navigating.current = null);
}
async function server_fallback(url, route, error, status, replace_state) {
	if (url.origin === origin && url.pathname === location.pathname && !hydrated) return await load_root_error_page({
		status,
		error,
		url,
		route
	});
	return await native_navigation(url, replace_state);
}
function setup_preload() {
	let mousemove_timeout;
	let current_a;
	let current_priority;
	container.addEventListener("mousemove", (event) => {
		const target$1 = event.target;
		clearTimeout(mousemove_timeout);
		mousemove_timeout = setTimeout(() => {
			preload(target$1, PRELOAD_PRIORITIES.hover);
		}, 20);
	});
	function tap(event) {
		if (event.defaultPrevented) return;
		preload(event.composedPath()[0], PRELOAD_PRIORITIES.tap);
	}
	container.addEventListener("mousedown", tap);
	container.addEventListener("touchstart", tap, { passive: true });
	const observer = new IntersectionObserver((entries) => {
		for (const entry of entries) if (entry.isIntersecting) {
			_preload_code(new URL(entry.target.href));
			observer.unobserve(entry.target);
		}
	}, { threshold: 0 });
	async function preload(element, priority) {
		const a = find_anchor(element, container);
		if (!a || a === current_a && priority >= current_priority) return;
		const { url, external, download } = get_link_info(a, base, app.hash);
		if (external || download) return;
		const options = get_router_options(a);
		const same_url = url && get_page_key(current.url) === get_page_key(url);
		if (options.reload || same_url) return;
		if (priority <= options.preload_data) {
			current_a = a;
			current_priority = PRELOAD_PRIORITIES.tap;
			const intent = await get_navigation_intent(url, false);
			if (!intent) return;
			_preload_data(intent);
		} else if (priority <= options.preload_code) {
			current_a = a;
			current_priority = priority;
			_preload_code(url);
		}
	}
	function after_navigate() {
		observer.disconnect();
		for (const a of container.querySelectorAll("a")) {
			const { url, external, download } = get_link_info(a, base, app.hash);
			if (external || download) continue;
			const options = get_router_options(a);
			if (options.reload) continue;
			if (options.preload_code === PRELOAD_PRIORITIES.viewport) observer.observe(a);
			if (options.preload_code === PRELOAD_PRIORITIES.eager) _preload_code(url);
		}
	}
	after_navigate_callbacks.add(after_navigate);
	after_navigate();
}
function handle_error(error, event) {
	if (error instanceof HttpError) return error.body;
	const status = get_status(error);
	const message = get_message(error);
	return app.hooks.handleError({
		error,
		event,
		status,
		message
	}) ?? { message };
}
function goto(url, opts = {}) {
	url = new URL(resolve_url(url));
	if (url.origin !== origin) return Promise.reject(/* @__PURE__ */ new Error("goto: invalid URL"));
	return _goto(url, opts, 0);
}
function push_invalidated(resource) {
	if (typeof resource === "function") invalidated.push(resource);
	else {
		const { href } = new URL(resource, location.href);
		invalidated.push((url) => url.href === href);
	}
}
function _start_router() {
	history.scrollRestoration = "manual";
	addEventListener("beforeunload", (e) => {
		let should_block = false;
		persist_state();
		if (!is_navigating) {
			const nav = create_navigation(current, void 0, null, "leave");
			const navigation = {
				...nav.navigation,
				cancel: () => {
					should_block = true;
					nav.reject(/* @__PURE__ */ new Error("navigation cancelled"));
				}
			};
			before_navigate_callbacks.forEach((fn) => fn(navigation));
		}
		if (should_block) {
			e.preventDefault();
			e.returnValue = "";
		} else history.scrollRestoration = "auto";
	});
	addEventListener("visibilitychange", () => {
		if (document.visibilityState === "hidden") persist_state();
	});
	if (!navigator.connection?.saveData) setup_preload();
	container.addEventListener("click", async (event) => {
		if (event.button || event.which !== 1) return;
		if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
		if (event.defaultPrevented) return;
		const a = find_anchor(event.composedPath()[0], container);
		if (!a) return;
		const { url, external, target: target$1, download } = get_link_info(a, base, app.hash);
		if (!url) return;
		if (target$1 === "_parent" || target$1 === "_top") {
			if (window.parent !== window) return;
		} else if (target$1 && target$1 !== "_self") return;
		const options = get_router_options(a);
		if (!(a instanceof SVGAElement) && url.protocol !== location.protocol && !(url.protocol === "https:" || url.protocol === "http:")) return;
		if (download) return;
		const [nonhash, hash$1] = (app.hash ? url.hash.replace(/^#/, "") : url.href).split("#");
		const same_pathname = nonhash === strip_hash(location);
		if (external || options.reload && (!same_pathname || !hash$1)) {
			if (_before_navigate({
				url,
				type: "link",
				event
			})) is_navigating = true;
			else event.preventDefault();
			return;
		}
		if (hash$1 !== void 0 && same_pathname) {
			const [, current_hash] = current.url.href.split("#");
			if (current_hash === hash$1) {
				event.preventDefault();
				if (hash$1 === "" || hash$1 === "top" && a.ownerDocument.getElementById("top") === null) scrollTo({ top: 0 });
				else {
					const element = a.ownerDocument.getElementById(decodeURIComponent(hash$1));
					if (element) {
						element.scrollIntoView();
						element.focus();
					}
				}
				return;
			}
			hash_navigating = true;
			update_scroll_positions(current_history_index);
			update_url(url);
			if (!options.replace_state) return;
			hash_navigating = false;
		}
		event.preventDefault();
		await new Promise((fulfil) => {
			requestAnimationFrame(() => {
				setTimeout(fulfil, 0);
			});
			setTimeout(fulfil, 100);
		});
		await navigate({
			type: "link",
			url,
			keepfocus: options.keepfocus,
			noscroll: options.noscroll,
			replace_state: options.replace_state ?? url.href === location.href,
			event
		});
	});
	container.addEventListener("submit", (event) => {
		if (event.defaultPrevented) return;
		const form = HTMLFormElement.prototype.cloneNode.call(event.target);
		const submitter = event.submitter;
		if ((submitter?.formTarget || form.target) === "_blank") return;
		if ((submitter?.formMethod || form.method) !== "get") return;
		const url = new URL(submitter?.hasAttribute("formaction") && submitter?.formAction || form.action);
		if (is_external_url(url, base, false)) return;
		const event_form = event.target;
		const options = get_router_options(event_form);
		if (options.reload) return;
		event.preventDefault();
		event.stopPropagation();
		const data = new FormData(event_form, submitter);
		url.search = new URLSearchParams(data).toString();
		navigate({
			type: "form",
			url,
			keepfocus: options.keepfocus,
			noscroll: options.noscroll,
			replace_state: options.replace_state ?? url.href === location.href,
			event
		});
	});
	addEventListener("popstate", async (event) => {
		if (resetting_focus) return;
		if (event.state?.["sveltekit:history"]) {
			const history_index = event.state[HISTORY_INDEX];
			token = {};
			if (history_index === current_history_index) return;
			const scroll = scroll_positions[history_index];
			const state$1 = event.state["sveltekit:states"] ?? {};
			const url = new URL(event.state["sveltekit:pageurl"] ?? location.href);
			const navigation_index = event.state[NAVIGATION_INDEX];
			const is_hash_change = current.url ? strip_hash(location) === strip_hash(current.url) : false;
			if (navigation_index === current_navigation_index && (has_navigated || is_hash_change)) {
				if (state$1 !== page.state) page.state = state$1;
				update_url(url);
				scroll_positions[current_history_index] = scroll_state();
				if (scroll) scrollTo(scroll.x, scroll.y);
				current_history_index = history_index;
				return;
			}
			const delta = history_index - current_history_index;
			await navigate({
				type: "popstate",
				url,
				popped: {
					state: state$1,
					scroll,
					delta
				},
				accept: () => {
					current_history_index = history_index;
					current_navigation_index = navigation_index;
				},
				block: () => {
					history.go(-delta);
				},
				nav_token: token,
				event
			});
		} else if (!hash_navigating) {
			update_url(new URL(location.href));
			if (app.hash) location.reload();
		}
	});
	addEventListener("hashchange", () => {
		if (hash_navigating) {
			hash_navigating = false;
			history.replaceState({
				...history.state,
				[HISTORY_INDEX]: ++current_history_index,
				[NAVIGATION_INDEX]: current_navigation_index
			}, "", location.href);
		}
	});
	for (const link of document.querySelectorAll("link")) if (ICON_REL_ATTRIBUTES.has(link.rel)) link.href = link.href;
	addEventListener("pageshow", (event) => {
		if (event.persisted) stores.navigating.set(navigating.current = null);
	});
	function update_url(url) {
		current.url = page.url = url;
		stores.page.set(clone_page(page));
		stores.page.notify();
	}
}
async function _hydrate(target$1, { status = 200, error, node_ids, params, route, server_route, data: server_data_nodes, form }) {
	hydrated = true;
	const url = new URL(location.href);
	let parsed_route;
	({params = {}, route = { id: null }} = await get_navigation_intent(url, false) || {});
	parsed_route = routes.find(({ id }) => id === route.id);
	let result;
	let hydrate = true;
	try {
		const branch_promises = node_ids.map(async (n, i) => {
			const server_data_node = server_data_nodes[i];
			if (server_data_node?.uses) server_data_node.uses = deserialize_uses(server_data_node.uses);
			return load_node({
				loader: app.nodes[n],
				url,
				params,
				route,
				parent: async () => {
					const data = {};
					for (let j = 0; j < i; j += 1) Object.assign(data, (await branch_promises[j]).data);
					return data;
				},
				server_data_node: create_data_node(server_data_node)
			});
		});
		const branch = await Promise.all(branch_promises);
		if (parsed_route) {
			const layouts = parsed_route.layouts;
			for (let i = 0; i < layouts.length; i++) if (!layouts[i]) branch.splice(i, 0, void 0);
		}
		result = get_navigation_result_from_branch({
			url,
			params,
			branch,
			status,
			error,
			form,
			route: parsed_route ?? null
		});
	} catch (error$1) {
		if (error$1 instanceof Redirect) {
			await native_navigation(new URL(error$1.location, location.href));
			return;
		}
		result = await load_root_error_page({
			status: get_status(error$1),
			error: await handle_error(error$1, {
				url,
				params,
				route
			}),
			url,
			route
		});
		target$1.textContent = "";
		hydrate = false;
	}
	if (result.props.page) result.props.page.state = {};
	initialize(result, target$1, hydrate);
}
function deserialize_uses(uses) {
	return {
		dependencies: new Set(uses?.dependencies ?? []),
		params: new Set(uses?.params ?? []),
		parent: !!uses?.parent,
		route: !!uses?.route,
		url: !!uses?.url,
		search_params: new Set(uses?.search_params ?? [])
	};
}
var resetting_focus = false;
function reset_focus(url, scroll = null) {
	const autofocus = document.querySelector("[autofocus]");
	if (autofocus) autofocus.focus();
	else {
		const id = get_id(url);
		if (id && document.getElementById(id)) {
			const { x, y } = scroll ?? scroll_state();
			setTimeout(() => {
				const history_state = history.state;
				resetting_focus = true;
				location.replace(`#${id}`);
				if (app.hash) location.replace(url.hash);
				history.replaceState(history_state, "", url.hash);
				scrollTo(x, y);
				resetting_focus = false;
			});
		} else {
			const root$1 = document.body;
			const tabindex = root$1.getAttribute("tabindex");
			root$1.tabIndex = -1;
			root$1.focus({
				preventScroll: true,
				focusVisible: false
			});
			if (tabindex !== null) root$1.setAttribute("tabindex", tabindex);
			else root$1.removeAttribute("tabindex");
		}
		const selection = getSelection();
		if (selection && selection.type !== "None") {
			const ranges = [];
			for (let i = 0; i < selection.rangeCount; i += 1) ranges.push(selection.getRangeAt(i));
			setTimeout(() => {
				if (selection.rangeCount !== ranges.length) return;
				for (let i = 0; i < selection.rangeCount; i += 1) {
					const a = ranges[i];
					const b = selection.getRangeAt(i);
					if (a.commonAncestorContainer !== b.commonAncestorContainer || a.startContainer !== b.startContainer || a.endContainer !== b.endContainer || a.startOffset !== b.startOffset || a.endOffset !== b.endOffset) return;
				}
				selection.removeAllRanges();
			});
		}
	}
}
function create_navigation(current$1, intent, url, type) {
	let fulfil;
	let reject;
	const complete = new Promise((f, r) => {
		fulfil = f;
		reject = r;
	});
	complete.catch(() => {});
	return {
		navigation: {
			from: {
				params: current$1.params,
				route: { id: current$1.route?.id ?? null },
				url: current$1.url
			},
			to: url && {
				params: intent?.params ?? null,
				route: { id: intent?.route?.id ?? null },
				url
			},
			willUnload: !intent,
			type,
			complete
		},
		fulfil,
		reject
	};
}
function clone_page(page$1) {
	return {
		data: page$1.data,
		error: page$1.error,
		form: page$1.form,
		params: page$1.params,
		route: page$1.route,
		state: page$1.state,
		status: page$1.status,
		url: page$1.url
	};
}
function decode_hash(url) {
	const new_url = new URL(url);
	new_url.hash = decodeURIComponent(url.hash);
	return new_url;
}
function get_id(url) {
	let id;
	if (app.hash) {
		const [, , second] = url.hash.split("#", 3);
		id = second ?? "";
	} else id = url.hash.slice(1);
	return decodeURIComponent(id);
}
export { page as a, navigating as i, start as n, updated as o, stores as r, load_css as s, goto as t };
