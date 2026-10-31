<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { canOpenRoute, getSessionMenuModule } from '#core/modules.svelte.ts';
  import { Core, tr } from '#core/store.svelte.ts';
  import ButtonLayer from '#components/buttons/ButtonLayer.svelte';
  import HeaderConfig from '#domain/HeaderConfig.svelte';
  import ModuleSelector from '#domain/ModuleSelector.svelte';
  import ReloadPageButton from '#domain/ReloadPageButton.svelte';
  import { SideMenu, useUI, type MenuGroup } from '@genix/ui';

  const ui = useUI();

  // Always one module's menus (db/modules.toml); on "/", those of the module the user came from.
  // Pages the session cannot open are hidden, and genix-ui drops a group left without options.
  const menuModel = $derived.by<MenuGroup[]>(() =>
    (getSessionMenuModule(page.url.pathname)?.menus || []).map((menuGroup) => ({
      id: menuGroup.id,
      name: menuGroup.name,
      minName: menuGroup.minName,
      options: menuGroup.pages.map((menuPage) => ({
        name: menuPage.name,
        route: menuPage.route,
        icon: menuPage.icon,
        disabled: menuPage.pending,
      })),
    })),
  );
</script>

<!-- Genix owns menu declaration, access policy, routing, branding, and agent visibility. -->
<div data-menu-root="true">
  <SideMenu
    model={menuModel}
    activePath={page.url.pathname}
    bind:open={ui.state.mobileMenuOpen}
    useTopMinimalMenu={ui.state.useTopMinimalMenu}
    canAccess={(menuItem) => canOpenRoute(menuItem.route || '')}
    translate={(value) => tr(value, Core.languaje)}
    onNavigate={(route) => goto(route)}
  >
    {#snippet header(isMobile)}<ModuleSelector {isMobile} />{/snippet}
    <!-- The module selector took the logo's place at the top, so the brand sits at the bottom. -->
    {#snippet desktopFooter()}
      <!-- pl-13 + the slot's 6px centers the 32px logo in the 70px collapsed menu. -->
      <div class="flex items-center pl-13">
        <img class="w-32 h-32 shrink-0" src="/images/genix_logo4.svg" alt="Genix" />
        <span class="side-menu-expanded-only ff-bold text-xl -ml-2 whitespace-nowrap">enix</span>
      </div>
    {/snippet}
    <!-- Mobile only: the header keeps just the info button there, so settings and reload live here.
         The layer keeps its own open state: binding ui.state.headerSettingsOpen like the header's gear
         would let the hidden instance's click-outside close the visible one. -->
    {#snippet footer()}
      <div class="flex items-center justify-between gap-10">
        <img src="/images/genix_logo3.svg" alt="Genix" class="size-36" />
        <div class="flex items-center gap-10">
          <ButtonLayer layerClass="px-8 py-6"
            buttonClass="w-40 h-40 rounded-full bg-surface-muted hover:bg-surface-strong flex items-center justify-center transition-colors shadow-sm"
            contentCss="px-4 pb-8"
            label="Opens application settings and configuration panel.">
            {#snippet button()}<i class="text-fg-soft text-lg icon-[fa--cog]"></i>{/snippet}
            <HeaderConfig />
          </ButtonLayer>
          <ReloadPageButton css="flex bg-surface-muted hover:bg-surface-strong" iconCss="text-fg-soft" />
        </div>
      </div>
    {/snippet}
  </SideMenu>
</div>
