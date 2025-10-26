/**
 * Svelte 5 Virtual Scroller
 * Simple API for virtual scrolling with Svelte 5 runes
 */

export interface VirtualItem {
	index: number;
	start: number;
	size: number;
	end: number;
	key: string | number;
}

export interface VirtualizerOptions {
	count: number;
	getScrollElement: () => HTMLElement | null;
	estimateSize: (index: number) => number;
	overscan?: number;
	getCount?: () => number;
}

interface VirtualizerStore {
	subscribe: (callback: () => void) => () => void;
	getVirtualItems: () => VirtualItem[];
	getTotalSize: () => number;
	refresh: () => void;
}

/**
 * Create a virtualizer that returns a subscribable store
 */
export function createVirtualizer(options: VirtualizerOptions): VirtualizerStore {
	let scrollElement: HTMLElement | null = null;
	let scrollOffset = 0;
	let measurements = new Map<number, number>();
	let rafId: number | null = null;
	let subscribers = new Set<() => void>();
	let isInitialized = false;

	let resizeObserver: ResizeObserver | null = null;

	// Initialize
	function init() {
		if (isInitialized) return;
		
		scrollElement = options.getScrollElement();
		if (!scrollElement) return;

		isInitialized = true;

		// Read current scroll position immediately
		scrollOffset = scrollElement.scrollTop;

		const handleScroll = () => {
			if (rafId !== null) {
				cancelAnimationFrame(rafId);
			}

			rafId = requestAnimationFrame(() => {
				if (scrollElement) {
					scrollOffset = scrollElement.scrollTop;
					notifySubscribers();
				}
				rafId = null;
			});
		};

		scrollElement.addEventListener('scroll', handleScroll, { passive: true });

		// Watch for container resize
		resizeObserver = new ResizeObserver(() => {
			notifySubscribers();
		});
		resizeObserver.observe(scrollElement);
	}

	function cleanup() {
		if (scrollElement) {
			scrollElement.removeEventListener('scroll', () => {});
		}
		if (resizeObserver) {
			resizeObserver.disconnect();
			resizeObserver = null;
		}
	}

	function notifySubscribers() {
		subscribers.forEach(callback => callback());
	}

	function getSize(index: number): number {
		return measurements.get(index) ?? options.estimateSize(index);
	}

	function getCount(): number {
		return options.getCount ? options.getCount() : options.count;
	}

	function calculateTotalSize(): number {
		let total = 0;
		const count = getCount();
		for (let i = 0; i < count; i++) {
			total += getSize(i);
		}
		return total;
	}

	function calculateVirtualItems(): VirtualItem[] {
		if (!scrollElement) return [];

		// Always read the latest scroll position
		scrollOffset = scrollElement.scrollTop;

		const count = getCount();
		const overscan = options.overscan ?? 3;
		const containerHeight = scrollElement.clientHeight;

		// If container has no height yet, return empty
		if (containerHeight <= 0) return [];

		// Find the range of items to render
		let start = 0;
		let offset = 0;

		// Find start index
		for (let i = 0; i < count; i++) {
			const size = getSize(i);
			if (offset + size > scrollOffset) {
				start = Math.max(0, i - overscan);
				break;
			}
			offset += size;
		}

		// Calculate offset for start index
		offset = 0;
		for (let i = 0; i < start; i++) {
			offset += getSize(i);
		}

		// Build items array
		const items: VirtualItem[] = [];
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

			// Stop when we're past the visible area + overscan
			if (offset > scrollEnd + (overscan * options.estimateSize(0))) {
				break;
			}
		}

		return items;
	}

	// Auto-initialize on first access
	init();

	return {
		subscribe: (callback: () => void) => {
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

export type { VirtualizerStore };

