<script lang="ts">
  // Round reload button, shared by the header (desktop) and the side menu footer (mobile).
  // Callers give the display and the colors, since each sits on a different background.
  import { Env } from '#core/env.ts';
  import { tr } from '#core/store.svelte.ts';
  import { markPageServicesForRefresh } from '@genix/ui/runtime';

  let { css, iconCss }: { css: string, iconCss: string } = $props()
  let isReloading = $state(false)

  // Reloading by itself would replay the same cached responses, so the routes this page read are
  // flagged in the service worker first: the reload then re-fetches them from the server.
  // A failed mark only means the reload reads the cache, which is no worse than a plain reload.
  const reloadPageFromServer = async () => {
    isReloading = true
    try {
      await markPageServicesForRefresh(Env.getPathname())
    } catch (error) {
      console.warn('[ReloadPageButton] Could not mark the page routes for refresh.', error)
    }
    window.location.reload()
  }
</script>

<button class="w-40 h-40 rounded-full items-center justify-center transition-colors shadow-sm cursor-pointer {css}"
  class:animate-spin={isReloading}
  aria-label={tr('Reload the page from the server|Recargar la página desde el servidor')}
  title={tr('Reload|Recargar')} disabled={isReloading} onclick={reloadPageFromServer}>
  <i class="text-lg icon-[fa--refresh] {iconCss}"></i>
</button>
