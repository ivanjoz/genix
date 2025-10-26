import { getContext, setContext } from 'svelte';

const POPOVER_ROOT_KEY = Symbol('popover-root');

export const PopoverRootContext = {
	key: POPOVER_ROOT_KEY,

	provide(value: () => any) {
		setContext(POPOVER_ROOT_KEY, value);
		return value;
	},

	consume() {
		const context = getContext<(() => any) | undefined>(POPOVER_ROOT_KEY);
		if (!context) {
			throw new Error('PopoverRootContext must be provided before consuming. Make sure to wrap your popover components in a Popover.Root component.');
		}
		return context;
	}
};
