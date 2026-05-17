<script lang="ts">
import Button from '$components/buttons/Button.svelte';
import FilterInput from '$components/form/FilterInput.svelte';
import Input from '$components/form/Input.svelte';
import Layer from '$components/layers/Layer.svelte';
import SearchSelect from '$components/form/SearchSelect.svelte';
import VTable from '$components/vTable/VTable.svelte';
import { Core } from '$core/store.svelte';
import Page from '$domain/Page.svelte';
import { ConfirmWarn, formatN, Loading, Notify } from '$libs/helpers';
import type { ExcelTableColumn } from '$libs/excel/excelBuilder';
import { productoMonedaOptions } from '$core/products-lists';
import { ListasCompartidasService } from '$services/negocio/listas-compartidas.svelte';
import { SupplyMaterialService, type ISupplyMaterial } from './supply-material.svelte';

  // Insumos catalog (master data) + Marcas list for the BrandID picker.
  const supplies = new SupplyMaterialService(true);
  // ListaID=2 corresponds to "Marcas" (same convention used by the productos page).
  const listas = new ListasCompartidasService([2], true);

  let filterText = $state("");
  let supplyForm = $state({} as ISupplyMaterial);

  // Lookup brand name by ID for the table cell — reuses the same shared list as productos.
  const brandLabelById = $derived.by(() => {
    const records = listas.ListaRecordsMap.get(2) || [];
    return new Map(records.map((brandRecord) => [brandRecord.ID, brandRecord.Nombre]));
  });

  const supplyColumns: ExcelTableColumn<ISupplyMaterial>[] = [
    {
      header: "ID",
      field: "ID",
      css: "c-blue text-center",
      headerCss: "w-40", headerInnerCss: "min-w-36",
      getValue: (record) => record.ID || "",
      render: (record) => record.ID ? String(record.ID) : `<i class="icon-arrows-cw"></i>`,
      mobile: { order: 1, css: "col-span-6 ff-bold", icon: "tag" },
    },
    {
      header: "Insumo",
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
      header: "Marca",
      getValue: (record) => brandLabelById.get(record.BrandID) || "",
      mobile: { order: 4, css: "col-span-12", labelLeft: "Marca:" },
    },
    {
      header: "Precio",
      css: "text-right",
      // Backend stores Price in cents; UI shows two-decimal currency.
      getValue: (record) => formatN(record.Price / 100, 2),
      field: "Price",
      mobile: { order: 5, css: "col-span-12", labelLeft: "Precio:", render: (record) => formatN(record.Price / 100, 2) },
    },
  ];

  const doPostSupplies = async (payload: ISupplyMaterial[]) => {
    return await supplies.postAndSync(payload);
  };

  const onSave = async (isDelete?: boolean) => {
    if ((supplyForm.Name?.length || 0) < 2) {
      Notify.failure("El nombre debe tener al menos 2 caracteres.");
      return;
    }
    // Soft-delete: backend evicts records with ss=0 from the active list on next sync.
    if (isDelete) { supplyForm.ss = 0; }

    Loading.standard("Guardando insumo...");
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

<Page title="Insumos">
  <div class="grid grid-cols-12 md:flex md:flex-row items-center mb-8">
    <FilterInput label="Filtrar insumos"
      css="w-full md:w-200 col-span-9"
      icon="icon-search"
      bind:value={filterText}
    />
    <Button name="Nuevo" label="Shows the form to create a new supply in a side layer."
      color="green"
      icon="icon-plus"
      hideNameOnMobile
      css="col-span-3 ml-auto"
      onClick={() => {
        supplyForm = { ss: 1 } as ISupplyMaterial;
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
        supplyForm = { ...record };
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
      <Input label="Nombre"
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
      <Input label="Precio Base"
        saveOn={supplyForm}
        css="col-span-12 md:col-span-6"
        save="Price"
        type="number"
        baseDecimals={2}
      />
      <SearchSelect label="Moneda"
        saveOn={supplyForm}
        css="col-span-12 md:col-span-6"
        save="CurrencyID"
        keyId="i"
        keyName="v"
        options={productoMonedaOptions}
      />
      <SearchSelect label="Marca"
        saveOn={supplyForm}
        css="col-span-24 md:col-span-12"
        save="BrandID"
        keyId="ID"
        keyName="Nombre"
        options={listas.ListaRecordsMap.get(2) || []}
      />
      <Input label="Descripción"
        saveOn={supplyForm}
        css="col-span-24"
        save="Description"
      />
    </div>
  </Layer>
</Page>
