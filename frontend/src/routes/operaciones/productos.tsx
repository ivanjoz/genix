import { Show, createSignal } from "solid-js";
import { SideLayer, openLayers, setOpenLayers } from "~/components/Modals";
import { QTable } from "~/components/QTable";
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
            setProductoForm({ ss: 1 } as IProducto)
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
      </SideLayer>
    </Show>
  </PageContainer>
}