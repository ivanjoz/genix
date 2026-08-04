<script lang="ts">
import T from '$components/misc/T.svelte';
import { useUI } from '@genix/ui';

// Route being navigated to. Empty on boot (service worker init), where there is no target yet.
let { path = '' }: { path?: string } = $props()

const ui = useUI()

const pathSuffix = $derived(path ? ` ${path}` : '')
</script>

<!-- Covers the content region while a route chunk is in flight. SvelteKit keeps the previous
     page mounted until the new module resolves, so an opaque overlay is what turns a click
     into visible feedback. Geometry mirrors Page.svelte's content box. -->
<div class="_1" class:useTopMinimalMenu={ui.state.useTopMinimalMenu}
  aria-busy="true" aria-live="polite" aria-label="Page is loading"
>
  <div class="flex flex-col items-center gap-12">
    <!-- Same URL AppHeader already renders, so it comes from cache and costs no request
         at the exact moment we are trying to hide latency. -->
    <div class="_2 size-64 rounded-[14px] flex items-center justify-center">
      <img src="/images/genix_logo4.svg" alt="Genix" class="size-40" />
    </div>
    <h2 class="text-gray-600 text-center px-16 break-all">
      <T text={`Loading${pathSuffix}...|Cargando${pathSuffix}...`} />
    </h2>
  </div>
</div>

<style>
  ._1 {
    position: fixed;
    top: var(--header-height);
    left: var(--menu-min-width);
    right: 0;
    bottom: 0;
    background-color: #e9eef1;
    display: flex;
    align-items: center;
    justify-content: center;
    /* Above page content, side layers (101) and modals (175) so a stale open panel is
       covered too; below the side menu (300) so another route stays one click away. */
    z-index: 200;
    /* 80ms delay: routes already in the module cache resolve before this paints, so
       warm navigations never flash a loader. */
    animation: reveal 0.12s 0.08s both;
  }

  /* Top minimal menu puts navigation in the header row, so the content box is full width. */
  ._1.useTopMinimalMenu {
    left: 0;
  }

  ._2 {
    background-color: var(--primary);
    animation: pulse 1.1s ease-in-out infinite;
  }

  @keyframes reveal {
    from { opacity: 0 }
    to { opacity: 1 }
  }

  @keyframes pulse {
    0%, 100% { transform: scale(1); opacity: 1 }
    50% { transform: scale(0.9); opacity: 0.7 }
  }

  @media (max-width: 750px) {
    ._1 {
      left: 0;
    }
  }
</style>
