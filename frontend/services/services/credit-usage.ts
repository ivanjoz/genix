import { GET } from '$libs/ui-runtime.svelte';

export interface ICreditUsageDay {
	Day: number;
	CPU: number;
	Inference: number;
}

export interface ICreditUsageScope {
	CPU24hLimit: number;
	Inference24hLimit: number;
	Days: ICreditUsageDay[];
}

export interface ICreditUsageResponse {
	User: ICreditUsageScope;
	Company: ICreditUsageScope;
}

export const getCreditUsage = async (): Promise<ICreditUsageResponse> => {
	// Usage snapshots change every 15 seconds, so this report intentionally bypasses delta caching.
	return GET({ route: 'credit-usage' });
};
