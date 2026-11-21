<script lang="ts">
  import type { Snippet } from 'svelte'

  // A 390px phone on a gray stage, centered in the space it is given: how the ecommerce builder's
  // mobile preview and the carta builder show a mobile layout. The content fills the screen; the
  // page decides whether it is an iframe (real media queries) or plain markup.
  let { children, scrollable = false }: { children: Snippet, scrollable?: boolean } = $props()
</script>

<div class="phone-stage">
  <!-- 390px ≈ a modern phone in CSS px, with a device-like frame. -->
  <div class="phone-screen" class:scrollable>
    {@render children()}
  </div>
</div>

<style>
  .phone-stage {
    /* flex:1 so the stage fills its column and justify-center truly centers the phone. The gray
       backdrop makes the white phone read as a device on a stage. */
    flex: 1;
    display: flex;
    justify-content: center;
    padding-top: 18px;
    height: calc(100vh - var(--header-height));
    /* Radial spotlight behind the device: lighter just above-center, falling off to a darker
       vignette at the edges — reads as depth/stage. */
    background: radial-gradient(ellipse 75% 65% at 50% 32%, #c2c4ca 0%, #a0a2a9 48%, #7c7e86 100%);
  }

  .phone-screen {
    width: 390px;
    height: calc(100vh - 36px - var(--header-height));
    border-radius: 12px;
    background: #fff;
    box-shadow: 0 12px 40px rgba(15, 23, 42, 0.25);
    outline: 8px solid #000000c2;
    overflow: hidden;
  }

  .phone-screen.scrollable {
    overflow-y: auto;
  }

  .phone-screen :global(iframe) {
    width: 100%;
    height: 100%;
    border: 0;
    display: block;
  }
</style>
