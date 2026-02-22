<script lang="ts">
  import { getRecordWithCache, type IRecordRef, type IMinimalRecord } from "$libs/cache/cache-by-ids.svelte";
  import { formatN } from "$libs/helpers";

  interface ProductSearchResultCardProps {
    productID: number;
    fallbackName?: string;
    fallbackBrand?: string;
    fallbackCategory?: string;
  }

  interface ISearchProductRecord extends IMinimalRecord {
    Nombre?: string;
    Precio?: number;
    PrecioFinal?: number;
    MarcaNombre?: string;
    _marcaNombre?: string;
    CategoriaNombre?: string;
    _categoriaNombre?: string;
  }

  const {
    productID,
    fallbackName = "",
    fallbackBrand = "",
    fallbackCategory = ""
  }: ProductSearchResultCardProps = $props();

  let productRecordRef = $state<IRecordRef<ISearchProductRecord> | null>(null);

  const fetchedProductRecord = $derived(productRecordRef?.record || null);
  const isLoadingRecord = $derived(productRecordRef?.loading || false);

  const resolvedProductName = $derived(
    fetchedProductRecord?.Nombre || fallbackName || `Producto #${productID}`
  );

  const resolvedProductPriceCents = $derived(
    fetchedProductRecord?.PrecioFinal || fetchedProductRecord?.Precio || 0
  );

  const resolvedProductPriceText = $derived(
    resolvedProductPriceCents > 0
      ? `S/. ${formatN(resolvedProductPriceCents / 100, 2)}`
      : "Precio no disponible"
  );

  const resolvedBrandOrCategory = $derived(
    fetchedProductRecord?.MarcaNombre ||
      fetchedProductRecord?._marcaNombre ||
      fetchedProductRecord?.CategoriaNombre ||
      fetchedProductRecord?._categoriaNombre ||
      fallbackBrand ||
      fallbackCategory ||
      "Sin marca"
  );

  $effect(() => {
    // Card consumes a small reactive ref. Local-hit render is immediate; stale data refreshes in background.
    const selectedProductID = productID;
    if (!(selectedProductID > 0)) {
      productRecordRef = null;
      return;
    }
    productRecordRef = getRecordWithCache<ISearchProductRecord>("productos-ids", selectedProductID);
  });
</script>

<article class="result-card" aria-busy={isLoadingRecord}>
  <div class="result-name-row">
    <div class="result-name" title={resolvedProductName}>
      {resolvedProductName}
    </div>
    {#if isLoadingRecord}
      <span class="loading-spinner" aria-label="Cargando producto"></span>
    {/if}
  </div>
  <div class="result-meta-row">
    <div class="result-brand-or-category" title={resolvedBrandOrCategory}>
      {resolvedBrandOrCategory}
    </div>
    <div class="result-price" title={resolvedProductPriceText}>
      {resolvedProductPriceText}
    </div>
  </div>
</article>

<style>
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

  .result-name-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .loading-spinner {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    border: 2px solid #c6d0e2;
    border-top-color: #365f90;
    animation: card-spin 0.8s linear infinite;
    flex: 0 0 auto;
  }

  .result-meta-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-top: 8px;
  }

  .result-brand-or-category {
    color: #5f6d88;
    font-size: 12px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .result-price {
    color: #375680;
    font-size: 12px;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  @keyframes card-spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  @media (max-width: 740px) {
    .result-card {
      min-height: 80px;
      padding: 9px;
    }
  }
</style>
