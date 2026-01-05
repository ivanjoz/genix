var __create = Object.create;
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __getProtoOf = Object.getPrototypeOf;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __commonJSMin = (cb, mod) => () => (mod || cb((mod = { exports: {} }).exports, mod), mod.exports);
var __exportAll = (all, symbols) => {
	let target = {};
	for (var name in all) __defProp(target, name, {
		get: all[name],
		enumerable: true
	});
	if (symbols) __defProp(target, Symbol.toStringTag, { value: "Module" });
	return target;
};
var __copyProps = (to, from, except, desc) => {
	if (from && typeof from === "object" || typeof from === "function") for (var keys = __getOwnPropNames(from), i = 0, n = keys.length, key; i < n; i++) {
		key = keys[i];
		if (!__hasOwnProp.call(to, key) && key !== except) __defProp(to, key, {
			get: ((k) => from[k]).bind(null, key),
			enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable
		});
	}
	return to;
};
var __reExport = (target, mod, secondTarget) => (__copyProps(target, mod, "default"), secondTarget && __copyProps(secondTarget, mod, "default"));
var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", {
	value: mod,
	enumerable: true
}) : target, mod));
var __require = /* @__PURE__ */ ((x) => typeof require !== "undefined" ? require : typeof Proxy !== "undefined" ? new Proxy(x, { get: (a, b) => (typeof require !== "undefined" ? require : a)[b] }) : x)(function(x) {
	if (typeof require !== "undefined") return require.apply(this, arguments);
	throw Error("Calling `require` for \"" + x + "\" in an environment that doesn't expose the `require` function.");
});
var false_default = false;
var is_array = Array.isArray;
var index_of = Array.prototype.indexOf;
var array_from = Array.from;
Object.keys;
var define_property = Object.defineProperty;
var get_descriptor = Object.getOwnPropertyDescriptor;
var get_descriptors = Object.getOwnPropertyDescriptors;
var object_prototype = Object.prototype;
var array_prototype = Array.prototype;
var get_prototype_of = Object.getPrototypeOf;
var is_extensible = Object.isExtensible;
const noop = () => {};
function run(fn) {
	return fn();
}
function run_all(arr) {
	for (var i = 0; i < arr.length; i++) arr[i]();
}
function deferred() {
	var resolve;
	var reject;
	return {
		promise: new Promise((res, rej) => {
			resolve = res;
			reject = rej;
		}),
		resolve,
		reject
	};
}
function to_array(value, n) {
	if (Array.isArray(value)) return value;
	if (n === void 0 || !(Symbol.iterator in value)) return Array.from(value);
	const array = [];
	for (const element of value) {
		array.push(element);
		if (array.length === n) break;
	}
	return array;
}
const CLEAN = 1024;
const DIRTY = 2048;
const MAYBE_DIRTY = 4096;
const INERT = 8192;
const DESTROYED = 16384;
const EFFECT_RAN = 32768;
const EFFECT_TRANSPARENT = 65536;
const HEAD_EFFECT = 1 << 18;
const EFFECT_PRESERVED = 1 << 19;
const USER_EFFECT = 1 << 20;
const WAS_MARKED = 32768;
const REACTION_IS_UPDATING = 1 << 21;
const ASYNC = 1 << 22;
const ERROR_VALUE = 1 << 23;
const STATE_SYMBOL = Symbol("$state");
const LEGACY_PROPS = Symbol("legacy props");
const LOADING_ATTR_SYMBOL = Symbol("");
const PROXY_PATH_SYMBOL = Symbol("proxy path");
const STALE_REACTION = new class StaleReactionError extends Error {
	name = "StaleReactionError";
	message = "The reaction that called `getAbortSignal()` was re-run or destroyed";
}();
function lifecycle_outside_component(name) {
	throw new Error(`https://svelte.dev/e/lifecycle_outside_component`);
}
function missing_context() {
	throw new Error(`https://svelte.dev/e/missing_context`);
}
function async_derived_orphan() {
	throw new Error(`https://svelte.dev/e/async_derived_orphan`);
}
function effect_in_teardown(rune) {
	throw new Error(`https://svelte.dev/e/effect_in_teardown`);
}
function effect_in_unowned_derived() {
	throw new Error(`https://svelte.dev/e/effect_in_unowned_derived`);
}
function effect_orphan(rune) {
	throw new Error(`https://svelte.dev/e/effect_orphan`);
}
function effect_update_depth_exceeded() {
	throw new Error(`https://svelte.dev/e/effect_update_depth_exceeded`);
}
function experimental_async_fork() {
	throw new Error(`https://svelte.dev/e/experimental_async_fork`);
}
function fork_discarded() {
	throw new Error(`https://svelte.dev/e/fork_discarded`);
}
function fork_timing() {
	throw new Error(`https://svelte.dev/e/fork_timing`);
}
function get_abort_signal_outside_reaction() {
	throw new Error(`https://svelte.dev/e/get_abort_signal_outside_reaction`);
}
function hydration_failed() {
	throw new Error(`https://svelte.dev/e/hydration_failed`);
}
function lifecycle_legacy_only(name) {
	throw new Error(`https://svelte.dev/e/lifecycle_legacy_only`);
}
function props_invalid_value(key) {
	throw new Error(`https://svelte.dev/e/props_invalid_value`);
}
function state_descriptors_fixed() {
	throw new Error(`https://svelte.dev/e/state_descriptors_fixed`);
}
function state_prototype_fixed() {
	throw new Error(`https://svelte.dev/e/state_prototype_fixed`);
}
function state_unsafe_mutation() {
	throw new Error(`https://svelte.dev/e/state_unsafe_mutation`);
}
function svelte_boundary_reset_onerror() {
	throw new Error(`https://svelte.dev/e/svelte_boundary_reset_onerror`);
}
const HYDRATION_ERROR = {};
const UNINITIALIZED = Symbol();
const NAMESPACE_HTML = "http://www.w3.org/1999/xhtml";
function hydration_mismatch(location) {
	console.warn(`https://svelte.dev/e/hydration_mismatch`);
}
function select_multiple_invalid_value() {
	console.warn(`https://svelte.dev/e/select_multiple_invalid_value`);
}
function svelte_boundary_reset_noop() {
	console.warn(`https://svelte.dev/e/svelte_boundary_reset_noop`);
}
let hydrating = false;
function set_hydrating(value) {
	hydrating = value;
}
let hydrate_node;
function set_hydrate_node(node) {
	if (node === null) {
		hydration_mismatch();
		throw HYDRATION_ERROR;
	}
	return hydrate_node = node;
}
function hydrate_next() {
	return set_hydrate_node(/* @__PURE__ */ get_next_sibling(hydrate_node));
}
function reset(node) {
	if (!hydrating) return;
	if (/* @__PURE__ */ get_next_sibling(hydrate_node) !== null) {
		hydration_mismatch();
		throw HYDRATION_ERROR;
	}
	hydrate_node = node;
}
function next(count = 1) {
	if (hydrating) {
		var i = count;
		var node = hydrate_node;
		while (i--) node = /* @__PURE__ */ get_next_sibling(node);
		hydrate_node = node;
	}
}
function skip_nodes(remove = true) {
	var depth = 0;
	var node = hydrate_node;
	while (true) {
		if (node.nodeType === 8) {
			var data = node.data;
			if (data === "]") {
				if (depth === 0) return node;
				depth -= 1;
			} else if (data === "[" || data === "[!") depth += 1;
		}
		var next$1 = /* @__PURE__ */ get_next_sibling(node);
		if (remove) node.remove();
		node = next$1;
	}
}
function read_hydration_instruction(node) {
	if (!node || node.nodeType !== 8) {
		hydration_mismatch();
		throw HYDRATION_ERROR;
	}
	return node.data;
}
function equals(value) {
	return value === this.v;
}
function safe_not_equal(a, b) {
	return a != a ? b == b : a !== b || a !== null && typeof a === "object" || typeof a === "function";
}
function safe_equals(value) {
	return !safe_not_equal(value, this.v);
}
let legacy_mode_flag = false;
function enable_legacy_mode_flag() {
	legacy_mode_flag = true;
}
var empty = [];
function snapshot(value, skip_warning = false, no_tojson = false) {
	return clone(value, /* @__PURE__ */ new Map(), "", empty, null, no_tojson);
}
function clone(value, cloned, path, paths, original = null, no_tojson = false) {
	if (typeof value === "object" && value !== null) {
		var unwrapped = cloned.get(value);
		if (unwrapped !== void 0) return unwrapped;
		if (value instanceof Map) return new Map(value);
		if (value instanceof Set) return new Set(value);
		if (is_array(value)) {
			var copy = Array(value.length);
			cloned.set(value, copy);
			if (original !== null) cloned.set(original, copy);
			for (var i = 0; i < value.length; i += 1) {
				var element = value[i];
				if (i in value) copy[i] = clone(element, cloned, path, paths, null, no_tojson);
			}
			return copy;
		}
		if (get_prototype_of(value) === object_prototype) {
			copy = {};
			cloned.set(value, copy);
			if (original !== null) cloned.set(original, copy);
			for (var key in value) copy[key] = clone(value[key], cloned, path, paths, null, no_tojson);
			return copy;
		}
		if (value instanceof Date) return structuredClone(value);
		if (typeof value.toJSON === "function" && !no_tojson) return clone(value.toJSON(), cloned, path, paths, value);
	}
	if (value instanceof EventTarget) return value;
	try {
		return structuredClone(value);
	} catch (e) {
		return value;
	}
}
function tag(source$1, label$1) {
	source$1.label = label$1;
	tag_proxy(source$1.v, label$1);
	return source$1;
}
function tag_proxy(value, label$1) {
	value?.[PROXY_PATH_SYMBOL]?.(label$1);
	return value;
}
function label(value) {
	if (typeof value === "symbol") return `Symbol(${value.description})`;
	if (typeof value === "function") return "<function>";
	if (typeof value === "object" && value) return "<object>";
	return String(value);
}
let component_context = null;
function set_component_context(context) {
	component_context = context;
}
function createContext() {
	const key = {};
	return [() => {
		if (!hasContext(key)) missing_context();
		return getContext(key);
	}, (context) => setContext(key, context)];
}
function getContext(key) {
	return get_or_init_context_map("getContext").get(key);
}
function setContext(key, context) {
	get_or_init_context_map("setContext").set(key, context);
	return context;
}
function hasContext(key) {
	return get_or_init_context_map("hasContext").has(key);
}
function getAllContexts() {
	return get_or_init_context_map("getAllContexts");
}
function push(props, runes = false, fn) {
	component_context = {
		p: component_context,
		i: false,
		c: null,
		e: null,
		s: props,
		x: null,
		l: legacy_mode_flag && !runes ? {
			s: null,
			u: null,
			$: []
		} : null
	};
}
function pop(component$1) {
	var context = component_context;
	var effects = context.e;
	if (effects !== null) {
		context.e = null;
		for (var fn of effects) create_user_effect(fn);
	}
	if (component$1 !== void 0) context.x = component$1;
	context.i = true;
	component_context = context.p;
	return component$1 ?? {};
}
function is_runes() {
	return !legacy_mode_flag || component_context !== null && component_context.l === null;
}
function get_or_init_context_map(name) {
	if (component_context === null) lifecycle_outside_component(name);
	return component_context.c ??= new Map(get_parent_context(component_context) || void 0);
}
function get_parent_context(component_context$1) {
	let parent = component_context$1.p;
	while (parent !== null) {
		const context_map = parent.c;
		if (context_map !== null) return context_map;
		parent = parent.p;
	}
	return null;
}
var micro_tasks = [];
function run_micro_tasks() {
	var tasks = micro_tasks;
	micro_tasks = [];
	run_all(tasks);
}
function queue_micro_task(fn) {
	if (micro_tasks.length === 0 && !is_flushing_sync) {
		var tasks = micro_tasks;
		queueMicrotask(() => {
			if (tasks === micro_tasks) run_micro_tasks();
		});
	}
	micro_tasks.push(fn);
}
function flush_tasks() {
	while (micro_tasks.length > 0) run_micro_tasks();
}
function handle_error(error) {
	var effect$1 = active_effect;
	if (effect$1 === null) {
		active_reaction.f |= ERROR_VALUE;
		return error;
	}
	if ((effect$1.f & 32768) === 0) {
		if ((effect$1.f & 128) === 0) throw error;
		effect$1.b.error(error);
	} else invoke_error_boundary(error, effect$1);
}
function invoke_error_boundary(error, effect$1) {
	while (effect$1 !== null) {
		if ((effect$1.f & 128) !== 0) try {
			effect$1.b.error(error);
			return;
		} catch (e) {
			error = e;
		}
		effect$1 = effect$1.parent;
	}
	throw error;
}
var batches = /* @__PURE__ */ new Set();
let current_batch = null;
let previous_batch = null;
let batch_values = null;
let effect_pending_updates = /* @__PURE__ */ new Set();
var queued_root_effects = [];
var last_scheduled_effect = null;
var is_flushing = false;
let is_flushing_sync = false;
var Batch = class Batch {
	committed = false;
	current = /* @__PURE__ */ new Map();
	previous = /* @__PURE__ */ new Map();
	#commit_callbacks = /* @__PURE__ */ new Set();
	#discard_callbacks = /* @__PURE__ */ new Set();
	#pending = 0;
	#blocking_pending = 0;
	#deferred = null;
	#dirty_effects = [];
	#maybe_dirty_effects = [];
	skipped_effects = /* @__PURE__ */ new Set();
	is_fork = false;
	process(root_effects) {
		queued_root_effects = [];
		previous_batch = null;
		this.apply();
		var target = {
			parent: null,
			effect: null,
			effects: [],
			render_effects: [],
			block_effects: []
		};
		for (const root of root_effects) this.#traverse_effect_tree(root, target);
		if (!this.is_fork) this.#resolve();
		if (this.#blocking_pending > 0 || this.is_fork) {
			this.#defer_effects(target.effects);
			this.#defer_effects(target.render_effects);
			this.#defer_effects(target.block_effects);
		} else {
			previous_batch = this;
			current_batch = null;
			flush_queued_effects(target.render_effects);
			flush_queued_effects(target.effects);
			previous_batch = null;
		}
		batch_values = null;
	}
	#traverse_effect_tree(root, target) {
		root.f ^= CLEAN;
		var effect$1 = root.first;
		while (effect$1 !== null) {
			var flags$1 = effect$1.f;
			var is_branch = (flags$1 & 96) !== 0;
			var skip = is_branch && (flags$1 & 1024) !== 0 || (flags$1 & 8192) !== 0 || this.skipped_effects.has(effect$1);
			if ((effect$1.f & 128) !== 0 && effect$1.b?.is_pending()) target = {
				parent: target,
				effect: effect$1,
				effects: [],
				render_effects: [],
				block_effects: []
			};
			if (!skip && effect$1.fn !== null) {
				if (is_branch) effect$1.f ^= CLEAN;
				else if ((flags$1 & 4) !== 0) target.effects.push(effect$1);
				else if (is_dirty(effect$1)) {
					if ((effect$1.f & 16) !== 0) target.block_effects.push(effect$1);
					update_effect(effect$1);
				}
				var child$1 = effect$1.first;
				if (child$1 !== null) {
					effect$1 = child$1;
					continue;
				}
			}
			var parent = effect$1.parent;
			effect$1 = effect$1.next;
			while (effect$1 === null && parent !== null) {
				if (parent === target.effect) {
					this.#defer_effects(target.effects);
					this.#defer_effects(target.render_effects);
					this.#defer_effects(target.block_effects);
					target = target.parent;
				}
				effect$1 = parent.next;
				parent = parent.parent;
			}
		}
	}
	#defer_effects(effects) {
		for (const e of effects) {
			((e.f & 2048) !== 0 ? this.#dirty_effects : this.#maybe_dirty_effects).push(e);
			set_signal_status(e, CLEAN);
		}
	}
	capture(source$1, value) {
		if (!this.previous.has(source$1)) this.previous.set(source$1, value);
		this.current.set(source$1, source$1.v);
		batch_values?.set(source$1, source$1.v);
	}
	activate() {
		current_batch = this;
	}
	deactivate() {
		current_batch = null;
		batch_values = null;
	}
	flush() {
		this.activate();
		if (queued_root_effects.length > 0) {
			flush_effects();
			if (current_batch !== null && current_batch !== this) return;
		} else if (this.#pending === 0) this.process([]);
		this.deactivate();
		for (const update$1 of effect_pending_updates) {
			effect_pending_updates.delete(update$1);
			update$1();
			if (current_batch !== null) break;
		}
	}
	discard() {
		for (const fn of this.#discard_callbacks) fn(this);
		this.#discard_callbacks.clear();
	}
	#resolve() {
		if (this.#blocking_pending === 0) {
			for (const fn of this.#commit_callbacks) fn();
			this.#commit_callbacks.clear();
		}
		if (this.#pending === 0) this.#commit();
	}
	#commit() {
		if (batches.size > 1) {
			this.previous.clear();
			var previous_batch_values = batch_values;
			var is_earlier = true;
			var dummy_target = {
				parent: null,
				effect: null,
				effects: [],
				render_effects: [],
				block_effects: []
			};
			for (const batch of batches) {
				if (batch === this) {
					is_earlier = false;
					continue;
				}
				const sources = [];
				for (const [source$1, value] of this.current) {
					if (batch.current.has(source$1)) if (is_earlier && value !== batch.current.get(source$1)) batch.current.set(source$1, value);
					else continue;
					sources.push(source$1);
				}
				if (sources.length === 0) continue;
				const others = [...batch.current.keys()].filter((s) => !this.current.has(s));
				if (others.length > 0) {
					const marked = /* @__PURE__ */ new Set();
					const checked = /* @__PURE__ */ new Map();
					for (const source$1 of sources) mark_effects(source$1, others, marked, checked);
					if (queued_root_effects.length > 0) {
						current_batch = batch;
						batch.apply();
						for (const root of queued_root_effects) batch.#traverse_effect_tree(root, dummy_target);
						queued_root_effects = [];
						batch.deactivate();
					}
				}
			}
			current_batch = null;
			batch_values = previous_batch_values;
		}
		this.committed = true;
		batches.delete(this);
		this.#deferred?.resolve();
	}
	increment(blocking) {
		this.#pending += 1;
		if (blocking) this.#blocking_pending += 1;
	}
	decrement(blocking) {
		this.#pending -= 1;
		if (blocking) this.#blocking_pending -= 1;
		this.revive();
	}
	revive() {
		for (const e of this.#dirty_effects) {
			set_signal_status(e, DIRTY);
			schedule_effect(e);
		}
		for (const e of this.#maybe_dirty_effects) {
			set_signal_status(e, MAYBE_DIRTY);
			schedule_effect(e);
		}
		this.#dirty_effects = [];
		this.#maybe_dirty_effects = [];
		this.flush();
	}
	oncommit(fn) {
		this.#commit_callbacks.add(fn);
	}
	ondiscard(fn) {
		this.#discard_callbacks.add(fn);
	}
	settled() {
		return (this.#deferred ??= deferred()).promise;
	}
	static ensure() {
		if (current_batch === null) {
			const batch = current_batch = new Batch();
			batches.add(current_batch);
			if (!is_flushing_sync) Batch.enqueue(() => {
				if (current_batch !== batch) return;
				batch.flush();
			});
		}
		return current_batch;
	}
	static enqueue(task) {
		queue_micro_task(task);
	}
	apply() {}
};
function flushSync(fn) {
	var was_flushing_sync = is_flushing_sync;
	is_flushing_sync = true;
	try {
		var result;
		if (fn) {
			if (current_batch !== null) flush_effects();
			result = fn();
		}
		while (true) {
			flush_tasks();
			if (queued_root_effects.length === 0) {
				current_batch?.flush();
				if (queued_root_effects.length === 0) {
					last_scheduled_effect = null;
					return result;
				}
			}
			flush_effects();
		}
	} finally {
		is_flushing_sync = was_flushing_sync;
	}
}
function flush_effects() {
	var was_updating_effect = is_updating_effect;
	is_flushing = true;
	try {
		var flush_count = 0;
		set_is_updating_effect(true);
		while (queued_root_effects.length > 0) {
			var batch = Batch.ensure();
			if (flush_count++ > 1e3) infinite_loop_guard();
			batch.process(queued_root_effects);
			old_values.clear();
		}
	} finally {
		is_flushing = false;
		set_is_updating_effect(was_updating_effect);
		last_scheduled_effect = null;
	}
}
function infinite_loop_guard() {
	try {
		effect_update_depth_exceeded();
	} catch (error) {
		invoke_error_boundary(error, last_scheduled_effect);
	}
}
let eager_block_effects = null;
function flush_queued_effects(effects) {
	var length = effects.length;
	if (length === 0) return;
	var i = 0;
	while (i < length) {
		var effect$1 = effects[i++];
		if ((effect$1.f & 24576) === 0 && is_dirty(effect$1)) {
			eager_block_effects = /* @__PURE__ */ new Set();
			update_effect(effect$1);
			if (effect$1.deps === null && effect$1.first === null && effect$1.nodes_start === null) if (effect$1.teardown === null && effect$1.ac === null) unlink_effect(effect$1);
			else effect$1.fn = null;
			if (eager_block_effects?.size > 0) {
				old_values.clear();
				for (const e of eager_block_effects) {
					if ((e.f & 24576) !== 0) continue;
					const ordered_effects = [e];
					let ancestor = e.parent;
					while (ancestor !== null) {
						if (eager_block_effects.has(ancestor)) {
							eager_block_effects.delete(ancestor);
							ordered_effects.push(ancestor);
						}
						ancestor = ancestor.parent;
					}
					for (let j = ordered_effects.length - 1; j >= 0; j--) {
						const e$1 = ordered_effects[j];
						if ((e$1.f & 24576) !== 0) continue;
						update_effect(e$1);
					}
				}
				eager_block_effects.clear();
			}
		}
	}
	eager_block_effects = null;
}
function mark_effects(value, sources, marked, checked) {
	if (marked.has(value)) return;
	marked.add(value);
	if (value.reactions !== null) for (const reaction of value.reactions) {
		const flags$1 = reaction.f;
		if ((flags$1 & 2) !== 0) mark_effects(reaction, sources, marked, checked);
		else if ((flags$1 & 4194320) !== 0 && (flags$1 & 2048) === 0 && depends_on(reaction, sources, checked)) {
			set_signal_status(reaction, DIRTY);
			schedule_effect(reaction);
		}
	}
}
function mark_eager_effects(value, effects) {
	if (value.reactions === null) return;
	for (const reaction of value.reactions) {
		const flags$1 = reaction.f;
		if ((flags$1 & 2) !== 0) mark_eager_effects(reaction, effects);
		else if ((flags$1 & 131072) !== 0) {
			set_signal_status(reaction, DIRTY);
			effects.add(reaction);
		}
	}
}
function depends_on(reaction, sources, checked) {
	const depends = checked.get(reaction);
	if (depends !== void 0) return depends;
	if (reaction.deps !== null) for (const dep of reaction.deps) {
		if (sources.includes(dep)) return true;
		if ((dep.f & 2) !== 0 && depends_on(dep, sources, checked)) {
			checked.set(dep, true);
			return true;
		}
	}
	checked.set(reaction, false);
	return false;
}
function schedule_effect(signal) {
	var effect$1 = last_scheduled_effect = signal;
	while (effect$1.parent !== null) {
		effect$1 = effect$1.parent;
		var flags$1 = effect$1.f;
		if (is_flushing && effect$1 === active_effect && (flags$1 & 16) !== 0) return;
		if ((flags$1 & 96) !== 0) {
			if ((flags$1 & 1024) === 0) return;
			effect$1.f ^= CLEAN;
		}
	}
	queued_root_effects.push(effect$1);
}
function fork(fn) {
	experimental_async_fork();
	if (current_batch !== null) fork_timing();
	var batch = Batch.ensure();
	batch.is_fork = true;
	var committed = false;
	var settled$1 = batch.settled();
	flushSync(fn);
	for (var [source$1, value] of batch.previous) source$1.v = value;
	return {
		commit: async () => {
			if (committed) {
				await settled$1;
				return;
			}
			if (!batches.has(batch)) fork_discarded();
			committed = true;
			batch.is_fork = false;
			for (var [source$2, value$1] of batch.current) source$2.v = value$1;
			flushSync(() => {
				var eager_effects$1 = /* @__PURE__ */ new Set();
				for (var source$3 of batch.current.keys()) mark_eager_effects(source$3, eager_effects$1);
				set_eager_effects(eager_effects$1);
				flush_eager_effects();
			});
			batch.revive();
			await settled$1;
		},
		discard: () => {
			if (!committed && batches.has(batch)) {
				batches.delete(batch);
				batch.discard();
			}
		}
	};
}
function createSubscriber(start) {
	let subscribers = 0;
	let version = source(0);
	let stop;
	return () => {
		if (effect_tracking()) {
			get$1(version);
			render_effect(() => {
				if (subscribers === 0) stop = untrack(() => start(() => increment(version)));
				subscribers += 1;
				return () => {
					queue_micro_task(() => {
						subscribers -= 1;
						if (subscribers === 0) {
							stop?.();
							stop = void 0;
							increment(version);
						}
					});
				};
			});
		}
	};
}
var flags = EFFECT_PRESERVED | 65664;
function boundary(node, props, children) {
	new Boundary(node, props, children);
}
var Boundary = class {
	parent;
	#pending = false;
	#anchor;
	#hydrate_open = hydrating ? hydrate_node : null;
	#props;
	#children;
	#effect;
	#main_effect = null;
	#pending_effect = null;
	#failed_effect = null;
	#offscreen_fragment = null;
	#pending_anchor = null;
	#local_pending_count = 0;
	#pending_count = 0;
	#is_creating_fallback = false;
	#effect_pending = null;
	#effect_pending_update = () => {
		if (this.#effect_pending) internal_set(this.#effect_pending, this.#local_pending_count);
	};
	#effect_pending_subscriber = createSubscriber(() => {
		this.#effect_pending = source(this.#local_pending_count);
		return () => {
			this.#effect_pending = null;
		};
	});
	constructor(node, props, children) {
		this.#anchor = node;
		this.#props = props;
		this.#children = children;
		this.parent = active_effect.b;
		this.#pending = !!this.#props.pending;
		this.#effect = block(() => {
			active_effect.b = this;
			if (hydrating) {
				const comment$1 = this.#hydrate_open;
				hydrate_next();
				if (comment$1.nodeType === 8 && comment$1.data === "[!") this.#hydrate_pending_content();
				else this.#hydrate_resolved_content();
			} else {
				var anchor = this.#get_anchor();
				try {
					this.#main_effect = branch(() => children(anchor));
				} catch (error) {
					this.error(error);
				}
				if (this.#pending_count > 0) this.#show_pending_snippet();
				else this.#pending = false;
			}
			return () => {
				this.#pending_anchor?.remove();
			};
		}, flags);
		if (hydrating) this.#anchor = hydrate_node;
	}
	#hydrate_resolved_content() {
		try {
			this.#main_effect = branch(() => this.#children(this.#anchor));
		} catch (error) {
			this.error(error);
		}
		this.#pending = false;
	}
	#hydrate_pending_content() {
		const pending = this.#props.pending;
		if (!pending) return;
		this.#pending_effect = branch(() => pending(this.#anchor));
		Batch.enqueue(() => {
			var anchor = this.#get_anchor();
			this.#main_effect = this.#run(() => {
				Batch.ensure();
				return branch(() => this.#children(anchor));
			});
			if (this.#pending_count > 0) this.#show_pending_snippet();
			else {
				pause_effect(this.#pending_effect, () => {
					this.#pending_effect = null;
				});
				this.#pending = false;
			}
		});
	}
	#get_anchor() {
		var anchor = this.#anchor;
		if (this.#pending) {
			this.#pending_anchor = create_text();
			this.#anchor.before(this.#pending_anchor);
			anchor = this.#pending_anchor;
		}
		return anchor;
	}
	is_pending() {
		return this.#pending || !!this.parent && this.parent.is_pending();
	}
	has_pending_snippet() {
		return !!this.#props.pending;
	}
	#run(fn) {
		var previous_effect = active_effect;
		var previous_reaction = active_reaction;
		var previous_ctx = component_context;
		set_active_effect(this.#effect);
		set_active_reaction(this.#effect);
		set_component_context(this.#effect.ctx);
		try {
			return fn();
		} catch (e) {
			handle_error(e);
			return null;
		} finally {
			set_active_effect(previous_effect);
			set_active_reaction(previous_reaction);
			set_component_context(previous_ctx);
		}
	}
	#show_pending_snippet() {
		const pending = this.#props.pending;
		if (this.#main_effect !== null) {
			this.#offscreen_fragment = document.createDocumentFragment();
			this.#offscreen_fragment.append(this.#pending_anchor);
			move_effect(this.#main_effect, this.#offscreen_fragment);
		}
		if (this.#pending_effect === null) this.#pending_effect = branch(() => pending(this.#anchor));
	}
	#update_pending_count(d) {
		if (!this.has_pending_snippet()) {
			if (this.parent) this.parent.#update_pending_count(d);
			return;
		}
		this.#pending_count += d;
		if (this.#pending_count === 0) {
			this.#pending = false;
			if (this.#pending_effect) pause_effect(this.#pending_effect, () => {
				this.#pending_effect = null;
			});
			if (this.#offscreen_fragment) {
				this.#anchor.before(this.#offscreen_fragment);
				this.#offscreen_fragment = null;
			}
		}
	}
	update_pending_count(d) {
		this.#update_pending_count(d);
		this.#local_pending_count += d;
		effect_pending_updates.add(this.#effect_pending_update);
	}
	get_effect_pending() {
		this.#effect_pending_subscriber();
		return get$1(this.#effect_pending);
	}
	error(error) {
		var onerror = this.#props.onerror;
		let failed = this.#props.failed;
		if (this.#is_creating_fallback || !onerror && !failed) throw error;
		if (this.#main_effect) {
			destroy_effect(this.#main_effect);
			this.#main_effect = null;
		}
		if (this.#pending_effect) {
			destroy_effect(this.#pending_effect);
			this.#pending_effect = null;
		}
		if (this.#failed_effect) {
			destroy_effect(this.#failed_effect);
			this.#failed_effect = null;
		}
		if (hydrating) {
			set_hydrate_node(this.#hydrate_open);
			next();
			set_hydrate_node(skip_nodes());
		}
		var did_reset = false;
		var calling_on_error = false;
		const reset$1 = () => {
			if (did_reset) {
				svelte_boundary_reset_noop();
				return;
			}
			did_reset = true;
			if (calling_on_error) svelte_boundary_reset_onerror();
			Batch.ensure();
			this.#local_pending_count = 0;
			if (this.#failed_effect !== null) pause_effect(this.#failed_effect, () => {
				this.#failed_effect = null;
			});
			this.#pending = this.has_pending_snippet();
			this.#main_effect = this.#run(() => {
				this.#is_creating_fallback = false;
				return branch(() => this.#children(this.#anchor));
			});
			if (this.#pending_count > 0) this.#show_pending_snippet();
			else this.#pending = false;
		};
		var previous_reaction = active_reaction;
		try {
			set_active_reaction(null);
			calling_on_error = true;
			onerror?.(error, reset$1);
			calling_on_error = false;
		} catch (error$1) {
			invoke_error_boundary(error$1, this.#effect && this.#effect.parent);
		} finally {
			set_active_reaction(previous_reaction);
		}
		if (failed) queue_micro_task(() => {
			this.#failed_effect = this.#run(() => {
				Batch.ensure();
				this.#is_creating_fallback = true;
				try {
					return branch(() => {
						failed(this.#anchor, () => error, () => reset$1);
					});
				} catch (error$1) {
					invoke_error_boundary(error$1, this.#effect.parent);
					return null;
				} finally {
					this.#is_creating_fallback = false;
				}
			});
		});
	}
};
function flatten(sync, async, fn) {
	const d = is_runes() ? derived : derived_safe_equal;
	if (async.length === 0) {
		fn(sync.map(d));
		return;
	}
	var batch = current_batch;
	var parent = active_effect;
	var restore = capture();
	var was_hydrating = hydrating;
	Promise.all(async.map((expression) => /* @__PURE__ */ async_derived(expression))).then((result) => {
		restore();
		try {
			fn([...sync.map(d), ...result]);
		} catch (error) {
			if ((parent.f & 16384) === 0) invoke_error_boundary(error, parent);
		}
		if (was_hydrating) set_hydrating(false);
		batch?.deactivate();
		unset_context();
	}).catch((error) => {
		invoke_error_boundary(error, parent);
	});
}
function capture() {
	var previous_effect = active_effect;
	var previous_reaction = active_reaction;
	var previous_component_context = component_context;
	var previous_batch$1 = current_batch;
	var was_hydrating = hydrating;
	if (was_hydrating) var previous_hydrate_node = hydrate_node;
	return function restore() {
		set_active_effect(previous_effect);
		set_active_reaction(previous_reaction);
		set_component_context(previous_component_context);
		previous_batch$1?.activate();
		if (was_hydrating) {
			set_hydrating(true);
			set_hydrate_node(previous_hydrate_node);
		}
	};
}
function unset_context() {
	set_active_effect(null);
	set_active_reaction(null);
	set_component_context(null);
}
/* @__NO_SIDE_EFFECTS__ */
function derived(fn) {
	var flags$1 = 2 | DIRTY;
	var parent_derived = active_reaction !== null && (active_reaction.f & 2) !== 0 ? active_reaction : null;
	if (active_effect === null || parent_derived !== null && (parent_derived.f & 256) !== 0) flags$1 |= 256;
	else active_effect.f |= EFFECT_PRESERVED;
	return {
		ctx: component_context,
		deps: null,
		effects: null,
		equals,
		f: flags$1,
		fn,
		reactions: null,
		rv: 0,
		v: UNINITIALIZED,
		wv: 0,
		parent: parent_derived ?? active_effect,
		ac: null
	};
}
/* @__NO_SIDE_EFFECTS__ */
function async_derived(fn, location) {
	let parent = active_effect;
	if (parent === null) async_derived_orphan();
	var boundary$1 = parent.b;
	var promise = void 0;
	var signal = source(UNINITIALIZED);
	var should_suspend = !active_reaction;
	var deferreds = /* @__PURE__ */ new Map();
	async_effect(() => {
		var d = deferred();
		promise = d.promise;
		try {
			Promise.resolve(fn()).then(d.resolve, d.reject).then(() => {
				if (batch === current_batch && batch.committed) batch.deactivate();
				unset_context();
			});
		} catch (error) {
			d.reject(error);
			unset_context();
		}
		var batch = current_batch;
		if (should_suspend) {
			var blocking = !boundary$1.is_pending();
			boundary$1.update_pending_count(1);
			batch.increment(blocking);
			deferreds.get(batch)?.reject(STALE_REACTION);
			deferreds.delete(batch);
			deferreds.set(batch, d);
		}
		const handler = (value, error = void 0) => {
			batch.activate();
			if (error) {
				if (error !== STALE_REACTION) {
					signal.f |= ERROR_VALUE;
					internal_set(signal, error);
				}
			} else {
				if ((signal.f & 8388608) !== 0) signal.f ^= ERROR_VALUE;
				internal_set(signal, value);
				for (const [b, d$1] of deferreds) {
					deferreds.delete(b);
					if (b === batch) break;
					d$1.reject(STALE_REACTION);
				}
			}
			if (should_suspend) {
				boundary$1.update_pending_count(-1);
				batch.decrement(blocking);
			}
		};
		d.promise.then(handler, (e) => handler(null, e || "unknown"));
	});
	teardown(() => {
		for (const d of deferreds.values()) d.reject(STALE_REACTION);
	});
	return new Promise((fulfil) => {
		function next$1(p) {
			function go() {
				if (p === promise) fulfil(signal);
				else next$1(promise);
			}
			p.then(go, go);
		}
		next$1(promise);
	});
}
/* @__NO_SIDE_EFFECTS__ */
function user_derived(fn) {
	const d = /* @__PURE__ */ derived(fn);
	push_reaction_value(d);
	return d;
}
/* @__NO_SIDE_EFFECTS__ */
function derived_safe_equal(fn) {
	const signal = /* @__PURE__ */ derived(fn);
	signal.equals = safe_equals;
	return signal;
}
function destroy_derived_effects(derived$1) {
	var effects = derived$1.effects;
	if (effects !== null) {
		derived$1.effects = null;
		for (var i = 0; i < effects.length; i += 1) destroy_effect(effects[i]);
	}
}
function get_derived_parent_effect(derived$1) {
	var parent = derived$1.parent;
	while (parent !== null) {
		if ((parent.f & 2) === 0) return parent;
		parent = parent.parent;
	}
	return null;
}
function execute_derived(derived$1) {
	var value;
	var prev_active_effect = active_effect;
	set_active_effect(get_derived_parent_effect(derived$1));
	try {
		derived$1.f &= ~WAS_MARKED;
		destroy_derived_effects(derived$1);
		value = update_reaction(derived$1);
	} finally {
		set_active_effect(prev_active_effect);
	}
	return value;
}
function update_derived(derived$1) {
	var value = execute_derived(derived$1);
	if (!derived$1.equals(value)) {
		derived$1.v = value;
		derived$1.wv = increment_write_version();
	}
	if (is_destroying_effect) return;
	if (batch_values !== null) batch_values.set(derived$1, derived$1.v);
	else set_signal_status(derived$1, (skip_reaction || (derived$1.f & 256) !== 0) && derived$1.deps !== null ? MAYBE_DIRTY : CLEAN);
}
let eager_effects = /* @__PURE__ */ new Set();
const old_values = /* @__PURE__ */ new Map();
function set_eager_effects(v) {
	eager_effects = v;
}
var eager_effects_deferred = false;
function source(v, stack$1) {
	return {
		f: 0,
		v,
		reactions: null,
		equals,
		rv: 0,
		wv: 0
	};
}
/* @__NO_SIDE_EFFECTS__ */
function state(v, stack$1) {
	const s = source(v, stack$1);
	push_reaction_value(s);
	return s;
}
/* @__NO_SIDE_EFFECTS__ */
function mutable_source(initial_value, immutable = false, trackable = true) {
	const s = source(initial_value);
	if (!immutable) s.equals = safe_equals;
	if (legacy_mode_flag && trackable && component_context !== null && component_context.l !== null) (component_context.l.s ??= []).push(s);
	return s;
}
function mutate(source$1, value) {
	set(source$1, untrack(() => get$1(source$1)));
	return value;
}
function set(source$1, value, should_proxy = false) {
	if (active_reaction !== null && (!untracking || (active_reaction.f & 131072) !== 0) && is_runes() && (active_reaction.f & 4325394) !== 0 && !current_sources?.includes(source$1)) state_unsafe_mutation();
	return internal_set(source$1, should_proxy ? proxy(value) : value);
}
function internal_set(source$1, value) {
	if (!source$1.equals(value)) {
		var old_value = source$1.v;
		if (is_destroying_effect) old_values.set(source$1, value);
		else old_values.set(source$1, old_value);
		source$1.v = value;
		var batch = Batch.ensure();
		batch.capture(source$1, old_value);
		if ((source$1.f & 2) !== 0) {
			if ((source$1.f & 2048) !== 0) execute_derived(source$1);
			set_signal_status(source$1, (source$1.f & 256) === 0 ? CLEAN : MAYBE_DIRTY);
		}
		source$1.wv = increment_write_version();
		mark_reactions(source$1, DIRTY);
		if (is_runes() && active_effect !== null && (active_effect.f & 1024) !== 0 && (active_effect.f & 96) === 0) if (untracked_writes === null) set_untracked_writes([source$1]);
		else untracked_writes.push(source$1);
		if (!batch.is_fork && eager_effects.size > 0 && !eager_effects_deferred) flush_eager_effects();
	}
	return value;
}
function flush_eager_effects() {
	eager_effects_deferred = false;
	const inspects = Array.from(eager_effects);
	for (const effect$1 of inspects) {
		if ((effect$1.f & 1024) !== 0) set_signal_status(effect$1, MAYBE_DIRTY);
		if (is_dirty(effect$1)) update_effect(effect$1);
	}
	eager_effects.clear();
}
function update(source$1, d = 1) {
	var value = get$1(source$1);
	var result = d === 1 ? value++ : value--;
	set(source$1, value);
	return result;
}
function increment(source$1) {
	set(source$1, source$1.v + 1);
}
function mark_reactions(signal, status) {
	var reactions = signal.reactions;
	if (reactions === null) return;
	var runes = is_runes();
	var length = reactions.length;
	for (var i = 0; i < length; i++) {
		var reaction = reactions[i];
		var flags$1 = reaction.f;
		if (!runes && reaction === active_effect) continue;
		var not_dirty = (flags$1 & DIRTY) === 0;
		if (not_dirty) set_signal_status(reaction, status);
		if ((flags$1 & 2) !== 0) {
			if ((flags$1 & 32768) === 0) {
				reaction.f |= WAS_MARKED;
				mark_reactions(reaction, MAYBE_DIRTY);
			}
		} else if (not_dirty) {
			if ((flags$1 & 16) !== 0) {
				if (eager_block_effects !== null) eager_block_effects.add(reaction);
			}
			schedule_effect(reaction);
		}
	}
}
function proxy(value) {
	if (typeof value !== "object" || value === null || STATE_SYMBOL in value) return value;
	const prototype = get_prototype_of(value);
	if (prototype !== object_prototype && prototype !== array_prototype) return value;
	var sources = /* @__PURE__ */ new Map();
	var is_proxied_array = is_array(value);
	var version = /* @__PURE__ */ state(0);
	var stack$1 = null;
	var parent_version = update_version;
	var with_parent = (fn) => {
		if (update_version === parent_version) return fn();
		var reaction = active_reaction;
		var version$1 = update_version;
		set_active_reaction(null);
		set_update_version(parent_version);
		var result = fn();
		set_active_reaction(reaction);
		set_update_version(version$1);
		return result;
	};
	if (is_proxied_array) sources.set("length", /* @__PURE__ */ state(value.length, stack$1));
	return new Proxy(value, {
		defineProperty(_, prop$1, descriptor) {
			if (!("value" in descriptor) || descriptor.configurable === false || descriptor.enumerable === false || descriptor.writable === false) state_descriptors_fixed();
			var s = sources.get(prop$1);
			if (s === void 0) s = with_parent(() => {
				var s$1 = /* @__PURE__ */ state(descriptor.value, stack$1);
				sources.set(prop$1, s$1);
				return s$1;
			});
			else set(s, descriptor.value, true);
			return true;
		},
		deleteProperty(target, prop$1) {
			var s = sources.get(prop$1);
			if (s === void 0) {
				if (prop$1 in target) {
					const s$1 = with_parent(() => /* @__PURE__ */ state(UNINITIALIZED, stack$1));
					sources.set(prop$1, s$1);
					increment(version);
				}
			} else {
				set(s, UNINITIALIZED);
				increment(version);
			}
			return true;
		},
		get(target, prop$1, receiver) {
			if (prop$1 === STATE_SYMBOL) return value;
			var s = sources.get(prop$1);
			var exists = prop$1 in target;
			if (s === void 0 && (!exists || get_descriptor(target, prop$1)?.writable)) {
				s = with_parent(() => {
					return /* @__PURE__ */ state(proxy(exists ? target[prop$1] : UNINITIALIZED), stack$1);
				});
				sources.set(prop$1, s);
			}
			if (s !== void 0) {
				var v = get$1(s);
				return v === UNINITIALIZED ? void 0 : v;
			}
			return Reflect.get(target, prop$1, receiver);
		},
		getOwnPropertyDescriptor(target, prop$1) {
			var descriptor = Reflect.getOwnPropertyDescriptor(target, prop$1);
			if (descriptor && "value" in descriptor) {
				var s = sources.get(prop$1);
				if (s) descriptor.value = get$1(s);
			} else if (descriptor === void 0) {
				var source$1 = sources.get(prop$1);
				var value$1 = source$1?.v;
				if (source$1 !== void 0 && value$1 !== UNINITIALIZED) return {
					enumerable: true,
					configurable: true,
					value: value$1,
					writable: true
				};
			}
			return descriptor;
		},
		has(target, prop$1) {
			if (prop$1 === STATE_SYMBOL) return true;
			var s = sources.get(prop$1);
			var has = s !== void 0 && s.v !== UNINITIALIZED || Reflect.has(target, prop$1);
			if (s !== void 0 || active_effect !== null && (!has || get_descriptor(target, prop$1)?.writable)) {
				if (s === void 0) {
					s = with_parent(() => {
						return /* @__PURE__ */ state(has ? proxy(target[prop$1]) : UNINITIALIZED, stack$1);
					});
					sources.set(prop$1, s);
				}
				if (get$1(s) === UNINITIALIZED) return false;
			}
			return has;
		},
		set(target, prop$1, value$1, receiver) {
			var s = sources.get(prop$1);
			var has = prop$1 in target;
			if (is_proxied_array && prop$1 === "length") for (var i = value$1; i < s.v; i += 1) {
				var other_s = sources.get(i + "");
				if (other_s !== void 0) set(other_s, UNINITIALIZED);
				else if (i in target) {
					other_s = with_parent(() => /* @__PURE__ */ state(UNINITIALIZED, stack$1));
					sources.set(i + "", other_s);
				}
			}
			if (s === void 0) {
				if (!has || get_descriptor(target, prop$1)?.writable) {
					s = with_parent(() => /* @__PURE__ */ state(void 0, stack$1));
					set(s, proxy(value$1));
					sources.set(prop$1, s);
				}
			} else {
				has = s.v !== UNINITIALIZED;
				var p = with_parent(() => proxy(value$1));
				set(s, p);
			}
			var descriptor = Reflect.getOwnPropertyDescriptor(target, prop$1);
			if (descriptor?.set) descriptor.set.call(receiver, value$1);
			if (!has) {
				if (is_proxied_array && typeof prop$1 === "string") {
					var ls = sources.get("length");
					var n = Number(prop$1);
					if (Number.isInteger(n) && n >= ls.v) set(ls, n + 1);
				}
				increment(version);
			}
			return true;
		},
		ownKeys(target) {
			get$1(version);
			var own_keys = Reflect.ownKeys(target).filter((key$1) => {
				var source$2 = sources.get(key$1);
				return source$2 === void 0 || source$2.v !== UNINITIALIZED;
			});
			for (var [key, source$1] of sources) if (source$1.v !== UNINITIALIZED && !(key in target)) own_keys.push(key);
			return own_keys;
		},
		setPrototypeOf() {
			state_prototype_fixed();
		}
	});
}
function get_proxied_value(value) {
	try {
		if (value !== null && typeof value === "object" && STATE_SYMBOL in value) return value[STATE_SYMBOL];
	} catch {}
	return value;
}
function is(a, b) {
	return Object.is(get_proxied_value(a), get_proxied_value(b));
}
var $window;
var $document;
var is_firefox;
var first_child_getter;
var next_sibling_getter;
function init_operations() {
	if ($window !== void 0) return;
	$window = window;
	$document = document;
	is_firefox = /Firefox/.test(navigator.userAgent);
	var element_prototype = Element.prototype;
	var node_prototype = Node.prototype;
	var text_prototype = Text.prototype;
	first_child_getter = get_descriptor(node_prototype, "firstChild").get;
	next_sibling_getter = get_descriptor(node_prototype, "nextSibling").get;
	if (is_extensible(element_prototype)) {
		element_prototype.__click = void 0;
		element_prototype.__className = void 0;
		element_prototype.__attributes = null;
		element_prototype.__style = void 0;
		element_prototype.__e = void 0;
	}
	if (is_extensible(text_prototype)) text_prototype.__t = void 0;
}
function create_text(value = "") {
	return document.createTextNode(value);
}
/* @__NO_SIDE_EFFECTS__ */
function get_first_child(node) {
	return first_child_getter.call(node);
}
/* @__NO_SIDE_EFFECTS__ */
function get_next_sibling(node) {
	return next_sibling_getter.call(node);
}
function child(node, is_text) {
	if (!hydrating) return /* @__PURE__ */ get_first_child(node);
	var child$1 = /* @__PURE__ */ get_first_child(hydrate_node);
	if (child$1 === null) child$1 = hydrate_node.appendChild(create_text());
	else if (is_text && child$1.nodeType !== 3) {
		var text$1 = create_text();
		child$1?.before(text$1);
		set_hydrate_node(text$1);
		return text$1;
	}
	set_hydrate_node(child$1);
	return child$1;
}
function first_child(fragment, is_text = false) {
	if (!hydrating) {
		var first = /* @__PURE__ */ get_first_child(fragment);
		if (first instanceof Comment && first.data === "") return /* @__PURE__ */ get_next_sibling(first);
		return first;
	}
	if (is_text && hydrate_node?.nodeType !== 3) {
		var text$1 = create_text();
		hydrate_node?.before(text$1);
		set_hydrate_node(text$1);
		return text$1;
	}
	return hydrate_node;
}
function sibling(node, count = 1, is_text = false) {
	let next_sibling = hydrating ? hydrate_node : node;
	var last_sibling;
	while (count--) {
		last_sibling = next_sibling;
		next_sibling = /* @__PURE__ */ get_next_sibling(next_sibling);
	}
	if (!hydrating) return next_sibling;
	if (is_text && next_sibling?.nodeType !== 3) {
		var text$1 = create_text();
		if (next_sibling === null) last_sibling?.after(text$1);
		else next_sibling.before(text$1);
		set_hydrate_node(text$1);
		return text$1;
	}
	set_hydrate_node(next_sibling);
	return next_sibling;
}
function clear_text_content(node) {
	node.textContent = "";
}
function should_defer_append() {
	return false;
}
function autofocus(dom, value) {
	if (value) {
		const body = document.body;
		dom.autofocus = true;
		queue_micro_task(() => {
			if (document.activeElement === body) dom.focus();
		});
	}
}
function remove_textarea_child(dom) {
	if (hydrating && /* @__PURE__ */ get_first_child(dom) !== null) clear_text_content(dom);
}
var listening_to_form_reset = false;
function add_form_reset_listener() {
	if (!listening_to_form_reset) {
		listening_to_form_reset = true;
		document.addEventListener("reset", (evt) => {
			Promise.resolve().then(() => {
				if (!evt.defaultPrevented) for (const e of evt.target.elements) e.__on_r?.();
			});
		}, { capture: true });
	}
}
function listen(target, events, handler, call_handler_immediately = true) {
	if (call_handler_immediately) handler();
	for (var name of events) target.addEventListener(name, handler);
	teardown(() => {
		for (var name$1 of events) target.removeEventListener(name$1, handler);
	});
}
function without_reactive_context(fn) {
	var previous_reaction = active_reaction;
	var previous_effect = active_effect;
	set_active_reaction(null);
	set_active_effect(null);
	try {
		return fn();
	} finally {
		set_active_reaction(previous_reaction);
		set_active_effect(previous_effect);
	}
}
function listen_to_event_and_reset_event(element, event$1, handler, on_reset = handler) {
	element.addEventListener(event$1, () => without_reactive_context(handler));
	const prev = element.__on_r;
	if (prev) element.__on_r = () => {
		prev();
		on_reset(true);
	};
	else element.__on_r = () => on_reset(true);
	add_form_reset_listener();
}
function validate_effect(rune) {
	if (active_effect === null && active_reaction === null) effect_orphan(rune);
	if (active_reaction !== null && (active_reaction.f & 256) !== 0 && active_effect === null) effect_in_unowned_derived();
	if (is_destroying_effect) effect_in_teardown(rune);
}
function push_effect(effect$1, parent_effect) {
	var parent_last = parent_effect.last;
	if (parent_last === null) parent_effect.last = parent_effect.first = effect$1;
	else {
		parent_last.next = effect$1;
		effect$1.prev = parent_last;
		parent_effect.last = effect$1;
	}
}
function create_effect(type, fn, sync, push$1 = true) {
	var parent = active_effect;
	if (parent !== null && (parent.f & 8192) !== 0) type |= INERT;
	var effect$1 = {
		ctx: component_context,
		deps: null,
		nodes_start: null,
		nodes_end: null,
		f: type | DIRTY,
		first: null,
		fn,
		last: null,
		next: null,
		parent,
		b: parent && parent.b,
		prev: null,
		teardown: null,
		transitions: null,
		wv: 0,
		ac: null
	};
	if (sync) try {
		update_effect(effect$1);
		effect$1.f |= EFFECT_RAN;
	} catch (e$1) {
		destroy_effect(effect$1);
		throw e$1;
	}
	else if (fn !== null) schedule_effect(effect$1);
	if (push$1) {
		var e = effect$1;
		if (sync && e.deps === null && e.teardown === null && e.nodes_start === null && e.first === e.last && (e.f & 524288) === 0) {
			e = e.first;
			if ((type & 16) !== 0 && (type & 65536) !== 0 && e !== null) e.f |= EFFECT_TRANSPARENT;
		}
		if (e !== null) {
			e.parent = parent;
			if (parent !== null) push_effect(e, parent);
			if (active_reaction !== null && (active_reaction.f & 2) !== 0 && (type & 64) === 0) {
				var derived$1 = active_reaction;
				(derived$1.effects ??= []).push(e);
			}
		}
	}
	return effect$1;
}
function effect_tracking() {
	return active_reaction !== null && !untracking;
}
function teardown(fn) {
	const effect$1 = create_effect(8, null, false);
	set_signal_status(effect$1, CLEAN);
	effect$1.teardown = fn;
	return effect$1;
}
function user_effect(fn) {
	validate_effect("$effect");
	var flags$1 = active_effect.f;
	if (!active_reaction && (flags$1 & 32) !== 0 && (flags$1 & 32768) === 0) {
		var context = component_context;
		(context.e ??= []).push(fn);
	} else return create_user_effect(fn);
}
function create_user_effect(fn) {
	return create_effect(4 | USER_EFFECT, fn, false);
}
function user_pre_effect(fn) {
	validate_effect("$effect.pre");
	return create_effect(8 | USER_EFFECT, fn, true);
}
function component_root(fn) {
	Batch.ensure();
	const effect$1 = create_effect(64 | EFFECT_PRESERVED, fn, true);
	return (options = {}) => {
		return new Promise((fulfil) => {
			if (options.outro) pause_effect(effect$1, () => {
				destroy_effect(effect$1);
				fulfil(void 0);
			});
			else {
				destroy_effect(effect$1);
				fulfil(void 0);
			}
		});
	};
}
function effect(fn) {
	return create_effect(4, fn, false);
}
function async_effect(fn) {
	return create_effect(ASYNC | EFFECT_PRESERVED, fn, true);
}
function render_effect(fn, flags$1 = 0) {
	return create_effect(8 | flags$1, fn, true);
}
function template_effect(fn, sync = [], async = []) {
	flatten(sync, async, (values) => {
		create_effect(8, () => fn(...values.map(get$1)), true);
	});
}
function block(fn, flags$1 = 0) {
	return create_effect(16 | flags$1, fn, true);
}
function branch(fn, push$1 = true) {
	return create_effect(32 | EFFECT_PRESERVED, fn, true, push$1);
}
function execute_effect_teardown(effect$1) {
	var teardown$1 = effect$1.teardown;
	if (teardown$1 !== null) {
		const previously_destroying_effect = is_destroying_effect;
		const previous_reaction = active_reaction;
		set_is_destroying_effect(true);
		set_active_reaction(null);
		try {
			teardown$1.call(null);
		} finally {
			set_is_destroying_effect(previously_destroying_effect);
			set_active_reaction(previous_reaction);
		}
	}
}
function destroy_effect_children(signal, remove_dom = false) {
	var effect$1 = signal.first;
	signal.first = signal.last = null;
	while (effect$1 !== null) {
		const controller = effect$1.ac;
		if (controller !== null) without_reactive_context(() => {
			controller.abort(STALE_REACTION);
		});
		var next$1 = effect$1.next;
		if ((effect$1.f & 64) !== 0) effect$1.parent = null;
		else destroy_effect(effect$1, remove_dom);
		effect$1 = next$1;
	}
}
function destroy_block_effect_children(signal) {
	var effect$1 = signal.first;
	while (effect$1 !== null) {
		var next$1 = effect$1.next;
		if ((effect$1.f & 32) === 0) destroy_effect(effect$1);
		effect$1 = next$1;
	}
}
function destroy_effect(effect$1, remove_dom = true) {
	var removed = false;
	if ((remove_dom || (effect$1.f & 262144) !== 0) && effect$1.nodes_start !== null && effect$1.nodes_end !== null) {
		remove_effect_dom(effect$1.nodes_start, effect$1.nodes_end);
		removed = true;
	}
	destroy_effect_children(effect$1, remove_dom && !removed);
	remove_reactions(effect$1, 0);
	set_signal_status(effect$1, DESTROYED);
	var transitions = effect$1.transitions;
	if (transitions !== null) for (const transition of transitions) transition.stop();
	execute_effect_teardown(effect$1);
	var parent = effect$1.parent;
	if (parent !== null && parent.first !== null) unlink_effect(effect$1);
	effect$1.next = effect$1.prev = effect$1.teardown = effect$1.ctx = effect$1.deps = effect$1.fn = effect$1.nodes_start = effect$1.nodes_end = effect$1.ac = null;
}
function remove_effect_dom(node, end) {
	while (node !== null) {
		var next$1 = node === end ? null : /* @__PURE__ */ get_next_sibling(node);
		node.remove();
		node = next$1;
	}
}
function unlink_effect(effect$1) {
	var parent = effect$1.parent;
	var prev = effect$1.prev;
	var next$1 = effect$1.next;
	if (prev !== null) prev.next = next$1;
	if (next$1 !== null) next$1.prev = prev;
	if (parent !== null) {
		if (parent.first === effect$1) parent.first = next$1;
		if (parent.last === effect$1) parent.last = prev;
	}
}
function pause_effect(effect$1, callback, destroy = true) {
	var transitions = [];
	pause_children(effect$1, transitions, true);
	run_out_transitions(transitions, () => {
		if (destroy) destroy_effect(effect$1);
		if (callback) callback();
	});
}
function run_out_transitions(transitions, fn) {
	var remaining = transitions.length;
	if (remaining > 0) {
		var check = () => --remaining || fn();
		for (var transition of transitions) transition.out(check);
	} else fn();
}
function pause_children(effect$1, transitions, local) {
	if ((effect$1.f & 8192) !== 0) return;
	effect$1.f ^= INERT;
	if (effect$1.transitions !== null) {
		for (const transition of effect$1.transitions) if (transition.is_global || local) transitions.push(transition);
	}
	var child$1 = effect$1.first;
	while (child$1 !== null) {
		var sibling$1 = child$1.next;
		var transparent = (child$1.f & 65536) !== 0 || (child$1.f & 32) !== 0 && (effect$1.f & 16) !== 0;
		pause_children(child$1, transitions, transparent ? local : false);
		child$1 = sibling$1;
	}
}
function resume_effect(effect$1) {
	resume_children(effect$1, true);
}
function resume_children(effect$1, local) {
	if ((effect$1.f & 8192) === 0) return;
	effect$1.f ^= INERT;
	if ((effect$1.f & 1024) === 0) {
		set_signal_status(effect$1, DIRTY);
		schedule_effect(effect$1);
	}
	var child$1 = effect$1.first;
	while (child$1 !== null) {
		var sibling$1 = child$1.next;
		var transparent = (child$1.f & 65536) !== 0 || (child$1.f & 32) !== 0;
		resume_children(child$1, transparent ? local : false);
		child$1 = sibling$1;
	}
	if (effect$1.transitions !== null) {
		for (const transition of effect$1.transitions) if (transition.is_global || local) transition.in();
	}
}
function move_effect(effect$1, fragment) {
	var node = effect$1.nodes_start;
	var end = effect$1.nodes_end;
	while (node !== null) {
		var next$1 = node === end ? null : /* @__PURE__ */ get_next_sibling(node);
		fragment.append(node);
		node = next$1;
	}
}
let captured_signals = null;
function capture_signals(fn) {
	var previous_captured_signals = captured_signals;
	try {
		captured_signals = /* @__PURE__ */ new Set();
		untrack(fn);
		if (previous_captured_signals !== null) for (var signal of captured_signals) previous_captured_signals.add(signal);
		return captured_signals;
	} finally {
		captured_signals = previous_captured_signals;
	}
}
function invalidate_inner_signals(fn) {
	for (var signal of capture_signals(fn)) internal_set(signal, signal.v);
}
let is_updating_effect = false;
function set_is_updating_effect(value) {
	is_updating_effect = value;
}
let is_destroying_effect = false;
function set_is_destroying_effect(value) {
	is_destroying_effect = value;
}
let active_reaction = null;
let untracking = false;
function set_active_reaction(reaction) {
	active_reaction = reaction;
}
let active_effect = null;
function set_active_effect(effect$1) {
	active_effect = effect$1;
}
let current_sources = null;
function push_reaction_value(value) {
	if (active_reaction !== null && true) if (current_sources === null) current_sources = [value];
	else current_sources.push(value);
}
var new_deps = null;
var skipped_deps = 0;
let untracked_writes = null;
function set_untracked_writes(value) {
	untracked_writes = value;
}
let write_version = 1;
var read_version = 0;
let update_version = read_version;
function set_update_version(value) {
	update_version = value;
}
let skip_reaction = false;
function increment_write_version() {
	return ++write_version;
}
function is_dirty(reaction) {
	var flags$1 = reaction.f;
	if ((flags$1 & 2048) !== 0) return true;
	if ((flags$1 & 4096) !== 0) {
		var dependencies = reaction.deps;
		var is_unowned = (flags$1 & 256) !== 0;
		if (flags$1 & 2) reaction.f &= ~WAS_MARKED;
		if (dependencies !== null) {
			var i;
			var dependency;
			var is_disconnected = (flags$1 & 512) !== 0;
			var is_unowned_connected = is_unowned && active_effect !== null && !skip_reaction;
			var length = dependencies.length;
			if ((is_disconnected || is_unowned_connected) && (active_effect === null || (active_effect.f & 16384) === 0)) {
				var derived$1 = reaction;
				var parent = derived$1.parent;
				for (i = 0; i < length; i++) {
					dependency = dependencies[i];
					if (is_disconnected || !dependency?.reactions?.includes(derived$1)) (dependency.reactions ??= []).push(derived$1);
				}
				if (is_disconnected) derived$1.f ^= 512;
				if (is_unowned_connected && parent !== null && (parent.f & 256) === 0) derived$1.f ^= 256;
			}
			for (i = 0; i < length; i++) {
				dependency = dependencies[i];
				if (is_dirty(dependency)) update_derived(dependency);
				if (dependency.wv > reaction.wv) return true;
			}
		}
		if (!is_unowned || active_effect !== null && !skip_reaction) set_signal_status(reaction, CLEAN);
	}
	return false;
}
function schedule_possible_effect_self_invalidation(signal, effect$1, root = true) {
	var reactions = signal.reactions;
	if (reactions === null) return;
	if (current_sources?.includes(signal)) return;
	for (var i = 0; i < reactions.length; i++) {
		var reaction = reactions[i];
		if ((reaction.f & 2) !== 0) schedule_possible_effect_self_invalidation(reaction, effect$1, false);
		else if (effect$1 === reaction) {
			if (root) set_signal_status(reaction, DIRTY);
			else if ((reaction.f & 1024) !== 0) set_signal_status(reaction, MAYBE_DIRTY);
			schedule_effect(reaction);
		}
	}
}
function update_reaction(reaction) {
	var previous_deps = new_deps;
	var previous_skipped_deps = skipped_deps;
	var previous_untracked_writes = untracked_writes;
	var previous_reaction = active_reaction;
	var previous_skip_reaction = skip_reaction;
	var previous_sources = current_sources;
	var previous_component_context = component_context;
	var previous_untracking = untracking;
	var previous_update_version = update_version;
	var flags$1 = reaction.f;
	new_deps = null;
	skipped_deps = 0;
	untracked_writes = null;
	skip_reaction = (flags$1 & 256) !== 0 && (untracking || !is_updating_effect || active_reaction === null);
	active_reaction = (flags$1 & 96) === 0 ? reaction : null;
	current_sources = null;
	set_component_context(reaction.ctx);
	untracking = false;
	update_version = ++read_version;
	if (reaction.ac !== null) {
		without_reactive_context(() => {
			reaction.ac.abort(STALE_REACTION);
		});
		reaction.ac = null;
	}
	try {
		reaction.f |= REACTION_IS_UPDATING;
		var fn = reaction.fn;
		var result = fn();
		var deps = reaction.deps;
		if (new_deps !== null) {
			var i;
			remove_reactions(reaction, skipped_deps);
			if (deps !== null && skipped_deps > 0) {
				deps.length = skipped_deps + new_deps.length;
				for (i = 0; i < new_deps.length; i++) deps[skipped_deps + i] = new_deps[i];
			} else reaction.deps = deps = new_deps;
			if (!skip_reaction || (flags$1 & 2) !== 0 && reaction.reactions !== null) for (i = skipped_deps; i < deps.length; i++) (deps[i].reactions ??= []).push(reaction);
		} else if (deps !== null && skipped_deps < deps.length) {
			remove_reactions(reaction, skipped_deps);
			deps.length = skipped_deps;
		}
		if (is_runes() && untracked_writes !== null && !untracking && deps !== null && (reaction.f & 6146) === 0) for (i = 0; i < untracked_writes.length; i++) schedule_possible_effect_self_invalidation(untracked_writes[i], reaction);
		if (previous_reaction !== null && previous_reaction !== reaction) {
			read_version++;
			if (untracked_writes !== null) if (previous_untracked_writes === null) previous_untracked_writes = untracked_writes;
			else previous_untracked_writes.push(...untracked_writes);
		}
		if ((reaction.f & 8388608) !== 0) reaction.f ^= ERROR_VALUE;
		return result;
	} catch (error) {
		return handle_error(error);
	} finally {
		reaction.f ^= REACTION_IS_UPDATING;
		new_deps = previous_deps;
		skipped_deps = previous_skipped_deps;
		untracked_writes = previous_untracked_writes;
		active_reaction = previous_reaction;
		skip_reaction = previous_skip_reaction;
		current_sources = previous_sources;
		set_component_context(previous_component_context);
		untracking = previous_untracking;
		update_version = previous_update_version;
	}
}
function remove_reaction(signal, dependency) {
	let reactions = dependency.reactions;
	if (reactions !== null) {
		var index$1 = index_of.call(reactions, signal);
		if (index$1 !== -1) {
			var new_length = reactions.length - 1;
			if (new_length === 0) reactions = dependency.reactions = null;
			else {
				reactions[index$1] = reactions[new_length];
				reactions.pop();
			}
		}
	}
	if (reactions === null && (dependency.f & 2) !== 0 && (new_deps === null || !new_deps.includes(dependency))) {
		set_signal_status(dependency, MAYBE_DIRTY);
		if ((dependency.f & 768) === 0) dependency.f ^= 512;
		destroy_derived_effects(dependency);
		remove_reactions(dependency, 0);
	}
}
function remove_reactions(signal, start_index) {
	var dependencies = signal.deps;
	if (dependencies === null) return;
	for (var i = start_index; i < dependencies.length; i++) remove_reaction(signal, dependencies[i]);
}
function update_effect(effect$1) {
	var flags$1 = effect$1.f;
	if ((flags$1 & 16384) !== 0) return;
	set_signal_status(effect$1, CLEAN);
	var previous_effect = active_effect;
	var was_updating_effect = is_updating_effect;
	active_effect = effect$1;
	is_updating_effect = true;
	try {
		if ((flags$1 & 16) !== 0) destroy_block_effect_children(effect$1);
		else destroy_effect_children(effect$1);
		execute_effect_teardown(effect$1);
		var teardown$1 = update_reaction(effect$1);
		effect$1.teardown = typeof teardown$1 === "function" ? teardown$1 : null;
		effect$1.wv = write_version;
	} finally {
		is_updating_effect = was_updating_effect;
		active_effect = previous_effect;
	}
}
async function tick() {
	await Promise.resolve();
	flushSync();
}
function settled() {
	return Batch.ensure().settled();
}
function get$1(signal) {
	var is_derived = (signal.f & 2) !== 0;
	captured_signals?.add(signal);
	if (active_reaction !== null && !untracking) {
		if (!(active_effect !== null && (active_effect.f & 16384) !== 0) && !current_sources?.includes(signal)) {
			var deps = active_reaction.deps;
			if ((active_reaction.f & 2097152) !== 0) {
				if (signal.rv < read_version) {
					signal.rv = read_version;
					if (new_deps === null && deps !== null && deps[skipped_deps] === signal) skipped_deps++;
					else if (new_deps === null) new_deps = [signal];
					else if (!skip_reaction || !new_deps.includes(signal)) new_deps.push(signal);
				}
			} else {
				(active_reaction.deps ??= []).push(signal);
				var reactions = signal.reactions;
				if (reactions === null) signal.reactions = [active_reaction];
				else if (!reactions.includes(active_reaction)) reactions.push(active_reaction);
			}
		}
	} else if (is_derived && signal.deps === null && signal.effects === null) {
		var derived$1 = signal;
		var parent = derived$1.parent;
		if (parent !== null && (parent.f & 256) === 0) derived$1.f ^= 256;
	}
	if (is_destroying_effect) {
		if (old_values.has(signal)) return old_values.get(signal);
		if (is_derived) {
			derived$1 = signal;
			var value = derived$1.v;
			if ((derived$1.f & 1024) === 0 && derived$1.reactions !== null || depends_on_old_values(derived$1)) value = execute_derived(derived$1);
			old_values.set(derived$1, value);
			return value;
		}
	} else if (is_derived) {
		derived$1 = signal;
		if (batch_values?.has(derived$1)) return batch_values.get(derived$1);
		if (is_dirty(derived$1)) update_derived(derived$1);
	}
	if (batch_values?.has(signal)) return batch_values.get(signal);
	if ((signal.f & 8388608) !== 0) throw signal.v;
	return signal.v;
}
function depends_on_old_values(derived$1) {
	if (derived$1.v === UNINITIALIZED) return true;
	if (derived$1.deps === null) return false;
	for (const dep of derived$1.deps) {
		if (old_values.has(dep)) return true;
		if ((dep.f & 2) !== 0 && depends_on_old_values(dep)) return true;
	}
	return false;
}
function untrack(fn) {
	var previous_untracking = untracking;
	try {
		untracking = true;
		return fn();
	} finally {
		untracking = previous_untracking;
	}
}
var STATUS_MASK = ~(MAYBE_DIRTY | 3072);
function set_signal_status(signal, status) {
	signal.f = signal.f & STATUS_MASK | status;
}
function deep_read_state(value) {
	if (typeof value !== "object" || !value || value instanceof EventTarget) return;
	if (STATE_SYMBOL in value) deep_read(value);
	else if (!Array.isArray(value)) for (let key in value) {
		const prop$1 = value[key];
		if (typeof prop$1 === "object" && prop$1 && STATE_SYMBOL in prop$1) deep_read(prop$1);
	}
}
function deep_read(value, visited = /* @__PURE__ */ new Set()) {
	if (typeof value === "object" && value !== null && !(value instanceof EventTarget) && !visited.has(value)) {
		visited.add(value);
		if (value instanceof Date) value.getTime();
		for (let key in value) try {
			deep_read(value[key], visited);
		} catch (e) {}
		const proto = get_prototype_of(value);
		if (proto !== Object.prototype && proto !== Array.prototype && proto !== Map.prototype && proto !== Set.prototype && proto !== Date.prototype) {
			const descriptors = get_descriptors(proto);
			for (let key in descriptors) {
				const get$2 = descriptors[key].get;
				if (get$2) try {
					get$2.call(value);
				} catch (e) {}
			}
		}
	}
}
function is_capture_event(name) {
	return name.endsWith("capture") && name !== "gotpointercapture" && name !== "lostpointercapture";
}
var DELEGATED_EVENTS = [
	"beforeinput",
	"click",
	"change",
	"dblclick",
	"contextmenu",
	"focusin",
	"focusout",
	"input",
	"keydown",
	"keyup",
	"mousedown",
	"mousemove",
	"mouseout",
	"mouseover",
	"mouseup",
	"pointerdown",
	"pointermove",
	"pointerout",
	"pointerover",
	"pointerup",
	"touchend",
	"touchmove",
	"touchstart"
];
function can_delegate_event(event_name) {
	return DELEGATED_EVENTS.includes(event_name);
}
var DOM_BOOLEAN_ATTRIBUTES = [
	"allowfullscreen",
	"async",
	"autofocus",
	"autoplay",
	"checked",
	"controls",
	"default",
	"disabled",
	"formnovalidate",
	"indeterminate",
	"inert",
	"ismap",
	"loop",
	"multiple",
	"muted",
	"nomodule",
	"novalidate",
	"open",
	"playsinline",
	"readonly",
	"required",
	"reversed",
	"seamless",
	"selected",
	"webkitdirectory",
	"defer",
	"disablepictureinpicture",
	"disableremoteplayback"
];
var ATTRIBUTE_ALIASES = {
	formnovalidate: "formNoValidate",
	ismap: "isMap",
	nomodule: "noModule",
	playsinline: "playsInline",
	readonly: "readOnly",
	defaultvalue: "defaultValue",
	defaultchecked: "defaultChecked",
	srcobject: "srcObject",
	novalidate: "noValidate",
	allowfullscreen: "allowFullscreen",
	disablepictureinpicture: "disablePictureInPicture",
	disableremoteplayback: "disableRemotePlayback"
};
function normalize_attribute(name) {
	name = name.toLowerCase();
	return ATTRIBUTE_ALIASES[name] ?? name;
}
[...DOM_BOOLEAN_ATTRIBUTES];
var PASSIVE_EVENTS = ["touchstart", "touchmove"];
function is_passive_event(name) {
	return PASSIVE_EVENTS.includes(name);
}
const all_registered_events = /* @__PURE__ */ new Set();
const root_event_handles = /* @__PURE__ */ new Set();
function create_event(event_name, dom, handler, options = {}) {
	function target_handler(event$1) {
		if (!options.capture) handle_event_propagation.call(dom, event$1);
		if (!event$1.cancelBubble) return without_reactive_context(() => {
			return handler?.call(this, event$1);
		});
	}
	if (event_name.startsWith("pointer") || event_name.startsWith("touch") || event_name === "wheel") queue_micro_task(() => {
		dom.addEventListener(event_name, target_handler, options);
	});
	else dom.addEventListener(event_name, target_handler, options);
	return target_handler;
}
function event(event_name, dom, handler, capture$1, passive) {
	var options = {
		capture: capture$1,
		passive
	};
	var target_handler = create_event(event_name, dom, handler, options);
	if (dom === document.body || dom === window || dom === document || dom instanceof HTMLMediaElement) teardown(() => {
		dom.removeEventListener(event_name, target_handler, options);
	});
}
function delegate(events) {
	for (var i = 0; i < events.length; i++) all_registered_events.add(events[i]);
	for (var fn of root_event_handles) fn(events);
}
var last_propagated_event = null;
function handle_event_propagation(event$1) {
	var handler_element = this;
	var owner_document = handler_element.ownerDocument;
	var event_name = event$1.type;
	var path = event$1.composedPath?.() || [];
	var current_target = path[0] || event$1.target;
	last_propagated_event = event$1;
	var path_idx = 0;
	var handled_at = last_propagated_event === event$1 && event$1.__root;
	if (handled_at) {
		var at_idx = path.indexOf(handled_at);
		if (at_idx !== -1 && (handler_element === document || handler_element === window)) {
			event$1.__root = handler_element;
			return;
		}
		var handler_idx = path.indexOf(handler_element);
		if (handler_idx === -1) return;
		if (at_idx <= handler_idx) path_idx = at_idx;
	}
	current_target = path[path_idx] || event$1.target;
	if (current_target === handler_element) return;
	define_property(event$1, "currentTarget", {
		configurable: true,
		get() {
			return current_target || owner_document;
		}
	});
	var previous_reaction = active_reaction;
	var previous_effect = active_effect;
	set_active_reaction(null);
	set_active_effect(null);
	try {
		var throw_error;
		var other_errors = [];
		while (current_target !== null) {
			var parent_element = current_target.assignedSlot || current_target.parentNode || current_target.host || null;
			try {
				var delegated = current_target["__" + event_name];
				if (delegated != null && (!current_target.disabled || event$1.target === current_target)) delegated.call(current_target, event$1);
			} catch (error) {
				if (throw_error) other_errors.push(error);
				else throw_error = error;
			}
			if (event$1.cancelBubble || parent_element === handler_element || parent_element === null) break;
			current_target = parent_element;
		}
		if (throw_error) {
			for (let error of other_errors) queueMicrotask(() => {
				throw error;
			});
			throw throw_error;
		}
	} finally {
		event$1.__root = handler_element;
		delete event$1.currentTarget;
		set_active_reaction(previous_reaction);
		set_active_effect(previous_effect);
	}
}
var head_anchor;
function reset_head_anchor() {
	head_anchor = void 0;
}
function head(render_fn) {
	let previous_hydrate_node = null;
	let was_hydrating = hydrating;
	var anchor;
	if (hydrating) {
		previous_hydrate_node = hydrate_node;
		if (head_anchor === void 0) head_anchor = /* @__PURE__ */ get_first_child(document.head);
		while (head_anchor !== null && (head_anchor.nodeType !== 8 || head_anchor.data !== "[")) head_anchor = /* @__PURE__ */ get_next_sibling(head_anchor);
		if (head_anchor === null) set_hydrating(false);
		else head_anchor = set_hydrate_node(/* @__PURE__ */ get_next_sibling(head_anchor));
	}
	if (!hydrating) anchor = document.head.appendChild(create_text());
	try {
		block(() => render_fn(anchor), HEAD_EFFECT);
	} finally {
		if (was_hydrating) {
			set_hydrating(true);
			head_anchor = hydrate_node;
			set_hydrate_node(previous_hydrate_node);
		}
	}
}
function create_fragment_from_html(html$1) {
	var elem = document.createElement("template");
	elem.innerHTML = html$1.replaceAll("<!>", "<!---->");
	return elem.content;
}
function assign_nodes(start, end) {
	var effect$1 = active_effect;
	if (effect$1.nodes_start === null) {
		effect$1.nodes_start = start;
		effect$1.nodes_end = end;
	}
}
/* @__NO_SIDE_EFFECTS__ */
function from_html(content, flags$1) {
	var is_fragment = (flags$1 & 1) !== 0;
	var use_import_node = (flags$1 & 2) !== 0;
	var node;
	var has_start = !content.startsWith("<!>");
	return () => {
		if (hydrating) {
			assign_nodes(hydrate_node, null);
			return hydrate_node;
		}
		if (node === void 0) {
			node = create_fragment_from_html(has_start ? content : "<!>" + content);
			if (!is_fragment) node = /* @__PURE__ */ get_first_child(node);
		}
		var clone$1 = use_import_node || is_firefox ? document.importNode(node, true) : node.cloneNode(true);
		if (is_fragment) {
			var start = /* @__PURE__ */ get_first_child(clone$1);
			var end = clone$1.lastChild;
			assign_nodes(start, end);
		} else assign_nodes(clone$1, clone$1);
		return clone$1;
	};
}
function text(value = "") {
	if (!hydrating) {
		var t = create_text(value + "");
		assign_nodes(t, t);
		return t;
	}
	var node = hydrate_node;
	if (node.nodeType !== 3) {
		node.before(node = create_text());
		set_hydrate_node(node);
	}
	assign_nodes(node, node);
	return node;
}
function comment() {
	if (hydrating) {
		assign_nodes(hydrate_node, null);
		return hydrate_node;
	}
	var frag = document.createDocumentFragment();
	var start = document.createComment("");
	var anchor = create_text();
	frag.append(start, anchor);
	assign_nodes(start, anchor);
	return frag;
}
function append(anchor, dom) {
	if (hydrating) {
		active_effect.nodes_end = hydrate_node;
		hydrate_next();
		return;
	}
	if (anchor === null) return;
	anchor.before(dom);
}
function set_text(text$1, value) {
	var str = value == null ? "" : typeof value === "object" ? value + "" : value;
	if (str !== (text$1.__t ??= text$1.nodeValue)) {
		text$1.__t = str;
		text$1.nodeValue = str + "";
	}
}
function mount(component$1, options) {
	return _mount(component$1, options);
}
function hydrate(component$1, options) {
	init_operations();
	options.intro = options.intro ?? false;
	const target = options.target;
	const was_hydrating = hydrating;
	const previous_hydrate_node = hydrate_node;
	try {
		var anchor = /* @__PURE__ */ get_first_child(target);
		while (anchor && (anchor.nodeType !== 8 || anchor.data !== "[")) anchor = /* @__PURE__ */ get_next_sibling(anchor);
		if (!anchor) throw HYDRATION_ERROR;
		set_hydrating(true);
		set_hydrate_node(anchor);
		const instance = _mount(component$1, {
			...options,
			anchor
		});
		set_hydrating(false);
		return instance;
	} catch (error) {
		if (error instanceof Error && error.message.split("\n").some((line) => line.startsWith("https://svelte.dev/e/"))) throw error;
		if (error !== HYDRATION_ERROR) console.warn("Failed to hydrate: ", error);
		if (options.recover === false) hydration_failed();
		init_operations();
		clear_text_content(target);
		set_hydrating(false);
		return mount(component$1, options);
	} finally {
		set_hydrating(was_hydrating);
		set_hydrate_node(previous_hydrate_node);
		reset_head_anchor();
	}
}
var document_listeners = /* @__PURE__ */ new Map();
function _mount(Component, { target, anchor, props = {}, events, context, intro = true }) {
	init_operations();
	var registered_events = /* @__PURE__ */ new Set();
	var event_handle = (events$1) => {
		for (var i = 0; i < events$1.length; i++) {
			var event_name = events$1[i];
			if (registered_events.has(event_name)) continue;
			registered_events.add(event_name);
			var passive = is_passive_event(event_name);
			target.addEventListener(event_name, handle_event_propagation, { passive });
			var n = document_listeners.get(event_name);
			if (n === void 0) {
				document.addEventListener(event_name, handle_event_propagation, { passive });
				document_listeners.set(event_name, 1);
			} else document_listeners.set(event_name, n + 1);
		}
	};
	event_handle(array_from(all_registered_events));
	root_event_handles.add(event_handle);
	var component$1 = void 0;
	var unmount$1 = component_root(() => {
		var anchor_node = anchor ?? target.appendChild(create_text());
		boundary(anchor_node, { pending: () => {} }, (anchor_node$1) => {
			if (context) {
				push({});
				var ctx = component_context;
				ctx.c = context;
			}
			if (events) props.$$events = events;
			if (hydrating) assign_nodes(anchor_node$1, null);
			component$1 = Component(anchor_node$1, props) || {};
			if (hydrating) {
				active_effect.nodes_end = hydrate_node;
				if (hydrate_node === null || hydrate_node.nodeType !== 8 || hydrate_node.data !== "]") {
					hydration_mismatch();
					throw HYDRATION_ERROR;
				}
			}
			if (context) pop();
		});
		return () => {
			for (var event_name of registered_events) {
				target.removeEventListener(event_name, handle_event_propagation);
				var n = document_listeners.get(event_name);
				if (--n === 0) {
					document.removeEventListener(event_name, handle_event_propagation);
					document_listeners.delete(event_name);
				} else document_listeners.set(event_name, n);
			}
			root_event_handles.delete(event_handle);
			if (anchor_node !== anchor) anchor_node.parentNode?.removeChild(anchor_node);
		};
	});
	mounted_components.set(component$1, unmount$1);
	return component$1;
}
var mounted_components = /* @__PURE__ */ new WeakMap();
function unmount(component$1, options) {
	const fn = mounted_components.get(component$1);
	if (fn) {
		mounted_components.delete(component$1);
		return fn(options);
	}
	return Promise.resolve();
}
var BranchManager = class {
	anchor;
	#batches = /* @__PURE__ */ new Map();
	#onscreen = /* @__PURE__ */ new Map();
	#offscreen = /* @__PURE__ */ new Map();
	#transition = true;
	constructor(anchor, transition = true) {
		this.anchor = anchor;
		this.#transition = transition;
	}
	#commit = () => {
		var batch = current_batch;
		if (!this.#batches.has(batch)) return;
		var key = this.#batches.get(batch);
		var onscreen = this.#onscreen.get(key);
		if (onscreen) resume_effect(onscreen);
		else {
			var offscreen = this.#offscreen.get(key);
			if (offscreen) {
				this.#onscreen.set(key, offscreen.effect);
				this.#offscreen.delete(key);
				offscreen.fragment.lastChild.remove();
				this.anchor.before(offscreen.fragment);
				onscreen = offscreen.effect;
			}
		}
		for (const [b, k] of this.#batches) {
			this.#batches.delete(b);
			if (b === batch) break;
			const offscreen$1 = this.#offscreen.get(k);
			if (offscreen$1) {
				destroy_effect(offscreen$1.effect);
				this.#offscreen.delete(k);
			}
		}
		for (const [k, effect$1] of this.#onscreen) {
			if (k === key) continue;
			const on_destroy = () => {
				if (Array.from(this.#batches.values()).includes(k)) {
					var fragment = document.createDocumentFragment();
					move_effect(effect$1, fragment);
					fragment.append(create_text());
					this.#offscreen.set(k, {
						effect: effect$1,
						fragment
					});
				} else destroy_effect(effect$1);
				this.#onscreen.delete(k);
			};
			if (this.#transition || !onscreen) pause_effect(effect$1, on_destroy, false);
			else on_destroy();
		}
	};
	#discard = (batch) => {
		this.#batches.delete(batch);
		const keys = Array.from(this.#batches.values());
		for (const [k, branch$1] of this.#offscreen) if (!keys.includes(k)) {
			destroy_effect(branch$1.effect);
			this.#offscreen.delete(k);
		}
	};
	ensure(key, fn) {
		var batch = current_batch;
		var defer = should_defer_append();
		if (fn && !this.#onscreen.has(key) && !this.#offscreen.has(key)) if (defer) {
			var fragment = document.createDocumentFragment();
			var target = create_text();
			fragment.append(target);
			this.#offscreen.set(key, {
				effect: branch(() => fn(target)),
				fragment
			});
		} else this.#onscreen.set(key, branch(() => fn(this.anchor)));
		this.#batches.set(batch, key);
		if (defer) {
			for (const [k, effect$1] of this.#onscreen) if (k === key) batch.skipped_effects.delete(effect$1);
			else batch.skipped_effects.add(effect$1);
			for (const [k, branch$1] of this.#offscreen) if (k === key) batch.skipped_effects.delete(branch$1.effect);
			else batch.skipped_effects.add(branch$1.effect);
			batch.oncommit(this.#commit);
			batch.ondiscard(this.#discard);
		} else {
			if (hydrating) this.anchor = hydrate_node;
			this.#commit();
		}
	}
};
function if_block(node, fn, elseif = false) {
	if (hydrating) hydrate_next();
	var branches = new BranchManager(node);
	var flags$1 = elseif ? EFFECT_TRANSPARENT : 0;
	function update_branch(condition, fn$1) {
		if (hydrating) {
			if (condition === (read_hydration_instruction(node) === "[!")) {
				var anchor = skip_nodes();
				set_hydrate_node(anchor);
				branches.anchor = anchor;
				set_hydrating(false);
				branches.ensure(condition, fn$1);
				set_hydrating(true);
				return;
			}
		}
		branches.ensure(condition, fn$1);
	}
	block(() => {
		var has_branch = false;
		fn((fn$1, flag = true) => {
			has_branch = true;
			update_branch(flag, fn$1);
		});
		if (!has_branch) update_branch(false, null);
	}, flags$1);
}
let current_each_item = null;
function index(_, i) {
	return i;
}
function pause_effects(state$1, items, controlled_anchor) {
	var items_map = state$1.items;
	var transitions = [];
	var length = items.length;
	for (var i = 0; i < length; i++) pause_children(items[i].e, transitions, true);
	var is_controlled = length > 0 && transitions.length === 0 && controlled_anchor !== null;
	if (is_controlled) {
		var parent_node = controlled_anchor.parentNode;
		clear_text_content(parent_node);
		parent_node.append(controlled_anchor);
		items_map.clear();
		link(state$1, items[0].prev, items[length - 1].next);
	}
	run_out_transitions(transitions, () => {
		for (var i$1 = 0; i$1 < length; i$1++) {
			var item = items[i$1];
			if (!is_controlled) {
				items_map.delete(item.k);
				link(state$1, item.prev, item.next);
			}
			destroy_effect(item.e, !is_controlled);
		}
	});
}
function each(node, flags$1, get_collection, get_key, render_fn, fallback_fn = null) {
	var anchor = node;
	var state$1 = {
		flags: flags$1,
		items: /* @__PURE__ */ new Map(),
		first: null
	};
	if ((flags$1 & 4) !== 0) {
		var parent_node = node;
		anchor = hydrating ? set_hydrate_node(/* @__PURE__ */ get_first_child(parent_node)) : parent_node.appendChild(create_text());
	}
	if (hydrating) hydrate_next();
	var fallback = null;
	var was_empty = false;
	var offscreen_items = /* @__PURE__ */ new Map();
	var each_array = /* @__PURE__ */ derived_safe_equal(() => {
		var collection = get_collection();
		return is_array(collection) ? collection : collection == null ? [] : array_from(collection);
	});
	var array;
	var each_effect;
	function commit() {
		reconcile(each_effect, array, state$1, offscreen_items, anchor, render_fn, flags$1, get_key, get_collection);
		if (fallback_fn !== null) {
			if (array.length === 0) if (fallback) resume_effect(fallback);
			else fallback = branch(() => fallback_fn(anchor));
			else if (fallback !== null) pause_effect(fallback, () => {
				fallback = null;
			});
		}
	}
	block(() => {
		each_effect ??= active_effect;
		array = get$1(each_array);
		var length = array.length;
		if (was_empty && length === 0) return;
		was_empty = length === 0;
		let mismatch = false;
		if (hydrating) {
			if (read_hydration_instruction(anchor) === "[!" !== (length === 0)) {
				anchor = skip_nodes();
				set_hydrate_node(anchor);
				set_hydrating(false);
				mismatch = true;
			}
		}
		if (hydrating) {
			var prev = null;
			var item;
			for (var i = 0; i < length; i++) {
				if (hydrate_node.nodeType === 8 && hydrate_node.data === "]") {
					anchor = hydrate_node;
					mismatch = true;
					set_hydrating(false);
					break;
				}
				var value = array[i];
				var key = get_key(value, i);
				item = create_item(hydrate_node, state$1, prev, null, value, key, i, render_fn, flags$1, get_collection);
				state$1.items.set(key, item);
				prev = item;
			}
			if (length > 0) set_hydrate_node(skip_nodes());
		}
		if (hydrating) {
			if (length === 0 && fallback_fn) fallback = branch(() => fallback_fn(anchor));
		} else if (should_defer_append()) {
			var keys = /* @__PURE__ */ new Set();
			var batch = current_batch;
			for (i = 0; i < length; i += 1) {
				value = array[i];
				key = get_key(value, i);
				var existing = state$1.items.get(key) ?? offscreen_items.get(key);
				if (existing) {
					if ((flags$1 & 3) !== 0) update_item(existing, value, i, flags$1);
				} else {
					item = create_item(null, state$1, null, null, value, key, i, render_fn, flags$1, get_collection, true);
					offscreen_items.set(key, item);
				}
				keys.add(key);
			}
			for (const [key$1, item$1] of state$1.items) if (!keys.has(key$1)) batch.skipped_effects.add(item$1.e);
			batch.oncommit(commit);
		} else commit();
		if (mismatch) set_hydrating(true);
		get$1(each_array);
	});
	if (hydrating) anchor = hydrate_node;
}
function reconcile(each_effect, array, state$1, offscreen_items, anchor, render_fn, flags$1, get_key, get_collection) {
	var is_animated = (flags$1 & 8) !== 0;
	var should_update = (flags$1 & 3) !== 0;
	var length = array.length;
	var items = state$1.items;
	var current = state$1.first;
	var seen;
	var prev = null;
	var to_animate;
	var matched = [];
	var stashed = [];
	var value;
	var key;
	var item;
	var i;
	if (is_animated) for (i = 0; i < length; i += 1) {
		value = array[i];
		key = get_key(value, i);
		item = items.get(key);
		if (item !== void 0) {
			item.a?.measure();
			(to_animate ??= /* @__PURE__ */ new Set()).add(item);
		}
	}
	for (i = 0; i < length; i += 1) {
		value = array[i];
		key = get_key(value, i);
		item = items.get(key);
		if (item === void 0) {
			var pending = offscreen_items.get(key);
			if (pending !== void 0) {
				offscreen_items.delete(key);
				items.set(key, pending);
				var next$1 = prev ? prev.next : current;
				link(state$1, prev, pending);
				link(state$1, pending, next$1);
				move(pending, next$1, anchor);
				prev = pending;
			} else prev = create_item(current ? current.e.nodes_start : anchor, state$1, prev, prev === null ? state$1.first : prev.next, value, key, i, render_fn, flags$1, get_collection);
			items.set(key, prev);
			matched = [];
			stashed = [];
			current = prev.next;
			continue;
		}
		if (should_update) update_item(item, value, i, flags$1);
		if ((item.e.f & 8192) !== 0) {
			resume_effect(item.e);
			if (is_animated) {
				item.a?.unfix();
				(to_animate ??= /* @__PURE__ */ new Set()).delete(item);
			}
		}
		if (item !== current) {
			if (seen !== void 0 && seen.has(item)) {
				if (matched.length < stashed.length) {
					var start = stashed[0];
					var j;
					prev = start.prev;
					var a = matched[0];
					var b = matched[matched.length - 1];
					for (j = 0; j < matched.length; j += 1) move(matched[j], start, anchor);
					for (j = 0; j < stashed.length; j += 1) seen.delete(stashed[j]);
					link(state$1, a.prev, b.next);
					link(state$1, prev, a);
					link(state$1, b, start);
					current = start;
					prev = b;
					i -= 1;
					matched = [];
					stashed = [];
				} else {
					seen.delete(item);
					move(item, current, anchor);
					link(state$1, item.prev, item.next);
					link(state$1, item, prev === null ? state$1.first : prev.next);
					link(state$1, prev, item);
					prev = item;
				}
				continue;
			}
			matched = [];
			stashed = [];
			while (current !== null && current.k !== key) {
				if ((current.e.f & 8192) === 0) (seen ??= /* @__PURE__ */ new Set()).add(current);
				stashed.push(current);
				current = current.next;
			}
			if (current === null) continue;
			item = current;
		}
		matched.push(item);
		prev = item;
		current = item.next;
	}
	if (current !== null || seen !== void 0) {
		var to_destroy = seen === void 0 ? [] : array_from(seen);
		while (current !== null) {
			if ((current.e.f & 8192) === 0) to_destroy.push(current);
			current = current.next;
		}
		var destroy_length = to_destroy.length;
		if (destroy_length > 0) {
			var controlled_anchor = (flags$1 & 4) !== 0 && length === 0 ? anchor : null;
			if (is_animated) {
				for (i = 0; i < destroy_length; i += 1) to_destroy[i].a?.measure();
				for (i = 0; i < destroy_length; i += 1) to_destroy[i].a?.fix();
			}
			pause_effects(state$1, to_destroy, controlled_anchor);
		}
	}
	if (is_animated) queue_micro_task(() => {
		if (to_animate === void 0) return;
		for (item of to_animate) item.a?.apply();
	});
	each_effect.first = state$1.first && state$1.first.e;
	each_effect.last = prev && prev.e;
	for (var unused of offscreen_items.values()) destroy_effect(unused.e);
	offscreen_items.clear();
}
function update_item(item, value, index$1, type) {
	if ((type & 1) !== 0) internal_set(item.v, value);
	if ((type & 2) !== 0) internal_set(item.i, index$1);
	else item.i = index$1;
}
function create_item(anchor, state$1, prev, next$1, value, key, index$1, render_fn, flags$1, get_collection, deferred$1) {
	var previous_each_item = current_each_item;
	var reactive = (flags$1 & 1) !== 0;
	var mutable = (flags$1 & 16) === 0;
	var v = reactive ? mutable ? /* @__PURE__ */ mutable_source(value, false, false) : source(value) : value;
	var i = (flags$1 & 2) === 0 ? index$1 : source(index$1);
	var item = {
		i,
		v,
		k: key,
		a: null,
		e: null,
		prev,
		next: next$1
	};
	current_each_item = item;
	try {
		if (anchor === null) document.createDocumentFragment().append(anchor = create_text());
		item.e = branch(() => render_fn(anchor, v, i, get_collection), hydrating);
		item.e.prev = prev && prev.e;
		item.e.next = next$1 && next$1.e;
		if (prev === null) {
			if (!deferred$1) state$1.first = item;
		} else {
			prev.next = item;
			prev.e.next = item.e;
		}
		if (next$1 !== null) {
			next$1.prev = item;
			next$1.e.prev = item.e;
		}
		return item;
	} finally {
		current_each_item = previous_each_item;
	}
}
function move(item, next$1, anchor) {
	var end = item.next ? item.next.e.nodes_start : anchor;
	var dest = next$1 ? next$1.e.nodes_start : anchor;
	var node = item.e.nodes_start;
	while (node !== null && node !== end) {
		var next_node = /* @__PURE__ */ get_next_sibling(node);
		dest.before(node);
		node = next_node;
	}
}
function link(state$1, prev, next$1) {
	if (prev === null) state$1.first = next$1;
	else {
		prev.next = next$1;
		prev.e.next = next$1 && next$1.e;
	}
	if (next$1 !== null) {
		next$1.prev = prev;
		next$1.e.prev = prev && prev.e;
	}
}
function html(node, get_value, svg = false, mathml = false, skip_warning = false) {
	var anchor = node;
	var value = "";
	template_effect(() => {
		var effect$1 = active_effect;
		if (value === (value = get_value() ?? "")) {
			if (hydrating) hydrate_next();
			return;
		}
		if (effect$1.nodes_start !== null) {
			remove_effect_dom(effect$1.nodes_start, effect$1.nodes_end);
			effect$1.nodes_start = effect$1.nodes_end = null;
		}
		if (value === "") return;
		if (hydrating) {
			hydrate_node.data;
			var next$1 = hydrate_next();
			var last = next$1;
			while (next$1 !== null && (next$1.nodeType !== 8 || next$1.data !== "")) {
				last = next$1;
				next$1 = /* @__PURE__ */ get_next_sibling(next$1);
			}
			if (next$1 === null) {
				hydration_mismatch();
				throw HYDRATION_ERROR;
			}
			assign_nodes(hydrate_node, last);
			anchor = set_hydrate_node(next$1);
			return;
		}
		var html$1 = value + "";
		if (svg) html$1 = `<svg>${html$1}</svg>`;
		else if (mathml) html$1 = `<math>${html$1}</math>`;
		var node$1 = create_fragment_from_html(html$1);
		if (svg || mathml) node$1 = /* @__PURE__ */ get_first_child(node$1);
		assign_nodes(/* @__PURE__ */ get_first_child(node$1), node$1.lastChild);
		if (svg || mathml) while (/* @__PURE__ */ get_first_child(node$1)) anchor.before(/* @__PURE__ */ get_first_child(node$1));
		else anchor.before(node$1);
	});
}
function snippet(node, get_snippet, ...args) {
	var branches = new BranchManager(node);
	block(() => {
		const snippet$1 = get_snippet() ?? null;
		branches.ensure(snippet$1, snippet$1 && ((anchor) => snippet$1(anchor, ...args)));
	}, EFFECT_TRANSPARENT);
}
function createRawSnippet(fn) {
	return (anchor, ...params) => {
		var snippet$1 = fn(...params);
		var element;
		if (hydrating) {
			element = hydrate_node;
			hydrate_next();
		} else {
			element = /* @__PURE__ */ get_first_child(create_fragment_from_html(snippet$1.render().trim()));
			anchor.before(element);
		}
		const result = snippet$1.setup?.(element);
		assign_nodes(element, element);
		if (typeof result === "function") teardown(result);
	};
}
function component(node, get_component, render_fn) {
	if (hydrating) hydrate_next();
	var branches = new BranchManager(node);
	block(() => {
		var component$1 = get_component() ?? null;
		branches.ensure(component$1, component$1 && ((target) => render_fn(target, component$1)));
	}, EFFECT_TRANSPARENT);
}
function action(dom, action$1, get_value) {
	effect(() => {
		var payload = untrack(() => action$1(dom, get_value?.()) || {});
		if (get_value && payload?.update) {
			var inited = false;
			var prev = {};
			render_effect(() => {
				var value = get_value();
				deep_read_state(value);
				if (inited && safe_not_equal(prev, value)) {
					prev = value;
					payload.update(value);
				}
			});
			inited = true;
		}
		if (payload?.destroy) return () => payload.destroy();
	});
}
function attach(node, get_fn) {
	var fn = void 0;
	var e;
	block(() => {
		if (fn !== (fn = get_fn())) {
			if (e) {
				destroy_effect(e);
				e = null;
			}
			if (fn) e = branch(() => {
				effect(() => fn(node));
			});
		}
	});
}
function r(e) {
	var t, f, n = "";
	if ("string" == typeof e || "number" == typeof e) n += e;
	else if ("object" == typeof e) if (Array.isArray(e)) {
		var o = e.length;
		for (t = 0; t < o; t++) e[t] && (f = r(e[t])) && (n && (n += " "), n += f);
	} else for (f in e) e[f] && (n && (n += " "), n += f);
	return n;
}
function clsx$1() {
	for (var e, t, f = 0, n = "", o = arguments.length; f < o; f++) (e = arguments[f]) && (t = r(e)) && (n && (n += " "), n += t);
	return n;
}
function clsx(value) {
	if (typeof value === "object") return clsx$1(value);
	else return value ?? "";
}
var whitespace = [..." 	\n\r\f\xA0\v﻿"];
function to_class(value, hash$1, directives) {
	var classname = value == null ? "" : "" + value;
	if (hash$1) classname = classname ? classname + " " + hash$1 : hash$1;
	if (directives) {
		for (var key in directives) if (directives[key]) classname = classname ? classname + " " + key : key;
		else if (classname.length) {
			var len = key.length;
			var a = 0;
			while ((a = classname.indexOf(key, a)) >= 0) {
				var b = a + len;
				if ((a === 0 || whitespace.includes(classname[a - 1])) && (b === classname.length || whitespace.includes(classname[b]))) classname = (a === 0 ? "" : classname.substring(0, a)) + classname.substring(b + 1);
				else a = b;
			}
		}
	}
	return classname === "" ? null : classname;
}
function append_styles(styles, important = false) {
	var separator = important ? " !important;" : ";";
	var css = "";
	for (var key in styles) {
		var value = styles[key];
		if (value != null && value !== "") css += " " + key + ": " + value + separator;
	}
	return css;
}
function to_css_name(name) {
	if (name[0] !== "-" || name[1] !== "-") return name.toLowerCase();
	return name;
}
function to_style(value, styles) {
	if (styles) {
		var new_style = "";
		var normal_styles;
		var important_styles;
		if (Array.isArray(styles)) {
			normal_styles = styles[0];
			important_styles = styles[1];
		} else normal_styles = styles;
		if (value) {
			value = String(value).replaceAll(/\s*\/\*.*?\*\/\s*/g, "").trim();
			var in_str = false;
			var in_apo = 0;
			var in_comment = false;
			var reserved_names = [];
			if (normal_styles) reserved_names.push(...Object.keys(normal_styles).map(to_css_name));
			if (important_styles) reserved_names.push(...Object.keys(important_styles).map(to_css_name));
			var start_index = 0;
			var name_index = -1;
			const len = value.length;
			for (var i = 0; i < len; i++) {
				var c = value[i];
				if (in_comment) {
					if (c === "/" && value[i - 1] === "*") in_comment = false;
				} else if (in_str) {
					if (in_str === c) in_str = false;
				} else if (c === "/" && value[i + 1] === "*") in_comment = true;
				else if (c === "\"" || c === "'") in_str = c;
				else if (c === "(") in_apo++;
				else if (c === ")") in_apo--;
				if (!in_comment && in_str === false && in_apo === 0) {
					if (c === ":" && name_index === -1) name_index = i;
					else if (c === ";" || i === len - 1) {
						if (name_index !== -1) {
							var name = to_css_name(value.substring(start_index, name_index).trim());
							if (!reserved_names.includes(name)) {
								if (c !== ";") i++;
								var property = value.substring(start_index, i).trim();
								new_style += " " + property + ";";
							}
						}
						start_index = i + 1;
						name_index = -1;
					}
				}
			}
		}
		if (normal_styles) new_style += append_styles(normal_styles);
		if (important_styles) new_style += append_styles(important_styles, true);
		new_style = new_style.trim();
		return new_style === "" ? null : new_style;
	}
	return value == null ? null : String(value);
}
function set_class(dom, is_html, value, hash$1, prev_classes, next_classes) {
	var prev = dom.__className;
	if (hydrating || prev !== value || prev === void 0) {
		var next_class_name = to_class(value, hash$1, next_classes);
		if (!hydrating || next_class_name !== dom.getAttribute("class")) if (next_class_name == null) dom.removeAttribute("class");
		else if (is_html) dom.className = next_class_name;
		else dom.setAttribute("class", next_class_name);
		dom.__className = value;
	} else if (next_classes && prev_classes !== next_classes) for (var key in next_classes) {
		var is_present = !!next_classes[key];
		if (prev_classes == null || is_present !== !!prev_classes[key]) dom.classList.toggle(key, is_present);
	}
	return next_classes;
}
function update_styles(dom, prev = {}, next$1, priority) {
	for (var key in next$1) {
		var value = next$1[key];
		if (prev[key] !== value) if (next$1[key] == null) dom.style.removeProperty(key);
		else dom.style.setProperty(key, value, priority);
	}
}
function set_style(dom, value, prev_styles, next_styles) {
	var prev = dom.__style;
	if (hydrating || prev !== value) {
		var next_style_attr = to_style(value, next_styles);
		if (!hydrating || next_style_attr !== dom.getAttribute("style")) if (next_style_attr == null) dom.removeAttribute("style");
		else dom.style.cssText = next_style_attr;
		dom.__style = value;
	} else if (next_styles) if (Array.isArray(next_styles)) {
		update_styles(dom, prev_styles?.[0], next_styles[0]);
		update_styles(dom, prev_styles?.[1], next_styles[1], "important");
	} else update_styles(dom, prev_styles, next_styles);
	return next_styles;
}
function select_option(select, value, mounting = false) {
	if (select.multiple) {
		if (value == void 0) return;
		if (!is_array(value)) return select_multiple_invalid_value();
		for (var option of select.options) option.selected = value.includes(get_option_value(option));
		return;
	}
	for (option of select.options) if (is(get_option_value(option), value)) {
		option.selected = true;
		return;
	}
	if (!mounting || value !== void 0) select.selectedIndex = -1;
}
function init_select(select) {
	var observer = new MutationObserver(() => {
		select_option(select, select.__value);
	});
	observer.observe(select, {
		childList: true,
		subtree: true,
		attributes: true,
		attributeFilter: ["value"]
	});
	teardown(() => {
		observer.disconnect();
	});
}
function bind_select_value(select, get$2, set$1 = get$2) {
	var batches$1 = /* @__PURE__ */ new WeakSet();
	var mounting = true;
	listen_to_event_and_reset_event(select, "change", (is_reset) => {
		var query = is_reset ? "[selected]" : ":checked";
		var value;
		if (select.multiple) value = [].map.call(select.querySelectorAll(query), get_option_value);
		else {
			var selected_option = select.querySelector(query) ?? select.querySelector("option:not([disabled])");
			value = selected_option && get_option_value(selected_option);
		}
		set$1(value);
		if (current_batch !== null) batches$1.add(current_batch);
	});
	effect(() => {
		var value = get$2();
		if (select === document.activeElement) {
			var batch = previous_batch ?? current_batch;
			if (batches$1.has(batch)) return;
		}
		select_option(select, value, mounting);
		if (mounting && value === void 0) {
			var selected_option = select.querySelector(":checked");
			if (selected_option !== null) {
				value = get_option_value(selected_option);
				set$1(value);
			}
		}
		select.__value = value;
		mounting = false;
	});
	init_select(select);
}
function get_option_value(option) {
	if ("__value" in option) return option.__value;
	else return option.value;
}
const CLASS = Symbol("class");
const STYLE = Symbol("style");
var IS_CUSTOM_ELEMENT = Symbol("is custom element");
var IS_HTML = Symbol("is html");
function remove_input_defaults(input) {
	if (!hydrating) return;
	var already_removed = false;
	var remove_defaults = () => {
		if (already_removed) return;
		already_removed = true;
		if (input.hasAttribute("value")) {
			var value = input.value;
			set_attribute(input, "value", null);
			input.value = value;
		}
		if (input.hasAttribute("checked")) {
			var checked = input.checked;
			set_attribute(input, "checked", null);
			input.checked = checked;
		}
	};
	input.__on_r = remove_defaults;
	queue_micro_task(remove_defaults);
	add_form_reset_listener();
}
function set_value(element, value) {
	var attributes = get_attributes(element);
	if (attributes.value === (attributes.value = value ?? void 0) || element.value === value && (value !== 0 || element.nodeName !== "PROGRESS")) return;
	element.value = value ?? "";
}
function set_selected(element, selected) {
	if (selected) {
		if (!element.hasAttribute("selected")) element.setAttribute("selected", "");
	} else element.removeAttribute("selected");
}
function set_attribute(element, attribute, value, skip_warning) {
	var attributes = get_attributes(element);
	if (hydrating) {
		attributes[attribute] = element.getAttribute(attribute);
		if (attribute === "src" || attribute === "srcset" || attribute === "href" && element.nodeName === "LINK") {
			if (!skip_warning) check_src_in_dev_hydration(element, attribute, value ?? "");
			return;
		}
	}
	if (attributes[attribute] === (attributes[attribute] = value)) return;
	if (attribute === "loading") element[LOADING_ATTR_SYMBOL] = value;
	if (value == null) element.removeAttribute(attribute);
	else if (typeof value !== "string" && get_setters(element).includes(attribute)) element[attribute] = value;
	else element.setAttribute(attribute, value);
}
function set_attributes(element, prev, next$1, css_hash, should_remove_defaults = false, skip_warning = false) {
	if (hydrating && should_remove_defaults && element.tagName === "INPUT") {
		var input = element;
		if (!((input.type === "checkbox" ? "defaultChecked" : "defaultValue") in next$1)) remove_input_defaults(input);
	}
	var attributes = get_attributes(element);
	var is_custom_element = attributes[IS_CUSTOM_ELEMENT];
	var preserve_attribute_case = !attributes[IS_HTML];
	let is_hydrating_custom_element = hydrating && is_custom_element;
	if (is_hydrating_custom_element) set_hydrating(false);
	var current = prev || {};
	var is_option_element = element.tagName === "OPTION";
	for (var key in prev) if (!(key in next$1)) next$1[key] = null;
	if (next$1.class) next$1.class = clsx(next$1.class);
	else if (css_hash || next$1[CLASS]) next$1.class = null;
	if (next$1[STYLE]) next$1.style ??= null;
	var setters = get_setters(element);
	for (const key$1 in next$1) {
		let value = next$1[key$1];
		if (is_option_element && key$1 === "value" && value == null) {
			element.value = element.__value = "";
			current[key$1] = value;
			continue;
		}
		if (key$1 === "class") {
			set_class(element, element.namespaceURI === "http://www.w3.org/1999/xhtml", value, css_hash, prev?.[CLASS], next$1[CLASS]);
			current[key$1] = value;
			current[CLASS] = next$1[CLASS];
			continue;
		}
		if (key$1 === "style") {
			set_style(element, value, prev?.[STYLE], next$1[STYLE]);
			current[key$1] = value;
			current[STYLE] = next$1[STYLE];
			continue;
		}
		var prev_value = current[key$1];
		if (value === prev_value && !(value === void 0 && element.hasAttribute(key$1))) continue;
		current[key$1] = value;
		var prefix = key$1[0] + key$1[1];
		if (prefix === "$$") continue;
		if (prefix === "on") {
			const opts = {};
			const event_handle_key = "$$" + key$1;
			let event_name = key$1.slice(2);
			var delegated = can_delegate_event(event_name);
			if (is_capture_event(event_name)) {
				event_name = event_name.slice(0, -7);
				opts.capture = true;
			}
			if (!delegated && prev_value) {
				if (value != null) continue;
				element.removeEventListener(event_name, current[event_handle_key], opts);
				current[event_handle_key] = null;
			}
			if (value != null) if (!delegated) {
				function handle(evt) {
					current[key$1].call(this, evt);
				}
				current[event_handle_key] = create_event(event_name, element, handle, opts);
			} else {
				element[`__${event_name}`] = value;
				delegate([event_name]);
			}
			else if (delegated) element[`__${event_name}`] = void 0;
		} else if (key$1 === "style") set_attribute(element, key$1, value);
		else if (key$1 === "autofocus") autofocus(element, Boolean(value));
		else if (!is_custom_element && (key$1 === "__value" || key$1 === "value" && value != null)) element.value = element.__value = value;
		else if (key$1 === "selected" && is_option_element) set_selected(element, value);
		else {
			var name = key$1;
			if (!preserve_attribute_case) name = normalize_attribute(name);
			var is_default = name === "defaultValue" || name === "defaultChecked";
			if (value == null && !is_custom_element && !is_default) {
				attributes[key$1] = null;
				if (name === "value" || name === "checked") {
					let input$1 = element;
					const use_default = prev === void 0;
					if (name === "value") {
						let previous = input$1.defaultValue;
						input$1.removeAttribute(name);
						input$1.defaultValue = previous;
						input$1.value = input$1.__value = use_default ? previous : null;
					} else {
						let previous = input$1.defaultChecked;
						input$1.removeAttribute(name);
						input$1.defaultChecked = previous;
						input$1.checked = use_default ? previous : false;
					}
				} else element.removeAttribute(key$1);
			} else if (is_default || setters.includes(name) && (is_custom_element || typeof value !== "string")) {
				element[name] = value;
				if (name in attributes) attributes[name] = UNINITIALIZED;
			} else if (typeof value !== "function") set_attribute(element, name, value, skip_warning);
		}
	}
	if (is_hydrating_custom_element) set_hydrating(true);
	return current;
}
function attribute_effect(element, fn, sync = [], async = [], css_hash, should_remove_defaults = false, skip_warning = false) {
	flatten(sync, async, (values) => {
		var prev = void 0;
		var effects = {};
		var is_select = element.nodeName === "SELECT";
		var inited = false;
		block(() => {
			var next$1 = fn(...values.map(get$1));
			var current = set_attributes(element, prev, next$1, css_hash, should_remove_defaults, skip_warning);
			if (inited && is_select && "value" in next$1) select_option(element, next$1.value);
			for (let symbol of Object.getOwnPropertySymbols(effects)) if (!next$1[symbol]) destroy_effect(effects[symbol]);
			for (let symbol of Object.getOwnPropertySymbols(next$1)) {
				var n = next$1[symbol];
				if (symbol.description === "@attach" && (!prev || n !== prev[symbol])) {
					if (effects[symbol]) destroy_effect(effects[symbol]);
					effects[symbol] = branch(() => attach(element, () => n));
				}
				current[symbol] = n;
			}
			prev = current;
		});
		if (is_select) {
			var select = element;
			effect(() => {
				select_option(select, prev.value, true);
				init_select(select);
			});
		}
		inited = true;
	});
}
function get_attributes(element) {
	return element.__attributes ??= {
		[IS_CUSTOM_ELEMENT]: element.nodeName.includes("-"),
		[IS_HTML]: element.namespaceURI === NAMESPACE_HTML
	};
}
var setters_cache = /* @__PURE__ */ new Map();
function get_setters(element) {
	var cache_key = element.getAttribute("is") || element.nodeName;
	var setters = setters_cache.get(cache_key);
	if (setters) return setters;
	setters_cache.set(cache_key, setters = []);
	var descriptors;
	var proto = element;
	var element_proto = Element.prototype;
	while (element_proto !== proto) {
		descriptors = get_descriptors(proto);
		for (var key in descriptors) if (descriptors[key].set) setters.push(key);
		proto = get_prototype_of(proto);
	}
	return setters;
}
function check_src_in_dev_hydration(element, attribute, value) {}
function bind_value(input, get$2, set$1 = get$2) {
	var batches$1 = /* @__PURE__ */ new WeakSet();
	listen_to_event_and_reset_event(input, "input", async (is_reset) => {
		var value = is_reset ? input.defaultValue : input.value;
		value = is_numberlike_input(input) ? to_number(value) : value;
		set$1(value);
		if (current_batch !== null) batches$1.add(current_batch);
		await tick();
		if (value !== (value = get$2())) {
			var start = input.selectionStart;
			var end = input.selectionEnd;
			var length = input.value.length;
			input.value = value ?? "";
			if (end !== null) {
				var new_length = input.value.length;
				if (start === end && end === length && new_length > length) {
					input.selectionStart = new_length;
					input.selectionEnd = new_length;
				} else {
					input.selectionStart = start;
					input.selectionEnd = Math.min(end, new_length);
				}
			}
		}
	});
	if (hydrating && input.defaultValue !== input.value || untrack(get$2) == null && input.value) {
		set$1(is_numberlike_input(input) ? to_number(input.value) : input.value);
		if (current_batch !== null) batches$1.add(current_batch);
	}
	render_effect(() => {
		var value = get$2();
		if (input === document.activeElement) {
			var batch = previous_batch ?? current_batch;
			if (batches$1.has(batch)) return;
		}
		if (is_numberlike_input(input) && value === to_number(input.value)) return;
		if (input.type === "date" && !value && !input.value) return;
		if (value !== input.value) input.value = value ?? "";
	});
}
function bind_checked(input, get$2, set$1 = get$2) {
	listen_to_event_and_reset_event(input, "change", (is_reset) => {
		set$1(is_reset ? input.defaultChecked : input.checked);
	});
	if (hydrating && input.defaultChecked !== input.checked || untrack(get$2) == null) set$1(input.checked);
	render_effect(() => {
		var value = get$2();
		input.checked = Boolean(value);
	});
}
function is_numberlike_input(input) {
	var type = input.type;
	return type === "number" || type === "range";
}
function to_number(value) {
	return value === "" ? null : +value;
}
function is_bound_this(bound_value, element_or_component) {
	return bound_value === element_or_component || bound_value?.[STATE_SYMBOL] === element_or_component;
}
function bind_this(element_or_component = {}, update$1, get_value, get_parts) {
	effect(() => {
		var old_parts;
		var parts;
		render_effect(() => {
			old_parts = parts;
			parts = get_parts?.() || [];
			untrack(() => {
				if (element_or_component !== get_value(...parts)) {
					update$1(element_or_component, ...parts);
					if (old_parts && is_bound_this(get_value(...old_parts), element_or_component)) update$1(null, ...old_parts);
				}
			});
		});
		return () => {
			queue_micro_task(() => {
				if (parts && is_bound_this(get_value(...parts), element_or_component)) update$1(null, ...parts);
			});
		};
	});
	return element_or_component;
}
function bind_window_size(type, set$1) {
	listen(window, ["resize"], () => without_reactive_context(() => set$1(window[type])));
}
function init(immutable = false) {
	const context = component_context;
	const callbacks = context.l.u;
	if (!callbacks) return;
	let props = () => deep_read_state(context.s);
	if (immutable) {
		let version = 0;
		let prev = {};
		const d = /* @__PURE__ */ derived(() => {
			let changed = false;
			const props$1 = context.s;
			for (const key in props$1) if (props$1[key] !== prev[key]) {
				prev[key] = props$1[key];
				changed = true;
			}
			if (changed) version++;
			return version;
		});
		props = () => get$1(d);
	}
	if (callbacks.b.length) user_pre_effect(() => {
		observe_all(context, props);
		run_all(callbacks.b);
	});
	user_effect(() => {
		const fns = untrack(() => callbacks.m.map(run));
		return () => {
			for (const fn of fns) if (typeof fn === "function") fn();
		};
	});
	if (callbacks.a.length) user_effect(() => {
		observe_all(context, props);
		run_all(callbacks.a);
	});
}
function observe_all(context, props) {
	if (context.l.s) for (const signal of context.l.s) get$1(signal);
	props();
}
function reactive_import(fn) {
	var s = source(0);
	return function() {
		if (arguments.length === 1) {
			set(s, get$1(s) + 1);
			return arguments[0];
		} else {
			get$1(s);
			return fn();
		}
	};
}
function subscribe_to_store(store, run$1, invalidate) {
	if (store == null) {
		run$1(void 0);
		if (invalidate) invalidate(void 0);
		return noop;
	}
	const unsub = untrack(() => store.subscribe(run$1, invalidate));
	return unsub.unsubscribe ? () => unsub.unsubscribe() : unsub;
}
var subscriber_queue = [];
function writable(value, start = noop) {
	let stop = null;
	const subscribers = /* @__PURE__ */ new Set();
	function set$1(new_value) {
		if (safe_not_equal(value, new_value)) {
			value = new_value;
			if (stop) {
				const run_queue = !subscriber_queue.length;
				for (const subscriber of subscribers) {
					subscriber[1]();
					subscriber_queue.push(subscriber, value);
				}
				if (run_queue) {
					for (let i = 0; i < subscriber_queue.length; i += 2) subscriber_queue[i][0](subscriber_queue[i + 1]);
					subscriber_queue.length = 0;
				}
			}
		}
	}
	function update$1(fn) {
		set$1(fn(value));
	}
	function subscribe(run$1, invalidate = noop) {
		const subscriber = [run$1, invalidate];
		subscribers.add(subscriber);
		if (subscribers.size === 1) stop = start(set$1, update$1) || noop;
		run$1(value);
		return () => {
			subscribers.delete(subscriber);
			if (subscribers.size === 0 && stop) {
				stop();
				stop = null;
			}
		};
	}
	return {
		set: set$1,
		update: update$1,
		subscribe
	};
}
function get(store) {
	let value;
	subscribe_to_store(store, (_) => value = _)();
	return value;
}
var is_store_binding = false;
var IS_UNMOUNTED = Symbol();
function store_get(store, store_name, stores) {
	const entry = stores[store_name] ??= {
		store: null,
		source: /* @__PURE__ */ mutable_source(void 0),
		unsubscribe: noop
	};
	if (entry.store !== store && !(IS_UNMOUNTED in stores)) {
		entry.unsubscribe();
		entry.store = store ?? null;
		if (store == null) {
			entry.source.v = void 0;
			entry.unsubscribe = noop;
		} else {
			var is_synchronous_callback = true;
			entry.unsubscribe = subscribe_to_store(store, (v) => {
				if (is_synchronous_callback) entry.source.v = v;
				else set(entry.source, v);
			});
			is_synchronous_callback = false;
		}
	}
	if (store && IS_UNMOUNTED in stores) return get(store);
	return get$1(entry.source);
}
function setup_stores() {
	const stores = {};
	function cleanup() {
		teardown(() => {
			for (var store_name in stores) stores[store_name].unsubscribe();
			define_property(stores, IS_UNMOUNTED, {
				enumerable: false,
				value: true
			});
		});
	}
	return [stores, cleanup];
}
function capture_store_binding(fn) {
	var previous_is_store_binding = is_store_binding;
	try {
		is_store_binding = false;
		return [fn(), is_store_binding];
	} finally {
		is_store_binding = previous_is_store_binding;
	}
}
function prop(props, key, flags$1, fallback) {
	var runes = !legacy_mode_flag || (flags$1 & 2) !== 0;
	var bindable = (flags$1 & 8) !== 0;
	var lazy = (flags$1 & 16) !== 0;
	var fallback_value = fallback;
	var fallback_dirty = true;
	var get_fallback = () => {
		if (fallback_dirty) {
			fallback_dirty = false;
			fallback_value = lazy ? untrack(fallback) : fallback;
		}
		return fallback_value;
	};
	var setter;
	if (bindable) {
		var is_entry_props = STATE_SYMBOL in props || LEGACY_PROPS in props;
		setter = get_descriptor(props, key)?.set ?? (is_entry_props && key in props ? (v) => props[key] = v : void 0);
	}
	var initial_value;
	var is_store_sub = false;
	if (bindable) [initial_value, is_store_sub] = capture_store_binding(() => props[key]);
	else initial_value = props[key];
	if (initial_value === void 0 && fallback !== void 0) {
		initial_value = get_fallback();
		if (setter) {
			if (runes) props_invalid_value(key);
			setter(initial_value);
		}
	}
	var getter;
	if (runes) getter = () => {
		var value = props[key];
		if (value === void 0) return get_fallback();
		fallback_dirty = true;
		return value;
	};
	else getter = () => {
		var value = props[key];
		if (value !== void 0) fallback_value = void 0;
		return value === void 0 ? fallback_value : value;
	};
	if (runes && (flags$1 & 4) === 0) return getter;
	if (setter) {
		var legacy_parent = props.$$legacy;
		return (function(value, mutation) {
			if (arguments.length > 0) {
				if (!runes || !mutation || legacy_parent || is_store_sub) setter(mutation ? getter() : value);
				return value;
			}
			return getter();
		});
	}
	var overridden = false;
	var d = ((flags$1 & 1) !== 0 ? derived : derived_safe_equal)(() => {
		overridden = false;
		return getter();
	});
	if (bindable) get$1(d);
	var parent_effect = active_effect;
	return (function(value, mutation) {
		if (arguments.length > 0) {
			const new_value = mutation ? get$1(d) : runes && bindable ? proxy(value) : value;
			set(d, new_value);
			overridden = true;
			if (fallback_value !== void 0) fallback_value = new_value;
			return value;
		}
		if (is_destroying_effect && overridden || (parent_effect.f & 16384) !== 0) return d.v;
		return get$1(d);
	});
}
function asClassComponent(component$1) {
	return class extends Svelte4Component {
		constructor(options) {
			super({
				component: component$1,
				...options
			});
		}
	};
}
var Svelte4Component = class {
	#events;
	#instance;
	constructor(options) {
		var sources = /* @__PURE__ */ new Map();
		var add_source = (key, value) => {
			var s = /* @__PURE__ */ mutable_source(value, false, false);
			sources.set(key, s);
			return s;
		};
		const props = new Proxy({
			...options.props || {},
			$$events: {}
		}, {
			get(target, prop$1) {
				return get$1(sources.get(prop$1) ?? add_source(prop$1, Reflect.get(target, prop$1)));
			},
			has(target, prop$1) {
				if (prop$1 === LEGACY_PROPS) return true;
				get$1(sources.get(prop$1) ?? add_source(prop$1, Reflect.get(target, prop$1)));
				return Reflect.has(target, prop$1);
			},
			set(target, prop$1, value) {
				set(sources.get(prop$1) ?? add_source(prop$1, value), value);
				return Reflect.set(target, prop$1, value);
			}
		});
		this.#instance = (options.hydrate ? hydrate : mount)(options.component, {
			target: options.target,
			anchor: options.anchor,
			props,
			context: options.context,
			intro: options.intro ?? false,
			recover: options.recover
		});
		if (!options?.props?.$$host || options.sync === false) flushSync();
		this.#events = props.$$events;
		for (const key of Object.keys(this.#instance)) {
			if (key === "$set" || key === "$destroy" || key === "$on") continue;
			define_property(this, key, {
				get() {
					return this.#instance[key];
				},
				set(value) {
					this.#instance[key] = value;
				},
				enumerable: true
			});
		}
		this.#instance.$set = (next$1) => {
			Object.assign(props, next$1);
		};
		this.#instance.$destroy = () => {
			unmount(this.#instance);
		};
	}
	$set(props) {
		this.#instance.$set(props);
	}
	$on(event$1, callback) {
		this.#events[event$1] = this.#events[event$1] || [];
		const cb = (...args) => callback.call(this, ...args);
		this.#events[event$1].push(cb);
		return () => {
			this.#events[event$1] = this.#events[event$1].filter((fn) => fn !== cb);
		};
	}
	$destroy() {
		this.#instance.$destroy();
	}
};
if (typeof HTMLElement === "function") HTMLElement;
var index_client_exports = /* @__PURE__ */ __exportAll({
	afterUpdate: () => afterUpdate,
	beforeUpdate: () => beforeUpdate,
	createContext: () => createContext,
	createEventDispatcher: () => createEventDispatcher,
	createRawSnippet: () => createRawSnippet,
	flushSync: () => flushSync,
	fork: () => fork,
	getAbortSignal: () => getAbortSignal,
	getAllContexts: () => getAllContexts,
	getContext: () => getContext,
	hasContext: () => hasContext,
	hydrate: () => hydrate,
	mount: () => mount,
	onDestroy: () => onDestroy,
	onMount: () => onMount,
	setContext: () => setContext,
	settled: () => settled,
	tick: () => tick,
	unmount: () => unmount,
	untrack: () => untrack
}, 1);
function getAbortSignal() {
	if (active_reaction === null) get_abort_signal_outside_reaction();
	return (active_reaction.ac ??= new AbortController()).signal;
}
function onMount(fn) {
	if (component_context === null) lifecycle_outside_component("onMount");
	if (legacy_mode_flag && component_context.l !== null) init_update_callbacks(component_context).m.push(fn);
	else user_effect(() => {
		const cleanup = untrack(fn);
		if (typeof cleanup === "function") return cleanup;
	});
}
function onDestroy(fn) {
	if (component_context === null) lifecycle_outside_component("onDestroy");
	onMount(() => () => untrack(fn));
}
function create_custom_event(type, detail, { bubbles = false, cancelable = false } = {}) {
	return new CustomEvent(type, {
		detail,
		bubbles,
		cancelable
	});
}
function createEventDispatcher() {
	const active_component_context = component_context;
	if (active_component_context === null) lifecycle_outside_component("createEventDispatcher");
	return (type, detail, options) => {
		const events = active_component_context.s.$$events?.[type];
		if (events) {
			const callbacks = is_array(events) ? events.slice() : [events];
			const event$1 = create_custom_event(type, detail, options);
			for (const fn of callbacks) fn.call(active_component_context.x, event$1);
			return !event$1.defaultPrevented;
		}
		return true;
	};
}
function beforeUpdate(fn) {
	if (component_context === null) lifecycle_outside_component("beforeUpdate");
	if (component_context.l === null) lifecycle_legacy_only("beforeUpdate");
	init_update_callbacks(component_context).b.push(fn);
}
function afterUpdate(fn) {
	if (component_context === null) lifecycle_outside_component("afterUpdate");
	if (component_context.l === null) lifecycle_legacy_only("afterUpdate");
	init_update_callbacks(component_context).a.push(fn);
}
function init_update_callbacks(context) {
	var l = context.l;
	return l.u ??= {
		a: [],
		b: [],
		m: []
	};
}
export { proxy as $, if_block as A, tick as B, clsx as C, __reExport as Ct, html as D, snippet as E, text as F, user_effect as G, update_version as H, head as I, $document as J, user_pre_effect as K, delegate as L, append as M, comment as N, each as O, from_html as P, sibling as Q, event as R, set_class as S, __exportAll as St, component as T, __toESM as Tt, invalidate_inner_signals as U, untrack as V, template_effect as W, child as X, $window as Y, first_child as Z, remove_input_defaults as _, reset as _t, prop as a, state as at, bind_select_value as b, false_default as bt, writable as c, getContext as ct, bind_window_size as d, setContext as dt, increment as et, bind_this as f, label as ft, attribute_effect as g, next as gt, STYLE as h, enable_legacy_mode_flag as ht, asClassComponent as i, source as it, set_text as j, index as k, reactive_import as l, pop as lt, bind_value as m, snapshot as mt, onDestroy as n, mutate as nt, setup_stores as o, update as ot, bind_checked as p, tag as pt, remove_textarea_child as q, onMount as r, set as rt, store_get as s, user_derived as st, index_client_exports as t, mutable_source as tt, init as u, push as ut, set_attribute as v, noop as vt, action as w, __require as wt, set_style as x, __commonJSMin as xt, set_value as y, to_array as yt, get$1 as z };
