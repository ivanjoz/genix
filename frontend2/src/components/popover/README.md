# Popover Component

A custom popover component inspired by Skeleton UI, built with Svelte 5 and Zag.js patterns.

## Recent Fixes (Oct 2025)

The following issues were fixed to make the popover fully functional:

1. **Reactive State**: Changed from plain object to Svelte 5 `$state()` for proper reactivity
2. **Context System**: Replaced global context with Svelte's `setContext`/`getContext` API
3. **Element References**: Implemented `bind:this` instead of React-style `ref` props
4. **Positioning**: Added dynamic position calculation based on trigger element
5. **Click Outside**: Added click outside handler to close the popover
6. **Multiple Instances**: Fixed issue where multiple popovers would share the same state

## Installation

The popover component is already installed and available in the components folder. Make sure to import the CSS:

```svelte
<script>
	import { Popover } from './components/popover';
	import './components/popover/popover.css';
</script>
```

## Basic Usage

```svelte
<script>
	import { Popover } from '$lib/components/popover';
</script>

<Popover.Root>
	<Popover.Trigger>
		Click me!
	</Popover.Trigger>

	<Popover.Positioner>
		<Popover.Content>
			<Popover.Title>Popover Title</Popover.Title>
			<Popover.Description>
				Your popover content goes here.
			</Popover.Description>

			<Popover.CloseTrigger>
				✕
			</Popover.CloseTrigger>
		</Popover.Content>
	</Popover.Positioner>
</Popover.Root>
```

## Components

### Popover.Root
The main container component that manages the popover state.

**Props:**
- `open?: boolean` - Controls the open state
- `onOpenChange?: (open: boolean) => void` - Callback when open state changes

### Popover.Trigger
The element that triggers the popover when clicked.

### Popover.Positioner
Positions the popover content relative to the trigger.

### Popover.Content
The main content area of the popover.

### Popover.Arrow
Optional arrow that points to the trigger element.

### Popover.ArrowTip
The tip of the arrow (usually styled differently from the base).

### Popover.Title
A title element for the popover content.

### Popover.Description
A description element for the popover content.

### Popover.CloseTrigger
An optional close button for the popover.

## Styling

The popover uses CSS custom properties for theming:

```css
:root {
	--popover-bg: white;
	--popover-border: #e5e7eb;
	--popover-shadow: 0 10px 15px -3px rgb(0 0 0 / 0.1);
	--popover-radius: 6px;
	--popover-padding: 0.75rem;
}
```

## Example with Custom Styling

```svelte
<Popover.Root>
	<Popover.Trigger class="my-custom-trigger">
		Custom Trigger
	</Popover.Trigger>

	<Popover.Positioner>
		<Popover.Content class="my-custom-content">
			<Popover.Arrow>
				<Popover.ArrowTip />
			</Popover.Arrow>

			<Popover.Title>My Custom Title</Popover.Title>
			<Popover.Description>
				Custom styled popover content.
			</Popover.Description>
		</Popover.Content>
	</Popover.Positioner>
</Popover.Root>
```

## Features

- ✅ Fully accessible
- ✅ Keyboard navigation support
- ✅ Click outside to close
- ✅ Customizable styling
- ✅ TypeScript support
- ✅ Svelte 5 compatible
- ✅ Modular component architecture
