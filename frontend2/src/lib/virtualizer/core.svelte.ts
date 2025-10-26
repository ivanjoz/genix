/**
 * Virtual Scroller Core for Svelte 5
 * A lightweight virtual scrolling implementation using Svelte 5 runes
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
	paddingStart?: number;
	paddingEnd?: number;
	scrollMargin?: number;
	scrollPaddingStart?: number;
	scrollPaddingEnd?: number;
	initialOffset?: number;
}

export interface VirtualizerState {
	scrollOffset: number;
	scrollDirection: 'forward' | 'backward' | null;
	isScrolling: boolean;
}

export class Virtualizer {
	private options: VirtualizerOptions;
	private scrollElement: HTMLElement | null = null;
	private state = $state<VirtualizerState>({
		scrollOffset: 0,
		scrollDirection: null,
		isScrolling: false
	});
	private measurements = $state<Map<number, number>>(new Map());
	private isScrollingTimeout: ReturnType<typeof setTimeout> | null = null;
	private rafId: number | null = null;

	constructor(options: VirtualizerOptions) {
		this.options = options;
	}

	// Initialize the virtualizer with scroll element
	init() {
		this.scrollElement = this.options.getScrollElement();
		if (!this.scrollElement) return;

		this.scrollElement.addEventListener('scroll', this.handleScroll, { passive: true });
		this.updateScrollOffset();
	}

	// Clean up event listeners
	destroy() {
		if (this.scrollElement) {
			this.scrollElement.removeEventListener('scroll', this.handleScroll);
		}
		if (this.isScrollingTimeout) {
			clearTimeout(this.isScrollingTimeout);
		}
		if (this.rafId !== null) {
			cancelAnimationFrame(this.rafId);
		}
	}

	private handleScroll = () => {
		if (this.rafId !== null) {
			cancelAnimationFrame(this.rafId);
		}

		this.rafId = requestAnimationFrame(() => {
			this.updateScrollOffset();
			this.rafId = null;
		});

		// Set isScrolling state
		if (!this.state.isScrolling) {
			this.state.isScrolling = true;
		}

		// Clear timeout and set new one
		if (this.isScrollingTimeout) {
			clearTimeout(this.isScrollingTimeout);
		}

		this.isScrollingTimeout = setTimeout(() => {
			this.state.isScrolling = false;
			this.isScrollingTimeout = null;
		}, 150);
	};

	private updateScrollOffset() {
		if (!this.scrollElement) return;

		const prevOffset = this.state.scrollOffset;
		const scrollOffset = this.scrollElement.scrollTop;

		this.state.scrollOffset = scrollOffset;

		if (prevOffset !== scrollOffset) {
			this.state.scrollDirection = prevOffset < scrollOffset ? 'forward' : 'backward';
		}
	}

	// Measure an element at a specific index
	measureElement(element: HTMLElement | null, index: number) {
		if (!element) return;

		const size = element.getBoundingClientRect().height;
		if (this.measurements.get(index) !== size) {
			this.measurements.set(index, size);
		}
	}

	// Get the measured or estimated size for an index
	private getSize(index: number): number {
		return this.measurements.get(index) ?? this.options.estimateSize(index);
	}

	// Calculate total size of all items
	getTotalSize(): number {
		let total = 0;
		for (let i = 0; i < this.options.count; i++) {
			total += this.getSize(i);
		}
		return total + (this.options.paddingStart || 0) + (this.options.paddingEnd || 0);
	}

	// Get virtual items to render
	getVirtualItems(): VirtualItem[] {
		if (!this.scrollElement) return [];

		const count = this.options.count;
		const overscan = this.options.overscan ?? 3;
		const scrollOffset = this.state.scrollOffset;
		const containerHeight = this.scrollElement.clientHeight;
		const paddingStart = this.options.paddingStart || 0;

		// Calculate start and end offsets
		const scrollStart = Math.max(0, scrollOffset - paddingStart);
		const scrollEnd = scrollOffset + containerHeight;

		// Find the range of items to render
		let start = 0;
		let end = 0;
		let offset = paddingStart;

		// Find start index
		for (let i = 0; i < count; i++) {
			const size = this.getSize(i);
			if (offset + size > scrollStart) {
				start = Math.max(0, i - overscan);
				break;
			}
			offset += size;
		}

		// Calculate offset for start index
		offset = paddingStart;
		for (let i = 0; i < start; i++) {
			offset += this.getSize(i);
		}

		// Find end index
		const items: VirtualItem[] = [];
		for (let i = start; i < count; i++) {
			const size = this.getSize(i);
			const itemEnd = offset + size;

			items.push({
				index: i,
				start: offset,
				size,
				end: itemEnd,
				key: i
			});

			offset = itemEnd;

			if (offset > scrollEnd + (overscan * this.options.estimateSize(0))) {
				end = i;
				break;
			}
		}

		if (end === 0) {
			end = count - 1;
		}

		return items;
	}

	// Get scroll offset
	getScrollOffset(): number {
		return this.state.scrollOffset;
	}

	// Scroll to a specific offset
	scrollToOffset(offset: number, options?: ScrollToOptions) {
		if (!this.scrollElement) return;

		this.scrollElement.scrollTo({
			top: offset,
			behavior: options?.behavior ?? 'auto'
		});
	}

	// Scroll to a specific index
	scrollToIndex(index: number, options?: { align?: 'start' | 'center' | 'end' | 'auto'; behavior?: ScrollBehavior }) {
		if (!this.scrollElement) return;

		let offset = this.options.paddingStart || 0;
		for (let i = 0; i < index; i++) {
			offset += this.getSize(i);
		}

		const align = options?.align ?? 'start';
		const size = this.getSize(index);
		const containerHeight = this.scrollElement.clientHeight;

		if (align === 'center') {
			offset = offset - containerHeight / 2 + size / 2;
		} else if (align === 'end') {
			offset = offset - containerHeight + size;
		}

		this.scrollToOffset(Math.max(0, offset), { behavior: options?.behavior });
	}
}

/**
 * Create a virtualizer instance
 */
export function createVirtualizer(options: VirtualizerOptions) {
	const virtualizer = new Virtualizer(options);
	
	return {
		virtualizer,
		getVirtualItems: () => virtualizer.getVirtualItems(),
		getTotalSize: () => virtualizer.getTotalSize(),
		getScrollOffset: () => virtualizer.getScrollOffset(),
		scrollToOffset: (offset: number, opts?: ScrollToOptions) => virtualizer.scrollToOffset(offset, opts),
		scrollToIndex: (index: number, opts?: { align?: 'start' | 'center' | 'end' | 'auto'; behavior?: ScrollBehavior }) => 
			virtualizer.scrollToIndex(index, opts),
		measureElement: (element: HTMLElement | null, index: number) => virtualizer.measureElement(element, index),
		init: () => virtualizer.init(),
		destroy: () => virtualizer.destroy()
	};
}

