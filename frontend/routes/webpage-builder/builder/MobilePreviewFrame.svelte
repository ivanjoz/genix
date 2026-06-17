<script lang="ts">
import { mount, unmount } from 'svelte';
import { liveCSS } from '../stores/live-css.svelte';
import MobileCanvas from './MobileCanvas.svelte';
// The storefront's base stylesheet — its custom utilities (e.g. .fx-c, which centers
// the mobile cart button in Header) live here. It's imported only by the storefront
// +layout, never by the builder route, so the cloned sheets below don't include it.
// `?inline` gives us the processed CSS text to inject straight into the iframe.
import storeCss from '$ecommerce/routes/store.css?inline';

  interface Props {
    // Palette CSS vars, forwarded into the mounted iframe tree (kept reactive below).
    paletteStyles: string;
  }

  let { paletteStyles }: Props = $props();

  let iframeEl = $state<HTMLIFrameElement>();

  // Reactive props bag handed to mount(): mutating its fields updates the mounted
  // MobileCanvas in place, so palette changes propagate without re-mounting.
  const mountedProps = $state({ paletteStyles: '' });
  $effect(() => { mountedProps.paletteStyles = paletteStyles; });

  // Mount the section tree into the SAME-ORIGIN iframe body. Same JS runtime => the
  // mounted tree shares the editorStore singleton, so click-to-select works directly.
  // Real Tailwind media queries (md/lg = min-width) then resolve against the iframe's
  // own ~390px viewport, producing a true mobile layout instead of a squashed desktop.
  $effect(() => {
    const doc = iframeEl?.contentDocument;
    if (!doc) return;

    // The iframe head starts empty: clone the parent's stylesheets so build-time
    // Tailwind (storefront chrome) is present. The runtime utility style tag is
    // handled separately below, so skip cloning it to avoid a duplicate id.
    for (const node of document.querySelectorAll('link[rel="stylesheet"], style')) {
      if (node.id === 'live-tailwind-jit') continue;
      doc.head.appendChild(node.cloneNode(true));
    }
    // Inject the storefront base stylesheet so chrome (Header etc.) renders with the
    // same custom utilities/resets it gets on the live store — scoped to the iframe so
    // the builder document's own body/font styles are untouched.
    const storeStyle = doc.createElement('style');
    storeStyle.textContent = storeCss;
    doc.head.appendChild(storeStyle);

    // Drop the default body margin so the preview sits flush like the real storefront.
    doc.body.style.margin = '0';

    const app = mount(MobileCanvas, { target: doc.body, props: mountedProps });
    return () => unmount(app);
  });

  // Mirror runtime-authored Tailwind into the iframe head; re-runs whenever liveCSS
  // recompiles so editor/agent classes apply at the iframe's viewport width.
  $effect(() => {
    const doc = iframeEl?.contentDocument;
    if (!doc) return;
    let styleEl = doc.getElementById('live-tailwind-jit') as HTMLStyleElement | null;
    if (!styleEl) {
      styleEl = doc.createElement('style');
      styleEl.id = 'live-tailwind-jit';
      doc.head.appendChild(styleEl);
    }
    styleEl.textContent = liveCSS.css;
  });
</script>

<div class="mobile-preview">
  <!-- 390px ≈ a modern phone in CSS px; centered on the canvas with a device-like frame. -->
  <iframe bind:this={iframeEl} title="Mobile preview"></iframe>
</div>

<style>
  .mobile-preview {
    /* flex:1 so this fills the canvas column; justify-center then truly centers the
       390px frame in the space left of the editor panel. Gray backdrop makes the
       white phone read as a device on a stage. */
    flex: 1;
    display: flex;
    justify-content: center;
    padding-top: 18px;
    height: calc(100vh - var(--header-height));
    /* Radial spotlight behind the device: lighter just above-center where the phone
       sits, falling off to a darker vignette at the edges — reads as depth/stage. */
    background: radial-gradient(ellipse 75% 65% at 50% 32%, #c2c4ca 0%, #a0a2a9 48%, #7c7e86 100%);
  }

  iframe {
    width: 390px;
    height: calc(100vh - 36px - var(--header-height));
    border: 0;
    border-radius: 12px;
    background: #fff;
    box-shadow: 0 12px 40px rgba(15, 23, 42, 0.25);
    outline: 8px solid #000000c2;
  }
</style>
