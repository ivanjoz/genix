<script lang="ts">
  import LoginForm from '$components/form/LoginForm.svelte';
  import MobileLayerVertical from '$components/layers/MobileLayerVertical.svelte';
  import { mainMenuOptions, suscribeUrlFlag } from '$core/store.svelte';
  import {
    MobileMenu,
    useUI,
    type MobileMenuItem,
  } from '@genix/ui';

  const ui = useUI();
  const menuItems: MobileMenuItem[] = mainMenuOptions.map((option, optionIndex) => ({
    id: optionIndex,
    name: option.name,
    icon: option.icon,
  }));

  const selectMenuItem = (item: MobileMenuItem) => {
    mainMenuOptions[Number(item.id)]?.onClick?.();
  };

  $effect(() => {
    if (!ui.state.mobileMenuOpen) { return; }
    suscribeUrlFlag('mob-menu', () => {
      ui.state.mobileMenuOpen = false;
    });
  });
</script>

<!-- Storefront actions and login content remain owned by Genix. -->
<div id="mob-menu">
  <MobileMenu
    items={menuItems}
    bind:open={ui.state.mobileMenuOpen}
    onSelect={selectMenuItem}
    closeLabel="Cerrar menú"
  />
</div>

<MobileLayerVertical title="Iniciar Sesión" id={1}>
  <LoginForm />
</MobileLayerVertical>
