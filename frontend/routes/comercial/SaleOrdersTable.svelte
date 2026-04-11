<script lang="ts" generics="T extends ISaleOrderTableRecord">
  import InlineButton from '$components/micro/InlineButton.svelte';
  import VTable from '$components/vTable/VTable.svelte';
  import type { ITableColumn } from '$components/vTable/types';
  import { formatN, formatTime } from '$libs/helpers';

  interface ISaleOrderTableRecord {
    ID: number;
    ClientID?: number;
    Created: number;
    DetailProductsIDs?: number[];
    DetailPrices?: number[];
    DetailQuantities?: number[];
    TotalAmount?: number;
    DebtAmount?: number;
    ss: number;
  }

  interface ITopProductCard {
    productID: number;
    productName: string;
    totalQuantity: number;
    totalAmount: number;
  }

  interface SaleOrdersTableProps<T extends ISaleOrderTableRecord> {
    data: T[];
    getProductName: (productID: number) => string;
    getClientName?: (saleOrder: T) => string;
    selected?: T | number;
    isSelected?: (saleOrder: T, selected: T | number) => boolean;
    onRowClick?: (saleOrder: T, index: number) => void;
    css?: string;
    maxHeight?: string;
    filterText?: string;
    getFilterContent?: (saleOrder: T) => string;
    emptyMessage?: string;
    showClientColumn?: boolean;
  }

  let {
    data,
    getProductName,
    getClientName,
    selected,
    isSelected,
    onRowClick,
    css = '',
    maxHeight,
    filterText = '',
    getFilterContent,
    emptyMessage,
    showClientColumn = true,
  }: SaleOrdersTableProps<T> = $props();

  function isSaleOrderPaid(saleOrder: T): boolean {
    return saleOrder.ss === 2 || saleOrder.ss === 4;
  }

  function isSaleOrderDelivered(saleOrder: T): boolean {
    return saleOrder.ss === 3 || saleOrder.ss === 4;
  }

  function getTopProductsCards(saleOrder: T): ITopProductCard[] {
    const detailCount = Math.min(
      saleOrder.DetailProductsIDs?.length || 0,
      saleOrder.DetailPrices?.length || 0,
      saleOrder.DetailQuantities?.length || 0,
    );
    if (detailCount <= 0) {
      return [];
    }

    const aggregatedProductTotals = new Map<number, { totalAmount: number; totalQuantity: number }>();

    // Aggregate once from detail arrays so both pages render the same ranking and quantities.
    for (let detailIndex = 0; detailIndex < detailCount; detailIndex += 1) {
      const productID = saleOrder.DetailProductsIDs?.[detailIndex] || 0;
      const unitPrice = saleOrder.DetailPrices?.[detailIndex] || 0;
      const quantity = saleOrder.DetailQuantities?.[detailIndex] || 0;
      const lineAmount = unitPrice * quantity;
      if (!productID || quantity <= 0 || lineAmount <= 0) {
        continue;
      }

      const previousTotals = aggregatedProductTotals.get(productID) || { totalAmount: 0, totalQuantity: 0 };
      previousTotals.totalAmount += lineAmount;
      previousTotals.totalQuantity += quantity;
      aggregatedProductTotals.set(productID, previousTotals);
    }

    return Array.from(aggregatedProductTotals.entries())
      .sort((leftProduct, rightProduct) => {
        if (rightProduct[1].totalAmount !== leftProduct[1].totalAmount) {
          return rightProduct[1].totalAmount - leftProduct[1].totalAmount;
        }
        return leftProduct[0] - rightProduct[0];
      })
      .slice(0, 3)
      .map(([productID, productTotals]) => ({
        productID,
        productName: getProductName(productID),
        totalQuantity: productTotals.totalQuantity,
        totalAmount: productTotals.totalAmount,
      }));
  }

  function getTopProductsSummary(saleOrder: T): string {
    const topProductsCards = getTopProductsCards(saleOrder);
    if (topProductsCards.length === 0) {
      return '-';
    }

    return topProductsCards
      .map((topProductCard) => {
        return `${topProductCard.totalQuantity}x ${topProductCard.productName} (${formatN(topProductCard.totalAmount / 100, 2)})`;
      })
      .join(', ');
  }

  // Keep the shared sales-order columns in one place so both pages render the same structure.
  const columns = $derived.by(() => {
    const saleOrderColumns: ITableColumn<T>[] = [
      {
        header: 'ID',
        getValue: saleOrder => saleOrder.ID,
        css: 'ff-mono fs15 text-right',
        headerCss: 'w-52',
        mobile: { order: 1, css: 'col-span-6', icon: 'tag' }
      },
      {
        header: 'Fecha Hora',
        getValue: saleOrder => formatTime(saleOrder.Created, 'd-M h:n') as string,
        css: 'text-right',
        headerCss: 'w-100',
        cellCss: 'px-6 whitespace-nowrap',
        mobile: { order: 2, css: 'col-span-6', icon: 'clock' }
      },
      {
        header: 'Entregado /Pagado',
        id: 'delivery-payment-status',
        headerCss: 'w-82',
        css: 'text-center',
        mobile: {
          order: 4,
          css: 'col-span-24'
        }
      },
      {
        header: 'Total',
        css: 'ff-mono text-right',
        getValue: saleOrder => formatN((saleOrder.TotalAmount || 0) / 100, 2),
        mobile: { order: 5, css: 'col-span-12', labelLeft: 'Total:' }
      },
      {
        header: 'Deuda',
        css: 'ff-mono text-right',
        getValue: saleOrder => formatN((saleOrder.DebtAmount || 0) / 100, 2),
        mobile: { order: 6, css: 'col-span-12', labelLeft: 'Deuda:' }
      },
    ];

    if (showClientColumn) {
      saleOrderColumns.push({
        header: 'Cliente',
        getValue: saleOrder => getClientName?.(saleOrder) || '-',
        headerCss: 'w-220',
        cellCss: 'px-6 line-clamp-2 whitespace-nowrap',
        mobile: { order: 3, css: 'col-span-24', labelTop: 'Cliente' }
      });
    }

    saleOrderColumns.push({
      header: 'Top Productos',
      headerCss: 'w-[60%]',
      css: 'px-0 py-0',
      cellCss: 'px-0 py-0 align-top',
      id: 'top-products',
      mobile: {
        order: 7,
        css: 'col-span-24',
        labelTop: 'Top Productos',
        render: saleOrder => getTopProductsSummary(saleOrder)
      }
    });

    return saleOrderColumns;
  });

  function getSaleOrderFilterContent(saleOrder: T): string {
    const extraFilterContent = getFilterContent?.(saleOrder) || '';
    return [
      saleOrder.ID,
      getClientName?.(saleOrder) || '',
      getTopProductsSummary(saleOrder),
      extraFilterContent
    ].join(' ').toLowerCase();
  }
</script>

<VTable
  data={data}
  {columns}
  {selected}
  {isSelected}
  {onRowClick}
  {css}
  {maxHeight}
  {emptyMessage}
  {filterText}
  getFilterContent={getSaleOrderFilterContent}
  estimateSize={38}
  mobileCardCss="mb-10"
  tableCss="w-full"
>
  {#snippet cellRenderer(saleOrder: T, columnDefinition: ITableColumn<T>, _defaultValue: string)}
    {#if columnDefinition.id === 'delivery-payment-status'}
      <div class="flex items-center justify-center gap-6">
        <!-- Keep both state toggles adjacent so operators can scan the workflow quickly. -->
        <InlineButton label="P" color="green" mode={isSaleOrderPaid(saleOrder) ? 'checked' : 'default'} />
        <InlineButton label="E" color="blue" mode={isSaleOrderDelivered(saleOrder) ? 'checked' : 'default'} />
      </div>
    {:else if columnDefinition.id === 'top-products'}
      {@const topProductsCards = getTopProductsCards(saleOrder)}
      <div class="flex w-full min-w-0">
        {#if topProductsCards.length === 0}
          <div class="px-8 py-6 text-gray-400">-</div>
        {:else}
          {#each topProductsCards as topProductCard, productIndex (topProductCard.productID)}
            <div
              class={`min-w-0 flex items-center shrink-0 basis-[25%] px-10 ${productIndex < topProductsCards.length - 1 ? 'border-r border-slate-200' : ''}`}
            >
              <!-- Group quantity and amount to preserve the original scan order inside each card. -->
              <div class="mr-4 min-w-48 flex flex-col items-end justify-center">
                <div class="text-sm ff-mono leading-[1] mb-1 text-slate-900">{topProductCard.totalQuantity}x</div>
                <div class="ff-bold text-sm leading-[1] text-blue-600">
                  {formatN(topProductCard.totalAmount / 100, 2)}
                </div>
              </div>
              <!-- Clamp product names so the table row keeps a stable height. -->
              <div class="line-clamp-2 text-sm leading-[1.15] text-slate-800">
                {topProductCard.productName}
              </div>
            </div>
          {/each}
        {/if}
      </div>
    {/if}
  {/snippet}
</VTable>
