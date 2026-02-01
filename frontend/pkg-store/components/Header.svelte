<script lang="ts">
  import { browser } from "$app/environment";
  import { Core } from '$core/store.svelte';
  import CartMenu from '$store/components/CartMenu.svelte';
  import SearchBar from '$store/components/SearchBar.svelte';
  import UsuarioMenu from '$store/components/UsuarioMenu.svelte';
  import { onMount } from "svelte";
  import { layerOpenedState, ProductsSelectedMap } from "./store.svelte";

  // State for mobile menu
  let mobileMenuOpen = $state(false);

  const cartCant = $derived(ProductsSelectedMap.size)

  // Handle scroll effect
  onMount(() => {

  	if(browser){
			const subheader0 = document.getElementById("sh-0")
			const container = subheader0?.offsetParent

			if(container){
			  const handleScroll = () => {
			    if (container.scrollTop > 48) {
			      subheader0?.classList.add("s2")
			    } else {
			      subheader0?.classList.remove("s2")
			    }
			  };
			
			  container.addEventListener("scroll", handleScroll);
			
			  return () => {
			    container.removeEventListener("scroll", handleScroll);
			  };	
			}
   	}
  });

  // Toggle mobile menu
  function toggleMobileMenu() {
    mobileMenuOpen = !mobileMenuOpen;
  }

  // Close mobile menu when clicking outside
  function closeMobileMenu() {
    mobileMenuOpen = false;
  }

</script>

<div class="_2 flex justify-between text-white h-48"></div>
<div id="sh-0" class="header-0 flex justify-between w-full h-68">
  header
</div>
<div id="sh-1" class="header-1 _1 h-58 md:h-68 top-48 absolute flex items-center md:justify-between w-full left-0 px-4 md:px-80"
>
  <div class="hidden md:block"></div>
  <button aria-label="page-menu" class="_6 fx-c w-[14vw] md:hidden!" onclick={ev => {
    ev.stopPropagation()
    if (document.startViewTransition) {
      document.startViewTransition(() => {
        Core.mobileMenuOpen = 1
      })
    }
  }}>
    <i class="text-[22px] icon-menu"></i>
  </button>
  <SearchBar />
  <div class="flex items-center h-42 ml-auto md:ml-0">
    <CartMenu css="mr-8 hidden md:block relative w-120 h-full" id={1}/>
    <UsuarioMenu />
    <div class={"relative w-[15vw] fx-c flex md:hidden! "+(layerOpenedState.id === 2 ? "s1" : "")}>
      <button class="_3 absolute w-full fx-c" onclick={ev => {
        ev.stopPropagation()
        layerOpenedState.id = layerOpenedState.id ? 0 : 2
      }}>
        <i class={"icon1-basket mt-4 "+(cartCant > 0 ? "mb-[-2px] mr-2" : "mb-2")}></i>
        {#if cartCant > 0}
          <div class="_4 fs14 fx-c w-22 h-22 mb-14 ml-26 absolute rounded-[50%]">{cartCant}</div>
        {/if}
      </button>
      <button aria-label="close_layer" class="_5 absolute fx-c w-40 h-40 rounded-[50%]" onclick={ev => {
        ev.stopPropagation()
        layerOpenedState.id = layerOpenedState.id ? 0 : 2
      }}>
        <i class="icon-cancel"></i>
      </button>
    </div>
  </div>
  <CartMenu isMobile={true} id={2} css="absolute w-full top-[100%] left-0"/>
</div>

<style>
  ._2 {
    background-color: #336686;
    padding: 0 80px 0 80px;
  }
  ._4 {
    background-color: rgb(209, 86, 86);
    color: white;
  }
  ._3 {
    font-size: 21px;
    background-color: transparent;
    transform: rotateY(0deg);
    z-index: 11;
  }
  ._5 {
    background-color: #dd6464;
    color: white;
    transform: rotateY(180deg);
    pointer-events: none;
    font-size: 20px;
  }
  ._3, ._5 {
    transition: transform 0.6s ease-in-out;
    backface-visibility: hidden;
    outline: none;
    border: none;
  }
  .s1 ._3 {
    transform: rotateY(180deg);
    pointer-events: none;
  }
  .s1 ._5 {
    transform: rotateY(0deg);
    pointer-events: all;
  }
  ._6 {
    outline: none;
    border: none;
    background-color: transparent;
  }
  
  .header-0, .header-1 {
    transition: height 0.3s ease;
  }
  .header-1 {
    background-color: #ffffff;
      box-shadow:
      rgba(0, 0, 0, 0.16) 0px 2px 6px,
      rgba(0, 0, 0, 0.2) 0px 2px 8px;
    z-index: 200;
  }
  .header-0:global(.s1), .header-0:global(.s1) + .header-1,
  .header-0:global(.s2), .header-0:global(.s2) + .header-1 {
    height: 52px;
  }
  .header-0:global(.s1) + .header-1 {
    position: fixed;
    top: 0;
  }
  .header-0:global(.s2) + .header-1 {
    position: fixed;
    top: var(--header-height);
    width: calc(100% - var(--store-editor-collapsed-width) - 12px);
    
  }
  @media (max-width: 739px) {
   .header-0:global(.s1), .header-0:global(.s1) + .header-1 {
      height: 47px;
    }
  }
</style>
