<script lang="ts" module>
    import type { ElementAST } from './micro/Renderer.svelte';
    import Renderer from './micro/Renderer.svelte';


	export interface ICellEditableProps<T> {
		saveOn: T;
		save?: string;
		css?: string;
		contentClass?: string;
		inputClass?: string;
		onChange?: (newValue: string | number) => void;
		render?: (value: number | string) => ElementAST[];
		getValue?: (e: T) => number | string;
		required?: boolean;
		type?: string;
	}
</script>

<script lang="ts" generics="T">
	let {
		saveOn = $bindable(),
		save,
		css,
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
		console.log("currentValue", currentValue)
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

</script>

<div class="_2">{currentValue}</div>
<div class="_1 {css}">
	<div class="{contentClass}"
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
		{#if render}
			<Renderer elements={render(currentValue)}/>
		{:else}
			{currentValue}
		{/if}
		{#if !currentValue && required}
			<i class="icon-attention c-red"></i>
		{/if}
	</div>
	{#if isEditing}
		<input bind:this={inputRef} {type}
			value={currentValue || ''}
			class={`w-full ${inputClass}`}
			onkeyup={handleKeyUp}
			onblur={handleBlur}
		/>
	{/if}
</div>

<style>
	._2 {
		opacity: 0;
		pointer-events: none;
	}
	._1 {
    position: absolute;
    top: 0;
    left: 0;
    display: flex;
    align-items: center;
    padding: 0 6px;
    width: 100%;
    height: 100%;
    border: 1px solid transparent;
	}
	._1 > div:first-of-type {
		height: 100%;
		width: 100%;
		display: flex;
		align-items: center;
	}
	._1 > input:first-of-type {
		border: none;
		outline: none;
		position: absolute;
		top: 0;
		left: 0;
		height: 100%;
		width: 100%;
		padding-left: 6px;
	}

	._1:hover {
		border: 1px solid rgba(0, 0, 0, 0.596);
	}
	._1:focus-within {
		box-shadow: inset 0 0 0px 1px #dbc1ff;
    border-color: #b17bff;
		background-color: #f9f4ff;
	}

	.ai-center {
		align-items: center;
	}

	.p-rel {
		position: relative;
	}

	.c-red {
		color: #dc3545;
	}
</style>

