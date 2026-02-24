<script lang="ts">
  import ButtonLayer from "$components/ButtonLayer.svelte";
  import ProductSearchLayer from "$ecommerce/components/ProductSearchLayer.svelte";

  // Keep query text local to the search bar so the layer can react on each keystroke.
  let queryText = $state("");
  let isSearchLayerOpen = $state(false);
  let isSearchInputFocused = $state(false);
  const SEARCH_RESULTS_LIMIT = 20;
  const hasQueryText = $derived(queryText.trim().length > 0);

  const handleSearchKeydown = (event: KeyboardEvent) => {
    const target = event.target as HTMLTextAreaElement;
    if (
      target &&
      (event.key === "Enter" || event.keyCode === 13 || event.charCode === 13)
    ) {
      // Prevent multiline input because search behavior is single-line.
      event.preventDefault();
    }
  };

  const handleSearchInputFocus = () => {
    isSearchInputFocused = true;
    isSearchLayerOpen = true;
  };

  const handleSearchPointerDown = () => {
    // Open on pointer down so first click already shows the layer before focus settles.
    isSearchLayerOpen = true;
  };

  const handleSearchInputBlur = () => {
    isSearchInputFocused = false;
    if (!hasQueryText) {
      isSearchLayerOpen = false;
    }
  };

  $effect(() => {
    if (hasQueryText) {
      isSearchLayerOpen = true;
    }
  });
</script>

<div
  class="_1 relative lg:w-340 w-[34vw]"
>
  <ButtonLayer
    bind:isOpen={isSearchLayerOpen}
    wrapperClass="w-full"
    buttonClass="w-full"
    layerClass="w-[min(940px,calc(100vw-20px))]! rounded-[14px]!"
    contentCss="p-8 search-layer-content-scroll"
    horizontalOffset={-240}
    edgeMargin={10}
    disableTriggerToggle={true}
    useBig
  >
    {#snippet button(_open)}
      <div class="w-full relative">
        <textarea
          class="_2 w-full pl-12 rounded-[16px] pt-10 pl-14"
          cols="1"
          bind:value={queryText}
          onpointerdown={handleSearchPointerDown}
          onkeydown={handleSearchKeydown}
          onfocus={handleSearchInputFocus}
          onblur={handleSearchInputBlur}
          placeholder="Buscar..."
        ></textarea>
        <i class="icon1-search absolute top-8 md:top-9 right-7 md:right-10"></i>
      </div>
    {/snippet}

    {#if !hasQueryText}
      <div class="search-hint-row">Escriba el nombre de un producto...</div>
    {:else}
      <ProductSearchLayer
        queryText={queryText}
        maxResults={SEARCH_RESULTS_LIMIT}
        renderAsInnerContent={true}
      />
    {/if}
  </ButtonLayer>
</div>

<style>

  ._1, ._2 {
    height: 36px;
  }
  ._1 > i {
    color: #6d698d;
  }
  ._2 {
    box-shadow:#20202329 0px 1px 3px, #20202329 0px 1px 2px;
    line-height: 1;
    outline: none;
    border: none;
    background-color: #eae9ef;
    resize: none;
  }
  
  ._2:focus {
  	outline: 2px solid #a7a3ff;
   	background-color: #f6f4ff;
    box-shadow: #b7b7ff63 0px 0 2px 4px, #20202329 0px 1px 2px;
  }

  .search-hint-row {
    color: #4e556d;
    font-size: 14px;
    padding: 10px 8px;
  }

  @media (max-width: 740px) {
    ._1, ._2 {
      height: 36px;
      box-shadow: none;
    }
  }

</style>
