// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		interface PageState {
			// libs/back-steps.svelte.ts: how many back steps the current history entry has open.
			backStepDepth?: number
		}
		// interface Platform {}
	}
}

export {};
