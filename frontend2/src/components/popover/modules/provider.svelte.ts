import type { Api, Props } from '@zag-js/popover';
import type { PropTypes } from '@zag-js/svelte';

export function usePopover(props: Props | (() => Props)): () => Api<PropTypes> {
	const propsValue = typeof props === 'function' ? props() : props;
	
	// Create reactive state for each popover instance
	let open = $state(propsValue.open || propsValue.defaultOpen || false);
	let triggerElement = $state<HTMLElement | null>(null);
	let positionerElement = $state<HTMLElement | null>(null);

	// Calculate position based on trigger element
	const calculatePosition = () => {
		if (!triggerElement || !positionerElement) return { top: 0, left: 0 };
		
		const triggerRect = triggerElement.getBoundingClientRect();
		const positionerRect = positionerElement.getBoundingClientRect();
		
		// Position below the trigger by default
		const top = triggerRect.bottom + 8; // 8px offset
		const left = triggerRect.left + (triggerRect.width / 2) - (positionerRect.width / 2);
		
		return { top, left };
	};

	return () => ({
		get open() {
			return open;
		},
		get triggerElement() {
			return triggerElement;
		},
		get positionerElement() {
			return positionerElement;
		},
		setTriggerElement: (el: HTMLElement | null) => {
			triggerElement = el;
		},
		setPositionerElement: (el: HTMLElement | null) => {
			positionerElement = el;
		},
		calculatePosition,
		onOpenChange: (details: { open: boolean }) => {
			open = details.open;
			if (propsValue.onOpenChange) {
				propsValue.onOpenChange(details);
			}
		},
		getTriggerProps: () => ({
			'onclick': () => {
				const newOpen = !open;
				open = newOpen;
				if (propsValue.onOpenChange) {
					propsValue.onOpenChange({ open: newOpen });
				}
			},
		}),
		getPositionerProps: () => ({}),
		getContentProps: () => ({}),
		getArrowProps: () => ({}),
		getArrowTipProps: () => ({}),
		getTitleProps: () => ({}),
		getDescriptionProps: () => ({}),
		getCloseTriggerProps: () => ({
			'onclick': () => {
				open = false;
				if (propsValue.onOpenChange) {
					propsValue.onOpenChange({ open: false });
				}
			},
		}),
	});
}
