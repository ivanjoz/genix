<script lang="ts">
  import Page from '$domain/Page.svelte';
  import OptionsStrip from '$components/OptionsStrip.svelte';
  import VTable from '$components/vTable/VTable.svelte';
  import { SaleOrdersService, SaleOrderGroup, type ISaleOrder } from './sale_order_status.svelte';
  import type { ITableColumn } from '$components/vTable/types';
  import { ProductosService } from '$routes/negocio/productos/productos.svelte';
  import { formatN } from '$libs/helpers';
    import { untrack } from 'svelte';

  let selectedGroup = $state(SaleOrderGroup.PENDIENTE_DE_PAGO);
  let service = $state<SaleOrdersService | null>(null);

  const productosService = new ProductosService();
  
  // Options for OptionsStrip
  const options = [
    [SaleOrderGroup.PENDIENTE_DE_PAGO, 'Pend. Pago'],
    [SaleOrderGroup.PENDIENTE_DE_ENTREGA, 'Pend. Entrega'],
    [SaleOrderGroup.FINALIZADO, 'Finalizadas']
  ];

  function renderTopProductsSummary(saleOrder: ISaleOrder): string {
    if (!saleOrder.TopPaidProducts || saleOrder.TopPaidProducts.length === 0) {
      return '-';
    }

    // Keep product-name lookup in the view as requested; service only provides ranked IDs/amounts.
    return saleOrder.TopPaidProducts
      .map((topProduct) => {
        const productName = productosService.productosMap.get(topProduct.ProductID)?.Nombre || `Producto #${topProduct.ProductID}`;
        return `${productName} (${formatN(topProduct.LineAmount / 100, 2)})`;
      })
      .join(', ');
  }

  // Instantiate and fetch when the selected group changes
  $effect(() => {
  	selectedGroup;
  	untrack(() => {
 		console.log("productos service::",$state.snapshot(productosService.productos))
   
			service = new SaleOrdersService(selectedGroup);
			service.fetch();
   	})
  });
  
  $effect(() => {
  	service?.records;
   	console.log("Registros obtenidos:",$state.snapshot(service?.records))
  });

  const columns: ITableColumn<ISaleOrder>[] = [
    { 
      header: 'ID', 
      getValue: r => r.ID,
      mobile: { order: 1, css: 'col-span-6', icon: 'tag' }
    },
    { 
      header: 'Total', 
      getValue: r => formatN(r.TotalAmount / 100, 2),
      mobile: { order: 3, css: 'col-span-12', labelLeft: 'Total:' }
    },
    { 
      header: 'Deuda', 
      getValue: r => formatN(r.DebtAmount / 100, 2),
      mobile: { order: 4, css: 'col-span-12', labelLeft: 'Deuda:' }
    },
    {
      header: 'Top Productos',
      id: 'top-products',
      getValue: r => renderTopProductsSummary(r),
      mobile: { order: 5, css: 'col-span-24', labelTop: 'Top Productos', render: r => renderTopProductsSummary(r) }
    },
    {
      header: 'Estado',
      id: 'status',
      getValue: r => {
        switch(r.ss) {
          case 1: return 'Generado';
          case 2: return 'Pagado';
          case 3: return 'Entregado';
          case 4: return 'Finalizado';
          default: return 'Desconocido';
        }
      },
      mobile: { order: 6, css: 'col-span-24' }
    }
  ];

</script>

<Page title="Gestión de Pedidos">
  <div class="p-10">
    <OptionsStrip 
      {options}
      selected={selectedGroup} 
      onSelect={(opt) => selectedGroup = opt[0] as number}
      useMobileGrid
      css="mb-10"
    />
    
    <VTable 
      {columns} 
      data={service?.records || []}
      mobileCardCss="mb-10"
    />
  </div>
</Page>
