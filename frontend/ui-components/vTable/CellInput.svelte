<script lang="ts" module>
import type { ElementAST } from '$components/misc/Renderer.svelte';
import Renderer from '$components/misc/Renderer.svelte';
import { getVTableAgentContext } from '$components/vTable/agentContext';

	export interface ICellInputProps<T> {
		saveOn?: T;
		save?: string;
		css?: string;
		contentClass?: string;
		inputClass?: string;
		onChange?: (newValue: string | number) => void;
		onBeforeCellChange?: (newValue: string | number) => boolean;
		render?: (value: number | string) => string | ElementAST | ElementAST[];
		getValue?: (e: T) => number | string;
		required?: boolean;
		type?: string;
		// Cell coordinate within its parent table; the table assigns it as
		// (rowIndex+1)*100 + (columnIndex + 1) so cells never collide with row
		// IDs (multiples of 100) and the lowest cell id is 101 — ids of 0 read
		// as "missing" through too many code paths.
		cellID?: number;
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
		onBeforeCellChange,
		render,
		getValue,
		required = false,
		type = 'text',
		cellID,
	}: ICellInputProps<T> = $props();

	let isEditing = $state(false);
	let inputRef = $state<HTMLInputElement>();

	const GetValue = () => {
		return getValue ? getValue(saveOn as T) : ((saveOn || ({} as T))[save as keyof T] as number | string)
	}
	
	let currentValue = $state<number | string>(GetValue());

	$effect(() => {
		if (isEditing) return;
		const freshValue = GetValue()
		if (currentValue !== freshValue){ currentValue = freshValue }
	});
	
	const renderedContent = $derived(render ? render(currentValue) : null);

	// Focus input when editing starts
	$effect(() => {
		if (isEditing && inputRef) { 	inputRef.focus();	}
	});

	const extractValue = (newValue?: string | number): string | number  => {
		if (type === 'number') {
			return parseFloat((newValue as string) || '0');
		}
		return newValue as string | number;
	}

	const handleClick = (ev: MouseEvent) => {
		ev.stopPropagation();
		// Always hydrate the input from getValue so edit mode uses the raw persisted value.
		currentValue = getValue
			? getValue(saveOn as T)
			: ((saveOn || ({} as T))[save as keyof T] as number | string);
		isEditing = true;
	}

	const handleBlur = (ev: FocusEvent) => {
		ev.stopPropagation()
		const newValue = extractValue((ev.target as HTMLInputElement).value);
		if (currentValue !== newValue) {
			if (onBeforeCellChange && onBeforeCellChange(newValue) === false) {
				isEditing = false;
				return;
			}
			if (onChange) { onChange(newValue); }
			currentValue = newValue
		}
		isEditing = false;
	}

	// Apply a value the same way blur does: parse, run the guard, fire onChange.
	const applyAgentValue = (value: string | number) => {
		const next = extractValue(value);
		if (currentValue === next) { return; }
		if (onBeforeCellChange && onBeforeCellChange(next) === false) { return; }
		if (onChange) { onChange(next); }
		currentValue = next;
	}

	// Hand setValue to the parent table; the table is the only Agent handle.
	const tableAgentCtx = getVTableAgentContext();

	$effect(() => {
		if (!tableAgentCtx || cellID === undefined) { return; }
		return tableAgentCtx.registerCell(cellID, {
			setValue: (value) => { applyAgentValue(value); },
		});
	});

	// data-value mirrors the visible cell value so the agent can read it from
	// the DOM snapshot without an extra round trip.
	const agentDataValue = $derived(
		currentValue === undefined || currentValue === null || currentValue === ''
			? ''
			: String(currentValue),
	);
	const agentDataType = $derived(type === 'number' ? 'number' : (type === 'text' ? 'text' : 'other'));
	// Composite addressing: "<tableID>:<cellID>". Skipped when there's no
	// parent table context (component used standalone).
	const agentDataID = $derived(
		tableAgentCtx && cellID !== undefined ? `${tableAgentCtx.tableID}:${cellID}` : undefined,
	);
</script>

<div class="_root"
	data-id={agentDataID}
	data-cell-type={agentDataID ? 'CellInput' : undefined}
	data-value={agentDataValue}
	data-label={String(save || '')}
	data-type={agentDataType}
>
<div class="_2 {contentClass}">
	{#if typeof renderedContent === 'string'}
		{renderedContent}
	{:else if renderedContent}
		<Renderer elements={renderedContent}/>
	{:else}
		{currentValue}
	{/if}
</div>
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
		{#if typeof renderedContent === 'string'}
			{renderedContent}
		{:else if renderedContent}
			<Renderer elements={renderedContent}/>
		{:else}
			{currentValue}
		{/if}
		{#if !currentValue && required}
			<i class="icon-attention text-red-500"></i>
		{/if}
	</div>
	{#if isEditing}
		<input bind:this={inputRef} {type}
			value={currentValue || ''}
			class={`w-full ${inputClass}`}
			onblur={handleBlur}
		/>
	{/if}
</div>
</div>

<style>
	._root {
		display: contents;
	}
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
   /* padding: 0 6px; */
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

	.text-red-500 {
		color: #dc3545;
	}
</style>
