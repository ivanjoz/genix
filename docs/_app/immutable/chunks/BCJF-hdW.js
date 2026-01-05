import { $ as proxy, A as if_block, B as tick, C as clsx, D as html, E as snippet, F as text, G as user_effect, L as delegate, M as append, N as comment, O as each, P as from_html, Q as sibling, R as event, S as set_class, V as untrack, W as template_effect, X as child, Z as first_child, _ as remove_input_defaults, _t as reset, a as prop, at as state, f as bind_this, g as attribute_effect, h as STYLE, j as set_text, k as index, lt as pop, ot as update, r as onMount, rt as set, st as user_derived, ut as push, v as set_attribute, vt as noop, w as action, x as set_style, y as set_value, z as get } from "./CTB4HzdN.js";
import { n as true_default } from "./D9PGjZ6Y.js";
import { n as WeakSearchRef, t as Core } from "./BwrZ3UQO.js";
import { a as throttle, n as highlString, r as include } from "./DwKmqXYU.js";
import { t as Popover2 } from "./DnSz4jvr.js";
function createVirtualizer(options) {
	let scrollElement = null;
	let scrollOffset = 0;
	let measurements = /* @__PURE__ */ new Map();
	let rafId = null;
	let subscribers = /* @__PURE__ */ new Set();
	let isInitialized = false;
	let resizeObserver = null;
	function init() {
		if (isInitialized) return;
		scrollElement = options.getScrollElement();
		if (!scrollElement) return;
		isInitialized = true;
		scrollOffset = scrollElement.scrollTop;
		const handleScroll = () => {
			if (rafId !== null) cancelAnimationFrame(rafId);
			rafId = requestAnimationFrame(() => {
				if (scrollElement) {
					scrollOffset = scrollElement.scrollTop;
					notifySubscribers();
				}
				rafId = null;
			});
		};
		scrollElement.addEventListener("scroll", handleScroll, { passive: true });
		resizeObserver = new ResizeObserver(() => {
			notifySubscribers();
		});
		resizeObserver.observe(scrollElement);
	}
	function notifySubscribers() {
		subscribers.forEach((callback) => callback());
	}
	function getSize(index$1) {
		return measurements.get(index$1) ?? options.estimateSize(index$1);
	}
	function getCount() {
		return options.getCount ? options.getCount() : options.count;
	}
	function calculateTotalSize() {
		let total = 0;
		const count = getCount();
		for (let i = 0; i < count; i++) total += getSize(i);
		return total;
	}
	function calculateVirtualItems() {
		if (!scrollElement) return [];
		scrollOffset = scrollElement.scrollTop;
		const count = getCount();
		const overscan = options.overscan ?? 3;
		const containerHeight = scrollElement.clientHeight;
		if (containerHeight <= 0) return [];
		let start = 0;
		let offset = 0;
		for (let i = 0; i < count; i++) {
			const size = getSize(i);
			if (offset + size > scrollOffset) {
				start = Math.max(0, i - overscan);
				break;
			}
			offset += size;
		}
		offset = 0;
		for (let i = 0; i < start; i++) offset += getSize(i);
		const items = [];
		const scrollEnd = scrollOffset + containerHeight;
		for (let i = start; i < count; i++) {
			const size = getSize(i);
			const itemEnd = offset + size;
			items.push({
				index: i,
				start: offset,
				size,
				end: itemEnd,
				key: i
			});
			offset = itemEnd;
			if (offset > scrollEnd + overscan * options.estimateSize(0)) break;
		}
		return items;
	}
	init();
	return {
		subscribe: (callback) => {
			subscribers.add(callback);
			return () => {
				subscribers.delete(callback);
			};
		},
		getVirtualItems: () => {
			if (!isInitialized) init();
			return calculateVirtualItems();
		},
		getTotalSize: () => {
			if (!isInitialized) init();
			return calculateTotalSize();
		},
		refresh: () => {
			notifySubscribers();
		}
	};
}
var root_2$2 = from_html(`<button><!> <!></button>`);
var root_6 = from_html(`<div><!> <!></div>`);
function Renderer($$anchor, $$props) {
	const renderElement = ($$anchor$1, element = noop) => {
		var fragment = comment();
		var node = first_child(fragment);
		var consequent_2 = ($$anchor$2) => {
			var button = root_2$2();
			button.__click = () => handleClick(element());
			var node_1 = child(button);
			var consequent = ($$anchor$3) => {
				var text$1 = text();
				template_effect(() => set_text(text$1, element().text));
				append($$anchor$3, text$1);
			};
			if_block(node_1, ($$render) => {
				if (element().text) $$render(consequent);
			});
			var node_2 = sibling(node_1, 2);
			var consequent_1 = ($$anchor$3) => {
				var fragment_2 = comment();
				each(first_child(fragment_2), 17, () => element().children, index, ($$anchor$4, child$1) => {
					renderElement($$anchor$4, () => get(child$1));
				});
				append($$anchor$3, fragment_2);
			};
			if_block(node_2, ($$render) => {
				if (element().children) $$render(consequent_1);
			});
			reset(button);
			template_effect(() => set_class(button, 1, clsx(element().css)));
			append($$anchor$2, button);
		};
		var alternate = ($$anchor$2) => {
			var div = root_6();
			var node_4 = child(div);
			var consequent_3 = ($$anchor$3) => {
				var text_1 = text();
				template_effect(() => set_text(text_1, element().text));
				append($$anchor$3, text_1);
			};
			if_block(node_4, ($$render) => {
				if (element().text) $$render(consequent_3);
			});
			var node_5 = sibling(node_4, 2);
			var consequent_4 = ($$anchor$3) => {
				var fragment_5 = comment();
				each(first_child(fragment_5), 17, () => element().children, index, ($$anchor$4, child$1) => {
					renderElement($$anchor$4, () => get(child$1));
				});
				append($$anchor$3, fragment_5);
			};
			if_block(node_5, ($$render) => {
				if (element().children) $$render(consequent_4);
			});
			reset(div);
			template_effect(() => set_class(div, 1, clsx(element().css)));
			append($$anchor$2, div);
		};
		if_block(node, ($$render) => {
			if (element().tagName === "BUTTON") $$render(consequent_2);
			else $$render(alternate, false);
		});
		append($$anchor$1, fragment);
	};
	const handleClick = (element) => {
		if (element.onClick) element.onClick(element.id || 0);
	};
	var fragment_7 = comment();
	var node_7 = first_child(fragment_7);
	var consequent_5 = ($$anchor$1) => {
		var fragment_8 = comment();
		each(first_child(fragment_8), 17, () => $$props.elements, index, ($$anchor$2, element) => {
			renderElement($$anchor$2, () => get(element));
		});
		append($$anchor$1, fragment_8);
	};
	var alternate_1 = ($$anchor$1) => {
		renderElement($$anchor$1, () => $$props.elements);
	};
	if_block(node_7, ($$render) => {
		if (Array.isArray($$props.elements)) $$render(consequent_5);
		else $$render(alternate_1, false);
	});
	append($$anchor, fragment_7);
}
delegate(["click"]);
var root_3$1 = from_html(`<i class="icon-attention aJY aN4"></i>`);
var root_4$2 = from_html(`<input/>`);
var root$3 = from_html(`<div class="_2 aN4"> </div> <div><div role="button" tabindex="0"><!> <!></div> <!></div>`, 1);
function CellEditable($$anchor, $$props) {
	push($$props, true);
	let contentClass = prop($$props, "contentClass", 3, ""), inputClass = prop($$props, "inputClass", 3, ""), required = prop($$props, "required", 3, false), type = prop($$props, "type", 3, "text");
	let isEditing = state(false);
	let inputRef = state(void 0);
	let currentValue = state(proxy($$props.getValue ? $$props.getValue($$props.saveOn) : ($$props.saveOn || {})[$$props.save]));
	user_effect(() => {
		if (get(isEditing) && get(inputRef)) get(inputRef).focus();
	});
	function extractValue(newValue) {
		if (type() === "number") return parseFloat(newValue || "0");
		return newValue;
	}
	function handleClick(ev) {
		ev.stopPropagation();
		set(isEditing, true);
	}
	function handleBlur(ev) {
		ev.stopPropagation();
		const newValue = extractValue(ev.target.value);
		if (get(currentValue) !== newValue) {
			console.log("cambiando::");
			if ($$props.onChange) $$props.onChange(newValue);
			set(currentValue, newValue, true);
		}
		set(isEditing, false);
	}
	var fragment = root$3();
	var div = first_child(fragment);
	var text$1 = child(div, true);
	reset(div);
	var div_1 = sibling(div, 2);
	var div_2 = child(div_1);
	div_2.__click = handleClick;
	div_2.__keydown = (ev) => {
		if (ev.key === "Enter" || ev.key === " ") {
			ev.preventDefault();
			handleClick(ev);
		}
	};
	let styles;
	var node = child(div_2);
	var consequent = ($$anchor$1) => {
		{
			let $0 = user_derived(() => $$props.render(get(currentValue)));
			Renderer($$anchor$1, { get elements() {
				return get($0);
			} });
		}
	};
	var alternate = ($$anchor$1) => {
		var text_1 = text();
		template_effect(() => set_text(text_1, get(currentValue)));
		append($$anchor$1, text_1);
	};
	if_block(node, ($$render) => {
		if ($$props.render) $$render(consequent);
		else $$render(alternate, false);
	});
	var node_1 = sibling(node, 2);
	var consequent_1 = ($$anchor$1) => {
		append($$anchor$1, root_3$1());
	};
	if_block(node_1, ($$render) => {
		if (!get(currentValue) && required()) $$render(consequent_1);
	});
	reset(div_2);
	var node_2 = sibling(div_2, 2);
	var consequent_2 = ($$anchor$1) => {
		var input = root_4$2();
		remove_input_defaults(input);
		bind_this(input, ($$value) => set(inputRef, $$value), () => get(inputRef));
		template_effect(() => {
			set_attribute(input, "type", type());
			set_value(input, get(currentValue) || "");
			set_class(input, 1, `w-full ${inputClass()}`, "aN4");
		});
		event("blur", input, handleBlur);
		append($$anchor$1, input);
	};
	if_block(node_2, ($$render) => {
		if (get(isEditing)) $$render(consequent_2);
	});
	reset(div_1);
	template_effect(() => {
		set_text(text$1, get(currentValue));
		set_class(div_1, 1, `_1 ${$$props.css ?? ""}`, "aN4");
		set_class(div_2, 1, contentClass(), "aN4");
		styles = set_style(div_2, "", styles, { visibility: get(isEditing) ? "hidden" : "visible" });
	});
	append($$anchor, fragment);
	pop();
}
delegate(["click", "keydown"]);
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
const updateHeightAndScroll = (state$1, setters, immediate = false) => {
	const { initialized, mode, containerElement, viewportElement, calculatedItemHeight, scrollTop } = state$1;
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
		let offset$1 = 0;
		for (let i = 0; i < safeIdx; i++) {
			const raw = heightCache[i];
			offset$1 += Number.isFinite(raw) && raw > 0 ? raw : calculatedItemHeight;
		}
		return offset$1;
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
var root_1$2 = from_html(`<!> <div><!></div>`, 1);
var root$2 = from_html(`<div><div><div><div></div></div></div></div>`);
function SvelteVirtualList($$anchor, $$props) {
	push($$props, true);
	const rafSchedule = createRafScheduler();
	const GLOBAL_CORRECTION_COOLDOWN = 16;
	const lastCorrectionTimestampByViewport = /* @__PURE__ */ new WeakMap();
	const INTERNAL_DEBUG = Boolean(typeof process !== "undefined" && ({}?.PUBLIC_SVELTE_VIRTUAL_LIST_DEBUG === "true" || {}?.SVELTE_VIRTUAL_LIST_DEBUG === "true"));
	const items = prop($$props, "items", 19, () => []), defaultEstimatedItemHeight = prop($$props, "defaultEstimatedItemHeight", 3, 40), debug = prop($$props, "debug", 3, false), mode = prop($$props, "mode", 3, "topToBottom"), bufferSize = prop($$props, "bufferSize", 3, 20);
	const itemElements = proxy([]);
	let height = state(0);
	const isCalculatingHeight = false;
	let isScrolling = state(false);
	let lastMeasuredIndex = state(-1);
	let lastScrollTopSnapshot = state(0);
	let heightUpdateTimeout = null;
	let resizeObserver = null;
	let itemResizeObserver = null;
	const dirtyItems = proxy(/* @__PURE__ */ new Set());
	let dirtyItemsCount = state(0);
	let measuredFallbackHeight = state(0);
	let prevVisibleRange = state(null);
	let prevHeight = state(0);
	let prevTotalHeightForScrollCorrection = state(0);
	let lastBottomDistance = state(null);
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
	let suppressBottomAnchoringUntilMs = state(0);
	const displayItems = user_derived(() => () => {
		const visibleRange = get(visibleItems)();
		return (mode() === "bottomToTop" ? items().slice(visibleRange.start, visibleRange.end).reverse() : items().slice(visibleRange.start, visibleRange.end)).map((item, sliceIndex) => ({
			item,
			originalIndex: mode() === "bottomToTop" ? visibleRange.end - 1 - sliceIndex : visibleRange.start + sliceIndex,
			sliceIndex
		}));
	});
	const handleHeightChangesScrollCorrection = (heightChanges) => {
		if (!heightManager.viewportElement || !heightManager.initialized || get(userHasScrolledAway)) return;
		if (mode() === "bottomToTop" && wasAtBottomBeforeHeightChange && !get(programmaticScrollInProgress) && performance.now() >= get(suppressBottomAnchoringUntilMs)) {
			const now = performance.now();
			const viewportEl = heightManager.viewport;
			if (now - (lastCorrectionTimestampByViewport.get(viewportEl) ?? 0) < GLOBAL_CORRECTION_COOLDOWN) {
				set(suppressBottomAnchoringUntilMs, now + 50);
				return;
			}
			lastCorrectionTimestampByViewport.set(viewportEl, now);
			const approximateScrollTop = Math.max(0, get(totalHeight)() - get(height));
			log("b2t-correction-approx", { approximateScrollTop });
			heightManager.viewport.scrollTop = approximateScrollTop;
			heightManager.scrollTop = approximateScrollTop;
			tick().then(() => {
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
					set(suppressBottomAnchoringUntilMs, performance.now() + 200);
				}
			});
			return;
		}
		const currentScrollTop = heightManager.viewport.scrollTop;
		const maxScrollTop = Math.max(0, get(totalHeight)() - get(height));
		let heightChangeAboveViewport = 0;
		const currentVisibleRange = get(visibleItems)();
		for (const change of heightChanges) if (change.index < currentVisibleRange.start) heightChangeAboveViewport += change.delta;
		if (Math.abs(heightChangeAboveViewport) > 1) {
			const newScrollTop = Math.min(maxScrollTop, Math.max(0, currentScrollTop + heightChangeAboveViewport));
			heightManager.viewport.scrollTop = newScrollTop;
			heightManager.scrollTop = newScrollTop;
		}
	};
	const triggerHeightUpdate = createAdvancedThrottledCallback(() => {
		if (get(dirtyItemsCount) > 0) {
			wasAtBottomBeforeHeightChange = get(atBottom);
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
		set(prevTotalHeightForScrollCorrection, heightManager.totalHeight, true);
		heightUpdateTimeout = calculateAverageHeightDebounced(isCalculatingHeight, heightUpdateTimeout, get(visibleItems), itemElements, heightManager.getHeightCache(), get(lastMeasuredIndex), heightManager.averageHeight, (result) => {
			heightManager.itemHeight = result.newHeight;
			set(lastMeasuredIndex, result.newLastMeasuredIndex, true);
			if (result.heightChanges.length > 0) heightManager.processDirtyHeights(result.heightChanges);
			if (result.heightChanges.length > 0 && mode() === "bottomToTop") {
				const changes = result.heightChanges;
				queueMicrotask(() => handleHeightChangesScrollCorrection(changes));
			}
			if (mode() === "topToBottom" && heightManager.isReady && heightManager.initialized) {
				const oldTotal = get(prevTotalHeightForScrollCorrection);
				const newTotal = heightManager.totalHeight;
				const deltaTotal = newTotal - oldTotal;
				if (Math.abs(deltaTotal) > 1) {
					const maxScrollTop = Math.max(0, newTotal - (get(height) || 0));
					const tolerance = Math.max(heightManager.averageHeight, 10);
					const currentScrollTop = heightManager.viewport.scrollTop;
					if (Math.abs(currentScrollTop - maxScrollTop) <= tolerance) {
						const adjusted = Math.min(maxScrollTop, Math.max(0, currentScrollTop + deltaTotal));
						heightManager.viewport.scrollTop = adjusted;
						heightManager.scrollTop = adjusted;
					}
				}
			}
			untrack(() => {
				dirtyItems.clear();
				set(dirtyItemsCount, 0);
				wasAtBottomBeforeHeightChange = false;
			});
			heightManager.endDynamicUpdate();
		}, get(lastMeasuredIndex) < 0 ? 0 : 100, dirtyItems, 0, 0, mode());
	};
	let userHasScrolledAway = state(false);
	let programmaticScrollInProgress = state(false);
	let lastCalculatedHeight = state(0);
	let lastItemsLength = state(0);
	const totalHeight = user_derived(() => () => heightManager.totalHeight);
	const atBottom = user_derived(() => heightManager.scrollTop >= get(totalHeight)() - get(height) - 1);
	let wasAtBottomBeforeHeightChange = false;
	let lastVisibleRange = null;
	function updateDebugTailDistance() {
		if (!heightManager.viewportElement) return;
		const last = heightManager.viewport.querySelector(`[data-original-index="${items().length - 1}"]`);
		if (!last) return;
		const v = heightManager.viewport.getBoundingClientRect();
		const r = last.getBoundingClientRect();
		set(lastBottomDistance, Math.round(Math.abs(r.bottom - v.bottom)), true);
		if (INTERNAL_DEBUG) console.info("[SVL] bottomDistance(px):", get(lastBottomDistance));
	}
	user_effect(() => {
		if (heightManager.initialized && mode() === "bottomToTop" && heightManager.viewportElement) {
			const targetScrollTop = Math.max(0, get(totalHeight)() - get(height));
			const currentScrollTop = heightManager.viewport.scrollTop;
			const scrollDifference = Math.abs(currentScrollTop - targetScrollTop);
			const heightChanged = Math.abs(heightManager.averageHeight - get(lastCalculatedHeight)) > 1;
			const maxScrollTop = Math.max(0, get(totalHeight)() - get(height));
			const isAtBottom = Math.abs(currentScrollTop - maxScrollTop) < heightManager.averageHeight;
			if (heightChanged && !get(userHasScrolledAway) && !isAtBottom && !get(programmaticScrollInProgress) && performance.now() >= get(suppressBottomAnchoringUntilMs) && !heightManager.isDynamicUpdateInProgress && scrollDifference > heightManager.averageHeight * 3) {
				const roundedTargetScrollTop = Math.round(targetScrollTop);
				heightManager.viewport.scrollTop = roundedTargetScrollTop;
				heightManager.scrollTop = roundedTargetScrollTop;
			}
			if (scrollDifference > heightManager.averageHeight * 5) set(userHasScrolledAway, true);
			set(lastCalculatedHeight, heightManager.averageHeight, true);
		}
	});
	user_effect(() => {
		const currentItemsLength = items().length;
		if (heightManager.initialized && mode() === "bottomToTop" && heightManager.isReady && get(lastItemsLength) > 0) {
			if (currentItemsLength - get(lastItemsLength) !== 0) {
				const currentScrollTop = heightManager.viewport.scrollTop;
				const currentCalculatedItemHeight = heightManager.averageHeight;
				const currentHeight = get(height);
				const currentTotalHeight = get(totalHeight)();
				const maxScrollTop = Math.max(0, currentTotalHeight - currentHeight);
				if (Math.abs(currentScrollTop - Math.max(0, get(lastItemsLength) * currentCalculatedItemHeight - currentHeight)) < currentCalculatedItemHeight * 2 || currentScrollTop === 0) heightManager.runDynamicUpdate(() => {
					const newScrollTop = maxScrollTop;
					heightManager.viewport.scrollTop = newScrollTop;
					heightManager.scrollTop = newScrollTop;
					set(userHasScrolledAway, false);
				});
			}
		}
		set(lastItemsLength, currentItemsLength, true);
	});
	user_effect(() => {
		if (heightManager.isReady) {
			const h = heightManager.container.getBoundingClientRect().height;
			if (Number.isFinite(h) && h > 0) set(height, h, true);
		}
	});
	user_effect(() => {
		if (get(height) === 0 && heightManager.isReady) {
			const h = heightManager.container.getBoundingClientRect().height;
			if (Number.isFinite(h) && h > 0) set(measuredFallbackHeight, h, true);
		}
	});
	const visibleItems = user_derived(() => () => {
		if (!items().length) return {
			start: 0,
			end: 0
		};
		const viewportHeight = get(height) || 0;
		if (mode() === "bottomToTop" && !heightManager.initialized && heightManager.scrollTop === 0 && viewportHeight > 0) {
			lastVisibleRange = calculateVisibleRange(Math.max(0, get(totalHeight)() - viewportHeight), viewportHeight, heightManager.averageHeight, items().length, bufferSize(), mode(), get(atBottom), wasAtBottomBeforeHeightChange, lastVisibleRange, get(totalHeight)(), heightManager.getHeightCache());
			return lastVisibleRange;
		}
		lastVisibleRange = calculateVisibleRange(heightManager.scrollTop, viewportHeight, heightManager.averageHeight, items().length, bufferSize(), mode(), get(atBottom), wasAtBottomBeforeHeightChange, lastVisibleRange, get(totalHeight)(), heightManager.getHeightCache());
		return lastVisibleRange;
	});
	const handleScroll = () => {
		if (!heightManager.viewportElement) return;
		if (!get(isScrolling)) {
			set(isScrolling, true);
			rafSchedule(() => {
				const current = heightManager.viewport.scrollTop;
				if (mode() === "bottomToTop") {
					if (get(lastScrollTopSnapshot) - current > .5) {
						set(suppressBottomAnchoringUntilMs, performance.now() + 450);
						set(userHasScrolledAway, true);
					}
				}
				set(lastScrollTopSnapshot, current, true);
				heightManager.scrollTop = current;
				updateDebugTailDistance();
				if (INTERNAL_DEBUG) {
					const vr = get(visibleItems)();
					log("scroll", {
						mode: mode(),
						scrollTop: heightManager.scrollTop,
						height: get(height),
						totalHeight: get(totalHeight)(),
						averageItemHeight: heightManager.averageHeight,
						visibleRange: vr
					});
				}
				set(isScrolling, false);
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
			tick().then(() => {
				requestAnimationFrame(() => {
					requestAnimationFrame(() => {
						if (!heightManager.isReady) return;
						const measuredHeight = heightManager.container.getBoundingClientRect().height;
						set(height, measuredHeight, true);
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
								tick().then(() => {
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
			height: get(height),
			scrollTop: heightManager.scrollTop
		}, {
			setHeight: (h) => set(height, h, true),
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
			get(visibleItems)();
			for (const entry of entries) {
				const element = entry.target;
				const elementIndex = itemElements.indexOf(element);
				const actualIndex = parseInt(element.dataset.originalIndex || "-1", 10);
				if (elementIndex !== -1) {
					if (actualIndex >= 0) {
						const currentHeight = element.getBoundingClientRect().height;
						if (isSignificantHeightChange(actualIndex, currentHeight, heightManager.getHeightCache())) {
							if (get(dirtyItemsCount) === 0) wasAtBottomBeforeHeightChange = get(atBottom);
							dirtyItems.add(actualIndex);
							set(dirtyItemsCount, dirtyItems.size, true);
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
	onMount(() => {
		log("onMount-enter", {
			mode: mode(),
			items: items().length
		});
		updateHeightAndScroll$1();
		tick().then(() => requestAnimationFrame(() => requestAnimationFrame(() => {
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
			set(prevVisibleRange, get(visibleItems)(), true);
			set(prevHeight, heightManager.averageHeight, true);
		}
	});
	const scrollToIndex = (index$1, smoothScroll = true, shouldThrowOnBounds = true) => {
		console.warn("SvelteVirtualList: scrollToIndex is deprecated and will be removed in a future version. Use the new scroll method from the component instance instead.");
		scroll({
			index: index$1,
			smoothScroll,
			shouldThrowOnBounds
		});
	};
	const scroll = async (options) => {
		const { index: index$1, smoothScroll, shouldThrowOnBounds, align } = {
			...DEFAULT_SCROLL_OPTIONS,
			...options
		};
		if (!items().length) return;
		if (!heightManager.viewportElement) {
			tick().then(() => {
				if (!heightManager.viewportElement) return;
				scroll({
					index: index$1,
					smoothScroll,
					shouldThrowOnBounds,
					align
				});
			});
			return;
		}
		let targetIndex = index$1;
		if (targetIndex < 0 || targetIndex >= items().length) if (shouldThrowOnBounds) throw new Error(`scroll: index ${targetIndex} is out of bounds (0-${items().length - 1})`);
		else targetIndex = Math.max(0, Math.min(targetIndex, items().length - 1));
		const { start: firstVisibleIndex, end: lastVisibleIndex } = get(visibleItems)();
		const scrollTarget = calculateScrollTarget({
			mode: mode(),
			align: align || "auto",
			targetIndex,
			itemsLength: items().length,
			calculatedItemHeight: heightManager.averageHeight,
			height: get(height),
			scrollTop: heightManager.scrollTop,
			firstVisibleIndex,
			lastVisibleIndex,
			heightCache: heightManager.getHeightCache()
		});
		if (scrollTarget === null) return;
		set(programmaticScrollInProgress, true);
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
				const index$2 = parseInt(el.getAttribute("data-original-index") || "-1");
				if (index$2 > maxIndex) {
					maxIndex = index$2;
					maxElement = el;
				}
			}
			maxElement?.scrollIntoView({ behavior: "smooth" });
			await tick();
			await new Promise((resolve) => setTimeout(resolve, 100));
			await tick();
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
			set(programmaticScrollInProgress, false);
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
	var div = root$2();
	attribute_effect(div, () => ({
		id: "virtual-list-container",
		...$$props.testId ? { "data-testid": `${$$props.testId}-container` } : {},
		class: $$props.containerClass ?? "virtual-list-container"
	}), void 0, void 0, "aN7");
	var div_1 = child(div);
	attribute_effect(div_1, () => ({
		id: "virtual-list-viewport",
		...$$props.testId ? { "data-testid": `${$$props.testId}-viewport` } : {},
		class: $$props.viewportClass ?? "virtual-list-viewport",
		onscroll: handleScroll
	}), void 0, void 0, "aN7");
	var div_2 = child(div_1);
	attribute_effect(div_2, ($0) => ({
		id: "virtual-list-content",
		...$$props.testId ? { "data-testid": `${$$props.testId}-content` } : {},
		class: $$props.contentClass ?? "virtual-list-content",
		[STYLE]: $0
	}), [() => ({ height: `${Math.max(get(height), get(totalHeight)()) ?? ""}px` })], void 0, "aN7");
	var div_3 = child(div_2);
	attribute_effect(div_3, ($0) => ({
		id: "virtual-list-items",
		...$$props.testId ? { "data-testid": `${$$props.testId}-items` } : {},
		class: $$props.itemsClass ?? "virtual-list-items",
		[STYLE]: $0
	}), [() => ({
		visibility: get(height) === 0 && mode() === "bottomToTop" ? "hidden" : "visible",
		transform: `translateY(${(() => {
			const viewportHeight = get(height) || get(measuredFallbackHeight) || 0;
			const visibleRange = get(visibleItems)();
			const effectiveHeight = viewportHeight === 0 ? 400 : viewportHeight;
			return Math.round(calculateTransformY(mode(), items().length, visibleRange.end, visibleRange.start, heightManager.averageHeight, effectiveHeight, get(totalHeight)(), heightManager.getHeightCache(), get(measuredFallbackHeight)));
		})() ?? ""}px)`
	})], void 0, "aN7");
	each(div_3, 23, () => get(displayItems)(), (currentItemWithIndex) => currentItemWithIndex.originalIndex, ($$anchor$1, currentItemWithIndex, i) => {
		var fragment = root_1$2();
		var node = first_child(fragment);
		var consequent = ($$anchor$2) => {
			const debugInfo = user_derived(() => createDebugInfo(get(visibleItems)(), items().length, Object.keys(heightManager.getHeightCache()).length, heightManager.averageHeight, heightManager.scrollTop, get(height) || 0, get(totalHeight)()));
			var text$1 = text();
			template_effect(($0) => set_text(text$1, $0), [() => $$props.debugFunction ? $$props.debugFunction(get(debugInfo)) : console.info("Virtual List Debug:", get(debugInfo))]);
			append($$anchor$2, text$1);
		};
		if_block(node, ($$render) => {
			if (debug() && get(i) === 0 && shouldShowDebugInfo(get(prevVisibleRange), get(visibleItems)(), get(prevHeight), heightManager.averageHeight)) $$render(consequent);
		});
		var div_4 = sibling(node, 2);
		attribute_effect(div_4, () => ({
			"data-original-index": get(currentItemWithIndex).originalIndex,
			...$$props.testId ? { "data-testid": `${$$props.testId}-item-${get(currentItemWithIndex).originalIndex}` } : {}
		}), void 0, void 0, "aN7");
		snippet(child(div_4), () => $$props.renderItem, () => get(currentItemWithIndex).item, () => get(currentItemWithIndex).originalIndex);
		reset(div_4);
		bind_this(div_4, ($$value, currentItemWithIndex$1) => itemElements[currentItemWithIndex$1.sliceIndex] = $$value, (currentItemWithIndex$1) => itemElements?.[currentItemWithIndex$1.sliceIndex], () => [get(currentItemWithIndex)]);
		action(div_4, ($$node) => autoObserveItemResize?.($$node));
		append($$anchor$1, fragment);
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
	#_totalMeasuredHeight = state(0);
	get _totalMeasuredHeight() {
		return get(this.#_totalMeasuredHeight);
	}
	set _totalMeasuredHeight(value) {
		set(this.#_totalMeasuredHeight, value, true);
	}
	#_measuredCount = state(0);
	get _measuredCount() {
		return get(this.#_measuredCount);
	}
	set _measuredCount(value) {
		set(this.#_measuredCount, value, true);
	}
	#_itemLength = state(0);
	get _itemLength() {
		return get(this.#_itemLength);
	}
	set _itemLength(value) {
		set(this.#_itemLength, value, true);
	}
	#_itemHeight = state(40);
	get _itemHeight() {
		return get(this.#_itemHeight);
	}
	set _itemHeight(value) {
		set(this.#_itemHeight, value, true);
	}
	#_averageHeight = state(40);
	get _averageHeight() {
		return get(this.#_averageHeight);
	}
	set _averageHeight(value) {
		set(this.#_averageHeight, value, true);
	}
	#_totalHeight = state(0);
	get _totalHeight() {
		return get(this.#_totalHeight);
	}
	set _totalHeight(value) {
		set(this.#_totalHeight, value, true);
	}
	_measuredFlags = null;
	#_initialized = state(false);
	get _initialized() {
		return get(this.#_initialized);
	}
	set _initialized(value) {
		set(this.#_initialized, value, true);
	}
	#_scrollTop = state(0);
	get _scrollTop() {
		return get(this.#_scrollTop);
	}
	set _scrollTop(value) {
		set(this.#_scrollTop, value, true);
	}
	#_containerElement = state(null);
	get _containerElement() {
		return get(this.#_containerElement);
	}
	set _containerElement(value) {
		set(this.#_containerElement, value, true);
	}
	#_viewportElement = state(null);
	get _viewportElement() {
		return get(this.#_viewportElement);
	}
	set _viewportElement(value) {
		set(this.#_viewportElement, value, true);
	}
	_internalDebug = false;
	#_isReady = state(false);
	get _isReady() {
		return get(this.#_isReady);
	}
	set _isReady(value) {
		set(this.#_isReady, value, true);
	}
	#_dynamicUpdateInProgress = state(false);
	get _dynamicUpdateInProgress() {
		return get(this.#_dynamicUpdateInProgress);
	}
	set _dynamicUpdateInProgress(value) {
		set(this.#_dynamicUpdateInProgress, value, true);
	}
	#_dynamicUpdateDepth = state(0);
	get _dynamicUpdateDepth() {
		return get(this.#_dynamicUpdateDepth);
	}
	set _dynamicUpdateDepth(value) {
		set(this.#_dynamicUpdateDepth, value, true);
	}
	#_itemsWrapperElement = state(null);
	get _itemsWrapperElement() {
		return get(this.#_itemsWrapperElement);
	}
	set _itemsWrapperElement(value) {
		set(this.#_itemsWrapperElement, value, true);
	}
	#_gridDetected = state(false);
	get _gridDetected() {
		return get(this.#_gridDetected);
	}
	set _gridDetected(value) {
		set(this.#_gridDetected, value, true);
	}
	#_gridColumns = state(1);
	get _gridColumns() {
		return get(this.#_gridColumns);
	}
	set _gridColumns(value) {
		set(this.#_gridColumns, value, true);
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
			const { index: index$1, oldHeight, newHeight } = change;
			if (oldHeight !== void 0) {
				heightDelta -= oldHeight;
				countDelta -= 1;
			}
			if (newHeight !== void 0) {
				heightDelta += newHeight;
				countDelta += 1;
				this._heightCache[index$1] = newHeight;
			} else delete this._heightCache[index$1];
			if (this._measuredFlags && index$1 >= 0 && index$1 < this._measuredFlags.length) this._measuredFlags[index$1] = 1;
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
	setMeasuredHeight(index$1, height) {
		if (index$1 < 0 || index$1 >= this._itemLength) return;
		const prev = this._heightCache[index$1];
		if (Number.isFinite(prev) && prev > 0) this._totalMeasuredHeight -= prev;
		else this._measuredCount += 1;
		if (Number.isFinite(height) && height > 0) {
			this._heightCache[index$1] = height;
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
var root_1$1 = from_html(`<i class="icon-attention c-red"></i>`);
var root_2$1 = from_html(`<input class="aMA"/>`);
var root_4$1 = from_html(`<div class="_3 min-h-28 px-8 py-4 flex items-center aMA" role="button" tabindex="0"> </div>`);
var root_3 = from_html(`<div class="h-200 w-400 p-4 _4 overflow-auto aMA"><!></div>`);
var root$1 = from_html(`<div class="_2 aMA"> </div> <div class="_1 aMA" role="button" tabindex="0"><div> <!></div> <!> <!></div>`, 1);
function CellSelector($$anchor, $$props) {
	push($$props, true);
	let saveOn = prop($$props, "saveOn", 7);
	let show = user_derived(() => Core.popoverShowID === $$props.id);
	let refElement = state(null);
	let inputRef = state(null);
	let filterValue = state("");
	let selected = state(null);
	let avoidBlur = false;
	const buildMap = () => {
		if (WeakSearchRef.has($$props.options)) return;
		const idToRecord = /* @__PURE__ */ new Map();
		const valueToRecord = /* @__PURE__ */ new Map();
		for (const e of $$props.options) {
			idToRecord.set(e[$$props.keyId], e);
			valueToRecord.set(e[$$props.keyName].toLowerCase(), e);
		}
		WeakSearchRef.set($$props.options, {
			idToRecord,
			valueToRecord
		});
	};
	buildMap();
	let optionsFiltered = user_derived(() => {
		console.log("filtrando valores::", get(filterValue));
		if (!get(filterValue)) return $$props.options;
		else return $$props.options.filter((x) => String(x[$$props.keyName] || "").toLowerCase().includes(get(filterValue)));
	});
	const clearValueIfNotFound = (target) => {
		set(selected, (WeakSearchRef.get($$props.options)?.valueToRecord || /* @__PURE__ */ new Map()).get(String(target.value || "").toLowerCase()), true);
		if (get(selected)) target.value = get(selected)[$$props.keyName];
		else target.value = "";
		if (saveOn() && $$props.save) if (get(selected)) saveOn()[$$props.save] = get(selected)[$$props.keyId];
		else delete saveOn()[$$props.save];
	};
	const onSelect = (e) => {
		console.log("selected", e);
		set(selected, e, true);
		if (get(inputRef)) get(inputRef).value = e[$$props.keyName];
		if (saveOn() && $$props.save) saveOn()[$$props.save] = e[$$props.keyId];
		Core.popoverShowID = 0;
	};
	user_effect(() => {
		if (get(selected) || !get(selected)) console.log("selected 2", get(selected));
	});
	const renderContent = user_derived(() => $$props.render ? $$props.render(get(selected), get(show)) : get(selected)?.[$$props.keyName] || "");
	const handlwShowClick = () => {
		set(filterValue, "");
		Core.popoverShowID = $$props.id;
	};
	user_effect(() => {
		if (get(show) && get(inputRef)) {
			if (get(selected)) get(inputRef).value = get(selected)[$$props.keyName];
			get(inputRef).focus();
		}
	});
	user_effect(() => {
		if (saveOn() && $$props.save) {
			buildMap();
			untrack(() => {
				const idToRecord = WeakSearchRef.get($$props.options)?.idToRecord || /* @__PURE__ */ new Map();
				set(selected, saveOn()[$$props.save] ? idToRecord.get(saveOn()[$$props.save]) : null, true);
				console.log("selected 3", get(selected));
				if (get(selected)) {
					if (get(inputRef)) get(inputRef).value = get(selected)[$$props.keyName];
				} else if (get(inputRef)) get(inputRef).value = "";
			});
		}
	});
	var fragment = root$1();
	var div = first_child(fragment);
	var text$1 = child(div, true);
	reset(div);
	var div_1 = sibling(div, 2);
	div_1.__keydown = (ev) => {
		if (ev.key === "Enter" || ev.key === " ") ev.preventDefault();
	};
	div_1.__click = (ev) => {
		ev.stopPropagation();
		handlwShowClick();
	};
	var div_2 = child(div_1);
	let styles;
	var text_1 = child(div_2);
	var node = sibling(text_1);
	var consequent = ($$anchor$1) => {
		append($$anchor$1, root_1$1());
	};
	if_block(node, ($$render) => {
		if (!get(selected) && $$props.required) $$render(consequent);
	});
	reset(div_2);
	var node_1 = sibling(div_2, 2);
	var consequent_1 = ($$anchor$1) => {
		var input = root_2$1();
		input.__keyup = (ev) => {
			const value = (ev.target.value || "").trim();
			throttle(() => {
				set(filterValue, String(value || "").toLowerCase(), true);
			}, 200);
		};
		bind_this(input, ($$value) => set(inputRef, $$value), () => get(inputRef));
		template_effect(($0) => set_attribute(input, "id", $0), [() => String($$props.id)]);
		event("focus", input, () => {});
		event("blur", input, (ev) => {
			if (avoidBlur) {
				avoidBlur = false;
				return;
			}
			clearValueIfNotFound(ev.target);
			Core.popoverShowID = 0;
			set(filterValue, "");
		});
		append($$anchor$1, input);
	};
	if_block(node_1, ($$render) => {
		if (get(show)) $$render(consequent_1);
	});
	Popover2(sibling(node_1, 2), {
		get referenceElement() {
			return get(refElement);
		},
		get open() {
			return get(show);
		},
		placement: "bottom",
		children: ($$anchor$1, $$slotProps) => {
			var div_3 = root_3();
			var node_3 = child(div_3);
			{
				const renderItem = ($$anchor$2, item = noop) => {
					var div_4 = root_4$1();
					div_4.__click = (ev) => {
						ev.stopPropagation();
						onSelect(item());
					};
					div_4.__mousedown = () => {
						avoidBlur = true;
					};
					div_4.__keydown = (ev) => {
						if (ev.key === "Enter" || ev.key === " ") {
							ev.preventDefault();
							onSelect(item());
						}
					};
					var text_2 = child(div_4, true);
					reset(div_4);
					template_effect(() => set_text(text_2, item()[$$props.keyName]));
					append($$anchor$2, div_4);
				};
				dist_default(node_3, {
					get items() {
						return get(optionsFiltered);
					},
					renderItem,
					$$slots: { renderItem: true }
				});
			}
			reset(div_3);
			append($$anchor$1, div_3);
		},
		$$slots: { default: true }
	});
	reset(div_1);
	bind_this(div_1, ($$value) => set(refElement, $$value), () => get(refElement));
	template_effect(() => {
		set_text(text$1, get(renderContent));
		set_class(div_2, 1, $$props.contentClass, "aMA");
		styles = set_style(div_2, "", styles, { visibility: get(show) ? "hidden" : "visible" });
		set_text(text_1, `${get(renderContent) ?? ""} `);
	});
	append($$anchor, fragment);
	pop();
}
delegate([
	"keydown",
	"click",
	"keyup",
	"mousedown"
]);
var root_2 = from_html(`<div class="aKf aMw"> </div>`);
var root_8 = from_html(`<i></i>`);
var root_9 = from_html(`<span class="aKm aMw"> </span>`);
var root_10 = from_html(`<div class="aKn aMw"><!></div>`);
var root_23 = from_html(`<div class="aKo aMw"><!></div>`);
var root_7 = from_html(`<div><div class="aKk aMw"> </div> <div class="aKl aMw"><!> <!> <!> <div class="aKp aMw"><!></div> <!></div></div>`);
var root_27 = from_html(`<i></i>`);
var root_28 = from_html(`<span class="aKm aMw"> </span>`);
var root_29 = from_html(`<div class="aKn aMw"><!></div>`);
var root_42 = from_html(`<div class="aKo aMw"><!></div>`);
var root_26 = from_html(`<div><!> <!> <!> <div class="aKp aMw"><!></div> <!></div>`);
var root_4 = from_html(`<div role="button" tabindex="0"><div class="aKh aMw"></div></div>`);
var root_1 = from_html(`<div class="aKe aMw"><!></div>`);
var root_46 = from_html(`<th><div class="aMw"> </div></th>`);
var root_48 = from_html(`<th><div class="aMw"> </div></th>`);
var root_47 = from_html(`<tr class="aK3 aK4 aMw"></tr>`);
var root_49 = from_html(`<tr><td class="aKc aMw"><div class="aKd aMw"> </div></td></tr>`);
var root_65 = from_html(`<span> </span>`);
var root_68 = from_html(`<button class="_11 _e aMw" title="edit"><i class="icon-pencil"></i></button>`);
var root_69 = from_html(`<button class="_11 _d aMw" title="delete"><i class="icon-trash"></i></button>`);
var root_67 = from_html(`<div class="flex gap-4 items-center justify-center aMw"><!> <!></div>`);
var root_53 = from_html(`<td><!> <!></td>`);
var root_70 = from_html(`<tr><td style="border: none;"></td></tr>`);
var root_52 = from_html(`<tr></tr> <!>`, 1);
var root_45 = from_html(`<table><thead class="aK2 aMw"><tr class="aK3 aMw"></tr><!></thead><tbody class="aK6 aMw"><!></tbody></table>`);
var root = from_html(`<div><!></div>`);
function VTable($$anchor, $$props) {
	push($$props, true);
	let maxHeight = prop($$props, "maxHeight", 3, "calc(100vh - 8rem)"), css = prop($$props, "css", 3, ""), tableCss = prop($$props, "tableCss", 3, ""), estimateSize = prop($$props, "estimateSize", 3, 34), overscan = prop($$props, "overscan", 3, 15), emptyMessage = prop($$props, "emptyMessage", 3, "No se encontraron registros."), useFilterCache = prop($$props, "useFilterCache", 3, false), mobileCardCss = prop($$props, "mobileCardCss", 3, "");
	let containerRef = state(void 0);
	let virtualItems = state(proxy([]));
	let totalSize = state(0);
	let virtualizerStore = null;
	let isInitialized = false;
	let dataVersion = state(0);
	const filterCache = /* @__PURE__ */ new WeakMap();
	let windowWidth = state(proxy(typeof window !== "undefined" ? window.innerWidth : 1024));
	user_effect(() => {
		const handleResize = () => {
			set(windowWidth, window.innerWidth, true);
		};
		window.addEventListener("resize", handleResize);
		return () => window.removeEventListener("resize", handleResize);
	});
	const processedColumns = user_derived(() => {
		const level1 = [];
		const level2 = [];
		const flatColumns = [];
		for (const col of $$props.columns) {
			const colWithSpan = { ...col };
			colWithSpan._colspan = col.subcols?.length || 0;
			level1.push(colWithSpan);
			if (col.subcols && col.subcols.length > 0) for (const subcol of col.subcols) {
				level2.push(subcol);
				flatColumns.push(subcol);
			}
			else flatColumns.push(colWithSpan);
		}
		return {
			level1,
			level2,
			flatColumns,
			hasSubcols: level2.length > 0
		};
	});
	const mobileColumns = user_derived(() => {
		return get(processedColumns).flatColumns.filter((col) => col.mobile).sort((a, b) => (a.mobile?.order || 0) - (b.mobile?.order || 0));
	});
	const isMobileView = user_derived(() => get(windowWidth) < 580 && get(mobileColumns).length > 0);
	const filterTextArray = user_derived(() => ($$props.filterText || "").toLowerCase().split(" ").filter((x) => x.length > 1));
	const filteredData = user_derived(() => {
		console.log("filterText", $$props.filterText);
		if ($$props.filterText && $$props.getFilterContent) {
			const filtered = $$props.data.filter((x) => {
				let content = "";
				if (useFilterCache() && typeof x === "object" && x !== null) {
					const cached = filterCache.get(x);
					if (cached) content = cached;
					else {
						content = $$props.getFilterContent(x).toLowerCase();
						filterCache.set(x, content);
					}
				} else content = $$props.getFilterContent(x).toLowerCase();
				return include(content, get(filterTextArray));
			});
			console.log("data filtrada::", filtered);
			return filtered;
		} else return $$props.data;
	});
	user_effect(() => {
		if ($$props.selected || !$$props.selected) console.log("selected::", $$props.selected);
	});
	user_effect(() => {
		if (get(isMobileView)) return;
		if (get(containerRef) && !virtualizerStore) {
			virtualizerStore = createVirtualizer({
				count: 0,
				getScrollElement: () => get(containerRef),
				estimateSize: () => estimateSize(),
				overscan: overscan(),
				getCount: () => get(filteredData).length
			});
			const updateVirtualItems = () => {
				if (!virtualizerStore) return;
				const items = virtualizerStore.getVirtualItems();
				const size = virtualizerStore.getTotalSize();
				set(virtualItems, [...items], true);
				set(totalSize, size, true);
			};
			const unsubscribe = virtualizerStore.subscribe(updateVirtualItems);
			requestAnimationFrame(() => {
				updateVirtualItems();
				isInitialized = true;
			});
			return () => {
				isInitialized = false;
				virtualizerStore = null;
				unsubscribe();
			};
		}
	});
	user_effect(() => {
		get(filteredData);
		get(filteredData).length;
		if (isInitialized && virtualizerStore) untrack(() => {
			update(dataVersion);
			virtualizerStore.refresh();
			const items = virtualizerStore.getVirtualItems();
			const size = virtualizerStore.getTotalSize();
			set(virtualItems, [...items], true);
			set(totalSize, size, true);
		});
	});
	function getCellContent(column, record, index$1) {
		const rec = {};
		if (column.render) {
			const renderedContent = column.render(record, index$1);
			if (typeof renderedContent === "string") rec.contentHTML = renderedContent;
			else if (renderedContent) rec.contentAST = renderedContent;
		}
		if (column.getValue) rec.content = column.getValue(record, index$1);
		if ($$props.cellRenderer && column.id) rec.useSnippet = true;
		rec.css = typeof column.cellCss === "string" ? column.cellCss : column.onCellEdit ? "relative" : "px-8 py-4";
		if (column.css) rec.css += " " + column.css;
		return rec;
	}
	function getHeaderContent(column) {
		return typeof column.header === "function" ? column.header() : column.header;
	}
	function isRowSelected(record, index$1) {
		if (!$$props.selected || !$$props.isSelected) return false;
		return $$props.isSelected(record, $$props.selected);
	}
	function handleRowClick(record, index$1) {
		if ($$props.onRowClick) $$props.onRowClick(record, index$1);
	}
	var div = root();
	let classes;
	var node = child(div);
	var consequent_25 = ($$anchor$1) => {
		var div_1 = root_1();
		var node_1 = child(div_1);
		var consequent = ($$anchor$2) => {
			var div_2 = root_2();
			var text$1 = child(div_2, true);
			reset(div_2);
			template_effect(() => set_text(text$1, emptyMessage()));
			append($$anchor$2, div_2);
		};
		var alternate_15 = ($$anchor$2) => {
			{
				const renderItem = ($$anchor$3, record = noop, index$1 = noop) => {
					var div_3 = root_4();
					div_3.__click = () => handleRowClick(record(), index$1());
					div_3.__keydown = (ev) => {
						if (ev.key === "Enter" || ev.key === " ") handleRowClick(record(), index$1());
					};
					var div_4 = child(div_3);
					each(div_4, 21, () => get(mobileColumns), index, ($$anchor$4, column) => {
						const mobile = user_derived(() => get(column).mobile);
						const shouldRender = user_derived(() => !get(mobile)?.if || get(mobile).if(record(), index$1()));
						var fragment_1 = comment();
						var node_2 = first_child(fragment_1);
						var consequent_24 = ($$anchor$5) => {
							const cellData = user_derived(() => getCellContent(get(column), record(), index$1()));
							var fragment_2 = comment();
							var node_3 = first_child(fragment_2);
							var consequent_12 = ($$anchor$6) => {
								var div_5 = root_7();
								var div_6 = child(div_5);
								var text_1 = child(div_6, true);
								reset(div_6);
								var div_7 = sibling(div_6, 2);
								var node_4 = child(div_7);
								var consequent_1 = ($$anchor$7) => {
									var i_1 = root_8();
									template_effect(() => set_class(i_1, 1, `icon-${get(mobile).icon ?? ""} ${(get(mobile)?.iconCss || "") ?? ""}`, "aMw"));
									append($$anchor$7, i_1);
								};
								if_block(node_4, ($$render) => {
									if (get(mobile)?.icon) $$render(consequent_1);
								});
								var node_5 = sibling(node_4, 2);
								var consequent_2 = ($$anchor$7) => {
									var span = root_9();
									var text_2 = child(span, true);
									reset(span);
									template_effect(() => set_text(text_2, get(mobile).labelLeft));
									append($$anchor$7, span);
								};
								if_block(node_5, ($$render) => {
									if (get(mobile)?.labelLeft) $$render(consequent_2);
								});
								var node_6 = sibling(node_5, 2);
								var consequent_4 = ($$anchor$7) => {
									var div_8 = root_10();
									var node_7 = child(div_8);
									var consequent_3 = ($$anchor$8) => {
										var fragment_3 = comment();
										html(first_child(fragment_3), () => get(mobile).elementLeft);
										append($$anchor$8, fragment_3);
									};
									var alternate = ($$anchor$8) => {
										Renderer($$anchor$8, { get elements() {
											return get(mobile).elementLeft;
										} });
									};
									if_block(node_7, ($$render) => {
										if (typeof get(mobile).elementLeft === "string") $$render(consequent_3);
										else $$render(alternate, false);
									});
									reset(div_8);
									append($$anchor$7, div_8);
								};
								if_block(node_6, ($$render) => {
									if (get(mobile)?.elementLeft) $$render(consequent_4);
								});
								var div_9 = sibling(node_6, 2);
								var node_9 = child(div_9);
								var consequent_6 = ($$anchor$7) => {
									const renderedContent = user_derived(() => get(mobile).render(record(), index$1()));
									var fragment_5 = comment();
									var node_10 = first_child(fragment_5);
									var consequent_5 = ($$anchor$8) => {
										var fragment_6 = comment();
										html(first_child(fragment_6), () => get(renderedContent));
										append($$anchor$8, fragment_6);
									};
									var alternate_1 = ($$anchor$8) => {
										Renderer($$anchor$8, { get elements() {
											return get(renderedContent);
										} });
									};
									if_block(node_10, ($$render) => {
										if (typeof get(renderedContent) === "string") $$render(consequent_5);
										else $$render(alternate_1, false);
									});
									append($$anchor$7, fragment_5);
								};
								var alternate_5 = ($$anchor$7) => {
									var fragment_8 = comment();
									var node_12 = first_child(fragment_8);
									var consequent_7 = ($$anchor$8) => {
										var fragment_9 = comment();
										snippet(first_child(fragment_9), () => $$props.cellRenderer, record, () => get(column), () => get(cellData).content, index$1);
										append($$anchor$8, fragment_9);
									};
									var alternate_4 = ($$anchor$8) => {
										var fragment_10 = comment();
										var node_14 = first_child(fragment_10);
										var consequent_8 = ($$anchor$9) => {
											Renderer($$anchor$9, { get elements() {
												return get(cellData).contentAST;
											} });
										};
										var alternate_3 = ($$anchor$9) => {
											var fragment_12 = comment();
											var node_15 = first_child(fragment_12);
											var consequent_9 = ($$anchor$10) => {
												var fragment_13 = comment();
												html(first_child(fragment_13), () => get(cellData).contentHTML);
												append($$anchor$10, fragment_13);
											};
											var alternate_2 = ($$anchor$10) => {
												var text_3 = text();
												template_effect(() => set_text(text_3, get(cellData).content));
												append($$anchor$10, text_3);
											};
											if_block(node_15, ($$render) => {
												if (get(cellData).contentHTML) $$render(consequent_9);
												else $$render(alternate_2, false);
											}, true);
											append($$anchor$9, fragment_12);
										};
										if_block(node_14, ($$render) => {
											if (get(cellData).contentAST) $$render(consequent_8);
											else $$render(alternate_3, false);
										}, true);
										append($$anchor$8, fragment_10);
									};
									if_block(node_12, ($$render) => {
										if (get(cellData).useSnippet && $$props.cellRenderer) $$render(consequent_7);
										else $$render(alternate_4, false);
									}, true);
									append($$anchor$7, fragment_8);
								};
								if_block(node_9, ($$render) => {
									if (get(mobile)?.render) $$render(consequent_6);
									else $$render(alternate_5, false);
								});
								reset(div_9);
								var node_17 = sibling(div_9, 2);
								var consequent_11 = ($$anchor$7) => {
									var div_10 = root_23();
									var node_18 = child(div_10);
									var consequent_10 = ($$anchor$8) => {
										var fragment_15 = comment();
										html(first_child(fragment_15), () => get(mobile).elementRight);
										append($$anchor$8, fragment_15);
									};
									var alternate_6 = ($$anchor$8) => {
										Renderer($$anchor$8, { get elements() {
											return get(mobile).elementRight;
										} });
									};
									if_block(node_18, ($$render) => {
										if (typeof get(mobile).elementRight === "string") $$render(consequent_10);
										else $$render(alternate_6, false);
									});
									reset(div_10);
									append($$anchor$7, div_10);
								};
								if_block(node_17, ($$render) => {
									if (get(mobile)?.elementRight) $$render(consequent_11);
								});
								reset(div_7);
								reset(div_5);
								template_effect(() => {
									set_class(div_5, 1, `aKi aKj ${(get(mobile)?.css || "col-span-full") ?? ""}`, "aMw");
									set_text(text_1, get(mobile).labelTop);
								});
								append($$anchor$6, div_5);
							};
							var alternate_14 = ($$anchor$6) => {
								var div_11 = root_26();
								var node_20 = child(div_11);
								var consequent_13 = ($$anchor$7) => {
									var i_2 = root_27();
									template_effect(() => set_class(i_2, 1, `icon-${get(mobile).icon ?? ""} ${(get(mobile)?.iconCss || "") ?? ""}`, "aMw"));
									append($$anchor$7, i_2);
								};
								if_block(node_20, ($$render) => {
									if (get(mobile)?.icon) $$render(consequent_13);
								});
								var node_21 = sibling(node_20, 2);
								var consequent_14 = ($$anchor$7) => {
									var span_1 = root_28();
									var text_4 = child(span_1, true);
									reset(span_1);
									template_effect(() => set_text(text_4, get(mobile).labelLeft));
									append($$anchor$7, span_1);
								};
								if_block(node_21, ($$render) => {
									if (get(mobile)?.labelLeft) $$render(consequent_14);
								});
								var node_22 = sibling(node_21, 2);
								var consequent_16 = ($$anchor$7) => {
									var div_12 = root_29();
									var node_23 = child(div_12);
									var consequent_15 = ($$anchor$8) => {
										var fragment_17 = comment();
										html(first_child(fragment_17), () => get(mobile).elementLeft);
										append($$anchor$8, fragment_17);
									};
									var alternate_7 = ($$anchor$8) => {
										Renderer($$anchor$8, { get elements() {
											return get(mobile).elementLeft;
										} });
									};
									if_block(node_23, ($$render) => {
										if (typeof get(mobile).elementLeft === "string") $$render(consequent_15);
										else $$render(alternate_7, false);
									});
									reset(div_12);
									append($$anchor$7, div_12);
								};
								if_block(node_22, ($$render) => {
									if (get(mobile)?.elementLeft) $$render(consequent_16);
								});
								var div_13 = sibling(node_22, 2);
								var node_25 = child(div_13);
								var consequent_18 = ($$anchor$7) => {
									const renderedContent = user_derived(() => get(mobile).render(record(), index$1()));
									var fragment_19 = comment();
									var node_26 = first_child(fragment_19);
									var consequent_17 = ($$anchor$8) => {
										var fragment_20 = comment();
										html(first_child(fragment_20), () => get(renderedContent));
										append($$anchor$8, fragment_20);
									};
									var alternate_8 = ($$anchor$8) => {
										Renderer($$anchor$8, { get elements() {
											return get(renderedContent);
										} });
									};
									if_block(node_26, ($$render) => {
										if (typeof get(renderedContent) === "string") $$render(consequent_17);
										else $$render(alternate_8, false);
									});
									append($$anchor$7, fragment_19);
								};
								var alternate_12 = ($$anchor$7) => {
									var fragment_22 = comment();
									var node_28 = first_child(fragment_22);
									var consequent_19 = ($$anchor$8) => {
										var fragment_23 = comment();
										snippet(first_child(fragment_23), () => $$props.cellRenderer, record, () => get(column), () => get(cellData).content, index$1);
										append($$anchor$8, fragment_23);
									};
									var alternate_11 = ($$anchor$8) => {
										var fragment_24 = comment();
										var node_30 = first_child(fragment_24);
										var consequent_20 = ($$anchor$9) => {
											Renderer($$anchor$9, { get elements() {
												return get(cellData).contentAST;
											} });
										};
										var alternate_10 = ($$anchor$9) => {
											var fragment_26 = comment();
											var node_31 = first_child(fragment_26);
											var consequent_21 = ($$anchor$10) => {
												var fragment_27 = comment();
												html(first_child(fragment_27), () => get(cellData).contentHTML);
												append($$anchor$10, fragment_27);
											};
											var alternate_9 = ($$anchor$10) => {
												var text_5 = text();
												template_effect(() => set_text(text_5, get(cellData).content));
												append($$anchor$10, text_5);
											};
											if_block(node_31, ($$render) => {
												if (get(cellData).contentHTML) $$render(consequent_21);
												else $$render(alternate_9, false);
											}, true);
											append($$anchor$9, fragment_26);
										};
										if_block(node_30, ($$render) => {
											if (get(cellData).contentAST) $$render(consequent_20);
											else $$render(alternate_10, false);
										}, true);
										append($$anchor$8, fragment_24);
									};
									if_block(node_28, ($$render) => {
										if (get(cellData).useSnippet && $$props.cellRenderer) $$render(consequent_19);
										else $$render(alternate_11, false);
									}, true);
									append($$anchor$7, fragment_22);
								};
								if_block(node_25, ($$render) => {
									if (get(mobile)?.render) $$render(consequent_18);
									else $$render(alternate_12, false);
								});
								reset(div_13);
								var node_33 = sibling(div_13, 2);
								var consequent_23 = ($$anchor$7) => {
									var div_14 = root_42();
									var node_34 = child(div_14);
									var consequent_22 = ($$anchor$8) => {
										var fragment_29 = comment();
										html(first_child(fragment_29), () => get(mobile).elementRight);
										append($$anchor$8, fragment_29);
									};
									var alternate_13 = ($$anchor$8) => {
										Renderer($$anchor$8, { get elements() {
											return get(mobile).elementRight;
										} });
									};
									if_block(node_34, ($$render) => {
										if (typeof get(mobile).elementRight === "string") $$render(consequent_22);
										else $$render(alternate_13, false);
									});
									reset(div_14);
									append($$anchor$7, div_14);
								};
								if_block(node_33, ($$render) => {
									if (get(mobile)?.elementRight) $$render(consequent_23);
								});
								reset(div_11);
								template_effect(() => set_class(div_11, 1, `aKi ${(get(mobile)?.css || "col-span-full") ?? ""}`, "aMw"));
								append($$anchor$6, div_11);
							};
							if_block(node_3, ($$render) => {
								if (get(mobile)?.labelTop) $$render(consequent_12);
								else $$render(alternate_14, false);
							});
							append($$anchor$5, fragment_2);
						};
						if_block(node_2, ($$render) => {
							if (get(shouldRender)) $$render(consequent_24);
						});
						append($$anchor$4, fragment_1);
					});
					reset(div_4);
					reset(div_3);
					template_effect(() => set_class(div_3, 1, `aKg mb-6 ${(mobileCardCss() || "") ?? ""}`, "aMw"));
					append($$anchor$3, div_3);
				};
				dist_default($$anchor$2, {
					get items() {
						return get(filteredData);
					},
					renderItem,
					$$slots: { renderItem: true }
				});
			}
		};
		if_block(node_1, ($$render) => {
			if (get(filteredData).length === 0) $$render(consequent);
			else $$render(alternate_15, false);
		});
		reset(div_1);
		template_effect(() => set_style(div_1, `height: ${maxHeight() ?? ""};`));
		append($$anchor$1, div_1);
	};
	var alternate_23 = ($$anchor$1) => {
		var table = root_45();
		var thead = child(table);
		var tr = child(thead);
		each(tr, 21, () => get(processedColumns).level1, index, ($$anchor$2, column) => {
			var th = root_46();
			let classes_1;
			var div_15 = child(th);
			var text_6 = child(div_15, true);
			reset(div_15);
			reset(th);
			template_effect(($0, $1) => {
				classes_1 = set_class(th, 1, `aK5 ${(get(column).headerCss || "") ?? ""}`, "aMw", classes_1, { hsc: (get(column).subcols || []).length > 0 });
				set_style(th, $0);
				set_attribute(th, "colspan", get(column)._colspan || 1);
				set_attribute(th, "rowspan", get(column)._colspan ? 1 : get(processedColumns).hasSubcols ? 2 : 1);
				set_text(text_6, $1);
			}, [() => get(column).headerStyle ? Object.entries(get(column).headerStyle).map(([k, v]) => `${k}: ${v}`).join("; ") : "", () => getHeaderContent(get(column))]);
			append($$anchor$2, th);
		});
		reset(tr);
		var node_36 = sibling(tr);
		var consequent_26 = ($$anchor$2) => {
			var tr_1 = root_47();
			each(tr_1, 21, () => get(processedColumns).level2, index, ($$anchor$3, column) => {
				var th_1 = root_48();
				var div_16 = child(th_1);
				var text_7 = child(div_16, true);
				reset(div_16);
				reset(th_1);
				template_effect(($0, $1) => {
					set_class(th_1, 1, `aK5 ${(get(column).headerCss || "") ?? ""}`, "aMw");
					set_style(th_1, $0);
					set_text(text_7, $1);
				}, [() => get(column).headerStyle ? Object.entries(get(column).headerStyle).map(([k, v]) => `${k}: ${v}`).join("; ") : "", () => getHeaderContent(get(column))]);
				append($$anchor$3, th_1);
			});
			reset(tr_1);
			append($$anchor$2, tr_1);
		};
		if_block(node_36, ($$render) => {
			if (get(processedColumns).hasSubcols) $$render(consequent_26);
		});
		reset(thead);
		var tbody = sibling(thead);
		var node_37 = child(tbody);
		var consequent_27 = ($$anchor$2) => {
			var tr_2 = root_49();
			var td = child(tr_2);
			var div_17 = child(td);
			var text_8 = child(div_17, true);
			reset(div_17);
			reset(td);
			reset(tr_2);
			template_effect(() => {
				set_attribute(td, "colspan", get(processedColumns).flatColumns.length);
				set_text(text_8, emptyMessage());
			});
			append($$anchor$2, tr_2);
		};
		var alternate_22 = ($$anchor$2) => {
			var fragment_31 = comment();
			each(first_child(fragment_31), 19, () => get(virtualItems), (row) => `${row.index}-${get(dataVersion)}`, ($$anchor$3, row, i) => {
				const firstItemStart = user_derived(() => get(virtualItems)[0]?.start || 0);
				const isFinal = user_derived(() => get(i) === get(virtualItems).length - 1);
				const remainingSize = user_derived(() => get(totalSize) - (get(virtualItems)[0]?.size || estimateSize()) * get(virtualItems).length);
				const record = user_derived(() => get(filteredData)[get(row).index]);
				var fragment_32 = comment();
				var node_39 = first_child(fragment_32);
				var consequent_38 = ($$anchor$4) => {
					const selected = user_derived(() => isRowSelected(get(record), get(row).index));
					var fragment_33 = root_52();
					var tr_3 = first_child(fragment_33);
					let classes_2;
					tr_3.__click = () => handleRowClick(get(record), get(row).index);
					each(tr_3, 23, () => get(processedColumns).flatColumns, (column, j) => `${j}_${$$props.filterText || ""}`, ($$anchor$5, column) => {
						const cellData = user_derived(() => getCellContent(get(column), get(record), get(row).index));
						var td_1 = root_53();
						td_1.__click = (ev) => {
							if (get(column).onCellEdit) ev.stopPropagation();
						};
						var node_40 = child(td_1);
						var consequent_28 = ($$anchor$6) => {
							{
								let $0 = user_derived(() => get(column).render ? () => get(column).render?.(get(record), get(row).index) : void 0);
								CellEditable($$anchor$6, {
									get contentClass() {
										return get(column).css;
									},
									get inputClass() {
										return get(column).inputCss;
									},
									getValue: () => get(cellData).content,
									get render() {
										return get($0);
									},
									onChange: (v) => {
										get(column).onCellEdit?.(get(record), v);
									}
								});
							}
						};
						var alternate_21 = ($$anchor$6) => {
							var fragment_35 = comment();
							var node_41 = first_child(fragment_35);
							var consequent_29 = ($$anchor$7) => {
								CellSelector($$anchor$7, {
									get options() {
										return get(column).cellOptions;
									},
									keyId: "",
									keyName: "",
									onChange: (v) => {
										get(column).onCellSelect?.(get(record), v);
									}
								});
							};
							var alternate_20 = ($$anchor$7) => {
								var fragment_37 = comment();
								var node_42 = first_child(fragment_37);
								var consequent_30 = ($$anchor$8) => {
									var fragment_38 = comment();
									snippet(first_child(fragment_38), () => $$props.cellRenderer, () => get(record), () => get(column), () => get(cellData).content, () => get(row).index);
									append($$anchor$8, fragment_38);
								};
								var alternate_19 = ($$anchor$8) => {
									var fragment_39 = comment();
									var node_44 = first_child(fragment_39);
									var consequent_31 = ($$anchor$9) => {
										Renderer($$anchor$9, { get elements() {
											return get(cellData).contentAST;
										} });
									};
									var alternate_18 = ($$anchor$9) => {
										var fragment_41 = comment();
										var node_45 = first_child(fragment_41);
										var consequent_32 = ($$anchor$10) => {
											var fragment_42 = comment();
											html(first_child(fragment_42), () => get(cellData).contentHTML);
											append($$anchor$10, fragment_42);
										};
										var alternate_17 = ($$anchor$10) => {
											var fragment_43 = comment();
											var node_47 = first_child(fragment_43);
											var consequent_33 = ($$anchor$11) => {
												var fragment_44 = comment();
												each(first_child(fragment_44), 17, () => highlString(get(cellData).content, get(filterTextArray)), index, ($$anchor$12, part) => {
													var span_2 = root_65();
													let classes_3;
													var text_9 = child(span_2, true);
													reset(span_2);
													template_effect(() => {
														classes_3 = set_class(span_2, 1, "aMw", null, classes_3, { _2: get(part).highl });
														set_text(text_9, get(part).text);
													});
													append($$anchor$12, span_2);
												});
												append($$anchor$11, fragment_44);
											};
											var alternate_16 = ($$anchor$11) => {
												var text_10 = text();
												template_effect(() => set_text(text_10, get(cellData).content));
												append($$anchor$11, text_10);
											};
											if_block(node_47, ($$render) => {
												if ($$props.filterText && get(column).highlight) $$render(consequent_33);
												else $$render(alternate_16, false);
											});
											append($$anchor$10, fragment_43);
										};
										if_block(node_45, ($$render) => {
											if (get(cellData).contentHTML) $$render(consequent_32);
											else $$render(alternate_17, false);
										}, true);
										append($$anchor$9, fragment_41);
									};
									if_block(node_44, ($$render) => {
										if (get(cellData).contentAST) $$render(consequent_31);
										else $$render(alternate_18, false);
									}, true);
									append($$anchor$8, fragment_39);
								};
								if_block(node_42, ($$render) => {
									if (get(cellData).useSnippet && $$props.cellRenderer) $$render(consequent_30);
									else $$render(alternate_19, false);
								}, true);
								append($$anchor$7, fragment_37);
							};
							if_block(node_41, ($$render) => {
								if (get(column).onCellSelect) $$render(consequent_29);
								else $$render(alternate_20, false);
							}, true);
							append($$anchor$6, fragment_35);
						};
						if_block(node_40, ($$render) => {
							if (get(column).onCellEdit) $$render(consequent_28);
							else $$render(alternate_21, false);
						});
						var node_49 = sibling(node_40, 2);
						var consequent_36 = ($$anchor$6) => {
							var div_18 = root_67();
							var node_50 = child(div_18);
							var consequent_34 = ($$anchor$7) => {
								var button = root_68();
								button.__click = (ev) => {
									ev.stopPropagation();
									get(column).buttonEditHandler?.(get(record));
								};
								append($$anchor$7, button);
							};
							if_block(node_50, ($$render) => {
								if (get(column).buttonEditHandler) $$render(consequent_34);
							});
							var node_51 = sibling(node_50, 2);
							var consequent_35 = ($$anchor$7) => {
								var button_1 = root_69();
								button_1.__click = (ev) => {
									ev.stopPropagation();
									get(column).buttonDeleteHandler?.(get(record));
								};
								append($$anchor$7, button_1);
							};
							if_block(node_51, ($$render) => {
								if (get(column).buttonDeleteHandler) $$render(consequent_35);
							});
							reset(div_18);
							append($$anchor$6, div_18);
						};
						if_block(node_49, ($$render) => {
							if (get(column).buttonEditHandler || get(column).buttonDeleteHandler) $$render(consequent_36);
						});
						reset(td_1);
						template_effect(($0) => {
							set_class(td_1, 1, get(cellData).css, "aMw");
							set_style(td_1, $0);
						}, [() => get(column).cellStyle ? Object.entries(get(column).cellStyle).map(([k, v]) => `${k}: ${v}`).join("; ") : ""]);
						append($$anchor$5, td_1);
					});
					reset(tr_3);
					var node_52 = sibling(tr_3, 2);
					var consequent_37 = ($$anchor$5) => {
						var tr_4 = root_70();
						template_effect(() => set_style(tr_4, `height: ${get(remainingSize) ?? ""}px; visibility: hidden;`));
						append($$anchor$5, tr_4);
					};
					if_block(node_52, ($$render) => {
						if (get(isFinal)) $$render(consequent_37);
					});
					template_effect(() => {
						classes_2 = set_class(tr_3, 1, "aK7 aMw", null, classes_2, {
							aK8: get(row).index % 2 === 0,
							aK9: get(row).index % 2 !== 0,
							aKa: get(selected)
						});
						set_style(tr_3, `transform: translateY(${get(firstItemStart) ?? ""}px);`);
					});
					append($$anchor$4, fragment_33);
				};
				if_block(node_39, ($$render) => {
					if (get(record)) $$render(consequent_38);
				});
				append($$anchor$3, fragment_32);
			});
			append($$anchor$2, fragment_31);
		};
		if_block(node_37, ($$render) => {
			if (get(filteredData).length === 0) $$render(consequent_27);
			else $$render(alternate_22, false);
		});
		reset(tbody);
		reset(table);
		template_effect(() => set_class(table, 1, `aK1 ${tableCss() ?? ""}`, "aMw"));
		append($$anchor$1, table);
	};
	if_block(node, ($$render) => {
		if (get(isMobileView)) $$render(consequent_25);
		else $$render(alternate_23, false);
	});
	reset(div);
	bind_this(div, ($$value) => set(containerRef, $$value), () => get(containerRef));
	template_effect(() => {
		classes = set_class(div, 1, `aJZ ${css() ?? ""}`, "aMw", classes, { _14: get(isMobileView) });
		set_style(div, `max-height: ${maxHeight() ?? ""}; overflow: ${get(isMobileView) ? "hidden" : "auto"};`);
	});
	append($$anchor, div);
	pop();
}
delegate(["click", "keydown"]);
export { dist_default as n, VTable as t };
