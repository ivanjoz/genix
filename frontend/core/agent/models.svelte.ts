import { browser } from '$app/environment';
import { Env } from '$core/env';
import { GetHandler } from '$libs/http.svelte';

export interface IAgentModelOption {
	ID: string;
	Hash: string;
	Notes: string;
}

const storageKey = () => `${Env.appId}AgentModelHash:${Env.enviroment || 'main'}`;

export const getSelectedAgentModelHash = (): string => {
	if (!browser) { return ''; }
	return localStorage.getItem(storageKey()) || '';
};

export const setSelectedAgentModelHash = (modelHash: string) => {
	if (!browser) { return; }
	if (modelHash) {
		localStorage.setItem(storageKey(), modelHash);
	} else {
		localStorage.removeItem(storageKey());
	}
	console.debug('[AgentModels] selected model hash changed.', { modelHash });
};

export class AgentModelsService extends GetHandler {
	route = 'agent-models';
	useCache = { min: 10, ver: 1 };

	records = $state<IAgentModelOption[]>([]);
	modelHashMap = $state(new Map<string, IAgentModelOption>());

	handler(result: IAgentModelOption[]): void {
		const models = Array.isArray(result) ? result : [];
		this.records = models;
		this.modelHashMap = new Map(models.map((modelOption) => [modelOption.Hash, modelOption]));
		console.debug('[AgentModels] models loaded.', {
			count: models.length,
			hashes: models.map((modelOption) => modelOption.Hash),
		});
	}

	constructor() {
		super();
		this.fetch();
	}
}
