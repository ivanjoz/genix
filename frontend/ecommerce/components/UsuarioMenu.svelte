<script lang="ts">
  import { layerOpenedState } from "./store.svelte";
  import ButtonLayer from '$components/ButtonLayer.svelte';
  import OptionsStrip from '$components/OptionsStrip.svelte';
  import s1 from "./styles.module.css"
  import { Env } from '$core/env';
import { accessHelper } from '$core/security';

  export interface IProps {
    id?: number;
    isMobile?: boolean;
    css?: string;
  }

  const { id = 3, isMobile = false, css = "" }: IProps = $props();

  let isOpen = $state(false);
  let selectedTab = $state(1);

  const userInfo = $derived(accessHelper.getUserInfo());

  const options = [
    { id: 1, name: "Mi Cuenta" },
    { id: 2, name: "Mis Pedidos" },
    { id: 3, name: "Mis Favoritos" },
    { id: 4, name: "Config" }
  ];

  $effect(() => {
    if (layerOpenedState.id === id) {
      isOpen = true;
    } else {
      isOpen = false;
    }
  });

  $effect(() => {
    if (isOpen) {
      if (layerOpenedState.id !== id) layerOpenedState.id = id;
    } else if (layerOpenedState.id === id) {
      layerOpenedState.id = 0;
    }
  });

  function handleLogout() {
    Env.clearAccesos?.();
    layerOpenedState.id = 0;
  }
</script>

{#snippet menuContent()}
  <div class="p-16 flex flex-col min-h-300">
    <OptionsStrip
      {options}
      selected={selectedTab}
      keyId="id"
      keyName="name"
      onSelect={(opt: any) => selectedTab = opt.id}
      css="mb-12"
    />

    <div class="grow">
      {#if selectedTab === 1}
        <div class="fs18 ff-bold mb-12">Detalles de la Cuenta</div>
        <div class="flex flex-col gap-6">
          <div class="flex flex-col gap-1">
            <span class="text-gray-500 text-sm">Nombre Completo</span>
            <span class="ff-semibold">{userInfo?.Nombre || userInfo?.Usuario || "Usuario Invitado"}</span>
          </div>
          <div class="flex flex-col gap-1">
            <span class="text-gray-500 text-sm">Correo Electrónico</span>
            <span class="ff-semibold">{userInfo?.Email || "No especificado"}</span>
          </div>
          <button
            onclick={handleLogout}
            class="bg-[#4c55d5] text-white py-2.5 px-6 rounded-lg ff-bold hover:bg-[#3b44b8] transition-all w-max mt-4">
            Cerrar Sesión
          </button>
        </div>
      {:else if selectedTab === 2}
        <div class="fs18 ff-bold mb-12">Historial de Pedidos</div>
        <div class="flex flex-col items-center justify-center py-12 text-gray-400">
          <i class="icon-basket text-[48px] mb-4 opacity-20"></i>
          <p>Aún no has realizado ningún pedido.</p>
        </div>
      {:else if selectedTab === 3}
        <div class="fs18 ff-bold mb-12">Productos Favoritos</div>
        <div class="flex flex-col items-center justify-center py-12 text-gray-400">
          <i class="icon-heart text-[48px] mb-4 opacity-20"></i>
          <p>No tienes productos en tu lista de deseos.</p>
        </div>
      {:else if selectedTab === 4}
        <div class="fs18 ff-bold mb-12">Configuración del Sitio</div>
        <div class="flex flex-col gap-4">
          <button class="flex items-center gap-3 p-3 rounded-lg hover:bg-gray-50 transition-colors text-left w-full">
            <i class="icon-lock"></i>
            <span>Cambiar Contraseña</span>
          </button>
          <button class="flex items-center gap-3 p-3 rounded-lg hover:bg-gray-50 transition-colors text-left w-full">
            <i class="icon-bell"></i>
            <span>Gestionar Notificaciones</span>
          </button>
          <button class="flex items-center gap-3 p-3 rounded-lg hover:bg-gray-50 transition-colors text-left w-full">
            <i class="icon-cog"></i>
            <span>Preferencias de Idioma</span>
          </button>
        </div>
      {/if}
    </div>
  </div>
{/snippet}

{#if !isMobile}
  <div class="hidden md:block w-120 relative h-full">
    <ButtonLayer bind:isOpen={isOpen}
      wrapperClass="w-full h-full"
      buttonClass="w-full h-full"
      edgeMargin={32}
      useBig layerClass="w-700 max-w-[90vw] rounded-[11px]">
      {#snippet button(open)}
        <button class={[s1.bn1, "w-full", open ? s1.button_menu_top : ""].join(" ")}>
          <i class="icon1-user"></i>
          <span>Mi Usuario</span>
        </button>
      {/snippet}

      {@render menuContent()}
    </ButtonLayer>
  </div>
{:else if layerOpenedState.id === id}
  <div class={css}>
    <div class="bg-white rounded-[11px] shadow-lg overflow-hidden">
      {@render menuContent()}
    </div>
  </div>
{/if}
