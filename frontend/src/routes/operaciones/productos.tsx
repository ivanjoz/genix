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
import { IProducto, useProductosAPI } from "~/services/operaciones/productos";
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes";

export default function Productos() {

  const [almacenes, setAlmacenes] = useSedesAlmacenesAPI()
  const [productos, setProductos] = useProductosAPI()

  const [filterText, setFilterText] = createSignal("")
  const [productoForm, setProductoForm] = createSignal({} as IProducto)
  const [layerView, setLayerView] = createSignal(1)
  const layerWidth = 0.48
  
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
            setProductoForm({ ss: 1, propiedades: [] } as IProducto)
          }}>
            <i class="icon-plus"></i>
          </button>
        </div>
      </div>
      <QTable data={productos().productos || []} tableCss="w-page"
        style={{ width: openLayers().length > 0 
          ? `calc(var(--page-width) * ${1-layerWidth})` : undefined  
        }}
        maxHeight="calc(80vh - 13rem)" 
        columns={[
          { header: "ID", headerStyle: { width: '3.4rem' }, css: "t-c c-purple",
            getValue: e => e.ID
          },
          { header: "Nombre", cardColumn: [3,2],
            getValue: e => e.Nombre, cardCss: "h5 c-steel",
          },
        ]}    
      />
      <SideLayer id={1} style={{ width: `calc(var(--page-width) * ${layerWidth})` }}
        title={"Producto " + (productoForm()?.Nombre||"(Nuevo)") }
        onClose={() => {

        }}
        onSave={() => {

        }}
      >
        <div></div>
        <BarOptions selectedID={layerView()} class="w100"
          options={[[1,'Información'],[2,'Ficha'],[3,'Fotos']]}
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
            <Input saveOn={productoForm()} save="SubUnidad" 
              css="w-06x mb-10" label="Nombre" 
            />
            <Input saveOn={productoForm()} save="SbnPrecio" 
              css="w-05x mb-10" label="Precio Base" type="number"
            />
            <Input saveOn={productoForm()} save="SbnDescuento" 
              css="w-04x mb-10" label="Descuento" type="number"
            />
            <Input saveOn={productoForm()} save="SbnPreciFinal" 
              css="w-05x mb-10" label="Precio Final" type="number"
            />
          </div>
          <div class="w100 py-04 px-04">
            <QTable data={productoForm().propiedades||[]}
              maxHeight="40rem" tableCss="single-color"
              columns={[
                { header: "Propiedad", headerStyle: { width: '8rem' },
                  render: e => {
                    return  <CellEditable save="nombre" saveOn={e} 
                      contentClass="flex ai-center nowrap" required={true} />
                  }
                },
                { header: "Opciones", cardColumn: [3,2],
                  render: e => {
                    return <CellTextOptions saveOn={e} save="options" />
                  }
                },
                { header: <div>
                    <button class="bn1 s2 b-green" onclick={ev => {
                      ev.stopPropagation()
                      const propiedades = [...productoForm().propiedades]
                      propiedades.push({})
                      productoForm().propiedades = propiedades
                      setProductoForm({...productoForm()})
                    }}>
                      <i class="icon-plus"></i>
                    </button>
                  </div>, 
                  headerStyle: { width: '2.6rem' }, css: "t-c",
                  cardColumn: [1,2],
                  render: (e,i) => {
                    const onclick = (ev: MouseEvent) => {
                      ev.stopPropagation()
                    }
                    return <button class="bnr2 b-red b-card-1" onClick={ev => {
                      ev.stopPropagation()
                      const propiedades = [...productoForm().propiedades]
                      propiedades.push({})
                      productoForm().propiedades = propiedades
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