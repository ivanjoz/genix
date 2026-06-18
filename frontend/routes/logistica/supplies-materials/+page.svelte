<script lang="ts">
import Button from '$components/buttons/Button.svelte';
import FilterInput from '$components/form/FilterInput.svelte';
import Input from '$components/form/Input.svelte';
import SearchSelect from '$components/form/SearchSelect.svelte';
import Layer from '$components/layers/Layer.svelte';
import CardsList from '$components/vTable/CardsList.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ICardCell } from '$components/vTable/types';
import { productoMonedaOptions } from '$core/products-lists';
import { Core, tr } from '$core/store.svelte';
import T from '$components/misc/T.svelte';
import Page from '$domain/Page.svelte';
import type { ExcelTableColumn } from '$libs/excel/excelBuilder';
import { ConfirmWarn, formatN, Loading, Notify } from '$libs/helpers';
import { ListasCompartidasService } from '$services/negocio/listas-compartidas.svelte';
import { ClientProviderService, ClientProviderType } from '../../negocio/clientes/clientes-proveedores.svelte';
import {
  createEmptyProviderSupplyRow,
  normalizeProviderSupplyRows,
  type IProductSupplyProviderRow,
} from '../gestion-compras/supply-management.svelte';
import { SupplyMaterialService, type ISupplyMaterial } from './supply-material.svelte';

  // Insumos catalog (master data) + Marcas list for the BrandID picker.
  const supplies = new SupplyMaterialService(true);
  // ListaID=2 corresponds to "Marcas" (same convention used by the productos page).
  const listas = new ListasCompartidasService([2], true);
  // Reuse the shared provider catalog so the picker mirrors the gestion-compras flow.
  const providers = new ClientProviderService(ClientProviderType.PROVIDER, true);

  let filterText = $state("");
  let supplyForm = $state({} as ISupplyMaterial);

  // Lookup brand name by ID for the table cell — reuses the same shared list as productos.
  const brandLabelById = $derived.by(() => {
    const records = listas.ListaRecordsMap.get(2) || [];
    return new Map(records.map((brandRecord) => [brandRecord.ID, brandRecord.Name]));
  });

  const supplyColumns: ExcelTableColumn<ISupplyMaterial>[] = [
    {
      header: "ID",
      field: "ID",
      css: "c-blue text-center",
      headerCss: "w-40", headerInnerCss: "min-w-36",
      getValue: (record) => record.ID || "",
      render: (record) => record.ID ? String(record.ID) : `<i class="icon-[fa--refresh]"></i>`,
      mobile: { order: 1, css: "col-span-6 ff-bold", icon: "tag" },
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
      Notify.failure(tr("Name must be at least 2 characters.|El nombre debe tener al menos 2 caracteres."));
      return;
    }
    // Soft-delete: backend evicts records with ss=0 from the active list on next sync.
    if (isDelete) { supplyForm.ss = 0; }
    // Strip empty rows so the backend only persists meaningful provider entries.
    supplyForm.ProviderSupply = normalizeProviderSupplyRows(supplyForm.ProviderSupply || []);

    Loading.standard(tr("Saving supply...|Guardando insumo..."));
    try {
      await doPostSupplies([supplyForm]);
    } catch (error) {
      Notify.failure(error as string);
      Loading.remove();
      return;
    }
    Loading.remove();
    supplyForm = {} as ISupplyMaterial;
    Core.openSideLayer(0);
  };
</script>

<Page title="Supplies & Materials|Insumos">
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
        supplyForm = { ss: 1, ProviderSupply: [] } as ISupplyMaterial;
        Core.openSideLayer(1);
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
        supplyForm = {
          ...record,
          ProviderSupply: normalizeProviderSupplyRows(record.ProviderSupply || []),
        };
        Core.openSideLayer(1);
      }}
      mobileCardCss="mb-2"
    />
  </Layer>

  <Layer
    type="side"
    sideLayerSize={680}
    css="px-8 py-8 md:px-16 md:py-10"
    title={supplyForm?.Name || "Nuevo Insumo"}
    titleCss="h2 mb-6"
    contentCss="px-0 md:px-0"
    id={1}
    onClose={() => { supplyForm = {} as ISupplyMaterial; }}
    onSave={() => { onSave(); }}
    onDelete={supplyForm?.ID ? () => {
      ConfirmWarn(
        "Eliminar Insumo",
        `¿Está seguro que desea eliminar "${supplyForm.Name}"?`,
        "SI", "NO",
        () => { onSave(true); },
      );
    } : undefined}
  >
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
      <Input label="Description|Descripción"
        saveOn={supplyForm}
        css="col-span-24"
        save="Description"
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
  </Layer>
</Page>
