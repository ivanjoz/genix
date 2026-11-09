<script lang="ts">
  // Module switcher at the top of the side menu: the active module's button opens a grid of the
  // modules the session can enter (db/modules.toml), each one landing on its last visited page.
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import ButtonLayer from '#components/buttons/ButtonLayer.svelte';
  import { logout } from '#core/env.ts';
  import { getSessionMenuModule, getSessionModuleEntries, rememberSelectedModule } from '#core/modules.svelte.ts';
  import { tr } from '#core/store.svelte.ts';
  import { useUI } from '@genix/ui';

  let { isMobile }: { isMobile: boolean } = $props()

  const ui = useUI()
  let isOpen = $state(false)
  // On "/" (no module lists it) this is the module the user came from.
  const activeModule = $derived(getSessionMenuModule(page.url.pathname))
  const moduleEntries = $derived(getSessionModuleEntries())

  // On mobile the selector lives in the drawer, which has to close along with the layer.
  const navigateFromLayer = (route: string, moduleCode = '') => {
    isOpen = false
    if (moduleCode) { rememberSelectedModule(moduleCode) }
    if (isMobile) { ui.state.mobileMenuOpen = false }
    goto(route)
  }
</script>

<ButtonLayer bind:isOpen wrapperClass="w-full" layerClass="md:w-560" contentCss="p-12"
  buttonClass="flex items-center gap-10 w-full px-10 cursor-pointer transition-colors
    {isMobile ? 'h-42 rounded-[6px] text-accent-fg bg-accent-bg hover:bg-accent-bg-strong' : 'h-38 side-menu-reveal rounded-lg text-[#f2d29d] bg-[#2c2f4e] hover:bg-[#393e7b]'}"
  label="Opens the module selector.">
  {#snippet button()}
    <i class="{activeModule?.icon || 'icon-[fa--th-large]'} text-lg shrink-0"></i>
    <!-- One fading wrapper: the hook sets `transition`, which would override the chevron's rotation. -->
    <span class="side-menu-expanded-only flex flex-1 min-w-0 items-center gap-10">
      <span class="flex-1 min-w-0 truncate text-left font-semibold">{tr(activeModule?.name || 'Modules|Módulos')}</span>
      <i class="icon-[fa--angle-down] text-[22px] shrink-0 transition-transform" class:rotate-180={isOpen}></i>
    </span>
  {/snippet}

  <div class="flex items-center gap-8 mb-12">
    <button type="button" class="flex items-center gap-6 h-34 px-12 rounded-md bg-surface-muted hover:bg-surface-strong text-fg-soft cursor-pointer"
      onclick={() => navigateFromLayer('/')}>
      <i class="icon-[fa--home]"></i>{tr('Home|Inicio')}
    </button>
    <button type="button" class="flex items-center gap-6 h-34 px-12 rounded-md bg-surface-muted hover:bg-surface-strong text-red-fg cursor-pointer"
      onclick={() => { isOpen = false; logout() }}>
      <i class="icon-[fa--sign-out]"></i>{tr('Sign out|Salir')}
    </button>
    <button type="button" class="ml-auto w-34 h-34 rounded-md bg-red-bg hover:bg-red-bg-strong text-red-fg cursor-pointer"
      aria-label={tr('Close|Cerrar')} onclick={() => { isOpen = false }}>
      <i class="icon-[fa--close]"></i>
    </button>
  </div>

  <div class="grid grid-cols-2 md:grid-cols-4 gap-10">
    {#each moduleEntries as { module, entryRoute } (module.code)}
      {@const isActiveModule = module.code === activeModule?.code}
      <a href={entryRoute} class="module-tile flex flex-col items-center justify-center gap-8 h-100 p-8 rounded-lg
        border text-center {isActiveModule ? 'is-active bg-accent-bg border-accent-solid outline outline-accent-solid text-accent-fg' : 'bg-surface-soft border-line text-fg'}"
        onclick={(clickEvent) => {
          // Keep modified clicks (new tab/window) with the browser.
          if (clickEvent.metaKey || clickEvent.ctrlKey || clickEvent.shiftKey || clickEvent.altKey) { return }
          clickEvent.preventDefault()
          navigateFromLayer(entryRoute, module.code)
        }}>
        <i class="{module.icon} text-3xl text-accent-solid shrink-0"></i>
        <span class="font-semibold leading-tight">{tr(module.name)}</span>
      </a>
    {:else}
      <p class="col-span-full text-fg-muted">{tr('No module is available for this user.|Ningún módulo está disponible para este usuario.')}</p>
    {/each}
  </div>
</ButtonLayer>

<style>
  .module-tile { transition: box-shadow 120ms, border-color 120ms; }
  .module-tile:not(.is-active):hover { border-color: var(--accent-border); box-shadow: 0 4px 14px color-mix(in srgb, var(--primary) 18%, transparent); }
</style>
