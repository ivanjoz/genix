<script lang="ts">
  import { layerOpenedState } from "./store.svelte";
  import ButtonLayer from '$components/ButtonLayer.svelte';
  import s1 from "./styles.module.css"

  interface Props {
    id?: number;
  }

  let { id = 3 }: Props = $props();

  let isOpen = $state(false);

  $effect(() => {
    isOpen = layerOpenedState.id === id;
  });

  $effect(() => {
    if (isOpen) {
      layerOpenedState.id = id;
    } else if (layerOpenedState.id === id) {
      layerOpenedState.id = 0;
    }
  });
</script>

<div class="hidden md:block w-120">
  <ButtonLayer bind:isOpen={isOpen}
    wrapperClass="w-full h-full"
    buttonClass="w-full h-full"
    useBig layerClass="w-300!">
    {#snippet button(open)}
      <button class={["bn1 w-full", open ? s1.button_menu_top : ""].join(" ")}>
        <i class="icon1-user"></i>
        <span>Mi Usuario</span>
      </button>
    {/snippet}

    <div class="p-16">
      <div class="fs18 ff-bold mb-12">Mi Cuenta</div>
      <div class="flex flex-col gap-8">
        <button class="text-left py-4 hover:text-[#4c55d5]">Mis Pedidos</button>
        <button class="text-left py-4 hover:text-[#4c55d5]">Mi Perfil</button>
        <button class="text-left py-4 hover:text-[#4c55d5]">Cerrar Sesión</button>
      </div>
    </div>
  </ButtonLayer>
</div>
