<script lang="ts">
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify';
  import { useUI } from '@genix/ui';
  const ui = useUI();
import Button from '#components/buttons/Button.svelte';
import Checkbox from '#components/form/Checkbox.svelte';
import FilterInput from '#components/form/FilterInput.svelte';
import Input from '#components/form/Input.svelte';
import SearchSelect from '#components/form/SearchSelect.svelte';
import Layer from '#components/layers/Layer.svelte';
import CardsList from '#components/vTable/CardsList.svelte';
import VTable from '#components/vTable/VTable.svelte';
import type { ICardCell } from '#components/vTable/types.ts';
import { productoMonedaOptions } from '#core/products-lists.ts';
import { Core, tr } from '#core/store.svelte.ts';
import T from '#components/misc/T.svelte';
import Page from '#domain/Page.svelte';
import PCGEAccountSelect from '#domain/PCGEAccountSelect.svelte';
import { PCGE_KIND_SUPPLY, pcgeAccountLabel } from '#core/pcge.ts';
import type { ExcelTableColumn } from '@genix/ui/excel';
import { formatN } from '#libs/helpers.ts';
import { SharedListsService } from '#services/business/shared-lists.svelte.ts';
import { ClientProviderService, ClientProviderType } from '#services/crm/client-provider.svelte.ts';
import {
  createEmptyProviderSupplyRow,
  normalizeProviderSupplyRows,
  ProductSupplyService,
  type IProductSupplyProviderRow,
} from '#routes/logistics/purchase-management/supply-management.svelte.ts';
import ProductProductionTab from '#routes/production/products/ProductProductionTab.svelte';
import { clearProductionFields, copyProductionFields, findProductionError } from '#routes/production/products/product-production.ts';
import {
  depreciationTerms,
  SUPPLY_ORIGIN_SEMI_FINISHED,
  SUPPLY_PRODUCT_STATUS,
  SupplyMaterialService,
  type ISupplyMaterial,
} from './supply-material.svelte';

  // Insumos catalog (master data) + Marcas list for the BrandID picker.
  const supplies = new SupplyMaterialService(true);
  // Minimum stock and provider rows live in product_supply, keyed by ProductID.
  const supplyConfigs = new ProductSupplyService(true);
  // ListaID=2 corresponds to "Marcas" (same convention used by the productos page).
  const listas = new SharedListsService([2], true);
  // Reuse the shared provider catalog so the picker mirrors the gestion-compras flow.
  const providers = new ClientProviderService(ClientProviderType.PROVIDER, true);

  let filterText = $state("");
  let supplyForm = $state({} as ISupplyMaterial);
  // 1 = the supply's data, 2 = the recipe of a semi-finished supply.
  let layerView = $state(1);
  // A semi-finished supply is made by the company (docs/MANUFACTURING_PLAN.md §2.1): it has a recipe
  // and its stock account is 231 by its origin, so the depreciation and account fields do not apply.
  const isSemiFinishedForm = $derived(supplyForm.Origin === SUPPLY_ORIGIN_SEMI_FINISHED);

  const depreciationOptions = $derived(
    depreciationTerms.map(term => ({ id: term.id, name: tr(term.label) })),
  );

  // Lookup brand name by ID for the table cell — reuses the same shared list as productos.
  const brandLabelById = $derived.by(() => {
    const records = listas.ListaRecordsMap.get(2) || [];
    return new Map(records.map((brandRecord) => [brandRecord.ID, brandRecord.Name]));
  });

  const supplyColumns: ExcelTableColumn<ISupplyMaterial>[] = [
    {
      header: "ID",
      field: "ID",
      css: "text-blue-fg text-center",
      headerCss: "w-40", headerInnerCss: "min-w-36",
      getValue: (record) => record.ID || "",
      render: (record) => record.ID ? String(record.ID) : `<i class="icon-[fa--refresh]"></i>`,
      mobile: { order: 1, css: "col-span-6 ff-bold", icon: "[fa--tag]" },
    },
    {
      header: "Supply|Insumo",
      highlight: true,
      field: "Name",
      getValue: (record) => record.Name,
      mobile: { order: 2, css: "col-span-18", render: (record) => `<strong>${record.Name}</strong>` },
    },
    {
      header: "SKU",
      field: "SKU",
      getValue: (record) => record.SKU || "",
      mobile: { order: 3, css: "col-span-12", labelLeft: "SKU:" },
    },
    {
      header: "Brand|Marca",
      getValue: (record) => brandLabelById.get(record.BrandID) || "",
      mobile: { order: 4, css: "col-span-12", labelLeft: "Marca:" },
    },
    {
      header: "Type|Tipo",
      getValue: (record) => record.Origin === SUPPLY_ORIGIN_SEMI_FINISHED ? tr("Semi-finished|Semielaborado")
        : record.DepreciationMonths > 0 ? tr("Asset|Activo") : tr("Consumable|Consumible"),
      mobile: { order: 6, css: "col-span-12", labelLeft: "Tipo:" },
    },
    {
      header: "Account|Cuenta",
      getValue: (record) => record.AccountCode ? pcgeAccountLabel(record.AccountCode) : "",
      mobile: { order: 7, css: "col-span-24", labelLeft: "Cuenta:" },
    },
    {
      header: "Price|Precio",
      css: "text-right",
      // Backend stores Price in cents; UI shows two-decimal currency.
      getValue: (record) => formatN(record.Price / 100, 2),
      field: "Price",
      mobile: { order: 5, css: "col-span-12", labelLeft: "Precio:", render: (record) => formatN(record.Price / 100, 2) },
    },
  ];

  // Provider rows for this supply — same shape and UX as the gestion-compras "Proveedores" list.
  // Wrapped in $derived so the picker options refresh after providers.records resolves.
  const providerSupplyCards = $derived.by((): ICardCell<IProductSupplyProviderRow>[] => {
    return [
      {
        label: 'Proveedor',
        field: 'ProviderID',
        itemCss: 'col-span-24 md:col-span-12',
        cellOptions: providers.records || [],
        cellOptionsKeyId: 'ID',
        cellOptionsKeyName: 'Name',
        onCellSelect: (providerSupplyRow, selectedValue) => {
          providerSupplyRow.ProviderID = Number(selectedValue || 0);
        },
      },
      {
        label: 'Capacidad',
        field: 'Capacity',
        type: 'number',
        itemCss: 'col-span-12 md:col-span-4',
        contentCss: 'w-full justify-end text-right pr-6',
        inputCss: 'text-right pr-6',
        getValue: (providerSupplyRow) => providerSupplyRow.Capacity || 0,
        onCellEdit: (providerSupplyRow, nextValue) => {
          providerSupplyRow.Capacity = parseInt(String(nextValue || '0'));
        },
      },
      {
        label: 'Entrega',
        field: 'DeliveryTime',
        type: 'number',
        itemCss: 'col-span-12 md:col-span-4',
        contentCss: 'w-full justify-end text-right pr-6',
        inputCss: 'text-right pr-6',
        getValue: (providerSupplyRow) => providerSupplyRow.DeliveryTime || 0,
        onCellEdit: (providerSupplyRow, nextValue) => {
          providerSupplyRow.DeliveryTime = parseInt(String(nextValue || '0'));
        },
      },
      {
        label: 'Precio',
        field: 'Price',
        itemCss: 'col-span-12 md:col-span-4',
        contentCss: 'w-full justify-end text-right pr-6',
        inputCss: 'text-right pr-6',
        // Backend stores Price in cents — render and parse the two-decimal currency view.
        getValue: (providerSupplyRow) => formatN((providerSupplyRow.Price || 0) / 100, 2),
        render: (providerSupplyRow) => providerSupplyRow.Price ? formatN((providerSupplyRow.Price || 0) / 100, 2) : "",
        onCellEdit: (providerSupplyRow, nextValue) => {
          const parsedPrice = parseFloat(String(nextValue || '0'));
          providerSupplyRow.Price = Math.round((parsedPrice || 0) * 100);
        },
      },
    ];
  });

  const addProviderSupplyRow = () => {
    supplyForm.ProviderSupply = [
      ...(supplyForm.ProviderSupply || []),
      createEmptyProviderSupplyRow(),
    ];
  };

  const removeProviderSupplyRow = (providerRowIndex: number) => {
    const currentProviderSupplyRows = [...(supplyForm.ProviderSupply || [])];
    currentProviderSupplyRows.splice(providerRowIndex, 1);
    supplyForm.ProviderSupply = currentProviderSupplyRows;
  };

  const doPostSupplies = async (payload: ISupplyMaterial[]) => {
    return await supplies.postAndSync(payload);
  };

  const onSave = async (isDelete?: boolean) => {
    if ((supplyForm.Name?.length || 0) < 2) {
      notifyFailure(tr("Name must be at least 2 characters.|El nombre debe tener al menos 2 caracteres."));
      return;
    }
    // Soft-delete: backend evicts records with ss=0 from the active list on next sync.
    if (isDelete) { supplyForm.ss = 0; }
    // Strip empty rows so the backend only persists meaningful provider entries.
    supplyForm.ProviderSupply = normalizeProviderSupplyRows(supplyForm.ProviderSupply || []);
    // Only a semi-finished supply has a recipe; its errors send the user to the tab to fix them.
    if (!isSemiFinishedForm) { clearProductionFields(supplyForm); }
    const productionError = isSemiFinishedForm ? findProductionError(supplyForm) : undefined;
    if (productionError && !isDelete) {
      notifyFailure(tr(productionError));
      layerView = 2;
      return;
    }

    showLoading(tr("Saving supply...|Guardando insumo..."));
    try {
      await doPostSupplies([supplyForm]);
    } catch (error) {
      notifyFailure(error as string);
      hideLoading();
      return;
    }
    hideLoading();
    supplyForm = {} as ISupplyMaterial;
    ui.openSideLayer(0);
  };
</script>

<Page title="Supplies & Materials|Insumos & Materiales">
  <div class="grid grid-cols-12 md:flex md:flex-row items-center mb-8">
    <FilterInput label="Filter supplies|Filtrar insumos"
      css="w-full md:w-200 col-span-9"
      icon="icon-[fa--search]"
      bind:value={filterText}
    />
    <Button name="New|Nuevo" label="Shows the form to create a new supply in a side layer."
      color="green"
      icon="icon-[fa--plus]"
      hideNameOnMobile
      css="col-span-3 ml-auto"
      onClick={() => {
        supplyForm = clearProductionFields({
          ss: SUPPLY_PRODUCT_STATUS, DepreciationMonths: 0, ProviderSupply: [], Origin: 0,
        } as unknown as ISupplyMaterial);
        layerView = 1;
        ui.openSideLayer(1);
      }}
    />
  </div>

  <Layer type="content">
    <VTable
      columns={supplyColumns}
      data={supplies.records}
      {filterText}
      selected={supplyForm?.ID}
      isSelected={(record, selectedID) => record.ID === selectedID}
      getFilterContent={(record) => `${record.Name} ${record.SKU || ""}`}
      onRowClick={(record) => {
        // Clone so editing the form doesn't mutate the cached service record.
        // Normalize ProviderSupply so the layer only renders meaningful rows on load.
        const supplyConfig = supplyConfigs.recordsMap.get(record.ID);
        supplyForm = {
          ...record,
          ...copyProductionFields(record),
          Origin: record.Origin || 0,
          MinimunStock: supplyConfig?.MinimunStock || 0,
          ProviderSupply: normalizeProviderSupplyRows(supplyConfig?.ProviderSupply || []),
        };
        layerView = 1;
        ui.openSideLayer(1);
      }}
      mobileCardCss="mb-2"
    />
  </Layer>

  <Layer
    type="side"
    sideLayerSize={680}
    css="px-8 py-8 md:px-16 md:py-10"
    title={supplyForm?.Name || "New Supply or Material|Nuevo Insumo o Material"}
    titleCss="h2 mb-6"
    contentCss="px-0 md:px-0"
    id={1}
    bind:selected={layerView}
    options={[
      [1, "Info|Información"],
      // Only a semi-finished supply has a recipe.
      ...(isSemiFinishedForm ? [[2, "Production|Producción"] as [number, string]] : []),
    ]}
    onClose={() => { supplyForm = {} as ISupplyMaterial; }}
    onSave={() => { onSave(); }}
    onDelete={supplyForm?.ID ? async () => {
      const confirmed = await confirmWarn({
        title: "Eliminar Insumo",
        message: `¿Está seguro que desea eliminar "${supplyForm.Name}"?`,
      });
      if (confirmed) { onSave(true); }
    } : undefined}
  >
    {#if layerView === 2 && isSemiFinishedForm}
      <ProductProductionTab product={supplyForm} excludeSupplyID={supplyForm.ID} />
    {:else}
    <div class="grid grid-cols-24 items-start gap-x-10 gap-y-10 mt-6 md:mt-16" aria-label="Supply Material Form">
      <Input label="Name|Nombre"
        saveOn={supplyForm}
        css="col-span-24 md:col-span-16"
        required={true}
        save="Name"
      />
      <Input label="SKU"
        saveOn={supplyForm}
        css="col-span-24 md:col-span-8"
        save="SKU"
      />
      <Input label="Base Price|Precio Base"
        saveOn={supplyForm}
        css="col-span-12 md:col-span-6"
        save="Price"
        type="number"
        baseDecimals={2}
      />
      <SearchSelect label="Currency|Moneda"
        saveOn={supplyForm}
        css="col-span-12 md:col-span-6"
        save="CurrencyID"
        keyId="i"
        keyName="v"
        options={productoMonedaOptions}
      />
      <Input label="Minimum Stock|Stock Mínimo"
        saveOn={supplyForm}
        css="col-span-12 md:col-span-6"
        save="MinimunStock"
        type="number"
      />
      <SearchSelect label="Brand|Marca"
        saveOn={supplyForm}
        css="col-span-24 md:col-span-6"
        save="BrandID"
        keyId="ID"
        keyName="Name"
        options={listas.ListaRecordsMap.get(2) || []}
      />
      <!-- Sub-unit: how a recipe takes a fraction of the supply ("kg" with 1000 "g"). Once set it can
           only be refined to a multiple, as on a product. -->
      <Input label="Sub-unit|Sub-unidad"
        saveOn={supplyForm}
        css="col-span-12 md:col-span-6"
        save="SbuUnit"
      />
      <Input label="Sub-units per unit|Sub-unidades por unidad"
        saveOn={supplyForm}
        css="col-span-12 md:col-span-6"
        save="SbuQuantity"
        type="number"
      />
      <!-- Semi-finished: made by the company with its own recipe (e.g. stitched uppers), never bought.
           Its stock account is 231 Productos en proceso. -->
      <Checkbox label="Semi-finished: made by the company|Semielaborado: lo fabrica la empresa"
        css="col-span-24"
        checked={isSemiFinishedForm}
        onToggle={(isChecked: boolean) => {
          supplyForm.Origin = isChecked ? SUPPLY_ORIGIN_SEMI_FINISHED : 0;
          if (isChecked) { supplyForm.DepreciationMonths = 0; supplyForm.AccountCode = 0; }
        }}
      />
      {#if !isSemiFinishedForm}
        <SearchSelect label="Depreciation|Depreciación"
          saveOn={supplyForm}
          css="col-span-24 md:col-span-12"
          save="DepreciationMonths"
          keyId="id"
          keyName="name"
          options={depreciationOptions}
        />
        <!-- The default every purchase line of this supply proposes. Left empty, the server
             picks equipment (3369) when it depreciates, else suministros (6032). -->
        <PCGEAccountSelect saveOn={supplyForm} save="AccountCode" css="col-span-24 md:col-span-12"
          kind={PCGE_KIND_SUPPLY} label="Default account|Cuenta contable por defecto" />
      {/if}
      <Input label="Description|Descripción"
        saveOn={supplyForm}
        css="col-span-24"
        save="Description"
      />
      <!-- Lot policy (docs/MANUFACTURING_PLAN.md §10): a supply that tracks lots is received with one. -->
      <Checkbox label="Tracks lots|Controla lotes"
        bind:saveOn={supplyForm}
        save="TracksLots"
        css="col-span-12 md:col-span-6"
      />
      <Input label="Shelf life (days)|Vida útil (días)"
        saveOn={supplyForm}
        css="col-span-12 md:col-span-8"
        save="ShelfLifeDays"
        type="number"
      />
      <Checkbox label="New lots enter in quarantine|Lotes nuevos en cuarentena"
        bind:saveOn={supplyForm}
        save="QualityHoldOnEntry"
        css="col-span-24 md:col-span-10"
      />
    </div>

    <!-- Provider configuration block — mirrors the gestion-compras "Proveedores" list. -->
    <div class="mt-16" aria-label="Supply providers configuration list">
      <div class="mb-8 flex items-center justify-between">
        <div class="h4 ff-bold">Proveedores</div>
        <Button color="green" icon="icon-[fa--plus]"
          label="Adds a new provider row to the supply configuration."
          css="h-32 px-10"
          onClick={addProviderSupplyRow}
        />
      </div>

      <CardsList nonVirtual={true} disableOverflow={true}
        css="w-full max-h-500"
        cardCss="p-14"
        itemsClass="p-4"
        cells={providerSupplyCards}
        data={supplyForm.ProviderSupply || []}
        emptyMessage="No hay proveedores agregados."
        buttonDeleteHandler={(_, providerRowIndex) => {
          removeProviderSupplyRow(providerRowIndex);
        }}
      />
    </div>
    {/if}
  </Layer>
</Page>
