# Popover2 Library

A lightweight, smart positioning library for Svelte 5 that renders floating elements in the document body with automatic collision detection and viewport awareness.

## Features

✅ **True Portal Rendering** - Actually renders in document.body, escapes overflow:hidden  
✅ **Smart Positioning** - Automatically finds the best placement based on available space  
✅ **Collision Detection** - Flips placement when there's not enough room  
✅ **Viewport Awareness** - Stays within viewport boundaries  
✅ **Scroll & Resize Handling** - Updates position dynamically  
✅ **Works with Overflow Hidden** - Popover appears even if parent has overflow:hidden  
✅ **Z-Index Freedom** - No stacking context issues  
✅ **TypeScript Support** - Fully typed with TypeScript  
✅ **Lightweight** - No external dependencies (except Svelte)  
✅ **CSS Customizable** - Easy to style and theme  
✅ **Pure Svelte 5 Runes** - Modern Svelte 5 implementation  

## Installation

The library is already in your components folder. Just import it:

```svelte
<script>
  import { Popover2 } from '$lib/components/popover2';
  import '$lib/components/popover2/popover2.css';
</script>
```

## Basic Usage

```svelte
<script lang="ts">
  import { Popover2 } from './components/popover2';
  import './components/popover2/popover2.css';
  
  let buttonElement: HTMLElement | null = $state(null);
  let showPopover = $state(false);
</script>

<button bind:this={buttonElement} onclick={() => showPopover = !showPopover}>
  Click me
</button>

<Popover2
  referenceElement={buttonElement}
  open={showPopover}
  placement="bottom"
>
  <div class="popover2-container">
    <div class="popover2-content">
      Your content here!
    </div>
  </div>
</Popover2>
```

## API Reference

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `referenceElement` | `HTMLElement \| null` | **required** | The element to position relative to |
| `open` | `boolean` | `false` | Whether the popover is visible |
| `placement` | `Placement` | `'bottom'` | Preferred placement (will auto-adjust) |
| `offset` | `number` | `8` | Distance from reference element in pixels |
| `fitViewport` | `boolean` | `true` | Constrain to viewport boundaries |
| `class` | `string` | `''` | Custom class for the container |
| `style` | `string` | `''` | Custom inline styles |
| `onPositionUpdate` | `function` | `undefined` | Callback when position changes |

### Placement Options

- `top`, `bottom`, `left`, `right` - Basic placements
- `top-start`, `top-end` - Top with left/right alignment
- `bottom-start`, `bottom-end` - Bottom with left/right alignment
- `left-start`, `left-end` - Left with top/bottom alignment
- `right-start`, `right-end` - Right with top/bottom alignment

## Examples

### Auto Placement with Callback

```svelte
<script>
  let currentPlacement = $state('bottom');
</script>

<Popover2
  referenceElement={buttonElement}
  open={showPopover}
  placement="bottom"
  onPositionUpdate={(pos) => currentPlacement = pos.placement}
>
  <div class="popover2-container">
    Currently placed: {currentPlacement}
  </div>
</Popover2>
```

### Custom Styling

```svelte
<Popover2
  referenceElement={buttonElement}
  open={showPopover}
  class="my-custom-popover"
  style="background: red; padding: 20px;"
>
  <div>Custom styled content</div>
</Popover2>
```

### Different Offsets

```svelte
<!-- Close to trigger -->
<Popover2 referenceElement={el} open={true} offset={4}>
  ...
</Popover2>

<!-- Far from trigger -->
<Popover2 referenceElement={el} open={true} offset={20}>
  ...
</Popover2>
```

## How It Works

1. **Portal Rendering**: 
   - Renders content normally in Svelte's component tree
   - Uses `$effect()` to move the actual DOM node to `document.body`
   - This escapes `overflow: hidden` containers and z-index stacking contexts
   - Content is properly managed by Svelte but positioned globally

2. **Position Calculation**: 
   - Uses `getBoundingClientRect()` to calculate the best position
   - Accounts for scroll position and viewport boundaries

3. **Collision Detection**: 
   - Checks viewport space in all 4 directions
   - Flips placement when there's insufficient space
   - Ensures popover never goes off-screen

4. **Dynamic Updates**: 
   - Listens to scroll and resize events
   - Recalculates position in real-time
   - Smooth repositioning as content changes

5. **Cleanup**: 
   - Effect cleanup removes element from body
   - Proper memory management and no leaks

## Positioning Algorithm

The library follows this priority order:

1. Try preferred placement
2. If doesn't fit, try opposite direction
3. If still doesn't fit, try perpendicular directions
4. Choose direction with most available space
5. Adjust position to stay within viewport

## Styling

The library includes base styles in `popover2.css`:

- White background with border
- Shadow for depth
- Arrow indicators based on placement
- Dark mode support
- Smooth fade-in animation

You can override any styles by:
1. Adding custom classes via the `class` prop
2. Using the `style` prop for inline styles
3. Targeting `.popover2-container` in your CSS

## Browser Support

Works in all modern browsers that support:
- Svelte 5
- `getBoundingClientRect()`
- CSS custom properties
- ES6+

## Comparison with Popover (v1)

| Feature | Popover (v1) | Popover2 |
|---------|-------------|----------|
| Portal rendering | ❌ | ✅ |
| Smart collision detection | Basic | Advanced |
| Context API | ✅ | ❌ (simpler) |
| Built-in trigger | ✅ | ❌ (more flexible) |
| Component structure | Complex | Simple |
| Bundle size | Larger | Smaller |

## License

MIT

## Contributing

Feel free to extend this library with additional features like:
- Animation options
- Focus management
- Keyboard navigation
- Custom arrow components
- Multiple reference elements

