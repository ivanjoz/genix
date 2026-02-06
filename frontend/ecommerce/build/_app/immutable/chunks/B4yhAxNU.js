import { n as __exportAll, r as __toESM, t as __commonJSMin } from "./hMJtJoiB.js";
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
const noop$1 = () => {};
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
const MANAGED_EFFECT = 1 << 24;
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
const EFFECT_OFFSCREEN = 1 << 25;
const WAS_MARKED = 32768;
const REACTION_IS_UPDATING = 1 << 21;
const ASYNC = 1 << 22;
const ERROR_VALUE = 1 << 23;
const STATE_SYMBOL = Symbol("$state");
const LEGACY_PROPS = Symbol("legacy props");
const LOADING_ATTR_SYMBOL = Symbol("");
const STALE_REACTION = new class StaleReactionError extends Error {
	name = "StaleReactionError";
	message = "The reaction that called `getAbortSignal()` was re-run or destroyed";
}();
function experimental_async_required(name) {
	throw new Error(`https://svelte.dev/e/experimental_async_required`);
}
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
function hydratable_missing_but_expected(key) {
	console.warn(`https://svelte.dev/e/hydratable_missing_but_expected`);
}
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
		var next = /* @__PURE__ */ get_next_sibling(node);
		if (remove) node.remove();
		node = next;
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
function pop(component) {
	var context = component_context;
	var effects = context.e;
	if (effects !== null) {
		context.e = null;
		for (var fn of effects) create_user_effect(fn);
	}
	if (component !== void 0) context.x = component;
	context.i = true;
	component_context = context.p;
	return component ?? {};
}
function is_runes() {
	return !legacy_mode_flag || component_context !== null && component_context.l === null;
}
function get_or_init_context_map(name) {
	if (component_context === null) lifecycle_outside_component(name);
	return component_context.c ??= new Map(get_parent_context(component_context) || void 0);
}
function get_parent_context(component_context) {
	let parent = component_context.p;
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
function handle_error$1(error) {
	var effect = active_effect;
	if (effect === null) {
		active_reaction.f |= ERROR_VALUE;
		return error;
	}
	if ((effect.f & 32768) === 0) {
		if ((effect.f & 128) === 0) throw error;
		effect.b.error(error);
	} else invoke_error_boundary(error, effect);
}
function invoke_error_boundary(error, effect) {
	while (effect !== null) {
		if ((effect.f & 128) !== 0) try {
			effect.b.error(error);
			return;
		} catch (e) {
			error = e;
		}
		effect = effect.parent;
	}
	throw error;
}
var STATUS_MASK = ~(MAYBE_DIRTY | 3072);
function set_signal_status(signal, status) {
	signal.f = signal.f & STATUS_MASK | status;
}
function update_derived_status(derived) {
	if ((derived.f & 512) !== 0 || derived.deps === null) set_signal_status(derived, CLEAN);
	else set_signal_status(derived, MAYBE_DIRTY);
}
function clear_marked(deps) {
	if (deps === null) return;
	for (const dep of deps) {
		if ((dep.f & 2) === 0 || (dep.f & 32768) === 0) continue;
		dep.f ^= WAS_MARKED;
		clear_marked(dep.deps);
	}
}
function defer_effect(effect, dirty_effects, maybe_dirty_effects) {
	if ((effect.f & 2048) !== 0) dirty_effects.add(effect);
	else if ((effect.f & 4096) !== 0) maybe_dirty_effects.add(effect);
	clear_marked(effect.deps);
	set_signal_status(effect, CLEAN);
}
var batches = /* @__PURE__ */ new Set();
let current_batch = null;
let previous_batch = null;
let batch_values = null;
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
	#dirty_effects = /* @__PURE__ */ new Set();
	#maybe_dirty_effects = /* @__PURE__ */ new Set();
	skipped_effects = /* @__PURE__ */ new Set();
	is_fork = false;
	#decrement_queued = false;
	is_deferred() {
		return this.is_fork || this.#blocking_pending > 0;
	}
	process(root_effects) {
		queued_root_effects = [];
		this.apply();
		var effects = [];
		var render_effects = [];
		for (const root of root_effects) this.#traverse_effect_tree(root, effects, render_effects);
		if (this.is_deferred()) {
			this.#defer_effects(render_effects);
			this.#defer_effects(effects);
		} else {
			for (const fn of this.#commit_callbacks) fn();
			this.#commit_callbacks.clear();
			if (this.#pending === 0) this.#commit();
			previous_batch = this;
			current_batch = null;
			flush_queued_effects(render_effects);
			flush_queued_effects(effects);
			previous_batch = null;
			this.#deferred?.resolve();
		}
		batch_values = null;
	}
	#traverse_effect_tree(root, effects, render_effects) {
		root.f ^= CLEAN;
		var effect = root.first;
		var pending_boundary = null;
		while (effect !== null) {
			var flags = effect.f;
			var is_branch = (flags & 96) !== 0;
			if (!(is_branch && (flags & 1024) !== 0 || (flags & 8192) !== 0 || this.skipped_effects.has(effect)) && effect.fn !== null) {
				if (is_branch) effect.f ^= CLEAN;
				else if (pending_boundary !== null && (flags & 16777228) !== 0) pending_boundary.b.defer_effect(effect);
				else if ((flags & 4) !== 0) effects.push(effect);
				else if (is_dirty(effect)) {
					if ((flags & 16) !== 0) this.#dirty_effects.add(effect);
					update_effect(effect);
				}
				var child = effect.first;
				if (child !== null) {
					effect = child;
					continue;
				}
			}
			var parent = effect.parent;
			effect = effect.next;
			while (effect === null && parent !== null) {
				if (parent === pending_boundary) pending_boundary = null;
				effect = parent.next;
				parent = parent.parent;
			}
		}
	}
	#defer_effects(effects) {
		for (var i = 0; i < effects.length; i += 1) defer_effect(effects[i], this.#dirty_effects, this.#maybe_dirty_effects);
	}
	capture(source, value) {
		if (value !== UNINITIALIZED && !this.previous.has(source)) this.previous.set(source, value);
		if ((source.f & 8388608) === 0) {
			this.current.set(source, source.v);
			batch_values?.set(source, source.v);
		}
	}
	activate() {
		current_batch = this;
		this.apply();
	}
	deactivate() {
		if (current_batch !== this) return;
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
	}
	discard() {
		for (const fn of this.#discard_callbacks) fn(this);
		this.#discard_callbacks.clear();
	}
	#commit() {
		if (batches.size > 1) {
			this.previous.clear();
			var previous_batch_values = batch_values;
			var is_earlier = true;
			for (const batch of batches) {
				if (batch === this) {
					is_earlier = false;
					continue;
				}
				const sources = [];
				for (const [source, value] of this.current) {
					if (batch.current.has(source)) if (is_earlier && value !== batch.current.get(source)) batch.current.set(source, value);
					else continue;
					sources.push(source);
				}
				if (sources.length === 0) continue;
				const others = [...batch.current.keys()].filter((s) => !this.current.has(s));
				if (others.length > 0) {
					var prev_queued_root_effects = queued_root_effects;
					queued_root_effects = [];
					const marked = /* @__PURE__ */ new Set();
					const checked = /* @__PURE__ */ new Map();
					for (const source of sources) mark_effects(source, others, marked, checked);
					if (queued_root_effects.length > 0) {
						current_batch = batch;
						batch.apply();
						for (const root of queued_root_effects) batch.#traverse_effect_tree(root, [], []);
						batch.deactivate();
					}
					queued_root_effects = prev_queued_root_effects;
				}
			}
			current_batch = null;
			batch_values = previous_batch_values;
		}
		this.committed = true;
		batches.delete(this);
	}
	increment(blocking) {
		this.#pending += 1;
		if (blocking) this.#blocking_pending += 1;
	}
	decrement(blocking) {
		this.#pending -= 1;
		if (blocking) this.#blocking_pending -= 1;
		if (this.#decrement_queued) return;
		this.#decrement_queued = true;
		queue_micro_task(() => {
			this.#decrement_queued = false;
			if (!this.is_deferred()) this.revive();
			else if (queued_root_effects.length > 0) this.flush();
		});
	}
	revive() {
		for (const e of this.#dirty_effects) {
			this.#maybe_dirty_effects.delete(e);
			set_signal_status(e, DIRTY);
			schedule_effect(e);
		}
		for (const e of this.#maybe_dirty_effects) {
			set_signal_status(e, MAYBE_DIRTY);
			schedule_effect(e);
		}
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
			if (!is_flushing_sync) queue_micro_task(() => {
				if (current_batch !== batch) return;
				batch.flush();
			});
		}
		return current_batch;
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
	is_flushing = true;
	try {
		var flush_count = 0;
		while (queued_root_effects.length > 0) {
			var batch = Batch.ensure();
			if (flush_count++ > 1e3) infinite_loop_guard();
			batch.process(queued_root_effects);
			old_values.clear();
		}
	} finally {
		is_flushing = false;
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
		var effect = effects[i++];
		if ((effect.f & 24576) === 0 && is_dirty(effect)) {
			eager_block_effects = /* @__PURE__ */ new Set();
			update_effect(effect);
			if (effect.deps === null && effect.first === null && effect.nodes === null) if (effect.teardown === null && effect.ac === null) unlink_effect(effect);
			else effect.fn = null;
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
						const e = ordered_effects[j];
						if ((e.f & 24576) !== 0) continue;
						update_effect(e);
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
		const flags = reaction.f;
		if ((flags & 2) !== 0) mark_effects(reaction, sources, marked, checked);
		else if ((flags & 4194320) !== 0 && (flags & 2048) === 0 && depends_on(reaction, sources, checked)) {
			set_signal_status(reaction, DIRTY);
			schedule_effect(reaction);
		}
	}
}
function mark_eager_effects(value, effects) {
	if (value.reactions === null) return;
	for (const reaction of value.reactions) {
		const flags = reaction.f;
		if ((flags & 2) !== 0) mark_eager_effects(reaction, effects);
		else if ((flags & 131072) !== 0) {
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
	var effect = last_scheduled_effect = signal;
	while (effect.parent !== null) {
		effect = effect.parent;
		var flags = effect.f;
		if (is_flushing && effect === active_effect && (flags & 16) !== 0 && (flags & 262144) === 0) return;
		if ((flags & 96) !== 0) {
			if ((flags & 1024) === 0) return;
			effect.f ^= CLEAN;
		}
	}
	queued_root_effects.push(effect);
}
function fork(fn) {
	experimental_async_required("fork");
	if (current_batch !== null) fork_timing();
	var batch = Batch.ensure();
	batch.is_fork = true;
	batch_values = /* @__PURE__ */ new Map();
	var committed = false;
	var settled = batch.settled();
	flushSync(fn);
	for (var [source, value] of batch.previous) source.v = value;
	for (source of batch.current.keys()) if ((source.f & 2) !== 0) set_signal_status(source, DIRTY);
	return {
		commit: async () => {
			if (committed) {
				await settled;
				return;
			}
			if (!batches.has(batch)) fork_discarded();
			committed = true;
			batch.is_fork = false;
			for (var [source, value] of batch.current) {
				source.v = value;
				source.wv = increment_write_version();
			}
			flushSync(() => {
				var eager_effects = /* @__PURE__ */ new Set();
				for (var source of batch.current.keys()) mark_eager_effects(source, eager_effects);
				set_eager_effects(eager_effects);
				flush_eager_effects();
			});
			batch.revive();
			await settled;
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
				if (subscribers === 0) stop = untrack$1(() => start(() => increment(version)));
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
	is_pending = false;
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
	#pending_count_update_queued = false;
	#is_creating_fallback = false;
	#dirty_effects = /* @__PURE__ */ new Set();
	#maybe_dirty_effects = /* @__PURE__ */ new Set();
	#effect_pending = null;
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
		this.is_pending = !!this.#props.pending;
		this.#effect = block(() => {
			active_effect.b = this;
			if (hydrating) {
				const comment = this.#hydrate_open;
				hydrate_next();
				if (comment.nodeType === 8 && comment.data === "[!") this.#hydrate_pending_content();
				else {
					this.#hydrate_resolved_content();
					if (this.#pending_count === 0) this.is_pending = false;
				}
			} else {
				var anchor = this.#get_anchor();
				try {
					this.#main_effect = branch(() => children(anchor));
				} catch (error) {
					this.error(error);
				}
				if (this.#pending_count > 0) this.#show_pending_snippet();
				else this.is_pending = false;
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
	}
	#hydrate_pending_content() {
		const pending = this.#props.pending;
		if (!pending) return;
		this.#pending_effect = branch(() => pending(this.#anchor));
		queue_micro_task(() => {
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
				this.is_pending = false;
			}
		});
	}
	#get_anchor() {
		var anchor = this.#anchor;
		if (this.is_pending) {
			this.#pending_anchor = create_text();
			this.#anchor.before(this.#pending_anchor);
			anchor = this.#pending_anchor;
		}
		return anchor;
	}
	defer_effect(effect) {
		defer_effect(effect, this.#dirty_effects, this.#maybe_dirty_effects);
	}
	is_rendered() {
		return !this.is_pending && (!this.parent || this.parent.is_rendered());
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
			handle_error$1(e);
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
			this.is_pending = false;
			for (const e of this.#dirty_effects) {
				set_signal_status(e, DIRTY);
				schedule_effect(e);
			}
			for (const e of this.#maybe_dirty_effects) {
				set_signal_status(e, MAYBE_DIRTY);
				schedule_effect(e);
			}
			this.#dirty_effects.clear();
			this.#maybe_dirty_effects.clear();
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
		if (!this.#effect_pending || this.#pending_count_update_queued) return;
		this.#pending_count_update_queued = true;
		queue_micro_task(() => {
			this.#pending_count_update_queued = false;
			if (this.#effect_pending) internal_set(this.#effect_pending, this.#local_pending_count);
		});
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
		const reset = () => {
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
			this.is_pending = this.has_pending_snippet();
			this.#main_effect = this.#run(() => {
				this.#is_creating_fallback = false;
				return branch(() => this.#children(this.#anchor));
			});
			if (this.#pending_count > 0) this.#show_pending_snippet();
			else this.is_pending = false;
		};
		var previous_reaction = active_reaction;
		try {
			set_active_reaction(null);
			calling_on_error = true;
			onerror?.(error, reset);
			calling_on_error = false;
		} catch (error) {
			invoke_error_boundary(error, this.#effect && this.#effect.parent);
		} finally {
			set_active_reaction(previous_reaction);
		}
		if (failed) queue_micro_task(() => {
			this.#failed_effect = this.#run(() => {
				Batch.ensure();
				this.#is_creating_fallback = true;
				try {
					return branch(() => {
						failed(this.#anchor, () => error, () => reset);
					});
				} catch (error) {
					invoke_error_boundary(error, this.#effect.parent);
					return null;
				} finally {
					this.#is_creating_fallback = false;
				}
			});
		});
	}
};
function flatten(blockers, sync, async, fn) {
	const d = is_runes() ? derived : derived_safe_equal;
	var pending = blockers.filter((b) => !b.settled);
	if (async.length === 0 && pending.length === 0) {
		fn(sync.map(d));
		return;
	}
	var batch = current_batch;
	var parent = active_effect;
	var restore = capture();
	var blocker_promise = pending.length === 1 ? pending[0].promise : pending.length > 1 ? Promise.all(pending.map((b) => b.promise)) : null;
	function finish(values) {
		restore();
		try {
			fn(values);
		} catch (error) {
			if ((parent.f & 16384) === 0) invoke_error_boundary(error, parent);
		}
		batch?.deactivate();
		unset_context();
	}
	if (async.length === 0) {
		blocker_promise.then(() => finish(sync.map(d)));
		return;
	}
	function run() {
		restore();
		Promise.all(async.map((expression) => /* @__PURE__ */ async_derived(expression))).then((result) => finish([...sync.map(d), ...result])).catch((error) => invoke_error_boundary(error, parent));
	}
	if (blocker_promise) blocker_promise.then(run);
	else run();
}
function capture() {
	var previous_effect = active_effect;
	var previous_reaction = active_reaction;
	var previous_component_context = component_context;
	var previous_batch = current_batch;
	return function restore(activate_batch = true) {
		set_active_effect(previous_effect);
		set_active_reaction(previous_reaction);
		set_component_context(previous_component_context);
		if (activate_batch) previous_batch?.activate();
	};
}
function unset_context() {
	set_active_effect(null);
	set_active_reaction(null);
	set_component_context(null);
}
/* @__NO_SIDE_EFFECTS__ */
function derived(fn) {
	var flags = 2 | DIRTY;
	var parent_derived = active_reaction !== null && (active_reaction.f & 2) !== 0 ? active_reaction : null;
	if (active_effect !== null) active_effect.f |= EFFECT_PRESERVED;
	return {
		ctx: component_context,
		deps: null,
		effects: null,
		equals,
		f: flags,
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
function async_derived(fn, label, location) {
	let parent = active_effect;
	if (parent === null) async_derived_orphan();
	var boundary = parent.b;
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
			var blocking = boundary.is_rendered();
			boundary.update_pending_count(1);
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
				for (const [b, d] of deferreds) {
					deferreds.delete(b);
					if (b === batch) break;
					d.reject(STALE_REACTION);
				}
			}
			if (should_suspend) {
				boundary.update_pending_count(-1);
				batch.decrement(blocking);
			}
		};
		d.promise.then(handler, (e) => handler(null, e || "unknown"));
	});
	teardown(() => {
		for (const d of deferreds.values()) d.reject(STALE_REACTION);
	});
	return new Promise((fulfil) => {
		function next(p) {
			function go() {
				if (p === promise) fulfil(signal);
				else next(promise);
			}
			p.then(go, go);
		}
		next(promise);
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
function destroy_derived_effects(derived) {
	var effects = derived.effects;
	if (effects !== null) {
		derived.effects = null;
		for (var i = 0; i < effects.length; i += 1) destroy_effect(effects[i]);
	}
}
function get_derived_parent_effect(derived) {
	var parent = derived.parent;
	while (parent !== null) {
		if ((parent.f & 2) === 0) return (parent.f & 16384) === 0 ? parent : null;
		parent = parent.parent;
	}
	return null;
}
function execute_derived(derived) {
	var value;
	var prev_active_effect = active_effect;
	set_active_effect(get_derived_parent_effect(derived));
	try {
		derived.f &= ~WAS_MARKED;
		destroy_derived_effects(derived);
		value = update_reaction(derived);
	} finally {
		set_active_effect(prev_active_effect);
	}
	return value;
}
function update_derived(derived) {
	var value = execute_derived(derived);
	if (!derived.equals(value)) {
		derived.wv = increment_write_version();
		if (!current_batch?.is_fork || derived.deps === null) {
			derived.v = value;
			if (derived.deps === null) {
				set_signal_status(derived, CLEAN);
				return;
			}
		}
	}
	if (is_destroying_effect) return;
	if (batch_values !== null) {
		if (effect_tracking() || current_batch?.is_fork) batch_values.set(derived, value);
	} else update_derived_status(derived);
}
let eager_effects = /* @__PURE__ */ new Set();
const old_values = /* @__PURE__ */ new Map();
function set_eager_effects(v) {
	eager_effects = v;
}
var eager_effects_deferred = false;
function source(v, stack) {
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
function state(v, stack) {
	const s = source(v, stack);
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
function set$1(source, value, should_proxy = false) {
	if (active_reaction !== null && (!untracking || (active_reaction.f & 131072) !== 0) && is_runes() && (active_reaction.f & 4325394) !== 0 && !current_sources?.includes(source)) state_unsafe_mutation();
	return internal_set(source, should_proxy ? proxy(value) : value);
}
function internal_set(source, value) {
	if (!source.equals(value)) {
		var old_value = source.v;
		if (is_destroying_effect) old_values.set(source, value);
		else old_values.set(source, old_value);
		source.v = value;
		var batch = Batch.ensure();
		batch.capture(source, old_value);
		if ((source.f & 2) !== 0) {
			const derived = source;
			if ((source.f & 2048) !== 0) execute_derived(derived);
			update_derived_status(derived);
		}
		source.wv = increment_write_version();
		mark_reactions(source, DIRTY);
		if (is_runes() && active_effect !== null && (active_effect.f & 1024) !== 0 && (active_effect.f & 96) === 0) if (untracked_writes === null) set_untracked_writes([source]);
		else untracked_writes.push(source);
		if (!batch.is_fork && eager_effects.size > 0 && !eager_effects_deferred) flush_eager_effects();
	}
	return value;
}
function flush_eager_effects() {
	eager_effects_deferred = false;
	for (const effect of eager_effects) {
		if ((effect.f & 1024) !== 0) set_signal_status(effect, MAYBE_DIRTY);
		if (is_dirty(effect)) update_effect(effect);
	}
	eager_effects.clear();
}
function increment(source) {
	set$1(source, source.v + 1);
}
function mark_reactions(signal, status) {
	var reactions = signal.reactions;
	if (reactions === null) return;
	var runes = is_runes();
	var length = reactions.length;
	for (var i = 0; i < length; i++) {
		var reaction = reactions[i];
		var flags = reaction.f;
		if (!runes && reaction === active_effect) continue;
		var not_dirty = (flags & DIRTY) === 0;
		if (not_dirty) set_signal_status(reaction, status);
		if ((flags & 2) !== 0) {
			var derived = reaction;
			batch_values?.delete(derived);
			if ((flags & 32768) === 0) {
				if (flags & 512) reaction.f |= WAS_MARKED;
				mark_reactions(derived, MAYBE_DIRTY);
			}
		} else if (not_dirty) {
			if ((flags & 16) !== 0 && eager_block_effects !== null) eager_block_effects.add(reaction);
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
	var stack = null;
	var parent_version = update_version;
	var with_parent = (fn) => {
		if (update_version === parent_version) return fn();
		var reaction = active_reaction;
		var version = update_version;
		set_active_reaction(null);
		set_update_version(parent_version);
		var result = fn();
		set_active_reaction(reaction);
		set_update_version(version);
		return result;
	};
	if (is_proxied_array) sources.set("length", /* @__PURE__ */ state(value.length, stack));
	return new Proxy(value, {
		defineProperty(_, prop, descriptor) {
			if (!("value" in descriptor) || descriptor.configurable === false || descriptor.enumerable === false || descriptor.writable === false) state_descriptors_fixed();
			var s = sources.get(prop);
			if (s === void 0) s = with_parent(() => {
				var s = /* @__PURE__ */ state(descriptor.value, stack);
				sources.set(prop, s);
				return s;
			});
			else set$1(s, descriptor.value, true);
			return true;
		},
		deleteProperty(target, prop) {
			var s = sources.get(prop);
			if (s === void 0) {
				if (prop in target) {
					const s = with_parent(() => /* @__PURE__ */ state(UNINITIALIZED, stack));
					sources.set(prop, s);
					increment(version);
				}
			} else {
				set$1(s, UNINITIALIZED);
				increment(version);
			}
			return true;
		},
		get(target, prop, receiver) {
			if (prop === STATE_SYMBOL) return value;
			var s = sources.get(prop);
			var exists = prop in target;
			if (s === void 0 && (!exists || get_descriptor(target, prop)?.writable)) {
				s = with_parent(() => {
					return /* @__PURE__ */ state(proxy(exists ? target[prop] : UNINITIALIZED), stack);
				});
				sources.set(prop, s);
			}
			if (s !== void 0) {
				var v = get$1(s);
				return v === UNINITIALIZED ? void 0 : v;
			}
			return Reflect.get(target, prop, receiver);
		},
		getOwnPropertyDescriptor(target, prop) {
			var descriptor = Reflect.getOwnPropertyDescriptor(target, prop);
			if (descriptor && "value" in descriptor) {
				var s = sources.get(prop);
				if (s) descriptor.value = get$1(s);
			} else if (descriptor === void 0) {
				var source = sources.get(prop);
				var value = source?.v;
				if (source !== void 0 && value !== UNINITIALIZED) return {
					enumerable: true,
					configurable: true,
					value,
					writable: true
				};
			}
			return descriptor;
		},
		has(target, prop) {
			if (prop === STATE_SYMBOL) return true;
			var s = sources.get(prop);
			var has = s !== void 0 && s.v !== UNINITIALIZED || Reflect.has(target, prop);
			if (s !== void 0 || active_effect !== null && (!has || get_descriptor(target, prop)?.writable)) {
				if (s === void 0) {
					s = with_parent(() => {
						return /* @__PURE__ */ state(has ? proxy(target[prop]) : UNINITIALIZED, stack);
					});
					sources.set(prop, s);
				}
				if (get$1(s) === UNINITIALIZED) return false;
			}
			return has;
		},
		set(target, prop, value, receiver) {
			var s = sources.get(prop);
			var has = prop in target;
			if (is_proxied_array && prop === "length") for (var i = value; i < s.v; i += 1) {
				var other_s = sources.get(i + "");
				if (other_s !== void 0) set$1(other_s, UNINITIALIZED);
				else if (i in target) {
					other_s = with_parent(() => /* @__PURE__ */ state(UNINITIALIZED, stack));
					sources.set(i + "", other_s);
				}
			}
			if (s === void 0) {
				if (!has || get_descriptor(target, prop)?.writable) {
					s = with_parent(() => /* @__PURE__ */ state(void 0, stack));
					set$1(s, proxy(value));
					sources.set(prop, s);
				}
			} else {
				has = s.v !== UNINITIALIZED;
				var p = with_parent(() => proxy(value));
				set$1(s, p);
			}
			var descriptor = Reflect.getOwnPropertyDescriptor(target, prop);
			if (descriptor?.set) descriptor.set.call(receiver, value);
			if (!has) {
				if (is_proxied_array && typeof prop === "string") {
					var ls = sources.get("length");
					var n = Number(prop);
					if (Number.isInteger(n) && n >= ls.v) set$1(ls, n + 1);
				}
				increment(version);
			}
			return true;
		},
		ownKeys(target) {
			get$1(version);
			var own_keys = Reflect.ownKeys(target).filter((key) => {
				var source = sources.get(key);
				return source === void 0 || source.v !== UNINITIALIZED;
			});
			for (var [key, source] of sources) if (source.v !== UNINITIALIZED && !(key in target)) own_keys.push(key);
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
var is_firefox;
var first_child_getter;
var next_sibling_getter;
function init_operations() {
	if ($window !== void 0) return;
	$window = window;
	document;
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
	var child = /* @__PURE__ */ get_first_child(hydrate_node);
	if (child === null) child = hydrate_node.appendChild(create_text());
	else if (is_text && child.nodeType !== 3) {
		var text = create_text();
		child?.before(text);
		set_hydrate_node(text);
		return text;
	}
	set_hydrate_node(child);
	return child;
}
function first_child(node, is_text = false) {
	if (!hydrating) {
		var first = /* @__PURE__ */ get_first_child(node);
		if (first instanceof Comment && first.data === "") return /* @__PURE__ */ get_next_sibling(first);
		return first;
	}
	if (is_text && hydrate_node?.nodeType !== 3) {
		var text = create_text();
		hydrate_node?.before(text);
		set_hydrate_node(text);
		return text;
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
		var text = create_text();
		if (next_sibling === null) last_sibling?.after(text);
		else next_sibling.before(text);
		set_hydrate_node(text);
		return text;
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
function listen_to_event_and_reset_event(element, event, handler, on_reset = handler) {
	element.addEventListener(event, () => without_reactive_context(handler));
	const prev = element.__on_r;
	if (prev) element.__on_r = () => {
		prev();
		on_reset(true);
	};
	else element.__on_r = () => on_reset(true);
	add_form_reset_listener();
}
function validate_effect(rune) {
	if (active_effect === null) {
		if (active_reaction === null) effect_orphan(rune);
		effect_in_unowned_derived();
	}
	if (is_destroying_effect) effect_in_teardown(rune);
}
function push_effect(effect, parent_effect) {
	var parent_last = parent_effect.last;
	if (parent_last === null) parent_effect.last = parent_effect.first = effect;
	else {
		parent_last.next = effect;
		effect.prev = parent_last;
		parent_effect.last = effect;
	}
}
function create_effect(type, fn, sync) {
	var parent = active_effect;
	if (parent !== null && (parent.f & 8192) !== 0) type |= INERT;
	var effect = {
		ctx: component_context,
		deps: null,
		nodes: null,
		f: type | 2560,
		first: null,
		fn,
		last: null,
		next: null,
		parent,
		b: parent && parent.b,
		prev: null,
		teardown: null,
		wv: 0,
		ac: null
	};
	if (sync) try {
		update_effect(effect);
		effect.f |= EFFECT_RAN;
	} catch (e) {
		destroy_effect(effect);
		throw e;
	}
	else if (fn !== null) schedule_effect(effect);
	var e = effect;
	if (sync && e.deps === null && e.teardown === null && e.nodes === null && e.first === e.last && (e.f & 524288) === 0) {
		e = e.first;
		if ((type & 16) !== 0 && (type & 65536) !== 0 && e !== null) e.f |= EFFECT_TRANSPARENT;
	}
	if (e !== null) {
		e.parent = parent;
		if (parent !== null) push_effect(e, parent);
		if (active_reaction !== null && (active_reaction.f & 2) !== 0 && (type & 64) === 0) {
			var derived = active_reaction;
			(derived.effects ??= []).push(e);
		}
	}
	return effect;
}
function effect_tracking() {
	return active_reaction !== null && !untracking;
}
function teardown(fn) {
	const effect = create_effect(8, null, false);
	set_signal_status(effect, CLEAN);
	effect.teardown = fn;
	return effect;
}
function user_effect(fn) {
	validate_effect("$effect");
	var flags = active_effect.f;
	if (!active_reaction && (flags & 32) !== 0 && (flags & 32768) === 0) {
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
	const effect = create_effect(64 | EFFECT_PRESERVED, fn, true);
	return (options = {}) => {
		return new Promise((fulfil) => {
			if (options.outro) pause_effect(effect, () => {
				destroy_effect(effect);
				fulfil(void 0);
			});
			else {
				destroy_effect(effect);
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
function render_effect(fn, flags = 0) {
	return create_effect(8 | flags, fn, true);
}
function template_effect(fn, sync = [], async = [], blockers = []) {
	flatten(blockers, sync, async, (values) => {
		create_effect(8, () => fn(...values.map(get$1)), true);
	});
}
function block(fn, flags = 0) {
	return create_effect(16 | flags, fn, true);
}
function managed(fn, flags = 0) {
	return create_effect(MANAGED_EFFECT | flags, fn, true);
}
function branch(fn) {
	return create_effect(32 | EFFECT_PRESERVED, fn, true);
}
function execute_effect_teardown(effect) {
	var teardown = effect.teardown;
	if (teardown !== null) {
		const previously_destroying_effect = is_destroying_effect;
		const previous_reaction = active_reaction;
		set_is_destroying_effect(true);
		set_active_reaction(null);
		try {
			teardown.call(null);
		} finally {
			set_is_destroying_effect(previously_destroying_effect);
			set_active_reaction(previous_reaction);
		}
	}
}
function destroy_effect_children(signal, remove_dom = false) {
	var effect = signal.first;
	signal.first = signal.last = null;
	while (effect !== null) {
		const controller = effect.ac;
		if (controller !== null) without_reactive_context(() => {
			controller.abort(STALE_REACTION);
		});
		var next = effect.next;
		if ((effect.f & 64) !== 0) effect.parent = null;
		else destroy_effect(effect, remove_dom);
		effect = next;
	}
}
function destroy_block_effect_children(signal) {
	var effect = signal.first;
	while (effect !== null) {
		var next = effect.next;
		if ((effect.f & 32) === 0) destroy_effect(effect);
		effect = next;
	}
}
function destroy_effect(effect, remove_dom = true) {
	var removed = false;
	if ((remove_dom || (effect.f & 262144) !== 0) && effect.nodes !== null && effect.nodes.end !== null) {
		remove_effect_dom(effect.nodes.start, effect.nodes.end);
		removed = true;
	}
	destroy_effect_children(effect, remove_dom && !removed);
	remove_reactions(effect, 0);
	set_signal_status(effect, DESTROYED);
	var transitions = effect.nodes && effect.nodes.t;
	if (transitions !== null) for (const transition of transitions) transition.stop();
	execute_effect_teardown(effect);
	var parent = effect.parent;
	if (parent !== null && parent.first !== null) unlink_effect(effect);
	effect.next = effect.prev = effect.teardown = effect.ctx = effect.deps = effect.fn = effect.nodes = effect.ac = null;
}
function remove_effect_dom(node, end) {
	while (node !== null) {
		var next = node === end ? null : /* @__PURE__ */ get_next_sibling(node);
		node.remove();
		node = next;
	}
}
function unlink_effect(effect) {
	var parent = effect.parent;
	var prev = effect.prev;
	var next = effect.next;
	if (prev !== null) prev.next = next;
	if (next !== null) next.prev = prev;
	if (parent !== null) {
		if (parent.first === effect) parent.first = next;
		if (parent.last === effect) parent.last = prev;
	}
}
function pause_effect(effect, callback, destroy = true) {
	var transitions = [];
	pause_children(effect, transitions, true);
	var fn = () => {
		if (destroy) destroy_effect(effect);
		if (callback) callback();
	};
	var remaining = transitions.length;
	if (remaining > 0) {
		var check = () => --remaining || fn();
		for (var transition of transitions) transition.out(check);
	} else fn();
}
function pause_children(effect, transitions, local) {
	if ((effect.f & 8192) !== 0) return;
	effect.f ^= INERT;
	var t = effect.nodes && effect.nodes.t;
	if (t !== null) {
		for (const transition of t) if (transition.is_global || local) transitions.push(transition);
	}
	var child = effect.first;
	while (child !== null) {
		var sibling = child.next;
		var transparent = (child.f & 65536) !== 0 || (child.f & 32) !== 0 && (effect.f & 16) !== 0;
		pause_children(child, transitions, transparent ? local : false);
		child = sibling;
	}
}
function resume_effect(effect) {
	resume_children(effect, true);
}
function resume_children(effect, local) {
	if ((effect.f & 8192) === 0) return;
	effect.f ^= INERT;
	if ((effect.f & 1024) === 0) {
		set_signal_status(effect, DIRTY);
		schedule_effect(effect);
	}
	var child = effect.first;
	while (child !== null) {
		var sibling = child.next;
		var transparent = (child.f & 65536) !== 0 || (child.f & 32) !== 0;
		resume_children(child, transparent ? local : false);
		child = sibling;
	}
	var t = effect.nodes && effect.nodes.t;
	if (t !== null) {
		for (const transition of t) if (transition.is_global || local) transition.in();
	}
}
function move_effect(effect, fragment) {
	if (!effect.nodes) return;
	var node = effect.nodes.start;
	var end = effect.nodes.end;
	while (node !== null) {
		var next = node === end ? null : /* @__PURE__ */ get_next_sibling(node);
		fragment.append(node);
		node = next;
	}
}
let captured_signals = null;
var is_updating_effect = false;
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
function set_active_effect(effect) {
	active_effect = effect;
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
function increment_write_version() {
	return ++write_version;
}
function is_dirty(reaction) {
	var flags = reaction.f;
	if ((flags & 2048) !== 0) return true;
	if (flags & 2) reaction.f &= ~WAS_MARKED;
	if ((flags & 4096) !== 0) {
		var dependencies = reaction.deps;
		var length = dependencies.length;
		for (var i = 0; i < length; i++) {
			var dependency = dependencies[i];
			if (is_dirty(dependency)) update_derived(dependency);
			if (dependency.wv > reaction.wv) return true;
		}
		if ((flags & 512) !== 0 && batch_values === null) set_signal_status(reaction, CLEAN);
	}
	return false;
}
function schedule_possible_effect_self_invalidation(signal, effect, root = true) {
	var reactions = signal.reactions;
	if (reactions === null) return;
	if (current_sources?.includes(signal)) return;
	for (var i = 0; i < reactions.length; i++) {
		var reaction = reactions[i];
		if ((reaction.f & 2) !== 0) schedule_possible_effect_self_invalidation(reaction, effect, false);
		else if (effect === reaction) {
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
	var previous_sources = current_sources;
	var previous_component_context = component_context;
	var previous_untracking = untracking;
	var previous_update_version = update_version;
	var flags = reaction.f;
	new_deps = null;
	skipped_deps = 0;
	untracked_writes = null;
	active_reaction = (flags & 96) === 0 ? reaction : null;
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
			if (effect_tracking() && (reaction.f & 512) !== 0) for (i = skipped_deps; i < deps.length; i++) (deps[i].reactions ??= []).push(reaction);
		} else if (deps !== null && skipped_deps < deps.length) {
			remove_reactions(reaction, skipped_deps);
			deps.length = skipped_deps;
		}
		if (is_runes() && untracked_writes !== null && !untracking && deps !== null && (reaction.f & 6146) === 0) for (i = 0; i < untracked_writes.length; i++) schedule_possible_effect_self_invalidation(untracked_writes[i], reaction);
		if (previous_reaction !== null && previous_reaction !== reaction) {
			read_version++;
			if (previous_reaction.deps !== null) for (let i = 0; i < previous_skipped_deps; i += 1) previous_reaction.deps[i].rv = read_version;
			if (previous_deps !== null) for (const dep of previous_deps) dep.rv = read_version;
			if (untracked_writes !== null) if (previous_untracked_writes === null) previous_untracked_writes = untracked_writes;
			else previous_untracked_writes.push(...untracked_writes);
		}
		if ((reaction.f & 8388608) !== 0) reaction.f ^= ERROR_VALUE;
		return result;
	} catch (error) {
		return handle_error$1(error);
	} finally {
		reaction.f ^= REACTION_IS_UPDATING;
		new_deps = previous_deps;
		skipped_deps = previous_skipped_deps;
		untracked_writes = previous_untracked_writes;
		active_reaction = previous_reaction;
		current_sources = previous_sources;
		set_component_context(previous_component_context);
		untracking = previous_untracking;
		update_version = previous_update_version;
	}
}
function remove_reaction(signal, dependency) {
	let reactions = dependency.reactions;
	if (reactions !== null) {
		var index = index_of.call(reactions, signal);
		if (index !== -1) {
			var new_length = reactions.length - 1;
			if (new_length === 0) reactions = dependency.reactions = null;
			else {
				reactions[index] = reactions[new_length];
				reactions.pop();
			}
		}
	}
	if (reactions === null && (dependency.f & 2) !== 0 && (new_deps === null || !new_deps.includes(dependency))) {
		var derived = dependency;
		if ((derived.f & 512) !== 0) {
			derived.f ^= 512;
			derived.f &= ~WAS_MARKED;
		}
		update_derived_status(derived);
		destroy_derived_effects(derived);
		remove_reactions(derived, 0);
	}
}
function remove_reactions(signal, start_index) {
	var dependencies = signal.deps;
	if (dependencies === null) return;
	for (var i = start_index; i < dependencies.length; i++) remove_reaction(signal, dependencies[i]);
}
function update_effect(effect) {
	var flags = effect.f;
	if ((flags & 16384) !== 0) return;
	set_signal_status(effect, CLEAN);
	var previous_effect = active_effect;
	var was_updating_effect = is_updating_effect;
	active_effect = effect;
	is_updating_effect = true;
	try {
		if ((flags & 16777232) !== 0) destroy_block_effect_children(effect);
		else destroy_effect_children(effect);
		execute_effect_teardown(effect);
		var teardown = update_reaction(effect);
		effect.teardown = typeof teardown === "function" ? teardown : null;
		effect.wv = write_version;
	} finally {
		is_updating_effect = was_updating_effect;
		active_effect = previous_effect;
	}
}
async function tick$1() {
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
					else new_deps.push(signal);
				}
			} else {
				(active_reaction.deps ??= []).push(signal);
				var reactions = signal.reactions;
				if (reactions === null) signal.reactions = [active_reaction];
				else if (!reactions.includes(active_reaction)) reactions.push(active_reaction);
			}
		}
	}
	if (is_destroying_effect && old_values.has(signal)) return old_values.get(signal);
	if (is_derived) {
		var derived = signal;
		if (is_destroying_effect) {
			var value = derived.v;
			if ((derived.f & 1024) === 0 && derived.reactions !== null || depends_on_old_values(derived)) value = execute_derived(derived);
			old_values.set(derived, value);
			return value;
		}
		var should_connect = (derived.f & 512) === 0 && !untracking && active_reaction !== null && (is_updating_effect || (active_reaction.f & 512) !== 0);
		var is_new = derived.deps === null;
		if (is_dirty(derived)) {
			if (should_connect) derived.f |= 512;
			update_derived(derived);
		}
		if (should_connect && !is_new) reconnect(derived);
	}
	if (batch_values?.has(signal)) return batch_values.get(signal);
	if ((signal.f & 8388608) !== 0) throw signal.v;
	return signal.v;
}
function reconnect(derived) {
	if (derived.deps === null) return;
	derived.f |= 512;
	for (const dep of derived.deps) {
		(dep.reactions ??= []).push(derived);
		if ((dep.f & 2) !== 0 && (dep.f & 512) === 0) reconnect(dep);
	}
}
function depends_on_old_values(derived) {
	if (derived.v === UNINITIALIZED) return true;
	if (derived.deps === null) return false;
	for (const dep of derived.deps) {
		if (old_values.has(dep)) return true;
		if ((dep.f & 2) !== 0 && depends_on_old_values(dep)) return true;
	}
	return false;
}
function untrack$1(fn) {
	var previous_untracking = untracking;
	try {
		untracking = true;
		return fn();
	} finally {
		untracking = previous_untracking;
	}
}
function deep_read_state(value) {
	if (typeof value !== "object" || !value || value instanceof EventTarget) return;
	if (STATE_SYMBOL in value) deep_read(value);
	else if (!Array.isArray(value)) for (let key in value) {
		const prop = value[key];
		if (typeof prop === "object" && prop && STATE_SYMBOL in prop) deep_read(prop);
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
				const get = descriptors[key].get;
				if (get) try {
					get.call(value);
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
function replay_events(dom) {
	if (!hydrating) return;
	dom.removeAttribute("onload");
	dom.removeAttribute("onerror");
	const event = dom.__e;
	if (event !== void 0) {
		dom.__e = void 0;
		queueMicrotask(() => {
			if (dom.isConnected) dom.dispatchEvent(event);
		});
	}
}
function create_event(event_name, dom, handler, options = {}) {
	function target_handler(event) {
		if (!options.capture) handle_event_propagation.call(dom, event);
		if (!event.cancelBubble) return without_reactive_context(() => {
			return handler?.call(this, event);
		});
	}
	if (event_name.startsWith("pointer") || event_name.startsWith("touch") || event_name === "wheel") queue_micro_task(() => {
		dom.addEventListener(event_name, target_handler, options);
	});
	else dom.addEventListener(event_name, target_handler, options);
	return target_handler;
}
function event(event_name, dom, handler, capture, passive) {
	var options = {
		capture,
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
function handle_event_propagation(event) {
	var handler_element = this;
	var owner_document = handler_element.ownerDocument;
	var event_name = event.type;
	var path = event.composedPath?.() || [];
	var current_target = path[0] || event.target;
	last_propagated_event = event;
	var path_idx = 0;
	var handled_at = last_propagated_event === event && event.__root;
	if (handled_at) {
		var at_idx = path.indexOf(handled_at);
		if (at_idx !== -1 && (handler_element === document || handler_element === window)) {
			event.__root = handler_element;
			return;
		}
		var handler_idx = path.indexOf(handler_element);
		if (handler_idx === -1) return;
		if (at_idx <= handler_idx) path_idx = at_idx;
	}
	current_target = path[path_idx] || event.target;
	if (current_target === handler_element) return;
	define_property(event, "currentTarget", {
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
				if (delegated != null && (!current_target.disabled || event.target === current_target)) delegated.call(current_target, event);
			} catch (error) {
				if (throw_error) other_errors.push(error);
				else throw_error = error;
			}
			if (event.cancelBubble || parent_element === handler_element || parent_element === null) break;
			current_target = parent_element;
		}
		if (throw_error) {
			for (let error of other_errors) queueMicrotask(() => {
				throw error;
			});
			throw throw_error;
		}
	} finally {
		event.__root = handler_element;
		delete event.currentTarget;
		set_active_reaction(previous_reaction);
		set_active_effect(previous_effect);
	}
}
function create_fragment_from_html(html) {
	var elem = document.createElement("template");
	elem.innerHTML = html.replaceAll("<!>", "<!---->");
	return elem.content;
}
function assign_nodes(start, end) {
	var effect = active_effect;
	if (effect.nodes === null) effect.nodes = {
		start,
		end,
		a: null,
		t: null
	};
}
/* @__NO_SIDE_EFFECTS__ */
function from_html(content, flags) {
	var is_fragment = (flags & 1) !== 0;
	var use_import_node = (flags & 2) !== 0;
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
		var clone = use_import_node || is_firefox ? document.importNode(node, true) : node.cloneNode(true);
		if (is_fragment) {
			var start = /* @__PURE__ */ get_first_child(clone);
			var end = clone.lastChild;
			assign_nodes(start, end);
		} else assign_nodes(clone, clone);
		return clone;
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
		var effect = active_effect;
		if ((effect.f & 32768) === 0 || effect.nodes.end === null) effect.nodes.end = hydrate_node;
		hydrate_next();
		return;
	}
	if (anchor === null) return;
	anchor.before(dom);
}
function set_text(text, value) {
	var str = value == null ? "" : typeof value === "object" ? value + "" : value;
	if (str !== (text.__t ??= text.nodeValue)) {
		text.__t = str;
		text.nodeValue = str + "";
	}
}
function mount(component, options) {
	return _mount(component, options);
}
function hydrate(component, options) {
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
		const instance = _mount(component, {
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
		return mount(component, options);
	} finally {
		set_hydrating(was_hydrating);
		set_hydrate_node(previous_hydrate_node);
	}
}
var document_listeners = /* @__PURE__ */ new Map();
function _mount(Component, { target, anchor, props = {}, events, context, intro = true }) {
	init_operations();
	var registered_events = /* @__PURE__ */ new Set();
	var event_handle = (events) => {
		for (var i = 0; i < events.length; i++) {
			var event_name = events[i];
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
	var component = void 0;
	var unmount = component_root(() => {
		var anchor_node = anchor ?? target.appendChild(create_text());
		boundary(anchor_node, { pending: () => {} }, (anchor_node) => {
			if (context) {
				push({});
				var ctx = component_context;
				ctx.c = context;
			}
			if (events) props.$$events = events;
			if (hydrating) assign_nodes(anchor_node, null);
			component = Component(anchor_node, props) || {};
			if (hydrating) {
				active_effect.nodes.end = hydrate_node;
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
	mounted_components.set(component, unmount);
	return component;
}
var mounted_components = /* @__PURE__ */ new WeakMap();
function unmount(component, options) {
	const fn = mounted_components.get(component);
	if (fn) {
		mounted_components.delete(component);
		return fn(options);
	}
	return Promise.resolve();
}
var BranchManager = class {
	anchor;
	#batches = /* @__PURE__ */ new Map();
	#onscreen = /* @__PURE__ */ new Map();
	#offscreen = /* @__PURE__ */ new Map();
	#outroing = /* @__PURE__ */ new Set();
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
		if (onscreen) {
			resume_effect(onscreen);
			this.#outroing.delete(key);
		} else {
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
			const offscreen = this.#offscreen.get(k);
			if (offscreen) {
				destroy_effect(offscreen.effect);
				this.#offscreen.delete(k);
			}
		}
		for (const [k, effect] of this.#onscreen) {
			if (k === key || this.#outroing.has(k)) continue;
			const on_destroy = () => {
				if (Array.from(this.#batches.values()).includes(k)) {
					var fragment = document.createDocumentFragment();
					move_effect(effect, fragment);
					fragment.append(create_text());
					this.#offscreen.set(k, {
						effect,
						fragment
					});
				} else destroy_effect(effect);
				this.#outroing.delete(k);
				this.#onscreen.delete(k);
			};
			if (this.#transition || !onscreen) {
				this.#outroing.add(k);
				pause_effect(effect, on_destroy, false);
			} else on_destroy();
		}
	};
	#discard = (batch) => {
		this.#batches.delete(batch);
		const keys = Array.from(this.#batches.values());
		for (const [k, branch] of this.#offscreen) if (!keys.includes(k)) {
			destroy_effect(branch.effect);
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
			for (const [k, effect] of this.#onscreen) if (k === key) batch.skipped_effects.delete(effect);
			else batch.skipped_effects.add(effect);
			for (const [k, branch] of this.#offscreen) if (k === key) batch.skipped_effects.delete(branch.effect);
			else batch.skipped_effects.add(branch.effect);
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
	var flags = elseif ? EFFECT_TRANSPARENT : 0;
	function update_branch(condition, fn) {
		if (hydrating) {
			if (condition === (read_hydration_instruction(node) === "[!")) {
				var anchor = skip_nodes();
				set_hydrate_node(anchor);
				branches.anchor = anchor;
				set_hydrating(false);
				branches.ensure(condition, fn);
				set_hydrating(true);
				return;
			}
		}
		branches.ensure(condition, fn);
	}
	block(() => {
		var has_branch = false;
		fn((fn, flag = true) => {
			has_branch = true;
			update_branch(flag, fn);
		});
		if (!has_branch) update_branch(false, null);
	}, flags);
}
function index(_, i) {
	return i;
}
function pause_effects(state, to_destroy, controlled_anchor) {
	var transitions = [];
	var length = to_destroy.length;
	var group;
	var remaining = to_destroy.length;
	for (var i = 0; i < length; i++) {
		let effect = to_destroy[i];
		pause_effect(effect, () => {
			if (group) {
				group.pending.delete(effect);
				group.done.add(effect);
				if (group.pending.size === 0) {
					var groups = state.outrogroups;
					destroy_effects(array_from(group.done));
					groups.delete(group);
					if (groups.size === 0) state.outrogroups = null;
				}
			} else remaining -= 1;
		}, false);
	}
	if (remaining === 0) {
		var fast_path = transitions.length === 0 && controlled_anchor !== null;
		if (fast_path) {
			var anchor = controlled_anchor;
			var parent_node = anchor.parentNode;
			clear_text_content(parent_node);
			parent_node.append(anchor);
			state.items.clear();
		}
		destroy_effects(to_destroy, !fast_path);
	} else {
		group = {
			pending: new Set(to_destroy),
			done: /* @__PURE__ */ new Set()
		};
		(state.outrogroups ??= /* @__PURE__ */ new Set()).add(group);
	}
}
function destroy_effects(to_destroy, remove_dom = true) {
	for (var i = 0; i < to_destroy.length; i++) destroy_effect(to_destroy[i], remove_dom);
}
var offscreen_anchor;
function each(node, flags, get_collection, get_key, render_fn, fallback_fn = null) {
	var anchor = node;
	var items = /* @__PURE__ */ new Map();
	if ((flags & 4) !== 0) {
		var parent_node = node;
		anchor = hydrating ? set_hydrate_node(/* @__PURE__ */ get_first_child(parent_node)) : parent_node.appendChild(create_text());
	}
	if (hydrating) hydrate_next();
	var fallback = null;
	var each_array = /* @__PURE__ */ derived_safe_equal(() => {
		var collection = get_collection();
		return is_array(collection) ? collection : collection == null ? [] : array_from(collection);
	});
	var array;
	var first_run = true;
	function commit() {
		state.fallback = fallback;
		reconcile(state, array, anchor, flags, get_key);
		if (fallback !== null) if (array.length === 0) if ((fallback.f & 33554432) === 0) resume_effect(fallback);
		else {
			fallback.f ^= EFFECT_OFFSCREEN;
			move(fallback, null, anchor);
		}
		else pause_effect(fallback, () => {
			fallback = null;
		});
	}
	var state = {
		effect: block(() => {
			array = get$1(each_array);
			var length = array.length;
			let mismatch = false;
			if (hydrating) {
				if (read_hydration_instruction(anchor) === "[!" !== (length === 0)) {
					anchor = skip_nodes();
					set_hydrate_node(anchor);
					set_hydrating(false);
					mismatch = true;
				}
			}
			var keys = /* @__PURE__ */ new Set();
			var batch = current_batch;
			var defer = should_defer_append();
			for (var index = 0; index < length; index += 1) {
				if (hydrating && hydrate_node.nodeType === 8 && hydrate_node.data === "]") {
					anchor = hydrate_node;
					mismatch = true;
					set_hydrating(false);
				}
				var value = array[index];
				var key = get_key(value, index);
				var item = first_run ? null : items.get(key);
				if (item) {
					if (item.v) internal_set(item.v, value);
					if (item.i) internal_set(item.i, index);
					if (defer) batch.skipped_effects.delete(item.e);
				} else {
					item = create_item(items, first_run ? anchor : offscreen_anchor ??= create_text(), value, key, index, render_fn, flags, get_collection);
					if (!first_run) item.e.f |= EFFECT_OFFSCREEN;
					items.set(key, item);
				}
				keys.add(key);
			}
			if (length === 0 && fallback_fn && !fallback) if (first_run) fallback = branch(() => fallback_fn(anchor));
			else {
				fallback = branch(() => fallback_fn(offscreen_anchor ??= create_text()));
				fallback.f |= EFFECT_OFFSCREEN;
			}
			if (hydrating && length > 0) set_hydrate_node(skip_nodes());
			if (!first_run) if (defer) {
				for (const [key, item] of items) if (!keys.has(key)) batch.skipped_effects.add(item.e);
				batch.oncommit(commit);
				batch.ondiscard(() => {});
			} else commit();
			if (mismatch) set_hydrating(true);
			get$1(each_array);
		}),
		flags,
		items,
		outrogroups: null,
		fallback
	};
	first_run = false;
	if (hydrating) anchor = hydrate_node;
}
function reconcile(state, array, anchor, flags, get_key) {
	var is_animated = (flags & 8) !== 0;
	var length = array.length;
	var items = state.items;
	var current = state.effect.first;
	var seen;
	var prev = null;
	var to_animate;
	var matched = [];
	var stashed = [];
	var value;
	var key;
	var effect;
	var i;
	if (is_animated) for (i = 0; i < length; i += 1) {
		value = array[i];
		key = get_key(value, i);
		effect = items.get(key).e;
		if ((effect.f & 33554432) === 0) {
			effect.nodes?.a?.measure();
			(to_animate ??= /* @__PURE__ */ new Set()).add(effect);
		}
	}
	for (i = 0; i < length; i += 1) {
		value = array[i];
		key = get_key(value, i);
		effect = items.get(key).e;
		if (state.outrogroups !== null) for (const group of state.outrogroups) {
			group.pending.delete(effect);
			group.done.delete(effect);
		}
		if ((effect.f & 33554432) !== 0) {
			effect.f ^= EFFECT_OFFSCREEN;
			if (effect === current) move(effect, null, anchor);
			else {
				var next = prev ? prev.next : current;
				if (effect === state.effect.last) state.effect.last = effect.prev;
				if (effect.prev) effect.prev.next = effect.next;
				if (effect.next) effect.next.prev = effect.prev;
				link(state, prev, effect);
				link(state, effect, next);
				move(effect, next, anchor);
				prev = effect;
				matched = [];
				stashed = [];
				current = prev.next;
				continue;
			}
		}
		if ((effect.f & 8192) !== 0) {
			resume_effect(effect);
			if (is_animated) {
				effect.nodes?.a?.unfix();
				(to_animate ??= /* @__PURE__ */ new Set()).delete(effect);
			}
		}
		if (effect !== current) {
			if (seen !== void 0 && seen.has(effect)) {
				if (matched.length < stashed.length) {
					var start = stashed[0];
					var j;
					prev = start.prev;
					var a = matched[0];
					var b = matched[matched.length - 1];
					for (j = 0; j < matched.length; j += 1) move(matched[j], start, anchor);
					for (j = 0; j < stashed.length; j += 1) seen.delete(stashed[j]);
					link(state, a.prev, b.next);
					link(state, prev, a);
					link(state, b, start);
					current = start;
					prev = b;
					i -= 1;
					matched = [];
					stashed = [];
				} else {
					seen.delete(effect);
					move(effect, current, anchor);
					link(state, effect.prev, effect.next);
					link(state, effect, prev === null ? state.effect.first : prev.next);
					link(state, prev, effect);
					prev = effect;
				}
				continue;
			}
			matched = [];
			stashed = [];
			while (current !== null && current !== effect) {
				(seen ??= /* @__PURE__ */ new Set()).add(current);
				stashed.push(current);
				current = current.next;
			}
			if (current === null) continue;
		}
		if ((effect.f & 33554432) === 0) matched.push(effect);
		prev = effect;
		current = effect.next;
	}
	if (state.outrogroups !== null) {
		for (const group of state.outrogroups) if (group.pending.size === 0) {
			destroy_effects(array_from(group.done));
			state.outrogroups?.delete(group);
		}
		if (state.outrogroups.size === 0) state.outrogroups = null;
	}
	if (current !== null || seen !== void 0) {
		var to_destroy = [];
		if (seen !== void 0) {
			for (effect of seen) if ((effect.f & 8192) === 0) to_destroy.push(effect);
		}
		while (current !== null) {
			if ((current.f & 8192) === 0 && current !== state.fallback) to_destroy.push(current);
			current = current.next;
		}
		var destroy_length = to_destroy.length;
		if (destroy_length > 0) {
			var controlled_anchor = (flags & 4) !== 0 && length === 0 ? anchor : null;
			if (is_animated) {
				for (i = 0; i < destroy_length; i += 1) to_destroy[i].nodes?.a?.measure();
				for (i = 0; i < destroy_length; i += 1) to_destroy[i].nodes?.a?.fix();
			}
			pause_effects(state, to_destroy, controlled_anchor);
		}
	}
	if (is_animated) queue_micro_task(() => {
		if (to_animate === void 0) return;
		for (effect of to_animate) effect.nodes?.a?.apply();
	});
}
function create_item(items, anchor, value, key, index, render_fn, flags, get_collection) {
	var v = (flags & 1) !== 0 ? (flags & 16) === 0 ? /* @__PURE__ */ mutable_source(value, false, false) : source(value) : null;
	var i = (flags & 2) !== 0 ? source(index) : null;
	return {
		v,
		i,
		e: branch(() => {
			render_fn(anchor, v ?? value, i ?? index, get_collection);
			return () => {
				items.delete(key);
			};
		})
	};
}
function move(effect, next, anchor) {
	if (!effect.nodes) return;
	var node = effect.nodes.start;
	var end = effect.nodes.end;
	var dest = next && (next.f & 33554432) === 0 ? next.nodes.start : anchor;
	while (node !== null) {
		var next_node = /* @__PURE__ */ get_next_sibling(node);
		dest.before(node);
		if (node === end) return;
		node = next_node;
	}
}
function link(state, prev, next) {
	if (prev === null) state.effect.first = next;
	else prev.next = next;
	if (next === null) state.effect.last = prev;
	else next.prev = prev;
}
function html(node, get_value, svg = false, mathml = false, skip_warning = false) {
	var anchor = node;
	var value = "";
	template_effect(() => {
		var effect = active_effect;
		if (value === (value = get_value() ?? "")) {
			if (hydrating) hydrate_next();
			return;
		}
		if (effect.nodes !== null) {
			remove_effect_dom(effect.nodes.start, effect.nodes.end);
			effect.nodes = null;
		}
		if (value === "") return;
		if (hydrating) {
			hydrate_node.data;
			var next = hydrate_next();
			var last = next;
			while (next !== null && (next.nodeType !== 8 || next.data !== "")) {
				last = next;
				next = /* @__PURE__ */ get_next_sibling(next);
			}
			if (next === null) {
				hydration_mismatch();
				throw HYDRATION_ERROR;
			}
			assign_nodes(hydrate_node, last);
			anchor = set_hydrate_node(next);
			return;
		}
		var html = value + "";
		if (svg) html = `<svg>${html}</svg>`;
		else if (mathml) html = `<math>${html}</math>`;
		var node = create_fragment_from_html(html);
		if (svg || mathml) node = /* @__PURE__ */ get_first_child(node);
		assign_nodes(/* @__PURE__ */ get_first_child(node), node.lastChild);
		if (svg || mathml) while (/* @__PURE__ */ get_first_child(node)) anchor.before(/* @__PURE__ */ get_first_child(node));
		else anchor.before(node);
	});
}
function slot(anchor, $$props, name, slot_props, fallback_fn) {
	if (hydrating) hydrate_next();
	var slot_fn = $$props.$$slots?.[name];
	var is_interop = false;
	if (slot_fn === true) {
		slot_fn = $$props[name === "default" ? "children" : name];
		is_interop = true;
	}
	if (slot_fn === void 0) {
		if (fallback_fn !== null) fallback_fn(anchor);
	} else slot_fn(anchor, is_interop ? () => slot_props : slot_props);
}
function snippet(node, get_snippet, ...args) {
	var branches = new BranchManager(node);
	block(() => {
		const snippet = get_snippet() ?? null;
		branches.ensure(snippet, snippet && ((anchor) => snippet(anchor, ...args)));
	}, EFFECT_TRANSPARENT);
}
function createRawSnippet(fn) {
	return (anchor, ...params) => {
		var snippet = fn(...params);
		var element;
		if (hydrating) {
			element = hydrate_node;
			hydrate_next();
		} else {
			element = /* @__PURE__ */ get_first_child(create_fragment_from_html(snippet.render().trim()));
			anchor.before(element);
		}
		const result = snippet.setup?.(element);
		assign_nodes(element, element);
		if (typeof result === "function") teardown(result);
	};
}
function component(node, get_component, render_fn) {
	if (hydrating) hydrate_next();
	var branches = new BranchManager(node);
	block(() => {
		var component = get_component() ?? null;
		branches.ensure(component, component && ((target) => render_fn(target, component)));
	}, EFFECT_TRANSPARENT);
}
function head(hash, render_fn) {
	let previous_hydrate_node = null;
	let was_hydrating = hydrating;
	var anchor;
	if (hydrating) {
		previous_hydrate_node = hydrate_node;
		var head_anchor = /* @__PURE__ */ get_first_child(document.head);
		while (head_anchor !== null && (head_anchor.nodeType !== 8 || head_anchor.data !== hash)) head_anchor = /* @__PURE__ */ get_next_sibling(head_anchor);
		if (head_anchor === null) set_hydrating(false);
		else {
			var start = /* @__PURE__ */ get_next_sibling(head_anchor);
			head_anchor.remove();
			set_hydrate_node(start);
		}
	}
	if (!hydrating) anchor = document.head.appendChild(create_text());
	try {
		block(() => render_fn(anchor), HEAD_EFFECT);
	} finally {
		if (was_hydrating) {
			set_hydrating(true);
			set_hydrate_node(previous_hydrate_node);
		}
	}
}
function action(dom, action, get_value) {
	effect(() => {
		var payload = untrack$1(() => action(dom, get_value?.()) || {});
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
	managed(() => {
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
function to_class(value, hash, directives) {
	var classname = value == null ? "" : "" + value;
	if (hash) classname = classname ? classname + " " + hash : hash;
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
function set_class(dom, is_html, value, hash, prev_classes, next_classes) {
	var prev = dom.__className;
	if (hydrating || prev !== value || prev === void 0) {
		var next_class_name = to_class(value, hash, next_classes);
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
function update_styles(dom, prev = {}, next, priority) {
	for (var key in next) {
		var value = next[key];
		if (prev[key] !== value) if (next[key] == null) dom.style.removeProperty(key);
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
function set_attributes(element, prev, next, css_hash, should_remove_defaults = false, skip_warning = false) {
	if (hydrating && should_remove_defaults && element.tagName === "INPUT") {
		var input = element;
		if (!((input.type === "checkbox" ? "defaultChecked" : "defaultValue") in next)) remove_input_defaults(input);
	}
	var attributes = get_attributes(element);
	var is_custom_element = attributes[IS_CUSTOM_ELEMENT];
	var preserve_attribute_case = !attributes[IS_HTML];
	let is_hydrating_custom_element = hydrating && is_custom_element;
	if (is_hydrating_custom_element) set_hydrating(false);
	var current = prev || {};
	var is_option_element = element.tagName === "OPTION";
	for (var key in prev) if (!(key in next)) next[key] = null;
	if (next.class) next.class = clsx(next.class);
	else if (css_hash || next[CLASS]) next.class = null;
	if (next[STYLE]) next.style ??= null;
	var setters = get_setters(element);
	for (const key in next) {
		let value = next[key];
		if (is_option_element && key === "value" && value == null) {
			element.value = element.__value = "";
			current[key] = value;
			continue;
		}
		if (key === "class") {
			set_class(element, element.namespaceURI === "http://www.w3.org/1999/xhtml", value, css_hash, prev?.[CLASS], next[CLASS]);
			current[key] = value;
			current[CLASS] = next[CLASS];
			continue;
		}
		if (key === "style") {
			set_style(element, value, prev?.[STYLE], next[STYLE]);
			current[key] = value;
			current[STYLE] = next[STYLE];
			continue;
		}
		var prev_value = current[key];
		if (value === prev_value && !(value === void 0 && element.hasAttribute(key))) continue;
		current[key] = value;
		var prefix = key[0] + key[1];
		if (prefix === "$$") continue;
		if (prefix === "on") {
			const opts = {};
			const event_handle_key = "$$" + key;
			let event_name = key.slice(2);
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
					current[key].call(this, evt);
				}
				current[event_handle_key] = create_event(event_name, element, handle, opts);
			} else {
				element[`__${event_name}`] = value;
				delegate([event_name]);
			}
			else if (delegated) element[`__${event_name}`] = void 0;
		} else if (key === "style") set_attribute(element, key, value);
		else if (key === "autofocus") autofocus(element, Boolean(value));
		else if (!is_custom_element && (key === "__value" || key === "value" && value != null)) element.value = element.__value = value;
		else if (key === "selected" && is_option_element) set_selected(element, value);
		else {
			var name = key;
			if (!preserve_attribute_case) name = normalize_attribute(name);
			var is_default = name === "defaultValue" || name === "defaultChecked";
			if (value == null && !is_custom_element && !is_default) {
				attributes[key] = null;
				if (name === "value" || name === "checked") {
					let input = element;
					const use_default = prev === void 0;
					if (name === "value") {
						let previous = input.defaultValue;
						input.removeAttribute(name);
						input.defaultValue = previous;
						input.value = input.__value = use_default ? previous : null;
					} else {
						let previous = input.defaultChecked;
						input.removeAttribute(name);
						input.defaultChecked = previous;
						input.checked = use_default ? previous : false;
					}
				} else element.removeAttribute(key);
			} else if (is_default || setters.includes(name) && (is_custom_element || typeof value !== "string")) {
				element[name] = value;
				if (name in attributes) attributes[name] = UNINITIALIZED;
			} else if (typeof value !== "function") set_attribute(element, name, value, skip_warning);
		}
	}
	if (is_hydrating_custom_element) set_hydrating(true);
	return current;
}
function attribute_effect(element, fn, sync = [], async = [], blockers = [], css_hash, should_remove_defaults = false, skip_warning = false) {
	flatten(blockers, sync, async, (values) => {
		var prev = void 0;
		var effects = {};
		var is_select = element.nodeName === "SELECT";
		var inited = false;
		managed(() => {
			var next = fn(...values.map(get$1));
			var current = set_attributes(element, prev, next, css_hash, should_remove_defaults, skip_warning);
			if (inited && is_select && "value" in next) select_option(element, next.value);
			for (let symbol of Object.getOwnPropertySymbols(effects)) if (!next[symbol]) destroy_effect(effects[symbol]);
			for (let symbol of Object.getOwnPropertySymbols(next)) {
				var n = next[symbol];
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
function bind_value(input, get, set = get) {
	var batches = /* @__PURE__ */ new WeakSet();
	listen_to_event_and_reset_event(input, "input", async (is_reset) => {
		var value = is_reset ? input.defaultValue : input.value;
		value = is_numberlike_input(input) ? to_number(value) : value;
		set(value);
		if (current_batch !== null) batches.add(current_batch);
		await tick$1();
		if (value !== (value = get())) {
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
	if (hydrating && input.defaultValue !== input.value || untrack$1(get) == null && input.value) {
		set(is_numberlike_input(input) ? to_number(input.value) : input.value);
		if (current_batch !== null) batches.add(current_batch);
	}
	render_effect(() => {
		var value = get();
		if (input === document.activeElement) {
			var batch = previous_batch ?? current_batch;
			if (batches.has(batch)) return;
		}
		if (is_numberlike_input(input) && value === to_number(input.value)) return;
		if (input.type === "date" && !value && !input.value) return;
		if (value !== input.value) input.value = value ?? "";
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
function bind_this(element_or_component = {}, update, get_value, get_parts) {
	effect(() => {
		var old_parts;
		var parts;
		render_effect(() => {
			old_parts = parts;
			parts = get_parts?.() || [];
			untrack$1(() => {
				if (element_or_component !== get_value(...parts)) {
					update(element_or_component, ...parts);
					if (old_parts && is_bound_this(get_value(...old_parts), element_or_component)) update(null, ...old_parts);
				}
			});
		});
		return () => {
			queue_micro_task(() => {
				if (parts && is_bound_this(get_value(...parts), element_or_component)) update(null, ...parts);
			});
		};
	});
	return element_or_component;
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
			const props = context.s;
			for (const key in props) if (props[key] !== prev[key]) {
				prev[key] = props[key];
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
		const fns = untrack$1(() => callbacks.m.map(run));
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
var subscriber_queue = [];
function writable(value, start = noop$1) {
	let stop = null;
	const subscribers = /* @__PURE__ */ new Set();
	function set(new_value) {
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
	function update(fn) {
		set(fn(value));
	}
	function subscribe(run, invalidate = noop$1) {
		const subscriber = [run, invalidate];
		subscribers.add(subscriber);
		if (subscribers.size === 1) stop = start(set, update) || noop$1;
		run(value);
		return () => {
			subscribers.delete(subscriber);
			if (subscribers.size === 0 && stop) {
				stop();
				stop = null;
			}
		};
	}
	return {
		set,
		update,
		subscribe
	};
}
var is_store_binding = false;
function capture_store_binding(fn) {
	var previous_is_store_binding = is_store_binding;
	try {
		is_store_binding = false;
		return [fn(), is_store_binding];
	} finally {
		is_store_binding = previous_is_store_binding;
	}
}
function prop(props, key, flags, fallback) {
	var runes = !legacy_mode_flag || (flags & 2) !== 0;
	var bindable = (flags & 8) !== 0;
	var lazy = (flags & 16) !== 0;
	var fallback_value = fallback;
	var fallback_dirty = true;
	var get_fallback = () => {
		if (fallback_dirty) {
			fallback_dirty = false;
			fallback_value = lazy ? untrack$1(fallback) : fallback;
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
	if (runes && (flags & 4) === 0) return getter;
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
	var d = ((flags & 1) !== 0 ? derived : derived_safe_equal)(() => {
		overridden = false;
		return getter();
	});
	if (bindable) get$1(d);
	var parent_effect = active_effect;
	return (function(value, mutation) {
		if (arguments.length > 0) {
			const new_value = mutation ? get$1(d) : runes && bindable ? proxy(value) : value;
			set$1(d, new_value);
			overridden = true;
			if (fallback_value !== void 0) fallback_value = new_value;
			return value;
		}
		if (is_destroying_effect && overridden || (parent_effect.f & 16384) !== 0) return d.v;
		return get$1(d);
	});
}
function asClassComponent(component) {
	return class extends Svelte4Component {
		constructor(options) {
			super({
				component,
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
			get(target, prop) {
				return get$1(sources.get(prop) ?? add_source(prop, Reflect.get(target, prop)));
			},
			has(target, prop) {
				if (prop === LEGACY_PROPS) return true;
				get$1(sources.get(prop) ?? add_source(prop, Reflect.get(target, prop)));
				return Reflect.has(target, prop);
			},
			set(target, prop, value) {
				set$1(sources.get(prop) ?? add_source(prop, value), value);
				return Reflect.set(target, prop, value);
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
		this.#instance.$set = (next) => {
			Object.assign(props, next);
		};
		this.#instance.$destroy = () => {
			unmount(this.#instance);
		};
	}
	$set(props) {
		this.#instance.$set(props);
	}
	$on(event, callback) {
		this.#events[event] = this.#events[event] || [];
		const cb = (...args) => callback.call(this, ...args);
		this.#events[event].push(cb);
		return () => {
			this.#events[event] = this.#events[event].filter((fn) => fn !== cb);
		};
	}
	$destroy() {
		this.#instance.$destroy();
	}
};
if (typeof HTMLElement === "function") HTMLElement;
function hydratable(key, fn) {
	experimental_async_required("hydratable");
	if (hydrating) {
		const store = window.__svelte?.h;
		if (store?.has(key)) return store.get(key);
		hydratable_missing_but_expected(key);
	}
	return fn();
}
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
	hydratable: () => hydratable,
	hydrate: () => hydrate,
	mount: () => mount,
	onDestroy: () => onDestroy,
	onMount: () => onMount$1,
	setContext: () => setContext,
	settled: () => settled,
	tick: () => tick$1,
	unmount: () => unmount,
	untrack: () => untrack$1
}, 1);
function getAbortSignal() {
	if (active_reaction === null) get_abort_signal_outside_reaction();
	return (active_reaction.ac ??= new AbortController()).signal;
}
function onMount$1(fn) {
	if (component_context === null) lifecycle_outside_component("onMount");
	if (legacy_mode_flag && component_context.l !== null) init_update_callbacks(component_context).m.push(fn);
	else user_effect(() => {
		const cleanup = untrack$1(fn);
		if (typeof cleanup === "function") return cleanup;
	});
}
function onDestroy(fn) {
	if (component_context === null) lifecycle_outside_component("onDestroy");
	onMount$1(() => () => untrack$1(fn));
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
			const event = create_custom_event(type, detail, options);
			for (const fn of callbacks) fn.call(active_component_context.x, event);
			return !event.defaultPrevented;
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
	constructor(status, location) {
		this.status = status;
		this.location = location;
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
function make_trackable(url, callback, search_params_callback, allow_hash = false) {
	const tracked = new URL(url);
	Object.defineProperty(tracked, "searchParams", {
		value: new Proxy(tracked.searchParams, { get(obj, key) {
			if (key === "get" || key === "getAll" || key === "has") return (param, ...rest) => {
				search_params_callback(param);
				return obj[key](param, ...rest);
			};
			callback();
			const value = Reflect.get(obj, key);
			return typeof value === "function" ? value.bind(obj) : value;
		} }),
		enumerable: true,
		configurable: true
	});
	const tracked_url_properties = [
		"href",
		"pathname",
		"search",
		"toString",
		"toJSON"
	];
	if (allow_hash) tracked_url_properties.push("hash");
	for (const property of tracked_url_properties) Object.defineProperty(tracked, property, {
		get() {
			callback();
			return url[property];
		},
		enumerable: true,
		configurable: true
	});
	return tracked;
}
function hash(...values) {
	let hash = 5381;
	for (const value of values) if (typeof value === "string") {
		let i = value.length;
		while (i) hash = hash * 33 ^ value.charCodeAt(--i);
	} else if (ArrayBuffer.isView(value)) {
		const buffer = new Uint8Array(value.buffer, value.byteOffset, value.byteLength);
		let i = buffer.length;
		while (i) hash = hash * 33 ^ buffer[--i];
	} else throw new TypeError("value must be a string or TypedArray");
	return (hash >>> 0).toString(36);
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
		pattern: id === "/" ? /^\/$/ : new RegExp(`^${get_route_segments(id).map((segment) => {
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
function get(key, parse = JSON.parse) {
	try {
		return parse(sessionStorage[key]);
	} catch {}
}
function set(key, value, stringify = JSON.stringify) {
	const data = stringify(value);
	try {
		sessionStorage[key] = data;
	} catch {}
}
const base = globalThis.__sveltekit_u26lzl?.base ?? "/store";
const assets = globalThis.__sveltekit_u26lzl?.assets ?? base ?? "";
const version = "1769905178834";
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
function find_anchor(element, target) {
	while (element && element !== target) {
		if (element.nodeName.toUpperCase() === "A" && element.hasAttribute("href")) return element;
		element = parent_element(element);
	}
}
function get_link_info(a, base, uses_hash_router) {
	let url;
	try {
		url = new URL(a instanceof SVGAElement ? a.href.baseVal : a.href, document.baseURI);
		if (uses_hash_router && url.hash.match(/^#[^/]/)) {
			const route = location.hash.split("#")[1] || "/";
			url.hash = `#${route}${url.hash}`;
		}
	} catch {}
	const target = a instanceof SVGAElement ? a.target.baseVal : a.target;
	const external = !url || !!target || is_external_url(url, base, uses_hash_router) || (a.getAttribute("rel") || "").split(/\s+/).includes("external");
	const download = url?.origin === origin && a.hasAttribute("download");
	return {
		url,
		external,
		target,
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
	function set(new_value) {
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
		set,
		subscribe
	};
}
const updated_listener = { v: () => {} };
function create_updated_store() {
	const { set, subscribe } = writable(false);
	let timeout;
	async function check() {
		clearTimeout(timeout);
		try {
			const res = await fetch(`${assets}/_app/version.json`, { headers: {
				pragma: "no-cache",
				"cache-control": "no-cache"
			} });
			if (!res.ok) return false;
			const updated = (await res.json()).version !== version;
			if (updated) {
				set(true);
				updated_listener.v();
				clearTimeout(timeout);
			}
			return updated;
		} catch {
			return false;
		}
	}
	return {
		subscribe,
		check
	};
}
function is_external_url(url, base, hash_routing) {
	if (url.origin !== origin || !url.pathname.startsWith(base)) return true;
	if (hash_routing) return url.pathname !== location.pathname;
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
let page$3;
let navigating$1;
let updated$1;
if (onMount$1.toString().includes("$$") || /function \w+\(\) \{\}/.test(onMount$1.toString())) {
	page$3 = {
		data: {},
		form: null,
		error: null,
		params: {},
		route: { id: null },
		state: {},
		status: -1,
		url: new URL("https://example.com")
	};
	navigating$1 = { current: null };
	updated$1 = { current: false };
} else {
	page$3 = new class Page {
		#data = /* @__PURE__ */ state({});
		get data() {
			return get$1(this.#data);
		}
		set data(value) {
			set$1(this.#data, value);
		}
		#form = /* @__PURE__ */ state(null);
		get form() {
			return get$1(this.#form);
		}
		set form(value) {
			set$1(this.#form, value);
		}
		#error = /* @__PURE__ */ state(null);
		get error() {
			return get$1(this.#error);
		}
		set error(value) {
			set$1(this.#error, value);
		}
		#params = /* @__PURE__ */ state({});
		get params() {
			return get$1(this.#params);
		}
		set params(value) {
			set$1(this.#params, value);
		}
		#route = /* @__PURE__ */ state({ id: null });
		get route() {
			return get$1(this.#route);
		}
		set route(value) {
			set$1(this.#route, value);
		}
		#state = /* @__PURE__ */ state({});
		get state() {
			return get$1(this.#state);
		}
		set state(value) {
			set$1(this.#state, value);
		}
		#status = /* @__PURE__ */ state(-1);
		get status() {
			return get$1(this.#status);
		}
		set status(value) {
			set$1(this.#status, value);
		}
		#url = /* @__PURE__ */ state(new URL("https://example.com"));
		get url() {
			return get$1(this.#url);
		}
		set url(value) {
			set$1(this.#url, value);
		}
	}();
	navigating$1 = new class Navigating {
		#current = /* @__PURE__ */ state(null);
		get current() {
			return get$1(this.#current);
		}
		set current(value) {
			set$1(this.#current, value);
		}
	}();
	updated$1 = new class Updated {
		#current = /* @__PURE__ */ state(false);
		get current() {
			return get$1(this.#current);
		}
		set current(value) {
			set$1(this.#current, value);
		}
	}();
	updated_listener.v = () => updated$1.current = true;
}
function update(new_page) {
	Object.assign(page$3, new_page);
}
const noop_span = {
	spanContext() {
		return noop_span_context;
	},
	setAttribute() {
		return this;
	},
	setAttributes() {
		return this;
	},
	addEvent() {
		return this;
	},
	setStatus() {
		return this;
	},
	updateName() {
		return this;
	},
	end() {
		return this;
	},
	isRecording() {
		return false;
	},
	recordException() {
		return this;
	},
	addLink() {
		return this;
	},
	addLinks() {
		return this;
	}
};
var noop_span_context = {
	traceId: "",
	spanId: "",
	traceFlags: 0
};
var scriptRel = "modulepreload";
var assetsURL = function(dep, importerUrl) {
	return new URL(dep, importerUrl).href;
};
var seen = {};
const __vitePreload = function preload(baseModule, deps, importerUrl) {
	let promise = Promise.resolve();
	if (deps && deps.length > 0) {
		const links = document.getElementsByTagName("link");
		const cspNonceMeta = document.querySelector("meta[property=csp-nonce]");
		const cspNonce = cspNonceMeta?.nonce || cspNonceMeta?.getAttribute("nonce");
		function allSettled(promises$2) {
			return Promise.all(promises$2.map((p) => Promise.resolve(p).then((value$1) => ({
				status: "fulfilled",
				value: value$1
			}), (reason) => ({
				status: "rejected",
				reason
			}))));
		}
		promise = allSettled(deps.map((dep) => {
			dep = assetsURL(dep, importerUrl);
			if (dep in seen) return;
			seen[dep] = true;
			const isCss = dep.endsWith(".css");
			const cssSelector = isCss ? "[rel=\"stylesheet\"]" : "";
			if (!!importerUrl) for (let i$1 = links.length - 1; i$1 >= 0; i$1--) {
				const link$1 = links[i$1];
				if (link$1.href === dep && (!isCss || link$1.rel === "stylesheet")) return;
			}
			else if (document.querySelector(`link[href="${dep}"]${cssSelector}`)) return;
			const link = document.createElement("link");
			link.rel = isCss ? "stylesheet" : scriptRel;
			if (!isCss) link.as = "script";
			link.crossOrigin = "";
			link.href = dep;
			if (cspNonce) link.setAttribute("nonce", cspNonce);
			document.head.appendChild(link);
			if (isCss) return new Promise((res, rej) => {
				link.addEventListener("load", res);
				link.addEventListener("error", () => rej(/* @__PURE__ */ new Error(`Unable to preload CSS for ${dep}`)));
			});
		}));
	}
	function handlePreloadError(err$2) {
		const e$1 = new Event("vite:preloadError", { cancelable: true });
		e$1.payload = err$2;
		window.dispatchEvent(e$1);
		if (!e$1.defaultPrevented) throw err$2;
	}
	return promise.then((res) => {
		for (const item of res || []) {
			if (item.status !== "rejected") continue;
			handlePreloadError(item.reason);
		}
		return baseModule().catch(handlePreloadError);
	});
};
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
function clear_onward_history(current_history_index, current_navigation_index) {
	let i = current_history_index + 1;
	while (scroll_positions[i]) {
		delete scroll_positions[i];
		i += 1;
	}
	i = current_navigation_index + 1;
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
function discard_load_cache() {
	load_cache?.fork?.then((f) => f?.discard());
	load_cache = null;
}
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
var root$16;
var current_history_index;
var current_navigation_index;
var token;
var preload_tokens = /* @__PURE__ */ new Set();
const query_map = /* @__PURE__ */ new Map();
async function start(_app, _target, hydrate) {
	if (globalThis.__sveltekit_u26lzl?.data) globalThis.__sveltekit_u26lzl.data;
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
	if (options.invalidateAll) discard_load_cache();
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
		discard_load_cache();
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
				if (result.type === "loaded" && result.state.error) discard_load_cache();
				return result;
			}),
			fork: null
		};
	}
	return load_cache.promise;
}
async function _preload_code(url) {
	const route = (await get_navigation_intent(url, false))?.route;
	if (route) await Promise.all([...route.layouts, route.leaf].map((load) => load?.[1]()));
}
async function initialize(result, target, hydrate) {
	current = result.state;
	const style = document.querySelector("style[data-sveltekit]");
	if (style) style.remove();
	Object.assign(page$3, result.props.page);
	root$16 = new app.root({
		target,
		props: {
			...result.props,
			stores,
			components
		},
		hydrate,
		sync: false
	});
	await Promise.resolve();
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
			page: clone_page(page$3)
		}
	};
	if (form !== void 0) result.props.form = form;
	let data = {};
	let data_changed = !page$3;
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
	if (!current.url || url.href !== current.url.href || current.error !== error || form !== void 0 && form !== page$3.form || data_changed) result.props.page = {
		error,
		params,
		route: { id: route?.id ?? null },
		state: {},
		status,
		url: new URL(url),
		form: form ?? null,
		data: data_changed ? data : page$3.data
	};
	return result;
}
async function load_node({ loader, parent, url, params, route, server_data_node }) {
	let data = null;
	let is_tracking = true;
	const uses = {
		dependencies: /* @__PURE__ */ new Set(),
		params: /* @__PURE__ */ new Set(),
		parent: false,
		route: false,
		url: false,
		search_params: /* @__PURE__ */ new Set()
	};
	const node = await loader();
	if (node.universal?.load) {
		function depends(...deps) {
			for (const dep of deps) {
				const { href } = new URL(dep, url);
				uses.dependencies.add(href);
			}
		}
		const load_input = {
			tracing: {
				enabled: false,
				root: noop_span,
				current: noop_span
			},
			route: new Proxy(route, { get: (target, key) => {
				if (is_tracking) uses.route = true;
				return target[key];
			} }),
			params: new Proxy(params, { get: (target, key) => {
				if (is_tracking) uses.params.add(key);
				return target[key];
			} }),
			data: server_data_node?.data ?? null,
			url: make_trackable(url, () => {
				if (is_tracking) uses.url = true;
			}, (param) => {
				if (is_tracking) uses.search_params.add(param);
			}, app.hash),
			async fetch(resource, init) {
				if (resource instanceof Request) init = {
					body: resource.method === "GET" || resource.method === "HEAD" ? void 0 : await resource.blob(),
					cache: resource.cache,
					credentials: resource.credentials,
					headers: [...resource.headers].length > 0 ? resource?.headers : void 0,
					integrity: resource.integrity,
					keepalive: resource.keepalive,
					method: resource.method,
					mode: resource.mode,
					redirect: resource.redirect,
					referrer: resource.referrer,
					referrerPolicy: resource.referrerPolicy,
					signal: resource.signal,
					...init
				};
				const { resolved, promise } = resolve_fetch_url(resource, init, url);
				if (is_tracking) depends(resolved.href);
				return promise;
			},
			setHeaders: () => {},
			depends,
			parent() {
				if (is_tracking) uses.parent = true;
				return parent();
			},
			untrack(fn) {
				is_tracking = false;
				try {
					return fn();
				} finally {
					is_tracking = true;
				}
			}
		};
		data = await node.universal.load.call(null, load_input) ?? null;
	}
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
			page: clone_page(page$3),
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
	} catch (error) {
		if (error instanceof Redirect) return _goto(new URL(error.location, location.href), {}, 0);
		throw error;
	}
}
async function get_rerouted_url(url) {
	const href = url.href;
	if (reroute_cache.has(href)) return reroute_cache.get(href);
	let rerouted;
	try {
		const promise = (async () => {
			let rerouted = await app.hooks.reroute({
				url: new URL(url),
				fetch: async (input, init) => {
					return resolve_fetch_url(input, init, url).promise;
				}
			}) ?? url;
			if (typeof rerouted === "string") {
				const tmp = new URL(url);
				if (app.hash) tmp.hash = rerouted;
				else tmp.pathname = rerouted;
				rerouted = tmp;
			}
			return rerouted;
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
async function navigate({ type, url, popped, keepfocus, noscroll, replace_state, state = {}, redirect_count = 0, nav_token = {}, accept = noop, block = noop, event }) {
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
	if (started && nav.navigation.type !== "enter") stores.navigating.set(navigating$1.current = nav.navigation);
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
				state,
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
	state = popped ? popped.state : state;
	if (!popped) {
		const change = replace_state ? 0 : 1;
		const entry = {
			[HISTORY_INDEX]: current_history_index += change,
			[NAVIGATION_INDEX]: current_navigation_index += change,
			[STATES_KEY]: state
		};
		(replace_state ? history.replaceState : history.pushState).call(history, entry, "", url);
		if (!replace_state) clear_onward_history(current_history_index, current_navigation_index);
	}
	const load_cache_fork = intent && load_cache?.id === intent.id ? load_cache.fork : null;
	load_cache = null;
	navigation_result.props.page.state = state;
	let commit_promise;
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
		const fork = load_cache_fork && await load_cache_fork;
		if (fork) commit_promise = fork.commit();
		else {
			root$16.$set(navigation_result.props);
			update(navigation_result.props.page);
			commit_promise = settled?.();
		}
		has_navigated = true;
	} else await initialize(navigation_result, target, false);
	const { activeElement } = document;
	await commit_promise;
	await tick$1();
	await tick$1();
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
	if (navigation_result.props.page) Object.assign(page$3, navigation_result.props.page);
	is_navigating = false;
	if (type === "popstate") restore_snapshot(current_navigation_index);
	nav.fulfil(void 0);
	after_navigate_callbacks.forEach((fn) => fn(nav.navigation));
	stores.navigating.set(navigating$1.current = null);
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
	let current_a = {
		element: void 0,
		href: void 0
	};
	let current_priority;
	container.addEventListener("mousemove", (event) => {
		const target = event.target;
		clearTimeout(mousemove_timeout);
		mousemove_timeout = setTimeout(() => {
			preload(target, PRELOAD_PRIORITIES.hover);
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
		const interacted = a === current_a.element && a?.href === current_a.href && priority >= current_priority;
		if (!a || interacted) return;
		const { url, external, download } = get_link_info(a, base, app.hash);
		if (external || download) return;
		const options = get_router_options(a);
		const same_url = url && get_page_key(current.url) === get_page_key(url);
		if (options.reload || same_url) return;
		if (priority <= options.preload_data) {
			current_a = {
				element: a,
				href: a.href
			};
			current_priority = PRELOAD_PRIORITIES.tap;
			const intent = await get_navigation_intent(url, false);
			if (!intent) return;
			_preload_data(intent);
		} else if (priority <= options.preload_code) {
			current_a = {
				element: a,
				href: a.href
			};
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
		const { url, external, target, download } = get_link_info(a, base, app.hash);
		if (!url) return;
		if (target === "_parent" || target === "_top") {
			if (window.parent !== window) return;
		} else if (target && target !== "_self") return;
		const options = get_router_options(a);
		if (!(a instanceof SVGAElement) && url.protocol !== location.protocol && !(url.protocol === "https:" || url.protocol === "http:")) return;
		if (download) return;
		const [nonhash, hash] = (app.hash ? url.hash.replace(/^#/, "") : url.href).split("#");
		const same_pathname = nonhash === strip_hash(location);
		if (external || options.reload && (!same_pathname || !hash)) {
			if (_before_navigate({
				url,
				type: "link",
				event
			})) is_navigating = true;
			else event.preventDefault();
			return;
		}
		if (hash !== void 0 && same_pathname) {
			const [, current_hash] = current.url.href.split("#");
			if (current_hash === hash) {
				event.preventDefault();
				if (hash === "" || hash === "top" && a.ownerDocument.getElementById("top") === null) scrollTo({ top: 0 });
				else {
					const element = a.ownerDocument.getElementById(decodeURIComponent(hash));
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
			const state = event.state["sveltekit:states"] ?? {};
			const url = new URL(event.state["sveltekit:pageurl"] ?? location.href);
			const navigation_index = event.state[NAVIGATION_INDEX];
			const is_hash_change = current.url ? strip_hash(location) === strip_hash(current.url) : false;
			if (navigation_index === current_navigation_index && (has_navigated || is_hash_change)) {
				if (state !== page$3.state) page$3.state = state;
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
					state,
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
		if (event.persisted) stores.navigating.set(navigating$1.current = null);
	});
	function update_url(url) {
		current.url = page$3.url = url;
		stores.page.set(clone_page(page$3));
		stores.page.notify();
	}
}
async function _hydrate(target, { status = 200, error, node_ids, params, route, server_route, data: server_data_nodes, form }) {
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
	} catch (error) {
		if (error instanceof Redirect) {
			await native_navigation(new URL(error.location, location.href));
			return;
		}
		result = await load_root_error_page({
			status: get_status(error),
			error: await handle_error(error, {
				url,
				params,
				route
			}),
			url,
			route
		});
		target.textContent = "";
		hydrate = false;
	}
	if (result.props.page) result.props.page.state = {};
	await initialize(result, target, hydrate);
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
			const root = document.body;
			const tabindex = root.getAttribute("tabindex");
			root.tabIndex = -1;
			root.focus({
				preventScroll: true,
				focusVisible: false
			});
			if (tabindex !== null) root.setAttribute("tabindex", tabindex);
			else root.removeAttribute("tabindex");
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
function create_navigation(current, intent, url, type) {
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
				params: current.params,
				route: { id: current.route?.id ?? null },
				url: current.url
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
function clone_page(page) {
	return {
		data: page.data,
		error: page.error,
		form: page.form,
		params: page.params,
		route: page.route,
		state: page.state,
		status: page.status,
		url: page.url
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
if (typeof window !== "undefined") ((window.__svelte ??= {}).v ??= /* @__PURE__ */ new Set()).add("5");
const browser = true;
const IsClient = () => {
	return browser;
};
var apiPrd = "https://nctduhfsuy2uq5ziomgtpiabjq0ukddl.lambda-url.us-east-1.on.aws/api/";
var apiLocal = "http://localhost:3589/api/";
{
	const host = window.location.host;
	if ((host.includes("localhost") || host.includes("127.0.0.1")) && host !== "localhost:8000") globalThis._isLocal = true;
}
const getWindow = () => {
	return window;
};
const Env = {
	appId: "genix",
	S3_URL: "https://d16qwm950j0pjf.cloudfront.net/",
	serviceWorker: "/sw.js",
	enviroment: "dev",
	SIGNALING_ENDPOINT: "https://2v7tfrfxenfpvgf356vcuw3iie.appsync-api.us-east-1.amazonaws.com",
	SIGNALING_API_KEY: "da2-czyagszanfbptkl57l3t45xmo4",
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
	API_ROUTES: { MAIN: globalThis._isLocal ? apiLocal : apiPrd },
	screen: window.screen,
	language: window.navigator?.language || "",
	deviceMemory: window.navigator?.deviceMemory || 0,
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
		return window.location.pathname;
	},
	getEmpresaID: () => {
		if (!Env.empresaID) {
			const localEmpresaID = localStorage.getItem(Env.appId + "EmpresaID");
			if (localEmpresaID) {
				Env.empresaID = parseInt(localEmpresaID);
				return Env.empresaID;
			}
			let pathname = "";
			pathname = document.head.querySelector(`meta[name="loc"]`)?.getAttribute("content") || "";
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
	},
	makeRoute: (route) => {
		const api = globalThis._isLocal ? apiLocal : apiPrd;
		if (route[0] === "/") route = route.substring(1);
		const sep = route.includes("?") ? "&" : "?";
		return api + route + sep + `empresa-id=${Env.empresaID}`;
	},
	makeImageRoute: (route) => {
		if (route.substring(0, 6) !== "http//" && route.substring(0, 7) !== "https//") route = Env.S3_URL + "img-productos/" + route;
		return route;
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
const { Notify, Loading, Confirm } = (/* @__PURE__ */ __toESM((/* @__PURE__ */ __commonJSMin(((exports, module) => {
	(function(t, e) {
		"function" == typeof define && define.amd ? define([], function() {
			return e(t);
		}) : "object" == typeof module && "object" == typeof module.exports ? module.exports = e(t) : t.Notiflix = e(t);
	})("undefined" == typeof global ? "undefined" == typeof window ? exports : window : global, function(t) {
		"use strict";
		if ("undefined" == typeof t && "undefined" == typeof t.document) return !1;
		var e, i, a, n, o, r = "\n\nVisit documentation page to learn more: https://notiflix.github.io/documentation", s = "-apple-system, BlinkMacSystemFont, \"Segoe UI\", Roboto, \"Helvetica Neue\", Arial, \"Noto Sans\", sans-serif", l = {
			Success: "Success",
			Failure: "Failure",
			Warning: "Warning",
			Info: "Info"
		}, m = {
			wrapID: "NotiflixNotifyWrap",
			overlayID: "NotiflixNotifyOverlay",
			width: "280px",
			position: "right-top",
			distance: "10px",
			opacity: 1,
			borderRadius: "5px",
			rtl: !1,
			timeout: 3e3,
			messageMaxLength: 110,
			backOverlay: !1,
			backOverlayColor: "rgba(0,0,0,0.5)",
			plainText: !0,
			showOnlyTheLastOne: !1,
			clickToClose: !1,
			pauseOnHover: !0,
			ID: "NotiflixNotify",
			className: "notiflix-notify",
			zindex: 4001,
			fontFamily: "Quicksand",
			fontSize: "13px",
			cssAnimation: !0,
			cssAnimationDuration: 400,
			cssAnimationStyle: "fade",
			closeButton: !1,
			useIcon: !0,
			useFontAwesome: !1,
			fontAwesomeIconStyle: "basic",
			fontAwesomeIconSize: "34px",
			success: {
				background: "#32c682",
				textColor: "#fff",
				childClassName: "notiflix-notify-success",
				notiflixIconColor: "rgba(0,0,0,0.2)",
				fontAwesomeClassName: "fas fa-check-circle",
				fontAwesomeIconColor: "rgba(0,0,0,0.2)",
				backOverlayColor: "rgba(50,198,130,0.2)"
			},
			failure: {
				background: "#ff5549",
				textColor: "#fff",
				childClassName: "notiflix-notify-failure",
				notiflixIconColor: "rgba(0,0,0,0.2)",
				fontAwesomeClassName: "fas fa-times-circle",
				fontAwesomeIconColor: "rgba(0,0,0,0.2)",
				backOverlayColor: "rgba(255,85,73,0.2)"
			},
			warning: {
				background: "#eebf31",
				textColor: "#fff",
				childClassName: "notiflix-notify-warning",
				notiflixIconColor: "rgba(0,0,0,0.2)",
				fontAwesomeClassName: "fas fa-exclamation-circle",
				fontAwesomeIconColor: "rgba(0,0,0,0.2)",
				backOverlayColor: "rgba(238,191,49,0.2)"
			},
			info: {
				background: "#26c0d3",
				textColor: "#fff",
				childClassName: "notiflix-notify-info",
				notiflixIconColor: "rgba(0,0,0,0.2)",
				fontAwesomeClassName: "fas fa-info-circle",
				fontAwesomeIconColor: "rgba(0,0,0,0.2)",
				backOverlayColor: "rgba(38,192,211,0.2)"
			}
		}, c = {
			Success: "Success",
			Failure: "Failure",
			Warning: "Warning",
			Info: "Info"
		}, p = {
			ID: "NotiflixReportWrap",
			className: "notiflix-report",
			width: "320px",
			backgroundColor: "#f8f8f8",
			borderRadius: "25px",
			rtl: !1,
			zindex: 4002,
			backOverlay: !0,
			backOverlayColor: "rgba(0,0,0,0.5)",
			backOverlayClickToClose: !1,
			fontFamily: "Quicksand",
			svgSize: "110px",
			plainText: !0,
			titleFontSize: "16px",
			titleMaxLength: 34,
			messageFontSize: "13px",
			messageMaxLength: 400,
			buttonFontSize: "14px",
			buttonMaxLength: 34,
			cssAnimation: !0,
			cssAnimationDuration: 360,
			cssAnimationStyle: "fade",
			success: {
				svgColor: "#32c682",
				titleColor: "#1e1e1e",
				messageColor: "#242424",
				buttonBackground: "#32c682",
				buttonColor: "#fff",
				backOverlayColor: "rgba(50,198,130,0.2)"
			},
			failure: {
				svgColor: "#ff5549",
				titleColor: "#1e1e1e",
				messageColor: "#242424",
				buttonBackground: "#ff5549",
				buttonColor: "#fff",
				backOverlayColor: "rgba(255,85,73,0.2)"
			},
			warning: {
				svgColor: "#eebf31",
				titleColor: "#1e1e1e",
				messageColor: "#242424",
				buttonBackground: "#eebf31",
				buttonColor: "#fff",
				backOverlayColor: "rgba(238,191,49,0.2)"
			},
			info: {
				svgColor: "#26c0d3",
				titleColor: "#1e1e1e",
				messageColor: "#242424",
				buttonBackground: "#26c0d3",
				buttonColor: "#fff",
				backOverlayColor: "rgba(38,192,211,0.2)"
			}
		}, f = {
			Show: "Show",
			Ask: "Ask",
			Prompt: "Prompt"
		}, d = {
			ID: "NotiflixConfirmWrap",
			className: "notiflix-confirm",
			width: "300px",
			zindex: 4003,
			position: "center",
			distance: "10px",
			backgroundColor: "#f8f8f8",
			borderRadius: "25px",
			backOverlay: !0,
			backOverlayColor: "rgba(0,0,0,0.5)",
			rtl: !1,
			fontFamily: "Quicksand",
			cssAnimation: !0,
			cssAnimationDuration: 300,
			cssAnimationStyle: "fade",
			plainText: !0,
			titleColor: "#32c682",
			titleFontSize: "16px",
			titleMaxLength: 34,
			messageColor: "#1e1e1e",
			messageFontSize: "14px",
			messageMaxLength: 110,
			buttonsFontSize: "15px",
			buttonsMaxLength: 34,
			okButtonColor: "#f8f8f8",
			okButtonBackground: "#32c682",
			cancelButtonColor: "#f8f8f8",
			cancelButtonBackground: "#a9a9a9"
		}, x = {
			Standard: "Standard",
			Hourglass: "Hourglass",
			Circle: "Circle",
			Arrows: "Arrows",
			Dots: "Dots",
			Pulse: "Pulse",
			Custom: "Custom",
			Notiflix: "Notiflix"
		}, g = {
			ID: "NotiflixLoadingWrap",
			className: "notiflix-loading",
			zindex: 4e3,
			backgroundColor: "rgba(0,0,0,0.8)",
			rtl: !1,
			fontFamily: "Quicksand",
			cssAnimation: !0,
			cssAnimationDuration: 400,
			clickToClose: !1,
			customSvgUrl: null,
			customSvgCode: null,
			svgSize: "80px",
			svgColor: "#32c682",
			messageID: "NotiflixLoadingMessage",
			messageFontSize: "15px",
			messageMaxLength: 34,
			messageColor: "#dcdcdc"
		}, b = {
			Standard: "Standard",
			Hourglass: "Hourglass",
			Circle: "Circle",
			Arrows: "Arrows",
			Dots: "Dots",
			Pulse: "Pulse"
		}, u = {
			ID: "NotiflixBlockWrap",
			querySelectorLimit: 200,
			className: "notiflix-block",
			position: "absolute",
			zindex: 1e3,
			backgroundColor: "rgba(255,255,255,0.9)",
			rtl: !1,
			fontFamily: "Quicksand",
			cssAnimation: !0,
			cssAnimationDuration: 300,
			svgSize: "45px",
			svgColor: "#383838",
			messageFontSize: "14px",
			messageMaxLength: 34,
			messageColor: "#383838"
		}, y = function(t) {
			return console.error("%c Notiflix Error ", "padding:2px;border-radius:20px;color:#fff;background:#ff5549", "\n" + t + r);
		}, k = function(t) {
			return console.log("%c Notiflix Info ", "padding:2px;border-radius:20px;color:#fff;background:#26c0d3", "\n" + t + r);
		}, w = function(e) {
			return e || (e = "head"), void 0 !== t.document[e] || (y("\nNotiflix needs to be appended to the \"<" + e + ">\" element, but you called it before the \"<" + e + ">\" element has been created."), !1);
		}, h = function(e, i) {
			if (!w("head")) return !1;
			if (null !== e() && !t.document.getElementById(i)) {
				var a = t.document.createElement("style");
				a.id = i, a.innerHTML = e(), t.document.head.appendChild(a);
			}
		}, v = function() {
			var t = {}, e = !1, a = 0;
			"[object Boolean]" === Object.prototype.toString.call(arguments[0]) && (e = arguments[0], a++);
			for (var n = function(i) {
				for (var a in i) Object.prototype.hasOwnProperty.call(i, a) && (t[a] = e && "[object Object]" === Object.prototype.toString.call(i[a]) ? v(t[a], i[a]) : i[a]);
			}; a < arguments.length; a++) n(arguments[a]);
			return t;
		}, N = function(e) {
			var i = t.document.createElement("div");
			return i.innerHTML = e, i.textContent || i.innerText || "";
		}, C = function(t, e) {
			t || (t = "110px"), e || (e = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" id=\"NXReportSuccess\" width=\"" + t + "\" height=\"" + t + "\" fill=\"" + e + "\" viewBox=\"0 0 120 120\"><style>@-webkit-keyframes NXReportSuccess1-animation{0%{-webkit-transform:translate(60px,57.7px) scale(.5,.5) translate(-60px,-57.7px);transform:translate(60px,57.7px) scale(.5,.5) translate(-60px,-57.7px)}50%,to{-webkit-transform:translate(60px,57.7px) scale(1,1) translate(-60px,-57.7px);transform:translate(60px,57.7px) scale(1,1) translate(-60px,-57.7px)}60%{-webkit-transform:translate(60px,57.7px) scale(.95,.95) translate(-60px,-57.7px);transform:translate(60px,57.7px) scale(.95,.95) translate(-60px,-57.7px)}}@keyframes NXReportSuccess1-animation{0%{-webkit-transform:translate(60px,57.7px) scale(.5,.5) translate(-60px,-57.7px);transform:translate(60px,57.7px) scale(.5,.5) translate(-60px,-57.7px)}50%,to{-webkit-transform:translate(60px,57.7px) scale(1,1) translate(-60px,-57.7px);transform:translate(60px,57.7px) scale(1,1) translate(-60px,-57.7px)}60%{-webkit-transform:translate(60px,57.7px) scale(.95,.95) translate(-60px,-57.7px);transform:translate(60px,57.7px) scale(.95,.95) translate(-60px,-57.7px)}}@-webkit-keyframes NXReportSuccess4-animation{0%{opacity:0}50%,to{opacity:1}}@keyframes NXReportSuccess4-animation{0%{opacity:0}50%,to{opacity:1}}@-webkit-keyframes NXReportSuccess3-animation{0%{opacity:0}40%,to{opacity:1}}@keyframes NXReportSuccess3-animation{0%{opacity:0}40%,to{opacity:1}}@-webkit-keyframes NXReportSuccess2-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@keyframes NXReportSuccess2-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}#NXReportSuccess *{-webkit-animation-duration:1.2s;animation-duration:1.2s;-webkit-animation-timing-function:cubic-bezier(0,0,1,1);animation-timing-function:cubic-bezier(0,0,1,1)}</style><g style=\"-webkit-animation-name:NXReportSuccess2-animation;animation-name:NXReportSuccess2-animation;-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\"><path d=\"M60 115.38C29.46 115.38 4.62 90.54 4.62 60 4.62 29.46 29.46 4.62 60 4.62c30.54 0 55.38 24.84 55.38 55.38 0 30.54-24.84 55.38-55.38 55.38zM60 0C26.92 0 0 26.92 0 60s26.92 60 60 60 60-26.92 60-60S93.08 0 60 0z\" style=\"-webkit-animation-name:NXReportSuccess3-animation;animation-name:NXReportSuccess3-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g><g style=\"-webkit-animation-name:NXReportSuccess1-animation;animation-name:NXReportSuccess1-animation;-webkit-transform:translate(60px,57.7px) scale(1,1) translate(-60px,-57.7px);-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\"><path d=\"M88.27 35.39L52.8 75.29 31.43 58.2c-.98-.81-2.44-.63-3.24.36-.79.99-.63 2.44.36 3.24l23.08 18.46c.43.34.93.51 1.44.51.64 0 1.27-.26 1.74-.78l36.91-41.53a2.3 2.3 0 0 0-.19-3.26c-.95-.86-2.41-.77-3.26.19z\" style=\"-webkit-animation-name:NXReportSuccess4-animation;animation-name:NXReportSuccess4-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g></svg>";
		}, z = function(t, e) {
			t || (t = "110px"), e || (e = "#ff5549");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" id=\"NXReportFailure\" width=\"" + t + "\" height=\"" + t + "\" fill=\"" + e + "\" viewBox=\"0 0 120 120\"><style>@-webkit-keyframes NXReportFailure2-animation{0%{opacity:0}40%,to{opacity:1}}@keyframes NXReportFailure2-animation{0%{opacity:0}40%,to{opacity:1}}@-webkit-keyframes NXReportFailure1-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@keyframes NXReportFailure1-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@-webkit-keyframes NXReportFailure3-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}50%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@keyframes NXReportFailure3-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}50%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@-webkit-keyframes NXReportFailure4-animation{0%{opacity:0}50%,to{opacity:1}}@keyframes NXReportFailure4-animation{0%{opacity:0}50%,to{opacity:1}}#NXReportFailure *{-webkit-animation-duration:1.2s;animation-duration:1.2s;-webkit-animation-timing-function:cubic-bezier(0,0,1,1);animation-timing-function:cubic-bezier(0,0,1,1)}</style><g style=\"-webkit-animation-name:NXReportFailure1-animation;animation-name:NXReportFailure1-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1);-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)\"><path d=\"M4.35 34.95c0-16.82 13.78-30.6 30.6-30.6h50.1c16.82 0 30.6 13.78 30.6 30.6v50.1c0 16.82-13.78 30.6-30.6 30.6h-50.1c-16.82 0-30.6-13.78-30.6-30.6v-50.1zM34.95 120h50.1c19.22 0 34.95-15.73 34.95-34.95v-50.1C120 15.73 104.27 0 85.05 0h-50.1C15.73 0 0 15.73 0 34.95v50.1C0 104.27 15.73 120 34.95 120z\" style=\"-webkit-animation-name:NXReportFailure2-animation;animation-name:NXReportFailure2-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g><g style=\"-webkit-animation-name:NXReportFailure3-animation;animation-name:NXReportFailure3-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1);-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)\"><path d=\"M82.4 37.6c-.9-.9-2.37-.9-3.27 0L60 56.73 40.86 37.6a2.306 2.306 0 0 0-3.26 3.26L56.73 60 37.6 79.13c-.9.9-.9 2.37 0 3.27.45.45 1.04.68 1.63.68.59 0 1.18-.23 1.63-.68L60 63.26 79.13 82.4c.45.45 1.05.68 1.64.68.58 0 1.18-.23 1.63-.68.9-.9.9-2.37 0-3.27L63.26 60 82.4 40.86c.9-.91.9-2.36 0-3.26z\" style=\"-webkit-animation-name:NXReportFailure4-animation;animation-name:NXReportFailure4-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g></svg>";
		}, S = function(t, e) {
			t || (t = "110px"), e || (e = "#eebf31");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" id=\"NXReportWarning\" width=\"" + t + "\" height=\"" + t + "\" fill=\"" + e + "\" viewBox=\"0 0 120 120\"><style>@-webkit-keyframes NXReportWarning2-animation{0%{opacity:0}40%,to{opacity:1}}@keyframes NXReportWarning2-animation{0%{opacity:0}40%,to{opacity:1}}@-webkit-keyframes NXReportWarning1-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@keyframes NXReportWarning1-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@-webkit-keyframes NXReportWarning3-animation{0%{-webkit-transform:translate(60px,66.6px) scale(.5,.5) translate(-60px,-66.6px);transform:translate(60px,66.6px) scale(.5,.5) translate(-60px,-66.6px)}50%,to{-webkit-transform:translate(60px,66.6px) scale(1,1) translate(-60px,-66.6px);transform:translate(60px,66.6px) scale(1,1) translate(-60px,-66.6px)}60%{-webkit-transform:translate(60px,66.6px) scale(.95,.95) translate(-60px,-66.6px);transform:translate(60px,66.6px) scale(.95,.95) translate(-60px,-66.6px)}}@keyframes NXReportWarning3-animation{0%{-webkit-transform:translate(60px,66.6px) scale(.5,.5) translate(-60px,-66.6px);transform:translate(60px,66.6px) scale(.5,.5) translate(-60px,-66.6px)}50%,to{-webkit-transform:translate(60px,66.6px) scale(1,1) translate(-60px,-66.6px);transform:translate(60px,66.6px) scale(1,1) translate(-60px,-66.6px)}60%{-webkit-transform:translate(60px,66.6px) scale(.95,.95) translate(-60px,-66.6px);transform:translate(60px,66.6px) scale(.95,.95) translate(-60px,-66.6px)}}@-webkit-keyframes NXReportWarning4-animation{0%{opacity:0}50%,to{opacity:1}}@keyframes NXReportWarning4-animation{0%{opacity:0}50%,to{opacity:1}}#NXReportWarning *{-webkit-animation-duration:1.2s;animation-duration:1.2s;-webkit-animation-timing-function:cubic-bezier(0,0,1,1);animation-timing-function:cubic-bezier(0,0,1,1)}</style><g style=\"-webkit-animation-name:NXReportWarning1-animation;animation-name:NXReportWarning1-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1);-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)\"><path d=\"M115.46 106.15l-54.04-93.8c-.61-1.06-2.23-1.06-2.84 0l-54.04 93.8c-.62 1.07.21 2.29 1.42 2.29h108.08c1.21 0 2.04-1.22 1.42-2.29zM65.17 10.2l54.04 93.8c2.28 3.96-.65 8.78-5.17 8.78H5.96c-4.52 0-7.45-4.82-5.17-8.78l54.04-93.8c2.28-3.95 8.03-4 10.34 0z\" style=\"-webkit-animation-name:NXReportWarning2-animation;animation-name:NXReportWarning2-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g><g style=\"-webkit-animation-name:NXReportWarning3-animation;animation-name:NXReportWarning3-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1);-webkit-transform:translate(60px,66.6px) scale(1,1) translate(-60px,-66.6px)\"><path d=\"M57.83 94.01c0 1.2.97 2.17 2.17 2.17s2.17-.97 2.17-2.17v-3.2c0-1.2-.97-2.17-2.17-2.17s-2.17.97-2.17 2.17v3.2zm0-14.15c0 1.2.97 2.17 2.17 2.17s2.17-.97 2.17-2.17V39.21c0-1.2-.97-2.17-2.17-2.17s-2.17.97-2.17 2.17v40.65z\" style=\"-webkit-animation-name:NXReportWarning4-animation;animation-name:NXReportWarning4-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g></svg>";
		}, L = function(t, e) {
			t || (t = "110px"), e || (e = "#26c0d3");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" id=\"NXReportInfo\" width=\"" + t + "\" height=\"" + t + "\" fill=\"" + e + "\" viewBox=\"0 0 120 120\"><style>@-webkit-keyframes NXReportInfo4-animation{0%{opacity:0}50%,to{opacity:1}}@keyframes NXReportInfo4-animation{0%{opacity:0}50%,to{opacity:1}}@-webkit-keyframes NXReportInfo3-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}50%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@keyframes NXReportInfo3-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}50%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@-webkit-keyframes NXReportInfo2-animation{0%{opacity:0}40%,to{opacity:1}}@keyframes NXReportInfo2-animation{0%{opacity:0}40%,to{opacity:1}}@-webkit-keyframes NXReportInfo1-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@keyframes NXReportInfo1-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}#NXReportInfo *{-webkit-animation-duration:1.2s;animation-duration:1.2s;-webkit-animation-timing-function:cubic-bezier(0,0,1,1);animation-timing-function:cubic-bezier(0,0,1,1)}</style><g style=\"-webkit-animation-name:NXReportInfo1-animation;animation-name:NXReportInfo1-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1);-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)\"><path d=\"M60 115.38C29.46 115.38 4.62 90.54 4.62 60 4.62 29.46 29.46 4.62 60 4.62c30.54 0 55.38 24.84 55.38 55.38 0 30.54-24.84 55.38-55.38 55.38zM60 0C26.92 0 0 26.92 0 60s26.92 60 60 60 60-26.92 60-60S93.08 0 60 0z\" style=\"-webkit-animation-name:NXReportInfo2-animation;animation-name:NXReportInfo2-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g><g style=\"-webkit-animation-name:NXReportInfo3-animation;animation-name:NXReportInfo3-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1);-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)\"><path d=\"M57.75 43.85c0-1.24 1.01-2.25 2.25-2.25s2.25 1.01 2.25 2.25v48.18c0 1.24-1.01 2.25-2.25 2.25s-2.25-1.01-2.25-2.25V43.85zm0-15.88c0-1.24 1.01-2.25 2.25-2.25s2.25 1.01 2.25 2.25v3.32c0 1.25-1.01 2.25-2.25 2.25s-2.25-1-2.25-2.25v-3.32z\" style=\"-webkit-animation-name:NXReportInfo4-animation;animation-name:NXReportInfo4-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g></svg>";
		}, W = function(t, e) {
			t || (t = "60px"), e || (e = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" stroke=\"" + e + "\" width=\"" + t + "\" height=\"" + t + "\" transform=\"scale(.8)\" viewBox=\"0 0 38 38\"><g fill=\"none\" fill-rule=\"evenodd\" stroke-width=\"2\" transform=\"translate(1 1)\"><circle cx=\"18\" cy=\"18\" r=\"18\" stroke-opacity=\".25\"/><path d=\"M36 18c0-9.94-8.06-18-18-18\"><animateTransform attributeName=\"transform\" dur=\"1s\" from=\"0 18 18\" repeatCount=\"indefinite\" to=\"360 18 18\" type=\"rotate\"/></path></g></svg>";
		}, I = function(t, e) {
			t || (t = "60px"), e || (e = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" id=\"NXLoadingHourglass\" fill=\"" + e + "\" width=\"" + t + "\" height=\"" + t + "\" viewBox=\"0 0 200 200\"><style>@-webkit-keyframes NXhourglass5-animation{0%{-webkit-transform:scale(1,1);transform:scale(1,1)}16.67%{-webkit-transform:scale(1,.8);transform:scale(1,.8)}33.33%{-webkit-transform:scale(.88,.6);transform:scale(.88,.6)}37.5%{-webkit-transform:scale(.85,.55);transform:scale(.85,.55)}41.67%{-webkit-transform:scale(.8,.5);transform:scale(.8,.5)}45.83%{-webkit-transform:scale(.75,.45);transform:scale(.75,.45)}50%{-webkit-transform:scale(.7,.4);transform:scale(.7,.4)}54.17%{-webkit-transform:scale(.6,.35);transform:scale(.6,.35)}58.33%{-webkit-transform:scale(.5,.3);transform:scale(.5,.3)}83.33%,to{-webkit-transform:scale(.2,0);transform:scale(.2,0)}}@keyframes NXhourglass5-animation{0%{-webkit-transform:scale(1,1);transform:scale(1,1)}16.67%{-webkit-transform:scale(1,.8);transform:scale(1,.8)}33.33%{-webkit-transform:scale(.88,.6);transform:scale(.88,.6)}37.5%{-webkit-transform:scale(.85,.55);transform:scale(.85,.55)}41.67%{-webkit-transform:scale(.8,.5);transform:scale(.8,.5)}45.83%{-webkit-transform:scale(.75,.45);transform:scale(.75,.45)}50%{-webkit-transform:scale(.7,.4);transform:scale(.7,.4)}54.17%{-webkit-transform:scale(.6,.35);transform:scale(.6,.35)}58.33%{-webkit-transform:scale(.5,.3);transform:scale(.5,.3)}83.33%,to{-webkit-transform:scale(.2,0);transform:scale(.2,0)}}@-webkit-keyframes NXhourglass3-animation{0%{-webkit-transform:scale(1,.02);transform:scale(1,.02)}79.17%,to{-webkit-transform:scale(1,1);transform:scale(1,1)}}@keyframes NXhourglass3-animation{0%{-webkit-transform:scale(1,.02);transform:scale(1,.02)}79.17%,to{-webkit-transform:scale(1,1);transform:scale(1,1)}}@-webkit-keyframes NXhourglass1-animation{0%,83.33%{-webkit-transform:rotate(0deg);transform:rotate(0deg)}to{-webkit-transform:rotate(180deg);transform:rotate(180deg)}}@keyframes NXhourglass1-animation{0%,83.33%{-webkit-transform:rotate(0deg);transform:rotate(0deg)}to{-webkit-transform:rotate(180deg);transform:rotate(180deg)}}#NXLoadingHourglass *{-webkit-animation-duration:1.2s;animation-duration:1.2s;-webkit-animation-iteration-count:infinite;animation-iteration-count:infinite;-webkit-animation-timing-function:cubic-bezier(0,0,1,1);animation-timing-function:cubic-bezier(0,0,1,1)}</style><g data-animator-group=\"true\" data-animator-type=\"1\" style=\"-webkit-animation-name:NXhourglass1-animation;animation-name:NXhourglass1-animation;-webkit-transform-origin:50% 50%;transform-origin:50% 50%;transform-box:fill-box\"><g id=\"NXhourglass2\" fill=\"inherit\"><g data-animator-group=\"true\" data-animator-type=\"2\" style=\"-webkit-animation-name:NXhourglass3-animation;animation-name:NXhourglass3-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1);-webkit-transform-origin:50% 100%;transform-origin:50% 100%;transform-box:fill-box\" opacity=\".4\"><path id=\"NXhourglass4\" d=\"M100 100l-34.38 32.08v31.14h68.76v-31.14z\"/></g><g data-animator-group=\"true\" data-animator-type=\"2\" style=\"-webkit-animation-name:NXhourglass5-animation;animation-name:NXhourglass5-animation;-webkit-transform-origin:50% 100%;transform-origin:50% 100%;transform-box:fill-box\" opacity=\".4\"><path id=\"NXhourglass6\" d=\"M100 100L65.62 67.92V36.78h68.76v31.14z\"/></g><path d=\"M51.14 38.89h8.33v14.93c0 15.1 8.29 28.99 23.34 39.1 1.88 1.25 3.04 3.97 3.04 7.08s-1.16 5.83-3.04 7.09c-15.05 10.1-23.34 23.99-23.34 39.09v14.93h-8.33a4.859 4.859 0 1 0 0 9.72h97.72a4.859 4.859 0 1 0 0-9.72h-8.33v-14.93c0-15.1-8.29-28.99-23.34-39.09-1.88-1.26-3.04-3.98-3.04-7.09s1.16-5.83 3.04-7.08c15.05-10.11 23.34-24 23.34-39.1V38.89h8.33a4.859 4.859 0 1 0 0-9.72H51.14a4.859 4.859 0 1 0 0 9.72zm79.67 14.93c0 15.87-11.93 26.25-19.04 31.03-4.6 3.08-7.34 8.75-7.34 15.15 0 6.41 2.74 12.07 7.34 15.15 7.11 4.78 19.04 15.16 19.04 31.03v14.93H69.19v-14.93c0-15.87 11.93-26.25 19.04-31.02 4.6-3.09 7.34-8.75 7.34-15.16 0-6.4-2.74-12.07-7.34-15.15-7.11-4.78-19.04-15.16-19.04-31.03V38.89h61.62v14.93z\"/></g></g></svg>";
		}, R = function(t, e) {
			t || (t = "60px"), e || (e = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"" + t + "\" height=\"" + t + "\" viewBox=\"25 25 50 50\" style=\"-webkit-animation:rotate 2s linear infinite;animation:rotate 2s linear infinite;height:" + t + ";-webkit-transform-origin:center center;-ms-transform-origin:center center;transform-origin:center center;width:" + t + ";position:absolute;top:0;left:0;margin:auto\"><style>@-webkit-keyframes rotate{to{-webkit-transform:rotate(360deg);transform:rotate(360deg)}}@keyframes rotate{to{-webkit-transform:rotate(360deg);transform:rotate(360deg)}}@-webkit-keyframes dash{0%{stroke-dasharray:1,200;stroke-dashoffset:0}50%{stroke-dasharray:89,200;stroke-dashoffset:-35}to{stroke-dasharray:89,200;stroke-dashoffset:-124}}@keyframes dash{0%{stroke-dasharray:1,200;stroke-dashoffset:0}50%{stroke-dasharray:89,200;stroke-dashoffset:-35}to{stroke-dasharray:89,200;stroke-dashoffset:-124}}</style><circle cx=\"50\" cy=\"50\" r=\"20\" fill=\"none\" stroke=\"" + e + "\" stroke-width=\"2\" style=\"-webkit-animation:dash 1.5s ease-in-out infinite,color 1.5s ease-in-out infinite;animation:dash 1.5s ease-in-out infinite,color 1.5s ease-in-out infinite\" stroke-dasharray=\"150 200\" stroke-dashoffset=\"-10\" stroke-linecap=\"round\"/></svg>";
		}, A = function(t, e) {
			t || (t = "60px"), e || (e = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" fill=\"" + e + "\" width=\"" + t + "\" height=\"" + t + "\" viewBox=\"0 0 128 128\"><g><path fill=\"inherit\" d=\"M109.25 55.5h-36l12-12a29.54 29.54 0 0 0-49.53 12H18.75A46.04 46.04 0 0 1 96.9 31.84l12.35-12.34v36zm-90.5 17h36l-12 12a29.54 29.54 0 0 0 49.53-12h16.97A46.04 46.04 0 0 1 31.1 96.16L18.74 108.5v-36z\"/><animateTransform attributeName=\"transform\" dur=\"1.5s\" from=\"0 64 64\" repeatCount=\"indefinite\" to=\"360 64 64\" type=\"rotate\"/></g></svg>";
		}, M = function(t, e) {
			t || (t = "60px"), e || (e = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" fill=\"" + e + "\" width=\"" + t + "\" height=\"" + t + "\" viewBox=\"0 0 100 100\"><g transform=\"translate(25 50)\"><circle r=\"9\" fill=\"inherit\" transform=\"scale(.239)\"><animateTransform attributeName=\"transform\" begin=\"-0.266s\" calcMode=\"spline\" dur=\"0.8s\" keySplines=\"0.3 0 0.7 1;0.3 0 0.7 1\" keyTimes=\"0;0.5;1\" repeatCount=\"indefinite\" type=\"scale\" values=\"0;1;0\"/></circle></g><g transform=\"translate(50 50)\"><circle r=\"9\" fill=\"inherit\" transform=\"scale(.00152)\"><animateTransform attributeName=\"transform\" begin=\"-0.133s\" calcMode=\"spline\" dur=\"0.8s\" keySplines=\"0.3 0 0.7 1;0.3 0 0.7 1\" keyTimes=\"0;0.5;1\" repeatCount=\"indefinite\" type=\"scale\" values=\"0;1;0\"/></circle></g><g transform=\"translate(75 50)\"><circle r=\"9\" fill=\"inherit\" transform=\"scale(.299)\"><animateTransform attributeName=\"transform\" begin=\"0s\" calcMode=\"spline\" dur=\"0.8s\" keySplines=\"0.3 0 0.7 1;0.3 0 0.7 1\" keyTimes=\"0;0.5;1\" repeatCount=\"indefinite\" type=\"scale\" values=\"0;1;0\"/></circle></g></svg>";
		}, B = function(t, e) {
			t || (t = "60px"), e || (e = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" stroke=\"" + e + "\" width=\"" + t + "\" height=\"" + t + "\" viewBox=\"0 0 44 44\"><g fill=\"none\" fill-rule=\"evenodd\" stroke-width=\"2\"><circle cx=\"22\" cy=\"22\" r=\"1\"><animate attributeName=\"r\" begin=\"0s\" calcMode=\"spline\" dur=\"1.8s\" keySplines=\"0.165, 0.84, 0.44, 1\" keyTimes=\"0; 1\" repeatCount=\"indefinite\" values=\"1; 20\"/><animate attributeName=\"stroke-opacity\" begin=\"0s\" calcMode=\"spline\" dur=\"1.8s\" keySplines=\"0.3, 0.61, 0.355, 1\" keyTimes=\"0; 1\" repeatCount=\"indefinite\" values=\"1; 0\"/></circle><circle cx=\"22\" cy=\"22\" r=\"1\"><animate attributeName=\"r\" begin=\"-0.9s\" calcMode=\"spline\" dur=\"1.8s\" keySplines=\"0.165, 0.84, 0.44, 1\" keyTimes=\"0; 1\" repeatCount=\"indefinite\" values=\"1; 20\"/><animate attributeName=\"stroke-opacity\" begin=\"-0.9s\" calcMode=\"spline\" dur=\"1.8s\" keySplines=\"0.3, 0.61, 0.355, 1\" keyTimes=\"0; 1\" repeatCount=\"indefinite\" values=\"1; 0\"/></circle></g></svg>";
		}, X = function(t, e, i) {
			t || (t = "60px"), e || (e = "#f8f8f8"), i || (i = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" id=\"NXLoadingNotiflixLib\" width=\"" + t + "\" height=\"" + t + "\" viewBox=\"0 0 200 200\"><defs><style>@keyframes notiflix-n{0%{stroke-dashoffset:1000}to{stroke-dashoffset:0}}@keyframes notiflix-x{0%{stroke-dashoffset:1000}to{stroke-dashoffset:0}}@keyframes notiflix-dot{0%,to{stroke-width:0}50%{stroke-width:12}}.nx-icon-line{stroke:" + e + ";stroke-width:12;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:22;fill:none}</style></defs><path d=\"M47.97 135.05a6.5 6.5 0 1 1 0 13 6.5 6.5 0 0 1 0-13z\" style=\"animation-name:notiflix-dot;animation-timing-function:ease-in-out;animation-duration:1.25s;animation-iteration-count:infinite;animation-direction:normal\" fill=\"" + i + "\" stroke=\"" + i + "\" stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-miterlimit=\"22\" stroke-width=\"12\"/><path class=\"nx-icon-line\" d=\"M10.14 144.76V87.55c0-5.68-4.54-41.36 37.83-41.36 42.36 0 37.82 35.68 37.82 41.36v57.21\" style=\"animation-name:notiflix-n;animation-timing-function:linear;animation-duration:2.5s;animation-delay:0s;animation-iteration-count:infinite;animation-direction:normal\" stroke-dasharray=\"500\"/><path class=\"nx-icon-line\" d=\"M115.06 144.49c24.98-32.68 49.96-65.35 74.94-98.03M114.89 46.6c25.09 32.58 50.19 65.17 75.29 97.75\" style=\"animation-name:notiflix-x;animation-timing-function:linear;animation-duration:2.5s;animation-delay:.2s;animation-iteration-count:infinite;animation-direction:normal\" stroke-dasharray=\"500\"/></svg>";
		}, D = function() {
			return "[id^=NotiflixNotifyWrap]{pointer-events:none;position:fixed;z-index:4001;opacity:1;right:10px;top:10px;width:280px;max-width:96%;-webkit-box-sizing:border-box;box-sizing:border-box;background:transparent}[id^=NotiflixNotifyWrap].nx-flex-center-center{max-height:calc(100vh - 20px);overflow-x:hidden;overflow-y:auto;display:-webkit-box;display:-webkit-flex;display:-ms-flexbox;display:flex;-webkit-flex-wrap:wrap;-ms-flex-wrap:wrap;flex-wrap:wrap;-webkit-box-orient:vertical;-webkit-box-direction:normal;-webkit-flex-direction:column;-ms-flex-direction:column;flex-direction:column;-webkit-box-pack:center;-webkit-justify-content:center;-ms-flex-pack:center;justify-content:center;-webkit-box-align:center;-webkit-align-items:center;-ms-flex-align:center;align-items:center;margin:auto}[id^=NotiflixNotifyWrap]::-webkit-scrollbar{width:0;height:0}[id^=NotiflixNotifyWrap]::-webkit-scrollbar-thumb{background:transparent}[id^=NotiflixNotifyWrap]::-webkit-scrollbar-track{background:transparent}[id^=NotiflixNotifyWrap] *{-webkit-box-sizing:border-box;box-sizing:border-box}[id^=NotiflixNotifyOverlay]{-webkit-transition:background .3s ease-in-out;-o-transition:background .3s ease-in-out;transition:background .3s ease-in-out}[id^=NotiflixNotifyWrap]>div{pointer-events:all;-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;font-family:\"Quicksand\",-apple-system,BlinkMacSystemFont,\"Segoe UI\",Roboto,\"Helvetica Neue\",Arial,sans-serif;width:100%;display:-webkit-inline-box;display:-webkit-inline-flex;display:-ms-inline-flexbox;display:inline-flex;-webkit-flex-wrap:wrap;-ms-flex-wrap:wrap;flex-wrap:wrap;-webkit-box-align:center;-webkit-align-items:center;-ms-flex-align:center;align-items:center;position:relative;margin:0 0 10px;border-radius:5px;background:#1e1e1e;color:#fff;padding:10px 12px;font-size:14px;line-height:1.4}[id^=NotiflixNotifyWrap]>div:last-child{margin:0}[id^=NotiflixNotifyWrap]>div.nx-with-callback{cursor:pointer}[id^=NotiflixNotifyWrap]>div.nx-with-icon{padding:8px;min-height:56px}[id^=NotiflixNotifyWrap]>div.nx-paused{cursor:auto}[id^=NotiflixNotifyWrap]>div.nx-notify-click-to-close{cursor:pointer}[id^=NotiflixNotifyWrap]>div.nx-with-close-button{padding:10px 36px 10px 12px}[id^=NotiflixNotifyWrap]>div.nx-with-icon.nx-with-close-button{padding:6px 36px 6px 6px}[id^=NotiflixNotifyWrap]>div>span.nx-message{cursor:inherit;font-weight:normal;font-family:inherit!important;word-break:break-all;word-break:break-word}[id^=NotiflixNotifyWrap]>div>span.nx-close-button{cursor:pointer;-webkit-transition:all .2s ease-in-out;-o-transition:all .2s ease-in-out;transition:all .2s ease-in-out;position:absolute;right:8px;top:0;bottom:0;margin:auto;color:inherit;width:20px;height:20px}[id^=NotiflixNotifyWrap]>div>span.nx-close-button:hover{-webkit-transform:rotate(90deg);transform:rotate(90deg)}[id^=NotiflixNotifyWrap]>div>span.nx-close-button>svg{position:absolute;width:16px;height:16px;right:2px;top:2px}[id^=NotiflixNotifyWrap]>div>.nx-message-icon{position:absolute;width:40px;height:40px;font-size:30px;line-height:40px;text-align:center;left:8px;top:0;bottom:0;margin:auto;border-radius:inherit}[id^=NotiflixNotifyWrap]>div>.nx-message-icon-fa.nx-message-icon-fa-shadow{color:inherit;background:rgba(0,0,0,.15);-webkit-box-shadow:inset 0 0 34px rgba(0,0,0,.2);box-shadow:inset 0 0 34px rgba(0,0,0,.2);text-shadow:0 0 10px rgba(0,0,0,.3)}[id^=NotiflixNotifyWrap]>div>span.nx-with-icon{position:relative;float:left;width:calc(100% - 40px);margin:0 0 0 40px;padding:0 0 0 10px;-webkit-box-sizing:border-box;box-sizing:border-box}[id^=NotiflixNotifyWrap]>div.nx-rtl-on>.nx-message-icon{left:auto;right:8px}[id^=NotiflixNotifyWrap]>div.nx-rtl-on>span.nx-with-icon{padding:0 10px 0 0;margin:0 40px 0 0}[id^=NotiflixNotifyWrap]>div.nx-rtl-on>span.nx-close-button{right:auto;left:8px}[id^=NotiflixNotifyWrap]>div.nx-with-icon.nx-with-close-button.nx-rtl-on{padding:6px 6px 6px 36px}[id^=NotiflixNotifyWrap]>div.nx-with-close-button.nx-rtl-on{padding:10px 12px 10px 36px}[id^=NotiflixNotifyOverlay].nx-with-animation,[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-fade{-webkit-animation:notify-animation-fade .3s ease-in-out 0s normal;animation:notify-animation-fade .3s ease-in-out 0s normal}@-webkit-keyframes notify-animation-fade{0%{opacity:0}100%{opacity:1}}@keyframes notify-animation-fade{0%{opacity:0}100%{opacity:1}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-zoom{-webkit-animation:notify-animation-zoom .3s ease-in-out 0s normal;animation:notify-animation-zoom .3s ease-in-out 0s normal}@-webkit-keyframes notify-animation-zoom{0%{-webkit-transform:scale(0);transform:scale(0)}50%{-webkit-transform:scale(1.05);transform:scale(1.05)}100%{-webkit-transform:scale(1);transform:scale(1)}}@keyframes notify-animation-zoom{0%{-webkit-transform:scale(0);transform:scale(0)}50%{-webkit-transform:scale(1.05);transform:scale(1.05)}100%{-webkit-transform:scale(1);transform:scale(1)}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-right{-webkit-animation:notify-animation-from-right .3s ease-in-out 0s normal;animation:notify-animation-from-right .3s ease-in-out 0s normal}@-webkit-keyframes notify-animation-from-right{0%{right:-300px;opacity:0}50%{right:8px;opacity:1}100%{right:0;opacity:1}}@keyframes notify-animation-from-right{0%{right:-300px;opacity:0}50%{right:8px;opacity:1}100%{right:0;opacity:1}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-left{-webkit-animation:notify-animation-from-left .3s ease-in-out 0s normal;animation:notify-animation-from-left .3s ease-in-out 0s normal}@-webkit-keyframes notify-animation-from-left{0%{left:-300px;opacity:0}50%{left:8px;opacity:1}100%{left:0;opacity:1}}@keyframes notify-animation-from-left{0%{left:-300px;opacity:0}50%{left:8px;opacity:1}100%{left:0;opacity:1}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-top{-webkit-animation:notify-animation-from-top .3s ease-in-out 0s normal;animation:notify-animation-from-top .3s ease-in-out 0s normal}@-webkit-keyframes notify-animation-from-top{0%{top:-50px;opacity:0}50%{top:8px;opacity:1}100%{top:0;opacity:1}}@keyframes notify-animation-from-top{0%{top:-50px;opacity:0}50%{top:8px;opacity:1}100%{top:0;opacity:1}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-bottom{-webkit-animation:notify-animation-from-bottom .3s ease-in-out 0s normal;animation:notify-animation-from-bottom .3s ease-in-out 0s normal}@-webkit-keyframes notify-animation-from-bottom{0%{bottom:-50px;opacity:0}50%{bottom:8px;opacity:1}100%{bottom:0;opacity:1}}@keyframes notify-animation-from-bottom{0%{bottom:-50px;opacity:0}50%{bottom:8px;opacity:1}100%{bottom:0;opacity:1}}[id^=NotiflixNotifyOverlay].nx-with-animation.nx-remove,[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-fade.nx-remove{opacity:0;-webkit-animation:notify-remove-fade .3s ease-in-out 0s normal;animation:notify-remove-fade .3s ease-in-out 0s normal}@-webkit-keyframes notify-remove-fade{0%{opacity:1}100%{opacity:0}}@keyframes notify-remove-fade{0%{opacity:1}100%{opacity:0}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-zoom.nx-remove{-webkit-transform:scale(0);transform:scale(0);-webkit-animation:notify-remove-zoom .3s ease-in-out 0s normal;animation:notify-remove-zoom .3s ease-in-out 0s normal}@-webkit-keyframes notify-remove-zoom{0%{-webkit-transform:scale(1);transform:scale(1)}50%{-webkit-transform:scale(1.05);transform:scale(1.05)}100%{-webkit-transform:scale(0);transform:scale(0)}}@keyframes notify-remove-zoom{0%{-webkit-transform:scale(1);transform:scale(1)}50%{-webkit-transform:scale(1.05);transform:scale(1.05)}100%{-webkit-transform:scale(0);transform:scale(0)}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-top.nx-remove{opacity:0;-webkit-animation:notify-remove-to-top .3s ease-in-out 0s normal;animation:notify-remove-to-top .3s ease-in-out 0s normal}@-webkit-keyframes notify-remove-to-top{0%{top:0;opacity:1}50%{top:8px;opacity:1}100%{top:-50px;opacity:0}}@keyframes notify-remove-to-top{0%{top:0;opacity:1}50%{top:8px;opacity:1}100%{top:-50px;opacity:0}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-right.nx-remove{opacity:0;-webkit-animation:notify-remove-to-right .3s ease-in-out 0s normal;animation:notify-remove-to-right .3s ease-in-out 0s normal}@-webkit-keyframes notify-remove-to-right{0%{right:0;opacity:1}50%{right:8px;opacity:1}100%{right:-300px;opacity:0}}@keyframes notify-remove-to-right{0%{right:0;opacity:1}50%{right:8px;opacity:1}100%{right:-300px;opacity:0}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-bottom.nx-remove{opacity:0;-webkit-animation:notify-remove-to-bottom .3s ease-in-out 0s normal;animation:notify-remove-to-bottom .3s ease-in-out 0s normal}@-webkit-keyframes notify-remove-to-bottom{0%{bottom:0;opacity:1}50%{bottom:8px;opacity:1}100%{bottom:-50px;opacity:0}}@keyframes notify-remove-to-bottom{0%{bottom:0;opacity:1}50%{bottom:8px;opacity:1}100%{bottom:-50px;opacity:0}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-left.nx-remove{opacity:0;-webkit-animation:notify-remove-to-left .3s ease-in-out 0s normal;animation:notify-remove-to-left .3s ease-in-out 0s normal}@-webkit-keyframes notify-remove-to-left{0%{left:0;opacity:1}50%{left:8px;opacity:1}100%{left:-300px;opacity:0}}@keyframes notify-remove-to-left{0%{left:0;opacity:1}50%{left:8px;opacity:1}100%{left:-300px;opacity:0}}";
		}, T = 0, F = function(a, n, o, r) {
			if (!w("body")) return !1;
			e || G.Notify.init({});
			var c = v(!0, e, {});
			if ("object" == typeof o && !Array.isArray(o) || "object" == typeof r && !Array.isArray(r)) {
				var p = {};
				"object" == typeof o ? p = o : "object" == typeof r && (p = r), e = v(!0, e, p);
			}
			var f = e[a.toLocaleLowerCase("en")];
			T++, "string" != typeof n && (n = "Notiflix " + a), e.plainText && (n = N(n)), !e.plainText && n.length > e.messageMaxLength && (e = v(!0, e, {
				closeButton: !0,
				messageMaxLength: 150
			}), n = "Possible HTML Tags Error: The \"plainText\" option is \"false\" and the notification content length is more than the \"messageMaxLength\" option."), n.length > e.messageMaxLength && (n = n.substring(0, e.messageMaxLength) + "..."), "shadow" === e.fontAwesomeIconStyle && (f.fontAwesomeIconColor = f.background), e.cssAnimation || (e.cssAnimationDuration = 0);
			var d = t.document.getElementById(m.wrapID) || t.document.createElement("div");
			if (d.id = m.wrapID, d.style.width = e.width, d.style.zIndex = e.zindex, d.style.opacity = e.opacity, "center-center" === e.position ? (d.style.left = e.distance, d.style.top = e.distance, d.style.right = e.distance, d.style.bottom = e.distance, d.style.margin = "auto", d.classList.add("nx-flex-center-center"), d.style.maxHeight = "calc((100vh - " + e.distance + ") - " + e.distance + ")", d.style.display = "flex", d.style.flexWrap = "wrap", d.style.flexDirection = "column", d.style.justifyContent = "center", d.style.alignItems = "center", d.style.pointerEvents = "none") : "center-top" === e.position ? (d.style.left = e.distance, d.style.right = e.distance, d.style.top = e.distance, d.style.bottom = "auto", d.style.margin = "auto") : "center-bottom" === e.position ? (d.style.left = e.distance, d.style.right = e.distance, d.style.bottom = e.distance, d.style.top = "auto", d.style.margin = "auto") : "right-bottom" === e.position ? (d.style.right = e.distance, d.style.bottom = e.distance, d.style.top = "auto", d.style.left = "auto") : "left-top" === e.position ? (d.style.left = e.distance, d.style.top = e.distance, d.style.right = "auto", d.style.bottom = "auto") : "left-bottom" === e.position ? (d.style.left = e.distance, d.style.bottom = e.distance, d.style.top = "auto", d.style.right = "auto") : (d.style.right = e.distance, d.style.top = e.distance, d.style.left = "auto", d.style.bottom = "auto"), e.backOverlay) {
				var x = t.document.getElementById(m.overlayID) || t.document.createElement("div");
				x.id = m.overlayID, x.style.width = "100%", x.style.height = "100%", x.style.position = "fixed", x.style.zIndex = e.zindex - 1, x.style.left = 0, x.style.top = 0, x.style.right = 0, x.style.bottom = 0, x.style.background = f.backOverlayColor || e.backOverlayColor, x.className = e.cssAnimation ? "nx-with-animation" : "", x.style.animationDuration = e.cssAnimation ? e.cssAnimationDuration + "ms" : "", t.document.getElementById(m.overlayID) || t.document.body.appendChild(x);
			}
			t.document.getElementById(m.wrapID) || t.document.body.appendChild(d);
			var g = t.document.createElement("div");
			g.id = e.ID + "-" + T, g.className = e.className + " " + f.childClassName + " " + (e.cssAnimation ? "nx-with-animation" : "") + " " + (e.useIcon ? "nx-with-icon" : "") + " nx-" + e.cssAnimationStyle + " " + (e.closeButton && "function" != typeof o ? "nx-with-close-button" : "") + " " + ("function" == typeof o ? "nx-with-callback" : "") + " " + (e.clickToClose ? "nx-notify-click-to-close" : ""), g.style.fontSize = e.fontSize, g.style.color = f.textColor, g.style.background = f.background, g.style.borderRadius = e.borderRadius, g.style.pointerEvents = "all", e.rtl && (g.setAttribute("dir", "rtl"), g.classList.add("nx-rtl-on")), g.style.fontFamily = "\"" + e.fontFamily + "\", " + s, e.cssAnimation && (g.style.animationDuration = e.cssAnimationDuration + "ms");
			var b = "";
			if (e.closeButton && "function" != typeof o && (b = "<span class=\"nx-close-button\"><svg xmlns=\"http://www.w3.org/2000/svg\" width=\"20\" height=\"20\" viewBox=\"0 0 20 20\"><g><path fill=\"" + f.notiflixIconColor + "\" d=\"M0.38 2.19l7.8 7.81 -7.8 7.81c-0.51,0.5 -0.51,1.31 -0.01,1.81 0.25,0.25 0.57,0.38 0.91,0.38 0.34,0 0.67,-0.14 0.91,-0.38l7.81 -7.81 7.81 7.81c0.24,0.24 0.57,0.38 0.91,0.38 0.34,0 0.66,-0.14 0.9,-0.38 0.51,-0.5 0.51,-1.31 0,-1.81l-7.81 -7.81 7.81 -7.81c0.51,-0.5 0.51,-1.31 0,-1.82 -0.5,-0.5 -1.31,-0.5 -1.81,0l-7.81 7.81 -7.81 -7.81c-0.5,-0.5 -1.31,-0.5 -1.81,0 -0.51,0.51 -0.51,1.32 0,1.82z\"/></g></svg></span>"), !e.useIcon) g.innerHTML = "<span class=\"nx-message\">" + n + "</span>" + (e.closeButton ? b : "");
			else if (e.useFontAwesome) g.innerHTML = "<i style=\"color:" + f.fontAwesomeIconColor + "; font-size:" + e.fontAwesomeIconSize + ";\" class=\"nx-message-icon nx-message-icon-fa " + f.fontAwesomeClassName + " " + ("shadow" === e.fontAwesomeIconStyle ? "nx-message-icon-fa-shadow" : "nx-message-icon-fa-basic") + "\"></i><span class=\"nx-message nx-with-icon\">" + n + "</span>" + (e.closeButton ? b : "");
			else {
				var u = "";
				a === l.Success ? u = "<svg class=\"nx-message-icon\" xmlns=\"http://www.w3.org/2000/svg\" width=\"40\" height=\"40\" viewBox=\"0 0 40 40\"><g><path fill=\"" + f.notiflixIconColor + "\" d=\"M20 0c11.03,0 20,8.97 20,20 0,11.03 -8.97,20 -20,20 -11.03,0 -20,-8.97 -20,-20 0,-11.03 8.97,-20 20,-20zm0 37.98c9.92,0 17.98,-8.06 17.98,-17.98 0,-9.92 -8.06,-17.98 -17.98,-17.98 -9.92,0 -17.98,8.06 -17.98,17.98 0,9.92 8.06,17.98 17.98,17.98zm-2.4 -13.29l11.52 -12.96c0.37,-0.41 1.01,-0.45 1.42,-0.08 0.42,0.37 0.46,1 0.09,1.42l-12.16 13.67c-0.19,0.22 -0.46,0.34 -0.75,0.34 -0.23,0 -0.45,-0.07 -0.63,-0.22l-7.6 -6.07c-0.43,-0.35 -0.5,-0.99 -0.16,-1.42 0.35,-0.43 0.99,-0.5 1.42,-0.16l6.85 5.48z\"/></g></svg>" : a === l.Failure ? u = "<svg class=\"nx-message-icon\" xmlns=\"http://www.w3.org/2000/svg\" width=\"40\" height=\"40\" viewBox=\"0 0 40 40\"><g><path fill=\"" + f.notiflixIconColor + "\" d=\"M20 0c11.03,0 20,8.97 20,20 0,11.03 -8.97,20 -20,20 -11.03,0 -20,-8.97 -20,-20 0,-11.03 8.97,-20 20,-20zm0 37.98c9.92,0 17.98,-8.06 17.98,-17.98 0,-9.92 -8.06,-17.98 -17.98,-17.98 -9.92,0 -17.98,8.06 -17.98,17.98 0,9.92 8.06,17.98 17.98,17.98zm1.42 -17.98l6.13 6.12c0.39,0.4 0.39,1.04 0,1.43 -0.19,0.19 -0.45,0.29 -0.71,0.29 -0.27,0 -0.53,-0.1 -0.72,-0.29l-6.12 -6.13 -6.13 6.13c-0.19,0.19 -0.44,0.29 -0.71,0.29 -0.27,0 -0.52,-0.1 -0.71,-0.29 -0.39,-0.39 -0.39,-1.03 0,-1.43l6.13 -6.12 -6.13 -6.13c-0.39,-0.39 -0.39,-1.03 0,-1.42 0.39,-0.39 1.03,-0.39 1.42,0l6.13 6.12 6.12 -6.12c0.4,-0.39 1.04,-0.39 1.43,0 0.39,0.39 0.39,1.03 0,1.42l-6.13 6.13z\"/></g></svg>" : a === l.Warning ? u = "<svg class=\"nx-message-icon\" xmlns=\"http://www.w3.org/2000/svg\" width=\"40\" height=\"40\" viewBox=\"0 0 40 40\"><g><path fill=\"" + f.notiflixIconColor + "\" d=\"M21.91 3.48l17.8 30.89c0.84,1.46 -0.23,3.25 -1.91,3.25l-35.6 0c-1.68,0 -2.75,-1.79 -1.91,-3.25l17.8 -30.89c0.85,-1.47 2.97,-1.47 3.82,0zm16.15 31.84l-17.8 -30.89c-0.11,-0.2 -0.41,-0.2 -0.52,0l-17.8 30.89c-0.12,0.2 0.05,0.4 0.26,0.4l35.6 0c0.21,0 0.38,-0.2 0.26,-0.4zm-19.01 -4.12l0 -1.05c0,-0.53 0.42,-0.95 0.95,-0.95 0.53,0 0.95,0.42 0.95,0.95l0 1.05c0,0.53 -0.42,0.95 -0.95,0.95 -0.53,0 -0.95,-0.42 -0.95,-0.95zm0 -4.66l0 -13.39c0,-0.52 0.42,-0.95 0.95,-0.95 0.53,0 0.95,0.43 0.95,0.95l0 13.39c0,0.53 -0.42,0.96 -0.95,0.96 -0.53,0 -0.95,-0.43 -0.95,-0.96z\"/></g></svg>" : a === l.Info && (u = "<svg class=\"nx-message-icon\" xmlns=\"http://www.w3.org/2000/svg\" width=\"40\" height=\"40\" viewBox=\"0 0 40 40\"><g><path fill=\"" + f.notiflixIconColor + "\" d=\"M20 0c11.03,0 20,8.97 20,20 0,11.03 -8.97,20 -20,20 -11.03,0 -20,-8.97 -20,-20 0,-11.03 8.97,-20 20,-20zm0 37.98c9.92,0 17.98,-8.06 17.98,-17.98 0,-9.92 -8.06,-17.98 -17.98,-17.98 -9.92,0 -17.98,8.06 -17.98,17.98 0,9.92 8.06,17.98 17.98,17.98zm-0.99 -23.3c0,-0.54 0.44,-0.98 0.99,-0.98 0.55,0 0.99,0.44 0.99,0.98l0 15.86c0,0.55 -0.44,0.99 -0.99,0.99 -0.55,0 -0.99,-0.44 -0.99,-0.99l0 -15.86zm0 -5.22c0,-0.55 0.44,-0.99 0.99,-0.99 0.55,0 0.99,0.44 0.99,0.99l0 1.09c0,0.54 -0.44,0.99 -0.99,0.99 -0.55,0 -0.99,-0.45 -0.99,-0.99l0 -1.09z\"/></g></svg>"), g.innerHTML = u + "<span class=\"nx-message nx-with-icon\">" + n + "</span>" + (e.closeButton ? b : "");
			}
			if ("left-bottom" === e.position || "right-bottom" === e.position) {
				var y = t.document.getElementById(m.wrapID);
				y.insertBefore(g, y.firstChild);
			} else t.document.getElementById(m.wrapID).appendChild(g);
			var k = t.document.getElementById(g.id);
			if (k) {
				var h, C, z = function() {
					k.classList.add("nx-remove");
					var e = t.document.getElementById(m.overlayID);
					e && 0 >= d.childElementCount && e.classList.add("nx-remove"), clearTimeout(h);
				}, S = function() {
					if (k && null !== k.parentNode && k.parentNode.removeChild(k), 0 >= d.childElementCount && null !== d.parentNode) {
						d.parentNode.removeChild(d);
						var e = t.document.getElementById(m.overlayID);
						e && null !== e.parentNode && e.parentNode.removeChild(e);
					}
					clearTimeout(C);
				};
				if (e.closeButton && "function" != typeof o) t.document.getElementById(g.id).querySelector("span.nx-close-button").addEventListener("click", function() {
					z();
					var t = setTimeout(function() {
						S(), clearTimeout(t);
					}, e.cssAnimationDuration);
				});
				if (("function" == typeof o || e.clickToClose) && k.addEventListener("click", function() {
					"function" == typeof o && o(), z();
					var t = setTimeout(function() {
						S(), clearTimeout(t);
					}, e.cssAnimationDuration);
				}), !e.closeButton && "function" != typeof o) {
					var W = function() {
						h = setTimeout(function() {
							z();
						}, e.timeout), C = setTimeout(function() {
							S();
						}, e.timeout + e.cssAnimationDuration);
					};
					W(), e.pauseOnHover && (k.addEventListener("mouseenter", function() {
						k.classList.add("nx-paused"), clearTimeout(h), clearTimeout(C);
					}), k.addEventListener("mouseleave", function() {
						k.classList.remove("nx-paused"), W();
					}));
				}
			}
			if (e.showOnlyTheLastOne && 0 < T) for (var I, R = t.document.querySelectorAll("[id^=" + e.ID + "-]:not([id=" + e.ID + "-" + T + "])"), A = 0; A < R.length; A++) I = R[A], null !== I.parentNode && I.parentNode.removeChild(I);
			e = v(!0, e, c);
		}, E = function() {
			return "[id^=NotiflixReportWrap]{position:fixed;z-index:4002;width:100%;height:100%;-webkit-box-sizing:border-box;box-sizing:border-box;font-family:\"Quicksand\",-apple-system,BlinkMacSystemFont,\"Segoe UI\",Roboto,\"Helvetica Neue\",Arial,sans-serif;left:0;top:0;padding:10px;color:#1e1e1e;border-radius:25px;background:transparent;display:-webkit-box;display:-webkit-flex;display:-ms-flexbox;display:flex;-webkit-flex-wrap:wrap;-ms-flex-wrap:wrap;flex-wrap:wrap;-webkit-box-orient:vertical;-webkit-box-direction:normal;-webkit-flex-direction:column;-ms-flex-direction:column;flex-direction:column;-webkit-box-align:center;-webkit-align-items:center;-ms-flex-align:center;align-items:center;-webkit-box-pack:center;-webkit-justify-content:center;-ms-flex-pack:center;justify-content:center}[id^=NotiflixReportWrap] *{-webkit-box-sizing:border-box;box-sizing:border-box}[id^=NotiflixReportWrap]>div[class*=\"-overlay\"]{width:100%;height:100%;left:0;top:0;background:rgba(255,255,255,.5);position:fixed;z-index:0}[id^=NotiflixReportWrap]>div.nx-report-click-to-close{cursor:pointer}[id^=NotiflixReportWrap]>div[class*=\"-content\"]{width:320px;max-width:100%;max-height:96vh;overflow-x:hidden;overflow-y:auto;border-radius:inherit;padding:10px;-webkit-filter:drop-shadow(0 0 5px rgba(0,0,0,0.05));filter:drop-shadow(0 0 5px rgba(0, 0, 0, .05));border:1px solid rgba(0,0,0,.03);background:#f8f8f8;position:relative;z-index:1}[id^=NotiflixReportWrap]>div[class*=\"-content\"]::-webkit-scrollbar{width:0;height:0}[id^=NotiflixReportWrap]>div[class*=\"-content\"]::-webkit-scrollbar-thumb{background:transparent}[id^=NotiflixReportWrap]>div[class*=\"-content\"]::-webkit-scrollbar-track{background:transparent}[id^=NotiflixReportWrap]>div[class*=\"-content\"]>div[class$=\"-icon\"]{-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;width:110px;height:110px;display:block;margin:6px auto 12px}[id^=NotiflixReportWrap]>div[class*=\"-content\"]>div[class$=\"-icon\"] svg{min-width:100%;max-width:100%;height:auto}[id^=NotiflixReportWrap]>*>h5{word-break:break-all;word-break:break-word;font-family:inherit!important;font-size:16px;font-weight:500;line-height:1.4;margin:0 0 10px;padding:0 0 10px;border-bottom:1px solid rgba(0,0,0,.1);float:left;width:100%;text-align:center}[id^=NotiflixReportWrap]>*>p{word-break:break-all;word-break:break-word;font-family:inherit!important;font-size:13px;line-height:1.4;font-weight:normal;float:left;width:100%;padding:0 10px;margin:0 0 10px}[id^=NotiflixReportWrap] a#NXReportButton{word-break:break-all;word-break:break-word;-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;font-family:inherit!important;-webkit-transition:all .25s ease-in-out;-o-transition:all .25s ease-in-out;transition:all .25s ease-in-out;cursor:pointer;float:right;padding:7px 17px;background:#32c682;font-size:14px;line-height:1.4;font-weight:500;border-radius:inherit!important;color:#fff}[id^=NotiflixReportWrap] a#NXReportButton:hover{-webkit-box-shadow:inset 0 -60px 5px -5px rgba(0,0,0,.25);box-shadow:inset 0 -60px 5px -5px rgba(0,0,0,.25)}[id^=NotiflixReportWrap].nx-rtl-on a#NXReportButton{float:left}[id^=NotiflixReportWrap]>div[class*=\"-overlay\"].nx-with-animation{-webkit-animation:report-overlay-animation .3s ease-in-out 0s normal;animation:report-overlay-animation .3s ease-in-out 0s normal}@-webkit-keyframes report-overlay-animation{0%{opacity:0}100%{opacity:1}}@keyframes report-overlay-animation{0%{opacity:0}100%{opacity:1}}[id^=NotiflixReportWrap]>div[class*=\"-content\"].nx-with-animation.nx-fade{-webkit-animation:report-animation-fade .3s ease-in-out 0s normal;animation:report-animation-fade .3s ease-in-out 0s normal}@-webkit-keyframes report-animation-fade{0%{opacity:0}100%{opacity:1}}@keyframes report-animation-fade{0%{opacity:0}100%{opacity:1}}[id^=NotiflixReportWrap]>div[class*=\"-content\"].nx-with-animation.nx-zoom{-webkit-animation:report-animation-zoom .3s ease-in-out 0s normal;animation:report-animation-zoom .3s ease-in-out 0s normal}@-webkit-keyframes report-animation-zoom{0%{opacity:0;-webkit-transform:scale(.5);transform:scale(.5)}50%{opacity:1;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}}@keyframes report-animation-zoom{0%{opacity:0;-webkit-transform:scale(.5);transform:scale(.5)}50%{opacity:1;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}}[id^=NotiflixReportWrap].nx-remove>div[class*=\"-overlay\"].nx-with-animation{opacity:0;-webkit-animation:report-overlay-animation-remove .3s ease-in-out 0s normal;animation:report-overlay-animation-remove .3s ease-in-out 0s normal}@-webkit-keyframes report-overlay-animation-remove{0%{opacity:1}100%{opacity:0}}@keyframes report-overlay-animation-remove{0%{opacity:1}100%{opacity:0}}[id^=NotiflixReportWrap].nx-remove>div[class*=\"-content\"].nx-with-animation.nx-fade{opacity:0;-webkit-animation:report-animation-fade-remove .3s ease-in-out 0s normal;animation:report-animation-fade-remove .3s ease-in-out 0s normal}@-webkit-keyframes report-animation-fade-remove{0%{opacity:1}100%{opacity:0}}@keyframes report-animation-fade-remove{0%{opacity:1}100%{opacity:0}}[id^=NotiflixReportWrap].nx-remove>div[class*=\"-content\"].nx-with-animation.nx-zoom{opacity:0;-webkit-animation:report-animation-zoom-remove .3s ease-in-out 0s normal;animation:report-animation-zoom-remove .3s ease-in-out 0s normal}@-webkit-keyframes report-animation-zoom-remove{0%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}50%{opacity:.5;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:0;-webkit-transform:scale(0);transform:scale(0)}}@keyframes report-animation-zoom-remove{0%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}50%{opacity:.5;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:0;-webkit-transform:scale(0);transform:scale(0)}}";
		}, j = function(e, a, n, o, r, l) {
			if (!w("body")) return !1;
			i || G.Report.init({});
			var m = {};
			if ("object" == typeof r && !Array.isArray(r) || "object" == typeof l && !Array.isArray(l)) {
				var f = {};
				"object" == typeof r ? f = r : "object" == typeof l && (f = l), m = v(!0, i, {}), i = v(!0, i, f);
			}
			var d = i[e.toLocaleLowerCase("en")];
			"string" != typeof a && (a = "Notiflix " + e), "string" != typeof n && (e === c.Success ? n = "\"Do not try to become a person of success but try to become a person of value.\" <br><br>- Albert Einstein" : e === c.Failure ? n = "\"Failure is simply the opportunity to begin again, this time more intelligently.\" <br><br>- Henry Ford" : e === c.Warning ? n = "\"The peoples who want to live comfortably without producing and fatigue; they are doomed to lose their dignity, then liberty, and then independence and destiny.\" <br><br>- Mustafa Kemal Ataturk" : e === c.Info && (n = "\"Knowledge rests not upon truth alone, but upon error also.\" <br><br>- Carl Gustav Jung")), "string" != typeof o && (o = "Okay"), i.plainText && (a = N(a), n = N(n), o = N(o)), i.plainText || (a.length > i.titleMaxLength && (a = "Possible HTML Tags Error", n = "The \"plainText\" option is \"false\" and the title content length is more than the \"titleMaxLength\" option.", o = "Okay"), n.length > i.messageMaxLength && (a = "Possible HTML Tags Error", n = "The \"plainText\" option is \"false\" and the message content length is more than the \"messageMaxLength\" option.", o = "Okay"), o.length > i.buttonMaxLength && (a = "Possible HTML Tags Error", n = "The \"plainText\" option is \"false\" and the button content length is more than the \"buttonMaxLength\" option.", o = "Okay")), a.length > i.titleMaxLength && (a = a.substring(0, i.titleMaxLength) + "..."), n.length > i.messageMaxLength && (n = n.substring(0, i.messageMaxLength) + "..."), o.length > i.buttonMaxLength && (o = o.substring(0, i.buttonMaxLength) + "..."), i.cssAnimation || (i.cssAnimationDuration = 0);
			var x = t.document.createElement("div");
			x.id = p.ID, x.className = i.className, x.style.zIndex = i.zindex, x.style.borderRadius = i.borderRadius, x.style.fontFamily = "\"" + i.fontFamily + "\", " + s, i.rtl && (x.setAttribute("dir", "rtl"), x.classList.add("nx-rtl-on")), x.style.display = "flex", x.style.flexWrap = "wrap", x.style.flexDirection = "column", x.style.alignItems = "center", x.style.justifyContent = "center";
			var g = "", b = !0 === i.backOverlayClickToClose;
			i.backOverlay && (g = "<div class=\"" + i.className + "-overlay" + (i.cssAnimation ? " nx-with-animation" : "") + (b ? " nx-report-click-to-close" : "") + "\" style=\"background:" + (d.backOverlayColor || i.backOverlayColor) + ";animation-duration:" + i.cssAnimationDuration + "ms;\"></div>");
			var u = "";
			if (e === c.Success ? u = C(i.svgSize, d.svgColor) : e === c.Failure ? u = z(i.svgSize, d.svgColor) : e === c.Warning ? u = S(i.svgSize, d.svgColor) : e === c.Info && (u = L(i.svgSize, d.svgColor)), x.innerHTML = g + "<div class=\"" + i.className + "-content" + (i.cssAnimation ? " nx-with-animation " : "") + " nx-" + i.cssAnimationStyle + "\" style=\"width:" + i.width + "; background:" + i.backgroundColor + "; animation-duration:" + i.cssAnimationDuration + "ms;\"><div style=\"width:" + i.svgSize + "; height:" + i.svgSize + ";\" class=\"" + i.className + "-icon\">" + u + "</div><h5 class=\"" + i.className + "-title\" style=\"font-weight:500; font-size:" + i.titleFontSize + "; color:" + d.titleColor + ";\">" + a + "</h5><p class=\"" + i.className + "-message\" style=\"font-size:" + i.messageFontSize + "; color:" + d.messageColor + ";\">" + n + "</p><a id=\"NXReportButton\" class=\"" + i.className + "-button\" style=\"font-weight:500; font-size:" + i.buttonFontSize + "; background:" + d.buttonBackground + "; color:" + d.buttonColor + ";\">" + o + "</a></div>", !t.document.getElementById(x.id)) {
				t.document.body.appendChild(x);
				var y = function() {
					var e = t.document.getElementById(x.id);
					e.classList.add("nx-remove");
					var a = setTimeout(function() {
						null !== e.parentNode && e.parentNode.removeChild(e), clearTimeout(a);
					}, i.cssAnimationDuration);
				};
				if (t.document.getElementById("NXReportButton").addEventListener("click", function() {
					"function" == typeof r && r(), y();
				}), g && b) t.document.querySelector(".nx-report-click-to-close").addEventListener("click", function() {
					y();
				});
			}
			i = v(!0, i, m);
		}, O = function() {
			return "[id^=NotiflixConfirmWrap]{position:fixed;z-index:4003;width:100%;height:100%;left:0;top:0;padding:10px;-webkit-box-sizing:border-box;box-sizing:border-box;background:transparent;font-family:\"Quicksand\",-apple-system,BlinkMacSystemFont,\"Segoe UI\",Roboto,\"Helvetica Neue\",Arial,sans-serif;display:-webkit-box;display:-webkit-flex;display:-ms-flexbox;display:flex;-webkit-flex-wrap:wrap;-ms-flex-wrap:wrap;flex-wrap:wrap;-webkit-box-orient:vertical;-webkit-box-direction:normal;-webkit-flex-direction:column;-ms-flex-direction:column;flex-direction:column;-webkit-box-align:center;-webkit-align-items:center;-ms-flex-align:center;align-items:center;-webkit-box-pack:center;-webkit-justify-content:center;-ms-flex-pack:center;justify-content:center}[id^=NotiflixConfirmWrap].nx-position-center-top{-webkit-box-pack:start;-webkit-justify-content:flex-start;-ms-flex-pack:start;justify-content:flex-start}[id^=NotiflixConfirmWrap].nx-position-center-bottom{-webkit-box-pack:end;-webkit-justify-content:flex-end;-ms-flex-pack:end;justify-content:flex-end}[id^=NotiflixConfirmWrap].nx-position-left-top{-webkit-box-align:start;-webkit-align-items:flex-start;-ms-flex-align:start;align-items:flex-start;-webkit-box-pack:start;-webkit-justify-content:flex-start;-ms-flex-pack:start;justify-content:flex-start}[id^=NotiflixConfirmWrap].nx-position-left-center{-webkit-box-align:start;-webkit-align-items:flex-start;-ms-flex-align:start;align-items:flex-start}[id^=NotiflixConfirmWrap].nx-position-left-bottom{-webkit-box-align:start;-webkit-align-items:flex-start;-ms-flex-align:start;align-items:flex-start;-webkit-box-pack:end;-webkit-justify-content:flex-end;-ms-flex-pack:end;justify-content:flex-end}[id^=NotiflixConfirmWrap].nx-position-right-top{-webkit-box-align:end;-webkit-align-items:flex-end;-ms-flex-align:end;align-items:flex-end;-webkit-box-pack:start;-webkit-justify-content:flex-start;-ms-flex-pack:start;justify-content:flex-start}[id^=NotiflixConfirmWrap].nx-position-right-center{-webkit-box-align:end;-webkit-align-items:flex-end;-ms-flex-align:end;align-items:flex-end}[id^=NotiflixConfirmWrap].nx-position-right-bottom{-webkit-box-align:end;-webkit-align-items:flex-end;-ms-flex-align:end;align-items:flex-end;-webkit-box-pack:end;-webkit-justify-content:flex-end;-ms-flex-pack:end;justify-content:flex-end}[id^=NotiflixConfirmWrap] *{-webkit-box-sizing:border-box;box-sizing:border-box}[id^=NotiflixConfirmWrap]>div[class*=\"-overlay\"]{width:100%;height:100%;left:0;top:0;background:rgba(255,255,255,.5);position:fixed;z-index:0}[id^=NotiflixConfirmWrap]>div[class*=\"-overlay\"].nx-with-animation{-webkit-animation:confirm-overlay-animation .3s ease-in-out 0s normal;animation:confirm-overlay-animation .3s ease-in-out 0s normal}@-webkit-keyframes confirm-overlay-animation{0%{opacity:0}100%{opacity:1}}@keyframes confirm-overlay-animation{0%{opacity:0}100%{opacity:1}}[id^=NotiflixConfirmWrap].nx-remove>div[class*=\"-overlay\"].nx-with-animation{opacity:0;-webkit-animation:confirm-overlay-animation-remove .3s ease-in-out 0s normal;animation:confirm-overlay-animation-remove .3s ease-in-out 0s normal}@-webkit-keyframes confirm-overlay-animation-remove{0%{opacity:1}100%{opacity:0}}@keyframes confirm-overlay-animation-remove{0%{opacity:1}100%{opacity:0}}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]{width:300px;max-width:100%;max-height:96vh;overflow-x:hidden;overflow-y:auto;border-radius:25px;padding:10px;margin:0;-webkit-filter:drop-shadow(0 0 5px rgba(0,0,0,0.05));filter:drop-shadow(0 0 5px rgba(0, 0, 0, .05));background:#f8f8f8;color:#1e1e1e;position:relative;z-index:1;text-align:center}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]::-webkit-scrollbar{width:0;height:0}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]::-webkit-scrollbar-thumb{background:transparent}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]::-webkit-scrollbar-track{background:transparent}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]{float:left;width:100%;text-align:inherit}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>h5{float:left;width:100%;margin:0;padding:0 0 10px;border-bottom:1px solid rgba(0,0,0,.1);color:#32c682;font-family:inherit!important;font-size:16px;line-height:1.4;font-weight:500;text-align:inherit}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>div{font-family:inherit!important;margin:15px 0 20px;padding:0 10px;float:left;width:100%;font-size:14px;line-height:1.4;font-weight:normal;color:inherit;text-align:inherit}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>div>div{font-family:inherit!important;float:left;width:100%;margin:15px 0 0;padding:0}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>div>div>input{font-family:inherit!important;float:left;width:100%;height:40px;margin:0;padding:0 15px;border:1px solid rgba(0,0,0,.1);border-radius:25px;font-size:14px;font-weight:normal;line-height:1;-webkit-transition:all .25s ease-in-out;-o-transition:all .25s ease-in-out;transition:all .25s ease-in-out;text-align:left}[id^=NotiflixConfirmWrap].nx-rtl-on>div[class*=\"-content\"]>div[class*=\"-head\"]>div>div>input{text-align:right}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>div>div>input:hover{border-color:rgba(0,0,0,.1)}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>div>div>input:focus{border-color:rgba(0,0,0,.3)}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>div>div>input.nx-validation-failure{border-color:#ff5549}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>div>div>input.nx-validation-success{border-color:#32c682}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-buttons\"]{-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;border-radius:inherit;float:left;width:100%;text-align:inherit}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-buttons\"]>a{cursor:pointer;font-family:inherit!important;-webkit-transition:all .25s ease-in-out;-o-transition:all .25s ease-in-out;transition:all .25s ease-in-out;float:left;width:48%;padding:9px 5px;border-radius:inherit!important;font-weight:500;font-size:15px;line-height:1.4;color:#f8f8f8;text-align:inherit}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-buttons\"]>a.nx-confirm-button-ok{margin:0 2% 0 0;background:#32c682}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-buttons\"]>a.nx-confirm-button-cancel{margin:0 0 0 2%;background:#a9a9a9}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-buttons\"]>a.nx-full{margin:0;width:100%}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-buttons\"]>a:hover{-webkit-box-shadow:inset 0 -60px 5px -5px rgba(0,0,0,.25);box-shadow:inset 0 -60px 5px -5px rgba(0,0,0,.25)}[id^=NotiflixConfirmWrap].nx-rtl-on>div[class*=\"-content\"]>div[class*=\"-buttons\"],[id^=NotiflixConfirmWrap].nx-rtl-on>div[class*=\"-content\"]>div[class*=\"-buttons\"]>a{-webkit-transform:rotateY(180deg);transform:rotateY(180deg)}[id^=NotiflixConfirmWrap].nx-with-animation.nx-fade>div[class*=\"-content\"]{-webkit-animation:confirm-animation-fade .3s ease-in-out 0s normal;animation:confirm-animation-fade .3s ease-in-out 0s normal}@-webkit-keyframes confirm-animation-fade{0%{opacity:0}100%{opacity:1}}@keyframes confirm-animation-fade{0%{opacity:0}100%{opacity:1}}[id^=NotiflixConfirmWrap].nx-with-animation.nx-zoom>div[class*=\"-content\"]{-webkit-animation:confirm-animation-zoom .3s ease-in-out 0s normal;animation:confirm-animation-zoom .3s ease-in-out 0s normal}@-webkit-keyframes confirm-animation-zoom{0%{opacity:0;-webkit-transform:scale(.5);transform:scale(.5)}50%{opacity:1;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}}@keyframes confirm-animation-zoom{0%{opacity:0;-webkit-transform:scale(.5);transform:scale(.5)}50%{opacity:1;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}}[id^=NotiflixConfirmWrap].nx-with-animation.nx-fade.nx-remove>div[class*=\"-content\"]{opacity:0;-webkit-animation:confirm-animation-fade-remove .3s ease-in-out 0s normal;animation:confirm-animation-fade-remove .3s ease-in-out 0s normal}@-webkit-keyframes confirm-animation-fade-remove{0%{opacity:1}100%{opacity:0}}@keyframes confirm-animation-fade-remove{0%{opacity:1}100%{opacity:0}}[id^=NotiflixConfirmWrap].nx-with-animation.nx-zoom.nx-remove>div[class*=\"-content\"]{opacity:0;-webkit-animation:confirm-animation-zoom-remove .3s ease-in-out 0s normal;animation:confirm-animation-zoom-remove .3s ease-in-out 0s normal}@-webkit-keyframes confirm-animation-zoom-remove{0%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}50%{opacity:.5;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:0;-webkit-transform:scale(0);transform:scale(0)}}@keyframes confirm-animation-zoom-remove{0%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}50%{opacity:.5;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:0;-webkit-transform:scale(0);transform:scale(0)}}";
		}, H = function(e, i, n, o, r, l, m, c, p) {
			if (!w("body")) return !1;
			a || G.Confirm.init({});
			var x = v(!0, a, {});
			"object" != typeof p || Array.isArray(p) || (a = v(!0, a, p)), "string" != typeof i && (i = "Notiflix Confirm"), "string" != typeof n && (n = "Do you agree with me?"), "string" != typeof r && (r = "Yes"), "string" != typeof l && (l = "No"), "function" != typeof m && (m = void 0), "function" != typeof c && (c = void 0), a.plainText && (i = N(i), n = N(n), r = N(r), l = N(l)), a.plainText || (i.length > a.titleMaxLength && (i = "Possible HTML Tags Error", n = "The \"plainText\" option is \"false\" and the title content length is more than \"titleMaxLength\" option.", r = "Okay", l = "..."), n.length > a.messageMaxLength && (i = "Possible HTML Tags Error", n = "The \"plainText\" option is \"false\" and the message content length is more than \"messageMaxLength\" option.", r = "Okay", l = "..."), (r.length || l.length) > a.buttonsMaxLength && (i = "Possible HTML Tags Error", n = "The \"plainText\" option is \"false\" and the buttons content length is more than \"buttonsMaxLength\" option.", r = "Okay", l = "...")), i.length > a.titleMaxLength && (i = i.substring(0, a.titleMaxLength) + "..."), n.length > a.messageMaxLength && (n = n.substring(0, a.messageMaxLength) + "..."), r.length > a.buttonsMaxLength && (r = r.substring(0, a.buttonsMaxLength) + "..."), l.length > a.buttonsMaxLength && (l = l.substring(0, a.buttonsMaxLength) + "..."), a.cssAnimation || (a.cssAnimationDuration = 0);
			var g = t.document.createElement("div");
			g.id = d.ID, g.className = a.className + (a.cssAnimation ? " nx-with-animation nx-" + a.cssAnimationStyle : ""), g.style.zIndex = a.zindex, g.style.padding = a.distance, a.rtl && (g.setAttribute("dir", "rtl"), g.classList.add("nx-rtl-on"));
			var b = "string" == typeof a.position ? a.position.trim() : "center";
			g.classList.add("nx-position-" + b), g.style.fontFamily = "\"" + a.fontFamily + "\", " + s;
			var u = "";
			a.backOverlay && (u = "<div class=\"" + a.className + "-overlay" + (a.cssAnimation ? " nx-with-animation" : "") + "\" style=\"background:" + a.backOverlayColor + ";animation-duration:" + a.cssAnimationDuration + "ms;\"></div>");
			var y = "";
			"function" == typeof m && (y = "<a id=\"NXConfirmButtonCancel\" class=\"nx-confirm-button-cancel\" style=\"color:" + a.cancelButtonColor + ";background:" + a.cancelButtonBackground + ";font-size:" + a.buttonsFontSize + ";\">" + l + "</a>");
			var k = "", h = null, C = void 0;
			if (e === f.Ask || e === f.Prompt) {
				h = o || "";
				var z = e === f.Ask ? Math.ceil(1.5 * h.length) : 200 < h.length ? Math.ceil(1.5 * h.length) : 250;
				k = "<div><input id=\"NXConfirmValidationInput\" type=\"text\" " + (e === f.Prompt ? "value=\"" + h + "\"" : "") + " maxlength=\"" + z + "\" style=\"font-size:" + a.messageFontSize + ";border-radius: " + a.borderRadius + ";\" autocomplete=\"off\" spellcheck=\"false\" autocapitalize=\"none\" /></div>";
			}
			if (g.innerHTML = u + "<div class=\"" + a.className + "-content\" style=\"width:" + a.width + "; background:" + a.backgroundColor + "; animation-duration:" + a.cssAnimationDuration + "ms; border-radius: " + a.borderRadius + ";\"><div class=\"" + a.className + "-head\"><h5 style=\"color:" + a.titleColor + ";font-size:" + a.titleFontSize + ";\">" + i + "</h5><div style=\"color:" + a.messageColor + ";font-size:" + a.messageFontSize + ";\">" + n + k + "</div></div><div class=\"" + a.className + "-buttons\"><a id=\"NXConfirmButtonOk\" class=\"nx-confirm-button-ok" + ("function" == typeof m ? "" : " nx-full") + "\" style=\"color:" + a.okButtonColor + ";background:" + a.okButtonBackground + ";font-size:" + a.buttonsFontSize + ";\">" + r + "</a>" + y + "</div></div>", !t.document.getElementById(g.id)) {
				t.document.body.appendChild(g);
				var L = t.document.getElementById(g.id), W = t.document.getElementById("NXConfirmButtonOk"), I = t.document.getElementById("NXConfirmValidationInput");
				if (I && (I.focus(), I.setSelectionRange(0, (I.value || "").length), I.addEventListener("keyup", function(t) {
					var i = t.target.value;
					if (e === f.Ask && i !== h) t.preventDefault(), I.classList.add("nx-validation-failure"), I.classList.remove("nx-validation-success");
					else {
						e === f.Ask && (I.classList.remove("nx-validation-failure"), I.classList.add("nx-validation-success"));
						("enter" === (t.key || "").toLocaleLowerCase("en") || 13 === t.keyCode) && W.dispatchEvent(new Event("click"));
					}
				})), W.addEventListener("click", function(t) {
					if (e === f.Ask && h && I) {
						if ((I.value || "").toString() !== h) return I.focus(), I.classList.add("nx-validation-failure"), t.stopPropagation(), t.preventDefault(), t.returnValue = !1, t.cancelBubble = !0, !1;
						I.classList.remove("nx-validation-failure");
					}
					"function" == typeof m && (e === f.Prompt && I && (C = I.value || ""), m(C)), L.classList.add("nx-remove");
					var n = setTimeout(function() {
						null !== L.parentNode && (L.parentNode.removeChild(L), clearTimeout(n));
					}, a.cssAnimationDuration);
				}), "function" == typeof m) t.document.getElementById("NXConfirmButtonCancel").addEventListener("click", function() {
					"function" == typeof c && (e === f.Prompt && I && (C = I.value || ""), c(C)), L.classList.add("nx-remove");
					var t = setTimeout(function() {
						null !== L.parentNode && (L.parentNode.removeChild(L), clearTimeout(t));
					}, a.cssAnimationDuration);
				});
			}
			a = v(!0, a, x);
		}, P = function() {
			return "[id^=NotiflixLoadingWrap]{-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;position:fixed;z-index:4000;width:100%;height:100%;left:0;top:0;right:0;bottom:0;margin:auto;display:-webkit-box;display:-webkit-flex;display:-ms-flexbox;display:flex;-webkit-flex-wrap:wrap;-ms-flex-wrap:wrap;flex-wrap:wrap;-webkit-box-orient:vertical;-webkit-box-direction:normal;-webkit-flex-direction:column;-ms-flex-direction:column;flex-direction:column;-webkit-box-align:center;-webkit-align-items:center;-ms-flex-align:center;align-items:center;-webkit-box-pack:center;-webkit-justify-content:center;-ms-flex-pack:center;justify-content:center;text-align:center;-webkit-box-sizing:border-box;box-sizing:border-box;background:rgba(0,0,0,.8);font-family:\"Quicksand\",-apple-system,BlinkMacSystemFont,\"Segoe UI\",Roboto,\"Helvetica Neue\",Arial,sans-serif}[id^=NotiflixLoadingWrap] *{-webkit-box-sizing:border-box;box-sizing:border-box}[id^=NotiflixLoadingWrap].nx-loading-click-to-close{cursor:pointer}[id^=NotiflixLoadingWrap]>div[class*=\"-icon\"]{width:60px;height:60px;position:relative;-webkit-transition:top .2s ease-in-out;-o-transition:top .2s ease-in-out;transition:top .2s ease-in-out;margin:0 auto}[id^=NotiflixLoadingWrap]>div[class*=\"-icon\"] img,[id^=NotiflixLoadingWrap]>div[class*=\"-icon\"] svg{max-width:unset;max-height:unset;width:100%;height:auto;position:absolute;left:0;top:0}[id^=NotiflixLoadingWrap]>p{position:relative;margin:10px auto 0;font-family:inherit!important;font-weight:normal;font-size:15px;line-height:1.4;padding:0 10px;width:100%;text-align:center}[id^=NotiflixLoadingWrap].nx-with-animation{-webkit-animation:loading-animation-fade .3s ease-in-out 0s normal;animation:loading-animation-fade .3s ease-in-out 0s normal}@-webkit-keyframes loading-animation-fade{0%{opacity:0}100%{opacity:1}}@keyframes loading-animation-fade{0%{opacity:0}100%{opacity:1}}[id^=NotiflixLoadingWrap].nx-with-animation.nx-remove{opacity:0;-webkit-animation:loading-animation-fade-remove .3s ease-in-out 0s normal;animation:loading-animation-fade-remove .3s ease-in-out 0s normal}@-webkit-keyframes loading-animation-fade-remove{0%{opacity:1}100%{opacity:0}}@keyframes loading-animation-fade-remove{0%{opacity:1}100%{opacity:0}}[id^=NotiflixLoadingWrap]>p.nx-loading-message-new{-webkit-animation:loading-new-message-fade .3s ease-in-out 0s normal;animation:loading-new-message-fade .3s ease-in-out 0s normal}@-webkit-keyframes loading-new-message-fade{0%{opacity:0}100%{opacity:1}}@keyframes loading-new-message-fade{0%{opacity:0}100%{opacity:1}}";
		}, U = function(e, i, a, o, r) {
			if (!w("body")) return !1;
			n || G.Loading.init({});
			var l = v(!0, n, {});
			if ("object" == typeof i && !Array.isArray(i) || "object" == typeof a && !Array.isArray(a)) {
				var m = {};
				"object" == typeof i ? m = i : "object" == typeof a && (m = a), n = v(!0, n, m);
			}
			var c = "";
			if ("string" == typeof i && 0 < i.length && (c = i), o) {
				c = c.length > n.messageMaxLength ? N(c).toString().substring(0, n.messageMaxLength) + "..." : N(c).toString();
				var p = "";
				0 < c.length && (p = "<p id=\"" + n.messageID + "\" class=\"nx-loading-message\" style=\"color:" + n.messageColor + ";font-size:" + n.messageFontSize + ";\">" + c + "</p>"), n.cssAnimation || (n.cssAnimationDuration = 0);
				var f = "";
				if (e === x.Standard) f = W(n.svgSize, n.svgColor);
				else if (e === x.Hourglass) f = I(n.svgSize, n.svgColor);
				else if (e === x.Circle) f = R(n.svgSize, n.svgColor);
				else if (e === x.Arrows) f = A(n.svgSize, n.svgColor);
				else if (e === x.Dots) f = M(n.svgSize, n.svgColor);
				else if (e === x.Pulse) f = B(n.svgSize, n.svgColor);
				else if (e === x.Custom && null !== n.customSvgCode && null === n.customSvgUrl) f = n.customSvgCode || "";
				else if (e === x.Custom && null !== n.customSvgUrl && null === n.customSvgCode) f = "<img class=\"nx-custom-loading-icon\" width=\"" + n.svgSize + "\" height=\"" + n.svgSize + "\" src=\"" + n.customSvgUrl + "\" alt=\"Notiflix\">";
				else {
					if (e === x.Custom && (null === n.customSvgUrl || null === n.customSvgCode)) return y("You have to set a static SVG url to \"customSvgUrl\" option to use Loading Custom."), !1;
					f = X(n.svgSize, "#f8f8f8", "#32c682");
				}
				var d = parseInt((n.svgSize || "").replace(/[^0-9]/g, "")), b = t.innerWidth, u = d >= b ? b - 40 + "px" : d + "px", k = "<div style=\"width:" + u + "; height:" + u + ";\" class=\"" + n.className + "-icon" + (0 < c.length ? " nx-with-message" : "") + "\">" + f + "</div>", h = t.document.createElement("div");
				if (h.id = g.ID, h.className = n.className + (n.cssAnimation ? " nx-with-animation" : "") + (n.clickToClose ? " nx-loading-click-to-close" : ""), h.style.zIndex = n.zindex, h.style.background = n.backgroundColor, h.style.animationDuration = n.cssAnimationDuration + "ms", h.style.fontFamily = "\"" + n.fontFamily + "\", " + s, h.style.display = "flex", h.style.flexWrap = "wrap", h.style.flexDirection = "column", h.style.alignItems = "center", h.style.justifyContent = "center", n.rtl && (h.setAttribute("dir", "rtl"), h.classList.add("nx-rtl-on")), h.innerHTML = k + p, !t.document.getElementById(h.id) && (t.document.body.appendChild(h), n.clickToClose)) t.document.getElementById(h.id).addEventListener("click", function() {
					h.classList.add("nx-remove");
					var t = setTimeout(function() {
						null !== h.parentNode && (h.parentNode.removeChild(h), clearTimeout(t));
					}, n.cssAnimationDuration);
				});
			} else if (t.document.getElementById(g.ID)) var z = t.document.getElementById(g.ID), S = setTimeout(function() {
				z.classList.add("nx-remove");
				var t = setTimeout(function() {
					null !== z.parentNode && (z.parentNode.removeChild(z), clearTimeout(t));
				}, n.cssAnimationDuration);
				clearTimeout(S);
			}, r);
			n = v(!0, n, l);
		}, V = function(e) {
			"string" != typeof e && (e = "");
			var i = t.document.getElementById(g.ID);
			if (i) if (0 < e.length) {
				e = e.length > n.messageMaxLength ? N(e).substring(0, n.messageMaxLength) + "..." : N(e);
				var a = i.getElementsByTagName("p")[0];
				if (a) a.innerHTML = e;
				else {
					var o = t.document.createElement("p");
					o.id = n.messageID, o.className = "nx-loading-message nx-loading-message-new", o.style.color = n.messageColor, o.style.fontSize = n.messageFontSize, o.innerHTML = e, i.appendChild(o);
				}
			} else y("Where is the new message?");
		}, q = function() {
			return "[id^=NotiflixBlockWrap]{-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;-webkit-box-sizing:border-box;box-sizing:border-box;position:absolute;z-index:1000;font-family:\"Quicksand\",-apple-system,BlinkMacSystemFont,\"Segoe UI\",Roboto,\"Helvetica Neue\",Arial,sans-serif;background:rgba(255,255,255,.9);text-align:center;animation-duration:.4s;width:100%;height:100%;left:0;top:0;border-radius:inherit;display:-webkit-box;display:-webkit-flex;display:-ms-flexbox;display:flex;-webkit-flex-wrap:wrap;-ms-flex-wrap:wrap;flex-wrap:wrap;-webkit-box-orient:vertical;-webkit-box-direction:normal;-webkit-flex-direction:column;-ms-flex-direction:column;flex-direction:column;-webkit-box-align:center;-webkit-align-items:center;-ms-flex-align:center;align-items:center;-webkit-box-pack:center;-webkit-justify-content:center;-ms-flex-pack:center;justify-content:center}[id^=NotiflixBlockWrap] *{-webkit-box-sizing:border-box;box-sizing:border-box}[id^=NotiflixBlockWrap]>span[class*=\"-icon\"]{display:block;width:45px;height:45px;position:relative;margin:0 auto}[id^=NotiflixBlockWrap]>span[class*=\"-icon\"] svg{width:inherit;height:inherit}[id^=NotiflixBlockWrap]>span[class*=\"-message\"]{position:relative;display:block;width:100%;margin:10px auto 0;padding:0 10px;font-family:inherit!important;font-weight:normal;font-size:14px;line-height:1.4}[id^=NotiflixBlockWrap].nx-with-animation{-webkit-animation:block-animation-fade .3s ease-in-out 0s normal;animation:block-animation-fade .3s ease-in-out 0s normal}@-webkit-keyframes block-animation-fade{0%{opacity:0}100%{opacity:1}}@keyframes block-animation-fade{0%{opacity:0}100%{opacity:1}}[id^=NotiflixBlockWrap].nx-with-animation.nx-remove{opacity:0;-webkit-animation:block-animation-fade-remove .3s ease-in-out 0s normal;animation:block-animation-fade-remove .3s ease-in-out 0s normal}@-webkit-keyframes block-animation-fade-remove{0%{opacity:1}100%{opacity:0}}@keyframes block-animation-fade-remove{0%{opacity:1}100%{opacity:0}}";
		}, Q = 0, Y = function(e, i, a, n, r, l) {
			var m;
			if (Array.isArray(a)) {
				if (1 > a.length) return y("Array of HTMLElements should contains at least one HTMLElement."), !1;
				m = a;
			} else if (Object.prototype.isPrototypeOf.call(NodeList.prototype, a)) {
				if (1 > a.length) return y("NodeListOf<HTMLElement> should contains at least one HTMLElement."), !1;
				m = Array.prototype.slice.call(a);
			} else {
				if ("string" != typeof a || 1 > (a || "").length || 1 === (a || "").length && ("#" === (a || "")[0] || "." === (a || "")[0])) return y("The selector parameter must be a string and matches a specified CSS selector(s)."), !1;
				var p = t.document.querySelectorAll(a);
				if (1 > p.length) return y("You called the \"Notiflix.Block...\" function with \"" + a + "\" selector, but there is no such element(s) in the document."), !1;
				m = p;
			}
			o || G.Block.init({});
			var f = v(!0, o, {});
			if ("object" == typeof n && !Array.isArray(n) || "object" == typeof r && !Array.isArray(r)) {
				var d = {};
				"object" == typeof n ? d = n : "object" == typeof r && (d = r), o = v(!0, o, d);
			}
			var x = "";
			"string" == typeof n && 0 < n.length && (x = n), o.cssAnimation || (o.cssAnimationDuration = 0);
			var g = u.className;
			"string" == typeof o.className && (g = o.className.trim());
			var h = "number" == typeof o.querySelectorLimit ? o.querySelectorLimit : 200, C = (m || []).length >= h ? h : m.length, z = "nx-block-temporary-position";
			if (e) {
				for (var S, L = [
					"area",
					"base",
					"br",
					"col",
					"command",
					"embed",
					"hr",
					"img",
					"input",
					"keygen",
					"link",
					"meta",
					"param",
					"source",
					"track",
					"wbr",
					"html",
					"head",
					"title",
					"script",
					"style",
					"iframe"
				], X = 0; X < C; X++) if (S = m[X], S) {
					if (-1 < L.indexOf(S.tagName.toLocaleLowerCase("en"))) break;
					var D = S.querySelectorAll("[id^=" + u.ID + "]");
					if (1 > D.length) {
						var T = "";
						i && (i === b.Hourglass ? T = I(o.svgSize, o.svgColor) : i === b.Circle ? T = R(o.svgSize, o.svgColor) : i === b.Arrows ? T = A(o.svgSize, o.svgColor) : i === b.Dots ? T = M(o.svgSize, o.svgColor) : i === b.Pulse ? T = B(o.svgSize, o.svgColor) : T = W(o.svgSize, o.svgColor));
						var F = "<span class=\"" + g + "-icon\" style=\"width:" + o.svgSize + ";height:" + o.svgSize + ";\">" + T + "</span>", E = "";
						0 < x.length && (x = x.length > o.messageMaxLength ? N(x).substring(0, o.messageMaxLength) + "..." : N(x), E = "<span style=\"font-size:" + o.messageFontSize + ";color:" + o.messageColor + ";\" class=\"" + g + "-message\">" + x + "</span>"), Q++;
						var j = t.document.createElement("div");
						j.id = u.ID + "-" + Q, j.className = g + (o.cssAnimation ? " nx-with-animation" : ""), j.style.position = o.position, j.style.zIndex = o.zindex, j.style.background = o.backgroundColor, j.style.animationDuration = o.cssAnimationDuration + "ms", j.style.fontFamily = "\"" + o.fontFamily + "\", " + s, j.style.display = "flex", j.style.flexWrap = "wrap", j.style.flexDirection = "column", j.style.alignItems = "center", j.style.justifyContent = "center", o.rtl && (j.setAttribute("dir", "rtl"), j.classList.add("nx-rtl-on")), j.innerHTML = F + E;
						var O = t.getComputedStyle(S).getPropertyValue("position"), H = "string" == typeof O ? O.toLocaleLowerCase("en") : "relative", P = Math.round(1.25 * parseInt(o.svgSize)) + 40, U = S.offsetHeight || 0, V = "";
						P > U && (V = "min-height:" + P + "px;");
						var q = "";
						q = S.getAttribute("id") ? "#" + S.getAttribute("id") : S.classList[0] ? "." + S.classList[0] : (S.tagName || "").toLocaleLowerCase("en");
						var Y = "", K = -1 >= [
							"absolute",
							"relative",
							"fixed",
							"sticky"
						].indexOf(H);
						if (K || 0 < V.length) {
							if (!w("head")) return !1;
							K && (Y = "position:relative!important;");
							var $ = "<style id=\"Style-" + u.ID + "-" + Q + "\">" + q + "." + z + "{" + Y + V + "}</style>", J = t.document.createRange();
							J.selectNode(t.document.head);
							var Z = J.createContextualFragment($);
							t.document.head.appendChild(Z), S.classList.add(z);
						}
						S.appendChild(j);
					}
				}
			} else var _ = function(e) {
				var i = setTimeout(function() {
					null !== e.parentNode && e.parentNode.removeChild(e);
					var a = e.getAttribute("id"), n = t.document.getElementById("Style-" + a);
					n && null !== n.parentNode && n.parentNode.removeChild(n), clearTimeout(i);
				}, o.cssAnimationDuration);
			}, tt = function(t) {
				if (t && 0 < t.length) for (var e, n = 0; n < t.length; n++) e = t[n], e && (e.classList.add("nx-remove"), _(e));
				else "string" == typeof a ? k("\"Notiflix.Block.remove();\" function called with \"" + a + "\" selector, but this selector does not have a \"Block\" element to remove.") : k("\"Notiflix.Block.remove();\" function called with \"" + a + "\", but this \"Array<HTMLElement>\" or \"NodeListOf<HTMLElement>\" does not have a \"Block\" element to remove.");
			}, et = function(t) {
				var e = setTimeout(function() {
					t.classList.remove(z), clearTimeout(e);
				}, o.cssAnimationDuration + 300);
			}, it = setTimeout(function() {
				for (var t, e = 0; e < C; e++) t = m[e], t && (et(t), D = t.querySelectorAll("[id^=" + u.ID + "]"), tt(D));
				clearTimeout(it);
			}, l);
			o = v(!0, o, f);
		}, G = {
			Notify: {
				init: function(t) {
					e = v(!0, m, t), h(D, "NotiflixNotifyInternalCSS");
				},
				merge: function(t) {
					return e ? void (e = v(!0, e, t)) : (y("You have to initialize the Notify module before call Merge function."), !1);
				},
				success: function(t, e, i) {
					F(l.Success, t, e, i);
				},
				failure: function(t, e, i) {
					F(l.Failure, t, e, i);
				},
				warning: function(t, e, i) {
					F(l.Warning, t, e, i);
				},
				info: function(t, e, i) {
					F(l.Info, t, e, i);
				}
			},
			Report: {
				init: function(t) {
					i = v(!0, p, t), h(E, "NotiflixReportInternalCSS");
				},
				merge: function(t) {
					return i ? void (i = v(!0, i, t)) : (y("You have to initialize the Report module before call Merge function."), !1);
				},
				success: function(t, e, i, a, n) {
					j(c.Success, t, e, i, a, n);
				},
				failure: function(t, e, i, a, n) {
					j(c.Failure, t, e, i, a, n);
				},
				warning: function(t, e, i, a, n) {
					j(c.Warning, t, e, i, a, n);
				},
				info: function(t, e, i, a, n) {
					j(c.Info, t, e, i, a, n);
				}
			},
			Confirm: {
				init: function(t) {
					a = v(!0, d, t), h(O, "NotiflixConfirmInternalCSS");
				},
				merge: function(t) {
					return a ? void (a = v(!0, a, t)) : (y("You have to initialize the Confirm module before call Merge function."), !1);
				},
				show: function(t, e, i, a, n, o, r) {
					H(f.Show, t, e, null, i, a, n, o, r);
				},
				ask: function(t, e, i, a, n, o, r, s) {
					H(f.Ask, t, e, i, a, n, o, r, s);
				},
				prompt: function(t, e, i, a, n, o, r, s) {
					H(f.Prompt, t, e, i, a, n, o, r, s);
				}
			},
			Loading: {
				init: function(t) {
					n = v(!0, g, t), h(P, "NotiflixLoadingInternalCSS");
				},
				merge: function(t) {
					return n ? void (n = v(!0, n, t)) : (y("You have to initialize the Loading module before call Merge function."), !1);
				},
				standard: function(t, e) {
					U(x.Standard, t, e, !0, 0);
				},
				hourglass: function(t, e) {
					U(x.Hourglass, t, e, !0, 0);
				},
				circle: function(t, e) {
					U(x.Circle, t, e, !0, 0);
				},
				arrows: function(t, e) {
					U(x.Arrows, t, e, !0, 0);
				},
				dots: function(t, e) {
					U(x.Dots, t, e, !0, 0);
				},
				pulse: function(t, e) {
					U(x.Pulse, t, e, !0, 0);
				},
				custom: function(t, e) {
					U(x.Custom, t, e, !0, 0);
				},
				notiflix: function(t, e) {
					U(x.Notiflix, t, e, !0, 0);
				},
				remove: function(t) {
					"number" != typeof t && (t = 0), U(null, null, null, !1, t);
				},
				change: function(t) {
					V(t);
				}
			},
			Block: {
				init: function(t) {
					o = v(!0, u, t), h(q, "NotiflixBlockInternalCSS");
				},
				merge: function(t) {
					return o ? void (o = v(!0, o, t)) : (y("You have to initialize the \"Notiflix.Block\" module before call Merge function."), !1);
				},
				standard: function(t, e, i) {
					Y(!0, b.Standard, t, e, i);
				},
				hourglass: function(t, e, i) {
					Y(!0, b.Hourglass, t, e, i);
				},
				circle: function(t, e, i) {
					Y(!0, b.Circle, t, e, i);
				},
				arrows: function(t, e, i) {
					Y(!0, b.Arrows, t, e, i);
				},
				dots: function(t, e, i) {
					Y(!0, b.Dots, t, e, i);
				},
				pulse: function(t, e, i) {
					Y(!0, b.Pulse, t, e, i);
				},
				remove: function(t, e) {
					"number" != typeof e && (e = 0), Y(!1, null, t, null, null, e);
				}
			}
		};
		return "object" == typeof t.Notiflix ? v(!0, t.Notiflix, {
			Notify: G.Notify,
			Report: G.Report,
			Confirm: G.Confirm,
			Loading: G.Loading,
			Block: G.Block
		}) : {
			Notify: G.Notify,
			Report: G.Report,
			Confirm: G.Confirm,
			Loading: G.Loading,
			Block: G.Block
		};
	});
})))(), 1)).default;
var throttleTimer;
if (typeof window !== "undefined") Loading.init({ zindex: 400 });
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
const cn = (...classNames) => {
	return classNames.filter((x) => x).join(" ");
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
const arrayToMapN = (array, keys) => {
	const map = /* @__PURE__ */ new Map();
	if (typeof keys === "string") for (const e of array) map.set(e[keys], e);
	else if (Array.isArray(keys)) for (const e of array) {
		const keyGrouped = keys.map((key) => e[key] || "").join("_");
		map.set(keyGrouped, e);
	}
	else console.warn("No es un array::", array);
	return map;
};
function formatN(x, decimal, fixedLen, charF) {
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
}
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
var TOKEN_REFRESH_THRESHOLD = 2400;
var TOKEN_CHECK_INTERVAL = 240;
var REFRESH_LOCK_DURATION = 30;
var REFRESH_LOCK_KEY = Env.appId + "TokenRefreshLock";
var reloadLoginFn = async () => {
	console.warn("reloadLogin function not registered");
};
const getToken = (noError) => {
	const userToken = LocalStorage.getItem(Env.appId + "UserToken");
	const expTime = parseInt(LocalStorage.getItem(Env.appId + "TokenExpTime") || "0");
	const nowTime = Math.floor(Date.now() / 1e3);
	if (!userToken) {
		if (!noError) console.error("No se encontró la data del usuario. ¿Está logeado?:", Env.appId);
		return "";
	} else if (!expTime || nowTime > expTime) {
		if (!noError) {
			Notify.failure("La sesión ha expirado, vuelva a iniciar sesión.");
			Env.clearAccesos?.();
		}
		return "";
	}
	if (expTime - nowTime < 900) throttle(() => {
		Notify.warning(`La sesión expirará en 15 minutos`);
	}, 20);
	else if (expTime - nowTime < 300) throttle(() => {
		Notify.warning(`La sesión expirará en 5 minutos`);
	}, 20);
	return userToken || "";
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
		await reloadLoginFn();
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
			for (let nivel of niveles) {
				const code = (accesoID * 10 + nivel).toString(32);
				if (!this.#cachedResults.has(code)) {
					const hasAccess = new RegExp(`[${b32ls}]${code}[${b32ls}]`).test(this.#accesos);
					this.#cachedResults.set(code, hasAccess);
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
			get$1(this.#version);
			return false;
		}
		get$1(s);
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
			get$1(this.#version);
			return;
		}
		get$1(s);
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
			set$1(this.#size, super.size);
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
			set$1(this.#size, super.size);
			set$1(s, -1);
			increment(this.#version);
		}
		return res;
	}
	clear() {
		if (super.size === 0) return;
		super.clear();
		var sources = this.#sources;
		set$1(this.#size, 0);
		for (var s of sources.values()) set$1(s, -1);
		increment(this.#version);
		sources.clear();
	}
	#read_all() {
		get$1(this.#version);
		var sources = this.#sources;
		if (this.#size.v !== sources.size) {
			for (var key of super.keys()) if (!sources.has(key)) {
				var s = this.#source(0);
				sources.set(key, s);
			}
		}
		for ([, s] of this.#sources) get$1(s);
	}
	keys() {
		get$1(this.#version);
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
		get$1(this.#size);
		return super.size;
	}
};
URLSearchParams, Symbol.iterator;
const getDeviceType = () => {
	let view = 1;
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
	openLayers: [],
	pageOptions: [],
	pageOptionSelected: 1,
	showMobileSearchLayer: null,
	webRTCConnected: false,
	webRTCConnecting: false,
	webRTCReconnectAttempts: 0,
	webRTCError: null,
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
	},
	ecommerce: { cartOption: 1 }
});
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
const mainMenuOptions = [
	{
		name: "Iniciar Sesión",
		icon: "icon-user",
		onClick: () => {
			if (!Core.openLayers.includes(1)) Core.openLayers.push(1);
		}
	},
	{
		name: "Regístrate",
		icon: "icon-doc"
	},
	{
		name: "Mis Pedidos",
		icon: "icon-box"
	},
	{
		name: "Tienda",
		icon: "icon-home"
	}
];
var navFlags = [];
var navFlagCounter = 0;
var navReturns = 0;
const suscribeUrlFlag = (elementId, callbackOnClose) => {
	navFlags = navFlags.filter((x) => document.getElementById(x.elementId));
	let flag = navFlags.find((x) => x.elementId === elementId);
	const isIncluded = !!flag;
	if (!flag) {
		flag = {
			id: navFlags.length + 1,
			close: callbackOnClose,
			elementId
		};
		navFlags.push(flag);
	}
	flag.updated = Date.now();
	let uriParams = window.location.search.substring(1).split("&").filter((x) => x);
	const nf = (uriParams.find((x) => x.substring(0, 3) === "nf=") || "").replace("nf=", "");
	if (navFlagCounter === 0 && nf) navFlagCounter = nf.split(",").map((x) => parseInt(x))[0];
	if (!isIncluded || navReturns > 0) {
		navFlagCounter++;
		if (navReturns > 0) navReturns--;
		uriParams = uriParams.filter((x) => x.substring(0, 3) !== "nf=");
		uriParams.push(`nf=${navFlagCounter},${navFlags.map((x) => x.id).join(",")}`);
		goto(window.location.pathname + "?" + uriParams.join("&"), {
			noScroll: true,
			replaceState: false
		});
	}
};
if (typeof window !== "undefined") window.addEventListener("popstate", () => {
	navFlags = navFlags.filter((x) => document.getElementById(x.elementId));
	let flag;
	for (const e of navFlags) if (!flag || e.updated > flag.updated) flag = e;
	if (flag) {
		flag.close();
		navFlags = navFlags.filter((x) => x.id !== flag.id);
	}
});
var tempID = parseInt(String(Math.floor(Date.now() / 1e3)).substring(4));
var serviceWorkerResolverMap = /* @__PURE__ */ new Map();
var serviceWorkerHandlerMap = /* @__PURE__ */ new Map();
var successfulResponses = /* @__PURE__ */ new Set();
var getTS = () => {
	const d = /* @__PURE__ */ new Date();
	return `${d.getHours().toString().padStart(2, "0")}:${d.getMinutes().toString().padStart(2, "0")}:${d.getSeconds().toString().padStart(2, "0")}.${d.getMilliseconds().toString().padStart(3, "0")}`;
};
var nowTime = Date.now();
var swReadyPromise = null;
var swIsInitialized = false;
const doInitServiceWorker = () => {
	if (typeof navigator.serviceWorker === "undefined") {
		console.log("serviceWorker es undefined");
		return Promise.resolve(0);
	}
	if (swReadyPromise) return Promise.resolve(1);
	navigator.serviceWorker.addEventListener("message", ({ data }) => {
		console.log(`[${getTS()}] [SW-Cache] Message from Service Worker:`, data);
		if (data.__response__ === 5) setFetchProgress(data.bytes);
		else if (data.__response__ === 40) {
			const handler = serviceWorkerHandlerMap.get(40);
			if (handler) handler(data);
			else console.log("No handler registered for action 40 (WebRTC signal)");
		} else if (data.__response__ > 0 && data.__req__ > 0) {
			if (data.__response__ === 3) fetchEvent(data.__req__, 0);
			successfulResponses.add(data.__req__);
			if (serviceWorkerResolverMap.get(data.__req__)) {
				serviceWorkerResolverMap.get(data.__req__)?.(data);
				serviceWorkerResolverMap.delete(data.__req__);
			}
		}
	});
	swReadyPromise = new Promise((resolve) => {
		navigator.serviceWorker.register(Env.serviceWorker, {
			scope: "/",
			type: "module"
		}).then(() => {
			console.log(`[${getTS()}] [SW-Cache] Service Worker registered!`);
			return navigator.serviceWorker.ready;
		}).then(() => {
			console.log(`[${getTS()}] [SW-Cache] Service Worker ready (Iniciado en: ${Date.now() - nowTime}ms)`);
			swIsInitialized = true;
			resolve();
		});
	});
	return swReadyPromise.then(() => 1);
};
const sendServiceMessage = async (accion, content) => {
	if (!swIsInitialized) await doInitServiceWorker();
	const reqID = tempID;
	if (accion === 3) {
		[
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
			status.updated = Date.now();
			if (status.tryCount > 0) console.log(`[${getTS()}] [SW-Cache] Retrying fetch (Action: ${accion}, Attempt: ${status.tryCount})`);
			else console.log(`[${getTS()}] [SW-Cache] First fetch attempt (Action: ${accion})`);
			let route = `${window.location.origin}/_sw_?accion=${accion}&req=${reqID}&env=${Env.enviroment}`;
			if (content.route) route += "&r=" + content.route.replaceAll("?", "_").replaceAll("&", "_");
			status.tryCount++;
			fetch(route, {
				method: "POST",
				body: JSON.stringify(content),
				headers: { "Content-Type": "application/json" }
			}).then((res) => res.json()).then((res) => {
				console.log(`[${getTS()}] [SW-Cache] Fetch response received (Action: ${accion}):`, res);
				status.id = 2;
				status.updated = Date.now();
			}).catch((err) => {
				console.log(`[${getTS()}] [SW-Cache] Fetch error (Action: ${accion}):`, err);
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
const fetchCache = async (args) => {
	args.routeParsed = Env.makeRoute(args.route);
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
		Notify.failure(String(errMessage));
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
var progressTimeStart = 0;
var progressBytes = 0;
const setFetchProgress = (bytesLen) => {
	const nowTime = Date.now();
	if (!progressBytes) progressTimeStart = nowTime;
	progressBytes += bytesLen;
	let mbps = 0;
	const kb = progressBytes / 1e3;
	const elapsed = nowTime - progressTimeStart;
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
	const routeParsed = Env.makeRoute(props.route);
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
var productosPromiseMap = /* @__PURE__ */ new Map();
const productosServiceState = proxy({
	productos: [],
	productosMap: /* @__PURE__ */ new Map(),
	categorias: [],
	categoriasMap: /* @__PURE__ */ new Map()
});
const getProductos = async (categoriasIDs) => {
	const apiRoute = `p-productos-cms?categorias=${(categoriasIDs || [0]).join(".")}`;
	if (!productosPromiseMap.has(apiRoute)) {
		new Headers().append("Authorization", `Bearer 1`);
		console.log("Consultando Productos | API:", apiRoute);
		productosPromiseMap.set(apiRoute, new Promise((resolve, reject) => {
			GET({ route: apiRoute }).then((res) => {
				console.log("Productos response:", Object.keys(res));
				console.log(res);
				for (const e of res.productos || []) e.Image = (e.Images || [])[0] || { n: "" };
				res.productosMap = arrayToMapN(res.productos || [], "ID");
				res.categoriasMap = arrayToMapN(res.categorias || [], "ID");
				res.updated = Math.floor(Date.now() / 1e3);
				productosServiceState.productos = res.productos;
				productosServiceState.productosMap = res.productosMap;
				productosServiceState.categorias = res.categorias;
				productosServiceState.categoriasMap = res.categoriasMap;
				resolve(res);
			}).catch((err) => {
				reject(err);
			});
		}));
	}
	return await productosPromiseMap.get(apiRoute);
};
enable_legacy_mode_flag();
const page$2 = {
	get data() {
		return page$3.data;
	},
	get error() {
		return page$3.error;
	},
	get form() {
		return page$3.form;
	},
	get params() {
		return page$3.params;
	},
	get route() {
		return page$3.route;
	},
	get state() {
		return page$3.state;
	},
	get status() {
		return page$3.status;
	},
	get url() {
		return page$3.url;
	}
};
Object.defineProperty({
	get from() {
		return navigating$1.current ? navigating$1.current.from : null;
	},
	get to() {
		return navigating$1.current ? navigating$1.current.to : null;
	},
	get type() {
		return navigating$1.current ? navigating$1.current.type : null;
	},
	get willUnload() {
		return navigating$1.current ? navigating$1.current.willUnload : null;
	},
	get delta() {
		return navigating$1.current ? navigating$1.current.delta : null;
	},
	get complete() {
		return navigating$1.current ? navigating$1.current.complete : null;
	}
}, "current", { get() {
	throw new Error("Replace navigating.current.<prop> with navigating.<prop>");
} });
stores.updated.check;
const page = page$2;
var root$15 = /* @__PURE__ */ from_html(`<h1> </h1> <p> </p>`, 1);
function Error$1($$anchor, $$props) {
	push($$props, false);
	init();
	var fragment = root$15();
	var h1 = first_child(fragment);
	var text = child(h1, true);
	reset(h1);
	var p = sibling(h1, 2);
	var text_1 = child(p, true);
	reset(p);
	template_effect(() => {
		set_text(text, page.status);
		set_text(text_1, page.error?.message);
	});
	append($$anchor, fragment);
	pop();
}
var components_module_default = {
	input: "afn",
	input_lab: "afo",
	input_div: "afp",
	input_div_1: "afq",
	input_inp: "afr",
	input_lab_cell_left: "afs",
	input_lab_cell_right: "aft",
	input_shadow_layer: "afu",
	card_image_1: "afv",
	card_input: "afw",
	card_image_img1: "afx",
	card_image_layer: "afy",
	card_image_layer_bn_close2: "afz",
	s1: "afA",
	card_image_btn: "afB",
	card_image_layer_loading: "afC",
	card_image_upload_text: "afD",
	card_image_layer_botton: "afE",
	card_image_textarea: "afF",
	card_input_layer: "afG",
	image_loading_layer: "afH",
	image_card_default: "afI",
	image_card_desc: "afJ",
	input_post_value: "afK"
};
var root_1$11 = /* @__PURE__ */ from_html(`<div><div></div></div> <div> <!></div> <div><div></div></div> <div><div></div></div>`, 1);
var root_2$6 = /* @__PURE__ */ from_html(`<div><div></div></div>`);
var root_3$5 = /* @__PURE__ */ from_html(`<textarea></textarea>`);
var root_4$3 = /* @__PURE__ */ from_html(`<input/>`);
var root_6$3 = /* @__PURE__ */ from_html(`<div> </div>`);
var root$14 = /* @__PURE__ */ from_html(`<div><!> <div><!> <!> <!> <!></div></div>`);
function Input($$anchor, $$props) {
	push($$props, true);
	const saveOn = prop($$props, "saveOn", 15);
	const baseDecimalsValue = /* @__PURE__ */ user_derived(() => $$props.baseDecimals ? 10 ** $$props.baseDecimals : 0);
	const checkIfInputIsValid = () => {
		if (!$$props.required || $$props.disabled) return 0;
		if (!saveOn() || !$$props.save) return 1;
		const value = saveOn()[$$props.save];
		let pass = !$$props.required;
		if ($$props.validator) pass = $$props.validator(value);
		else if (value || value === 0) pass = true;
		return pass ? 2 : 1;
	};
	let inputValue = /* @__PURE__ */ state("");
	let isInputValid = /* @__PURE__ */ state(proxy(checkIfInputIsValid()));
	let isChange = 0;
	let focusValue = null;
	const onKeyUp = (ev, isBlur) => {
		ev.stopPropagation();
		let value = ev.target.value;
		if ($$props.type === "number") {
			if (!isBlur && !value && ev.key === "-") return;
			if (isNaN(value)) value = void 0;
			else value = parseFloat(value);
		}
		if (isBlur && $$props.validator && !$$props.validator(value)) {
			set$1(inputValue, focusValue, true);
			if (saveOn() && $$props.save) {
				if (get$1(baseDecimalsValue) && typeof get$1(inputValue) === "number") set$1(inputValue, Math.round(get$1(inputValue) * get$1(baseDecimalsValue)), true);
				saveOn(saveOn()[$$props.save] = get$1(inputValue), true);
			}
			return;
		}
		if ($$props.transform && isBlur) value = $$props.transform(value);
		untrack$1(() => {
			if (saveOn() && $$props.save) {
				let valueSaved = value;
				if (get$1(baseDecimalsValue) && typeof valueSaved === "number") valueSaved = Math.round(valueSaved * get$1(baseDecimalsValue));
				saveOn(saveOn()[$$props.save] = valueSaved, true);
				set$1(isInputValid, checkIfInputIsValid(), true);
			}
		});
		if (!isBlur) isChange = 1;
		set$1(inputValue, value, true);
	};
	const iconValid = () => {
		if (!get$1(isInputValid)) return null;
		else if (get$1(isInputValid) === 2) return `<i class="v-icon icon-ok c-green"></i>`;
		else if (get$1(isInputValid) === 1) return `<i class="v-icon icon-attention text-red-500"></i>`;
		return null;
	};
	let lastSaveOn;
	const doSave = () => {
		untrack$1(() => {
			const v = saveOn()[$$props.save];
			set$1(inputValue, typeof v === "number" ? v : v || "", true);
			if (get$1(baseDecimalsValue) && typeof get$1(inputValue) === "number") set$1(inputValue, get$1(inputValue) / get$1(baseDecimalsValue));
			set$1(isInputValid, checkIfInputIsValid(), true);
		});
	};
	user_effect(() => {
		if (!saveOn() || !$$props.save) return;
		if (lastSaveOn === saveOn()) return;
		lastSaveOn = saveOn();
		if (saveOn()[$$props.save] !== get$1(inputValue)) doSave();
	});
	user_effect(() => {
		if ($$props.dependencyValue) doSave();
	});
	let cN = /* @__PURE__ */ user_derived(() => `${components_module_default.input} p-rel${$$props.css ? ` ${$$props.css}` : ""}${!$$props.label ? " no-label" : ""}`);
	var div = root$14();
	var node = child(div);
	var consequent = ($$anchor) => {
		var fragment = root_1$11();
		var div_1 = first_child(fragment);
		var div_2 = sibling(div_1, 2);
		var text = child(div_2, true);
		html(sibling(text), () => iconValid() || "");
		reset(div_2);
		var div_3 = sibling(div_2, 2);
		var div_4 = sibling(div_3, 2);
		template_effect(() => {
			set_class(div_1, 1, clsx(components_module_default.input_lab_cell_left));
			set_class(div_2, 1, clsx(components_module_default.input_lab));
			set_text(text, $$props.label);
			set_class(div_3, 1, clsx(components_module_default.input_lab_cell_right));
			set_class(div_4, 1, clsx(components_module_default.input_shadow_layer));
		});
		append($$anchor, fragment);
	};
	if_block(node, ($$render) => {
		if ($$props.label) $$render(consequent);
	});
	var div_5 = sibling(node, 2);
	var node_2 = child(div_5);
	var consequent_1 = ($$anchor) => {
		var div_6 = root_2$6();
		template_effect(() => set_class(div_6, 1, clsx(components_module_default.input_div_1)));
		append($$anchor, div_6);
	};
	if_block(node_2, ($$render) => {
		if ($$props.label) $$render(consequent_1);
	});
	var node_3 = sibling(node_2, 2);
	var consequent_2 = ($$anchor) => {
		var textarea = root_3$5();
		remove_textarea_child(textarea);
		textarea.__keyup = (ev) => {
			onKeyUp(ev);
		};
		template_effect(() => {
			set_class(textarea, 1, `w-full ${components_module_default.input_inp} ${$$props.inputCss || ""}`);
			set_attribute(textarea, "placeholder", $$props.placeholder || "");
			textarea.disabled = $$props.disabled;
			set_attribute(textarea, "rows", $$props.rows);
		});
		event("blur", textarea, (ev) => {
			console.log("input saveon:", snapshot(saveOn()));
			onKeyUp(ev, true);
			if ($$props.onChange && isChange) {
				$$props.onChange();
				isChange = 0;
			}
		});
		bind_value(textarea, () => get$1(inputValue), ($$value) => set$1(inputValue, $$value));
		append($$anchor, textarea);
	};
	var alternate = ($$anchor) => {
		var input = root_4$3();
		remove_input_defaults(input);
		input.__keyup = (ev) => {
			onKeyUp(ev);
		};
		template_effect(() => {
			set_class(input, 1, `w-full ${components_module_default.input_inp ?? ""} ${($$props.inputCss || "") ?? ""}`);
			set_attribute(input, "type", $$props.type || "search");
			set_attribute(input, "placeholder", $$props.placeholder || "");
			input.disabled = $$props.disabled;
		});
		event("focus", input, (ev) => {
			focusValue = ev.target.value;
		});
		event("blur", input, (ev) => {
			onKeyUp(ev, true);
			if ($$props.onChange && isChange) {
				$$props.onChange();
				isChange = 0;
			}
			focusValue = null;
		});
		bind_value(input, () => get$1(inputValue), ($$value) => set$1(inputValue, $$value));
		append($$anchor, input);
	};
	if_block(node_3, ($$render) => {
		if ($$props.useTextArea) $$render(consequent_2);
		else $$render(alternate, false);
	});
	var node_4 = sibling(node_3, 2);
	var consequent_3 = ($$anchor) => {
		var fragment_1 = comment();
		html(first_child(fragment_1), () => iconValid() || "");
		append($$anchor, fragment_1);
	};
	if_block(node_4, ($$render) => {
		if (!$$props.label) $$render(consequent_3);
	});
	var node_6 = sibling(node_4, 2);
	var consequent_4 = ($$anchor) => {
		var div_7 = root_6$3();
		var text_1 = child(div_7, true);
		reset(div_7);
		template_effect(() => {
			set_class(div_7, 1, clsx(components_module_default.input_post_value));
			set_text(text_1, $$props.postValue);
		});
		append($$anchor, div_7);
	};
	if_block(node_6, ($$render) => {
		if ($$props.postValue) $$render(consequent_4);
	});
	reset(div_5);
	reset(div);
	template_effect(() => {
		set_class(div, 1, clsx(get$1(cN)));
		set_class(div_5, 1, `${components_module_default.input_div} flex w-full`);
	});
	append($$anchor, div);
	pop();
}
delegate(["keyup"]);
function isObject$1(subject) {
	return Object.prototype.toString.call(subject) === "[object Object]";
}
function isRecord(subject) {
	return isObject$1(subject) || Array.isArray(subject);
}
function canUseDOM() {
	return !!(typeof window !== "undefined" && window.document && window.document.createElement);
}
function areOptionsEqual(optionsA, optionsB) {
	const optionsAKeys = Object.keys(optionsA);
	const optionsBKeys = Object.keys(optionsB);
	if (optionsAKeys.length !== optionsBKeys.length) return false;
	if (JSON.stringify(Object.keys(optionsA.breakpoints || {})) !== JSON.stringify(Object.keys(optionsB.breakpoints || {}))) return false;
	return optionsAKeys.every((key) => {
		const valueA = optionsA[key];
		const valueB = optionsB[key];
		if (typeof valueA === "function") return `${valueA}` === `${valueB}`;
		if (!isRecord(valueA) || !isRecord(valueB)) return valueA === valueB;
		return areOptionsEqual(valueA, valueB);
	});
}
function sortAndMapPluginToOptions(plugins) {
	return plugins.concat().sort((a, b) => a.name > b.name ? 1 : -1).map((plugin) => plugin.options);
}
function arePluginsEqual(pluginsA, pluginsB) {
	if (pluginsA.length !== pluginsB.length) return false;
	const optionsA = sortAndMapPluginToOptions(pluginsA);
	const optionsB = sortAndMapPluginToOptions(pluginsB);
	return optionsA.every((optionA, index) => {
		const optionB = optionsB[index];
		return areOptionsEqual(optionA, optionB);
	});
}
function isNumber(subject) {
	return typeof subject === "number";
}
function isString(subject) {
	return typeof subject === "string";
}
function isBoolean(subject) {
	return typeof subject === "boolean";
}
function isObject(subject) {
	return Object.prototype.toString.call(subject) === "[object Object]";
}
function mathAbs(n) {
	return Math.abs(n);
}
function mathSign(n) {
	return Math.sign(n);
}
function deltaAbs(valueB, valueA) {
	return mathAbs(valueB - valueA);
}
function factorAbs(valueB, valueA) {
	if (valueB === 0 || valueA === 0) return 0;
	if (mathAbs(valueB) <= mathAbs(valueA)) return 0;
	return mathAbs(deltaAbs(mathAbs(valueB), mathAbs(valueA)) / valueB);
}
function roundToTwoDecimals(num) {
	return Math.round(num * 100) / 100;
}
function arrayKeys(array) {
	return objectKeys(array).map(Number);
}
function arrayLast(array) {
	return array[arrayLastIndex(array)];
}
function arrayLastIndex(array) {
	return Math.max(0, array.length - 1);
}
function arrayIsLastIndex(array, index) {
	return index === arrayLastIndex(array);
}
function arrayFromNumber(n, startAt = 0) {
	return Array.from(Array(n), (_, i) => startAt + i);
}
function objectKeys(object) {
	return Object.keys(object);
}
function objectsMergeDeep(objectA, objectB) {
	return [objectA, objectB].reduce((mergedObjects, currentObject) => {
		objectKeys(currentObject).forEach((key) => {
			const valueA = mergedObjects[key];
			const valueB = currentObject[key];
			mergedObjects[key] = isObject(valueA) && isObject(valueB) ? objectsMergeDeep(valueA, valueB) : valueB;
		});
		return mergedObjects;
	}, {});
}
function isMouseEvent(evt, ownerWindow) {
	return typeof ownerWindow.MouseEvent !== "undefined" && evt instanceof ownerWindow.MouseEvent;
}
function Alignment(align, viewSize) {
	const predefined = {
		start,
		center,
		end
	};
	function start() {
		return 0;
	}
	function center(n) {
		return end(n) / 2;
	}
	function end(n) {
		return viewSize - n;
	}
	function measure(n, index) {
		if (isString(align)) return predefined[align](n);
		return align(viewSize, n, index);
	}
	return { measure };
}
function EventStore() {
	let listeners = [];
	function add(node, type, handler, options = { passive: true }) {
		let removeListener;
		if ("addEventListener" in node) {
			node.addEventListener(type, handler, options);
			removeListener = () => node.removeEventListener(type, handler, options);
		} else {
			const legacyMediaQueryList = node;
			legacyMediaQueryList.addListener(handler);
			removeListener = () => legacyMediaQueryList.removeListener(handler);
		}
		listeners.push(removeListener);
		return self;
	}
	function clear() {
		listeners = listeners.filter((remove) => remove());
	}
	const self = {
		add,
		clear
	};
	return self;
}
function Animations(ownerDocument, ownerWindow, update, render) {
	const documentVisibleHandler = EventStore();
	const fixedTimeStep = 1e3 / 60;
	let lastTimeStamp = null;
	let accumulatedTime = 0;
	let animationId = 0;
	function init() {
		documentVisibleHandler.add(ownerDocument, "visibilitychange", () => {
			if (ownerDocument.hidden) reset();
		});
	}
	function destroy() {
		stop();
		documentVisibleHandler.clear();
	}
	function animate(timeStamp) {
		if (!animationId) return;
		if (!lastTimeStamp) {
			lastTimeStamp = timeStamp;
			update();
			update();
		}
		const timeElapsed = timeStamp - lastTimeStamp;
		lastTimeStamp = timeStamp;
		accumulatedTime += timeElapsed;
		while (accumulatedTime >= fixedTimeStep) {
			update();
			accumulatedTime -= fixedTimeStep;
		}
		render(accumulatedTime / fixedTimeStep);
		if (animationId) animationId = ownerWindow.requestAnimationFrame(animate);
	}
	function start() {
		if (animationId) return;
		animationId = ownerWindow.requestAnimationFrame(animate);
	}
	function stop() {
		ownerWindow.cancelAnimationFrame(animationId);
		lastTimeStamp = null;
		accumulatedTime = 0;
		animationId = 0;
	}
	function reset() {
		lastTimeStamp = null;
		accumulatedTime = 0;
	}
	return {
		init,
		destroy,
		start,
		stop,
		update,
		render
	};
}
function Axis(axis, contentDirection) {
	const isRightToLeft = contentDirection === "rtl";
	const isVertical = axis === "y";
	const scroll = isVertical ? "y" : "x";
	const cross = isVertical ? "x" : "y";
	const sign = !isVertical && isRightToLeft ? -1 : 1;
	const startEdge = getStartEdge();
	const endEdge = getEndEdge();
	function measureSize(nodeRect) {
		const { height, width } = nodeRect;
		return isVertical ? height : width;
	}
	function getStartEdge() {
		if (isVertical) return "top";
		return isRightToLeft ? "right" : "left";
	}
	function getEndEdge() {
		if (isVertical) return "bottom";
		return isRightToLeft ? "left" : "right";
	}
	function direction(n) {
		return n * sign;
	}
	return {
		scroll,
		cross,
		startEdge,
		endEdge,
		measureSize,
		direction
	};
}
function Limit(min = 0, max = 0) {
	const length = mathAbs(min - max);
	function reachedMin(n) {
		return n < min;
	}
	function reachedMax(n) {
		return n > max;
	}
	function reachedAny(n) {
		return reachedMin(n) || reachedMax(n);
	}
	function constrain(n) {
		if (!reachedAny(n)) return n;
		return reachedMin(n) ? min : max;
	}
	function removeOffset(n) {
		if (!length) return n;
		return n - length * Math.ceil((n - max) / length);
	}
	return {
		length,
		max,
		min,
		constrain,
		reachedAny,
		reachedMax,
		reachedMin,
		removeOffset
	};
}
function Counter(max, start, loop) {
	const { constrain } = Limit(0, max);
	const loopEnd = max + 1;
	let counter = withinLimit(start);
	function withinLimit(n) {
		return !loop ? constrain(n) : mathAbs((loopEnd + n) % loopEnd);
	}
	function get() {
		return counter;
	}
	function set(n) {
		counter = withinLimit(n);
		return self;
	}
	function add(n) {
		return clone().set(get() + n);
	}
	function clone() {
		return Counter(max, get(), loop);
	}
	const self = {
		get,
		set,
		add,
		clone
	};
	return self;
}
function DragHandler(axis, rootNode, ownerDocument, ownerWindow, target, dragTracker, location, animation, scrollTo, scrollBody, scrollTarget, index, eventHandler, percentOfView, dragFree, dragThreshold, skipSnaps, baseFriction, watchDrag) {
	const { cross: crossAxis, direction } = axis;
	const focusNodes = [
		"INPUT",
		"SELECT",
		"TEXTAREA"
	];
	const nonPassiveEvent = { passive: false };
	const initEvents = EventStore();
	const dragEvents = EventStore();
	const goToNextThreshold = Limit(50, 225).constrain(percentOfView.measure(20));
	const snapForceBoost = {
		mouse: 300,
		touch: 400
	};
	const freeForceBoost = {
		mouse: 500,
		touch: 600
	};
	const baseSpeed = dragFree ? 43 : 25;
	let isMoving = false;
	let startScroll = 0;
	let startCross = 0;
	let pointerIsDown = false;
	let preventScroll = false;
	let preventClick = false;
	let isMouse = false;
	function init(emblaApi) {
		if (!watchDrag) return;
		function downIfAllowed(evt) {
			if (isBoolean(watchDrag) || watchDrag(emblaApi, evt)) down(evt);
		}
		const node = rootNode;
		initEvents.add(node, "dragstart", (evt) => evt.preventDefault(), nonPassiveEvent).add(node, "touchmove", () => void 0, nonPassiveEvent).add(node, "touchend", () => void 0).add(node, "touchstart", downIfAllowed).add(node, "mousedown", downIfAllowed).add(node, "touchcancel", up).add(node, "contextmenu", up).add(node, "click", click, true);
	}
	function destroy() {
		initEvents.clear();
		dragEvents.clear();
	}
	function addDragEvents() {
		const node = isMouse ? ownerDocument : rootNode;
		dragEvents.add(node, "touchmove", move, nonPassiveEvent).add(node, "touchend", up).add(node, "mousemove", move, nonPassiveEvent).add(node, "mouseup", up);
	}
	function isFocusNode(node) {
		const nodeName = node.nodeName || "";
		return focusNodes.includes(nodeName);
	}
	function forceBoost() {
		return (dragFree ? freeForceBoost : snapForceBoost)[isMouse ? "mouse" : "touch"];
	}
	function allowedForce(force, targetChanged) {
		const next = index.add(mathSign(force) * -1);
		const baseForce = scrollTarget.byDistance(force, !dragFree).distance;
		if (dragFree || mathAbs(force) < goToNextThreshold) return baseForce;
		if (skipSnaps && targetChanged) return baseForce * .5;
		return scrollTarget.byIndex(next.get(), 0).distance;
	}
	function down(evt) {
		const isMouseEvt = isMouseEvent(evt, ownerWindow);
		isMouse = isMouseEvt;
		preventClick = dragFree && isMouseEvt && !evt.buttons && isMoving;
		isMoving = deltaAbs(target.get(), location.get()) >= 2;
		if (isMouseEvt && evt.button !== 0) return;
		if (isFocusNode(evt.target)) return;
		pointerIsDown = true;
		dragTracker.pointerDown(evt);
		scrollBody.useFriction(0).useDuration(0);
		target.set(location);
		addDragEvents();
		startScroll = dragTracker.readPoint(evt);
		startCross = dragTracker.readPoint(evt, crossAxis);
		eventHandler.emit("pointerDown");
	}
	function move(evt) {
		if (!isMouseEvent(evt, ownerWindow) && evt.touches.length >= 2) return up(evt);
		const lastScroll = dragTracker.readPoint(evt);
		const lastCross = dragTracker.readPoint(evt, crossAxis);
		const diffScroll = deltaAbs(lastScroll, startScroll);
		const diffCross = deltaAbs(lastCross, startCross);
		if (!preventScroll && !isMouse) {
			if (!evt.cancelable) return up(evt);
			preventScroll = diffScroll > diffCross;
			if (!preventScroll) return up(evt);
		}
		const diff = dragTracker.pointerMove(evt);
		if (diffScroll > dragThreshold) preventClick = true;
		scrollBody.useFriction(.3).useDuration(.75);
		animation.start();
		target.add(direction(diff));
		evt.preventDefault();
	}
	function up(evt) {
		const targetChanged = scrollTarget.byDistance(0, false).index !== index.get();
		const rawForce = dragTracker.pointerUp(evt) * forceBoost();
		const force = allowedForce(direction(rawForce), targetChanged);
		const forceFactor = factorAbs(rawForce, force);
		const speed = baseSpeed - 10 * forceFactor;
		const friction = baseFriction + forceFactor / 50;
		preventScroll = false;
		pointerIsDown = false;
		dragEvents.clear();
		scrollBody.useDuration(speed).useFriction(friction);
		scrollTo.distance(force, !dragFree);
		isMouse = false;
		eventHandler.emit("pointerUp");
	}
	function click(evt) {
		if (preventClick) {
			evt.stopPropagation();
			evt.preventDefault();
			preventClick = false;
		}
	}
	function pointerDown() {
		return pointerIsDown;
	}
	return {
		init,
		destroy,
		pointerDown
	};
}
function DragTracker(axis, ownerWindow) {
	const logInterval = 170;
	let startEvent;
	let lastEvent;
	function readTime(evt) {
		return evt.timeStamp;
	}
	function readPoint(evt, evtAxis) {
		const coord = `client${(evtAxis || axis.scroll) === "x" ? "X" : "Y"}`;
		return (isMouseEvent(evt, ownerWindow) ? evt : evt.touches[0])[coord];
	}
	function pointerDown(evt) {
		startEvent = evt;
		lastEvent = evt;
		return readPoint(evt);
	}
	function pointerMove(evt) {
		const diff = readPoint(evt) - readPoint(lastEvent);
		const expired = readTime(evt) - readTime(startEvent) > logInterval;
		lastEvent = evt;
		if (expired) startEvent = evt;
		return diff;
	}
	function pointerUp(evt) {
		if (!startEvent || !lastEvent) return 0;
		const diffDrag = readPoint(lastEvent) - readPoint(startEvent);
		const diffTime = readTime(evt) - readTime(startEvent);
		const expired = readTime(evt) - readTime(lastEvent) > logInterval;
		const force = diffDrag / diffTime;
		return diffTime && !expired && mathAbs(force) > .1 ? force : 0;
	}
	return {
		pointerDown,
		pointerMove,
		pointerUp,
		readPoint
	};
}
function NodeRects() {
	function measure(node) {
		const { offsetTop, offsetLeft, offsetWidth, offsetHeight } = node;
		return {
			top: offsetTop,
			right: offsetLeft + offsetWidth,
			bottom: offsetTop + offsetHeight,
			left: offsetLeft,
			width: offsetWidth,
			height: offsetHeight
		};
	}
	return { measure };
}
function PercentOfView(viewSize) {
	function measure(n) {
		return viewSize * (n / 100);
	}
	return { measure };
}
function ResizeHandler(container, eventHandler, ownerWindow, slides, axis, watchResize, nodeRects) {
	const observeNodes = [container].concat(slides);
	let resizeObserver;
	let containerSize;
	let slideSizes = [];
	let destroyed = false;
	function readSize(node) {
		return axis.measureSize(nodeRects.measure(node));
	}
	function init(emblaApi) {
		if (!watchResize) return;
		containerSize = readSize(container);
		slideSizes = slides.map(readSize);
		function defaultCallback(entries) {
			for (const entry of entries) {
				if (destroyed) return;
				const isContainer = entry.target === container;
				const slideIndex = slides.indexOf(entry.target);
				const lastSize = isContainer ? containerSize : slideSizes[slideIndex];
				if (mathAbs(readSize(isContainer ? container : slides[slideIndex]) - lastSize) >= .5) {
					emblaApi.reInit();
					eventHandler.emit("resize");
					break;
				}
			}
		}
		resizeObserver = new ResizeObserver((entries) => {
			if (isBoolean(watchResize) || watchResize(emblaApi, entries)) defaultCallback(entries);
		});
		ownerWindow.requestAnimationFrame(() => {
			observeNodes.forEach((node) => resizeObserver.observe(node));
		});
	}
	function destroy() {
		destroyed = true;
		if (resizeObserver) resizeObserver.disconnect();
	}
	return {
		init,
		destroy
	};
}
function ScrollBody(location, offsetLocation, previousLocation, target, baseDuration, baseFriction) {
	let scrollVelocity = 0;
	let scrollDirection = 0;
	let scrollDuration = baseDuration;
	let scrollFriction = baseFriction;
	let rawLocation = location.get();
	let rawLocationPrevious = 0;
	function seek() {
		const displacement = target.get() - location.get();
		const isInstant = !scrollDuration;
		let scrollDistance = 0;
		if (isInstant) {
			scrollVelocity = 0;
			previousLocation.set(target);
			location.set(target);
			scrollDistance = displacement;
		} else {
			previousLocation.set(location);
			scrollVelocity += displacement / scrollDuration;
			scrollVelocity *= scrollFriction;
			rawLocation += scrollVelocity;
			location.add(scrollVelocity);
			scrollDistance = rawLocation - rawLocationPrevious;
		}
		scrollDirection = mathSign(scrollDistance);
		rawLocationPrevious = rawLocation;
		return self;
	}
	function settled() {
		return mathAbs(target.get() - offsetLocation.get()) < .001;
	}
	function duration() {
		return scrollDuration;
	}
	function direction() {
		return scrollDirection;
	}
	function velocity() {
		return scrollVelocity;
	}
	function useBaseDuration() {
		return useDuration(baseDuration);
	}
	function useBaseFriction() {
		return useFriction(baseFriction);
	}
	function useDuration(n) {
		scrollDuration = n;
		return self;
	}
	function useFriction(n) {
		scrollFriction = n;
		return self;
	}
	const self = {
		direction,
		duration,
		velocity,
		seek,
		settled,
		useBaseFriction,
		useBaseDuration,
		useFriction,
		useDuration
	};
	return self;
}
function ScrollBounds(limit, location, target, scrollBody, percentOfView) {
	const pullBackThreshold = percentOfView.measure(10);
	const edgeOffsetTolerance = percentOfView.measure(50);
	const frictionLimit = Limit(.1, .99);
	let disabled = false;
	function shouldConstrain() {
		if (disabled) return false;
		if (!limit.reachedAny(target.get())) return false;
		if (!limit.reachedAny(location.get())) return false;
		return true;
	}
	function constrain(pointerDown) {
		if (!shouldConstrain()) return;
		const diffToEdge = mathAbs(limit[limit.reachedMin(location.get()) ? "min" : "max"] - location.get());
		const diffToTarget = target.get() - location.get();
		const friction = frictionLimit.constrain(diffToEdge / edgeOffsetTolerance);
		target.subtract(diffToTarget * friction);
		if (!pointerDown && mathAbs(diffToTarget) < pullBackThreshold) {
			target.set(limit.constrain(target.get()));
			scrollBody.useDuration(25).useBaseFriction();
		}
	}
	function toggleActive(active) {
		disabled = !active;
	}
	return {
		shouldConstrain,
		constrain,
		toggleActive
	};
}
function ScrollContain(viewSize, contentSize, snapsAligned, containScroll, pixelTolerance) {
	const scrollBounds = Limit(-contentSize + viewSize, 0);
	const snapsBounded = measureBounded();
	const scrollContainLimit = findScrollContainLimit();
	const snapsContained = measureContained();
	function usePixelTolerance(bound, snap) {
		return deltaAbs(bound, snap) <= 1;
	}
	function findScrollContainLimit() {
		const startSnap = snapsBounded[0];
		const endSnap = arrayLast(snapsBounded);
		return Limit(snapsBounded.lastIndexOf(startSnap), snapsBounded.indexOf(endSnap) + 1);
	}
	function measureBounded() {
		return snapsAligned.map((snapAligned, index) => {
			const { min, max } = scrollBounds;
			const snap = scrollBounds.constrain(snapAligned);
			const isFirst = !index;
			const isLast = arrayIsLastIndex(snapsAligned, index);
			if (isFirst) return max;
			if (isLast) return min;
			if (usePixelTolerance(min, snap)) return min;
			if (usePixelTolerance(max, snap)) return max;
			return snap;
		}).map((scrollBound) => parseFloat(scrollBound.toFixed(3)));
	}
	function measureContained() {
		if (contentSize <= viewSize + pixelTolerance) return [scrollBounds.max];
		if (containScroll === "keepSnaps") return snapsBounded;
		const { min, max } = scrollContainLimit;
		return snapsBounded.slice(min, max);
	}
	return {
		snapsContained,
		scrollContainLimit
	};
}
function ScrollLimit(contentSize, scrollSnaps, loop) {
	const max = scrollSnaps[0];
	return { limit: Limit(loop ? max - contentSize : arrayLast(scrollSnaps), max) };
}
function ScrollLooper(contentSize, limit, location, vectors) {
	const jointSafety = .1;
	const { reachedMin, reachedMax } = Limit(limit.min + jointSafety, limit.max + jointSafety);
	function shouldLoop(direction) {
		if (direction === 1) return reachedMax(location.get());
		if (direction === -1) return reachedMin(location.get());
		return false;
	}
	function loop(direction) {
		if (!shouldLoop(direction)) return;
		const loopDistance = contentSize * (direction * -1);
		vectors.forEach((v) => v.add(loopDistance));
	}
	return { loop };
}
function ScrollProgress(limit) {
	const { max, length } = limit;
	function get(n) {
		const currentLocation = n - max;
		return length ? currentLocation / -length : 0;
	}
	return { get };
}
function ScrollSnaps(axis, alignment, containerRect, slideRects, slidesToScroll) {
	const { startEdge, endEdge } = axis;
	const { groupSlides } = slidesToScroll;
	const alignments = measureSizes().map(alignment.measure);
	const snaps = measureUnaligned();
	const snapsAligned = measureAligned();
	function measureSizes() {
		return groupSlides(slideRects).map((rects) => arrayLast(rects)[endEdge] - rects[0][startEdge]).map(mathAbs);
	}
	function measureUnaligned() {
		return slideRects.map((rect) => containerRect[startEdge] - rect[startEdge]).map((snap) => -mathAbs(snap));
	}
	function measureAligned() {
		return groupSlides(snaps).map((g) => g[0]).map((snap, index) => snap + alignments[index]);
	}
	return {
		snaps,
		snapsAligned
	};
}
function SlideRegistry(containSnaps, containScroll, scrollSnaps, scrollContainLimit, slidesToScroll, slideIndexes) {
	const { groupSlides } = slidesToScroll;
	const { min, max } = scrollContainLimit;
	const slideRegistry = createSlideRegistry();
	function createSlideRegistry() {
		const groupedSlideIndexes = groupSlides(slideIndexes);
		const doNotContain = !containSnaps || containScroll === "keepSnaps";
		if (scrollSnaps.length === 1) return [slideIndexes];
		if (doNotContain) return groupedSlideIndexes;
		return groupedSlideIndexes.slice(min, max).map((group, index, groups) => {
			const isFirst = !index;
			const isLast = arrayIsLastIndex(groups, index);
			if (isFirst) return arrayFromNumber(arrayLast(groups[0]) + 1);
			if (isLast) return arrayFromNumber(arrayLastIndex(slideIndexes) - arrayLast(groups)[0] + 1, arrayLast(groups)[0]);
			return group;
		});
	}
	return { slideRegistry };
}
function ScrollTarget(loop, scrollSnaps, contentSize, limit, targetVector) {
	const { reachedAny, removeOffset, constrain } = limit;
	function minDistance(distances) {
		return distances.concat().sort((a, b) => mathAbs(a) - mathAbs(b))[0];
	}
	function findTargetSnap(target) {
		const distance = loop ? removeOffset(target) : constrain(target);
		const { index } = scrollSnaps.map((snap, index) => ({
			diff: shortcut(snap - distance, 0),
			index
		})).sort((d1, d2) => mathAbs(d1.diff) - mathAbs(d2.diff))[0];
		return {
			index,
			distance
		};
	}
	function shortcut(target, direction) {
		const targets = [
			target,
			target + contentSize,
			target - contentSize
		];
		if (!loop) return target;
		if (!direction) return minDistance(targets);
		const matchingTargets = targets.filter((t) => mathSign(t) === direction);
		if (matchingTargets.length) return minDistance(matchingTargets);
		return arrayLast(targets) - contentSize;
	}
	function byIndex(index, direction) {
		return {
			index,
			distance: shortcut(scrollSnaps[index] - targetVector.get(), direction)
		};
	}
	function byDistance(distance, snap) {
		const target = targetVector.get() + distance;
		const { index, distance: targetSnapDistance } = findTargetSnap(target);
		const reachedBound = !loop && reachedAny(target);
		if (!snap || reachedBound) return {
			index,
			distance
		};
		return {
			index,
			distance: distance + shortcut(scrollSnaps[index] - targetSnapDistance, 0)
		};
	}
	return {
		byDistance,
		byIndex,
		shortcut
	};
}
function ScrollTo(animation, indexCurrent, indexPrevious, scrollBody, scrollTarget, targetVector, eventHandler) {
	function scrollTo(target) {
		const distanceDiff = target.distance;
		const indexDiff = target.index !== indexCurrent.get();
		targetVector.add(distanceDiff);
		if (distanceDiff) if (scrollBody.duration()) animation.start();
		else {
			animation.update();
			animation.render(1);
			animation.update();
		}
		if (indexDiff) {
			indexPrevious.set(indexCurrent.get());
			indexCurrent.set(target.index);
			eventHandler.emit("select");
		}
	}
	function distance(n, snap) {
		scrollTo(scrollTarget.byDistance(n, snap));
	}
	function index(n, direction) {
		const targetIndex = indexCurrent.clone().set(n);
		scrollTo(scrollTarget.byIndex(targetIndex.get(), direction));
	}
	return {
		distance,
		index
	};
}
function SlideFocus(root, slides, slideRegistry, scrollTo, scrollBody, eventStore, eventHandler, watchFocus) {
	const focusListenerOptions = {
		passive: true,
		capture: true
	};
	let lastTabPressTime = 0;
	function init(emblaApi) {
		if (!watchFocus) return;
		function defaultCallback(index) {
			if ((/* @__PURE__ */ new Date()).getTime() - lastTabPressTime > 10) return;
			eventHandler.emit("slideFocusStart");
			root.scrollLeft = 0;
			const group = slideRegistry.findIndex((group) => group.includes(index));
			if (!isNumber(group)) return;
			scrollBody.useDuration(0);
			scrollTo.index(group, 0);
			eventHandler.emit("slideFocus");
		}
		eventStore.add(document, "keydown", registerTabPress, false);
		slides.forEach((slide, slideIndex) => {
			eventStore.add(slide, "focus", (evt) => {
				if (isBoolean(watchFocus) || watchFocus(emblaApi, evt)) defaultCallback(slideIndex);
			}, focusListenerOptions);
		});
	}
	function registerTabPress(event) {
		if (event.code === "Tab") lastTabPressTime = (/* @__PURE__ */ new Date()).getTime();
	}
	return { init };
}
function Vector1D(initialValue) {
	let value = initialValue;
	function get() {
		return value;
	}
	function set(n) {
		value = normalizeInput(n);
	}
	function add(n) {
		value += normalizeInput(n);
	}
	function subtract(n) {
		value -= normalizeInput(n);
	}
	function normalizeInput(n) {
		return isNumber(n) ? n : n.get();
	}
	return {
		get,
		set,
		add,
		subtract
	};
}
function Translate(axis, container) {
	const translate = axis.scroll === "x" ? x : y;
	const containerStyle = container.style;
	let previousTarget = null;
	let disabled = false;
	function x(n) {
		return `translate3d(${n}px,0px,0px)`;
	}
	function y(n) {
		return `translate3d(0px,${n}px,0px)`;
	}
	function to(target) {
		if (disabled) return;
		const newTarget = roundToTwoDecimals(axis.direction(target));
		if (newTarget === previousTarget) return;
		containerStyle.transform = translate(newTarget);
		previousTarget = newTarget;
	}
	function toggleActive(active) {
		disabled = !active;
	}
	function clear() {
		if (disabled) return;
		containerStyle.transform = "";
		if (!container.getAttribute("style")) container.removeAttribute("style");
	}
	return {
		clear,
		to,
		toggleActive
	};
}
function SlideLooper(axis, viewSize, contentSize, slideSizes, slideSizesWithGaps, snaps, scrollSnaps, location, slides) {
	const roundingSafety = .5;
	const ascItems = arrayKeys(slideSizesWithGaps);
	const descItems = arrayKeys(slideSizesWithGaps).reverse();
	const loopPoints = startPoints().concat(endPoints());
	function removeSlideSizes(indexes, from) {
		return indexes.reduce((a, i) => {
			return a - slideSizesWithGaps[i];
		}, from);
	}
	function slidesInGap(indexes, gap) {
		return indexes.reduce((a, i) => {
			return removeSlideSizes(a, gap) > 0 ? a.concat([i]) : a;
		}, []);
	}
	function findSlideBounds(offset) {
		return snaps.map((snap, index) => ({
			start: snap - slideSizes[index] + roundingSafety + offset,
			end: snap + viewSize - roundingSafety + offset
		}));
	}
	function findLoopPoints(indexes, offset, isEndEdge) {
		const slideBounds = findSlideBounds(offset);
		return indexes.map((index) => {
			const initial = isEndEdge ? 0 : -contentSize;
			const altered = isEndEdge ? contentSize : 0;
			const boundEdge = isEndEdge ? "end" : "start";
			const loopPoint = slideBounds[index][boundEdge];
			return {
				index,
				loopPoint,
				slideLocation: Vector1D(-1),
				translate: Translate(axis, slides[index]),
				target: () => location.get() > loopPoint ? initial : altered
			};
		});
	}
	function startPoints() {
		const gap = scrollSnaps[0];
		return findLoopPoints(slidesInGap(descItems, gap), contentSize, false);
	}
	function endPoints() {
		return findLoopPoints(slidesInGap(ascItems, viewSize - scrollSnaps[0] - 1), -contentSize, true);
	}
	function canLoop() {
		return loopPoints.every(({ index }) => {
			return removeSlideSizes(ascItems.filter((i) => i !== index), viewSize) <= .1;
		});
	}
	function loop() {
		loopPoints.forEach((loopPoint) => {
			const { target, translate, slideLocation } = loopPoint;
			const shiftLocation = target();
			if (shiftLocation === slideLocation.get()) return;
			translate.to(shiftLocation);
			slideLocation.set(shiftLocation);
		});
	}
	function clear() {
		loopPoints.forEach((loopPoint) => loopPoint.translate.clear());
	}
	return {
		canLoop,
		clear,
		loop,
		loopPoints
	};
}
function SlidesHandler(container, eventHandler, watchSlides) {
	let mutationObserver;
	let destroyed = false;
	function init(emblaApi) {
		if (!watchSlides) return;
		function defaultCallback(mutations) {
			for (const mutation of mutations) if (mutation.type === "childList") {
				emblaApi.reInit();
				eventHandler.emit("slidesChanged");
				break;
			}
		}
		mutationObserver = new MutationObserver((mutations) => {
			if (destroyed) return;
			if (isBoolean(watchSlides) || watchSlides(emblaApi, mutations)) defaultCallback(mutations);
		});
		mutationObserver.observe(container, { childList: true });
	}
	function destroy() {
		if (mutationObserver) mutationObserver.disconnect();
		destroyed = true;
	}
	return {
		init,
		destroy
	};
}
function SlidesInView(container, slides, eventHandler, threshold) {
	const intersectionEntryMap = {};
	let inViewCache = null;
	let notInViewCache = null;
	let intersectionObserver;
	let destroyed = false;
	function init() {
		intersectionObserver = new IntersectionObserver((entries) => {
			if (destroyed) return;
			entries.forEach((entry) => {
				const index = slides.indexOf(entry.target);
				intersectionEntryMap[index] = entry;
			});
			inViewCache = null;
			notInViewCache = null;
			eventHandler.emit("slidesInView");
		}, {
			root: container.parentElement,
			threshold
		});
		slides.forEach((slide) => intersectionObserver.observe(slide));
	}
	function destroy() {
		if (intersectionObserver) intersectionObserver.disconnect();
		destroyed = true;
	}
	function createInViewList(inView) {
		return objectKeys(intersectionEntryMap).reduce((list, slideIndex) => {
			const index = parseInt(slideIndex);
			const { isIntersecting } = intersectionEntryMap[index];
			if (inView && isIntersecting || !inView && !isIntersecting) list.push(index);
			return list;
		}, []);
	}
	function get(inView = true) {
		if (inView && inViewCache) return inViewCache;
		if (!inView && notInViewCache) return notInViewCache;
		const slideIndexes = createInViewList(inView);
		if (inView) inViewCache = slideIndexes;
		if (!inView) notInViewCache = slideIndexes;
		return slideIndexes;
	}
	return {
		init,
		destroy,
		get
	};
}
function SlideSizes(axis, containerRect, slideRects, slides, readEdgeGap, ownerWindow) {
	const { measureSize, startEdge, endEdge } = axis;
	const withEdgeGap = slideRects[0] && readEdgeGap;
	const startGap = measureStartGap();
	const endGap = measureEndGap();
	const slideSizes = slideRects.map(measureSize);
	const slideSizesWithGaps = measureWithGaps();
	function measureStartGap() {
		if (!withEdgeGap) return 0;
		const slideRect = slideRects[0];
		return mathAbs(containerRect[startEdge] - slideRect[startEdge]);
	}
	function measureEndGap() {
		if (!withEdgeGap) return 0;
		const style = ownerWindow.getComputedStyle(arrayLast(slides));
		return parseFloat(style.getPropertyValue(`margin-${endEdge}`));
	}
	function measureWithGaps() {
		return slideRects.map((rect, index, rects) => {
			const isFirst = !index;
			const isLast = arrayIsLastIndex(rects, index);
			if (isFirst) return slideSizes[index] + startGap;
			if (isLast) return slideSizes[index] + endGap;
			return rects[index + 1][startEdge] - rect[startEdge];
		}).map(mathAbs);
	}
	return {
		slideSizes,
		slideSizesWithGaps,
		startGap,
		endGap
	};
}
function SlidesToScroll(axis, viewSize, slidesToScroll, loop, containerRect, slideRects, startGap, endGap, pixelTolerance) {
	const { startEdge, endEdge, direction } = axis;
	const groupByNumber = isNumber(slidesToScroll);
	function byNumber(array, groupSize) {
		return arrayKeys(array).filter((i) => i % groupSize === 0).map((i) => array.slice(i, i + groupSize));
	}
	function bySize(array) {
		if (!array.length) return [];
		return arrayKeys(array).reduce((groups, rectB, index) => {
			const rectA = arrayLast(groups) || 0;
			const isFirst = rectA === 0;
			const isLast = rectB === arrayLastIndex(array);
			const edgeA = containerRect[startEdge] - slideRects[rectA][startEdge];
			const edgeB = containerRect[startEdge] - slideRects[rectB][endEdge];
			const gapA = !loop && isFirst ? direction(startGap) : 0;
			const chunkSize = mathAbs(edgeB - (!loop && isLast ? direction(endGap) : 0) - (edgeA + gapA));
			if (index && chunkSize > viewSize + pixelTolerance) groups.push(rectB);
			if (isLast) groups.push(array.length);
			return groups;
		}, []).map((currentSize, index, groups) => {
			const previousSize = Math.max(groups[index - 1] || 0);
			return array.slice(previousSize, currentSize);
		});
	}
	function groupSlides(array) {
		return groupByNumber ? byNumber(array, slidesToScroll) : bySize(array);
	}
	return { groupSlides };
}
function Engine(root, container, slides, ownerDocument, ownerWindow, options, eventHandler) {
	const { align, axis: scrollAxis, direction, startIndex, loop, duration, dragFree, dragThreshold, inViewThreshold, slidesToScroll: groupSlides, skipSnaps, containScroll, watchResize, watchSlides, watchDrag, watchFocus } = options;
	const pixelTolerance = 2;
	const nodeRects = NodeRects();
	const containerRect = nodeRects.measure(container);
	const slideRects = slides.map(nodeRects.measure);
	const axis = Axis(scrollAxis, direction);
	const viewSize = axis.measureSize(containerRect);
	const percentOfView = PercentOfView(viewSize);
	const alignment = Alignment(align, viewSize);
	const containSnaps = !loop && !!containScroll;
	const { slideSizes, slideSizesWithGaps, startGap, endGap } = SlideSizes(axis, containerRect, slideRects, slides, loop || !!containScroll, ownerWindow);
	const slidesToScroll = SlidesToScroll(axis, viewSize, groupSlides, loop, containerRect, slideRects, startGap, endGap, pixelTolerance);
	const { snaps, snapsAligned } = ScrollSnaps(axis, alignment, containerRect, slideRects, slidesToScroll);
	const contentSize = -arrayLast(snaps) + arrayLast(slideSizesWithGaps);
	const { snapsContained, scrollContainLimit } = ScrollContain(viewSize, contentSize, snapsAligned, containScroll, pixelTolerance);
	const scrollSnaps = containSnaps ? snapsContained : snapsAligned;
	const { limit } = ScrollLimit(contentSize, scrollSnaps, loop);
	const index = Counter(arrayLastIndex(scrollSnaps), startIndex, loop);
	const indexPrevious = index.clone();
	const slideIndexes = arrayKeys(slides);
	const update = ({ dragHandler, scrollBody, scrollBounds, options: { loop } }) => {
		if (!loop) scrollBounds.constrain(dragHandler.pointerDown());
		scrollBody.seek();
	};
	const render = ({ scrollBody, translate, location, offsetLocation, previousLocation, scrollLooper, slideLooper, dragHandler, animation, eventHandler, scrollBounds, options: { loop } }, alpha) => {
		const shouldSettle = scrollBody.settled();
		const withinBounds = !scrollBounds.shouldConstrain();
		const hasSettled = loop ? shouldSettle : shouldSettle && withinBounds;
		const hasSettledAndIdle = hasSettled && !dragHandler.pointerDown();
		if (hasSettledAndIdle) animation.stop();
		const interpolatedLocation = location.get() * alpha + previousLocation.get() * (1 - alpha);
		offsetLocation.set(interpolatedLocation);
		if (loop) {
			scrollLooper.loop(scrollBody.direction());
			slideLooper.loop();
		}
		translate.to(offsetLocation.get());
		if (hasSettledAndIdle) eventHandler.emit("settle");
		if (!hasSettled) eventHandler.emit("scroll");
	};
	const animation = Animations(ownerDocument, ownerWindow, () => update(engine), (alpha) => render(engine, alpha));
	const friction = .68;
	const startLocation = scrollSnaps[index.get()];
	const location = Vector1D(startLocation);
	const previousLocation = Vector1D(startLocation);
	const offsetLocation = Vector1D(startLocation);
	const target = Vector1D(startLocation);
	const scrollBody = ScrollBody(location, offsetLocation, previousLocation, target, duration, friction);
	const scrollTarget = ScrollTarget(loop, scrollSnaps, contentSize, limit, target);
	const scrollTo = ScrollTo(animation, index, indexPrevious, scrollBody, scrollTarget, target, eventHandler);
	const scrollProgress = ScrollProgress(limit);
	const eventStore = EventStore();
	const slidesInView = SlidesInView(container, slides, eventHandler, inViewThreshold);
	const { slideRegistry } = SlideRegistry(containSnaps, containScroll, scrollSnaps, scrollContainLimit, slidesToScroll, slideIndexes);
	const slideFocus = SlideFocus(root, slides, slideRegistry, scrollTo, scrollBody, eventStore, eventHandler, watchFocus);
	const engine = {
		ownerDocument,
		ownerWindow,
		eventHandler,
		containerRect,
		slideRects,
		animation,
		axis,
		dragHandler: DragHandler(axis, root, ownerDocument, ownerWindow, target, DragTracker(axis, ownerWindow), location, animation, scrollTo, scrollBody, scrollTarget, index, eventHandler, percentOfView, dragFree, dragThreshold, skipSnaps, friction, watchDrag),
		eventStore,
		percentOfView,
		index,
		indexPrevious,
		limit,
		location,
		offsetLocation,
		previousLocation,
		options,
		resizeHandler: ResizeHandler(container, eventHandler, ownerWindow, slides, axis, watchResize, nodeRects),
		scrollBody,
		scrollBounds: ScrollBounds(limit, offsetLocation, target, scrollBody, percentOfView),
		scrollLooper: ScrollLooper(contentSize, limit, offsetLocation, [
			location,
			offsetLocation,
			previousLocation,
			target
		]),
		scrollProgress,
		scrollSnapList: scrollSnaps.map(scrollProgress.get),
		scrollSnaps,
		scrollTarget,
		scrollTo,
		slideLooper: SlideLooper(axis, viewSize, contentSize, slideSizes, slideSizesWithGaps, snaps, scrollSnaps, offsetLocation, slides),
		slideFocus,
		slidesHandler: SlidesHandler(container, eventHandler, watchSlides),
		slidesInView,
		slideIndexes,
		slideRegistry,
		slidesToScroll,
		target,
		translate: Translate(axis, container)
	};
	return engine;
}
function EventHandler() {
	let listeners = {};
	let api;
	function init(emblaApi) {
		api = emblaApi;
	}
	function getListeners(evt) {
		return listeners[evt] || [];
	}
	function emit(evt) {
		getListeners(evt).forEach((e) => e(api, evt));
		return self;
	}
	function on(evt, cb) {
		listeners[evt] = getListeners(evt).concat([cb]);
		return self;
	}
	function off(evt, cb) {
		listeners[evt] = getListeners(evt).filter((e) => e !== cb);
		return self;
	}
	function clear() {
		listeners = {};
	}
	const self = {
		init,
		emit,
		off,
		on,
		clear
	};
	return self;
}
var defaultOptions = {
	align: "center",
	axis: "x",
	container: null,
	slides: null,
	containScroll: "trimSnaps",
	direction: "ltr",
	slidesToScroll: 1,
	inViewThreshold: 0,
	breakpoints: {},
	dragFree: false,
	dragThreshold: 10,
	loop: false,
	skipSnaps: false,
	duration: 25,
	startIndex: 0,
	active: true,
	watchDrag: true,
	watchResize: true,
	watchSlides: true,
	watchFocus: true
};
function OptionsHandler(ownerWindow) {
	function mergeOptions(optionsA, optionsB) {
		return objectsMergeDeep(optionsA, optionsB || {});
	}
	function optionsAtMedia(options) {
		const optionsAtMedia = options.breakpoints || {};
		return mergeOptions(options, objectKeys(optionsAtMedia).filter((media) => ownerWindow.matchMedia(media).matches).map((media) => optionsAtMedia[media]).reduce((a, mediaOption) => mergeOptions(a, mediaOption), {}));
	}
	function optionsMediaQueries(optionsList) {
		return optionsList.map((options) => objectKeys(options.breakpoints || {})).reduce((acc, mediaQueries) => acc.concat(mediaQueries), []).map(ownerWindow.matchMedia);
	}
	return {
		mergeOptions,
		optionsAtMedia,
		optionsMediaQueries
	};
}
function PluginsHandler(optionsHandler) {
	let activePlugins = [];
	function init(emblaApi, plugins) {
		activePlugins = plugins.filter(({ options }) => optionsHandler.optionsAtMedia(options).active !== false);
		activePlugins.forEach((plugin) => plugin.init(emblaApi, optionsHandler));
		return plugins.reduce((map, plugin) => Object.assign(map, { [plugin.name]: plugin }), {});
	}
	function destroy() {
		activePlugins = activePlugins.filter((plugin) => plugin.destroy());
	}
	return {
		init,
		destroy
	};
}
function EmblaCarousel(root, userOptions, userPlugins) {
	const ownerDocument = root.ownerDocument;
	const ownerWindow = ownerDocument.defaultView;
	const optionsHandler = OptionsHandler(ownerWindow);
	const pluginsHandler = PluginsHandler(optionsHandler);
	const mediaHandlers = EventStore();
	const eventHandler = EventHandler();
	const { mergeOptions, optionsAtMedia, optionsMediaQueries } = optionsHandler;
	const { on, off, emit } = eventHandler;
	const reInit = reActivate;
	let destroyed = false;
	let engine;
	let optionsBase = mergeOptions(defaultOptions, EmblaCarousel.globalOptions);
	let options = mergeOptions(optionsBase);
	let pluginList = [];
	let pluginApis;
	let container;
	let slides;
	function storeElements() {
		const { container: userContainer, slides: userSlides } = options;
		container = (isString(userContainer) ? root.querySelector(userContainer) : userContainer) || root.children[0];
		const customSlides = isString(userSlides) ? container.querySelectorAll(userSlides) : userSlides;
		slides = [].slice.call(customSlides || container.children);
	}
	function createEngine(options) {
		const engine = Engine(root, container, slides, ownerDocument, ownerWindow, options, eventHandler);
		if (options.loop && !engine.slideLooper.canLoop()) return createEngine(Object.assign({}, options, { loop: false }));
		return engine;
	}
	function activate(withOptions, withPlugins) {
		if (destroyed) return;
		optionsBase = mergeOptions(optionsBase, withOptions);
		options = optionsAtMedia(optionsBase);
		pluginList = withPlugins || pluginList;
		storeElements();
		engine = createEngine(options);
		optionsMediaQueries([optionsBase, ...pluginList.map(({ options }) => options)]).forEach((query) => mediaHandlers.add(query, "change", reActivate));
		if (!options.active) return;
		engine.translate.to(engine.location.get());
		engine.animation.init();
		engine.slidesInView.init();
		engine.slideFocus.init(self);
		engine.eventHandler.init(self);
		engine.resizeHandler.init(self);
		engine.slidesHandler.init(self);
		if (engine.options.loop) engine.slideLooper.loop();
		if (container.offsetParent && slides.length) engine.dragHandler.init(self);
		pluginApis = pluginsHandler.init(self, pluginList);
	}
	function reActivate(withOptions, withPlugins) {
		const startIndex = selectedScrollSnap();
		deActivate();
		activate(mergeOptions({ startIndex }, withOptions), withPlugins);
		eventHandler.emit("reInit");
	}
	function deActivate() {
		engine.dragHandler.destroy();
		engine.eventStore.clear();
		engine.translate.clear();
		engine.slideLooper.clear();
		engine.resizeHandler.destroy();
		engine.slidesHandler.destroy();
		engine.slidesInView.destroy();
		engine.animation.destroy();
		pluginsHandler.destroy();
		mediaHandlers.clear();
	}
	function destroy() {
		if (destroyed) return;
		destroyed = true;
		mediaHandlers.clear();
		deActivate();
		eventHandler.emit("destroy");
		eventHandler.clear();
	}
	function scrollTo(index, jump, direction) {
		if (!options.active || destroyed) return;
		engine.scrollBody.useBaseFriction().useDuration(jump === true ? 0 : options.duration);
		engine.scrollTo.index(index, direction || 0);
	}
	function scrollNext(jump) {
		scrollTo(engine.index.add(1).get(), jump, -1);
	}
	function scrollPrev(jump) {
		scrollTo(engine.index.add(-1).get(), jump, 1);
	}
	function canScrollNext() {
		return engine.index.add(1).get() !== selectedScrollSnap();
	}
	function canScrollPrev() {
		return engine.index.add(-1).get() !== selectedScrollSnap();
	}
	function scrollSnapList() {
		return engine.scrollSnapList;
	}
	function scrollProgress() {
		return engine.scrollProgress.get(engine.offsetLocation.get());
	}
	function selectedScrollSnap() {
		return engine.index.get();
	}
	function previousScrollSnap() {
		return engine.indexPrevious.get();
	}
	function slidesInView() {
		return engine.slidesInView.get();
	}
	function slidesNotInView() {
		return engine.slidesInView.get(false);
	}
	function plugins() {
		return pluginApis;
	}
	function internalEngine() {
		return engine;
	}
	function rootNode() {
		return root;
	}
	function containerNode() {
		return container;
	}
	function slideNodes() {
		return slides;
	}
	const self = {
		canScrollNext,
		canScrollPrev,
		containerNode,
		internalEngine,
		destroy,
		off,
		on,
		emit,
		plugins,
		previousScrollSnap,
		reInit,
		rootNode,
		scrollNext,
		scrollPrev,
		scrollProgress,
		scrollSnapList,
		scrollTo,
		selectedScrollSnap,
		slideNodes,
		slidesInView,
		slidesNotInView
	};
	activate(userOptions, userPlugins);
	setTimeout(() => eventHandler.emit("init"), 0);
	return self;
}
EmblaCarousel.globalOptions = void 0;
function emblaCarouselSvelte(emblaNode, emblaConfig = {
	options: {},
	plugins: []
}) {
	let storedEmblaConfig = emblaConfig;
	let emblaApi;
	if (canUseDOM()) {
		EmblaCarousel.globalOptions = emblaCarouselSvelte.globalOptions;
		emblaApi = EmblaCarousel(emblaNode, storedEmblaConfig.options, storedEmblaConfig.plugins);
		emblaApi.on("init", () => emblaNode.dispatchEvent(new CustomEvent("emblaInit", { detail: emblaApi })));
	}
	return {
		destroy: () => {
			if (emblaApi) emblaApi.destroy();
		},
		update: (newEmblaConfig) => {
			const optionsChanged = !areOptionsEqual(storedEmblaConfig.options, newEmblaConfig.options);
			const pluginsChanged = !arePluginsEqual(storedEmblaConfig.plugins, newEmblaConfig.plugins);
			if (!optionsChanged && !pluginsChanged) return;
			storedEmblaConfig = newEmblaConfig;
			if (emblaApi) emblaApi.reInit(storedEmblaConfig.options, storedEmblaConfig.plugins);
		}
	};
}
emblaCarouselSvelte.globalOptions = void 0;
var styles_module_default$1 = {
	mobile_size: "740px",
	product_card_ctn: "aeX",
	search_top_bar_ctn: "aeY",
	search_top_bar_input: "aeZ",
	button_menu_top: "af0",
	main_carrusel_ctn: "af1",
	product_cards_ctn: "af2",
	producto_caregoria_card: "af3"
};
var root_1$10 = /* @__PURE__ */ from_html(`<div><div class="w-full h-[calc(100%-40px)] relative px-12 pt-16"><img class="w-full h-full object-contain" alt=""/></div> <div class="text-[20px] h-40 w-full flex justify-center items-center mt-auto"> </div></div>`);
var root$13 = /* @__PURE__ */ from_html(`<div class="flex items-center  p-4 md:p-12"><div class="flex w-full justify-center items-center flex-wrap"></div></div> <div><div class="flex w-420"><h2>demo 1</h2></div> <div class="aeR h-full grow-1 aeW"><div class="aeS h-full aeW"><div class="aeQ s1 aeW">Slide 1</div> <div class="aeQ s2 aeW">Slide 2</div> <div class="aeQ aeW">Slide 3</div></div></div></div>`, 1);
function MainCarrusel($$anchor, $$props) {
	push($$props, false);
	let categorias = prop($$props, "categorias", 24, () => []);
	let emblaApi;
	let options = { loop: false };
	function onInit(event) {
		emblaApi = event.detail;
		console.log(emblaApi.slideNodes());
	}
	init();
	var fragment = root$13();
	var div = first_child(fragment);
	var div_1 = child(div);
	each(div_1, 5, categorias, index, ($$anchor, e) => {
		var div_2 = root_1$10();
		var div_3 = child(div_2);
		var img = child(div_3);
		reset(div_3);
		var div_4 = sibling(div_3, 2);
		var text = child(div_4, true);
		reset(div_4);
		reset(div_2);
		template_effect(() => {
			set_class(div_2, 1, (deep_read_state(styles_module_default$1), untrack$1(() => "flex flex-col w-[calc(50vw-16px)] h-160 md:h-240 md:w-240 m-6 md:m-12 " + styles_module_default$1.producto_caregoria_card)), "aeW");
			set_attribute(img, "src", (get$1(e), untrack$1(() => get$1(e).Image)));
			set_text(text, (get$1(e), untrack$1(() => get$1(e).Name)));
		});
		append($$anchor, div_2);
	});
	reset(div_1);
	reset(div);
	var div_5 = sibling(div, 2);
	var div_6 = sibling(child(div_5), 2);
	action(div_6, ($$node, $$action_arg) => emblaCarouselSvelte?.($$node, $$action_arg), () => ({ options }));
	reset(div_5);
	template_effect(() => set_class(div_5, 1, (deep_read_state(styles_module_default$1), untrack$1(() => "relative flex w-full h-700 " + styles_module_default$1.main_carrusel_ctn)), "aeW"));
	event("emblaInit", div_6, onInit);
	append($$anchor, fragment);
	pop();
}
var root$12 = /* @__PURE__ */ from_html(`<div class="_1 relative lg:w-340 w-[34vw] afd"><textarea class="_2 w-full pl-12 rounded-[16px] pt-10 pl-14 afd" cols="1" placeholder="Buscar..."></textarea> <i class="icon1-search absolute top-8 md:top-9 right-7 md:right-10 afd"></i></div>`);
function SearchBar($$anchor) {
	const handleKeydown = (event) => {
		if (event.target && (event.key === "Enter" || event.keyCode === 13 || event.charCode === 13)) event.preventDefault();
	};
	var div = root$12();
	var textarea = child(div);
	textarea.__keydown = handleKeydown;
	next(2);
	reset(div);
	append($$anchor, div);
}
delegate(["keydown"]);
let layerOpenedState = proxy({ id: 0 });
const ProductsSelectedMap = proxy(new SvelteMap());
const addProductoCant = (producto, fixedCant, incrementOrDecrement) => {
	if (typeof fixedCant === "number") if (fixedCant === 0) ProductsSelectedMap.delete(producto.ID);
	else ProductsSelectedMap.set(producto.ID, {
		producto,
		cant: fixedCant
	});
	else if (typeof incrementOrDecrement === "number") {
		const cant = ProductsSelectedMap.get(producto.ID)?.cant || 0;
		ProductsSelectedMap.set(producto.ID, {
			producto,
			cant: cant + incrementOrDecrement
		});
	}
};
var angle_default$1 = "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n<!-- Uploaded to: SVG Repo, www.svgrepo.com, Generator: SVG Repo Mixer Tools -->\n\n<svg\n   width=\"600\"\n   height=\"374.46808\"\n   viewBox=\"0 0 18 11.234042\"\n   version=\"1.1\"\n   id=\"svg1\"\n   sodipodi:docname=\"angle.svg\"\n   inkscape:version=\"1.4 (e7c3feb100, 2024-10-09)\"\n   xmlns:inkscape=\"http://www.inkscape.org/namespaces/inkscape\"\n   xmlns:sodipodi=\"http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd\"\n   xmlns=\"http://www.w3.org/2000/svg\"\n   xmlns:svg=\"http://www.w3.org/2000/svg\"\n   xmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\"\n   xmlns:cc=\"http://creativecommons.org/ns#\"\n   xmlns:dc=\"http://purl.org/dc/elements/1.1/\">\n  <defs\n     id=\"defs1\" />\n  <sodipodi:namedview\n     id=\"namedview1\"\n     pagecolor=\"#ffffff\"\n     bordercolor=\"#000000\"\n     borderopacity=\"0.25\"\n     inkscape:showpageshadow=\"2\"\n     inkscape:pageopacity=\"0.0\"\n     inkscape:pagecheckerboard=\"0\"\n     inkscape:deskcolor=\"#d1d1d1\"\n     inkscape:zoom=\"0.499375\"\n     inkscape:cx=\"-53.066333\"\n     inkscape:cy=\"163.20401\"\n     inkscape:window-width=\"1920\"\n     inkscape:window-height=\"1008\"\n     inkscape:window-x=\"0\"\n     inkscape:window-y=\"0\"\n     inkscape:window-maximized=\"1\"\n     inkscape:current-layer=\"svg1\" />\n  <title\n     id=\"title1\">triangle_fill</title>\n  <rect\n     style=\"fill:none;fill-opacity:1;fill-rule:evenodd;stroke:#000000;stroke-width:0;stroke-dasharray:none;stroke-opacity:0.403546\"\n     id=\"rect1\"\n     width=\"18\"\n     height=\"11.234042\"\n     x=\"0\"\n     y=\"0\" />\n  <g\n     id=\"Shape\"\n     transform=\"translate(-287.99968,-54.679037)\"\n     style=\"fill:none;fill-rule:evenodd;stroke:none;stroke-width:1\">\n    <g\n       id=\"triangle_fill\"\n       transform=\"translate(288,48)\">\n      <path\n         d=\"M 24,0 V 24 H 0 V 0 Z m -11.40651,23.257841 -0.01155,0.0017 -0.07106,0.03553 -0.019,0.0037 v 0 l -0.01516,-0.0037 -0.07106,-0.03553 c -0.0098,-0.0031 -0.01861,-4.89e-4 -0.02351,0.0054 l -0.0041,0.01092 -0.01709,0.427279 0.005,0.02039 0.01101,0.01222 0.103573,0.07398 0.01487,0.0039 v 0 l 0.01177,-0.0039 0.103575,-0.07398 0.0126,-0.01604 v 0 l 0.0034,-0.01656 -0.01709,-0.427279 c -0.002,-0.01013 -0.0085,-0.01653 -0.01607,-0.01799 z m 0.264901,-0.112555 -0.01384,0.002 -0.184705,0.09235 -0.01,0.01024 v 0 l -0.0027,0.01121 0.0179,0.429528 0.0048,0.01278 v 0 l 0.0085,0.0071 0.200954,0.09275 c 0.01209,0.0037 0.02289,-2.52e-4 0.02849,-0.008 l 0.004,-0.01406 -0.03415,-0.614631 c -0.0024,-0.01194 -0.01027,-0.01953 -0.01929,-0.02125 z m -0.715344,0.002 c -0.0098,-0.0049 -0.02087,-0.002 -0.02741,0.0053 l -0.0057,0.01394 -0.03415,0.614631 c -6.39e-4,0.0115 0.007,0.0207 0.01688,0.0234 l 0.01561,-0.0013 0.200955,-0.09275 0.0094,-0.0081 v 0 l 0.0039,-0.0118 0.0179,-0.429528 -0.0032,-0.01259 v 0 l -0.0095,-0.0089 z\"\n         id=\"MingCute\"\n         fill-rule=\"nonzero\" />\n    </g>\n  </g>\n  <path\n     d=\"m 7.9808726,1.6044182 c 0.4529693,-0.6776268 1.5853115,-0.67763372 2.0382024,-6.6e-6 l 6.77335,10.1335664 c 0.452888,0.677626 -0.113204,1.524658 -1.019062,1.524658 H 2.2266722 c -0.9058589,0 -1.47202265,-0.847032 -1.0190932,-1.524658 z\"\n     id=\"路径\"\n     fill=\"#09244b\"\n     style=\"fill:#ffffff;fill-rule:evenodd;stroke:#000000;stroke-width:2.13410997;stroke-linecap:square;stroke-linejoin:miter;stroke-dasharray:none;stroke-opacity:0.34;paint-order:stroke fill markers\" />\n  <metadata\n     id=\"metadata2\">\n    <rdf:RDF>\n      <cc:Work\n         rdf:about=\"\">\n        <dc:title>triangle_fill</dc:title>\n      </cc:Work>\n    </rdf:RDF>\n  </metadata>\n</svg>\n";
var flecha_fin_default = "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n<svg\n   width=\"7.4083mm\"\n   height=\"26.458mm\"\n   version=\"1.1\"\n   viewBox=\"0 0 7.4083 26.458\"\n   id=\"svg1\"\n   sodipodi:docname=\"flecha_fin.svg\"\n   inkscape:version=\"1.4 (e7c3feb100, 2024-10-09)\"\n   xmlns:inkscape=\"http://www.inkscape.org/namespaces/inkscape\"\n   xmlns:sodipodi=\"http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd\"\n   xmlns=\"http://www.w3.org/2000/svg\"\n   xmlns:svg=\"http://www.w3.org/2000/svg\">\n  <defs\n     id=\"defs1\" />\n  <sodipodi:namedview\n     id=\"namedview1\"\n     pagecolor=\"#ffffff\"\n     bordercolor=\"#000000\"\n     borderopacity=\"0.25\"\n     inkscape:showpageshadow=\"2\"\n     inkscape:pageopacity=\"0.0\"\n     inkscape:pagecheckerboard=\"0\"\n     inkscape:deskcolor=\"#d1d1d1\"\n     inkscape:document-units=\"mm\"\n     inkscape:zoom=\"7.9901006\"\n     inkscape:cx=\"13.954768\"\n     inkscape:cy=\"49.99937\"\n     inkscape:window-width=\"1920\"\n     inkscape:window-height=\"1008\"\n     inkscape:window-x=\"0\"\n     inkscape:window-y=\"0\"\n     inkscape:window-maximized=\"1\"\n     inkscape:current-layer=\"svg1\" />\n  <g\n     transform=\"translate(-35.799)\"\n     id=\"g1\"\n     style=\"fill:#ff0000\">\n    <path\n       transform=\"matrix(0 .45865 -.14829 0 48.893 -9.4455)\"\n       d=\"m78.282 88.301h-57.688l28.844-49.959z\"\n       stroke-width=\"0\"\n       style=\"paint-order:stroke fill markers;fill:#000000\"\n       id=\"path1\" />\n  </g>\n</svg>\n";
var flecha_inicio_default = "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n<svg\n   width=\"7.9375mm\"\n   height=\"26.458mm\"\n   version=\"1.1\"\n   viewBox=\"0 0 7.9375 26.458\"\n   id=\"svg1\"\n   sodipodi:docname=\"flecha_inicio.svg\"\n   inkscape:version=\"1.4 (e7c3feb100, 2024-10-09)\"\n   xmlns:inkscape=\"http://www.inkscape.org/namespaces/inkscape\"\n   xmlns:sodipodi=\"http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd\"\n   xmlns=\"http://www.w3.org/2000/svg\"\n   xmlns:svg=\"http://www.w3.org/2000/svg\">\n  <defs\n     id=\"defs1\" />\n  <sodipodi:namedview\n     id=\"namedview1\"\n     pagecolor=\"#ffffff\"\n     bordercolor=\"#000000\"\n     borderopacity=\"0.25\"\n     inkscape:showpageshadow=\"2\"\n     inkscape:pageopacity=\"0.0\"\n     inkscape:pagecheckerboard=\"0\"\n     inkscape:deskcolor=\"#d1d1d1\"\n     inkscape:document-units=\"mm\"\n     inkscape:zoom=\"5.6498543\"\n     inkscape:cx=\"9.8232621\"\n     inkscape:cy=\"38.673564\"\n     inkscape:window-width=\"1920\"\n     inkscape:window-height=\"1008\"\n     inkscape:window-x=\"0\"\n     inkscape:window-y=\"0\"\n     inkscape:window-maximized=\"1\"\n     inkscape:current-layer=\"svg1\" />\n  <g\n     transform=\"translate(-148.7 -128.06)\"\n     id=\"g1\"\n     style=\"fill:#ff0000\">\n    <path\n       d=\"m148.7 128.06 7.4083 13.229-7.4083 13.229h7.9375v-26.458z\"\n       style=\"paint-order:stroke fill markers;fill:#000000\"\n       id=\"path1\" />\n  </g>\n</svg>\n";
var styles_module_default = {
	image_hash_ctn: "afL",
	card_arrow_ctn: "afM",
	card_arrow_name: "afN",
	card_arrow_line: "afO",
	card_arrow_svg: "afP",
	card_arrow_ctn_selected: "afQ"
};
var root_3$4 = /* @__PURE__ */ from_html(`<div class="ff-semibold"> </div>`);
var root_1$9 = /* @__PURE__ */ from_html(`<div><img alt=""/> <div><!></div> <img alt=""/> <div></div></div>`);
var root$11 = /* @__PURE__ */ from_html(`<div class="grid mr-8"></div>`);
function ArrowSteps($$anchor, $$props) {
	push($$props, true);
	let onSelect = prop($$props, "onSelect", 3, () => {});
	function handleSelect(option) {
		onSelect()(option);
	}
	user_effect(() => {
		console.log("Ecommerce changed::", Core.ecommerce.cartOption, "|", $$props.selected);
	});
	const gridTemplateColumns = /* @__PURE__ */ user_derived(() => $$props.columnsTemplate || $$props.options.map(() => "1fr").join(" "));
	var div = root$11();
	let styles;
	each(div, 21, () => $$props.options, (option) => option.id, ($$anchor, option) => {
		var div_1 = root_1$9();
		div_1.__click = () => handleSelect(get$1(option));
		var img = child(div_1);
		var div_2 = sibling(img, 2);
		var node = child(div_2);
		var consequent = ($$anchor) => {
			var fragment = comment();
			snippet(first_child(fragment), () => $$props.optionRender, () => get$1(option));
			append($$anchor, fragment);
		};
		var alternate = ($$anchor) => {
			var div_3 = root_3$4();
			var text = child(div_3, true);
			reset(div_3);
			template_effect(() => set_text(text, get$1(option).name));
			append($$anchor, div_3);
		};
		if_block(node, ($$render) => {
			if ($$props.optionRender) $$render(consequent);
			else $$render(alternate, false);
		});
		reset(div_2);
		var img_1 = sibling(div_2, 2);
		var div_4 = sibling(img_1, 2);
		reset(div_1);
		template_effect(($0, $1, $2) => {
			set_class(div_1, 1, $0);
			set_class(img, 1, `h-full ${styles_module_default.card_arrow_svg ?? ""}`);
			set_attribute(img, "src", $1);
			set_class(div_2, 1, `h-full flex items-center justify-center ${styles_module_default.card_arrow_name ?? ""}`);
			set_class(img_1, 1, `h-full ${styles_module_default.card_arrow_svg ?? ""}`);
			set_attribute(img_1, "src", $2);
			set_class(div_4, 1, clsx(styles_module_default.card_arrow_line));
		}, [
			() => clsx(cn("flex relative items-center", styles_module_default.card_arrow_ctn, get$1(option).id === $$props.selected && styles_module_default.card_arrow_ctn_selected)),
			() => parseSVG(flecha_inicio_default),
			() => parseSVG(flecha_fin_default)
		]);
		append($$anchor, div_1);
	});
	reset(div);
	template_effect(() => styles = set_style(div, "", styles, { "grid-template-columns": get$1(gridTemplateColumns) }));
	append($$anchor, div);
	pop();
}
delegate(["click"]);
const checkDevice = () => {
	const Window = getWindow();
	if (Window.innerWidth <= 680) return 3;
	else if (Window.innerWidth <= 940) return 2;
	else return 1;
};
proxy({ deviceType: checkDevice() });
let Ecommerce = proxy({ cartOption: 1 });
const DEFAULT_SCROLL_OPTIONS = {
	smoothScroll: true,
	shouldThrowOnBounds: true,
	align: "auto"
};
const calculateScrollPosition = (totalItems, itemHeight, containerHeight) => {
	if (totalItems === 0) return 0;
	const totalHeight = totalItems * itemHeight;
	return Math.max(0, totalHeight - containerHeight);
};
const calculateVisibleRange = (scrollTop, viewportHeight, itemHeight, totalItems, bufferSize, mode, atBottom, wasAtBottomBeforeHeightChange, lastVisibleRange, totalContentHeight, heightCache) => {
	if (mode === "bottomToTop") {
		const visibleCount = Math.ceil(viewportHeight / itemHeight) + 1;
		const totalHeight = totalContentHeight ?? totalItems * itemHeight;
		const distanceFromStart = Math.max(0, totalHeight - viewportHeight) - scrollTop;
		const startIndex = Math.floor(distanceFromStart / itemHeight);
		if (startIndex < 0) return {
			start: 0,
			end: Math.min(totalItems, visibleCount + bufferSize * 2)
		};
		return {
			start: Math.max(0, startIndex - bufferSize),
			end: Math.min(totalItems, startIndex + visibleCount + bufferSize)
		};
	} else {
		const start = Math.floor(scrollTop / itemHeight);
		const end = Math.min(totalItems, start + Math.ceil(viewportHeight / itemHeight) + 1);
		const totalHeight = totalContentHeight ?? totalItems * itemHeight;
		const maxScrollTop = Math.max(0, totalHeight - viewportHeight);
		const tolerance = Math.max(1, Math.floor(itemHeight * .25));
		if (Math.abs(scrollTop - maxScrollTop) <= tolerance) {
			const adjustedEnd = totalItems;
			let startCore = adjustedEnd;
			let acc = 0;
			const getH = (i) => {
				const v = heightCache ? heightCache[i] : void 0;
				return Number.isFinite(v) && v > 0 ? v : itemHeight;
			};
			while (startCore > 0 && acc < viewportHeight) {
				const h = getH(startCore - 1);
				acc += h;
				startCore -= 1;
			}
			return {
				start: Math.max(0, startCore - bufferSize),
				end: adjustedEnd
			};
		}
		return {
			start: Math.max(0, start - bufferSize),
			end: Math.min(totalItems, end + bufferSize)
		};
	}
};
const calculateTransformY = (mode, totalItems, visibleEnd, visibleStart, itemHeight, viewportHeight, totalContentHeight, heightCache, measuredFallbackHeight) => {
	const effectiveViewport = viewportHeight || measuredFallbackHeight || 0;
	if (mode === "bottomToTop") {
		const actualTotalHeight = totalContentHeight ?? totalItems * itemHeight;
		return (totalItems - visibleEnd) * itemHeight + Math.max(0, effectiveViewport - actualTotalHeight);
	} else {
		if (heightCache) {
			const offset = getScrollOffsetForIndex(heightCache, itemHeight, visibleStart);
			return Math.max(0, Math.round(offset));
		}
		return visibleStart * itemHeight;
	}
};
const updateHeightAndScroll = (state, setters, immediate = false) => {
	const { initialized, mode, containerElement, viewportElement, calculatedItemHeight, scrollTop } = state;
	const { setHeight, setScrollTop } = setters;
	if (immediate) {
		if (containerElement && viewportElement && initialized) {
			const newHeight = containerElement.getBoundingClientRect().height;
			setHeight(newHeight);
			if (mode === "bottomToTop") {
				const newScrollTop = Math.floor(scrollTop / calculatedItemHeight) * calculatedItemHeight;
				viewportElement.scrollTop = newScrollTop;
				setScrollTop(newScrollTop);
			}
		}
	}
};
const calculateAverageHeight = (itemElements, visibleRange, heightCache, currentItemHeight, dirtyItems, currentTotalHeight = 0, currentValidCount = 0, mode = "topToBottom") => {
	const validElements = itemElements.filter((el) => el);
	if (validElements.length === 0) return {
		newHeight: currentItemHeight,
		newLastMeasuredIndex: visibleRange.start,
		updatedHeightCache: heightCache,
		clearedDirtyItems: /* @__PURE__ */ new Set(),
		newTotalHeight: currentTotalHeight,
		newValidCount: currentValidCount,
		heightChanges: []
	};
	const newHeightCache = { ...heightCache };
	const clearedDirtyItems = /* @__PURE__ */ new Set();
	const heightChanges = [];
	let totalValidHeight = currentTotalHeight;
	let validHeightCount = currentValidCount;
	if (dirtyItems.size > 0) dirtyItems.forEach((itemIndex) => {
		let elementIndex;
		if (mode === "bottomToTop") elementIndex = validElements.length - 1 - (itemIndex - visibleRange.start);
		else elementIndex = itemIndex - visibleRange.start;
		const element = validElements[elementIndex];
		if (element && elementIndex >= 0 && elementIndex < validElements.length) try {
			element.offsetHeight;
			const height = element.getBoundingClientRect().height;
			const oldHeight = newHeightCache[itemIndex];
			if (Number.isFinite(height) && height > 0) {
				if (!oldHeight || Math.abs(oldHeight - height) >= .1) {
					const actualOldHeight = oldHeight || currentItemHeight;
					const delta = height - actualOldHeight;
					heightChanges.push({
						index: itemIndex,
						oldHeight: actualOldHeight,
						newHeight: height,
						delta
					});
					if (oldHeight && Number.isFinite(oldHeight) && oldHeight > 0) totalValidHeight = totalValidHeight - oldHeight + height;
					else {
						totalValidHeight += height;
						validHeightCount++;
					}
					newHeightCache[itemIndex] = height;
				}
			}
			clearedDirtyItems.add(itemIndex);
		} catch {
			clearedDirtyItems.add(itemIndex);
		}
		else clearedDirtyItems.add(itemIndex);
	});
	else validElements.forEach((el, i) => {
		const itemIndex = mode === "bottomToTop" ? Math.max(0, (visibleRange.end ?? visibleRange.start + validElements.length) - 1 - i) : visibleRange.start + i;
		if (!newHeightCache[itemIndex]) try {
			const height = el.getBoundingClientRect().height;
			if (Number.isFinite(height) && height > 0) {
				totalValidHeight += height;
				validHeightCount++;
				newHeightCache[itemIndex] = height;
			}
		} catch {}
	});
	return {
		newHeight: validHeightCount > 0 ? totalValidHeight / validHeightCount : currentItemHeight,
		newLastMeasuredIndex: visibleRange.start,
		updatedHeightCache: newHeightCache,
		clearedDirtyItems,
		newTotalHeight: totalValidHeight,
		newValidCount: validHeightCount,
		heightChanges
	};
};
const getScrollOffsetForIndex = (heightCache, calculatedItemHeight, idx, blockSums, blockSize = 1e3) => {
	const safeIdx = Math.max(0, Math.floor(idx));
	if (safeIdx <= 0) return 0;
	if (!blockSums) {
		let offset = 0;
		for (let i = 0; i < safeIdx; i++) {
			const raw = heightCache[i];
			offset += Number.isFinite(raw) && raw > 0 ? raw : calculatedItemHeight;
		}
		return offset;
	}
	const blockIdx = Math.floor(safeIdx / blockSize);
	let offsetBase = 0;
	if (blockIdx > 0) {
		const base = blockSums[blockIdx - 1];
		offsetBase = Number.isFinite(base) ? base : 0;
	}
	let offset = offsetBase;
	const start = blockIdx * blockSize;
	for (let i = start; i < safeIdx; i++) {
		const raw = heightCache[i];
		offset += Number.isFinite(raw) && raw > 0 ? raw : calculatedItemHeight;
	}
	return offset;
};
const calculateAverageHeightDebounced = (isCalculatingHeight, heightUpdateTimeout, visibleItemsGetter, itemElements, heightCache, lastMeasuredIndex, calculatedItemHeight, onUpdate, debounceTime, dirtyItems, currentTotalHeight = 0, currentValidCount = 0, mode = "topToBottom") => {
	if (isCalculatingHeight) return null;
	const visibleRange = visibleItemsGetter();
	if (visibleRange.start === lastMeasuredIndex && dirtyItems.size === 0) return null;
	if (heightUpdateTimeout) clearTimeout(heightUpdateTimeout);
	return setTimeout(() => {
		const { newHeight, newLastMeasuredIndex, updatedHeightCache, clearedDirtyItems, newTotalHeight, newValidCount, heightChanges } = calculateAverageHeight(itemElements, visibleRange, heightCache, calculatedItemHeight, dirtyItems, currentTotalHeight, currentValidCount, mode);
		if (Math.abs(newHeight - calculatedItemHeight) > 1 || dirtyItems.size > 0) onUpdate({
			newHeight,
			newLastMeasuredIndex,
			updatedHeightCache,
			clearedDirtyItems,
			newTotalHeight,
			newValidCount,
			heightChanges
		});
	}, debounceTime);
};
const createRafScheduler = () => {
	let scheduled = false;
	let callback = null;
	return (_fn) => {
		callback = _fn;
		if (!scheduled) {
			scheduled = true;
			requestAnimationFrame(() => {
				scheduled = false;
				if (callback) {
					callback();
					callback = null;
				}
			});
		}
	};
};
const isSignificantHeightChange = (itemIndex, newHeight, heightCache, marginOfError = 1) => {
	const previousHeight = heightCache[itemIndex];
	if (previousHeight === void 0) return true;
	return Math.abs(newHeight - previousHeight) > marginOfError;
};
const shouldShowDebugInfo = (prevRange, currentRange, prevHeight, currentHeight) => {
	if (!prevRange) return true;
	return prevRange.start !== currentRange.start || prevRange.end !== currentRange.end || prevHeight !== currentHeight;
};
const createDebugInfo = (visibleRange, totalItems, processedItems, averageItemHeight, scrollTop, viewportHeight, totalHeight) => {
	const atTop = scrollTop <= 1;
	const atBottom = scrollTop >= totalHeight - viewportHeight - 1;
	return {
		visibleItemsCount: visibleRange.end - visibleRange.start,
		startIndex: visibleRange.start,
		endIndex: visibleRange.end,
		totalItems,
		processedItems,
		averageItemHeight,
		atTop,
		atBottom,
		totalHeight
	};
};
const calculateScrollTarget = (params) => {
	const { mode, align, targetIndex, itemsLength, calculatedItemHeight, height, scrollTop, firstVisibleIndex, lastVisibleIndex, heightCache } = params;
	if (mode === "bottomToTop") return calculateBottomToTopScrollTarget({
		align,
		targetIndex,
		itemsLength,
		calculatedItemHeight,
		height,
		scrollTop,
		firstVisibleIndex,
		lastVisibleIndex,
		heightCache
	});
	else return calculateTopToBottomScrollTarget({
		align,
		targetIndex,
		calculatedItemHeight,
		height,
		scrollTop,
		firstVisibleIndex,
		lastVisibleIndex,
		heightCache
	});
};
var calculateBottomToTopScrollTarget = (params) => {
	const { align, targetIndex, itemsLength, calculatedItemHeight, height, scrollTop, firstVisibleIndex, lastVisibleIndex, heightCache } = params;
	const totalHeight = getScrollOffsetForIndex(heightCache, calculatedItemHeight, itemsLength);
	const itemOffset = getScrollOffsetForIndex(heightCache, calculatedItemHeight, targetIndex);
	const itemHeight = calculatedItemHeight;
	if (align === "auto") if (targetIndex < firstVisibleIndex) return Math.max(0, totalHeight - (itemOffset + itemHeight));
	else if (targetIndex > lastVisibleIndex - 1) return Math.max(0, totalHeight - itemOffset - height);
	else {
		const itemTop = totalHeight - (itemOffset + itemHeight);
		const itemBottom = totalHeight - itemOffset;
		return Math.abs(scrollTop - itemTop) < Math.abs(scrollTop + height - itemBottom) ? itemTop : Math.max(0, itemBottom - height);
	}
	else if (align === "top") return Math.max(0, totalHeight - (itemOffset + itemHeight));
	else if (align === "bottom") return Math.max(0, totalHeight - itemOffset - height);
	else if (align === "nearest") {
		const itemTop = totalHeight - (itemOffset + itemHeight);
		const itemBottom = totalHeight - itemOffset;
		if (itemBottom <= scrollTop || itemTop >= scrollTop + height) return Math.abs(scrollTop - itemTop) < Math.abs(scrollTop + height - itemBottom) ? itemTop : Math.max(0, itemBottom - height);
		else return null;
	}
	return null;
};
var calculateTopToBottomScrollTarget = (params) => {
	const { align, targetIndex, calculatedItemHeight, height, scrollTop, firstVisibleIndex, lastVisibleIndex, heightCache } = params;
	if (align === "auto") if (targetIndex < firstVisibleIndex) return getScrollOffsetForIndex(heightCache, calculatedItemHeight, targetIndex);
	else if (targetIndex > lastVisibleIndex - 1) {
		const itemBottom = getScrollOffsetForIndex(heightCache, calculatedItemHeight, targetIndex + 1);
		return Math.max(0, itemBottom - height);
	} else {
		const itemTop = getScrollOffsetForIndex(heightCache, calculatedItemHeight, targetIndex);
		const itemBottom = getScrollOffsetForIndex(heightCache, calculatedItemHeight, targetIndex + 1);
		if (Math.abs(scrollTop - itemTop) < Math.abs(scrollTop + height - itemBottom)) return itemTop;
		else return Math.max(0, itemBottom - height);
	}
	else if (align === "top") return getScrollOffsetForIndex(heightCache, calculatedItemHeight, targetIndex);
	else if (align === "bottom") {
		const itemBottom = getScrollOffsetForIndex(heightCache, calculatedItemHeight, targetIndex + 1);
		return Math.max(0, itemBottom - height);
	} else if (align === "nearest") {
		const itemTop = getScrollOffsetForIndex(heightCache, calculatedItemHeight, targetIndex);
		const itemBottom = getScrollOffsetForIndex(heightCache, calculatedItemHeight, targetIndex + 1);
		if (itemBottom <= scrollTop || itemTop >= scrollTop + height) if (Math.abs(scrollTop - itemTop) < Math.abs(scrollTop + height - itemBottom)) return itemTop;
		else return Math.max(0, itemBottom - height);
		else return null;
	}
	return null;
};
const timeProvider = { now: () => {
	if (typeof performance !== "undefined" && performance.now) return performance.now();
	return Date.now();
} };
const createAdvancedThrottledCallback = (callback, delay, options = {}) => {
	const { leading = true, trailing = false } = options;
	let lastExecutionTime = 0;
	let trailingTimeoutId = null;
	let lastArgs = null;
	let isFirstCall = true;
	const execute = (args) => {
		lastExecutionTime = timeProvider.now();
		callback(...args);
	};
	return ((...args) => {
		const now = timeProvider.now();
		const timeSinceLastExecution = isFirstCall ? delay : now - lastExecutionTime;
		lastArgs = args;
		if (trailingTimeoutId) {
			clearTimeout(trailingTimeoutId);
			trailingTimeoutId = null;
		}
		if (timeSinceLastExecution >= delay) {
			if (leading) {
				isFirstCall = false;
				execute(args);
			}
			if (trailing && !leading) trailingTimeoutId = setTimeout(() => {
				if (lastArgs) execute(lastArgs);
				trailingTimeoutId = null;
			}, delay);
		} else if (isFirstCall && leading) {
			isFirstCall = false;
			execute(args);
		} else if (trailing) {
			const remainingTime = delay - timeSinceLastExecution;
			trailingTimeoutId = setTimeout(() => {
				if (lastArgs) execute(lastArgs);
				trailingTimeoutId = null;
			}, remainingTime);
		}
	});
};
var root_1$8 = /* @__PURE__ */ from_html(`<!> <div><!></div>`, 1);
var root$10 = /* @__PURE__ */ from_html(`<div><div><div><div></div></div></div></div>`);
function SvelteVirtualList($$anchor, $$props) {
	push($$props, true);
	const rafSchedule = createRafScheduler();
	const GLOBAL_CORRECTION_COOLDOWN = 16;
	const lastCorrectionTimestampByViewport = /* @__PURE__ */ new WeakMap();
	const INTERNAL_DEBUG = Boolean(typeof process !== "undefined" && ({}?.PUBLIC_SVELTE_VIRTUAL_LIST_DEBUG === "true" || {}?.SVELTE_VIRTUAL_LIST_DEBUG === "true"));
	const items = prop($$props, "items", 19, () => []), defaultEstimatedItemHeight = prop($$props, "defaultEstimatedItemHeight", 3, 40), debug = prop($$props, "debug", 3, false), mode = prop($$props, "mode", 3, "topToBottom"), bufferSize = prop($$props, "bufferSize", 3, 20);
	const itemElements = proxy([]);
	let height = /* @__PURE__ */ state(0);
	const isCalculatingHeight = false;
	let isScrolling = /* @__PURE__ */ state(false);
	let lastMeasuredIndex = /* @__PURE__ */ state(-1);
	let lastScrollTopSnapshot = /* @__PURE__ */ state(0);
	let heightUpdateTimeout = null;
	let resizeObserver = null;
	let itemResizeObserver = null;
	const dirtyItems = proxy(/* @__PURE__ */ new Set());
	let dirtyItemsCount = /* @__PURE__ */ state(0);
	let measuredFallbackHeight = /* @__PURE__ */ state(0);
	let prevVisibleRange = /* @__PURE__ */ state(null);
	let prevHeight = /* @__PURE__ */ state(0);
	let prevTotalHeightForScrollCorrection = /* @__PURE__ */ state(0);
	let lastBottomDistance = /* @__PURE__ */ state(null);
	const heightManager = new ReactiveListManager({
		itemLength: items().length,
		itemHeight: defaultEstimatedItemHeight(),
		internalDebug: INTERNAL_DEBUG
	});
	const instanceId = Math.random().toString(36).slice(2, 7);
	const log = (tag, payload) => {
		if (!debug() && !INTERNAL_DEBUG) return;
		try {
			const ts = (/* @__PURE__ */ new Date()).toISOString().split("T")[1]?.replace("Z", "");
			console.info(`[SVL][${instanceId}] ${ts} ${tag}`, payload ?? "");
		} catch {}
	};
	let suppressBottomAnchoringUntilMs = /* @__PURE__ */ state(0);
	const displayItems = /* @__PURE__ */ user_derived(() => () => {
		const visibleRange = get$1(visibleItems)();
		return (mode() === "bottomToTop" ? items().slice(visibleRange.start, visibleRange.end).reverse() : items().slice(visibleRange.start, visibleRange.end)).map((item, sliceIndex) => ({
			item,
			originalIndex: mode() === "bottomToTop" ? visibleRange.end - 1 - sliceIndex : visibleRange.start + sliceIndex,
			sliceIndex
		}));
	});
	const handleHeightChangesScrollCorrection = (heightChanges) => {
		if (!heightManager.viewportElement || !heightManager.initialized || get$1(userHasScrolledAway)) return;
		if (mode() === "bottomToTop" && wasAtBottomBeforeHeightChange && !get$1(programmaticScrollInProgress) && performance.now() >= get$1(suppressBottomAnchoringUntilMs)) {
			const now = performance.now();
			const viewportEl = heightManager.viewport;
			if (now - (lastCorrectionTimestampByViewport.get(viewportEl) ?? 0) < GLOBAL_CORRECTION_COOLDOWN) {
				set$1(suppressBottomAnchoringUntilMs, now + 50);
				return;
			}
			lastCorrectionTimestampByViewport.set(viewportEl, now);
			const approximateScrollTop = Math.max(0, get$1(totalHeight)() - get$1(height));
			log("b2t-correction-approx", { approximateScrollTop });
			heightManager.viewport.scrollTop = approximateScrollTop;
			heightManager.scrollTop = approximateScrollTop;
			tick$1().then(() => {
				const item0Element = heightManager.viewport.querySelector("[data-original-index=\"0\"]");
				if (item0Element) {
					const contRect = heightManager.viewport.getBoundingClientRect();
					const itemRect = item0Element.getBoundingClientRect();
					if (!(Math.abs(contRect.y + contRect.height - (itemRect.y + itemRect.height)) <= 4)) {
						item0Element.scrollIntoView({
							block: "end",
							behavior: "smooth",
							inline: "nearest"
						});
						log("b2t-correction-native", {
							containerBottom: contRect.y + contRect.height,
							itemBottom: itemRect.y + itemRect.height
						});
					}
					heightManager.scrollTop = heightManager.viewport.scrollTop;
					set$1(suppressBottomAnchoringUntilMs, performance.now() + 200);
				}
			});
			return;
		}
		const currentScrollTop = heightManager.viewport.scrollTop;
		const maxScrollTop = Math.max(0, get$1(totalHeight)() - get$1(height));
		let heightChangeAboveViewport = 0;
		const currentVisibleRange = get$1(visibleItems)();
		for (const change of heightChanges) if (change.index < currentVisibleRange.start) heightChangeAboveViewport += change.delta;
		if (Math.abs(heightChangeAboveViewport) > 1) {
			const newScrollTop = Math.min(maxScrollTop, Math.max(0, currentScrollTop + heightChangeAboveViewport));
			heightManager.viewport.scrollTop = newScrollTop;
			heightManager.scrollTop = newScrollTop;
		}
	};
	const triggerHeightUpdate = createAdvancedThrottledCallback(() => {
		if (get$1(dirtyItemsCount) > 0) {
			wasAtBottomBeforeHeightChange = get$1(atBottom);
			heightManager.startDynamicUpdate();
			updateHeight();
		}
	}, 16, {
		leading: true,
		trailing: true
	});
	user_effect(() => {
		triggerHeightUpdate();
	});
	user_effect(() => {
		heightManager.updateItemLength(items().length);
	});
	const updateHeight = () => {
		set$1(prevTotalHeightForScrollCorrection, heightManager.totalHeight, true);
		heightUpdateTimeout = calculateAverageHeightDebounced(isCalculatingHeight, heightUpdateTimeout, get$1(visibleItems), itemElements, heightManager.getHeightCache(), get$1(lastMeasuredIndex), heightManager.averageHeight, (result) => {
			heightManager.itemHeight = result.newHeight;
			set$1(lastMeasuredIndex, result.newLastMeasuredIndex, true);
			if (result.heightChanges.length > 0) heightManager.processDirtyHeights(result.heightChanges);
			if (result.heightChanges.length > 0 && mode() === "bottomToTop") {
				const changes = result.heightChanges;
				queueMicrotask(() => handleHeightChangesScrollCorrection(changes));
			}
			if (mode() === "topToBottom" && heightManager.isReady && heightManager.initialized) {
				const oldTotal = get$1(prevTotalHeightForScrollCorrection);
				const newTotal = heightManager.totalHeight;
				const deltaTotal = newTotal - oldTotal;
				if (Math.abs(deltaTotal) > 1) {
					const maxScrollTop = Math.max(0, newTotal - (get$1(height) || 0));
					const tolerance = Math.max(heightManager.averageHeight, 10);
					const currentScrollTop = heightManager.viewport.scrollTop;
					if (Math.abs(currentScrollTop - maxScrollTop) <= tolerance) {
						const adjusted = Math.min(maxScrollTop, Math.max(0, currentScrollTop + deltaTotal));
						heightManager.viewport.scrollTop = adjusted;
						heightManager.scrollTop = adjusted;
					}
				}
			}
			untrack$1(() => {
				dirtyItems.clear();
				set$1(dirtyItemsCount, 0);
				wasAtBottomBeforeHeightChange = false;
			});
			heightManager.endDynamicUpdate();
		}, get$1(lastMeasuredIndex) < 0 ? 0 : 100, dirtyItems, 0, 0, mode());
	};
	let userHasScrolledAway = /* @__PURE__ */ state(false);
	let programmaticScrollInProgress = /* @__PURE__ */ state(false);
	let lastCalculatedHeight = /* @__PURE__ */ state(0);
	let lastItemsLength = /* @__PURE__ */ state(0);
	let lastTotalHeightObserved = /* @__PURE__ */ state(0);
	const totalHeight = /* @__PURE__ */ user_derived(() => () => heightManager.totalHeight);
	const atBottom = /* @__PURE__ */ user_derived(() => heightManager.scrollTop >= get$1(totalHeight)() - get$1(height) - 1);
	let wasAtBottomBeforeHeightChange = false;
	let lastVisibleRange = null;
	function updateDebugTailDistance() {
		if (!heightManager.viewportElement) return;
		const last = heightManager.viewport.querySelector(`[data-original-index="${items().length - 1}"]`);
		if (!last) return;
		const v = heightManager.viewport.getBoundingClientRect();
		const r = last.getBoundingClientRect();
		set$1(lastBottomDistance, Math.round(Math.abs(r.bottom - v.bottom)), true);
		if (INTERNAL_DEBUG) console.info("[SVL] bottomDistance(px):", get$1(lastBottomDistance));
	}
	user_effect(() => {
		if (heightManager.initialized && mode() === "bottomToTop" && heightManager.viewportElement) {
			const targetScrollTop = Math.max(0, get$1(totalHeight)() - get$1(height));
			const currentScrollTop = heightManager.viewport.scrollTop;
			const scrollDifference = Math.abs(currentScrollTop - targetScrollTop);
			const heightChanged = Math.abs(heightManager.averageHeight - get$1(lastCalculatedHeight)) > 1;
			const maxScrollTop = Math.max(0, get$1(totalHeight)() - get$1(height));
			const isAtBottom = Math.abs(currentScrollTop - maxScrollTop) < heightManager.averageHeight;
			if (heightChanged && !get$1(userHasScrolledAway) && !isAtBottom && !get$1(programmaticScrollInProgress) && performance.now() >= get$1(suppressBottomAnchoringUntilMs) && !heightManager.isDynamicUpdateInProgress && scrollDifference > heightManager.averageHeight * 3) {
				const roundedTargetScrollTop = Math.round(targetScrollTop);
				heightManager.viewport.scrollTop = roundedTargetScrollTop;
				heightManager.scrollTop = roundedTargetScrollTop;
			}
			if (scrollDifference > heightManager.averageHeight * 5) set$1(userHasScrolledAway, true);
			set$1(lastCalculatedHeight, heightManager.averageHeight, true);
		}
	});
	user_effect(() => {
		const currentItemsLength = items().length;
		if (heightManager.initialized && mode() === "bottomToTop" && heightManager.isReady && get$1(lastItemsLength) > 0) {
			const itemsAdded = currentItemsLength - get$1(lastItemsLength);
			if (itemsAdded !== 0) {
				const currentScrollTop = heightManager.viewport.scrollTop;
				const currentCalculatedItemHeight = heightManager.averageHeight;
				const currentHeight = get$1(height);
				const currentTotalHeight = get$1(totalHeight)();
				const prevTotalHeight = get$1(lastTotalHeightObserved) || currentTotalHeight - itemsAdded * currentCalculatedItemHeight;
				const prevMaxScrollTop = Math.max(0, prevTotalHeight - currentHeight);
				const nextMaxScrollTop = Math.max(0, currentTotalHeight - currentHeight);
				const deltaMax = nextMaxScrollTop - prevMaxScrollTop;
				log("[SVL] items-length-change:before", {
					instanceId,
					itemsAdded,
					lastItemsLength: get$1(lastItemsLength),
					currentItemsLength,
					currentScrollTop,
					prevTotalHeight,
					currentTotalHeight,
					prevMaxScrollTop,
					nextMaxScrollTop,
					deltaMax,
					averageItemHeight: currentCalculatedItemHeight
				});
				set$1(programmaticScrollInProgress, true);
				heightManager.runDynamicUpdate(() => {
					const unclamped = currentScrollTop + deltaMax;
					const newScrollTop = Math.max(0, Math.min(nextMaxScrollTop, unclamped));
					heightManager.viewport.scrollTop = newScrollTop;
					heightManager.scrollTop = newScrollTop;
					log("[SVL] items-length-change:applied", {
						instanceId,
						previousScrollTop: currentScrollTop,
						appliedScrollTop: newScrollTop,
						prevMaxScrollTop,
						nextMaxScrollTop,
						deltaMax
					});
					requestAnimationFrame(() => {
						const beforeReconcileScrollTop = heightManager.viewport.scrollTop;
						const reconciledNextMax = Math.max(0, get$1(totalHeight)() - get$1(height));
						const reconciledDeltaMaxChange = reconciledNextMax - nextMaxScrollTop;
						const desiredScrollTop = Math.max(0, Math.min(reconciledNextMax, newScrollTop + reconciledDeltaMaxChange));
						const desiredRounded = Math.round(desiredScrollTop);
						const diffToDesired = desiredRounded - heightManager.viewport.scrollTop;
						if (Math.abs(diffToDesired) >= 1) {
							const adjusted = Math.max(0, Math.min(reconciledNextMax, desiredRounded));
							heightManager.viewport.scrollTop = adjusted;
							heightManager.scrollTop = adjusted;
							log("[SVL] items-length-change:reconciled", {
								instanceId,
								beforeReconcileScrollTop,
								adjustedScrollTop: adjusted,
								reconciledNextMax,
								reconciledDeltaMaxChange,
								desiredScrollTop,
								desiredRounded,
								diffToDesired
							});
						} else log("[SVL] items-length-change:reconciled-skip", {
							instanceId,
							beforeReconcileScrollTop,
							reconciledNextMax,
							reconciledDeltaMaxChange,
							desiredScrollTop,
							desiredRounded,
							diffToDesired
						});
						set$1(programmaticScrollInProgress, false);
					});
				});
			}
		}
		set$1(lastItemsLength, currentItemsLength, true);
		set$1(lastTotalHeightObserved, get$1(totalHeight)(), true);
	});
	user_effect(() => {
		if (heightManager.isReady) {
			const h = heightManager.container.getBoundingClientRect().height;
			if (Number.isFinite(h) && h > 0) set$1(height, h, true);
		}
	});
	user_effect(() => {
		if (get$1(height) === 0 && heightManager.isReady) {
			const h = heightManager.container.getBoundingClientRect().height;
			if (Number.isFinite(h) && h > 0) set$1(measuredFallbackHeight, h, true);
		}
	});
	const visibleItems = /* @__PURE__ */ user_derived(() => () => {
		if (!items().length) return {
			start: 0,
			end: 0
		};
		const viewportHeight = get$1(height) || 0;
		if (mode() === "bottomToTop" && !heightManager.initialized && heightManager.scrollTop === 0 && viewportHeight > 0) {
			lastVisibleRange = calculateVisibleRange(Math.max(0, get$1(totalHeight)() - viewportHeight), viewportHeight, heightManager.averageHeight, items().length, bufferSize(), mode(), get$1(atBottom), wasAtBottomBeforeHeightChange, lastVisibleRange, get$1(totalHeight)(), heightManager.getHeightCache());
			return lastVisibleRange;
		}
		lastVisibleRange = calculateVisibleRange(heightManager.scrollTop, viewportHeight, heightManager.averageHeight, items().length, bufferSize(), mode(), get$1(atBottom), wasAtBottomBeforeHeightChange, lastVisibleRange, get$1(totalHeight)(), heightManager.getHeightCache());
		return lastVisibleRange;
	});
	const handleScroll = () => {
		if (!heightManager.viewportElement) return;
		if (!get$1(isScrolling)) {
			set$1(isScrolling, true);
			rafSchedule(() => {
				const current = heightManager.viewport.scrollTop;
				if (mode() === "bottomToTop") {
					if (get$1(lastScrollTopSnapshot) - current > .5) {
						set$1(suppressBottomAnchoringUntilMs, performance.now() + 450);
						set$1(userHasScrolledAway, true);
					}
				}
				set$1(lastScrollTopSnapshot, current, true);
				heightManager.scrollTop = current;
				updateDebugTailDistance();
				if (INTERNAL_DEBUG) {
					const vr = get$1(visibleItems)();
					log("scroll", {
						mode: mode(),
						scrollTop: heightManager.scrollTop,
						height: get$1(height),
						totalHeight: get$1(totalHeight)(),
						averageItemHeight: heightManager.averageHeight,
						visibleRange: vr
					});
				}
				set$1(isScrolling, false);
			});
		}
	};
	const updateHeightAndScroll$1 = (immediate = false) => {
		log("updateHeightAndScroll-enter", {
			immediate,
			initialized: heightManager.initialized,
			mode: mode()
		});
		if (!heightManager.initialized && mode() === "bottomToTop") {
			tick$1().then(() => {
				requestAnimationFrame(() => {
					requestAnimationFrame(() => {
						if (!heightManager.isReady) return;
						const measuredHeight = heightManager.container.getBoundingClientRect().height;
						set$1(height, measuredHeight, true);
						const targetScrollTop = calculateScrollPosition(items().length, heightManager.averageHeight, measuredHeight);
						const suffix = String(instanceId).toLowerCase().replace(/[^a-z0-9]/g, "").slice(-4);
						const parsed = parseInt(suffix, 36);
						const jitterMs = Number.isNaN(parsed) ? Math.floor(Math.random() * 3) : parsed % 3;
						log("b2t-init", {
							measuredHeight,
							targetScrollTop,
							jitterMs
						});
						setTimeout(() => {
							heightManager.viewport.scrollTop = targetScrollTop;
							heightManager.scrollTop = targetScrollTop;
							requestAnimationFrame(() => {
								if (!heightManager.initialized) heightManager.initialized = true;
								tick$1().then(() => {
									const el = heightManager.viewport.querySelector("[data-original-index=\"0\"]");
									if (!el) return;
									const cont = heightManager.viewport.getBoundingClientRect();
									const r = el.getBoundingClientRect();
									if (!(Math.abs(cont.y + cont.height - (r.y + r.height)) <= 4)) {
										el.scrollIntoView({
											block: "end",
											inline: "nearest"
										});
										heightManager.scrollTop = heightManager.viewport.scrollTop;
										log("b2t-init-native-fallback", {
											containerBottom: cont.y + cont.height,
											itemBottom: r.y + r.height
										});
									}
								});
							});
						}, jitterMs);
					});
				});
			});
			return;
		}
		updateHeightAndScroll({
			initialized: heightManager.initialized,
			mode: mode(),
			containerElement: heightManager.containerElement,
			viewportElement: heightManager.viewportElement,
			calculatedItemHeight: heightManager.averageHeight,
			height: get$1(height),
			scrollTop: heightManager.scrollTop
		}, {
			setHeight: (h) => set$1(height, h, true),
			setScrollTop: (st) => heightManager.scrollTop = st,
			setInitialized: (i) => {
				if (i && heightManager.initialized) return;
				heightManager.initialized = i;
			}
		}, immediate);
		log("updateHeightAndScroll-exit", { immediate });
	};
	itemResizeObserver = new ResizeObserver((entries) => {
		rafSchedule(() => {
			log("item-resize-observer", { entries: entries.length });
			let shouldRecalculate = false;
			get$1(visibleItems)();
			for (const entry of entries) {
				const element = entry.target;
				const elementIndex = itemElements.indexOf(element);
				const actualIndex = parseInt(element.dataset.originalIndex || "-1", 10);
				if (elementIndex !== -1) {
					if (actualIndex >= 0) {
						const currentHeight = element.getBoundingClientRect().height;
						if (isSignificantHeightChange(actualIndex, currentHeight, heightManager.getHeightCache())) {
							if (get$1(dirtyItemsCount) === 0) wasAtBottomBeforeHeightChange = get$1(atBottom);
							dirtyItems.add(actualIndex);
							set$1(dirtyItemsCount, dirtyItems.size, true);
							shouldRecalculate = true;
						}
					}
				}
			}
			if (shouldRecalculate) {
				log("item-resize-recalc");
				updateHeight();
			}
		});
	});
	onMount$1(() => {
		log("onMount-enter", {
			mode: mode(),
			items: items().length
		});
		updateHeightAndScroll$1();
		tick$1().then(() => requestAnimationFrame(() => requestAnimationFrame(() => {
			log("post-hydration-measure");
			updateHeight();
		})));
		resizeObserver = new ResizeObserver(() => {
			if (!heightManager.initialized) {
				log("container-resize-ignored", "not-initialized");
				return;
			}
			log("container-resize");
			updateHeightAndScroll$1(true);
		});
		if (heightManager.isReady) resizeObserver.observe(heightManager.container);
		return () => {
			if (resizeObserver) resizeObserver.disconnect();
			if (itemResizeObserver) itemResizeObserver.disconnect();
		};
	});
	user_effect(() => {
		if (INTERNAL_DEBUG) {
			set$1(prevVisibleRange, get$1(visibleItems)(), true);
			set$1(prevHeight, heightManager.averageHeight, true);
		}
	});
	const scrollToIndex = (index, smoothScroll = true, shouldThrowOnBounds = true) => {
		console.warn("SvelteVirtualList: scrollToIndex is deprecated and will be removed in a future version. Use the new scroll method from the component instance instead.");
		scroll({
			index,
			smoothScroll,
			shouldThrowOnBounds
		});
	};
	const scroll = async (options) => {
		const { index, smoothScroll, shouldThrowOnBounds, align } = {
			...DEFAULT_SCROLL_OPTIONS,
			...options
		};
		if (!items().length) return;
		if (!heightManager.viewportElement) {
			tick$1().then(() => {
				if (!heightManager.viewportElement) return;
				scroll({
					index,
					smoothScroll,
					shouldThrowOnBounds,
					align
				});
			});
			return;
		}
		let targetIndex = index;
		if (targetIndex < 0 || targetIndex >= items().length) if (shouldThrowOnBounds) throw new Error(`scroll: index ${targetIndex} is out of bounds (0-${items().length - 1})`);
		else targetIndex = Math.max(0, Math.min(targetIndex, items().length - 1));
		const { start: firstVisibleIndex, end: lastVisibleIndex } = get$1(visibleItems)();
		const scrollTarget = calculateScrollTarget({
			mode: mode(),
			align: align || "auto",
			targetIndex,
			itemsLength: items().length,
			calculatedItemHeight: heightManager.averageHeight,
			height: get$1(height),
			scrollTop: heightManager.scrollTop,
			firstVisibleIndex,
			lastVisibleIndex,
			heightCache: heightManager.getHeightCache()
		});
		if (scrollTarget === null) return;
		set$1(programmaticScrollInProgress, true);
		if (INTERNAL_DEBUG && heightManager.viewportElement) {
			const domMax = Math.max(0, heightManager.viewport.scrollHeight - heightManager.viewport.clientHeight);
			console.info("[SVL] scroll-intent", {
				targetIndex,
				align: align || "auto",
				firstVisibleIndex,
				lastVisibleIndex,
				currentScrollTop: heightManager.scrollTop,
				scrollTarget,
				domMaxScrollTop: domMax
			});
		}
		if (mode() === "bottomToTop" && smoothScroll) {
			const visibleElements = heightManager.viewport.querySelectorAll("[data-original-index]");
			let maxIndex = -1;
			let maxElement = null;
			for (const el of visibleElements) {
				const index = parseInt(el.getAttribute("data-original-index") || "-1");
				if (index > maxIndex) {
					maxIndex = index;
					maxElement = el;
				}
			}
			maxElement?.scrollIntoView({ behavior: "smooth" });
			await tick$1();
			await new Promise((resolve) => setTimeout(resolve, 100));
			await tick$1();
		}
		heightManager.viewport.scrollTo({
			top: scrollTarget,
			behavior: smoothScroll ? "smooth" : "auto"
		});
		requestAnimationFrame(() => {
			heightManager.scrollTop = scrollTarget;
			if (INTERNAL_DEBUG && heightManager.viewportElement) {
				const domMax = Math.max(0, heightManager.viewport.scrollHeight - heightManager.viewport.clientHeight);
				console.info("[SVL] scroll-after-call", {
					scrollTop: heightManager.scrollTop,
					domMaxScrollTop: domMax
				});
			}
		});
		setTimeout(() => {
			set$1(programmaticScrollInProgress, false);
		}, smoothScroll ? 500 : 100);
	};
	function autoObserveItemResize(element) {
		if (itemResizeObserver) itemResizeObserver.observe(element);
		return { destroy() {
			if (itemResizeObserver) itemResizeObserver.unobserve(element);
		} };
	}
	var $$exports = {
		scrollToIndex,
		scroll
	};
	var div = root$10();
	attribute_effect(div, () => ({
		id: "virtual-list-container",
		...$$props.testId ? { "data-testid": `${$$props.testId}-container` } : {},
		class: $$props.containerClass ?? "virtual-list-container"
	}), void 0, void 0, void 0, "afS");
	var div_1 = child(div);
	attribute_effect(div_1, () => ({
		id: "virtual-list-viewport",
		...$$props.testId ? { "data-testid": `${$$props.testId}-viewport` } : {},
		class: $$props.viewportClass ?? "virtual-list-viewport",
		onscroll: handleScroll
	}), void 0, void 0, void 0, "afS");
	var div_2 = child(div_1);
	attribute_effect(div_2, ($0) => ({
		id: "virtual-list-content",
		...$$props.testId ? { "data-testid": `${$$props.testId}-content` } : {},
		class: $$props.contentClass ?? "virtual-list-content",
		[STYLE]: $0
	}), [() => ({ height: `${Math.max(get$1(height), get$1(totalHeight)()) ?? ""}px` })], void 0, void 0, "afS");
	var div_3 = child(div_2);
	attribute_effect(div_3, ($0) => ({
		id: "virtual-list-items",
		...$$props.testId ? { "data-testid": `${$$props.testId}-items` } : {},
		class: $$props.itemsClass ?? "virtual-list-items",
		[STYLE]: $0
	}), [() => ({
		visibility: get$1(height) === 0 && mode() === "bottomToTop" ? "hidden" : "visible",
		transform: `translateY(${(() => {
			const viewportHeight = get$1(height) || get$1(measuredFallbackHeight) || 0;
			const visibleRange = get$1(visibleItems)();
			const effectiveHeight = viewportHeight === 0 ? 400 : viewportHeight;
			return Math.round(calculateTransformY(mode(), items().length, visibleRange.end, visibleRange.start, heightManager.averageHeight, effectiveHeight, get$1(totalHeight)(), heightManager.getHeightCache(), get$1(measuredFallbackHeight)));
		})() ?? ""}px)`
	})], void 0, void 0, "afS");
	each(div_3, 23, () => get$1(displayItems)(), (currentItemWithIndex) => currentItemWithIndex.originalIndex, ($$anchor, currentItemWithIndex, i) => {
		var fragment = root_1$8();
		var node = first_child(fragment);
		var consequent = ($$anchor) => {
			const debugInfo = /* @__PURE__ */ user_derived(() => createDebugInfo(get$1(visibleItems)(), items().length, Object.keys(heightManager.getHeightCache()).length, heightManager.averageHeight, heightManager.scrollTop, get$1(height) || 0, get$1(totalHeight)()));
			var text$1 = text();
			template_effect(($0) => set_text(text$1, $0), [() => $$props.debugFunction ? $$props.debugFunction(get$1(debugInfo)) : console.info("Virtual List Debug:", get$1(debugInfo))]);
			append($$anchor, text$1);
		};
		if_block(node, ($$render) => {
			if (debug() && get$1(i) === 0 && shouldShowDebugInfo(get$1(prevVisibleRange), get$1(visibleItems)(), get$1(prevHeight), heightManager.averageHeight)) $$render(consequent);
		});
		var div_4 = sibling(node, 2);
		attribute_effect(div_4, () => ({
			"data-original-index": get$1(currentItemWithIndex).originalIndex,
			...$$props.testId ? { "data-testid": `${$$props.testId}-item-${get$1(currentItemWithIndex).originalIndex}` } : {}
		}), void 0, void 0, void 0, "afS");
		snippet(child(div_4), () => $$props.renderItem, () => get$1(currentItemWithIndex).item, () => get$1(currentItemWithIndex).originalIndex);
		reset(div_4);
		bind_this(div_4, ($$value, currentItemWithIndex) => itemElements[currentItemWithIndex.sliceIndex] = $$value, (currentItemWithIndex) => itemElements?.[currentItemWithIndex.sliceIndex], () => [get$1(currentItemWithIndex)]);
		action(div_4, ($$node) => autoObserveItemResize?.($$node));
		append($$anchor, fragment);
	});
	reset(div_3);
	reset(div_2);
	reset(div_1);
	bind_this(div_1, ($$value) => heightManager.viewportElement = $$value, () => heightManager?.viewportElement);
	reset(div);
	bind_this(div, ($$value) => heightManager.containerElement = $$value, () => heightManager?.containerElement);
	append($$anchor, div);
	return pop($$exports);
}
var RecomputeScheduler = class {
	onRecompute;
	isScheduled = false;
	isPending = false;
	blockDepth = 0;
	timeoutId = null;
	rafId = null;
	constructor(onRecompute) {
		this.onRecompute = onRecompute;
	}
	schedule = () => {
		if (this.blockDepth > 0) {
			this.isPending = true;
			return;
		}
		if (this.isScheduled) return;
		this.isScheduled = true;
		if (!(typeof window !== "undefined" && typeof requestAnimationFrame === "function")) {
			if (this.timeoutId) {
				clearTimeout(this.timeoutId);
				this.timeoutId = null;
			}
			this.timeoutId = setTimeout(() => {
				this.timeoutId = null;
				this.isScheduled = false;
				this.onRecompute();
			}, 0);
			return;
		}
		if (this.rafId !== null) cancelAnimationFrame(this.rafId);
		this.rafId = requestAnimationFrame(() => {
			this.rafId = null;
			this.isScheduled = false;
			this.onRecompute();
		});
	};
	block = () => {
		this.blockDepth += 1;
		if (this.timeoutId) {
			clearTimeout(this.timeoutId);
			this.timeoutId = null;
			this.isScheduled = false;
			this.isPending = true;
		}
		if (this.rafId !== null) {
			cancelAnimationFrame(this.rafId);
			this.rafId = null;
			this.isScheduled = false;
			this.isPending = true;
		}
	};
	unblock = () => {
		if (this.blockDepth === 0) return;
		this.blockDepth -= 1;
		if (this.blockDepth === 0 && this.isPending) {
			this.isPending = false;
			this.onRecompute();
		}
	};
	cancel = () => {
		if (this.timeoutId) {
			clearTimeout(this.timeoutId);
			this.timeoutId = null;
		}
		if (this.rafId !== null) {
			cancelAnimationFrame(this.rafId);
			this.rafId = null;
		}
		this.isScheduled = false;
		this.isPending = false;
	};
};
var ReactiveListManager = class {
	#_totalMeasuredHeight = /* @__PURE__ */ state(0);
	get _totalMeasuredHeight() {
		return get$1(this.#_totalMeasuredHeight);
	}
	set _totalMeasuredHeight(value) {
		set$1(this.#_totalMeasuredHeight, value, true);
	}
	#_measuredCount = /* @__PURE__ */ state(0);
	get _measuredCount() {
		return get$1(this.#_measuredCount);
	}
	set _measuredCount(value) {
		set$1(this.#_measuredCount, value, true);
	}
	#_itemLength = /* @__PURE__ */ state(0);
	get _itemLength() {
		return get$1(this.#_itemLength);
	}
	set _itemLength(value) {
		set$1(this.#_itemLength, value, true);
	}
	#_itemHeight = /* @__PURE__ */ state(40);
	get _itemHeight() {
		return get$1(this.#_itemHeight);
	}
	set _itemHeight(value) {
		set$1(this.#_itemHeight, value, true);
	}
	#_averageHeight = /* @__PURE__ */ state(40);
	get _averageHeight() {
		return get$1(this.#_averageHeight);
	}
	set _averageHeight(value) {
		set$1(this.#_averageHeight, value, true);
	}
	#_totalHeight = /* @__PURE__ */ state(0);
	get _totalHeight() {
		return get$1(this.#_totalHeight);
	}
	set _totalHeight(value) {
		set$1(this.#_totalHeight, value, true);
	}
	_measuredFlags = null;
	#_initialized = /* @__PURE__ */ state(false);
	get _initialized() {
		return get$1(this.#_initialized);
	}
	set _initialized(value) {
		set$1(this.#_initialized, value, true);
	}
	#_scrollTop = /* @__PURE__ */ state(0);
	get _scrollTop() {
		return get$1(this.#_scrollTop);
	}
	set _scrollTop(value) {
		set$1(this.#_scrollTop, value, true);
	}
	#_containerElement = /* @__PURE__ */ state(null);
	get _containerElement() {
		return get$1(this.#_containerElement);
	}
	set _containerElement(value) {
		set$1(this.#_containerElement, value, true);
	}
	#_viewportElement = /* @__PURE__ */ state(null);
	get _viewportElement() {
		return get$1(this.#_viewportElement);
	}
	set _viewportElement(value) {
		set$1(this.#_viewportElement, value, true);
	}
	_internalDebug = false;
	#_isReady = /* @__PURE__ */ state(false);
	get _isReady() {
		return get$1(this.#_isReady);
	}
	set _isReady(value) {
		set$1(this.#_isReady, value, true);
	}
	#_dynamicUpdateInProgress = /* @__PURE__ */ state(false);
	get _dynamicUpdateInProgress() {
		return get$1(this.#_dynamicUpdateInProgress);
	}
	set _dynamicUpdateInProgress(value) {
		set$1(this.#_dynamicUpdateInProgress, value, true);
	}
	#_dynamicUpdateDepth = /* @__PURE__ */ state(0);
	get _dynamicUpdateDepth() {
		return get$1(this.#_dynamicUpdateDepth);
	}
	set _dynamicUpdateDepth(value) {
		set$1(this.#_dynamicUpdateDepth, value, true);
	}
	#_itemsWrapperElement = /* @__PURE__ */ state(null);
	get _itemsWrapperElement() {
		return get$1(this.#_itemsWrapperElement);
	}
	set _itemsWrapperElement(value) {
		set$1(this.#_itemsWrapperElement, value, true);
	}
	#_gridDetected = /* @__PURE__ */ state(false);
	get _gridDetected() {
		return get$1(this.#_gridDetected);
	}
	set _gridDetected(value) {
		set$1(this.#_gridDetected, value, true);
	}
	#_gridColumns = /* @__PURE__ */ state(1);
	get _gridColumns() {
		return get$1(this.#_gridColumns);
	}
	set _gridColumns(value) {
		set$1(this.#_gridColumns, value, true);
	}
	_gridObserver = null;
	_mutationObserver = null;
	_heightCache = {};
	_scheduler = new RecomputeScheduler(() => this.recomputeDerivedHeights());
	recomputeDerivedHeights() {
		const average = this._measuredCount > 0 ? this._totalMeasuredHeight / this._measuredCount : this._itemHeight;
		this._averageHeight = average;
		const unmeasuredCount = this._itemLength - this._measuredCount;
		this._totalHeight = this._totalMeasuredHeight + unmeasuredCount * average;
	}
	recomputeIsReady() {
		this._isReady = !!this._containerElement && !!this._viewportElement;
	}
	scheduleRecomputeDerivedHeights() {
		const isJsdom = typeof navigator !== "undefined" && typeof navigator.userAgent === "string" ? /jsdom/i.test(navigator.userAgent) : false;
		if (typeof window === "undefined" || isJsdom) {
			this.recomputeDerivedHeights();
			return;
		}
		if (this._dynamicUpdateDepth > 0) {
			this._scheduler.block();
			return;
		}
		this._scheduler.schedule();
	}
	get totalMeasuredHeight() {
		return this._totalMeasuredHeight;
	}
	get measuredCount() {
		return this._measuredCount;
	}
	get itemLength() {
		return this._itemLength;
	}
	get itemHeight() {
		return this._itemHeight;
	}
	set itemHeight(value) {
		this._itemHeight = value;
		this.scheduleRecomputeDerivedHeights();
	}
	get initialized() {
		return this._initialized;
	}
	set initialized(value) {
		if (this._initialized) throw new Error("ReactiveListManager: initialized flag cannot be set to true after it has been set to true");
		this._initialized = value;
	}
	get scrollTop() {
		return this._scrollTop;
	}
	set scrollTop(value) {
		if (this._internalDebug) this.#debugCheckScrollTopRepeat(value);
		this._scrollTop = value;
	}
	get containerElement() {
		return this._containerElement;
	}
	get container() {
		if (!this._isReady) throw new Error("ReactiveListManager: container is not ready");
		return this._containerElement;
	}
	set containerElement(el) {
		this._containerElement = el;
		this.recomputeIsReady();
	}
	get viewportElement() {
		return this._viewportElement;
	}
	get viewport() {
		if (!this._isReady) throw new Error("ReactiveListManager: viewport is not ready");
		return this._viewportElement;
	}
	set viewportElement(el) {
		this._viewportElement = el;
		this.recomputeIsReady();
	}
	get itemsWrapperElement() {
		return this._itemsWrapperElement;
	}
	set itemsWrapperElement(el) {
		if (this._itemsWrapperElement !== el) {
			if (this._gridObserver) {
				try {
					this._gridObserver.disconnect();
				} catch {}
				this._gridObserver = null;
			}
			if (this._mutationObserver) {
				try {
					this._mutationObserver.disconnect();
				} catch {}
				this._mutationObserver = null;
			}
		}
		this._itemsWrapperElement = el;
		if (!el) {
			this._gridDetected = false;
			this._gridColumns = 1;
			return;
		}
		this.#attachGridObserver();
		this.#attachMutationObserver();
		this.#detectGridColumns();
	}
	get gridDetected() {
		return this._gridDetected;
	}
	get gridColumns() {
		return this._gridColumns;
	}
	get isReady() {
		return this._isReady;
	}
	get isDynamicUpdateInProgress() {
		return this._dynamicUpdateDepth > 0;
	}
	startDynamicUpdate() {
		const isOuter = this._dynamicUpdateDepth === 0;
		this._dynamicUpdateDepth += 1;
		if (isOuter) {
			this._dynamicUpdateInProgress = true;
			if (this._isReady && this._viewportElement) this._viewportElement.style.setProperty("overflow-anchor", "none");
		}
	}
	endDynamicUpdate() {
		if (this._dynamicUpdateDepth <= 0) return;
		this._dynamicUpdateDepth -= 1;
		if (this._dynamicUpdateDepth === 0) {
			if (this._isReady && this._viewportElement) this._viewportElement.style.setProperty("overflow-anchor", "auto");
			this._dynamicUpdateInProgress = false;
			this._scheduler.unblock();
		}
	}
	async runDynamicUpdate(fn) {
		this.startDynamicUpdate();
		try {
			const result = fn();
			return result instanceof Promise ? await result : result;
		} finally {
			this.endDynamicUpdate();
		}
	}
	#debugLastScrollValue = null;
	#debugWindowStartMs = 0;
	#debugRepeatCount = 0;
	#debugWarnedThisWindow = false;
	#debugCheckScrollTopRepeat(value) {
		const now = typeof performance !== "undefined" && performance.now ? performance.now() : Date.now();
		if (this.#debugLastScrollValue === value) if (now - this.#debugWindowStartMs <= 1e3) {
			this.#debugRepeatCount += 1;
			if (this.#debugRepeatCount > 10 && !this.#debugWarnedThisWindow) {
				this.#debugWarnedThisWindow = true;
				console.warn(`
================ SvelteVirtualList DEBUG ================
scrollTop assigned same value ${value} > 10 times within 1s\ncount=${this.#debugRepeatCount}, windowStart=${Math.round(this.#debugWindowStartMs)}ms\nThis may indicate redundant updates or feedback loops.
========================================================
`);
			}
		} else {
			this.#debugWindowStartMs = now;
			this.#debugRepeatCount = 1;
			this.#debugWarnedThisWindow = false;
		}
		else {
			this.#debugLastScrollValue = value;
			this.#debugWindowStartMs = now;
			this.#debugRepeatCount = 1;
			this.#debugWarnedThisWindow = false;
		}
	}
	get averageHeight() {
		return this._averageHeight;
	}
	get totalHeight() {
		return this._totalHeight;
	}
	flushRecompute = () => {
		this.recomputeDerivedHeights();
	};
	getHeightCache() {
		return this._heightCache;
	}
	constructor(config) {
		this._itemLength = config.itemLength;
		this._itemHeight = config.itemHeight;
		this._internalDebug = config.internalDebug ?? false;
		this._measuredFlags = new Uint8Array(Math.max(0, this._itemLength));
		this.recomputeDerivedHeights();
	}
	processDirtyHeights(dirtyResults) {
		if (dirtyResults.length === 0) return;
		let heightDelta = 0;
		let countDelta = 0;
		for (const change of dirtyResults) {
			const { index, oldHeight, newHeight } = change;
			if (oldHeight !== void 0) {
				heightDelta -= oldHeight;
				countDelta -= 1;
			}
			if (newHeight !== void 0) {
				heightDelta += newHeight;
				countDelta += 1;
				this._heightCache[index] = newHeight;
			} else delete this._heightCache[index];
			if (this._measuredFlags && index >= 0 && index < this._measuredFlags.length) this._measuredFlags[index] = 1;
		}
		const isJsdom = typeof navigator !== "undefined" && typeof navigator.userAgent === "string" ? /jsdom/i.test(navigator.userAgent) : false;
		if (typeof window === "undefined" || isJsdom) {
			if (heightDelta === 0 && countDelta === 0) return;
		} else if (countDelta === 0) return;
		this._totalMeasuredHeight += heightDelta;
		this._measuredCount += countDelta;
		this.scheduleRecomputeDerivedHeights();
	}
	updateItemLength(newLength) {
		this._itemLength = newLength;
		this._measuredFlags = new Uint8Array(Math.max(0, newLength));
		this.recomputeDerivedHeights();
	}
	updateEstimatedHeight(newEstimatedHeight) {
		this._itemHeight = newEstimatedHeight;
		this.scheduleRecomputeDerivedHeights();
	}
	setMeasuredHeight(index, height) {
		if (index < 0 || index >= this._itemLength) return;
		const prev = this._heightCache[index];
		if (Number.isFinite(prev) && prev > 0) this._totalMeasuredHeight -= prev;
		else this._measuredCount += 1;
		if (Number.isFinite(height) && height > 0) {
			this._heightCache[index] = height;
			this._totalMeasuredHeight += height;
			this.scheduleRecomputeDerivedHeights();
		}
	}
	reset() {
		this._totalMeasuredHeight = 0;
		this._measuredCount = 0;
		this._measuredFlags = this._itemLength > 0 ? new Uint8Array(this._itemLength) : null;
		this.scheduleRecomputeDerivedHeights();
	}
	getDebugInfo() {
		return {
			totalMeasuredHeight: this._totalMeasuredHeight,
			measuredCount: this._measuredCount,
			itemLength: this._itemLength,
			coveragePercent: this._itemLength > 0 ? this._measuredCount / this._itemLength * 100 : 0,
			itemHeight: this._itemHeight,
			averageHeight: this.averageHeight,
			totalHeight: this.totalHeight,
			gridDetected: this._gridDetected,
			gridColumns: this._gridColumns
		};
	}
	getMeasurementCoverage() {
		return this.getDebugInfo().coveragePercent;
	}
	hasSufficientMeasurements(threshold = 10) {
		return this.getMeasurementCoverage() >= threshold;
	}
	recomputeGridDetection() {
		this.#detectGridColumns();
	}
	#attachGridObserver() {
		const el = this._itemsWrapperElement;
		if (typeof window === "undefined" || !el) return;
		try {
			this._gridObserver = new ResizeObserver(() => {
				this.#detectGridColumns();
			});
			this._gridObserver.observe(el);
		} catch {
			this._gridObserver = null;
		}
	}
	#attachMutationObserver() {
		const el = this._itemsWrapperElement;
		if (typeof window === "undefined" || !el) return;
		try {
			this._mutationObserver = new MutationObserver((records) => {
				for (const rec of records) if (rec.type === "attributes" && (rec.attributeName === "class" || rec.attributeName === "style")) {
					this.#detectGridColumns();
					break;
				}
			});
			this._mutationObserver.observe(el, {
				attributes: true,
				attributeFilter: ["class", "style"]
			});
		} catch {
			this._mutationObserver = null;
		}
	}
	#detectGridColumns() {
		const el = this._itemsWrapperElement;
		if (!el) {
			this._gridDetected = false;
			this._gridColumns = 1;
			return;
		}
		let detected = false;
		let columns = 1;
		try {
			const style = getComputedStyle(el);
			if (style.display === "grid") {
				const template = style.gridTemplateColumns;
				const repeatMatch = /repeat\(\s*(\d+)\s*,/i.exec(template);
				if (repeatMatch && repeatMatch[1]) {
					columns = Math.max(1, parseInt(repeatMatch[1], 10));
					detected = true;
				} else if (template && template !== "none") {
					const count = this.#countTracksFromTemplate(template);
					if (Number.isFinite(count) && count > 0) {
						columns = count;
						detected = true;
					}
				}
			}
		} catch {}
		if (!detected) {
			const children = el.children;
			if (children && children.length > 0) {
				const firstTop = children[0].getBoundingClientRect().top;
				let countSameRow = 0;
				for (let i = 0; i < children.length; i += 1) {
					const top = children[i].getBoundingClientRect().top;
					if (Math.abs(top - firstTop) <= 1) countSameRow += 1;
					else break;
				}
				if (countSameRow > 0) {
					columns = countSameRow;
					detected = countSameRow > 1;
				}
			}
		}
		this._gridDetected = detected;
		this._gridColumns = Math.max(1, columns);
		if (this._internalDebug) console.info("[ReactiveListManager] grid detection:", {
			detected: this._gridDetected,
			columns: this._gridColumns
		});
	}
	#countTracksFromTemplate(template) {
		let depth = 0;
		let tokens = 0;
		let inToken = false;
		for (let i = 0; i < template.length; i += 1) {
			const ch = template[i];
			if (ch === "(") depth += 1;
			else if (ch === ")") depth = Math.max(0, depth - 1);
			if (depth === 0 && /\s/.test(ch)) {
				if (inToken) {
					tokens += 1;
					inToken = false;
				}
			} else if (ch !== " ") inToken = true;
		}
		if (inToken) tokens += 1;
		return tokens;
	}
};
var dist_default = SvelteVirtualList;
var root_1$7 = /* @__PURE__ */ from_html(`<div><div></div></div> <div> <!></div> <div><div></div></div> <div><div></div></div>`, 1);
var root_2$5 = /* @__PURE__ */ from_html(`<div><div></div></div>`);
var root_3$3 = /* @__PURE__ */ from_html(`<input/>`);
var root_6$2 = /* @__PURE__ */ from_html(`<div class="fs15 mt-2 _10 afR"> </div>`);
var root_4$2 = /* @__PURE__ */ from_html(`<div role="button" tabindex="0"><div class="h100 w100 flex items-center"><!></div></div>`);
var root_8$2 = /* @__PURE__ */ from_html(`<div>Cargando...</div>`);
var root_9$2 = /* @__PURE__ */ from_html(`<div><i></i></div>`);
var root_12 = /* @__PURE__ */ from_html(`<span> </span>`);
var root_11 = /* @__PURE__ */ from_html(`<div role="option" tabindex="0"><div></div></div>`);
var root_10$2 = /* @__PURE__ */ from_html(`<div role="button" tabindex="0"><!></div>`);
var root$9 = /* @__PURE__ */ from_html(`<div><!> <div><!> <!> <!></div> <!> <!> <!></div>`);
function SearchSelect($$anchor, $$props) {
	push($$props, true);
	const saveOn = prop($$props, "saveOn", 15), css = prop($$props, "css", 3, ""), options = prop($$props, "options", 19, () => []);
	prop($$props, "max", 3, 100);
	const notEmpty = prop($$props, "notEmpty", 3, false), required = prop($$props, "required", 3, false), disabled = prop($$props, "disabled", 3, false), clearOnSelect = prop($$props, "clearOnSelect", 3, false), avoidIDs = prop($$props, "avoidIDs", 19, () => []), inputCss = prop($$props, "inputCss", 3, ""), showLoading = prop($$props, "showLoading", 3, false);
	let show = /* @__PURE__ */ state(false);
	let filteredOptions = /* @__PURE__ */ state(proxy([...options()]));
	let arrowSelected = /* @__PURE__ */ state(-1);
	let avoidhover = /* @__PURE__ */ state(false);
	let isValid = /* @__PURE__ */ state(0);
	let selectedValue = /* @__PURE__ */ state("");
	let avoidBlur = false;
	let openUp = /* @__PURE__ */ state(false);
	let inputRef = /* @__PURE__ */ state(void 0);
	let vlRef = /* @__PURE__ */ state(void 0);
	let words = /* @__PURE__ */ state(proxy([]));
	const isMobile = /* @__PURE__ */ user_derived(() => Core.deviceType === 3);
	const useLayerPicker = /* @__PURE__ */ user_derived(() => get$1(isMobile));
	const isDisabled = /* @__PURE__ */ user_derived(() => disabled() || showLoading());
	function checkPosition() {
		if (get$1(inputRef)) {
			const rect = get$1(inputRef).getBoundingClientRect();
			const windowHeight = window.innerHeight;
			set$1(openUp, rect.top > windowHeight / 2);
		}
	}
	function changeValue(value) {
		if (get$1(useLayerPicker)) set$1(selectedValue, value, true);
		else if (get$1(inputRef)) get$1(inputRef).value = value;
	}
	function getSelectedFromProps() {
		let currValue = $$props.selected;
		if (typeof currValue === "undefined" && $$props.save && saveOn()) currValue = saveOn()[$$props.save];
		let selectedItem;
		if (currValue) selectedItem = options().find((x) => x[$$props.keyId] === currValue);
		return selectedItem;
	}
	function isRequired() {
		return required() && !disabled();
	}
	let prevSetValueTime = Date.now();
	const setValueSaveOn = (selectedItem, setOnInput) => {
		const nowTime = Date.now();
		if (nowTime - prevSetValueTime < 80) return;
		prevSetValueTime = nowTime;
		if (notEmpty() && !selectedItem) selectedItem = getSelectedFromProps();
		if (setOnInput) if (selectedItem && !clearOnSelect()) changeValue(selectedItem[$$props.keyName]);
		else changeValue("");
		const newValue = selectedItem ? selectedItem[$$props.keyId] : null;
		if (isRequired()) set$1(isValid, newValue ? 2 : 1, true);
		if (clearOnSelect()) {
			if ($$props.onChange) $$props.onChange(selectedItem);
		} else if (saveOn() && $$props.save) {
			if ((saveOn()[$$props.save] || null) !== newValue) {
				saveOn(saveOn()[$$props.save] = newValue, true);
				if ($$props.onChange) $$props.onChange(selectedItem);
			}
		} else if (typeof $$props.selected !== "undefined") {
			if (($$props.selected || null) !== newValue) {
				if ($$props.onChange) $$props.onChange(selectedItem);
			}
		}
	};
	const filter = (text) => {
		console.log("options filtered::", $$props.label, options());
		if (!text && (avoidIDs()?.length || 0) === 0) return [...options()];
		const avoidIDSet = new Set(avoidIDs() || []);
		const filtered = [];
		const searchWords = text.toLowerCase().split(" ").filter((x) => x.length > 1);
		for (const opt of options()) {
			if (avoidIDSet.size > 0 && avoidIDSet.has(opt[$$props.keyId])) continue;
			if (searchWords.length === 0) filtered.push(opt);
			else {
				const name = opt[$$props.keyName];
				if (typeof name === "string") {
					if (include(name.toLowerCase(), searchWords)) filtered.push(opt);
				}
			}
		}
		console.log("options filtered 2::", $$props.label, options(), filtered);
		return filtered;
	};
	function onKeyUp(ev) {
		ev.stopPropagation();
		const target = ev.target;
		const text = String(target.value || "").toLowerCase();
		throttle(() => {
			set$1(words, String(get$1(inputRef).value).toLowerCase().split(" "), true);
			set$1(filteredOptions, filter(text), true);
			set$1(arrowSelected, -1);
		}, 120);
	}
	function onKeyDown(ev) {
		console.log("avoid hover:: ", get$1(avoidhover));
		ev.stopPropagation();
		if (!get$1(show) || get$1(filteredOptions).length === 0) return;
		if (ev.key === "ArrowUp") {
			ev.preventDefault();
			set$1(arrowSelected, get$1(arrowSelected) <= 0 ? get$1(filteredOptions).length - 1 : get$1(arrowSelected) - 1, true);
			set$1(avoidhover, true);
			get$1(vlRef)?.scrollToIndex(get$1(arrowSelected), { align: "auto" });
		} else if (ev.key === "ArrowDown") {
			ev.preventDefault();
			set$1(arrowSelected, get$1(arrowSelected) >= get$1(filteredOptions).length - 1 ? 0 : get$1(arrowSelected) + 1, true);
			set$1(avoidhover, true);
			get$1(vlRef)?.scrollToIndex(get$1(arrowSelected), { align: "auto" });
		} else if (ev.key === "Enter" && get$1(arrowSelected) >= 0) {
			ev.preventDefault();
			onOptionClick(get$1(filteredOptions)[get$1(arrowSelected)]);
		}
	}
	function onOptionClick(opt) {
		setValueSaveOn(opt, true);
		if (get$1(inputRef)) get$1(inputRef).blur();
		set$1(show, false);
	}
	let cN = /* @__PURE__ */ user_derived(() => `${components_module_default.input} p-rel${css() ? ` ${css()}` : ""}${!$$props.label ? " no-label" : ""}`);
	function iconValid() {
		if (!get$1(isValid)) return null;
		if (get$1(isValid) === 2) return `<i class="v-icon icon-ok text-green-600"></i>`;
		else if (get$1(isValid) === 1) return `<i class="v-icon icon-attention text-red-600"></i>`;
		return null;
	}
	user_effect(() => {
		const selectedItem = getSelectedFromProps();
		if (selectedItem) changeValue(selectedItem[$$props.keyName]);
		else changeValue("");
		if (isRequired()) set$1(isValid, selectedItem ? 2 : 1, true);
	});
	user_effect(() => {
		set$1(filteredOptions, filter(""), true);
		set$1(arrowSelected, -1);
	});
	const handleOpenMobileLayer = () => {
		Core.showMobileSearchLayer = {
			options: get$1(filteredOptions),
			keyName: $$props.keyName,
			keyID: $$props.keyId,
			onSelect: (e) => {
				onOptionClick(e);
			},
			onRemove: (e) => {}
		};
	};
	var div = root$9();
	var node = child(div);
	var consequent = ($$anchor) => {
		var fragment = root_1$7();
		var div_1 = first_child(fragment);
		var div_2 = sibling(div_1, 2);
		var text_1 = child(div_2, true);
		html(sibling(text_1), () => iconValid() || "");
		reset(div_2);
		var div_3 = sibling(div_2, 2);
		var div_4 = sibling(div_3, 2);
		template_effect(() => {
			set_class(div_1, 1, clsx(components_module_default.input_lab_cell_left), "afR");
			set_class(div_2, 1, clsx(components_module_default.input_lab), "afR");
			set_text(text_1, $$props.label);
			set_class(div_3, 1, clsx(components_module_default.input_lab_cell_right), "afR");
			set_class(div_4, 1, clsx(components_module_default.input_shadow_layer), "afR");
		});
		append($$anchor, fragment);
	};
	if_block(node, ($$render) => {
		if ($$props.label) $$render(consequent);
	});
	var div_5 = sibling(node, 2);
	var node_2 = child(div_5);
	var consequent_1 = ($$anchor) => {
		var div_6 = root_2$5();
		template_effect(() => set_class(div_6, 1, clsx(components_module_default.input_div_1), "afR"));
		append($$anchor, div_6);
	};
	if_block(node_2, ($$render) => {
		if ($$props.label) $$render(consequent_1);
	});
	var node_3 = sibling(node_2, 2);
	var consequent_2 = ($$anchor) => {
		var input = root_3$3();
		input.__keyup = onKeyUp;
		input.__keydown = (ev) => {
			ev.stopPropagation();
			console.log(ev);
			onKeyDown(ev);
		};
		bind_this(input, ($$value) => set$1(inputRef, $$value), () => get$1(inputRef));
		template_effect(() => {
			set_class(input, 1, `w-full ${components_module_default.input_inp} ${inputCss()}`, "afR");
			set_attribute(input, "placeholder", showLoading() ? "" : $$props.placeholder || ":: seleccione ::");
			input.disabled = get$1(isDisabled) || get$1(isMobile);
		});
		event("paste", input, onKeyUp);
		event("cut", input, onKeyUp);
		event("focus", input, (ev) => {
			ev.stopPropagation();
			if (!get$1(show)) {
				checkPosition();
				set$1(words, [], true);
				set$1(filteredOptions, filter(""), true);
				set$1(show, true);
			}
		});
		event("blur", input, (ev) => {
			ev.stopPropagation();
			console.log("avoidBlur 2", avoidBlur);
			if (avoidBlur) {
				avoidBlur = false;
				get$1(inputRef).focus();
				return;
			}
			let inputValue = String(get$1(inputRef).value || "").toLowerCase();
			setValueSaveOn(options().find((x) => {
				return String(x[$$props.keyName] || "").toLowerCase() === inputValue;
			}), true);
			set$1(show, false);
		});
		append($$anchor, input);
	};
	var alternate_1 = ($$anchor) => {
		var div_7 = root_4$2();
		div_7.__keydown = (ev) => {
			if (ev.key === "Enter" || ev.key === " ") handleOpenMobileLayer();
		};
		div_7.__click = (ev) => {
			ev.stopPropagation();
			handleOpenMobileLayer();
		};
		var div_8 = child(div_7);
		var node_4 = child(div_8);
		var consequent_3 = ($$anchor) => {
			var text_2 = text();
			template_effect(() => set_text(text_2, get$1(selectedValue)));
			append($$anchor, text_2);
		};
		var alternate = ($$anchor) => {
			var div_9 = root_6$2();
			var text_3 = child(div_9, true);
			reset(div_9);
			template_effect(() => set_text(text_3, $$props.placeholder || ""));
			append($$anchor, div_9);
		};
		if_block(node_4, ($$render) => {
			if (get$1(selectedValue)) $$render(consequent_3);
			else $$render(alternate, false);
		});
		reset(div_8);
		reset(div_7);
		template_effect(() => set_class(div_7, 1, `w-full flex items-center ${components_module_default.input_inp} ${inputCss()}`, "afR"));
		append($$anchor, div_7);
	};
	if_block(node_3, ($$render) => {
		if (!get$1(useLayerPicker)) $$render(consequent_2);
		else $$render(alternate_1, false);
	});
	var node_5 = sibling(node_3, 2);
	var consequent_4 = ($$anchor) => {
		var fragment_2 = comment();
		html(first_child(fragment_2), () => iconValid() || "");
		append($$anchor, fragment_2);
	};
	if_block(node_5, ($$render) => {
		if (!$$props.label) $$render(consequent_4);
	});
	reset(div_5);
	var node_7 = sibling(div_5, 2);
	var consequent_5 = ($$anchor) => {
		append($$anchor, root_8$2());
	};
	if_block(node_7, ($$render) => {
		if (showLoading()) $$render(consequent_5);
	});
	var node_8 = sibling(node_7, 2);
	var consequent_6 = ($$anchor) => {
		var div_11 = root_9$2();
		var i_1 = child(div_11);
		reset(div_11);
		template_effect(() => {
			set_class(div_11, 1, `absolute bottom-8 right-6 pointer-events-none ${get$1(show) && !$$props.icon ? "show" : ""}`);
			set_class(i_1, 1, clsx($$props.icon || "icon-down-open-1"), "afR");
		});
		append($$anchor, div_11);
	};
	if_block(node_8, ($$render) => {
		if (!get$1(isDisabled)) $$render(consequent_6);
	});
	var node_9 = sibling(node_8, 2);
	var consequent_7 = ($$anchor) => {
		var div_12 = root_10$2();
		let classes;
		div_12.__mousedown = (ev) => {
			ev.stopPropagation();
			avoidBlur = true;
			console.log("avoidBlur 1", avoidBlur);
		};
		div_12.__mousemove = function(...$$args) {
			(get$1(avoidhover) ? (ev) => {
				console.log("hover aqui:: ", get$1(arrowSelected));
				ev.stopPropagation();
				if (get$1(avoidhover)) {
					set$1(arrowSelected, -1);
					set$1(avoidhover, false);
				}
			} : void 0)?.apply(this, $$args);
		};
		let styles;
		var node_10 = child(div_12);
		{
			const renderItem = ($$anchor, e = noop$1, i = noop$1) => {
				const name = /* @__PURE__ */ user_derived(() => String(e()[$$props.keyName]));
				const highlighted = /* @__PURE__ */ user_derived(() => highlString(get$1(name), get$1(words)));
				var div_13 = root_11();
				div_13.__click = (ev) => {
					ev.stopPropagation();
					onOptionClick(e());
				};
				var div_14 = child(div_13);
				each(div_14, 21, () => get$1(highlighted), index, ($$anchor, w) => {
					var span = root_12();
					let classes_1;
					var text_4 = child(span, true);
					reset(span);
					template_effect(() => {
						classes_1 = set_class(span, 1, clsx(get$1(w).highl ? "_8" : ""), "afR", classes_1, { "mr-4": get$1(w).isEnd });
						set_text(text_4, get$1(w).text);
					});
					append($$anchor, span);
				});
				reset(div_14);
				reset(div_13);
				template_effect(() => {
					set_class(div_13, 1, `flex ai-center afl${get$1(arrowSelected) === i() ? " afm" : ""}`, "afR");
					set_attribute(div_13, "aria-selected", get$1(arrowSelected) === i());
				});
				append($$anchor, div_13);
			};
			bind_this(dist_default(node_10, {
				get items() {
					return get$1(filteredOptions);
				},
				renderItem,
				$$slots: { renderItem: true }
			}), ($$value) => set$1(vlRef, $$value, true), () => get$1(vlRef));
		}
		reset(div_12);
		template_effect(($0) => {
			classes = set_class(div_12, 1, `_1 p-4 left-0 z-320 ${get$1(arrowSelected) >= 0 ? " on-arrow" : ""} ${($$props.optionsCss || "w-full") ?? ""}`, "afR", classes, { "open-up": get$1(openUp) });
			styles = set_style(div_12, "", styles, $0);
		}, [() => ({ height: Math.min(get$1(filteredOptions).length * 36 + 10, 300) + "px" })]);
		append($$anchor, div_12);
	};
	if_block(node_9, ($$render) => {
		if (get$1(show) && !get$1(useLayerPicker)) $$render(consequent_7);
	});
	reset(div);
	template_effect(() => {
		set_class(div, 1, clsx(get$1(cN)), "afR");
		set_class(div_5, 1, `${components_module_default.input_div} flex w-full`, "afR");
	});
	append($$anchor, div);
	pop();
}
delegate([
	"keyup",
	"keydown",
	"click",
	"mousedown",
	"mousemove"
]);
const ciudadesService = proxy({
	departamentos: [],
	provincias: [],
	distritos: [],
	ciudadHijosMap: /* @__PURE__ */ new Map(),
	ciudadesMap: /* @__PURE__ */ new Map()
});
var parseCiudades = (ciudades) => {
	const res = {
		departamentos: [],
		provincias: [],
		distritos: [],
		ciudadHijosMap: /* @__PURE__ */ new Map(),
		ciudadesMap: /* @__PURE__ */ new Map()
	};
	for (const cd of ciudades) {
		res.ciudadesMap.set(cd.ID, cd);
		if (cd.ID.length === 6) res.distritos.push(cd);
		else if (cd.ID.length === 4) res.provincias.push(cd);
		else if (cd.ID.length === 2) res.departamentos.push(cd);
		if (!cd.PadreID) continue;
		if (!cd.PadreID) continue;
		const hijos = res.ciudadHijosMap.get(cd.PadreID);
		if (hijos) hijos.push(cd);
		else res.ciudadHijosMap.set(cd.PadreID, [cd]);
	}
	return res;
};
var ciudadesPromise;
const useCiudadesAPI = () => {
	if (!ciudadesPromise) ciudadesPromise = new Promise((resolve, reject) => {
		fetch("/files/peru_ciudades.json").then((ciudades) => {
			return ciudades.json();
		}).then((ciudades) => {
			console.log("ciudades obtenidas::", ciudades);
			resolve(parseCiudades(ciudades));
		}).catch((err) => {
			reject(err);
		});
	});
	ciudadesPromise.then((res) => {
		ciudadesService.departamentos = res.departamentos;
		ciudadesService.provincias = res.provincias;
		ciudadesService.distritos = res.distritos;
		ciudadesService.ciudadHijosMap = res.ciudadHijosMap;
		ciudadesService.ciudadesMap = res.ciudadesMap;
	});
	return ciudadesService;
};
var root$8 = /* @__PURE__ */ from_html(`<!> <!> <!>`, 1);
function CiudadesSelector($$anchor, $$props) {
	push($$props, true);
	const saveOn = prop($$props, "saveOn", 7);
	let form = proxy({
		departamentoID: saveOn().departamentoID || "",
		provinciaID: saveOn().provinciaID || "",
		distritoID: saveOn().distritoID || ""
	});
	const ciudades = useCiudadesAPI();
	let provincias = /* @__PURE__ */ state(proxy([]));
	let distritos = /* @__PURE__ */ state(proxy([]));
	user_effect(() => {
		if (!saveOn() || !$$props.save || !ciudades.ciudadesMap) return;
		const ciudad = ciudades.ciudadesMap.get(saveOn()[$$props.save]);
		if (ciudad) {
			if (ciudad.ID.length === 2) form.departamentoID = ciudad.ID;
			else if (ciudad.ID.length === 4) {
				form.departamentoID = ciudad.PadreID;
				form.provinciaID = ciudad.ID;
			} else {
				form.departamentoID = ciudades.ciudadesMap.get(ciudad.PadreID).PadreID;
				form.provinciaID = ciudad.PadreID;
				form.distritoID = ciudad.ID;
			}
			if (form.departamentoID) set$1(provincias, ciudades.ciudadHijosMap.get(form.departamentoID), true);
			else set$1(provincias, [], true);
			if (form.provinciaID) set$1(distritos, ciudades.ciudadHijosMap.get(form.provinciaID), true);
			else set$1(distritos, [], true);
		} else {
			set$1(provincias, [], true);
			set$1(distritos, [], true);
			form.provinciaID = "";
			form.distritoID = "";
			form.provinciaID = "";
		}
	});
	const doSave = () => {
		if (!saveOn() || !$$props.save) return;
		const ciudadID = form.distritoID || form.provinciaID || form.departamentoID || "";
		saveOn()[$$props.save] = ciudadID;
		if ($$props.onChange) $$props.onChange();
	};
	var fragment = root$8();
	var node = first_child(fragment);
	SearchSelect(node, {
		get saveOn() {
			return form;
		},
		save: "departamentoID",
		get css() {
			return $$props.css;
		},
		label: "Departamento",
		keyId: "ID",
		keyName: "Nombre",
		required: true,
		get options() {
			return ciudades.departamentos;
		},
		onChange: (e) => {
			console.log("departamento::", e);
			form.distritoID = null;
			form.provinciaID = null;
			set$1(provincias, ciudades.ciudadHijosMap.get(e?.ID) || [], true);
			set$1(distritos, [], true);
			doSave();
		}
	});
	var node_1 = sibling(node, 2);
	SearchSelect(node_1, {
		get saveOn() {
			return form;
		},
		save: "provinciaID",
		get css() {
			return $$props.css;
		},
		label: "Provincia",
		keyId: "ID",
		keyName: "Nombre",
		required: true,
		get options() {
			return get$1(provincias);
		},
		onChange: (e) => {
			form.distritoID = null;
			set$1(distritos, ciudades.ciudadHijosMap.get(e?.ID) || [], true);
			doSave();
		}
	});
	SearchSelect(sibling(node_1, 2), {
		get saveOn() {
			return form;
		},
		save: "distritoID",
		get css() {
			return $$props.css;
		},
		label: "Distrito",
		keyId: "ID",
		keyName: "Nombre",
		required: true,
		get options() {
			return get$1(distritos);
		},
		onChange: () => {
			doSave();
		}
	});
	append($$anchor, fragment);
	pop();
}
var root_1$6 = /* @__PURE__ */ from_html(`<img class="afh afj" loading="lazy" alt=""/>`);
var root$7 = /* @__PURE__ */ from_html(`<div><!> <img class="afh afj" loading="lazy"/></div>`);
function Imagehash($$anchor, $$props) {
	push($$props, true);
	const size = prop($$props, "size", 3, 4);
	let imageSrc = /* @__PURE__ */ state(void 0);
	let placeholderSrc = /* @__PURE__ */ state("");
	if ($$props.hash?.length > 0) set$1(placeholderSrc, "/?" + $$props.hash);
	onMount$1(() => {
		if ($$props.hash?.length > 0) set$1(imageSrc, Env.S3_URL + ($$props.folder ? $$props.folder + "/" : "images/") + $$props.hash.substring(0, 12).replaceAll(".", "/").replaceAll("-", "=") + ".webp");
		else {
			const sl = $$props.src.split(".");
			const ext = sl[sl.length - 1];
			set$1(imageSrc, $$props.folder ? $$props.folder + "/" + $$props.src : $$props.src, true);
			if (sl.length < 2 || ![
				"jpeg",
				"jpg",
				"webp",
				"avif",
				"png"
			].includes(ext)) set$1(imageSrc, get$1(imageSrc) + `-x${size()}.avif`);
			set$1(imageSrc, Env.S3_URL + get$1(imageSrc));
		}
		console.log("image source::", get$1(imageSrc), "| folder", $$props.folder, "| src", $$props.src);
	});
	var div = root$7();
	var node = child(div);
	var consequent = ($$anchor) => {
		var img = root_1$6();
		template_effect(() => set_attribute(img, "role", `0/0/${$$props.src}`));
		append($$anchor, img);
	};
	if_block(node, ($$render) => {
		if (!!get$1(placeholderSrc)) $$render(consequent);
	});
	var img_1 = sibling(node, 2);
	reset(div);
	template_effect(($0) => {
		set_class(div, 1, $0, "afj");
		set_attribute(img_1, "role", `1/${size()}/${$$props.src}`);
		set_attribute(img_1, "src", get$1(imageSrc));
		set_attribute(img_1, "alt", $$props.alt);
	}, [() => clsx([styles_module_default.image_hash_ctn, $$props.css || ""].join(" "))]);
	event("load", img_1, () => {
		set$1(placeholderSrc, "");
	});
	replay_events(img_1);
	append($$anchor, div);
	pop();
}
var root$6 = /* @__PURE__ */ from_html(`<div class="flex relative _1 h-88 md:h-110 rounded-[7px] afi"><div class="p-8"><!></div> <div class="flex flex-col h-full relative pt-6 pb-4 w-full"><div> </div> <div class="flex items-center mt-auto w-full"><div class="flex"><button class="_2 h-28 w-28 rounded-[50%] afi">-</button> <input class="_3 text-center h-28 w-58 afi" type="number"/> <button class="_2 h-28 w-28 rounded-[50%] afi">+</button></div> <div class="flex ml-auto mr-6 fs17"><div class="mr-4">s/.</div> <div class="ff-bold"> </div></div></div></div> <button class="_4 absolute outline-0 border-none fx-c rounded-[50%] h-28 w-28 top-[-4px] right-[-4px] afi"><i class="icon-cancel"></i></button></div>`);
function ProductCardHorizonal($$anchor, $$props) {
	push($$props, true);
	prop($$props, "css", 3, "");
	const prodCant = /* @__PURE__ */ user_derived(() => {
		return ProductsSelectedMap.get($$props.producto.ID)?.cant || 0;
	});
	var div = root$6();
	var div_1 = child(div);
	var node = child(div_1);
	{
		let $0 = /* @__PURE__ */ user_derived(() => $$props.producto.Image?.n);
		Imagehash(node, {
			css: "w-88 md:w-100 h-[100%]",
			get src() {
				return get$1($0);
			},
			folder: "img-productos"
		});
	}
	reset(div_1);
	var div_2 = sibling(div_1, 2);
	var div_3 = child(div_2);
	var text = child(div_3, true);
	reset(div_3);
	var div_4 = sibling(div_3, 2);
	var div_5 = child(div_4);
	var button = child(div_5);
	button.__click = (ev) => {
		ev.stopPropagation();
		addProductoCant($$props.producto, null, -1);
	};
	var input = sibling(button, 2);
	remove_input_defaults(input);
	input.__change = (ev) => {
		ev.stopPropagation();
		addProductoCant($$props.producto, parseInt(ev.target.value));
	};
	var button_1 = sibling(input, 2);
	button_1.__click = (ev) => {
		ev.stopPropagation();
		addProductoCant($$props.producto, null, 1);
	};
	reset(div_5);
	var div_6 = sibling(div_5, 2);
	var div_7 = sibling(child(div_6), 2);
	var text_1 = child(div_7, true);
	reset(div_7);
	reset(div_6);
	reset(div_4);
	reset(div_2);
	var button_2 = sibling(div_2, 2);
	button_2.__click = (ev) => {
		ev.stopPropagation();
		addProductoCant($$props.producto, 0);
	};
	reset(div);
	template_effect(($0) => {
		set_text(text, $$props.producto.Nombre);
		set_value(input, get$1(prodCant));
		set_text(text_1, $0);
	}, [() => formatN($$props.producto.Precio / 100, 2)]);
	append($$anchor, div);
	pop();
}
delegate(["click", "change"]);
var angle_default = "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n<!-- Uploaded to: SVG Repo, www.svgrepo.com, Generator: SVG Repo Mixer Tools -->\n\n<svg\n   width=\"600\"\n   height=\"374.46808\"\n   viewBox=\"0 0 18 11.234042\"\n   version=\"1.1\"\n   id=\"svg1\"\n   sodipodi:docname=\"angle.svg\"\n   inkscape:version=\"1.4 (e7c3feb100, 2024-10-09)\"\n   xmlns:inkscape=\"http://www.inkscape.org/namespaces/inkscape\"\n   xmlns:sodipodi=\"http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd\"\n   xmlns=\"http://www.w3.org/2000/svg\"\n   xmlns:svg=\"http://www.w3.org/2000/svg\"\n   xmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\"\n   xmlns:cc=\"http://creativecommons.org/ns#\"\n   xmlns:dc=\"http://purl.org/dc/elements/1.1/\">\n  <defs\n     id=\"defs1\" />\n  <sodipodi:namedview\n     id=\"namedview1\"\n     pagecolor=\"#ffffff\"\n     bordercolor=\"#000000\"\n     borderopacity=\"0.25\"\n     inkscape:showpageshadow=\"2\"\n     inkscape:pageopacity=\"0.0\"\n     inkscape:pagecheckerboard=\"0\"\n     inkscape:deskcolor=\"#d1d1d1\"\n     inkscape:zoom=\"0.499375\"\n     inkscape:cx=\"-53.066333\"\n     inkscape:cy=\"163.20401\"\n     inkscape:window-width=\"1920\"\n     inkscape:window-height=\"1008\"\n     inkscape:window-x=\"0\"\n     inkscape:window-y=\"0\"\n     inkscape:window-maximized=\"1\"\n     inkscape:current-layer=\"svg1\" />\n  <title\n     id=\"title1\">triangle_fill</title>\n  <rect\n     style=\"fill:none;fill-opacity:1;fill-rule:evenodd;stroke:#000000;stroke-width:0;stroke-dasharray:none;stroke-opacity:0.403546\"\n     id=\"rect1\"\n     width=\"18\"\n     height=\"11.234042\"\n     x=\"0\"\n     y=\"0\" />\n  <g\n     id=\"Shape\"\n     transform=\"translate(-287.99968,-54.679037)\"\n     style=\"fill:none;fill-rule:evenodd;stroke:none;stroke-width:1\">\n    <g\n       id=\"triangle_fill\"\n       transform=\"translate(288,48)\">\n      <path\n         d=\"M 24,0 V 24 H 0 V 0 Z m -11.40651,23.257841 -0.01155,0.0017 -0.07106,0.03553 -0.019,0.0037 v 0 l -0.01516,-0.0037 -0.07106,-0.03553 c -0.0098,-0.0031 -0.01861,-4.89e-4 -0.02351,0.0054 l -0.0041,0.01092 -0.01709,0.427279 0.005,0.02039 0.01101,0.01222 0.103573,0.07398 0.01487,0.0039 v 0 l 0.01177,-0.0039 0.103575,-0.07398 0.0126,-0.01604 v 0 l 0.0034,-0.01656 -0.01709,-0.427279 c -0.002,-0.01013 -0.0085,-0.01653 -0.01607,-0.01799 z m 0.264901,-0.112555 -0.01384,0.002 -0.184705,0.09235 -0.01,0.01024 v 0 l -0.0027,0.01121 0.0179,0.429528 0.0048,0.01278 v 0 l 0.0085,0.0071 0.200954,0.09275 c 0.01209,0.0037 0.02289,-2.52e-4 0.02849,-0.008 l 0.004,-0.01406 -0.03415,-0.614631 c -0.0024,-0.01194 -0.01027,-0.01953 -0.01929,-0.02125 z m -0.715344,0.002 c -0.0098,-0.0049 -0.02087,-0.002 -0.02741,0.0053 l -0.0057,0.01394 -0.03415,0.614631 c -6.39e-4,0.0115 0.007,0.0207 0.01688,0.0234 l 0.01561,-0.0013 0.200955,-0.09275 0.0094,-0.0081 v 0 l 0.0039,-0.0118 0.0179,-0.429528 -0.0032,-0.01259 v 0 l -0.0095,-0.0089 z\"\n         id=\"MingCute\"\n         fill-rule=\"nonzero\" />\n    </g>\n  </g>\n  <path\n     d=\"m 7.9808726,1.6044182 c 0.4529693,-0.6776268 1.5853115,-0.67763372 2.0382024,-6.6e-6 l 6.77335,10.1335664 c 0.452888,0.677626 -0.113204,1.524658 -1.019062,1.524658 H 2.2266722 c -0.9058589,0 -1.47202265,-0.847032 -1.0190932,-1.524658 z\"\n     id=\"路径\"\n     fill=\"#09244b\"\n     style=\"fill:#ffffff;fill-rule:evenodd;stroke:#000000;stroke-width:2.13410997;stroke-linecap:square;stroke-linejoin:miter;stroke-dasharray:none;stroke-opacity:0.34;paint-order:stroke fill markers\" />\n  <metadata\n     id=\"metadata2\">\n    <rdf:RDF>\n      <cc:Work\n         rdf:about=\"\">\n        <dc:title>triangle_fill</dc:title>\n      </cc:Work>\n    </rdf:RDF>\n  </metadata>\n</svg>\n";
var root_1$5 = /* @__PURE__ */ from_html(`<div role="button" tabindex="0"><!></div>`);
var root_2$4 = /* @__PURE__ */ from_html(`<button type="button"> </button>`);
var root_3$2 = /* @__PURE__ */ from_html(`<div><div class="af9 afg"><img class="afa afg" alt=""/></div> <div><!></div></div>`);
var root$5 = /* @__PURE__ */ from_html(`<div><!> <!></div>`);
function ButtonLayer($$anchor, $$props) {
	push($$props, true);
	let buttonText = prop($$props, "buttonText", 3, "Open"), wrapperClass = prop($$props, "wrapperClass", 3, ""), buttonClass = prop($$props, "buttonClass", 3, ""), layerClass = prop($$props, "layerClass", 3, ""), horizontalOffset = prop($$props, "horizontalOffset", 3, 0), edgeMargin = prop($$props, "edgeMargin", 3, 10), isOpen = prop($$props, "isOpen", 15, false), defaultOpen = prop($$props, "defaultOpen", 3, false), contentCss = prop($$props, "contentCss", 3, "");
	if (defaultOpen()) isOpen(true);
	let buttonElement = /* @__PURE__ */ state(null);
	let layerElement = /* @__PURE__ */ state(null);
	let position = /* @__PURE__ */ state(proxy({
		top: 0,
		left: 0
	}));
	let angleLeft = /* @__PURE__ */ state(20);
	function toggleLayer() {
		isOpen(!isOpen());
		if (isOpen()) $$props.onOpen?.();
		else $$props.onClose?.();
	}
	function closeLayer() {
		if (isOpen()) {
			isOpen(false);
			$$props.onClose?.();
		}
	}
	async function updatePosition() {
		await tick$1();
		if (!get$1(buttonElement) || !get$1(layerElement)) return;
		const buttonRect = get$1(buttonElement).getBoundingClientRect();
		const layerRect = get$1(layerElement).getBoundingClientRect();
		const isMobile = window.innerWidth <= 748;
		const offset = 8;
		let top = buttonRect.bottom + offset;
		let left = isMobile ? 6 : buttonRect.left + horizontalOffset();
		if (!isMobile) {
			if (left + layerRect.width > window.innerWidth) left = window.innerWidth - layerRect.width - edgeMargin();
			if (left < edgeMargin()) left = edgeMargin();
		}
		if (top + layerRect.height > window.innerHeight) top = buttonRect.top - layerRect.height - offset;
		set$1(position, {
			top,
			left
		}, true);
		const buttonCenter = buttonRect.left + buttonRect.width / 2;
		if (isMobile) {
			set$1(angleLeft, buttonCenter - left - 12);
			const layerWidth = window.innerWidth - 12;
			const minAngleLeft = 10;
			const maxAngleLeft = layerWidth - 24;
			if (get$1(angleLeft) < minAngleLeft) set$1(angleLeft, minAngleLeft);
			if (get$1(angleLeft) > maxAngleLeft) set$1(angleLeft, maxAngleLeft);
		} else {
			set$1(angleLeft, buttonCenter - left - 12);
			const minAngleLeft = 10;
			const maxAngleLeft = layerRect.width - 24;
			if (get$1(angleLeft) < minAngleLeft) set$1(angleLeft, minAngleLeft);
			if (get$1(angleLeft) > maxAngleLeft) set$1(angleLeft, maxAngleLeft);
		}
	}
	user_effect(() => {
		if (isOpen() && get$1(buttonElement) && get$1(layerElement)) updatePosition();
	});
	user_effect(() => {
		if (!isOpen()) return;
		function handleClickOutside(event) {
			const target = event.target;
			if (get$1(buttonElement) && !get$1(buttonElement).contains(target) && get$1(layerElement) && !get$1(layerElement).contains(target)) closeLayer();
		}
		document.body.addEventListener("mousedown", handleClickOutside);
		return () => {
			document.body.removeEventListener("mousedown", handleClickOutside);
		};
	});
	user_effect(() => {
		if (!isOpen()) return;
		const handleUpdate = () => {
			if (isOpen() && get$1(buttonElement) && get$1(layerElement)) updatePosition();
		};
		window.addEventListener("scroll", handleUpdate, true);
		window.addEventListener("resize", handleUpdate);
		return () => {
			window.removeEventListener("scroll", handleUpdate, true);
			window.removeEventListener("resize", handleUpdate);
		};
	});
	var div = root$5();
	var node = child(div);
	var consequent = ($$anchor) => {
		var div_1 = root_1$5();
		div_1.__click = toggleLayer;
		div_1.__keydown = (e) => e.key === "Enter" && toggleLayer();
		snippet(child(div_1), () => $$props.button, isOpen);
		reset(div_1);
		bind_this(div_1, ($$value) => set$1(buttonElement, $$value), () => get$1(buttonElement));
		template_effect(() => set_class(div_1, 1, `button-layer-trigger-wrapper ${buttonClass() ?? ""}`, "afg"));
		append($$anchor, div_1);
	};
	var alternate = ($$anchor) => {
		var button_1 = root_2$4();
		button_1.__click = toggleLayer;
		var text = child(button_1, true);
		reset(button_1);
		bind_this(button_1, ($$value) => set$1(buttonElement, $$value), () => get$1(buttonElement));
		template_effect(() => {
			set_class(button_1, 1, `af5 ${buttonClass() ?? ""}`, "afg");
			set_text(text, buttonText());
		});
		append($$anchor, button_1);
	};
	if_block(node, ($$render) => {
		if ($$props.button) $$render(consequent);
		else $$render(alternate, false);
	});
	var node_2 = sibling(node, 2);
	var consequent_1 = ($$anchor) => {
		var div_2 = root_3$2();
		let classes;
		var div_3 = child(div_2);
		var img = child(div_3);
		reset(div_3);
		var div_4 = sibling(div_3, 2);
		snippet(child(div_4), () => $$props.children ?? noop$1);
		reset(div_4);
		reset(div_2);
		bind_this(div_2, ($$value) => set$1(layerElement, $$value), () => get$1(layerElement));
		template_effect(($0) => {
			classes = set_class(div_2, 1, `af6 min-w-200 ${(layerClass() || "w-[calc(100vw-12px)]") ?? ""}`, "afg", classes, { af7: $$props.useBig });
			set_style(div_2, `top: ${get$1(position).top ?? ""}px; left: ${get$1(position).left ?? ""}px;`);
			set_style(div_3, `left: ${get$1(angleLeft) ?? ""}px;`);
			set_attribute(img, "src", $0);
			set_class(div_4, 1, `af8 ${contentCss() ?? ""}`, "afg");
		}, [() => parseSVG(angle_default)]);
		append($$anchor, div_2);
	};
	if_block(node_2, ($$render) => {
		if (isOpen()) $$render(consequent_1);
	});
	reset(div);
	template_effect(() => set_class(div, 1, `af4 ${wrapperClass() ?? ""}`, "afg"));
	append($$anchor, div);
	pop();
}
delegate(["click", "keydown"]);
var root$4 = /* @__PURE__ */ from_html(`<div id="culqi-container" class="grow-1 w-full min-h-[500px]"><div class="flex flex-col items-center justify-center h-full text-gray-400"><i class="icon-spin5 animate-spin text-[32px] mb-2"></i> <p>Cargando pasarela de pago...</p></div></div>`);
function CulqiCheckout_1($$anchor, $$props) {
	push($$props, true);
	let culqiInitialized = false;
	async function loadCulqi() {
		if (window.CulqiCheckout) return window.CulqiCheckout;
		return new Promise((resolve) => {
			const script = document.createElement("script");
			script.src = "https://js.culqi.com/checkout-js";
			script.async = true;
			script.onload = () => {
				resolve(window.CulqiCheckout);
			};
			document.body.appendChild(script);
		});
	}
	async function initCulqi() {
		if (culqiInitialized) return;
		if ($$props.amount <= 0) {
			console.warn("Amount must be greater than 0 to initialize Culqi");
			return;
		}
		const CulqiCheckout = await loadCulqi();
		const settings = {
			title: Env.empresa.Nombre || "Genix Store",
			currency: "PEN",
			amount: Math.round($$props.amount)
		};
		if ($$props.order) settings.order = $$props.order;
		const client = { email: $$props.email || "" };
		const paymentMethods = {
			tarjeta: true,
			yape: !!$$props.order,
			billetera: !!$$props.order,
			bancaMovil: !!$$props.order,
			agente: !!$$props.order,
			cuotealo: !!$$props.order
		};
		const config = {
			settings,
			client,
			options: {
				lang: "auto",
				modal: false,
				container: "#culqi-container",
				paymentMethods,
				paymentMethodsSort: Object.keys(paymentMethods)
			},
			appearance: {
				theme: "default",
				menuType: "sidebar",
				hiddenCulqiLogo: true,
				hiddenBannerContent: true,
				hiddenBanner: false,
				hiddenToolBarAmount: false,
				buttonCardPayText: "Pagar S/ " + ($$props.amount / 100).toFixed(2)
			}
		};
		const Culqi = new CulqiCheckout(Env.empresa.CulqiLlave || "pk_test_...", config);
		window.culqi = () => {
			if (Culqi.token) {
				$$props.onSuccess?.(Culqi.token.id, "token");
				Culqi.close();
			} else if (Culqi.order) {
				$$props.onSuccess?.(Culqi.order, "order");
				Culqi.close();
			} else {
				console.error("Culqi Error:", Culqi.error);
				if ($$props.onError) $$props.onError(Culqi.error);
				else alert("Error en el pago: " + (Culqi.error?.user_message || "Desconocido"));
			}
		};
		Culqi.open();
		culqiInitialized = true;
	}
	onMount$1(() => {
		setTimeout(initCulqi, 100);
	});
	append($$anchor, root$4());
	pop();
}
var root_3$1 = /* @__PURE__ */ from_html(`<i></i>`);
var root_2$3 = /* @__PURE__ */ from_html(`<div class="flex items-center mt-1 ff-semibold"><!> <div> </div></div>`);
var root_4$1 = /* @__PURE__ */ from_html(`<div class="mt-8 mb-8 fs18 ff-bold">Total a pagar:</div> <div class="grid grid-cols-1 md:grid-cols-2 gap-12 pb-6"></div>`, 1);
var root_6$1 = /* @__PURE__ */ from_html(`<div class="mt-8 mb-8 fs18 ff-bold">Total a pagar:</div> <div class="grid grid-cols-12 gap-12"><!> <!> <!> <!> <!> <!></div>`, 1);
var root_7 = /* @__PURE__ */ from_html(`<div class="flex flex-col h-full"><div class="fs18 ff-bold mb-4 py-8"> </div> <!> <div class="py-4 text-center text-gray-500 text-sm">Pago seguro procesado por Culqi</div></div>`);
var root_8$1 = /* @__PURE__ */ from_html(`<div class="mt-12 text-center"><i class="icon-ok text-green-500 text-[64px] mb-4"></i> <div class="fs24 ff-bold mb-2">¡Gracias por tu compra!</div> <p class="text-gray-600 mb-8">Tu pedido ha sido procesado exitosamente. Recibirás un correo de
						confirmación a la brevedad.</p> <button class="bg-gray-800 text-white px-8 py-3 rounded-lg ff-bold hover:bg-black transition-all cursor-pointer">Volver a la tienda</button></div>`);
var root_1$4 = /* @__PURE__ */ from_html(`<div class="p-4 md:p-12 flex flex-col h-[calc(100vh-120px)]"><!> <div class="w-full px-4 overflow-auto grow"><!> <!> <!> <!></div></div>`);
var root_10$1 = /* @__PURE__ */ from_html(`<button><i class="icon1-basket"></i> <span>Carrito</span></button>`);
var root_9$1 = /* @__PURE__ */ from_html(`<div><!></div>`);
var root_13$1 = /* @__PURE__ */ from_html(`<div><img alt=""/> <div class="_2 absolute p-12 flex flex-col afe"><!></div></div>`);
function CartMenu($$anchor, $$props) {
	push($$props, true);
	const cartContent = ($$anchor, isMobileVersion = noop$1) => {
		var div = root_1$4();
		var node = child(div);
		{
			const optionRender = ($$anchor, e = noop$1) => {
				var div_1 = root_2$3();
				var node_1 = child(div_1);
				var consequent = ($$anchor) => {
					var i = root_3$1();
					template_effect(() => set_class(i, 1, `text-[18px] ${e().icon} mr-2`, "afe"));
					append($$anchor, i);
				};
				if_block(node_1, ($$render) => {
					if (!isMobileVersion()) $$render(consequent);
				});
				var div_2 = sibling(node_1, 2);
				var text = child(div_2, true);
				reset(div_2);
				reset(div_1);
				template_effect(($0) => {
					set_class(div_2, 1, $0, "afe");
					set_text(text, e().name);
				}, [() => clsx(["lh-11", !isMobileVersion() ? "text-left mr-6" : "text-center"].join(" "))]);
				append($$anchor, div_1);
			};
			let $0 = /* @__PURE__ */ user_derived(() => !isMobileVersion() && Core.deviceType === 3 ? "1fr 1fr 1fr 0.7fr" : "");
			ArrowSteps(node, {
				get selected() {
					return Ecommerce.cartOption;
				},
				get columnsTemplate() {
					return get$1($0);
				},
				onSelect: (e) => {
					Ecommerce.cartOption = e.id;
				},
				options: [
					{
						id: 1,
						name: "Carrito",
						icon: "icon-basket"
					},
					{
						id: 2,
						name: "Datos Envío",
						icon: "icon-doc-inv-alt"
					},
					{
						id: 3,
						name: "Pago",
						icon: "icon-shield"
					},
					{
						id: 4,
						name: "Confirma ción",
						icon: "icon-ok"
					}
				],
				optionRender,
				$$slots: { optionRender: true }
			});
		}
		var div_3 = sibling(node, 2);
		var node_2 = child(div_3);
		var consequent_1 = ($$anchor) => {
			var fragment = root_4$1();
			var div_4 = sibling(first_child(fragment), 2);
			each(div_4, 21, () => ProductsSelectedMap.values(), index, ($$anchor, cartProducto) => {
				ProductCardHorizonal($$anchor, { get producto() {
					return get$1(cartProducto).producto;
				} });
			});
			reset(div_4);
			append($$anchor, fragment);
		};
		if_block(node_2, ($$render) => {
			if (Ecommerce.cartOption === 1) $$render(consequent_1);
		});
		var node_3 = sibling(node_2, 2);
		var consequent_2 = ($$anchor) => {
			var fragment_2 = root_6$1();
			var div_5 = sibling(first_child(fragment_2), 2);
			var node_4 = child(div_5);
			Input(node_4, {
				label: "Nombres",
				css: "col-span-6",
				get saveOn() {
					return userForm;
				},
				save: "nombres",
				required: true
			});
			var node_5 = sibling(node_4, 2);
			Input(node_5, {
				label: "Apellidos",
				css: "col-span-6",
				get saveOn() {
					return userForm;
				},
				save: "apellidos",
				required: true
			});
			var node_6 = sibling(node_5, 2);
			Input(node_6, {
				label: "Correo Electrónico",
				css: "col-span-6",
				get saveOn() {
					return userForm;
				},
				save: "email",
				required: true
			});
			var node_7 = sibling(node_6, 2);
			CiudadesSelector(node_7, {
				get saveOn() {
					return userForm;
				},
				save: "ciudadID",
				css: "col-span-6"
			});
			var node_8 = sibling(node_7, 2);
			Input(node_8, {
				label: "Dirección",
				css: "col-span-6",
				get saveOn() {
					return userForm;
				},
				save: "direccion",
				required: true
			});
			Input(sibling(node_8, 2), {
				label: "Referencia",
				css: "col-span-6",
				get saveOn() {
					return userForm;
				},
				save: "referencia"
			});
			reset(div_5);
			append($$anchor, fragment_2);
		};
		if_block(node_3, ($$render) => {
			if (Ecommerce.cartOption === 2) $$render(consequent_2);
		});
		var node_10 = sibling(node_3, 2);
		var consequent_3 = ($$anchor) => {
			var div_6 = root_7();
			var div_7 = child(div_6);
			var text_1 = child(div_7);
			reset(div_7);
			var node_11 = sibling(div_7, 2);
			{
				let $0 = /* @__PURE__ */ user_derived(() => get$1(total) * 100);
				CulqiCheckout_1(node_11, {
					get amount() {
						return get$1($0);
					},
					get email() {
						return userForm.email;
					},
					onSuccess: () => {
						Ecommerce.cartOption = 4;
					}
				});
			}
			next(2);
			reset(div_6);
			template_effect(($0) => set_text(text_1, `Total a pagar: S/ ${$0 ?? ""}`), [() => get$1(total).toFixed(2)]);
			append($$anchor, div_6);
		};
		if_block(node_10, ($$render) => {
			if (Ecommerce.cartOption === 3) $$render(consequent_3);
		});
		var node_12 = sibling(node_10, 2);
		var consequent_4 = ($$anchor) => {
			var div_8 = root_8$1();
			var button_1 = sibling(child(div_8), 6);
			button_1.__click = () => {
				ProductsSelectedMap.clear();
				Ecommerce.cartOption = 1;
				layerOpenedState.id = 0;
			};
			reset(div_8);
			append($$anchor, div_8);
		};
		if_block(node_12, ($$render) => {
			if (Ecommerce.cartOption === 4) $$render(consequent_4);
		});
		reset(div_3);
		reset(div);
		append($$anchor, div);
	};
	const id = prop($$props, "id", 3, 0), isMobile = prop($$props, "isMobile", 3, false), css = prop($$props, "css", 3, "");
	let userForm = {};
	let isOpen = /* @__PURE__ */ state(false);
	const total = /* @__PURE__ */ user_derived(() => Array.from(ProductsSelectedMap.values()).reduce((acc, curr) => {
		return acc + (curr.producto.PrecioFinal || curr.producto.Precio || 0) * curr.cant;
	}, 0));
	user_effect(() => {
		if (layerOpenedState.id === id()) {
			set$1(isOpen, true);
			Env.loadEmpresaConfig();
		} else set$1(isOpen, false);
	});
	user_effect(() => {
		if (get$1(isOpen)) {
			if (layerOpenedState.id !== id()) layerOpenedState.id = id();
		} else if (layerOpenedState.id === id()) layerOpenedState.id = 0;
	});
	var fragment_3 = comment();
	var node_13 = first_child(fragment_3);
	var consequent_5 = ($$anchor) => {
		var div_9 = root_9$1();
		var node_14 = child(div_9);
		{
			const button = ($$anchor, open = noop$1) => {
				var button_2 = root_10$1();
				template_effect(($0) => set_class(button_2, 1, $0, "afe"), [() => clsx(["bn1 w-full", open() ? styles_module_default$1.button_menu_top : ""].join(" "))]);
				append($$anchor, button_2);
			};
			ButtonLayer(node_14, {
				wrapperClass: "w-full h-full",
				buttonClass: "w-full h-full",
				horizontalOffset: -200,
				edgeMargin: 32,
				layerClass: "w-768! max-w-[82vw]! rounded-[11px]!",
				useBig: true,
				get isOpen() {
					return get$1(isOpen);
				},
				set isOpen($$value) {
					set$1(isOpen, $$value, true);
				},
				button,
				children: ($$anchor, $$slotProps) => {
					cartContent($$anchor, () => false);
				},
				$$slots: {
					button: true,
					default: true
				}
			});
		}
		reset(div_9);
		template_effect(() => set_class(div_9, 1, clsx(css()), "afe"));
		append($$anchor, div_9);
	};
	var alternate = ($$anchor) => {
		var fragment_5 = comment();
		var node_15 = first_child(fragment_5);
		var consequent_6 = ($$anchor) => {
			var div_10 = root_13$1();
			var img = child(div_10);
			set_class(img, 1, "absolute h-20 _1 right-17 afe");
			var div_11 = sibling(img, 2);
			cartContent(child(div_11), () => true);
			reset(div_11);
			reset(div_10);
			template_effect(($0) => {
				set_class(div_10, 1, clsx(css()), "afe");
				set_attribute(img, "src", $0);
			}, [() => parseSVG(angle_default$1)]);
			append($$anchor, div_10);
		};
		if_block(node_15, ($$render) => {
			if (layerOpenedState.id === id()) $$render(consequent_6);
		}, true);
		append($$anchor, fragment_5);
	};
	if_block(node_13, ($$render) => {
		if (!isMobile()) $$render(consequent_5);
		else $$render(alternate, false);
	});
	append($$anchor, fragment_3);
	pop();
}
delegate(["click"]);
var root_2$2 = /* @__PURE__ */ from_html(`<span> </span>`);
var root_3 = /* @__PURE__ */ from_html(`<span> <br/> </span>`);
var root_1$3 = /* @__PURE__ */ from_html(`<button><!></button>`);
var root$3 = /* @__PURE__ */ from_html(`<div></div>`);
function OptionsStrip($$anchor, $$props) {
	push($$props, true);
	const getClass = (e) => {
		let cn = "";
		if ((Array.isArray(e) ? e[0] : ($$props.keyId ? e?.[$$props.keyId] : 0) || 0) === $$props.selected) cn += " _3";
		if ($$props.buttonCss) cn += " " + $$props.buttonCss;
		return cn;
	};
	const getValue = (e) => {
		if (Array.isArray(e)) {
			if (Core.deviceType === 3 && Array.isArray(e[2])) return e[2];
			return [e[1]];
		} else if ($$props.keyName) return [e[$$props.keyName]];
		else return [""];
	};
	var div = root$3();
	let classes;
	each(div, 21, () => $$props.options, index, ($$anchor, opt) => {
		const words = /* @__PURE__ */ user_derived(() => getValue(get$1(opt)));
		var button = root_1$3();
		button.__click = (ev) => {
			ev.stopPropagation();
			$$props.onSelect(get$1(opt));
		};
		var node = child(button);
		var consequent = ($$anchor) => {
			var span = root_2$2();
			var text = child(span, true);
			reset(span);
			template_effect(() => set_text(text, get$1(words)[0]));
			append($$anchor, span);
		};
		var alternate = ($$anchor) => {
			var span_1 = root_3();
			var text_1 = child(span_1, true);
			var text_2 = sibling(text_1, 2, true);
			reset(span_1);
			template_effect(() => {
				set_text(text_1, get$1(words)[0]);
				set_text(text_2, get$1(words)[1]);
			});
			append($$anchor, span_1);
		};
		if_block(node, ($$render) => {
			if (get$1(words).length === 1) $$render(consequent);
			else $$render(alternate, false);
		});
		reset(button);
		template_effect(($0) => set_class(button, 1, `flex items-center ff-bold _2 ${$0 ?? ""}`, "afk"), [() => getClass(get$1(opt))]);
		append($$anchor, button);
	});
	reset(div);
	template_effect(() => classes = set_class(div, 1, `_1 pb-4 md:pb-0 flex items-center shrink-0 max-w-[100%] overflow-x-auto overflow-y-hidden ${$$props.css ?? ""}`, "afk", classes, {
		_5: $$props.useMobileGrid,
		"grid-cols-2": $$props.useMobileGrid && $$props.options.length === 2,
		"grid-cols-3": $$props.useMobileGrid && $$props.options.length === 3,
		"grid-cols-4": $$props.useMobileGrid && $$props.options.length === 4,
		"grid-cols-5": $$props.useMobileGrid && $$props.options.length === 5
	}));
	append($$anchor, div);
	pop();
}
delegate(["click"]);
var root_2$1 = /* @__PURE__ */ from_html(`<div class="fs18 ff-bold mb-12">Detalles de la Cuenta</div> <div class="flex flex-col gap-6"><div class="flex flex-col gap-1"><span class="text-gray-500 text-sm">Nombre Completo</span> <span class="ff-semibold"> </span></div> <div class="flex flex-col gap-1"><span class="text-gray-500 text-sm">Correo Electrónico</span> <span class="ff-semibold"> </span></div> <button class="bg-[#4c55d5] text-white py-2.5 px-6 rounded-lg ff-bold hover:bg-[#3b44b8] transition-all w-max mt-4">Cerrar Sesión</button></div>`, 1);
var root_4 = /* @__PURE__ */ from_html(`<div class="fs18 ff-bold mb-12">Historial de Pedidos</div> <div class="flex flex-col items-center justify-center py-12 text-gray-400"><i class="icon-basket text-[48px] mb-4 opacity-20"></i> <p>Aún no has realizado ningún pedido.</p></div>`, 1);
var root_6 = /* @__PURE__ */ from_html(`<div class="fs18 ff-bold mb-12">Productos Favoritos</div> <div class="flex flex-col items-center justify-center py-12 text-gray-400"><i class="icon-heart text-[48px] mb-4 opacity-20"></i> <p>No tienes productos en tu lista de deseos.</p></div>`, 1);
var root_8 = /* @__PURE__ */ from_html(`<div class="fs18 ff-bold mb-12">Configuración del Sitio</div> <div class="flex flex-col gap-4"><button class="flex items-center gap-3 p-3 rounded-lg hover:bg-gray-50 transition-colors text-left w-full"><i class="icon-lock"></i> <span>Cambiar Contraseña</span></button> <button class="flex items-center gap-3 p-3 rounded-lg hover:bg-gray-50 transition-colors text-left w-full"><i class="icon-bell"></i> <span>Gestionar Notificaciones</span></button> <button class="flex items-center gap-3 p-3 rounded-lg hover:bg-gray-50 transition-colors text-left w-full"><i class="icon-cog"></i> <span>Preferencias de Idioma</span></button></div>`, 1);
var root_1$2 = /* @__PURE__ */ from_html(`<div class="p-16 flex flex-col min-h-300"><!> <div class="grow"><!></div></div>`);
var root_10 = /* @__PURE__ */ from_html(`<button><i class="icon1-user"></i> <span>Mi Usuario</span></button>`);
var root_9 = /* @__PURE__ */ from_html(`<div class="hidden md:block w-120 relative h-full"><!></div>`);
var root_13 = /* @__PURE__ */ from_html(`<div><div class="bg-white rounded-[11px] shadow-lg overflow-hidden"><!></div></div>`);
function UsuarioMenu($$anchor, $$props) {
	push($$props, true);
	const menuContent = ($$anchor) => {
		var div = root_1$2();
		var node = child(div);
		OptionsStrip(node, {
			get options() {
				return options;
			},
			get selected() {
				return get$1(selectedTab);
			},
			keyId: "id",
			keyName: "name",
			onSelect: (opt) => set$1(selectedTab, opt.id, true),
			css: "mb-12"
		});
		var div_1 = sibling(node, 2);
		var node_1 = child(div_1);
		var consequent = ($$anchor) => {
			var fragment = root_2$1();
			var div_2 = sibling(first_child(fragment), 2);
			var div_3 = child(div_2);
			var span = sibling(child(div_3), 2);
			var text = child(span, true);
			reset(span);
			reset(div_3);
			var div_4 = sibling(div_3, 2);
			var span_1 = sibling(child(div_4), 2);
			var text_1 = child(span_1, true);
			reset(span_1);
			reset(div_4);
			var button_1 = sibling(div_4, 2);
			button_1.__click = handleLogout;
			reset(div_2);
			template_effect(() => {
				set_text(text, get$1(userInfo)?.Nombre || get$1(userInfo)?.Usuario || "Usuario Invitado");
				set_text(text_1, get$1(userInfo)?.Email || "No especificado");
			});
			append($$anchor, fragment);
		};
		var alternate_2 = ($$anchor) => {
			var fragment_1 = comment();
			var node_2 = first_child(fragment_1);
			var consequent_1 = ($$anchor) => {
				var fragment_2 = root_4();
				next(2);
				append($$anchor, fragment_2);
			};
			var alternate_1 = ($$anchor) => {
				var fragment_3 = comment();
				var node_3 = first_child(fragment_3);
				var consequent_2 = ($$anchor) => {
					var fragment_4 = root_6();
					next(2);
					append($$anchor, fragment_4);
				};
				var alternate = ($$anchor) => {
					var fragment_5 = comment();
					var node_4 = first_child(fragment_5);
					var consequent_3 = ($$anchor) => {
						var fragment_6 = root_8();
						next(2);
						append($$anchor, fragment_6);
					};
					if_block(node_4, ($$render) => {
						if (get$1(selectedTab) === 4) $$render(consequent_3);
					}, true);
					append($$anchor, fragment_5);
				};
				if_block(node_3, ($$render) => {
					if (get$1(selectedTab) === 3) $$render(consequent_2);
					else $$render(alternate, false);
				}, true);
				append($$anchor, fragment_3);
			};
			if_block(node_2, ($$render) => {
				if (get$1(selectedTab) === 2) $$render(consequent_1);
				else $$render(alternate_1, false);
			}, true);
			append($$anchor, fragment_1);
		};
		if_block(node_1, ($$render) => {
			if (get$1(selectedTab) === 1) $$render(consequent);
			else $$render(alternate_2, false);
		});
		reset(div_1);
		reset(div);
		append($$anchor, div);
	};
	const id = prop($$props, "id", 3, 3), isMobile = prop($$props, "isMobile", 3, false), css = prop($$props, "css", 3, "");
	let isOpen = /* @__PURE__ */ state(false);
	let selectedTab = /* @__PURE__ */ state(1);
	const userInfo = /* @__PURE__ */ user_derived(() => accessHelper.getUserInfo());
	const options = [
		{
			id: 1,
			name: "Mi Cuenta"
		},
		{
			id: 2,
			name: "Mis Pedidos"
		},
		{
			id: 3,
			name: "Mis Favoritos"
		},
		{
			id: 4,
			name: "Config"
		}
	];
	user_effect(() => {
		if (layerOpenedState.id === id()) set$1(isOpen, true);
		else set$1(isOpen, false);
	});
	user_effect(() => {
		if (get$1(isOpen)) {
			if (layerOpenedState.id !== id()) layerOpenedState.id = id();
		} else if (layerOpenedState.id === id()) layerOpenedState.id = 0;
	});
	function handleLogout() {
		Env.clearAccesos?.();
		layerOpenedState.id = 0;
	}
	var fragment_7 = comment();
	var node_5 = first_child(fragment_7);
	var consequent_4 = ($$anchor) => {
		var div_5 = root_9();
		var node_6 = child(div_5);
		{
			const button = ($$anchor, open = noop$1) => {
				var button_2 = root_10();
				template_effect(($0) => set_class(button_2, 1, $0), [() => clsx(["bn1 w-full", open() ? styles_module_default$1.button_menu_top : ""].join(" "))]);
				append($$anchor, button_2);
			};
			ButtonLayer(node_6, {
				wrapperClass: "w-full h-full",
				buttonClass: "w-full h-full",
				edgeMargin: 32,
				useBig: true,
				layerClass: "w-700 max-w-[90vw] rounded-[11px]",
				get isOpen() {
					return get$1(isOpen);
				},
				set isOpen($$value) {
					set$1(isOpen, $$value, true);
				},
				button,
				children: ($$anchor, $$slotProps) => {
					menuContent($$anchor);
				},
				$$slots: {
					button: true,
					default: true
				}
			});
		}
		reset(div_5);
		append($$anchor, div_5);
	};
	var alternate_3 = ($$anchor) => {
		var fragment_9 = comment();
		var node_7 = first_child(fragment_9);
		var consequent_5 = ($$anchor) => {
			var div_6 = root_13();
			var div_7 = child(div_6);
			menuContent(child(div_7));
			reset(div_7);
			reset(div_6);
			template_effect(() => set_class(div_6, 1, clsx(css())));
			append($$anchor, div_6);
		};
		if_block(node_7, ($$render) => {
			if (layerOpenedState.id === id()) $$render(consequent_5);
		}, true);
		append($$anchor, fragment_9);
	};
	if_block(node_5, ($$render) => {
		if (!isMobile()) $$render(consequent_4);
		else $$render(alternate_3, false);
	});
	append($$anchor, fragment_7);
	pop();
}
delegate(["click"]);
var root_1$1 = /* @__PURE__ */ from_html(`<div class="_4 fs14 fx-c w-22 h-22 mb-14 ml-26 absolute rounded-[50%] aeV"> </div>`);
var root$2 = /* @__PURE__ */ from_html(`<div class="_2 flex justify-between text-white h-48 aeV"></div> <div id="sh-0" class="flex justify-between w-full h-68">header</div> <div id="sh-1" class="_1 h-58 w-full md:h-68 top-48 absolute flex items-center md:justify-between w-full left-0 px-4 md:px-80"><div class="hidden md:block"></div> <button aria-label="page-menu" class="_6 fx-c w-[14vw] md:hidden! aeV"><i class="text-[22px] icon-menu"></i></button> <!> <div class="flex items-center h-42 ml-auto md:ml-0"><!> <!> <div><button class="_3 absolute w-full fx-c aeV"><i></i> <!></button> <button aria-label="close_layer" class="_5 absolute fx-c w-40 h-40 rounded-[50%] aeV"><i class="icon-cancel"></i></button></div></div> <!></div>`, 1);
function Header($$anchor, $$props) {
	push($$props, true);
	const cartCant = /* @__PURE__ */ user_derived(() => ProductsSelectedMap.size);
	onMount$1(() => {});
	var fragment = root$2();
	var div = sibling(first_child(fragment), 4);
	var button = sibling(child(div), 2);
	button.__click = (ev) => {
		ev.stopPropagation();
		if (document.startViewTransition) document.startViewTransition(() => {
			Core.mobileMenuOpen = 1;
		});
	};
	var node = sibling(button, 2);
	SearchBar(node, {});
	var div_1 = sibling(node, 2);
	var node_1 = child(div_1);
	CartMenu(node_1, {
		css: "mr-8 hidden md:block relative w-120 h-full",
		id: 1
	});
	var node_2 = sibling(node_1, 2);
	UsuarioMenu(node_2, {});
	var div_2 = sibling(node_2, 2);
	var button_1 = child(div_2);
	button_1.__click = (ev) => {
		ev.stopPropagation();
		layerOpenedState.id = layerOpenedState.id ? 0 : 2;
	};
	var i = child(button_1);
	var node_3 = sibling(i, 2);
	var consequent = ($$anchor) => {
		var div_3 = root_1$1();
		var text = child(div_3, true);
		reset(div_3);
		template_effect(() => set_text(text, get$1(cartCant)));
		append($$anchor, div_3);
	};
	if_block(node_3, ($$render) => {
		if (get$1(cartCant) > 0) $$render(consequent);
	});
	reset(button_1);
	var button_2 = sibling(button_1, 2);
	button_2.__click = (ev) => {
		ev.stopPropagation();
		layerOpenedState.id = layerOpenedState.id ? 0 : 2;
	};
	reset(div_2);
	reset(div_1);
	CartMenu(sibling(div_1, 2), {
		isMobile: true,
		id: 2,
		css: "absolute w-full top-[100%] left-0"
	});
	reset(div);
	template_effect(() => {
		set_class(div_2, 1, "relative w-[15vw] fx-c flex md:hidden! " + (layerOpenedState.id === 2 ? "s1" : ""), "aeV");
		set_class(i, 1, "icon1-basket mt-4 " + (get$1(cartCant) > 0 ? "mb-[-2px] mr-2" : "mb-2"), "aeV");
	});
	append($$anchor, fragment);
	pop();
}
delegate(["click"]);
var root_1 = /* @__PURE__ */ from_html(`Agregar <i class="icon1-basket"></i>`, 1);
var root_2 = /* @__PURE__ */ from_html(` <i class="icon1-basket"></i>`, 1);
var root$1 = /* @__PURE__ */ from_html(`<div class="relative _8 aff"><div><!> <div class="_3 pb-2 aff"><div class="_5 mt-6 mb-4 min-h-26 md:min-h-32 fx-c aff"> </div> <div class="px-4 ff-bold fs17"> </div> <div class="_6 fx-c h-30 w-32 aff"><i class="icon1-basket"></i></div></div> <div class="_2 aff"><!> <!></div></div></div>`);
function ProductCard($$anchor, $$props) {
	push($$props, true);
	const css = prop($$props, "css", 3, "");
	const prodCant = /* @__PURE__ */ user_derived(() => {
		return ProductsSelectedMap.get($$props.producto.ID)?.cant || 0;
	});
	proxy({});
	user_effect(() => {});
	var div = root$1();
	var div_1 = child(div);
	var node = child(div_1);
	{
		let $0 = /* @__PURE__ */ user_derived(() => $$props.producto.Image?.n);
		Imagehash(node, {
			css: "w-full h-[36vw] md:h-200",
			get src() {
				return get$1($0);
			},
			folder: "img-productos"
		});
	}
	var div_2 = sibling(node, 2);
	var div_3 = child(div_2);
	var text = child(div_3, true);
	reset(div_3);
	var div_4 = sibling(div_3, 2);
	var text_1 = child(div_4);
	reset(div_4);
	next(2);
	reset(div_2);
	var div_5 = sibling(div_2, 2);
	div_5.__click = (ev) => {
		ev.stopPropagation();
		console.log("hola");
		ProductsSelectedMap.set($$props.producto.ID, {
			cant: get$1(prodCant) + 1,
			producto: $$props.producto
		});
		console.log("ProductsSelectedMap", ProductsSelectedMap);
	};
	var node_1 = child(div_5);
	var consequent = ($$anchor) => {
		var fragment = root_1();
		next();
		append($$anchor, fragment);
	};
	if_block(node_1, ($$render) => {
		if (get$1(prodCant) === 0) $$render(consequent);
	});
	var node_2 = sibling(node_1, 2);
	var consequent_1 = ($$anchor) => {
		var fragment_1 = root_2();
		var text_2 = first_child(fragment_1);
		next();
		template_effect(() => set_text(text_2, `Agregar más (${get$1(prodCant) ?? ""}) `));
		append($$anchor, fragment_1);
	};
	if_block(node_2, ($$render) => {
		if (get$1(prodCant) > 0) $$render(consequent_1);
	});
	reset(div_5);
	reset(div_1);
	reset(div);
	template_effect(($0, $1) => {
		set_class(div_1, 1, $0, "aff");
		set_text(text, $$props.producto.Nombre);
		set_text(text_1, `s/. ${$1 ?? ""}`);
	}, [() => clsx(["_1", css()].join(" ")), () => formatN($$props.producto.PrecioFinal / 100, 2)]);
	append($$anchor, div);
	pop();
}
delegate(["click"]);
var root = /* @__PURE__ */ from_html(`<div class="w-full flex justify-center overflow-x-hidden pt-2"><div></div></div>`);
function ProductCards($$anchor, $$props) {
	push($$props, true);
	const productos = productosServiceState;
	user_effect(() => {});
	user_effect(() => {
		console.log("=== STATE CHANGE ===");
		console.log("categorias:", productos.categorias);
		console.log("productos length:", productos.productos.length);
		console.log("productos data:", productos.productos);
	});
	var div = root();
	var div_1 = child(div);
	each(div_1, 21, () => productos.productos, index, ($$anchor, producto) => {
		ProductCard($$anchor, {
			css: "w-full md:w-240",
			get producto() {
				return get$1(producto);
			}
		});
	});
	reset(div_1);
	reset(div);
	template_effect(() => set_class(div_1, 1, "grid grid-cols-2 gap-x-12 md:gap-x-20 md:flex md:flex-wrap md:justify-center max-w-1680 w100-p12 p-8 md:p-0 " + styles_module_default$1.product_cards_ctn));
	append($$anchor, div);
	pop();
}
export { set_text as A, user_pre_effect as B, component as C, each as D, html as E, delegate as F, set$1 as G, first_child as H, get$1 as I, pop as J, state as K, tick$1 as L, comment as M, from_html as N, index as O, text as P, reset as Q, template_effect as R, head as S, slot as T, sibling as U, child as V, proxy as W, setContext as X, push as Y, next as Z, prop as _, Error$1 as a, set_class as b, Core as c, browser as d, start as f, asClassComponent as g, onMount$1 as h, Input as i, append as j, if_block as k, mainMenuOptions as l, load_css as m, Header as n, getProductos as o, __vitePreload as p, user_derived as q, MainCarrusel as r, productosServiceState as s, ProductCards as t, suscribeUrlFlag as u, bind_this as v, snippet as w, clsx as x, set_attribute as y, user_effect as z };
