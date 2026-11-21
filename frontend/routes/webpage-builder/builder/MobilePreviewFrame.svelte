<script lang="ts">
import { mount, unmount } from 'svelte';
import { liveCSS } from '../stores/live-css.svelte';
import MobileCanvas from './MobileCanvas.svelte';
import PhoneFrame from '#domain/PhoneFrame.svelte';
// The storefront's base stylesheet — its custom utilities (e.g. .fx-c, which centers
// the mobile cart button in Header) live here. It's imported only by the storefront
// +layout, never by the builder route, so the cloned sheets below don't include it.
// `?inline` gives us the processed CSS text to inject straight into the iframe.
import storeCss from '#ecommerce/routes/store.css?inline';

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

<PhoneFrame>
  <iframe bind:this={iframeEl} title="Mobile preview"></iframe>
</PhoneFrame>
