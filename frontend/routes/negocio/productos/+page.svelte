<script lang="ts">
import CheckboxOptions from '$components/CheckboxOptions.svelte';
import ImageUploader from '$components/ImageUploader.svelte';
import Input from '$components/Input.svelte';
import Layer from '$components/Layer.svelte';
import Modal from '$components/Modal.svelte';
import OptionsStrip from '$components/OptionsStrip.svelte';
import SearchCard from '$components/SearchCard.svelte'; 
import SearchSelect from '$components/SearchSelect.svelte';
import VTable from '$components/vTable/VTable.svelte';
import { Core } from '$core/store.svelte';
import HTMLEditor from '$domain/HTMLEditor/HTMLEditor.svelte';
import Page from '$domain/Page.svelte';
import { ConfirmWarn, formatN, Loading, Notify, throttle } from '$libs/helpers';
import { POST } from '$libs/http.svelte';
import type { ExcelTableColumn } from '$libs/excel/excelBuilder';
import Atributos from './Atributos.svelte';
import CategoriasMarcas from './CategoriasMarcas.svelte';
import { exportProductosToExcel, processProductosImportFile } from './productos.excel';
import { productoMonedaOptions, productoUnidadOptions } from '$core/products-lists';
import { ListasCompartidasService } from '$services/negocio/listas-compartidas.svelte';
import {
    ProductosService,
    type IProducto,
    type IProductoImage
} from "./productos.svelte";

  let filterText = $state("");
  const productos = new ProductosService(true);
  const listas = new ListasCompartidasService([1, 2], true);

  let view = $state(1);
  let layerView = $state(1);
  let productoForm = $state({} as IProducto);
  // svelte-ignore non_reactive_update
  let CategoriasLayer: CategoriasMarcas | null = null;
  // svelte-ignore non_reactive_update
  let MarcasLayer: CategoriasMarcas | null = null;
  let imageUploaderHandler: (() => void) | undefined;
  let importExcelRowsPreview = $state<IProducto[]>([]);
  let importExcelErrors = $state<string[]>([]);
  let isImportExcelProcessing = $state(false);

  const IMPORT_PRODUCTOS_MODAL_ID = 11;

  // Reuse option lists as the single source for labels in UI and Excel export.
  const monedaLabelById = new Map(productoMonedaOptions.map((option) => [option.i, option.v]));
  const unidadLabelById = new Map(productoUnidadOptions.map((option) => [option.i, option.v]));
  const getUpdatedFieldCellCss = (record: IProducto, fieldKey: string): string | undefined => {
    // Highlight imported cells that differ from the current persisted product value.
    if (!record._updatedFields?.includes(fieldKey)) return undefined;
    return "bg-purple-100";
  };

  const makeProductColumns = (isImport = false): ExcelTableColumn<IProducto>[] => [
    {
      header: "ID", 
      field: 'ID',
      css: "c-blue text-center",
      headerCss: "w-48", headerInnerCss: "min-w-42",
      getValue: (e) => e.ID || "",
      render: e => e.ID ? String(e.ID) : `<i class="icon-arrows-cw"></i>`,
      setCellCss: (record) => getUpdatedFieldCellCss(record, "ID"),
      excel: { type: "number"  },
      mobile: { order: 1, css: "col-span-6 ff-bold", icon: "tag" },
    },
    {
      header: "Producto",
      highlight: true,
      getValue: (e) => e.Nombre,
      field: "Nombre",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "Nombre"),
      excel: { type: "string" },
      mobile: {
        order: 2,
        css: "col-span-18",
        render: (e) => `<strong>${e.Nombre}</strong>`,
      },
    },
    {
      header: "Categorías",
      highlight: true,
      renderPrefix: e => (e.CategoriasIDs||[]).some(x => x <= 0) && `<i class="icon-arrows-cw text-purple-400"></i>`,
      getValue: (e) => {
        // Keep import preview readable by showing original labels from Excel.
        if (isImport && (e._categoriasNames || "").trim().length > 0) {
          return e._categoriasNames || "";
        }
        const nombres = [];
        for (const id of e.CategoriasIDs || []) {
          const nombre = listas.get(id)?.Nombre || `Categoría-${id}`;
          nombres.push(nombre);
        }
        return nombres.join(", ");
      },
      setCellCss: (record) => getUpdatedFieldCellCss(record, "CategoriasIDs"),
      // Record the raw category names on import so another step can resolve the IDs.
      excel: { type: "string", importField: "_categoriasNames" },
      mobile: {
        order: 3,
        css: "col-span-24",
        render: (e) => {
          const nombres = [];
          for (const id of e.CategoriasIDs || []) {
            const nombre = listas.get(id)?.Nombre || `Categoría-${id}`;
            nombres.push(nombre);
          }
          return `<div style="font-size: 0.85rem; color: #666;">${nombres.join(", ") || "Sin categorías"}</div>`;
        },
      },
    },
    {
      header: "Precio",
      css: "text-right",
      getValue: (e) => formatN(e.Precio / 100, 2),
      field: "Precio",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "Precio"),
      excel: {
        type: "number",
        format: "#,##0.00",
        exportValue: (e) => e.Precio / 100,
        // Converts imported decimal price into backend cents model.
        setValue: (rowDraft, parsedValue) => {
          if (typeof parsedValue !== "number" || !Number.isFinite(parsedValue)) return;
          rowDraft.Precio = Math.round(parsedValue * 100);
        },
      },
      mobile: {
        order: 4,
        css: "col-span-8",
        labelLeft: "Precio:",
        render: (e) => formatN(e.Precio / 100, 2),
      },
    },
    {
      header: "Descuento",
      css: "text-right",
      getValue: (e) => (e.Descuento ? String(e.Descuento) + "%" : ""),
      field: "Descuento",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "Descuento"),
      excel: { type: "number" },
      mobile: {
        order: 5,
        css: "col-span-8",
        labelLeft: "Desc:",
        render: (e) => (e.Descuento ? `${e.Descuento}%` : "-"),
      },
    },
    {
      header: "Precio Final",
      css: "text-right",
      getValue: (e) => formatN(e.PrecioFinal / 100, 2),
      field: "PrecioFinal",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "PrecioFinal"),
      excel: {
        type: "number",
        format: "#,##0.00",
        exportValue: (e) => e.PrecioFinal / 100,
        // Keeps imported final price aligned with backend cents model.
        setValue: (rowDraft, parsedValue) => {
          if (typeof parsedValue !== "number" || !Number.isFinite(parsedValue)) return;
          rowDraft.PrecioFinal = Math.round(parsedValue * 100);
        },
      },
      mobile: {
        order: 6,
        css: "col-span-8",
        labelLeft: "Final:",
        icon: "ok",
        render: (e) => `<strong>${formatN(e.PrecioFinal / 100, 2)}</strong>`,
      },
    },
    {
      header: "Sub Unidades",
      css: "text-right",
      getValue: (e) => {
        if (!e.SbnUnidad) return "";
        return `${e.SbnCantidad} x ${e.SbnUnidad}`;
      },
      excel: { type: "string" },
      mobile: {
        order: 7,
        css: "col-span-24",
        labelTop: "Sub-unidades",
        render: (e) => {
          if (!e.SbnUnidad) return "<span style='color: #999;'>-</span>";
          return `<div style="font-size: 0.9rem;">${e.SbnCantidad} x ${e.SbnUnidad}</div>`;
        },
      },
    },
    {
      header: "Marca",
      hidden: !isImport,
      renderPrefix: e => !(e.MarcaID > 0) && `<i class="icon-arrows-cw text-purple-500"></i>`,
      getValue: (e) => e._marcaNombre || listas.get(e.MarcaID)?.Nombre || "",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "MarcaID"),
      excel: { type: "string", importField: "_marcaNombre" },
    },
    {
      header: "Unidad",
      hidden: !isImport,
      getValue: (e) => e._unidadNombre || unidadLabelById.get(e.UnidadID) || "",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "UnidadID"),
      excel: { type: "string", importField: "_unidadNombre" },
    },
    {
      header: "Volumen",
      hidden: !isImport,
      getValue: (e) => e.Volumen || "",
      field: "Volumen",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "Volumen"),
      excel: { type: "number", format: "#,##0.00" },
    },
    {
      header: "Peso",
      hidden: !isImport,
      getValue: (e) => e.Peso || "",
      field: "Peso",
      setCellCss: (record) => getUpdatedFieldCellCss(record, "Peso"),
      excel: { type: "number", format: "#,##0.00" },
    },
    {
      header: "Moneda",
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
    Loading.standard("Generando archivo Excel...");
    try {
      await exportProductosToExcel(productoColumns, productos.records);
      Loading.remove();
      Notify.success("Excel generado correctamente.");
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
      Notify.failure("Corrige los errores de importación antes de guardar.");
      return;
    }
    if (importExcelRowsPreview.length === 0) {
      Notify.failure("No hay filas válidas para importar.");
      return;
    }

    try {
      Loading.standard("Guardando importación de productos...");
      const pendingSharedListRecords = listas.getTempRecords().length;
      let tempIDToNewID = new Map<number, number>();

      // Persist temporary categorías/marcas and get TempID -> ID mappings.
      if (pendingSharedListRecords > 0) {
        Loading.change(`Creando categorías/marcas nuevas (${pendingSharedListRecords})...`);
        tempIDToNewID = await listas.syncTempRecords();
      }

      const productosToSave = importExcelRowsPreview.map((importedProducto) => {
        importedProducto.MarcaID = tempIDToNewID.get(importedProducto.MarcaID) || importedProducto.MarcaID;
        importedProducto.CategoriasIDs = (importedProducto.CategoriasIDs || []).map((categoriaID) => {
          return tempIDToNewID.get(categoriaID) || categoriaID;
        });

        const existingProducto = productos.recordsMap.get(importedProducto.ID);
        if (existingProducto) {
          for (const fieldKey in existingProducto) {
            const typedFieldKey = fieldKey as keyof IProducto;
            if (importEditableProductKeys.has(typedFieldKey)) { continue; }
            (importedProducto as any)[typedFieldKey] = existingProducto[typedFieldKey];
          }
        }

        return importedProducto;
      });

      const unresolvedIDProducto = productosToSave.find((productoDraft) => {
        const hasInvalidCategoriaID = (productoDraft.CategoriasIDs || []).some((categoriaID) => categoriaID <= 0);
        return hasInvalidCategoriaID || productoDraft.MarcaID <= 0;
      });
      if (unresolvedIDProducto) {
        throw new Error(
          `No se pudieron resolver IDs temporales para el producto "${unresolvedIDProducto.Nombre || unresolvedIDProducto.ID}".`,
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
      
      Notify.success("Importación completada correctamente.");
      importExcelRowsPreview = [];
      Core.closeModal(IMPORT_PRODUCTOS_MODAL_ID);
      productoForm = {} as IProducto
    } catch (error) {
      console.error("[productos-import] save import failed:", error);
      Notify.failure(`No se pudo completar la importación: ${error}`);
      Loading.remove();
    }
  };

  const doPostProductos = async (payload: IProducto[]): Promise<Map<number, number>> => {
    for (const producto of payload) {
      // Keep local shape consistent even when backend omits optional fields in responses.
      producto.Image = producto.Images?.[0];
      producto.CategoriasIDs = producto.CategoriasIDs || [];
    }
    return await productos.postAndSync(payload);
  };

  const importEditableProductKeys = new Set<keyof IProducto>([
    "ID", "Nombre", "CategoriasIDs", "Precio", "Descuento", "PrecioFinal",
    "MarcaID", "UnidadID", "Volumen", "Peso", "MonedaID",
  ]);

  const onSave = async (isDelete?: boolean) => {
    if ((productoForm.Nombre?.length || 0) < 4) {
      Notify.failure("El nombre debe tener al menos 4 caracteres.");
      return;
    }
    if(imageUploaderHandler){
      Loading.standard("Guardando imagen...");
    	await imageUploaderHandler()
     	productoForm._imageSource = undefined
      Loading.change("Guardando producto...")
    }

    console.log("productor a enviar:", $state.snapshot(productoForm));    
    if (isDelete) {  productoForm.ss = 0; }

    Loading.standard("Guardando producto...");
    try {
    	await doPostProductos([productoForm]);
    } catch (error) {
      Notify.failure(error as string);
      Loading.remove();
      return;
    }
    Loading.remove();
    productoForm = {} as IProducto
    Core.openSideLayer(0);
  };

  $effect(() => {
    console.log("productor form:", $state.snapshot(productoForm));
  });

  const deleteProductoImage = async (ImageToDelete: string) => {
    Loading.standard("Eliminando Imagen...");
    try {
      await POST({
        data: { ProductoID: productoForm.ID, ImageToDelete },
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

<Page title="Productos">
  <div class="grid grid-cols-12 md:flex md:flex-row items-center mb-8">
    <OptionsStrip
      selected={view}
      css="col-span-12 mb-6 md:mb-0"
      options={[
        [1, "Productos"],
        [2, "Categorías"],
        [3, "Marcas"],
      ]}
      useMobileGrid={true}
      onSelect={(e) => {
        Core.openSideLayer(0);
        productoForm = { ID: 0 } as IProducto;
        view = e[0] as number;
      }}
    />
    <div class="i-search w-full md:w-200 md:ml-12 col-span-5">
      <div><i class="icon-search"></i></div>
      <input
        type="text"
        onkeyup={(ev) => {
          const value = String((ev.target as any).value || "");
          throttle(() => {
            filterText = value;
          }, 150);
        }}
      />
    </div>

    {#if view === 1}
      <button aria-label="Importar"
        class="bx-blue ml-auto mr-8 col-span-3"
        onclick={(ev) => {
          ev.stopPropagation();
          openImportProductosModal();
        }}
      >
        <i class="icon-upload"></i>
      </button>
      <button  aria-label="Descargar"
        class="bx-purple mr-8 col-span-3"
        onclick={(ev) => {
          ev.stopPropagation();
          exportProductosExcel();
        }}
      >
        <i class="icon-download"></i>
      </button>
    {/if}

    <button
      class={`bx-green col-span-7 ${view === 1 ? "" : "ml-auto"}`}
      onclick={(ev) => {
        ev.stopPropagation();
        if (view === 2) {
          CategoriasLayer?.newRecord();
        } else if (view === 3) {
          MarcasLayer?.newRecord();
        } else {
       		productoForm = { ss: 1 } as IProducto
          Core.openSideLayer(1);
        }
      }}
    >
      <i class="icon-plus"></i><span class="hidden md:block">Nuevo</span>
    </button>
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
          return e.Nombre;
        }}
        onRowClick={(e) => {
          productoForm = { ...e };
          productoForm.CategoriasIDs = [...(e.CategoriasIDs || [])];
          productoForm.Propiedades = [...(e.Propiedades || [])];
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
    title={productoForm?.Nombre || ""}
    titleCss="h2 mb-6"
    contentCss="px-0 md:px-0"
    id={1}
    bind:selected={layerView}
    onClose={() => {
      productoForm = {} as IProducto;
    }}
    onSave={() => {
      onSave();
    }}
    onDelete={() => {
      ConfirmWarn(
        "Eliminar Producto",
        `¿Está seguro que desea eliminar "${productoForm.Nombre}"?`,
        "SI",
        "NO",
        () => {
          onSave(true);
        },
      );
    }}
    options={[
      [1, "Información", ["Informa-", "ción"]],
      [2, "Ficha"],
      [3, "Presentaciones", ["Presenta-", "ciones"]],
      [4, "Fotos"],
    ]}
  >
    {#if layerView === 1}
      <div
        class="grid grid-cols-24 items-start gap-x-10 gap-y-10 mt-6 md:mt-16"
      >
        <Input
          label="Nombre"
          saveOn={productoForm}
          css="col-span-24"
          required={true}
          save="Nombre"
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
              e.ProductoID = productoForm.ID;
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
          label="Precio Base"
          saveOn={productoForm}
          css="col-span-12 md:col-span-5"
          save="Precio"
          type="number"
          baseDecimals={2}
          dependencyValue={productoForm.Precio}
          onChange={() => {
            console.log("productoForm", $state.snapshot(productoForm));
            productoForm.PrecioFinal = Math.round(
              (productoForm.Precio * (100 - (productoForm.Descuento || 0))) /
                100,
            );
          }}
        />
        <Input
          label="Descuento"
          saveOn={productoForm}
          css="col-span-12 md:col-span-5"
          save="Descuento"
          postValue="%"
          type="number"
          validator={(v) => {
            return !v || (v as number) < 100;
          }}
        />
        <Input
          label="Precio Final"
          saveOn={productoForm}
          css="col-span-12 md:col-span-5"
          dependencyValue={productoForm.PrecioFinal}
          save="PrecioFinal"
          type="number"
          baseDecimals={2}
          onChange={() => {
            console.log("productoForm", $state.snapshot(productoForm));
            productoForm.Precio = Math.round(
              (productoForm.PrecioFinal /
                (100 - (productoForm.Descuento || 0))) *
                100,
            );
          }}
        />
        <SearchSelect
          label="Moneda"
          saveOn={productoForm}
          css="col-span-12 md:col-span-5"
          save="MonedaID"
          keyId="i"
          keyName="v"
          options={productoMonedaOptions}
        />
        <Input
          label="Peso"
          saveOn={productoForm}
          css="col-span-12 md:col-span-5"
          save="Peso"
          type="number"
        />
        <SearchSelect
          label="Unidad"
          saveOn={productoForm}
          css="col-span-12 md:col-span-5"
          save="UnidadID"
          keyId="i"
          keyName="v"
          options={productoUnidadOptions}
        />
        <Input
          label="Volumen"
          saveOn={productoForm}
          css="col-span-12 md:col-span-5"
          save="Volumen"
          type="number"
        />
        <SearchSelect
          label="Marca"
          saveOn={productoForm}
          css="col-span-12 md:col-span-10 mb-2"
          save="MarcaID"
          keyId="ID"
          keyName="Nombre"
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
          save="Descripcion"
          css="col-span-24 mb-4"
          label="Descripción Corta"
        />
        <SearchCard
          css="col-span-24 flex items-start"
          label="CATEGORÍAS ::"
          options={categorias}
          keyId="ID"
          keyName="Nombre"
          cardCss="grow"
          inputCss="w-[35%] md:w-180"
          bind:saveOn={productoForm}
          save="CategoriasIDs"
          optionsCss="w-280"
        />
        <div class="ff-bold h3 col-span-24 ml-8">Sub-Unidades</div>
        <Input
          saveOn={productoForm}
          save="SbnUnidad"
          css="col-span-12 md:col-span-6"
          label="Nombre"
        />
        <Input
          bind:saveOn={productoForm}
          save="SbnPrecio"
          baseDecimals={2}
          css="col-span-12 md:col-span-6"
          label="Precio Base"
          type="number"
          onChange={() => {
            console.log();
            productoForm.PrecioFinal = Math.round(
              (productoForm.Precio * (100 - (productoForm.Descuento || 0))) /
                100,
            );
            productoForm = { ...productoForm };
          }}
        />
        <Input
          saveOn={productoForm}
          save="SbnDescuento"
          postValue="%"
          css="col-span-12 md:col-span-6"
          label="Descuento"
          type="number"
        />
        <Input
          saveOn={productoForm}
          save="SbnPreciFinal"
          baseDecimals={2}
          css="col-span-12 md:col-span-6"
          label="Precio Final"
          type="number"
        />
        <Input
          saveOn={productoForm}
          save="SbnCantidad"
          css="col-span-12 md:col-span-6"
          label="Cantidad"
          type="number"
        />
      </div>
    {/if}
    {#if layerView === 2}
      <HTMLEditor saveOn={productoForm} save="ContentHTML" css="mt-12" />
    {/if}
    {#if layerView === 3}
      <div class="mt-16 w-full">
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
            e.ProductoID = productoForm.ID;
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
    title="Importar Productos desde Excel"
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
        <VTable
          columns={productoImportColumns}
          data={importExcelRowsPreview}
          maxHeight="100%"
          emptyMessage={isImportExcelProcessing
            ? "Procesando archivo Excel..."
            : "Selecciona un archivo para visualizar las filas procesadas."}
        />
      </div>
    </div>
  </Modal>
</Page>
