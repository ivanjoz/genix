<script lang="ts">
  import { getContext, type Snippet } from 'svelte';
  import type { ComponentAST } from '$ecommerce/renderer/renderer-types';
  import { slideSync } from '$ecommerce/stores/slide-sync.svelte';
  import { EC_BUILDER_MODE } from '$ecommerce/renderer/builder-context';

  interface Props {
    /**
     * The direct child AST nodes of the `<Slider>` tag — one slide each. Injected
     * by AstRenderer (not an HTML attribute); see `childNodes`/`renderChild` there.
     */
    childNodes?: ComponentAST[];
    /** AstRenderer's node renderer, so a slide subtree renders without importing it. */
    renderChild?: Snippet<[ComponentAST]>;
    /** Advance slides automatically. */
    autoplay?: boolean;
    /** Autoplay delay in milliseconds. */
    interval?: number;
    /** Wrap from last slide back to first (and vice-versa). */
    loop?: boolean;
    /** Show the prev/next arrows. */
    arrows?: boolean;
    /** Show the navigation dots. */
    dots?: boolean;
    css?: string;
    style?: string;
  }

  let {
    childNodes = [],
    renderChild,
    autoplay = false,
    interval = 5000,
    loop = true,
    arrows = true,
    dots = true,
    css = '',
    style = '',
  }: Props = $props();

  const slides = $derived(childNodes ?? []);
  const count = $derived(slides.length);

  // In the builder the active slide is shared with the editor (so picking a slide in
  // the OptionsStrip moves this preview, and vice-versa). Keyed by the slide-set array
  // identity — the same `childNodes` reference the editor holds as the node's children.
  // In production we keep a plain local index (no shared store, no autoplay coupling).
  const builderMode = getContext(EC_BUILDER_MODE) === true;

  let internalCurrent = $state(0);
  const current = $derived(builderMode ? slideSync.get(childNodes) : internalCurrent);

  function setCurrent(index: number) {
    if (builderMode) slideSync.set(childNodes, index);
    else internalCurrent = index;
  }

  // Keep the index valid if the slide set shrinks (e.g. while editing).
  $effect(() => {
    if (current > count - 1) setCurrent(Math.max(0, count - 1));
  });

  function go(index: number) {
    if (count === 0) return;
    if (loop) setCurrent((index + count) % count);
    else setCurrent(Math.min(Math.max(index, 0), count - 1));
  }

  const next = () => go(current + 1);
  const prev = () => go(current - 1);

  $effect(() => {
    // No autoplay while editing — the editor drives the active slide.
    if (!autoplay || count <= 1 || builderMode) return;
    const id = setInterval(next, interval);
    return () => clearInterval(id);
  });
</script>

<!--
  Layout lives in the scoped <style> below, not in Tailwind utilities: the carousel
  mechanics (horizontal track, equal-height slides, content fill) must hold regardless
  of whether the build/runtime CSS engines emit a given utility. `css`/`style` from the
  HTML still flow onto the root for authored cosmetics.
-->
<div class="ec-slider {css}" {style}>
  <div class="ec-slider-track" style="transform: translateX(-{current * 100}%);">
    {#each slides as slide, i (i)}
      <div class="ec-slide">
        {#if renderChild}{@render renderChild(slide)}{/if}
      </div>
    {/each}
  </div>

  {#if arrows && count > 1}
    <button type="button" aria-label="Previous slide" onclick={prev} class="ec-arrow ec-arrow--prev">
      &#8249;
    </button>
    <button type="button" aria-label="Next slide" onclick={next} class="ec-arrow ec-arrow--next">
      &#8250;
    </button>
  {/if}

  {#if dots && count > 1}
    <div class="ec-dots">
      {#each slides as _, i (i)}
        <button
          type="button"
          aria-label="Go to slide {i + 1}"
          onclick={() => go(i)}
          class="ec-dot"
          class:ec-dot--active={i === current}
        ></button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .ec-slider {
    position: relative;
    width: 100%;
    overflow: hidden;
  }

  /* All slides sit side by side in a wide track; the whole track slides horizontally by
     index (the animated transition). The track's overflow is clipped by `.ec-slider`
     (overflow:hidden) for normal painting. This wide transform layer is NOT a problem
     for section reordering: the builder supplies its own drag-feedback image via
     setReorderDragImage (BuilderSectionRender), so the browser never snapshots this
     element for the drag ghost. align-items:stretch keeps every slide equal height. */
  .ec-slider-track {
    display: flex;
    align-items: stretch;
    transition: transform 0.45s ease;
    will-change: transform;
  }

  /* Each slide is exactly one viewport wide. We deliberately set NO height/min-height —
     that would override the slide roots' authored `min-h-*` and collapse them. */
  .ec-slide {
    position: relative;
    flex: 0 0 100%;
    max-width: 100%;
  }

  .ec-arrow {
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
    z-index: 30;
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 22px;
    line-height: 1;
    border: none;
    border-radius: 9999px;
    color: #fff;
    background: rgba(0, 0, 0, 0.4);
    cursor: pointer;
    transition: background 0.2s ease;
  }
  .ec-arrow:hover { background: rgba(0, 0, 0, 0.6); }
  .ec-arrow--prev { left: 12px; }
  .ec-arrow--next { right: 12px; }

  .ec-dots {
    position: absolute;
    bottom: 16px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 30;
    display: flex;
    gap: 8px;
  }
  .ec-dot {
    width: 10px;
    height: 10px;
    padding: 0;
    border: none;
    border-radius: 9999px;
    background: rgba(255, 255, 255, 0.4);
    cursor: pointer;
    transition: background 0.2s ease;
  }
  .ec-dot:hover { background: rgba(255, 255, 255, 0.7); }
  .ec-dot--active { background: #fff; }
</style>
