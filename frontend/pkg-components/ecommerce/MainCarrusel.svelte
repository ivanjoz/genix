<script lang="ts">
  export let categorias: ICategoriaProducto[] = [];

  import emblaCarouselSvelte from "embla-carousel-svelte";
  import s1 from "./styles.module.css";
  import type { ICategoriaProducto } from "./store.svelte.ts";

  let emblaApi;
  let options = { loop: false };

  function onInit(event: any) {
    emblaApi = event.detail;
    console.log(emblaApi.slideNodes()); // Access API
  }
</script>

<div class="flex items-center  p-4 md:p-12">
  <div class="flex w-full justify-center items-center flex-wrap">
    {#each categorias as e}
      <div
        class={"flex flex-col w-[calc(50vw-16px)] h-160 md:h-240 md:w-240 m-6 md:m-12 " +
          s1.producto_caregoria_card}
      >
        <div class="w-full h-[calc(100%-40px)] relative px-12 pt-16">
          <img class="w-full h-full object-contain" src={e.Image} alt="" />
        </div>
        <div
          class="text-[20px] h-40 w-full flex justify-center items-center mt-auto"
        >
          {e.Name}
        </div>
      </div>
    {/each}
  </div>
</div>
<div class={"relative flex w-full h-700 " + s1.main_carrusel_ctn}>
  <div class="flex w-420">
    <h2>demo 1</h2>
  </div>
  <div
    class="embla h-full grow-1"
    use:emblaCarouselSvelte={{ options } as any}
    onemblaInit={onInit}
  >
    <div class="embla__container h-full">
      <div class="embla__slide s1">Slide 1</div>
      <div class="embla__slide s2">Slide 2</div>
      <div class="embla__slide">Slide 3</div>
    </div>
  </div>
</div>

<style>
  .embla__slide.s1 {
    background-color: rebeccapurple;
  }
  .embla__slide.s2 {
    background-color: orange;
  }
  .embla {
    overflow: hidden;
  }
  .embla__container {
    display: flex;
  }
  .embla__slide {
    flex: 0 0 100%;
    min-width: 0;
  }
</style>
