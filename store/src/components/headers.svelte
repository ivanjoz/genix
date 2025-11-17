<script lang="ts">
  import { onMount } from "svelte";
  import SearchBar from "./search-bar.svelte";
  import CartMenu from "./cart-menu.svelte";
  import UsuarioMenu from "./usuario-menu.svelte";
    import { layerOpenedState, ProductsSelectedMap } from "./store.svelte";
    import { Core } from "../core/store.svelte";

  // State for mobile menu
  let mobileMenuOpen = $state(false);
  let isScrolled = $state(false);
  let isSubheaderFixed = $state(false);
  let subheaderElement: HTMLElement;

  const cartCant = $derived(ProductsSelectedMap.size)

  // Handle scroll effect
  onMount(() => {
    /*
    const handleScroll = () => {
      // console.log("comparison:", window.scrollY, subheaderElement.offsetTop);

      if (window.scrollY > subheaderElement.offsetTop) {
        isSubheaderFixed = true;
      } else {
        isSubheaderFixed = false;
      }
    };

    window.addEventListener("scroll", handleScroll);

    return () => {
      window.removeEventListener("scroll", handleScroll);
    };
    */
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
<div id="sh-0" class="flex justify-between w-full h-68">
  header
</div>
<div id="sh-1" class="_1 h-58 w-full md:h-68 top-48 absolute flex items-center md:justify-between w-full left-0 px-4 md:px-80"
>
  <div class="hidden md:block"></div>
  <button class="_6 fx-c w-[14vw] md:hidden!" onclick={ev => {
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
      <button class="_5 absolute fx-c w-40 h-40 rounded-[50%]" onclick={ev => {
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
</style>
