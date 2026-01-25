<script lang="ts">

  export interface IProps {
    id: number, isMobile?: boolean, css?: string
  }
  
  const { 
    id = 0, isMobile = false, css = ""
  }: IProps = $props();

  import { layerOpenedState, ProductsSelectedMap } from "./store.svelte";
  import angleSvg from "$lib/assets/angle.svg?raw";
  import { parseSVG } from "$core/helpers.ts";
  import s1 from "./styles.module.css"
  import ArrowSteps from "./ArrowSteps.svelte"
  import { Ecommerce } from "$lib/globals.svelte.ts";
  import Input from "$components/Input.svelte";
  import CiudadesSelector from "./CiudadesSelector.svelte";
  import ProductCardHorizonal from "./ProductCardHorizonal.svelte";
    import { Core } from "$core/store.svelte";

  let userForm = {} as any

  function toggleCartDiv() {
    layerOpenedState.id = layerOpenedState.id === id ? 0 : id;
  }
</script>

<div class={css}>
  {#if !isMobile}
    <button class={["bn1 w-full",layerOpenedState.id === id ? s1.button_menu_top : ""].join(" ")} 
      onclick={toggleCartDiv}>
      <i class="icon1-basket"></i>
      <span>Carrito</span>
    </button>
  {/if}

  {#if layerOpenedState.id === id}
    <img class={"absolute h-20 _1 "+(isMobile ? "right-17" : "right-[40%]")} alt="" 
      src={parseSVG(angleSvg)} 
    />
    <div class="_2 absolute p-12 flex flex-col">
      <ArrowSteps selected={Ecommerce.cartOption}
        columnsTemplate={Core.deviceType === 3 ? "1fr 1fr 1fr 0.7fr" : ""}
        onSelect={e => {
          Ecommerce.cartOption = e.id
        }}
        options={[ 
          { id: 1, name: 'Carrito', icon: "icon-basket" }, 
          { id: 2, name: 'Datos de Envío', icon: "icon-doc-inv-alt" }, 
          { id: 3, name: 'Pago', icon: "icon-shield" }, 
          { id: 4, 
            name: 'Confirmación', 
            icon: "icon-ok" 
          },
        ]}
      >
        {#snippet optionRender(e)}
          {#if !isMobile}
            <div class="flex items-center mt-1 ff-semibold">
              <i class={`text-[18px] ${e.icon} mr-2`}></i>
              <div class="mr-6 text-left lh-11">{e.name}</div>
            </div>
          {/if}
          {#if isMobile}
            <div class="mr-6 text-left lh-11">{e.name}</div>
          {/if}
        {/snippet}
      </ArrowSteps>
      <div class="w-full px-4 overflow-auto grow-1 mt-4">
        {#if Ecommerce.cartOption === 1}
          <div class="mt-8 mb-8 fs18 ff-bold">Total a pagar: </div>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-12 pb-6">
            {#each ProductsSelectedMap.values() as cartProducto}
              <ProductCardHorizonal producto={cartProducto.producto} />
            {/each}
          </div>
        {/if}
        {#if Ecommerce.cartOption === 2}
          <div class="mt-8 mb-8 fs18 ff-bold">Total a pagar: </div>
          <div class="grid grid-cols-12 gap-12">
            <Input label="Nombres" css="col-span-6" saveOn={userForm} save="nombres" required={true} />
            <Input label="Apellidos" css="col-span-6" saveOn={userForm} save="apellidos" required={true} />
            <Input label="Correo Electrónico" css="col-span-6" saveOn={userForm} save="email" required={true} />
            <CiudadesSelector saveOn={userForm} save="ciudadID" css="col-span-6"/>
            <Input label="Dirección" css="col-span-6" saveOn={userForm} save="direccion" required={true} />
            <Input label="Referencia" css="col-span-6" saveOn={userForm} save="referencia" />
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  ._1 {
    bottom: -10px;
    z-index: 121;
  }
  ._2 {
    top: 50px;
    width: 48rem;
    right: -12rem;
    max-width: calc(82vw - 24px);
    height: calc(100vh - 120px);
    background-color: #fff;
    z-index: 120;
    box-shadow:
      #46466059 0 2px 18px -2px,
      #00000059 0 0 6px;
    border-radius: 11px;
  }

  @media (max-width: 740px) {
    ._2 {
      top: calc(100% + 6px);
      width: 100vw;
      max-width: 100vw;
      left: 0;
      padding: 10px 6px;
    }
  }
</style>
