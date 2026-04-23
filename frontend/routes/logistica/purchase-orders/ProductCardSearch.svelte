<script lang="ts" module>
import type { ProductosService, IProducto, IProductoPresentacion } from '$routes/negocio/productos/productos.svelte'
import type { ClientProviderService } from '$routes/negocio/clientes/clientes-proveedores.svelte'
import type { ProductSupplyService } from '$routes/logistica/products-stock/supply-management.svelte'

export interface IProductCard {
  key: string
  productID: number
  presentationID: number
  productName: string
  presentationName: string
  displayName: string
  sku: string
  // Default unit price for adding to cart (lowest provider price, 0 when unknown).
  price: number
  priceMin: number
  priceMax: number
  priceLabel: string
  searchText: string
  producto: IProducto
  presentation?: IProductoPresentacion
}

export interface ProductCardSearchProps {
  productosService: ProductosService
  productSupplyService?: ProductSupplyService
  providersService?: ClientProviderService
  displayProviderFilter?: boolean
  selectedKeys?: Map<string, number>
  // Map of productID -> current total stock; missing keys render as 0.
  stockMap?: Map<number, number>
  onSelect: (productCard: IProductCard, cant?: number) => void
  height?: string
  maxColumns?: number
  estimatedRowHeight?: number
  searchPlaceholder?: string
  providerPlaceholder?: string
  emptyMessage?: string
}
</script>

<script lang="ts">
import SearchSelect from '$components/SearchSelect.svelte'
import VirtualCards from '$components/VirtualCards.svelte'
import { formatN, wordInclude } from '$libs/helpers'

  let {
    productosService,
    productSupplyService,
    providersService,
    displayProviderFilter = false,
    selectedKeys,
    stockMap,
    onSelect,
    height = 'calc(100vh - 76px - var(--header-height))',
    maxColumns = 2,
    estimatedRowHeight = 80,
    searchPlaceholder = 'PRODUCTO / SKU...',
    providerPlaceholder = 'PROVEEDOR',
    emptyMessage = 'No se encontraron productos',
  }: ProductCardSearchProps = $props()

  let filterText = $state('')
  let providerSelectedID = $state(0)

  const formatMoney = (cents: number) => formatN(cents / 100, 2)

  // Resolve purchase price range per product from supply records — range collapses to a single value when prices match.
  const productPriceMap = $derived.by(() => {
    const priceByProduct = new Map<number, { min: number, max: number, label: string }>()
    if (!productSupplyService) { return priceByProduct }

    for (const supplyRecord of productSupplyService.records) {
      const providerPrices = (supplyRecord.ProviderSupply || [])
        .map((providerSupplyRow) => providerSupplyRow.Price || 0)
        .filter((price) => price > 0)
      if (providerPrices.length === 0) { continue }

      const minPrice = Math.min(...providerPrices)
      const maxPrice = Math.max(...providerPrices)
      const label = minPrice === maxPrice
        ? formatMoney(minPrice)
        : `${formatMoney(minPrice)} → ${formatMoney(maxPrice)}`
      priceByProduct.set(supplyRecord.ProductID, { min: minPrice, max: maxPrice, label })
    }
    return priceByProduct
  })

  // Build a flat list of cards: one per product or per active presentation if any.
  const productCards = $derived.by((): IProductCard[] => {
    const cards: IProductCard[] = []
    for (const producto of productosService.records) {
      if ((producto.ss || 0) <= 0) { continue }
      const productSku = (producto.SKU || '').trim()
      const activePresentations = (producto.Presentaciones || []).filter((presentation) => (presentation.ss || 0) > 0)
      const priceInfo = productPriceMap.get(producto.ID)
      const priceMin = priceInfo?.min || 0
      const priceMax = priceInfo?.max || 0
      const priceLabel = priceInfo?.label || ''

      if (activePresentations.length === 0) {
        cards.push({
          key: `${producto.ID}_0`,
          productID: producto.ID,
          presentationID: 0,
          productName: producto.Nombre,
          presentationName: '',
          displayName: producto.Nombre,
          sku: productSku,
          price: priceMin,
          priceMin,
          priceMax,
          priceLabel,
          searchText: `${producto.Nombre} ${productSku}`.toLowerCase(),
          producto,
        })
        continue
      }

      for (const presentation of activePresentations) {
        const presentationSku = (presentation.sk || '').trim()
        const displayName = presentation.nm
          ? `${producto.Nombre} (${presentation.nm})`
          : producto.Nombre
        cards.push({
          key: `${producto.ID}_${presentation.id}`,
          productID: producto.ID,
          presentationID: presentation.id,
          productName: producto.Nombre,
          presentationName: presentation.nm || '',
          displayName,
          sku: presentationSku || productSku,
          price: priceMin,
          priceMin,
          priceMax,
          priceLabel,
          searchText: `${producto.Nombre} ${presentation.nm || ''} ${presentationSku} ${productSku}`.toLowerCase(),
          producto,
          presentation,
        })
      }
    }
    return cards
  })

  const productIDsForProvider = $derived.by(() => {
    if (!displayProviderFilter || !providerSelectedID || !productSupplyService) {
      return null
    }
    const productIDs = new Set<number>()
    for (const supplyRecord of productSupplyService.records) {
      const matches = (supplyRecord.ProviderSupply || []).some((providerSupplyRow) => {
        return providerSupplyRow.ProviderID === providerSelectedID
      })
      if (matches) { productIDs.add(supplyRecord.ProductID) }
    }
    return productIDs
  })

  const filteredCards = $derived.by(() => {
    const text = filterText.trim().toLowerCase()
    const terms = text ? text.split(' ').filter(Boolean) : []

    return productCards.filter((card) => {
      if (productIDsForProvider && !productIDsForProvider.has(card.productID)) { return false }
      if (terms.length === 0) { return true }
      return wordInclude(card.searchText, terms)
    })
  })

  const providerOptions = $derived.by(() => {
    if (!providersService) { return [] }
    return providersService.records.filter((provider) => (provider.ss || 0) > 0)
  })

  // Highlight matched text inline so users can see why a card was returned.
  const highlightHtml = (text: string, search: string) => {
    if (!search || !text) { return text }
    const escaped = search.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    return text.replace(new RegExp(`(${escaped})`, 'gi'), '<span class="bg-yellow-200 text-black font-bold">$1</span>')
  }
</script>

<div class="flex flex-col h-full min-h-0">
  <div class="mb-12 flex gap-6 md:gap-12">
    <div class={displayProviderFilter ? 'w-[60%] md:flex-1' : 'w-full md:flex-1'}>
      <input type="text"
        class="w-full px-12 py-8 bg-gray-50 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
        placeholder={searchPlaceholder}
        value={filterText}
        oninput={(ev) => { filterText = ev.currentTarget.value }}
      />
    </div>
    {#if displayProviderFilter}
      <div class="w-[40%] md:w-280">
        <SearchSelect
          label=""
          keyId="ID"
          keyName="Name"
          options={providerOptions}
          selected={providerSelectedID}
          placeholder={providerPlaceholder}
          onChange={(provider) => { providerSelectedID = provider?.ID || 0 }}
        />
      </div>
    {/if}
  </div>

  <div class="flex-1 min-h-0">
    <VirtualCards
      items={filteredCards}
      {height}
      maxColumns={maxColumns}
      mobileBreakpointPx={920}
      estimatedRowHeight={75}
      bufferSize={6}
      columnGapPx={8}
      rowGapPx={6}
      containerCss="h-full"
      emptyMessage={emptyMessage}
      useInnerPadding
    >
      {#snippet children(card, _idx)}
        {@const inCart = selectedKeys?.get(card.key) || 0}
        {@const stock = stockMap?.get(card.productID) || 0}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="relative min-h-58 px-8 py-4 border border-transparent rounded-lg bg-white hover:border-gray-200 hover:shadow-sm cursor-pointer group"
          onclick={() => onSelect(card, 1)}
        >
          {#if inCart > 0}
            <div class="absolute -right-4 -top-4 z-20 flex h-22 min-w-22 items-center justify-center rounded-full bg-blue-600 px-6 text-[11px] font-bold text-white shadow">
              {inCart}
            </div>
          {/if}
          <div class="flex flex-col gap-2">
            <div class="flex items-center gap-8">
              <div class="flex items-center gap-8 pr-20 leading-tight text-gray-700 md:pr-0">
                <span>{@html highlightHtml(card.productName, filterText)}</span>
                {#if card.presentationName}
                  <span class="text-blue-600 font-bold">({@html highlightHtml(card.presentationName, filterText)})</span>
                {/if}
              </div>
              <div class="h-[1px] bg-gray-200 grow group-hover:bg-gray-300 transition-colors"></div>
            </div>

            <div class="flex items-center gap-8">
              <div class="flex items-center gap-6 leading-none">
                <span class={'font-mono text-[12px] ' + (stock > 0 ? 'text-gray-700' : 'text-red-500')}>
                  {stock}
                </span>
                {#if card.sku}
                  <span class="text-[11px] font-mono text-gray-400">{@html highlightHtml(card.sku, filterText)}</span>
                {/if}
              </div>
              <div class="font-mono ml-auto text-sm font-medium text-gray-700 text-right whitespace-nowrap">
                {card.priceLabel || '—'}
              </div>
            </div>
          </div>
        </div>
      {/snippet}
    </VirtualCards>
  </div>
</div>
