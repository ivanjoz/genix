<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { canUserAccessRoute } from '$core/security';
  import { Core, tr } from '$core/store.svelte';
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

  const canAccessMenuItem = (item: MenuGroup['options'][number]) =>
    !String(item.route || '').trim() || canUserAccessRoute(item.route);
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
  />
</div>
