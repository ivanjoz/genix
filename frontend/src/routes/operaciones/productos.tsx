import { Loading, Notify } from "notiflix";
import { max } from "simple-statistics";
import { Show, createSignal } from "solid-js";
import { BarOptions } from "~/components/Cards";
import { CellEditable, CellTextOptions } from "~/components/Editables";
import { Input, refreshInput } from "~/components/Input";
import { SideLayer, openLayers, setOpenLayers } from "~/components/Modals";
import { QTable } from "~/components/QTable";
import { SearchSelect } from "~/components/SearchSelect";
import { throttle } from "~/core/main";
import { pageView } from "~/core/menu";
import { PageContainer } from "~/core/page";
import { IProducto, IProductoPropiedad, postProducto, useProductosAPI } from "~/services/operaciones/productos";
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes";
import { formatN } from "~/shared/main";

export default function Productos() {

  const [almacenes, setAlmacenes] = useSedesAlmacenesAPI()
  const [productos, setProductos] = useProductosAPI()

  const [filterText, setFilterText] = createSignal("")
  const [productoForm, setProductoForm] = createSignal({} as IProducto)
  const [layerView, setLayerView] = createSignal(1)
  const layerWidth = 0.48

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
    setOpenLayers([])
    Loading.remove()
  }
  
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
        css="selectable" tableCss="w-page"
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
            getValue: e => e.Precio ? formatN(e.Precio,2) : "", 
          },
          { header: "Descuento", cardColumn: [3,2],  css: "t-c",
            getValue: e => e.Descuento ? formatN(e.Descuento,1) + "%" : "",
          },
          { header: "Precio Final", cardColumn: [3,2],  css: "t-c",
            getValue: e => e.PrecioFinal ? formatN(e.PrecioFinal,2) : "",
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
            <div class="w-08x flex ai-center mb-10 wc-60-40 z20">
              <Input saveOn={productoForm()} save="Peso" 
                label="Peso" type="number"
              />
              <SearchSelect saveOn={productoForm()} save="PesoT" placeholder=" "
                label="-" keys="i.v" options={[
                  {i:1, v:"Kg"},{i:2, v:"g"},{i:3, v:"Libras"}
                ]}
              />
            </div>
            <div class="w-08x flex ai-center mb-10 wc-60-40 z20">
              <Input saveOn={productoForm()} save="Volumen" 
                label="Volumen" type="number"
              />
              <SearchSelect saveOn={productoForm()} save="VolumenT" placeholder=" "
                label="-" keys="i.v" options={[
                  {i:1, v:"Kg"},{i:2, v:"g"},{i:3, v:"Libras"}
                ]}
              />
            </div>
            <SearchSelect saveOn={productoForm()} save="Moneda" placeholder=" "
              css="w-08x mb-10" 
              label="Moneda" keys="i.v" options={[
                {i:1, v:"PEN (S/.)"},{i:2, v:"g"},{i:3, v:"Libras"}
              ]}
            />
            <Input saveOn={productoForm()} save="Precio" id={1}
              css="w-05x mb-10" label="Precio Base" type="number"
              onChange={() => {
                const form = productoForm()
                form.PrecioFinal = form.Precio * (1-(form.Descuento||100)/100)
                refreshInput([3])
              }}
            />
            <Input saveOn={productoForm()} save="Descuento" 
              css="w-05x mb-10" label="Descuento" type="number"
              postValue={<div class="p-abs pos-v c-steel1">%</div>}
              onChange={() => {
                const form = productoForm()
                form.PrecioFinal = form.Precio * (1-(form.Descuento||100)/100)
                refreshInput([3])
              }}
            />
            <Input saveOn={productoForm()} save="PrecioFinal" id={3}
              css="w-06x mb-10" label="Precio Final" type="number"
              onChange={() => {
                const form = productoForm()
                form.Precio = form.PrecioFinal / (1-(form.Descuento||100)/100)
                refreshInput([1])
              }}
            />
            <div class="w-24x">Sub-Unidades</div>
            <Input saveOn={productoForm()} save="SbnCantidad" 
              css="w-04x mb-10" label="Cantidad" type="number"
            />
            <Input saveOn={productoForm()} save="SbnUnidad" 
              css="w-06x mb-10" label="Nombre"
            />
            <Input saveOn={productoForm()} save="SbnPrecio" 
              css="w-05x mb-10" label="Precio Base" type="number"
            />
            <Input saveOn={productoForm()} save="SbnDescuento" 
              postValue={<div class="p-abs pos-v c-steel1">%</div>}
              css="w-04x mb-10" label="Descuento" type="number"
            />
            <Input saveOn={productoForm()} save="SbnPreciFinal" 
              css="w-05x mb-10" label="Precio Final" type="number"
            />
          </div>
          <div class="w100 py-04 px-04">
            <QTable data={productoForm().Propiedades||[]}
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
                    return <CellTextOptions saveOn={e} save="Options" />
                  }
                },
                { header: <div>
                    <button class="bn1 s2 b-green" onclick={ev => {
                      ev.stopPropagation()
                      const propiedades = [...productoForm().Propiedades]
                      const id = propiedades.length > 0 ? max(propiedades.map(x => x.ID)) : 0
                      propiedades.push({ ID: id + 1, Options: [] } as IProductoPropiedad)
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
      </SideLayer>
    </Show>
  </PageContainer>
}