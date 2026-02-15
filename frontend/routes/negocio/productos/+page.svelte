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
import { exportProductosToExcel } from './productos.excel';
import { productoMonedaOptions, productoUnidadOptions } from './lists/productos-lists';
import {
    ListasCompartidasService,
    postProducto,
    ProductosService,
    type IProducto,
    type IProductoImage
} from "./productos.svelte";

  type ProductoExcelColumn = ExcelTableColumn<IProducto>;

  let filterText = $state("");
  const productos = new ProductosService();
  const listas = new ListasCompartidasService([1, 2]);

  let view = $state(1);
  let layerView = $state(1);
  let productoForm = $state({} as IProducto);
  // svelte-ignore non_reactive_update
  let CategoriasLayer: CategoriasMarcas | null = null;
  // svelte-ignore non_reactive_update
  let MarcasLayer: CategoriasMarcas | null = null;
  let imageUploaderHandler: (() => void) | undefined;
  let importExcelInputElement: HTMLInputElement | null = null;
  let selectedImportExcelFile = $state<File | null>(null);

  const IMPORT_PRODUCTOS_MODAL_ID = 11;

  // Reuse option lists as the single source for labels in UI and Excel export.
  const monedaLabelById = new Map(productoMonedaOptions.map((option) => [option.i, option.v]));
  const unidadLabelById = new Map(productoUnidadOptions.map((option) => [option.i, option.v]));

  const makeProductColumns = (isImport = false): ProductoExcelColumn[] => [
    {
      header: "ID",
      css: "c-blue text-center",
      headerCss: "w-48",
      getValue: (e) => e.ID,
      excel: { type: "number" },
      mobile: { order: 1, css: "col-span-6 ff-bold", icon: "tag" },
    },
    {
      header: "Producto",
      highlight: true,
      getValue: (e) => e.Nombre,
      field: "Nombre",
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
      getValue: (e) => {
        const nombres = [];
        for (const id of e.CategoriasIDs || []) {
          const nombre = listas.get(id)?.Nombre || `Categoría-${id}`;
          nombres.push(nombre);
        }
        return nombres.join(", ");
      },
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
      excel: {
        type: "number",
        format: "#,##0.00",
        exportValue: (e) => e.Precio / 100,
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
      excel: {
        type: "number",
        format: "#,##0.00",
        exportValue: (e) => e.PrecioFinal / 100,
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
      getValue: (e) => listas.get(e.MarcaID)?.Nombre || "",
      excel: { type: "string" },
    },
    {
      header: "Unidad",
      hidden: !isImport,
      getValue: (e) => unidadLabelById.get(e.UnidadID) || "",
      excel: { type: "string" },
    },
    {
      header: "Volumen",
      hidden: !isImport,
      getValue: (e) => e.Volumen || "",
      field: "Volumen",
      excel: { type: "number", format: "#,##0.00" },
    },
    {
      header: "Peso",
      hidden: !isImport,
      getValue: (e) => e.Peso || "",
      field: "Peso",
      excel: { type: "number", format: "#,##0.00" },
    },
    {
      header: "Moneda",
      hidden: !isImport,
      getValue: (e) => monedaLabelById.get(e.MonedaID) || "",
      excel: { type: "string" },
    },
  ];

  const productoColumns = makeProductColumns(false);
  const productoImportColumns = makeProductColumns(true);

  const importColumnHeaders = $derived.by(() => {
    return productoImportColumns.map((column) => {
      return typeof column.header === "function" ? column.header() : String(column.header || "");
    }).filter((header) => header.length > 0);
  });

  const categorias = $derived.by(() => {
    return listas.ListaRecordsMap.get(1) || [];
  });

  const selectedImportExcelFileName = $derived.by(() => {
    return selectedImportExcelFile?.name || "Ningún archivo seleccionado";
  });

  const exportProductosExcel = async () => {
    Loading.standard("Generando archivo Excel...");
    try {
      await exportProductosToExcel(productoColumns, productos.productos);
      Loading.remove();
      Notify.success("Excel generado correctamente.");
    } catch (error) {
      console.error("Error exportando productos:", error);
      Loading.remove();
      Notify.failure(`No se pudo exportar el archivo: ${error}`);
    }
  };

  const openImportProductosModal = () => {
    selectedImportExcelFile = null;
    Core.openModal(IMPORT_PRODUCTOS_MODAL_ID);
  };

  const selectImportExcelFile = () => {
    importExcelInputElement?.click();
  };

  const onImportExcelFileSelected = (event: Event) => {
    const input = event.currentTarget as HTMLInputElement;
    selectedImportExcelFile = input.files?.[0] || null;
  };

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
    
    if (isDelete) {
      productoForm.ss = 0;
    }

    Loading.standard("Guardando producto...");
    try {
      var productosUpdated = await postProducto([productoForm]);
    } catch (error) {
      Notify.failure(error as string);
      Loading.remove();
      return;
    }
    Loading.remove();

    const producto = productos.productos.find((x) => x.ID === productoForm.ID);
    if (producto && isDelete) {
      productos.productos = productos.productos.filter(
        (x) => x.ID !== productoForm.ID,
      );
    } else if (producto) {
      Object.assign(producto, productoForm);
    } else {
      productoForm.ID = productosUpdated[0].ID;
      productos.productos.unshift(productoForm);
      productos.productos = [...productos.productos];
    }
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

<Page sideLayerSize={780} title="Productos">
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
      <button
        class="bx-blue ml-auto mr-8 col-span-3"
        onclick={(ev) => {
          ev.stopPropagation();
          openImportProductosModal();
        }}
      >
        <i class="icon-upload"></i><span class="hidden md:block">Importar</span>
      </button>
      <button
        class="bx-purple mr-8 col-span-3"
        onclick={(ev) => {
          ev.stopPropagation();
          exportProductosExcel();
        }}
      >
        <i class="icon-download"></i><span class="hidden md:block">Exportar</span>
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
        data={productos.productos}
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
    <CategoriasMarcas {listas} origin={1} bind:this={CategoriasLayer} />
  {/if}
  {#if view === 3}
    <CategoriasMarcas {listas} origin={2} bind:this={MarcasLayer} />
  {/if}

  <Modal id={IMPORT_PRODUCTOS_MODAL_ID} title="Importar Productos desde Excel" size={6}>
    <div class="px-6 py-6 md:px-10 md:py-8">
      <div class="mb-8 text-slate-700">
        <strong>Columnas esperadas para importación</strong>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-12 max-h-260 overflow-y-auto pr-4">
        {#each importColumnHeaders as columnHeader}
          <div class="px-4 py-3 rounded border border-slate-200 bg-slate-50">
            {columnHeader}
          </div>
        {/each}
      </div>

      <!-- Hidden file input keeps the modal button UX simple and consistent. -->
      <input
        bind:this={importExcelInputElement}
        class="hidden"
        type="file"
        accept=".xlsx,.xls"
        onchange={onImportExcelFileSelected}
      />

      <div class="flex flex-col md:flex-row md:items-center gap-5">
        <button class="bx-blue" onclick={selectImportExcelFile}>
          <i class="icon-upload"></i>
          <span>Seleccionar Excel</span>
        </button>
        <div class="text-slate-700 break-all">{selectedImportExcelFileName}</div>
      </div>
    </div>
  </Modal>
</Page>
