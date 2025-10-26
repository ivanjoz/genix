<script lang="ts" module>
	type CSSProperties = Record<string, string | number>;

	export interface ICellEditableProps<T> {
		saveOn: T;
		save: string;
		class?: string;
		contentClass?: string;
		inputClass?: string;
		onChange?: (newValue: string | number) => void;
		render?: (value: number | string, isEditing: boolean) => any;
		getValue?: (e: T) => number | string;
		required?: boolean;
		type?: string;
	}
</script>

<script lang="ts" generics="T">
	let {
		saveOn,
		save,
		class: className = '',
		contentClass = '',
		inputClass = '',
		onChange,
		render,
		getValue,
		required = false,
		type = 'text'
	}: ICellEditableProps<T> = $props();

	let isEditing = $state(false);
	let inputRef = $state<HTMLInputElement>();

	const initialValue = getValue
		? getValue(saveOn)
		: ((saveOn || ({} as T))[save as keyof T] as number | string);

	let currentValue = $state(initialValue);
	let prevValue = $state(initialValue);

	// Focus input when editing starts
	$effect(() => {
		if (isEditing && inputRef) {
			inputRef.focus();
		}
	});

	function extractValue(newValue?: string | number): string | number {
		if (type === 'number') {
			return parseFloat((newValue as string) || '0');
		}
		return newValue as string | number;
	}

	function reSetValue(newValue?: string | number) {
		const extracted = extractValue(newValue);
		(saveOn as any)[save as keyof T] = extracted;
		currentValue = extracted;
	}

	function handleClick(ev: MouseEvent) {
		ev.stopPropagation();
		prevValue = currentValue;
		isEditing = true;
	}

	function handleKeyUp(ev: KeyboardEvent) {
		ev.stopPropagation();
		reSetValue((ev.target as HTMLInputElement).value);
	}

	function handleBlur(ev: FocusEvent) {
		ev.stopPropagation();

		const newValue = extractValue((ev.target as HTMLInputElement).value);
		if (prevValue !== newValue) {
			prevValue = newValue;
			if (onChange) {
				onChange(newValue);
			}
		}
		isEditing = false;
	}

	const renderContent = $derived(render ? render(currentValue, isEditing) : currentValue);
</script>

<div class={`cell-ed-c flex ai-center p-rel ${className}`}>
	<div
		class={`h100 w100 ${contentClass}`}
		style:visibility={isEditing ? 'hidden' : 'visible'}
		onclick={handleClick}
		role="button"
		tabindex="0"
		onkeydown={(ev) => {
			if (ev.key === 'Enter' || ev.key === ' ') {
				ev.preventDefault();
				handleClick(ev as any);
			}
		}}
	>
		{renderContent}
		{#if !currentValue && required}
			<i class="icon-attention c-red"></i>
		{/if}
	</div>
	{#if isEditing}
		<div class="flex ai-center cell-ed h100 w100">
			<input
				bind:this={inputRef}
				value={currentValue || ''}
				{type}
				class={`w100 ${inputClass}`}
				onkeyup={handleKeyUp}
				onblur={handleBlur}
			/>
		</div>
	{/if}
</div>

<style>
	.cell-ed-c {
		position: relative;
		display: flex;
		align-items: center;
		width: 100%;
		height: 100%;
		padding: 0.5rem 0.75rem;
	}

	.cell-ed {
		position: absolute;
		top: 0;
		left: 0;
		padding: 0.5rem 0.75rem;
	}

	.cell-ed input {
		padding: 0.25rem 0.5rem;
		border: 1px solid #4042a3;
		border-radius: 4px;
		outline: none;
		font-size: inherit;
		font-family: inherit;
	}

	.flex {
		display: flex;
	}

	.ai-center {
		align-items: center;
	}

	.p-rel {
		position: relative;
	}

	.h100 {
		height: 100%;
	}

	.w100 {
		width: 100%;
	}

	.c-red {
		color: #dc3545;
	}
</style>

