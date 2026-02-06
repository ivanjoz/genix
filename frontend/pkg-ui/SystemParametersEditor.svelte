<script lang="ts">
	import { systemParameters } from '../pkg-services/system-paremeters';
	import { SystemParametersService, saveSystemParameters, type ISystemParameter } from '../pkg-services/services/system-parameters.svelte';

	import { Loading } from '$core/helpers';
    import SearchCard from '$components/SearchCard.svelte';
    import Checkbox from '$components/Checkbox.svelte';
    import { untrack } from 'svelte';

	const service = new SystemParametersService();

	// We use an object where keys are ParameterID and values are the record data
	// This allows us to use the saveOn={form[id]} pattern
	let form = $state({} as Record<number, ISystemParameter>);

	// Initialize form when records are loaded or changed
	$effect(() => {		
		if(!service.isReady){ return }
		untrack(() => {
			const newForm = { ...form };
			for (const config of systemParameters) {
				const record = service.recordsMap.get(config.id);
				if (record) {
					newForm[config.id] = { ...record };
				} else if (!newForm[config.id]) {
					newForm[config.id] = {
						ParameterID: config.id,
						ValueText: "",
						ValueInts: [],
						Value: 0,
						Updated: 0,
						UpdatedBy: 0
					} as ISystemParameter;
				}
			}
			form = newForm;
		})
	});

	async function save() {
		Loading.standard("Guardando...");
		const toSave = Object.values(form);

		try {
			await saveSystemParameters(toSave);
		} catch (e) {
			console.error("Error saving parameters", e);
		} finally {
			Loading.remove();
		}
	}
</script>

<div class="p-24 space-y-24 max-w-600 bg-white rounded-xl shadow-sm border border-gray-100">
	<div class="flex items-center justify-between mb-24 pb-16 border-b border-gray-100">
		<div>
			<h2 class="text-xl font-bold text-gray-800">Configuración de Ventas</h2>
			<p class="text-sm text-gray-500 mt-2">Personaliza el comportamiento del módulo de ventas</p>
		</div>
		<button class="bx-blue py-10 px-20 flex items-center gap-8 shadow-sm hover:shadow-md transition-all" onclick={save}>
			<span>Guardar</span>
			<i class="icon-floppy"></i>
		</button>
	</div>

	<div class="space-y-20">
		{#each systemParameters as param}
			{#if form[param.id]}
				<div class="bg-gray-50/50 p-20 rounded-xl border border-gray-100 hover:border-blue-100 transition-colors group">
					<div class="pl-4">
						{#if param.type === 'multiselect'}
							<SearchCard 
								options={param.options || []}
								keyId="id"
								keyName="name"
								label={param.name}
								bind:saveOn={form[param.id]}
								save="ValueInts"
								cardCss="bg-white"
							/>
						{:else if param.type === 'checkbox'}
							<Checkbox useNumber
								label={param.name}
								bind:saveOn={form[param.id]}
								save="Value"
								css="py-8 font-bold text-gray-700"
							/>
						{/if}
					</div>
				</div>
			{/if}
		{/each}
	</div>
</div>
