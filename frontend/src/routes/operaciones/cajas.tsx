import { createSignal } from "solid-js";
import { deviceType } from "~/app";
import { BarOptions } from "~/components/Cards";
import { CheckBox, Input } from "~/components/Input";
import { Modal, setOpenModals } from "~/components/Modals";
import { ITableColumn, QTable } from "~/components/QTable";
import { SearchSelect } from "~/components/SearchSelect";
import { Loading, Notify, throttle } from "~/core/main";
import { PageContainer } from "~/core/page";
import { cajaTipos } from "~/services/admin/shared";
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes";
import { ICaja, postCaja, useCajasAPI } from "~/services/operaciones/ventas";

export default function Cajas() {
  const [almacenes] = useSedesAlmacenesAPI()
  
  const [filterText, setFilterText] = createSignal("")
  const [layerView, setLayerView] = createSignal(1)
  const [cajaForm, setCajaForm] = createSignal({} as ICaja)
  const [cajas, setCajas] = useCajasAPI()
  
  const columns: ITableColumn<any>[] = [
    { header: "ID", headerStyle: { width: '2.6rem' }, css: 'c c-purple2',
      getValue: e => e.id
    },
    { header: "Tipo", field: "Tipo",
      getValue: e => e.nombre, cardColumn: [1,1],
    },
    { header: "Nombre", field: "Nombre",
      getValue: e => e.nombre, cardColumn: [1,1],
    },
    { header: "Balance", field: "Nombre",
      getValue: e => e.nombre, cardColumn: [1,1],
    },
  ]
  
  const saveCaja = async () => {
    const caja = cajaForm()
    if(!caja.Nombre || !caja.Tipo || !caja.SedeID){
      Notify.failure("Los inputs Nombre, Tipo y Sede son obligatorios")
      return
    }

    Loading.standard("Guardando caja...")
    try {
      postCaja(caja)
    } catch (error) {
      console.warn(error)
      return
    }
    Loading.remove()
  }

  return <PageContainer title="Cajas & Cuentas">
    <div class="flex jc-between mb-06"
      classList={{ "column": [2,3].includes(deviceType()) }}
    >
      <div class="" 
        style={{ width: deviceType() === 1 ? '34%' : '100%' }}
      >
        <div class="flex jc-between w100 mb-10">
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
              setCajaForm({ ID: -1, ss: 1 } as ICaja)
              setOpenModals([1])
            }}>
              <i class="icon-plus"></i>
            </button> 
          </div>
        </div>
        <QTable css="selectable w100" columns={columns} 
          maxHeight="calc(100vh - 8rem - 16px)"
          styleMobile={{ height: '100vh' }}
          data={cajas()?.Records || []}
          selected={1}
          filterText={filterText()} filterKeys={["nombre"]}
          isSelected={(e,id) => e?.ID === id as number}
          onRowCLick={e => {

          }}
        />
      </div>
      <div style={{ width: deviceType() === 1 ? 'calc(64.5% + 1rem)' : '100%' }}
        class="card-c4"
      >
        <BarOptions selectedID={layerView()} class="w100"
          options={[[1,'Información'],[2,'Ficha'],[3,'Fotos']]}
          buttonStyle={{ "min-height": '2.1rem' }} buttonClass="ff-bold"
          onSelect={id => {
            setLayerView(id)
          }}
        />
      </div>
    </div>
    <Modal id={1} title="Cajas"
      onSave={() => {
        saveCaja()
      }}
      onDelete={() => {

      }}
    >
      <div class="w100-10 flex-wrap in-s2">        
        <SearchSelect saveOn={cajaForm()} save="Tipo" css="w-10x mb-10"
          label="Tipo" keys="id.name" options={cajaTipos}
          placeholder="" required={true}
        />
        <Input saveOn={cajaForm()} save="Nombre" 
          css="w-14x mb-10" label="Nombre"required={true}
        />
        <Input saveOn={cajaForm()} save="Descripcion"
          css="w-24x mb-10" label="Descripcion"
        />
        <SearchSelect saveOn={cajaForm()} save="SedeID" 
          css="w-10x mb-10" label="Sede"required={true} options={almacenes().Sedes}
          keys="ID.Nombre"
        />
        <div class="w100 flex jc-between ai-center">
          <div></div>
          <CheckBox label="Balance Negativo" saveOn={cajaForm()} save="Nombre"/>
        </div>
      </div>
    </Modal>
  </PageContainer>
}