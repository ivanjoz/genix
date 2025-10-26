# Svelte 5 Virtual Scroller

A lightweight virtual scrolling library built specifically for Svelte 5 using runes.

## Features

- ✅ Built for Svelte 5 with runes (`$state`, `$derived`, `$effect`)
- ✅ Lightweight and performant
- ✅ Supports large datasets (50k+ items)
- ✅ Configurable overscan for smooth scrolling
- ✅ Reactive and type-safe

## Installation

The library is already part of your project in `src/lib/virtualizer/`.

## Basic Usage

```svelte
<script lang="ts">
	import { createVirtualizer } from '$lib/virtualizer/index.svelte';

	let data = $state([/* your data */]);
	let containerRef = $state<HTMLDivElement>();
	
	let virtualizerStore = $state<ReturnType<typeof createVirtualizer> | null>(null);
	
	let virtualItems = $derived.by(() => {
		if (!containerRef) return [];
		return virtualizerStore?.getVirtualItems() || [];
	});
	
	let totalSize = $derived.by(() => {
		if (!containerRef) return 0;
		return virtualizerStore?.getTotalSize() || 0;
	});

	$effect(() => {
		if (containerRef) {
			virtualizerStore = createVirtualizer({
				count: data.length,
				getScrollElement: () => containerRef!,
				estimateSize: () => 40, // estimated row height
				overscan: 5 // render 5 extra items above and below
			});

			const unsubscribe = virtualizerStore.subscribe(() => {
				// Triggers reactivity on scroll
			});

			return unsubscribe;
		}
	});
</script>

<div bind:this={containerRef} class="container">
	<table>
		<tbody>
			{#each virtualItems as row (row.index)}
				{@const item = data[row.index]}
				{@const firstItemStart = virtualItems[0]?.start || 0}
				<tr style="height: {row.size}px; transform: translateY({firstItemStart}px);">
					<td>{item.name}</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>

<style>
	.container {
		overflow: auto;
		max-height: 600px;
	}
</style>
```

## API

### `createVirtualizer(options)`

Creates a virtualizer instance.

**Options:**
- `count`: Total number of items
- `getScrollElement`: Function that returns the scrollable container element
- `estimateSize`: Function that returns estimated item height (can be dynamic per index)
- `overscan`: Number of extra items to render above/below viewport (default: 3)

**Returns:**
- `subscribe(callback)`: Subscribe to scroll changes
- `getVirtualItems()`: Get array of virtual items to render
- `getTotalSize()`: Get total size of all items

### `VirtualItem`

Each virtual item has:
- `index`: Item index in the data array
- `start`: Start position (px) in the virtual space
- `size`: Height of the item (px)
- `end`: End position (px) in the virtual space
- `key`: Unique key for the item

## Why Not TanStack Virtual?

@tanstack/svelte-virtual is built for Svelte 4 and doesn't fully leverage Svelte 5's new reactivity system. This library:
- Uses Svelte 5 runes (`$state`, `$derived`, `$effect`)
- Simpler API tailored for Svelte 5
- Smaller bundle size
- Better TypeScript integration

## Performance Tips

1. Use `estimateSize` accurately - closer estimates = better performance
2. Adjust `overscan` based on your needs (higher = smoother scroll, more items rendered)
3. Use `{@const}` in templates to avoid recalculations
4. Keep row heights consistent when possible

## License

MIT

