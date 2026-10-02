<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { security } from '#libs/ui-runtime.svelte.ts';
  import { Env } from '#core/env.ts';
  import { SAAS_COMPANY_ID } from '#core/modules.ts';
  import { Core, tr } from '#core/store.svelte.ts';
  import ButtonLayer from '#components/buttons/ButtonLayer.svelte';
  import HeaderConfig from '#domain/HeaderConfig.svelte';
  import ReloadPageButton from '#domain/ReloadPageButton.svelte';
  import { SideMenu, useUI, type MenuGroup } from '@genix/ui';

  const ui = useUI();

  const menuModel = $derived<MenuGroup[]>(
    (Core.module?.menus || []).map((menu) => ({
      id: menu.id,
      name: menu.name,
      minName: menu.minName,
      meta: {
        onlySaaS: menu.onlySaaS,
        descripcion: menu.descripcion,
        pageTabs: menu.pageTabs,
      },
      options: (menu.options || []).map((option) => ({
        id: option.id,
        name: option.name,
        route: option.route,
        icon: option.icon,
        minName: option.minName,
        meta: {
          onlySaaS: option.onlySaaS,
          descripcion: option.descripcion,
          pageTabs: option.pageTabs,
        },
      })),
    })),
  );

  const canAccessMenuItem = (item: MenuGroup['options'][number]) => {
    // Sin opciones visibles, SideMenu descarta el grupo completo: el módulo SYSTEM desaparece.
    if (item.meta?.onlySaaS && Env.getCompanyID() !== SAAS_COMPANY_ID) { return false }
    return !String(item.route || '').trim() || security.canAccessRoute(item.route);
  };
</script>

<!-- Genix owns menu declaration, access policy, routing, branding, and agent visibility. -->
<div data-menu-root="true">
  <SideMenu
    model={menuModel}
    activePath={page.url.pathname}
    bind:open={ui.state.mobileMenuOpen}
    useTopMinimalMenu={ui.state.useTopMinimalMenu}
    canAccess={canAccessMenuItem}
    translate={(value) => tr(value, Core.languaje)}
    onNavigate={(route) => goto(route)}
    desktopLogoSrc="/images/genix_logo4.svg"
    mobileLogoSrc="/images/genix_logo3.svg"
    desktopBrandName="enix"
    mobileBrandName="GENIX"
  >
    <!-- Mobile only: the header keeps just the info button there, so settings and reload live here.
         The layer keeps its own open state: binding ui.state.headerSettingsOpen like the header's gear
         would let the hidden instance's click-outside close the visible one. -->
    {#snippet footer()}
      <div class="flex items-center justify-end gap-10">
        <ButtonLayer layerClass="px-8 py-6"
          buttonClass="w-40 h-40 rounded-full bg-gray-100 hover:bg-gray-200 flex items-center justify-center transition-colors shadow-sm"
          contentCss="px-4 pb-8"
          label="Opens application settings and configuration panel.">
          {#snippet button()}<i class="text-gray-700 text-lg icon-[fa--cog]"></i>{/snippet}
          <HeaderConfig />
        </ButtonLayer>
        <ReloadPageButton css="flex bg-gray-100 hover:bg-gray-200" iconCss="text-gray-700" />
      </div>
    {/snippet}
  </SideMenu>
</div>
