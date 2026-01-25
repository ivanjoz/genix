export type Placement = 'top' | 'bottom' | 'left' | 'right' | 'top-start' | 'top-end' | 'bottom-start' | 'bottom-end' | 'left-start' | 'left-end' | 'right-start' | 'right-end';

export interface PositionResult {
	top: number;
	left: number;
	placement: Placement;
	maxHeight?: number;
	maxWidth?: number;
}

export interface PositionOptions {
	offset?: number;
	preferredPlacement?: Placement;
	fitViewport?: boolean;
}

/**
 * Calculate the best position for a floating element relative to a reference element
 * with smart collision detection and viewport awareness
 */
export function calculatePosition(
	referenceElement: HTMLElement,
	floatingElement: HTMLElement,
	options: PositionOptions = {}
): PositionResult {
	const {
		offset = 8,
		preferredPlacement = 'bottom',
		fitViewport = true
	} = options;

	const refRect = referenceElement.getBoundingClientRect();
	const floatRect = floatingElement.getBoundingClientRect();
	const viewportWidth = window.innerWidth;
	const viewportHeight = window.innerHeight;
	const scrollX = window.scrollX || window.pageXOffset;
	const scrollY = window.scrollY || window.pageYOffset;

	// Calculate available space in each direction
	const spaces = {
		top: refRect.top,
		bottom: viewportHeight - refRect.bottom,
		left: refRect.left,
		right: viewportWidth - refRect.right,
	};

	// Determine which placements fit
	const fits = {
		top: spaces.top >= floatRect.height + offset,
		bottom: spaces.bottom >= floatRect.height + offset,
		left: spaces.left >= floatRect.width + offset,
		right: spaces.right >= floatRect.width + offset,
	};

	// Get base placement (without -start/-end)
	const basePlacement = preferredPlacement.split('-')[0] as 'top' | 'bottom' | 'left' | 'right';

	// Try preferred placement first, then fallback to best available
	let finalPlacement: Placement = preferredPlacement;
	let baseFinalPlacement: 'top' | 'bottom' | 'left' | 'right' = basePlacement;

	if (!fits[basePlacement]) {
		// Find the placement with the most space
		if (basePlacement === 'bottom' || basePlacement === 'top') {
			// Was trying vertical, check if we should flip
			if (basePlacement === 'bottom' && fits.top) {
				baseFinalPlacement = 'top';
			} else if (basePlacement === 'top' && fits.bottom) {
				baseFinalPlacement = 'bottom';
			} else {
				// Try horizontal placements
				if (fits.right) baseFinalPlacement = 'right';
				else if (fits.left) baseFinalPlacement = 'left';
				else baseFinalPlacement = spaces.bottom >= spaces.top ? 'bottom' : 'top';
			}
		} else {
			// Was trying horizontal, check if we should flip
			if (basePlacement === 'right' && fits.left) {
				baseFinalPlacement = 'left';
			} else if (basePlacement === 'left' && fits.right) {
				baseFinalPlacement = 'right';
			} else {
				// Try vertical placements
				if (fits.bottom) baseFinalPlacement = 'bottom';
				else if (fits.top) baseFinalPlacement = 'top';
				else baseFinalPlacement = spaces.right >= spaces.left ? 'right' : 'left';
			}
		}
		finalPlacement = baseFinalPlacement;
	}

	// Calculate base position
	let top = 0;
	let left = 0;

	switch (baseFinalPlacement) {
		case 'bottom':
			top = refRect.bottom + offset;
			left = refRect.left + (refRect.width / 2) - (floatRect.width / 2);
			break;
		case 'top':
			top = refRect.top - floatRect.height - offset;
			left = refRect.left + (refRect.width / 2) - (floatRect.width / 2);
			break;
		case 'right':
			top = refRect.top + (refRect.height / 2) - (floatRect.height / 2);
			left = refRect.right + offset;
			break;
		case 'left':
			top = refRect.top + (refRect.height / 2) - (floatRect.height / 2);
			left = refRect.left - floatRect.width - offset;
			break;
	}

	// Handle alignment (-start, -end)
	if (preferredPlacement.includes('-')) {
		const alignment = preferredPlacement.split('-')[1] as 'start' | 'end';
		
		if (baseFinalPlacement === 'top' || baseFinalPlacement === 'bottom') {
			if (alignment === 'start') {
				left = refRect.left;
			} else if (alignment === 'end') {
				left = refRect.right - floatRect.width;
			}
		} else {
			if (alignment === 'start') {
				top = refRect.top;
			} else if (alignment === 'end') {
				top = refRect.bottom - floatRect.height;
			}
		}
	}

	// Add scroll offset for absolute positioning
	top += scrollY;
	left += scrollX;

	// Constrain to viewport if enabled
	if (fitViewport) {
		const minLeft = offset + scrollX;
		const maxLeft = viewportWidth - floatRect.width - offset + scrollX;
		const minTop = offset + scrollY;
		const maxTop = viewportHeight - floatRect.height - offset + scrollY;

		left = Math.max(minLeft, Math.min(left, maxLeft));
		top = Math.max(minTop, Math.min(top, maxTop));
	}

	// Calculate max dimensions to fit in viewport
	const maxHeight = viewportHeight - offset * 2;
	const maxWidth = viewportWidth - offset * 2;

	return {
		top: Math.round(top),
		left: Math.round(left),
		placement: finalPlacement,
		maxHeight,
		maxWidth,
	};
}

/**
 * Detect if an element would overflow the viewport at a given position
 */
export function detectOverflow(
	element: HTMLElement,
	position: { top: number; left: number }
): {
	overflowTop: number;
	overflowBottom: number;
	overflowLeft: number;
	overflowRight: number;
} {
	const rect = element.getBoundingClientRect();
	const viewportWidth = window.innerWidth;
	const viewportHeight = window.innerHeight;

	return {
		overflowTop: Math.max(0, -position.top),
		overflowBottom: Math.max(0, position.top + rect.height - viewportHeight),
		overflowLeft: Math.max(0, -position.left),
		overflowRight: Math.max(0, position.left + rect.width - viewportWidth),
	};
}

