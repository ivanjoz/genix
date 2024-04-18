import { ConfirmWarn, Loading, Notify } from "~/core/main";
import { max } from "simple-statistics";
import { Show, createEffect, createMemo, createSignal } from "solid-js";
import { BarOptions, CardSelect } from "~/components/Cards";
import { CellEditable, CellTextOptions } from "~/components/Editables";
import { CheckBoxContainer, Input, refreshInput } from "~/components/Input";
import { SideLayer, openLayers, setOpenLayers } from "~/components/Modals";
import { QTable } from "~/components/QTable";
import { SearchSelect } from "~/components/SearchSelect";
import { throttle } from "~/core/main";
import { pageView } from "~/core/menu";
import { PageContainer } from "~/core/page";
import { IProducto, IProductoImage, IProductoPropiedad, IProductoPropiedades, postProducto, useProductosAPI } from "~/services/operaciones/productos";
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes";
import { formatN } from "~/shared/main";
import { ImageUploader } from "~/components/Uploaders";
import { POST } from "~/shared/http";
import { useListasCompartidasAPI } from "~/services/admin/listas-compartidas";
import { ListasCompartidasLayer } from "~/routes-components/admin/listas-compartidas";

export default function Productos() {

  const [almacenes] = useSedesAlmacenesAPI()
  const [listasCompartidas] = useListasCompartidasAPI([1])
  const [productos, setProductos] = useProductosAPI()

  const [filterText, setFilterText] = createSignal("")
  const [productoForm, setProductoForm] = createSignal({} as IProducto)
  const [layerView, setLayerView] = createSignal(1)
  const layerWidth = 0.48

  createEffect(() => {
    console.log("listas compartidas::",listasCompartidas())
  })

  const categorias = createMemo(() => {
    return listasCompartidas()?.Records?.filter(x => x.ListaID === 1) || []
  })
  
  const saveProducto = async (isDelete?: boolean) => {
    const form = productoForm()
    if((form.Nombre?.length||0) < 4){
      Notify.failure("El nombre debe tener al menos 4 caracteres.")
      return
    }

    Loading.standard("Creando /Actualizando Producto...")
    try {
      var result = await postProducto([form])
    } catch (error) {
      Notify.failure(error as string); Loading.remove(); return
    }

    let productos_ = [...productos().productos]

    if(form.ID){
      const selected = productos().productos.find(x => x.ID === form.ID)
      if(selected){ Object.assign(selected, form) }
      if(isDelete){ productos_ = productos_.filter(x => x.ID !== form.ID) }
    } else {
      form.ID = result[0].ID
      productos_.unshift(form)
    }
    
    setProductos({...productos(), productos: productos_})
    setProductoForm({} as IProducto)
    setOpenLayers([])
    Loading.remove()
  }
  
  const deleteProductoImage = async (ImageToDelete: string) => {
    try {
      await POST({
        data: { ProductoID: productoForm().ID, ImageToDelete },
        route: "producto-image",
        refreshIndexDBCache: "productos",
      }) 
    } catch (error) {
      Notify.failure(`Error al guardar el producto: ${error}`)
      return
    }    
    productoForm().Images = productoForm().Images.filter(e => e.n !== ImageToDelete)
    setProductoForm({...productoForm()})
  }

  let counter = -1
  
  return <PageContainer title="Productos"
    views={[[1,"Productos"],[2,"Parámetros"]]}
  >
    <Show when={pageView() === 1}>
      <div class="flex ai-center jc-between mb-06">
        <div class="search-c4 mr-16 w16rem">
          <div><i class="icon-search"></i></div>
          <input class="w100" autocomplete="off" type="text" onKeyUp={ev => {
            ev.stopPropagation()
            throttle(() => {
              setFilterText(((ev.target as any).value||"").toLowerCase().trim())
            },150)
          }}/>
        </div>
        <div class="flex ai-center">
          <button class="bn1 b-green" onClick={ev => {
            ev.stopPropagation()
            setOpenLayers([1])
            setProductoForm({ ss: 1, Propiedades: [] } as IProducto)
          }}>
            <i class="icon-plus"></i>
          </button>
        </div>
      </div>
      <QTable data={productos().productos || []} 
        css="selectable" tableCss="w-page-t"
        isSelected={(e,id) => e.ID === id as number}
        selected={productoForm().ID}
        filterText={filterText()} filterKeys={["Nombre"]}
        style={{ width: openLayers().length > 0 
          ? `calc(var(--page-width) * ${1-layerWidth})` : undefined  
        }}
        maxHeight="calc(80vh - 13rem)" 
        columns={[
          { header: "ID", headerStyle: { width: '3rem' }, css: "t-c c-purple",
            getValue: e => e.ID
          },
          { header: "Nombre", cardColumn: [3,2], field: "Nombre",
            getValue: e => e.Nombre, cardCss: "h5 c-steel",
          },
          { header: "Precio", cardColumn: [3,2], cardCss: "h5 c-steel", css: "t-c",
            getValue: e => e.Precio ? formatN(e.Precio/100,2) : "", 
          },
          { header: "Descuento", cardColumn: [3,2],  css: "t-c",
            getValue: e => e.Descuento ? formatN(e.Descuento,1) + "%" : "",
          },
          { header: "Precio Final", cardColumn: [3,2],  css: "t-c",
            getValue: e => e.PrecioFinal ? formatN(e.PrecioFinal/100,2) : "",
          },
          { header: "Sub-Unidades", cardColumn: [3,2],
            getValue: e => {
              if(!e.SbnUnidad) return ""
              return `${e.SbnCantidad} x ${e.SbnUnidad}`
            }
          },
          { header: "Grupos", cardColumn: [3,2],  css: "t-c",
            getValue: e => ""
          },
        ]}
        onRowCLick={e => {
          if(e.ID === productoForm().ID){
            setProductoForm({} as IProducto) 
            setOpenLayers([]) 
          } else {
            setProductoForm({...e})
            setOpenLayers([1]) 
          }
        }}
      />
      <SideLayer id={1} style={{ width: `calc(var(--page-width) * ${layerWidth})` }}
        title={"Producto " + (productoForm()?.Nombre||"(Nuevo)") }
        onClose={() => {
          setProductoForm({} as IProducto) 
        }}
        onSave={() => {
          saveProducto()
        }}
      >
        <div></div>
        <BarOptions selectedID={layerView()} class="w100"
          options={[[1,'Información'],[2,'Ficha'],[3,'Fotos']]}
          buttonStyle={{ "min-height": '2.1rem' }} buttonClass="ff-bold"
          onSelect={id => {
            setLayerView(id)
          }}
        />
        <Show when={layerView() === 1}>
          <div class="flex-wrap w100 mt-12">
            <Input saveOn={productoForm()} save="Nombre" required={true}
              css="w-24x mb-10" label="Nombre" 
            />
            <div class="flex-wrap ac-baseline w100 p-rel" 
                style={{ "min-height": 'calc(14vh + 1rem)' }}>
              <div class="w-05x p-rel">
                <ImageUploader  cardStyle={{ width: '100%', position: 'absolute', 
                  top: '0', "z-index": 12 }}
                  types={["avif","webp"]}
                  src={productoForm().Image
                    ? `img-productos/${productoForm().Image.n}-x2` : ""}
                />
              </div>
              <Input saveOn={productoForm()} save="Peso" css="w-055x mb-10"
                label="Peso" type="number"
              />
              <SearchSelect saveOn={productoForm()} save="PesoT" placeholder=" "
                label="-" keys="i.v" css="w-04x mb-10"
                options={[
                  {i:1, v:"Kg"},{i:2, v:"g"},{i:3, v:"Libras"}
                ]}
              />
              <Input saveOn={productoForm()} save="Volumen" css="w-055x mb-10"
                label="Volumen" type="number"
              />
              <SearchSelect saveOn={productoForm()} save="VolumenT" placeholder=" "
                label="-" keys="i.v" css="w-04x mb-10"
                options={[
                  {i:1, v:"Kg"},{i:2, v:"g"},{i:3, v:"Libras"}
                ]}
              />
              <div class="w-05x p-rel"></div>
              <Input saveOn={productoForm()} save="Precio" id={1}
                css="w-055x mb-10" label="Precio Base" type="number"
                baseDecimals={2}
                onChange={() => {
                  const form = productoForm()
                  form.PrecioFinal = Math.floor(form.Precio * (1-(form.Descuento||0)/100))
                  refreshInput([3])
                }}
              />
              <Input saveOn={productoForm()} save="Descuento" 
                css="w-04x mb-10" label="Desc." type="number"
                postValue={<div class="p-abs pos-v c-steel1">%</div>}
                onChange={() => {
                  const form = productoForm()
                  form.PrecioFinal = Math.floor(form.Precio * (1-(form.Descuento||0)/100))
                  refreshInput([3])
                  console.log("form::", form)
                }}
              />
              <Input saveOn={productoForm()} save="PrecioFinal" id={3}
                css="w-055x mb-10" label="Precio Final" type="number"
                baseDecimals={2}
                onChange={() => {
                  const form = productoForm()
                  form.Precio = Math.floor(form.PrecioFinal / (1-(form.Descuento||0)/100))
                  refreshInput([1])
                }}
              />
              <SearchSelect saveOn={productoForm()} save="Moneda" placeholder=" "
                css="w-04x mb-10" 
                label="Moneda" keys="i.v" options={[
                  {i:1, v:"PEN (S/.)"},{i:2, v:"g"},{i:3, v:"Libras"}
                ]}
              />
             
            </div>
            <div class="w-24x mb-02 flex jc-end">
              <CheckBoxContainer options={[ { v: 1, n: 'SKU Individual' } ]} 
                keys="v.n" saveOn={productoForm()} save="Params" />
            </div>
            <div class="w100 mb-10">
              <CardSelect label="Categorías" options={categorias()} keys="ID.Nombre"
                css="w-145x" saveOn={productoForm()} save="CategoriasIDs" />
            </div>
            <div class="ff-bold h3 mb-06">
              <div class="ml-08">Sub-Unidades</div>
            </div>
            <div class="flex w100">
              <Input saveOn={productoForm()} save="SbnUnidad" 
                css="w-05x mb-10" label="Nombre"
              />
              <Input saveOn={productoForm()} save="SbnPrecio" baseDecimals={2}
                css="w-055x mb-10" label="Precio Base" type="number"
              />
              <Input saveOn={productoForm()} save="SbnDescuento" 
                postValue={<div class="p-abs pos-v c-steel1">%</div>}
                css="w-04x mb-10" label="Descuento" type="number"
              />
              <Input saveOn={productoForm()} save="SbnPreciFinal" baseDecimals={2}
                css="w-055x mb-10" label="Precio Final" type="number"
              />
              <Input saveOn={productoForm()} save="SbnCantidad" 
                css="w-04x mb-10" label="Cantidad" type="number"
              />
            </div>
          </div>
          <div class="w100 py-04 px-04">
            <QTable data={(productoForm().Propiedades||[]).filter(x => x.Status)}
              maxHeight="40rem" tableCss="single-color"
              columns={[
                { header: "Propiedad", headerStyle: { width: '8rem' },
                  render: e => {
                    return  <CellEditable save="Nombre" saveOn={e} 
                      contentClass="flex ai-center nowrap" required={true} />
                  }
                },
                { header: "Opciones", cardColumn: [3,2],
                  render: e => {
                    return <CellTextOptions saveOn={e} save="Options" 
                      getValue={(e: IProductoPropiedad) => e.nm}
                      skipt={e => !e.ss}
                      setValue={(e,content) => {
                        e.nm = content
                        e.ss = 1
                        if(!e.id){ e.id = counter; counter-- }
                        return e
                      }}
                    />
                  }
                },
                { header: <div>
                    <button class="bn1 s2 b-green" onclick={ev => {
                      ev.stopPropagation()
                      const propiedades = [...productoForm().Propiedades]
                      const id = propiedades.length > 0 ? max(propiedades.map(x => x.ID)) : 0
                      propiedades.push({ ID: id + 1, Status: 1, Options: [] } as IProductoPropiedades)
                      productoForm().Propiedades = propiedades
                      setProductoForm({...productoForm()})
                    }}>
                      <i class="icon-plus"></i>
                    </button>
                  </div>, 
                  headerStyle: { width: '2.6rem' }, css: "t-c",
                  cardColumn: [1,2],
                  render: e => {
                    return <button class="bnr2 b-red b-card-1" onClick={ev => {
                      ev.stopPropagation()
                      const propiedades = productoForm().Propiedades.filter(x => x !== e)
                      productoForm().Propiedades = propiedades
                      setProductoForm({...productoForm()})
                    }}>
                      <i class="icon-trash"></i>
                    </button>
                  }
                }
              ]}    
            />
          </div>
        </Show>
        <Show when={layerView() === 3}>
          <div class="w100 px-02 py-08"
            style={{ display: 'grid', 
              'grid-template-columns': '1fr 1fr 1fr 1fr', 'grid-gap': '8px' }}
          >
            <ImageUploader saveAPI="producto-image"
              refreshIndexDBCache="productos"
              clearOnUpload={true}
              cardStyle={{ width: '100%' }}
              setDataToSend={e => {
                e.ProductoID = productoForm().ID
              }}
              onUploaded={src => {
                const name = src.split("/")[1]
                const images = productoForm().Images || []
                images.push({ n: name } as IProductoImage)
                productoForm().Images = images
                setProductoForm({...productoForm()})
              }}
            />
            { (productoForm().Images||[]).map(e => {
                return <ImageUploader src={"img-productos/"+ e.n + "-x2"}
                  cardStyle={{ width: '100%' }}
                  types={["avif","webp"]} 
                  onDelete={() => {
                    ConfirmWarn("ELIMINAR IMAGEN",
                      `Eliminar la imagen ${e.d ? `"${e.d}"` : "seleccionada"}`,
                      "SI","NO",
                      () => {
                        deleteProductoImage(e.n)
                      }
                    )
                  }}  
                />
              })
            }
          </div>
        </Show>
      </SideLayer>
    </Show>
    <Show when={pageView() === 2}>
      <div class="flex ai-center jc-between mb-06">
        <ListasCompartidasLayer listaID={1} listas={listasCompartidas()} />
      </div>
    </Show>
  </PageContainer>
}

