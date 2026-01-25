<script lang="ts">
import LoginForm from '$components/ecommerce/forms/LoginForm.svelte';
import MobileLayerVertical from '$ui/components/MobileLayerVertical.svelte';
  import { Core, mainMenuOptions, suscribeUrlFlag } from "./store.svelte";

  let divContainer: HTMLDivElement

  const closeMenu = (event?: MouseEvent) => {
    event?.stopPropagation?.();
    divContainer.style.setProperty("view-transition-name","mobile-side-menu")
    setTimeout(() => {
      divContainer.style.setProperty("view-transition-name","")
    },400)

    if (document.startViewTransition) {
      document.startViewTransition(() => {
        Core.mobileMenuOpen = 0;
      });
    } else {
      Core.mobileMenuOpen = 0;
    }
  };

  $effect(() => {
    if(Core.mobileMenuOpen){
      suscribeUrlFlag("mob-menu", () => { closeMenu() })
    }
  })

  let css = $derived.by(() => {
    let css = "_1 w-[74vw] h-[100vh] fixed top-0 left-0";
    if (Core.mobileMenuOpen === 1) {
      divContainer.style.setProperty("view-transition-name","mobile-side-menu")
      css += " _2"

      setTimeout(() => {
        divContainer.style.setProperty("view-transition-name","")
      },400)
    }
    return css;
  });

  let overlayCss = $derived.by(() => {
    let css = "_4 top-0 left-0 fixed w-[100vw] h-[100vh]";
    if (Core.mobileMenuOpen) {
      css += " _5";
    }
    return css;
  });
</script>

<div class={overlayCss} onclick={closeMenu}></div>

<div id="mob-menu" class={css} bind:this={divContainer}>
  <button class="_3 absolute top-4 right-4 w-40 h-40" onclick={closeMenu}>
    <i class="icon-cancel"></i>
  </button>
  <div class="grid gap-8 grid-cols-2 p-8 mt-54">
  {#each mainMenuOptions as opt}
    <div class="_7" onclick={ev => {
      ev.stopPropagation()
      if(opt.onClick){ opt.onClick() }
    }}>
      <div class="_8"></div>
      <div class="h-24 fs20 mt-[-4px]"><i class={opt.icon}></i></div>
      <div class="flex items-center text-center grow-1">{opt.name}</div>
    </div>
  {/each}
  </div>
</div>
<MobileLayerVertical title="Iniciar Sesión" id={1}>
  <LoginForm></LoginForm>
</MobileLayerVertical>

<style>
  ._1 {
    background-color: white;
    position: fixed;
    opacity: 0;
    pointer-events: none;
    z-index: -1;
  }
  ._2 {
    opacity: 1;
    pointer-events: all;
    z-index: 210;
  }
  ._3 {
    border-radius: 50%;
    border: none;
    outline: none;
    background-color: rgb(255, 224, 224);
    color: rgb(187, 82, 82);
    font-size: 20px;
  }
  ._4 {
    background-color: rgba(15, 23, 42, 0);
    opacity: 0;
    transition: opacity 200ms ease-in-out;
    pointer-events: none;
    z-index: -2;
  }
  ._5 {
    opacity: 1;
    background-color: rgba(15, 23, 42, 0.5);
    pointer-events: all;
    z-index: 208;
  }
  ._6 {
    background-color: rgb(240, 240, 240);
    min-height: 14vw;
    border-radius: 8px;
    padding: 0 6px 6px 6px;
    line-height: 1.1;
    text-align: center;
    background-color: rgb(243 242 249);
    color: #3b384b;
    margin-bottom: calc(1vw + 4px);
    box-shadow: rgb(189 190 210) 0px 1px 2px;
  }
  ._7 {
    position: relative;
    background-color: rgb(240, 240, 240);
    min-height: 14vw;
    border-radius: 8px;
    padding: 0 6px 6px 6px;
    line-height: 1.1;
    text-align: center;
    background-color: rgb(243 242 249);
    color: #3b384b;
    margin-bottom: calc(1vw + 4px);
    box-shadow: rgb(189 190 210) 0px 1px 2px;
    display: flex;
    align-items: center;
    flex-direction: column;
  }
  ._8 {
    background-color: rgb(243 242 249);
    position: absolute;
    height: 2.4rem;
    width: 3rem;
    border-radius: 50%;
    z-index: -1;
    top: -8px;
  }
</style>