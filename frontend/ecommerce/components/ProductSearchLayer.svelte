<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { Env } from "$core/env";
  import { readBuildSunixTimeFromHeader } from "$libs/index-decoder/decoder";
  import { ProductIndex } from "$libs/index-decoder/product-index";
  import type { ProductSearchHit } from "$libs/index-decoder/types";

  interface ProductSearchLayerProps {
    queryText?: string;
    maxResults?: number;
  }

  // Receive search text from the top-bar input and limit visible cards for quick scanning.
  const { queryText = "", maxResults = 12 }: ProductSearchLayerProps = $props();
  const SEARCH_QUERY_THROTTLE_MS = 120;
  const ENABLE_FULL_PRODUCT_SEARCH_DEBUG = Env.PRODUCT_SEARCH_FULL_DEBUG_LOG_ENABLED;

  let productIndexInstance = $state<ProductIndex | null>(null);
  let isProductIndexLoading = $state(false);
  let productIndexLoadErrorMessage = $state("");
  let throttledQueryText = $state("");
  let pendingQueryTimer: ReturnType<typeof setTimeout> | null = null;

  const trimmedQueryText = $derived(queryText.trim());
  const shouldRenderLayer = $derived(trimmedQueryText.length > 0);

  $effect(() => {
    // Debounce query updates so the search index is not recomputed on every keystroke.
    const nextQueryText = trimmedQueryText;
    if (pendingQueryTimer) {
      clearTimeout(pendingQueryTimer);
      pendingQueryTimer = null;
    }
    if (ENABLE_FULL_PRODUCT_SEARCH_DEBUG) {
      console.log("[ProductSearchLayer] Query throttle scheduled", {
        queryText: nextQueryText,
        throttleMs: SEARCH_QUERY_THROTTLE_MS
      });
    }
    pendingQueryTimer = setTimeout(() => {
      throttledQueryText = nextQueryText;
      if (ENABLE_FULL_PRODUCT_SEARCH_DEBUG) {
        console.log("[ProductSearchLayer] Query throttle applied", {
          queryText: throttledQueryText
        });
      }
      pendingQueryTimer = null;
    }, SEARCH_QUERY_THROTTLE_MS);
  });

  const topProductSearchHits = $derived.by<ProductSearchHit[]>(() => {
    if (!productIndexInstance || throttledQueryText.length === 0) {
      return [];
    }
    // Persist and print full ranking-debug payload so sort decisions are inspectable from DevTools.
    const searchStartTimeMs = typeof performance !== "undefined" ? performance.now() : Date.now();
    const rankedHits = productIndexInstance
      .search(throttledQueryText, { enableFullDebugLog: ENABLE_FULL_PRODUCT_SEARCH_DEBUG })
      .slice(0, maxResults);
    const rankingDebugSnapshot = productIndexInstance.getLastSearchDebugSnapshot();
    const searchElapsedMs =
      (typeof performance !== "undefined" ? performance.now() : Date.now()) - searchStartTimeMs;

    const productSearchDebugObject = {
      queryText: throttledQueryText,
      maxResults,
      elapsedMs: Number(searchElapsedMs.toFixed(2)),
      visibleHits: rankedHits.map((searchHit) => ({
        productID: searchHit.product.productID,
        rank: searchHit.rank,
        name: searchHit.product.productNameLossy,
        brand: searchHit.product.brandName
      })),
      rankingDebugSnapshot
    };
    if (ENABLE_FULL_PRODUCT_SEARCH_DEBUG && typeof window !== "undefined") {
      // Expose debug payload globally for direct inspection: window.__GENIX_PRODUCT_SEARCH_DEBUG__.
      (window as any).__GENIX_PRODUCT_SEARCH_DEBUG__ = productSearchDebugObject;
      // Expose ranking-specific payloads so full scoring details are easy to inspect.
      (window as any).__GENIX_PRODUCT_SEARCH_RANKING__ = rankingDebugSnapshot;
      (window as any).__GENIX_PRODUCT_SEARCH_RANKED_PRODUCTS__ =
        rankingDebugSnapshot?.rankedProducts ?? [];
      console.log("[ProductSearchLayer] Product search debug object", productSearchDebugObject);
      console.log("[ProductSearchLayer] Ranking debug snapshot", rankingDebugSnapshot);
      console.log(
        "[ProductSearchLayer] Ranked products debug (first 20)",
        (rankingDebugSnapshot?.rankedProducts ?? []).slice(0, 20)
      );
      console.log(
        "[ProductSearchLayer] Ranked products debug JSON (first 20)",
        JSON.stringify((rankingDebugSnapshot?.rankedProducts ?? []).slice(0, 20), null, 2)
      );
    }
    if (ENABLE_FULL_PRODUCT_SEARCH_DEBUG) {
      console.log("[ProductSearchLayer] Search executed", {
        queryText: throttledQueryText,
        resultsCount: rankedHits.length,
        elapsedMs: Number(searchElapsedMs.toFixed(2))
      });
    }
    return rankedHits;
  });

  // Load and decode products.idx once so each keystroke only runs in-memory search.
  const loadProductIndexFromCDN = async () => {
    if (isProductIndexLoading || productIndexInstance) {
      return;
    }

    isProductIndexLoading = true;
    productIndexLoadErrorMessage = "";
    const productsIndexUrl = Env.makeCDNRoute("live", `c${1}_products.idx`);
    if (ENABLE_FULL_PRODUCT_SEARCH_DEBUG) {
      console.log("[ProductSearchLayer] Fetching product index", { productsIndexUrl });
    }

    try {
      const indexResponse = await fetch(productsIndexUrl);
      if (!indexResponse.ok) {
        throw new Error(`products.idx fetch failed with status=${indexResponse.status}`);
      }

      const indexBinaryBuffer = await indexResponse.arrayBuffer();
      const indexBinaryBytes = new Uint8Array(indexBinaryBuffer);
      const updatedSunix = readBuildSunixTimeFromHeader(indexBinaryBytes);
      const updatedUnix = updatedSunix * 2 + 1_000_000_000;
      const createdAtISODate = new Date(updatedUnix * 1000).toISOString();

      productIndexInstance = new ProductIndex(indexBinaryBytes);
      if (ENABLE_FULL_PRODUCT_SEARCH_DEBUG) {
        console.info("[ProductSearchLayer] Product index loaded", {
          productsCount: productIndexInstance.size,
          updatedSunix,
          updatedUnix,
          createdAtISODate
        });
      }
    } catch (loadError) {
      productIndexLoadErrorMessage =
        loadError instanceof Error ? loadError.message : "Unknown error loading product index";
      console.error("[ProductSearchLayer] Product index load error", loadError);
    } finally {
      isProductIndexLoading = false;
    }
  };

  onMount(() => {
    void loadProductIndexFromCDN();
  });

  onDestroy(() => {
    // Ensure no delayed query update runs after this layer is removed.
    if (pendingQueryTimer) {
      clearTimeout(pendingQueryTimer);
      pendingQueryTimer = null;
    }
  });
</script>

{#if shouldRenderLayer}
  <div class="search-layer" role="dialog" aria-label="Resultados de busqueda de productos">
    {#if isProductIndexLoading}
      <div class="status-row">Cargando indice de productos...</div>
    {:else if productIndexLoadErrorMessage}
      <div class="status-row error-state">No se pudo cargar el indice ({productIndexLoadErrorMessage}).</div>
    {:else if topProductSearchHits.length === 0}
      <div class="status-row">Sin resultados para "{trimmedQueryText}".</div>
    {:else}
      <div class="results-grid">
        {#each topProductSearchHits as searchHit (searchHit.product.productID)}
          <article class="result-card">
            <div class="result-name" title={searchHit.product.productNameLossy}>
              {searchHit.product.productNameLossy}
            </div>
            <div class="result-brand" title={searchHit.product.brandName || "Sin marca"}>
              {searchHit.product.brandName || "Sin marca"}
            </div>
          </article>
        {/each}
      </div>
    {/if}
  </div>
{/if}

<style>
  .search-layer {
    position: absolute;
    top: calc(100% + 10px);
    left: 50%;
    transform: translateX(-50%);
    width: min(940px, calc(100vw - 20px));
    max-height: min(68vh, 620px);
    overflow-y: auto;
    background: #fff;
    border: 1px solid #e3e7f0;
    border-radius: 14px;
    padding: 12px;
    box-shadow: rgba(33, 39, 55, 0.2) 0 10px 28px;
    z-index: 350;
  }

  .status-row {
    color: #4e556d;
    font-size: 14px;
    padding: 10px 8px;
  }

  .error-state {
    color: #be3131;
  }

  .results-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 10px;
  }

  .result-card {
    border: 1px solid #e4e7ef;
    border-radius: 10px;
    background: #f9fbff;
    padding: 10px;
    min-height: 88px;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
  }

  .result-name {
    color: #1d253b;
    font-weight: 600;
    line-height: 1.2;
    font-size: 14px;
    display: -webkit-box;
    line-clamp: 2;
    overflow: hidden;
    text-overflow: ellipsis;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }

  .result-brand {
    color: #63708e;
    font-size: 12px;
    margin-top: 8px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  @media (max-width: 1140px) {
    .results-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

  @media (max-width: 740px) {
    .search-layer {
      top: calc(100% + 8px);
      width: min(94vw, 580px);
      padding: 10px;
      border-radius: 12px;
    }

    .results-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 8px;
    }

    .result-card {
      min-height: 80px;
      padding: 9px;
    }
  }
</style>
