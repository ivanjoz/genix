<script lang="ts">
import Button from '$components/buttons/Button.svelte';
import FilterInput from '$components/form/FilterInput.svelte';
import CheckboxOptions from '$components/form/CheckboxOptions.svelte';
import ImageUploader from '$components/files/ImageUploader.svelte';
import Input from '$components/form/Input.svelte';
import Layer from '$components/layers/Layer.svelte';
import Modal from '$components/layers/Modal.svelte';
import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
import SearchCard from '$components/cards/SearchCard.svelte'; 
import SearchSelect from '$components/form/SearchSelect.svelte';
import VTable from '$components/vTable/VTable.svelte';
import { Core, tr } from '$core/store.svelte';
import T from '$components/misc/T.svelte';
import HTMLEditor from '$domain/HTMLEditor/HTMLEditor.svelte';
import Page from '$domain/Page.svelte';
import { ConfirmWarn, formatN, Loading, Notify } from '$libs/helpers';
import { POST } from '$libs/http.svelte';
import type { ExcelTableColumn } from '$libs/excel/excelBuilder';
import Atributos from './Atributos.svelte';
import CategoriasMarcas from './CategoriasMarcas.svelte';
import { exportProductosToExcel, processProductosImportFile } from './productos.excel';
import { productoMonedaOptions, productoUnidadOptions } from '$core/products-lists';
import { ListasCompartidasService } from '$services/negocio/listas-compartidas.svelte';
import {
    ProductosService,
    type IProduct,
    type IProductoImage
} from "./productos.svelte";
    import TableGrid from '$components/vTable/TableGrid.svelte';

  let filterText = $state("");
  const productos = new ProductosService(true);
  const listas = new ListasCompartidasService([1, 2], true);

  let view = $state(1);
  let layerView = $state(1);
  let productoForm = $state({} as IProduct);
  // svelte-ignore non_reactive_update
  let CategoriasLayer: CategoriasMarcas | null = null;
  // svelte-ignore non_reactive_update
  let MarcasLayer: CategoriasMarcas | null = null;
  let imageUploaderHandler: (() => void) | undefined;
  let importExcelRowsPreview = $state<IProduct[]>([]);
  let importExcelErrors = $state<string[]>([]);
  let isImportExcelProcessing = $state(false);

  const IMPORT_PRODUCTOS_MODAL_ID = 11;

  // Reuse option lists as the single source for labels in UI and Excel export.
  const monedaLabelById = new Map(productoMonedaOptions.map((option) => [option.i, option.v]));
  const unidadLabelById = new Map(productoUnidadOptions.map((option) => [option.i, option.v]));
  const getUpdatedFieldCellCss = (record: IProduct, fieldKey: string): string | undefined => {
    // Highlight imported cells that differ from the current persisted product value.
    if (!record._updatedFields?.includes(fieldKey)) return undefined;
    return "bg-purple-100";
  };

  const makeProductColumns = (isImport = false): ExcelTableColumn<IProduct>[] => [
    {
      header: "ID", 
      field: 'ID', width: "40px",
      css: "c-blue text-center",
      headerCss: "w-48", headerInnerCss: "min-w-42",
      getValue: (e) => e.ID || "",
      render: e => e.ID ? String(e.ID) : `<i class="icon-arrows-cw"></i>`,
      setCellCss: (record) => getUpdatedFieldCellCss(record, "ID"),
      excel: { type: "number"  },
      mobile: { order: 1, css: "col-span-6 ff-bold", icon: "tag" },
    },
    {
      header: "Product|Producto", useLineClamp: true,
      highlight: true, width: "300px",
      getValue: (e) => e.Name,
      field: "Name",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "Name"),
      excel: { type: "string" },
      mobile: {
        order: 2,
        css: "col-span-18",
        render: (e) => `<strong>${e.Name}</strong>`,
      },
    },
    {
      header: "Categories|Categorías", useLineClamp: true,
      highlight: true, width: "160px",
      renderPrefix: e => (e.CategoryIDs||[]).some(x => x <= 0) && `<i class="icon-arrows-cw -ml-4 text-purple-400"></i>`,
      getValue: (e) => {
        // Keep import preview readable by showing original labels from Excel.
        if (isImport && (e._categoriasNames || "").trim().length > 0) {
          return e._categoriasNames || "";
        }
        const nombres = [];
        for (const id of e.CategoryIDs || []) {
          const nombre = listas.get(id)?.Name || `Categoría-${id}`;
          nombres.push(nombre);
        }
        return nombres.join(", ");
      },
      setCellCss: (record) => getUpdatedFieldCellCss(record, "CategoryIDs"),
      // Record the raw category names on import so another step can resolve the IDs.
      excel: { type: "string", importField: "_categoriasNames" },
      mobile: {
        order: 3,
        css: "col-span-24",
        render: (e) => {
          const nombres = [];
          for (const id of e.CategoryIDs || []) {
            const nombre = listas.get(id)?.Name || `Categoría-${id}`;
            nombres.push(nombre);
          }
          return `<div style="font-size: 0.85rem; color: #666;">${nombres.join(", ") || "Sin categorías"}</div>`;
        },
      },
    },
    {
      header: "Price|Precio", width: "100px",
      css: "text-right",
      getValue: (e) => formatN(e.Price / 100, 2),
      field: "Price",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "Price"),
      excel: {
        type: "number",
        format: "#,##0.00",
        exportValue: (e) => e.Price / 100,
        // Converts imported decimal price into backend cents model.
        setValue: (rowDraft, parsedValue) => {
          if (typeof parsedValue !== "number" || !Number.isFinite(parsedValue)) return;
          rowDraft.Price = Math.round(parsedValue * 100);
        },
      },
      mobile: {
        order: 4,
        css: "col-span-8",
        labelLeft: "Precio:",
        render: (e) => formatN(e.Price / 100, 2),
      },
    },
    {
      header: "Discount|Descuento", width: "100px",
      css: "text-right",
      getValue: (e) => (e.Discount ? String(e.Discount) + "%" : ""),
      field: "Discount",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "Discount"),
      excel: { type: "number" },
      mobile: {
        order: 5,
        css: "col-span-8",
        labelLeft: "Desc:",
        render: (e) => (e.Discount ? `${e.Discount}%` : "-"),
      },
    },
    {
      header: "Final Price|Precio Final", width: "100px",
      css: "text-right",
      getValue: (e) => formatN(e.FinalPrice / 100, 2),
      field: "FinalPrice",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "FinalPrice"),
      excel: {
        type: "number",
        format: "#,##0.00",
        exportValue: (e) => e.FinalPrice / 100,
        // Keeps imported final price aligned with backend cents model.
        setValue: (rowDraft, parsedValue) => {
          if (typeof parsedValue !== "number" || !Number.isFinite(parsedValue)) return;
          rowDraft.FinalPrice = Math.round(parsedValue * 100);
        },
      },
      mobile: {
        order: 6,
        css: "col-span-8",
        labelLeft: "Final:",
        icon: "ok",
        render: (e) => `<strong>${formatN(e.FinalPrice / 100, 2)}</strong>`,
      },
    },
    {
      header: "Sub-units|Sub Unidades", width: "120px",
      css: "text-right",
      getValue: (e) => {
        if (!e.SbuUnit) return "";
        return `${e.SbuQuantity} x ${e.SbuUnit}`;
      },
      excel: { type: "string" },
      mobile: {
        order: 7,
        css: "col-span-24",
        labelTop: "Sub-unidades",
        render: (e) => {
          if (!e.SbuUnit) return "<span style='color: #999;'>-</span>";
          return `<div style="font-size: 0.9rem;">${e.SbuQuantity} x ${e.SbuUnit}</div>`;
        },
      },
    },
    {
      header: "Brand|Marca", width: "160px",
      hidden: !isImport,
      renderPrefix: e => !(e.BrandID > 0) && `<i class="-ml-4 icon-arrows-cw text-purple-500"></i>`,
      getValue: (e) => e._marcaNombre || listas.get(e.BrandID)?.Name || "",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "BrandID"),
      excel: { type: "string", importField: "_marcaNombre" },
    },
    {
      header: "Unit|Unidad",
      hidden: !isImport,
      getValue: (e) => e._unidadNombre || unidadLabelById.get(e.UnidadID) || "",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "UnidadID"),
      excel: { type: "string", importField: "_unidadNombre" },
    },
    {
      header: "Volume|Volumen", width: "100px",
      hidden: !isImport,
      getValue: (e) => e.Volumen || "",
      field: "Volumen",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "Volumen"),
      excel: { type: "number", format: "#,##0.00" },
    },
    {
      header: "Weight|Peso",
      hidden: !isImport,
      getValue: (e) => e.Peso || "",
      field: "Peso",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "Peso"),
      excel: { type: "number", format: "#,##0.00" },
    },
    {
      header: "Currency|Moneda",
      hidden: !isImport,
      getValue: (e) => e._monedaNombre || monedaLabelById.get(e.MonedaID) || "",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "MonedaID"),
      excel: { type: "string", importField: "_monedaNombre" },
    },
  ];

  const productoColumns = makeProductColumns(false);
  const productoImportColumns = makeProductColumns(true);

  const categorias = $derived.by(() => {
    return listas.ListaRecordsMap.get(1) || [];
  });

  const exportProductosExcel = async () => {
    Loading.standard(tr("Generating Excel file...|Generando archivo Excel..."));
    try {
      await exportProductosToExcel(productoColumns, productos.records);
      Loading.remove();
      Notify.success(tr("Excel generated successfully.|Excel generado correctamente."));
    } catch (error) {
      console.error("Error exportando productos:", error);
      Loading.remove();
      Notify.failure(`No se pudo exportar el archivo: ${error}`);
    }
  };

  const openImportProductosModal = () => {
    importExcelRowsPreview = [];
    importExcelErrors = [];
    isImportExcelProcessing = false;
    listas.clearTempRecords();
    Core.openModal(IMPORT_PRODUCTOS_MODAL_ID);
  };

  const onImportExcelFileChange = async (file?: File, isRemoved?: boolean) => {
    importExcelRowsPreview = [];
    importExcelErrors = [];
    listas.clearTempRecords();

    if (isRemoved || !file) {
      isImportExcelProcessing = false;
      console.log("[productos-import] file removed or empty selection");
      return;
    }

    isImportExcelProcessing = true;
    console.log("[productos-import] starting excel import:", file.name);

    try {
      const importResult = await processProductosImportFile(productoImportColumns, file, listas, productos);

      if(!importResult.rows){
     		Notify.warning(tr("No new records detected to save.|No se detectaron registros nuevos a guardar."))
      }
      
      importExcelRowsPreview = importResult.rows;
      importExcelErrors = importResult.errors;

      console.log("[productos-import] import completed:", {
        rows: importResult.rows.length,
        mappedColumns: importResult.mappedColumns,
        ignoredHeaders: importResult.ignoredHeaders,
        errors: importExcelErrors.length,
        pendingSharedListRecords: listas.getTempRecords().length,
      });
    } catch (error) {
      console.error("[productos-import] import failed:", error);
      importExcelErrors = [`Error procesando el archivo: ${error}`];
      Notify.failure(`No se pudo procesar el Excel: ${error}`);
    } finally {
      isImportExcelProcessing = false;
    }
  };

  const saveImportProductos = async () => {
    if (importExcelErrors.length > 0) {
      Notify.failure(tr("Fix import errors before saving.|Corrige los errores de importación antes de guardar."));
      return;
    }
    if (importExcelRowsPreview.length === 0) {
      Notify.failure(tr("No valid rows to import.|No hay filas válidas para importar."));
      return;
    }

    try {
      Loading.standard(tr("Saving product import...|Guardando importación de productos..."));
      const pendingSharedListRecords = listas.getTempRecords().length;
      let tempIDToNewID = new Map<number, number>();

      // Persist temporary categorías/marcas and get TempID -> ID mappings.
      if (pendingSharedListRecords > 0) {
        Loading.change(`Creando categorías/marcas nuevas (${pendingSharedListRecords})...`);
        tempIDToNewID = await listas.syncTempRecords();
      }

      const productosToSave = importExcelRowsPreview.map((importedProducto) => {
        importedProducto.BrandID = tempIDToNewID.get(importedProducto.BrandID) || importedProducto.BrandID;
        importedProducto.CategoryIDs = (importedProducto.CategoryIDs || []).map((categoriaID) => {
          return tempIDToNewID.get(categoriaID) || categoriaID;
        });

        const existingProducto = productos.recordsMap.get(importedProducto.ID);
        if (existingProducto) {
          for (const fieldKey in existingProducto) {
            const typedFieldKey = fieldKey as keyof IProduct;
            if (importEditableProductKeys.has(typedFieldKey)) { continue; }
            (importedProducto as any)[typedFieldKey] = existingProducto[typedFieldKey];
          }
        }

        return importedProducto;
      });

      const unresolvedIDProducto = productosToSave.find((productoDraft) => {
        const hasInvalidCategoriaID = (productoDraft.CategoryIDs || []).some((categoriaID) => categoriaID <= 0);
        return hasInvalidCategoriaID || productoDraft.BrandID <= 0;
      });
      if (unresolvedIDProducto) {
        throw new Error(
          `No se pudieron resolver IDs temporales para el producto "${unresolvedIDProducto.Name || unresolvedIDProducto.ID}".`,
        );
      }

      console.log("[productos-import] final payload before productos POST:", {
        productos: productosToSave.length,
        tempIDMappings: Array.from(tempIDToNewID.entries()),
      });

      Loading.change(`Guardando productos (${productosToSave.length})...`);
      await doPostProductos(productosToSave);
      for(const e of productos.records){ delete e._updatedFields }
      Loading.remove();
      
      Notify.success(tr("Import completed successfully.|Importación completada correctamente."));
      importExcelRowsPreview = [];
      Core.closeModal(IMPORT_PRODUCTOS_MODAL_ID);
      productoForm = {} as IProduct
    } catch (error) {
      console.error("[productos-import] save import failed:", error);
      Notify.failure(`No se pudo completar la importación: ${error}`);
      Loading.remove();
    }
  };

  const doPostProductos = async (payload: IProduct[]): Promise<Map<number, number>> => {
    for (const producto of payload) {
      // Keep local shape consistent even when backend omits optional fields in responses.
      producto.Image = producto.Images?.[0];
      producto.CategoryIDs = producto.CategoryIDs || [];
    }
    return await productos.postAndSync(payload);
  };

  const importEditableProductKeys = new Set<keyof IProduct>([
    "ID", "Name", "CategoryIDs", "Price", "Discount", "FinalPrice",
    "BrandID", "UnidadID", "Volumen", "Peso", "MonedaID",
  ]);

  const onSave = async (isDelete?: boolean) => {
    if ((productoForm.Name?.length || 0) < 4) {
      Notify.failure(tr("Name must be at least 4 characters.|El nombre debe tener al menos 4 caracteres."));
      return;
    }
    if(imageUploaderHandler){
      Loading.standard(tr("Saving image...|Guardando imagen..."));
    	await imageUploaderHandler()
     	productoForm._imageSource = undefined
      Loading.change(tr("Saving product...|Guardando producto..."))
    }

    console.log("productor a enviar:", $state.snapshot(productoForm));    
    if (isDelete) {  productoForm.ss = 0; }

    Loading.standard(tr("Saving product...|Guardando producto..."));
    try {
    	await doPostProductos([productoForm]);
    } catch (error) {
      Notify.failure(error as string);
      Loading.remove();
      return;
    }
    Loading.remove();
    productoForm = {} as IProduct
    Core.openSideLayer(0);
  };

  $effect(() => {
    console.log("productor form:", $state.snapshot(productoForm));
  });

  const deleteProductoImage = async (ImageToDelete: string) => {
    Loading.standard(tr("Deleting image...|Eliminando Imagen..."));
    try {
      await POST({
        data: { ProductID: productoForm.ID, ImageToDelete },
        route: "producto-image",
        refreshRoutes: ["productos"],
      });
    } catch (error) {
      Notify.failure(`Error al eliminar imagen: ${error}`);
      Loading.remove();
      return;
    }
    Loading.remove();
    productoForm.Images = (productoForm.Images || []).filter(
      (e) => e.n !== ImageToDelete,
    );
  };
</script>

<Page title="Products|Productos">
  <div class="grid grid-cols-12 md:flex md:flex-row items-center mb-8">
    <OptionsStrip
      selected={view}
      css="col-span-12 mb-6 md:mb-0"
      options={[
        [1, "Products|Productos"],
        [2, "Categories|Categorías"],
        [3, "Brands|Marcas"],
      ]}
      useMobileGrid={true}
      onSelect={(e) => {
        Core.openSideLayer(0);
        productoForm = { ID: 0 } as IProduct;
        view = e[0] as number;
      }}
    />
    <FilterInput label="Filter products|Filtrar productos"
      css="w-full md:w-200 md:ml-12 col-span-5"
      icon="icon-search"
      bind:value={filterText}
    />

    {#if view === 1}
      <Button label="Opens the import modal to load products from an Excel file."
        color="blue"
        icon="icon-upload"
        css="ml-auto mr-8 col-span-3"
        onClick={openImportProductosModal}
      />
      <Button label="Exports the current product list to an Excel file for download."
        color="purple"
        icon="icon-download"
        css="mr-8 col-span-3"
        onClick={exportProductosExcel}
      />
    {/if}

    <Button name="New|Nuevo" label="Shows the form to create a new product in a side layer."
      color="green"
      icon="icon-plus"
      hideNameOnMobile
      css={`col-span-7 ${view === 1 ? "" : "ml-auto"}`}
      onClick={() => {
        if (view === 2) {
          CategoriasLayer?.newRecord();
        } else if (view === 3) {
          MarcasLayer?.newRecord();
        } else {
       		productoForm = { ss: 1 } as IProduct
          Core.openSideLayer(1);
        }
      }}
    />
  </div>

  {#if view === 1}
    <Layer type="content">
      <VTable
        columns={productoColumns}
        data={productos.records}
        {filterText}
        selected={productoForm?.ID}
        isSelected={(e, id) => e.ID === id}
        getFilterContent={(e) => {
          return e.Name;
        }}
        onRowClick={(e) => {
          productoForm = { ...e };
          productoForm.CategoryIDs = [...(e.CategoryIDs || [])];
          productoForm.Properties = [...(e.Properties || [])];
          Core.openSideLayer(1);
        }}
        mobileCardCss="mb-2"
      />
    </Layer>
  {/if}
  <Layer
    type="side"
    sideLayerSize={780}
    css="px-8 py-8 md:px-16 md:py-10"
    title={productoForm?.Name || ""}
    titleCss="h2 mb-6"
    contentCss="px-0 md:px-0"
    id={1}
    bind:selected={layerView}
    onClose={() => {
      productoForm = {} as IProduct;
    }}
    onSave={() => {
      onSave();
    }}
    onDelete={() => {
      ConfirmWarn(
        "Eliminar Producto",
        `¿Está seguro que desea eliminar "${productoForm.Name}"?`,
        "SI",
        "NO",
        () => {
          onSave(true);
        },
      );
    }}
    options={[
      [1, "Info|Información", ["Info", ""]],
      [2, "Sheet|Ficha"],
      [3, "Presentations|Presentaciones", ["Presenta-", "tions"]],
      [4, "Photos|Fotos"],
    ]}
  >
    {#if layerView === 1}
      <div
        class="grid grid-cols-24 items-start gap-x-10 gap-y-10 mt-6 md:mt-16"
        aria-label="Product Form"
      >
        <Input
          label="Name|Nombre"
          saveOn={productoForm}
          css="col-span-24"
          required={true}
          save="Name"
        />
        <div class="col-span-24 md:col-span-9 md:row-span-4">
          <ImageUploader
            saveAPI="producto-image"
            useConvertAvif={true}
            clearOnUpload={true}
            types={["avif", "webp"]}
            folder="img-productos"
            size={2}
            src={productoForm.Image?.n}
            cardCss="w-full h-180 p-4"
            imageSource={productoForm._imageSource}
            setDataToSend={(e) => {
              e.ProductID = productoForm.ID;
            }}
            onChange={(e, uploadHandler) => {
              imageUploaderHandler = uploadHandler;
              productoForm._imageSource = e;
              productoForm.Image = undefined
            }}
            onUploaded={(imagePath, description) => {
              if (imagePath.includes("/")) {
                imagePath = imagePath.split("/")[1];
              }
              productoForm._imageSource = undefined;
              productoForm.Image = {
                n: imagePath,
                d: description,
              } as IProductoImage;
              productoForm.Images = productoForm.Images || [];
              productoForm.Images.unshift(productoForm.Image);
            }}
          />
        </div>
        <Input
          label="Base Price|Precio Base"
          saveOn={productoForm}
          css="col-span-12 md:col-span-5"
          save="Price"
          type="number"
          baseDecimals={2}
          dependencyValue={productoForm.Price}
          onChange={() => {
            console.log("productoForm", $state.snapshot(productoForm));
            productoForm.FinalPrice = Math.round(
              (productoForm.Price * (100 - (productoForm.Discount || 0))) /
                100,
            );
          }}
        />
        <Input
          label="Discount|Descuento"
          saveOn={productoForm}
          css="col-span-12 md:col-span-5"
          save="Discount"
          postValue="%"
          type="number"
          validator={(v) => {
            return !v || (v as number) < 100;
          }}
        />
        <Input
          label="Final Price|Precio Final"
          saveOn={productoForm}
          css="col-span-12 md:col-span-5"
          dependencyValue={productoForm.FinalPrice}
          save="FinalPrice"
          type="number"
          baseDecimals={2}
          onChange={() => {
            console.log("productoForm", $state.snapshot(productoForm));
            productoForm.Price = Math.round(
              (productoForm.FinalPrice /
                (100 - (productoForm.Discount || 0))) *
                100,
            );
          }}
        />
        <SearchSelect
          label="Currency|Moneda"
          saveOn={productoForm}
          css="col-span-12 md:col-span-5"
          save="MonedaID"
          keyId="i"
          keyName="v"
          options={productoMonedaOptions}
        />
        <Input
          label="Weight|Peso"
          saveOn={productoForm}
          css="col-span-12 md:col-span-5"
          save="Peso"
          type="number"
        />
        <SearchSelect
          label="Unit|Unidad"
          saveOn={productoForm}
          css="col-span-12 md:col-span-5"
          save="UnidadID"
          keyId="i"
          keyName="v"
          options={productoUnidadOptions}
        />
        <Input
          label="Volume|Volumen"
          saveOn={productoForm}
          css="col-span-12 md:col-span-5"
          save="Volumen"
          type="number"
        />
        <SearchSelect
          label="Brand|Marca"
          saveOn={productoForm}
          css="col-span-12 md:col-span-10 mb-2"
          save="BrandID"
          keyId="ID"
          keyName="Name"
          options={listas.ListaRecordsMap.get(2) || []}
        />
        <div class="col-span-24 md:col-span-14">
          <CheckboxOptions
            save="Params"
            saveOn={productoForm}
            type="multiple"
            options={[{ i: 1, v: "SKU Individual" }]}
            keyId="i"
            keyName="v"
          />
        </div>
        <Input
          saveOn={productoForm}
          save="SKU"
          css="col-span-12 md:col-span-10"
          label="SKU"
        />
        <Input
          saveOn={productoForm}
          save="Description"
          css="col-span-24 mb-4"
          label="Short Description|Descripción Corta"
        />
        <SearchCard
          css="col-span-24 flex items-start"
          label="CATEGORIES|CATEGORÍAS ::"
          options={categorias}
          keyId="ID"
          keyName="Name"
          cardCss="grow"
          inputCss="w-[35%] md:w-180"
          bind:saveOn={productoForm}
          save="CategoryIDs"
          optionsCss="w-280"
        />
        <div class="ff-bold h3 col-span-24 ml-8">Sub-Unidades</div>
        <Input
          saveOn={productoForm}
          save="SbuUnit"
          css="col-span-12 md:col-span-6"
          label="Name|Nombre"
        />
        <Input
          bind:saveOn={productoForm}
          save="SbuPrice"
          baseDecimals={2}
          css="col-span-12 md:col-span-6"
          label="Base Price|Precio Base"
          type="number"
          onChange={() => {
            console.log();
            productoForm.FinalPrice = Math.round(
              (productoForm.Price * (100 - (productoForm.Discount || 0))) /
                100,
            );
            productoForm = { ...productoForm };
          }}
        />
        <Input
          saveOn={productoForm}
          save="SbuDiscount"
          postValue="%"
          css="col-span-12 md:col-span-6"
          label="Discount|Descuento"
          type="number"
        />
        <Input
          saveOn={productoForm}
          save="SbuFinalPrice"
          baseDecimals={2}
          css="col-span-12 md:col-span-6"
          label="Final Price|Precio Final"
          type="number"
        />
        <Input
          saveOn={productoForm}
          save="SbuQuantity"
          css="col-span-12 md:col-span-6"
          label="Quantity|Cantidad"
          type="number"
        />
      </div>
    {/if}
    {#if layerView === 2}
      <HTMLEditor saveOn={productoForm} save="ContentHTML" css="mt-12" />
    {/if}
    {#if layerView === 3}
      <div class="mt-16 w-full" aria-label="Product attributes and presentations editor">
        <Atributos bind:producto={productoForm}></Atributos>
      </div>
    {/if}
    {#if layerView === 4}
      <div
        class="grid grid-cols-2 md:grid-cols-4 items-start gap-x-10 gap-y-10 mt-16"
      >
        <ImageUploader
          saveAPI="producto-image"
          useConvertAvif={true}
          clearOnUpload={true}
          types={["avif", "webp"]}
          folder="img-productos"
          cardCss="w-full h-170 p-4"
          setDataToSend={(e) => {
            e.ProductID = productoForm.ID;
          }}
          onUploaded={(imagePath, description) => {
            if (imagePath.includes("/")) {
              imagePath = imagePath.split("/")[1];
            }
            productoForm.Image = {
              n: imagePath,
              d: description,
            } as IProductoImage;
            productoForm.Images = productoForm.Images || [];
            productoForm.Images.unshift(productoForm.Image);
          }}
        />
        {#each productoForm.Images || [] as image}
          <ImageUploader
            saveAPI="producto-image"
            size={2}
            clearOnUpload={true}
            types={["avif", "webp"]}
            folder="img-productos"
            cardCss="w-full h-170 p-4"
            src={image?.n}
            useConvertAvif={true}
            onDelete={() => {
              ConfirmWarn(
                "ELIMINAR IMAGEN",
                `Eliminar la imagen ${image.d ? `"${image.d}"` : "seleccionada"}`,
                "SI",
                "NO",
                () => {
                  deleteProductoImage(image.n);
                },
              );
            }}
          />
        {/each}
      </div>
    {/if}
  </Layer>
  {#if view === 2}
    <CategoriasMarcas {listas} {filterText} origin={1} bind:this={CategoriasLayer} />
  {/if}
  {#if view === 3}
    <CategoriasMarcas {listas} {filterText} origin={2} bind:this={MarcasLayer} />
  {/if}

  <Modal id={IMPORT_PRODUCTOS_MODAL_ID}
    title="Import Products from Excel|Importar Productos desde Excel"
    size={9} css="px-4"
    useFileImportWithErrors={true}
    fileErrors={importExcelErrors}
    onFileChange={onImportExcelFileChange}
    onSave={importExcelRowsPreview.length > 0 ? saveImportProductos : undefined}
    saveButtonLabel="Importar"
    saveIcon="icon-upload"
  >
    <div class="">
      <div class="border border-slate-200 rounded-md overflow-hidden h-[58vh] min-h-[260px]">
        <!-- VTable virtualizes against its own scroll container, so it needs a concrete height -->
        <TableGrid columns={productoImportColumns} css="h-full" height="100%"
          data={importExcelRowsPreview} rowHeight={46}
          emptyMessage={isImportExcelProcessing
            ? "Procesando archivo Excel..."
            : "Selecciona un archivo para visualizar las filas procesadas."}
        />
      </div>
    </div>
  </Modal>
</Page>
