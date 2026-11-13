<script lang="ts">
  // The storefront's hamburger menu (phones). It holds the customer account: an account tile swaps
  // the tiles for that UsuarioMenu tab, full width, with a back arrow to the tiles.
  import { suscribeUrlFlag } from '#core/store.svelte.ts';
  import UsuarioMenu from '#ecommerce/components/UsuarioMenu.svelte';
  import { WebStore } from '#ecommerce/services/web-store.svelte.ts';
  import {
    MobileMenu,
    useUI,
    type MobileMenuItem,
  } from '@genix/ui';

  const ui = useUI();

  // accountTab is the UsuarioMenu tab a tile opens (1 Login / Mi Cuenta, 2 Regístrate / Mis Pedidos,
  // 3 Mis Favoritos, 4 Config); 0 is "Tienda", which just closes the menu.
  type IMenuTile = MobileMenuItem & { accountTab: number };
  const STORE_TILE: IMenuTile = { id: 'store', name: "Tienda", icon: "icon-[fa--home]", accountTab: 0 };
  // A guest's "Mis Pedidos" needs a session first: it opens the login.
  const GUEST_TILES: IMenuTile[] = [
    { id: 'login', name: "Iniciar Sesión", icon: "icon-[fa--user]", accountTab: 1 },
    { id: 'register', name: "Regístrate", icon: "icon-[fa--file-text]", accountTab: 2 },
    { id: 'orders', name: "Mis Pedidos", icon: "icon-[fa--cube]", accountTab: 1 },
    STORE_TILE,
  ];
  const CUSTOMER_TILES: IMenuTile[] = [
    { id: 'account', name: "Mi Cuenta", icon: "icon-[fa--user]", accountTab: 1 },
    { id: 'orders', name: "Mis Pedidos", icon: "icon-[fa--cube]", accountTab: 2 },
    { id: 'favorites', name: "Mis Favoritos", icon: "icon-[fa--heart]", accountTab: 3 },
    { id: 'config', name: "Config", icon: "icon-[fa--cog]", accountTab: 4 },
    STORE_TILE,
  ];
  const menuTiles = $derived(WebStore.customer ? CUSTOMER_TILES : GUEST_TILES);

  // The UsuarioMenu tab shown instead of the tiles; 0 shows the tiles. UsuarioMenu moves it itself
  // after a login or a logout.
  let shownAccountTab = $state(0);
  const shownTabName = $derived(
    (WebStore.customer ? CUSTOMER_TILES : GUEST_TILES).find((tile) => tile.accountTab === shownAccountTab)?.name || ""
  );

  const selectTile = (item: MobileMenuItem) => {
    const accountTab = (item as IMenuTile).accountTab;
    if (accountTab === 0) { ui.state.mobileMenuOpen = false; return }
    shownAccountTab = accountTab;
  };

  $effect(() => {
    // A closed menu reopens on its tiles.
    if (!ui.state.mobileMenuOpen) { shownAccountTab = 0; return; }
    suscribeUrlFlag('mob-menu', () => {
      ui.state.mobileMenuOpen = false;
    });
  });
</script>

{#snippet accountSection()}
  <!-- Its top row sits left of the drawer's close button. -->
  <div class="flex items-center gap-8 h-48 pl-6 pr-56">
    <button class="w-40 h-40 shrink-0 fx-c rounded-full hover:bg-gray-100 cursor-pointer" aria-label="Volver al menú"
      onclick={() => { shownAccountTab = 0 }}>
      <i class="icon-[fa--arrow-left] text-[18px]"></i>
    </button>
    <span class="ff-bold text-[18px] truncate">{shownTabName}</span>
  </div>
  <UsuarioMenu isMobile={true} bind:selectedTab={shownAccountTab} />
{/snippet}

<div id="mob-menu">
  <MobileMenu
    items={menuTiles}
    bind:open={ui.state.mobileMenuOpen}
    onSelect={selectTile}
    closeLabel="Cerrar menú"
    content={shownAccountTab > 0 ? accountSection : undefined}
    isFullWidth={shownAccountTab > 0}
  />
</div>
