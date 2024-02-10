import { Loading, Notify } from "~/core/main";
import { Show, Switch, createEffect, createSignal, on } from "solid-js";
import { CheckBoxContainer, Input } from "~/components/Input";
import { ContentLayer, Modal, setOpenModals } from "~/components/Modals";
import { ITableColumn, QTable } from "~/components/QTable";
import { SearchCard, SearchSelect } from "~/components/SearchSelect";
import { throttle } from "~/core/main";
import Modules from "~/core/modules";
import { PageContainer } from "~/core/page";
import { deviceType } from "~/app";
import { IAcceso, IPefil, postPerfil, postSeguridadAccesos, useAccesosAPI, usePerfilesAPI } from "~/services/admin/empresas";
import { arrayToMapN } from "~/shared/main";

const accesosGrupos = [
  { id: 1, name: "Gestión" },
  { id: 2, name: "Seguridad" },
  { id: 3, name: "Maestros" },
  { id: 4, name: "Productos" },
  { id: 5, name: "Reportes" },
]

const accesoAcciones = [
  { id: 1, name: "Visualizar", short:  "VER", 
    icon: "icon-eye", color: "#00c07d", color2: "#49c99c" },
  { id: 2, name: "Editar", short:  "EDITAR", 
    icon: "icon-pencil", color: "#0080f9" },
  { id: 3, name: "Eliminar", short:  "ELIMINAR", 
    icon: "icon-trash", color: "#0080f9" },
  { id: 7, name: "Todo", short:  "EDITAR", 
    icon: "icon-shield", color: "#af12eb", color2: "#d35eff" },
]

const accesoAccionesMap = arrayToMapN(accesoAcciones,'id')


export default function PerfilesAccesos() {
  const [accesos, setAccesos]  = useAccesosAPI()
  const [perfiles, setPerfiles] = usePerfilesAPI()

  const accesosGruposMap = arrayToMapN(accesosGrupos,'id')
  const modulesMap = arrayToMapN(Modules,'id')

  const [accesoForm, setAccesoForm] = createSignal({} as IAcceso)
  const [perfilForm, setPerfilForm] = createSignal({} as IPefil)
  const [moduleSelected, setModuleSelected] = createSignal(0)
  const [filterText, setFilterText] = createSignal("")
  const [accesoEdit, setAccesoEdit] = createSignal(false)

  const accesosGrouped = () => {
    const gruposMap: Map<string,IAcceso[]> = new Map()
    const moduleSelectedID = moduleSelected()

    for(let acs of accesos()){
      for(let md of acs.modulosIDs){
        if(moduleSelectedID && moduleSelectedID !== md){ continue }
        const key = [md, acs.grupo].join("_")
        if(!gruposMap.has(key)){ gruposMap.set(key, []) }
        gruposMap.get(key).push(acs)
        break
      }
    }

    const accesosGrouped_ = []
    for(let [key, accesosGroup] of gruposMap){
      const [moduleID, group] = key.split("_").map(x => parseInt(x))
      accesosGrouped_.push({
        moduleID, 
        group, 
        accesos: accesosGroup,
        groupName: accesosGruposMap.get(group)?.name||"",
        moduleName: moduleSelectedID ? "" : modulesMap.get(moduleID)?.name || ""
      })
    }

    return accesosGrouped_
  }

  const columns: ITableColumn<any>[] = [
    { header: "ID", headerStyle: { width: '2.6rem' }, css: 'c c-purple2',
      getValue: e => e.id
    },
    { header: "Perfil", field: "nombre",
      getValue: e => e.nombre, cardColumn: [1,1],
    },
    { header: "...", headerStyle: { width: '2.6rem' }, cardColumn: [1,2],
      render: e => {
        return <button class="bnr2 b-blue" onClick={ev => {
          ev.stopPropagation()
          setPerfilForm({...e})
          setOpenModals([2])
        }}>
          <i class="icon-pencil"></i>
        </button>
      }
    }
  ]
  
  const saveAcceso = async () => {
    const form = accesoForm()
    if(!form.nombre || !form.orden || (form.acciones?.length||0) == 0 
      || !form.grupo || (form.modulosIDs.length||0) === 0){
      Notify.failure("Faltan propiedades para agregar el acceso.")
      return
    }

    Loading.standard("Actualizando Acceso...")

    try {
      var result = await postSeguridadAccesos(form)
    } catch (error) {
      Notify.failure(error as string); Loading.remove(); return
    }

    Loading.remove()
    console.log("acceso resultado::", result)

    let accesos_ = [...accesos()]
    if((form.id||0) <= 0) form.id = result.id
    const current = accesos_.find(x => x.id === form.id)

    if(current){
      Object.assign(current, form)
    } else {
      accesos().push(form) 
    }
    setAccesoForm({} as IAcceso)
    setAccesos([...accesos()])
    setOpenModals([])
  }

  const savePerfil = async (onDelete?: boolean, isAccesos?: boolean) => {
    const form = perfilForm()
    if(!form.nombre){
      Notify.failure("Faltan propiedades para agregar el acceso.")
      return
    }

    if(isAccesos){
      form.accesos = []
      for(let [accesoID, niveles] of form.accesosMap){
        if(niveles.length === 0){ form.accesosMap.delete(accesoID) }
        for(let n of niveles){ 
          form.accesos.push( accesoID * 10 + n )
        }
      }
      const accesosFiltered = accesos().filter(x => form.accesosMap.has(x.id))
      const modulosIDSet: Set<number> = new Set()
      for(let e of accesosFiltered){
        for(let md of e.modulosIDs){ modulosIDSet.add(md) }
      }
      form.modulosIDs = [...modulosIDSet]
    }

    console.log("perfil:: ", form)
    Loading.standard("Actualizando Perfil...")

    try {
      var result = await postPerfil(form)
    } catch (error) {
      Notify.failure(error as string); Loading.remove(); return
    }

    const perfiles_ = [...perfiles().perfiles]
    perfiles().perfiles = perfiles_

    if((form.id||0) <= 0) form.id = result.id
    const current = perfiles_.find(x => x.id === form.id)
    form._open = false

    if(current){
      Object.assign(current, form)
    } else {
      perfiles_.push(form) 
    }

    Loading.remove()
    console.log("perfil resultado::", result)
    setPerfilForm({} as IPefil)
    setPerfiles({...perfiles()})
    setOpenModals([])
  }

  const checkIfSelected = () => {
    return perfilForm().id
  }
  
  return <PageContainer title="Perfiles & Accesos" fetchLoading={true}>

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
              setPerfilForm({ ss: 1 } as IPefil)
              setOpenModals([2])
            }}>
              <i class="icon-plus"></i>
            </button> 
          </div>
        </div>
        <QTable css="selectable w100" columns={columns} 
          maxHeight="calc(100vh - 8rem - 16px)"
          styleMobile={{ height: '100vh' }}
          data={perfiles().perfiles}
          selected={perfilForm().id}
          filterText={filterText()} filterKeys={["nombre"]}
          isSelected={(e,id) => e.id === id as number}
          onRowCLick={e => {
            if(e.id === perfilForm().id){
              setPerfilForm({} as IPefil) 
            } else {
              setPerfilForm({...e, _open: true }) 
            }
          }}
        />
      </div>
      <ContentLayer showMobileLayer={perfilForm()._open}
        style={{ width: deviceType() === 1 ? '64.5%' : '100%' }}
        onClose={() => {
          setPerfilForm({} as IPefil) 
        }}
      >
        <div class="flex jc-between w100">
          <div class="ff-bold h2">
            <span class="">Accesos</span>
            <Show when={accesoEdit()}><span class="c-red">(Modo Edición)</span></Show>
            <Show when={perfilForm().id > 0}>
              <span class="mr-04">:</span>
              <span class="c-purple2 ml-04">{perfilForm().nombre}</span>
            </Show>
          </div>
          <div class="flex ai-center m-corner-r">
            { accesoEdit() &&
              <button class="bn1 b-green mr-08" onClick={ev => {
                ev.stopPropagation()
                setAccesoForm({ ss: 1 } as IAcceso)
                setOpenModals([1])
              }}>
                <i class="icon-plus"></i>
              </button>
            }
            { perfilForm().id > 0 &&
              <button class="bn1 d-blue" onClick={ev => {
                ev.stopPropagation()
                savePerfil(false, true)
              }}>
                <i class="icon-floppy"></i><span class="m-hide">Guardar</span>
              </button>
            }
            <Show when={!perfilForm().id}>
              <button class="bn1 b-white" onClick={ev => {
                ev.stopPropagation()
                setAccesoEdit(!accesoEdit())
              }}>
                { accesoEdit() ?
                  <i class="c-red icon-cancel"></i> :
                  <i class="icon-pencil"></i>
                }
              </button> 
            </Show>
          </div>
        </div>
        <Show when={!accesoEdit() && !perfilForm().id}>
          <div class="box-error-ms mb-08 w-fit">
            Debe seleccionar un perfil para editar sus accesos.
          </div>
        </Show>
        { accesosGrouped().map(ag => {
            return <>
              <div class="ff-bold c-purple4 h5">
                {ag.moduleName.toUpperCase()}{" > "}{ag.groupName.toUpperCase()}
              </div>
              <div class="flex-wrap" 
                classList={{ 'px-04': deviceType() !== 1 }}
                style={{ margin: '0 -8px', width: 'calc(100% + 16px)' }}>
                { ag.accesos.map(x => {
                    return <AccesoCard isEdit={accesoEdit()} acceso={x} 
                      saveOn={perfilForm().id ? perfilForm() : null}
                      onEdit={() => { 
                        console.log("acceso seleccionado::", x)
                        setAccesoForm({...x})
                        setOpenModals([1])
                      }}
                    />
                  })
                }
              </div>
            </>
          })
        }
      </ContentLayer>
    </div>
    <Modal id={1} title={accesoForm().id ? "Editando Acceso" : "Creando Acceso"}
      css="w44-64 in-s2" isEdit={!!accesoForm().id}
      onSave={()=> { saveAcceso() }}
      onDelete={()=> { }}
    >
      <div class="flex-wrap w100-10">
        <Input saveOn={accesoForm()} save="nombre" 
          css="w-14x mb-10" label="Nombre" required={true}
        />
        <SearchSelect saveOn={accesoForm()} save="grupo" 
          css="w-10x mb-10" label="Grupo" required={true}
          options={accesosGrupos} keys="id.name"
        />
        <Input saveOn={accesoForm()} save="descripcion" 
          css="w-16x mb-10" label="Descripcion"
        />
        <Input saveOn={accesoForm()} save="orden" 
          css="w-08x mb-10" label="Orden" type="number"
        />
        <SearchCard saveOn={accesoForm()} save="modulosIDs" 
          css="w100 mb-10" label="Modulos" required={true}
          options={Modules} keys="id.name"
        />
        <div class="flex ai-center line-c1 w-24x mb-04">
          <div></div><div class="ff-bold h3">Acciones</div><div></div>
        </div>
        <CheckBoxContainer save="acciones" saveOn={accesoForm()} 
          css="w-24x mb-10" options={accesoAcciones} keys="id.name"
        />
      </div>
    </Modal>
    <Modal id={2} title={perfilForm().id ? "Editando Perfil" : "Creando Perfil"}
      css="w44-64 in-s2" isEdit={!!accesoForm().id}
      onSave={()=> { savePerfil() }}
      onClose={() => { setPerfilForm({} as IPefil) }}
      onDelete={()=> { }}
    >
      <div class="flex-wrap w100-10">
        <Input saveOn={perfilForm()} save="nombre" 
          css="w-24x mb-10" label="Nombre" required={true}
        />
        <Input saveOn={perfilForm()} save="descripcion" 
          css="w-24x mb-10" label="Descripcion"
        />
      </div>
    </Modal>
  </PageContainer>
}

interface IAccesoCard {
  acceso: IAcceso
  saveOn: IPefil
  isEdit: boolean
  onEdit: (()=> void)
}

const AccesoCard = (props: IAccesoCard) => {
  const [acciones, setAcciones] = createSignal([] as number[])

  createEffect(on(()=> props.saveOn,
    ()=> {
      setAcciones(props.saveOn?.accesosMap?.get(props.acceso.id)||[])
    })
  )

  const accionColor = () => {
    if(acciones().length === 0 || props.isEdit) return undefined
    const acciones_ = acciones()
    acciones_.sort().reverse()
    const accion = accesoAccionesMap.get(acciones_[0]||0)
    return accion?.color2 || accion?.color || ""
  }

  const cN = () => {
    let cN = "acceso-card"
    if(props.isEdit){ cN += " sel" }
    if([2,3].includes(deviceType())){ cN += " mobile" }
    return cN
  }

  return <div class={cN()}
      style={{ "border-left-color": accionColor() }}
      onClick={ev => {
        if([1].includes(deviceType())){ return }
        ev.stopPropagation()
        if(acciones().length >= props.acceso.acciones.length){
          setAcciones([])
        } else {
          const acciones_ = [...acciones()]
          const missing = props.acceso.acciones.filter(x => !acciones_.includes(x))
          acciones_.push(missing[0])
          setAcciones(acciones_.filter(x => x))
        }
      }}
    >
    <div class="mr-04">{ props.acceso.nombre }</div>
    <Show when={[2,3].includes(deviceType())}>
      <div class="line-1 p-abs" style={{ 'background-color': accionColor() }}></div>
    </Show>
    <Show when={props.isEdit}>
      <div class="p-abs flex-center bn1 i-edit" onclick={ev => {
        ev.stopPropagation()
        if(props.onEdit){ props.onEdit() }
      }}>
        <i class="icon-pencil"></i>
      </div>
      <div class="p-abs flex-center a-id c-purple3">{props.acceso.id}</div>
    </Show>
    <Show when={!props.isEdit && props.saveOn}>
      <div class="acciones-ac1">
        { acciones().map(id => {
            console.log("acciones ids:: ",id)
            const accion = accesoAccionesMap.get(id)
            return <div class={"bnc1"} style={{ "background-color": accion.color }}>
              <i class={accion.icon}></i>
            </div>  
          })
        }
      </div>
      <div class="acciones-ac2 w100 flex jc-center z10">
        { props.acceso.acciones.map(id => {
            const accion = accesoAccionesMap.get(id)
            const selected = acciones().includes(id)
            return <div class="flex-center h5 ff-bold lh-10"
              style={{ 
                "background-color": selected ? accion.color : undefined,
                "border-color": selected ? accion.color : undefined,
                "color": selected ? "white" : undefined,
              }}
              onClick={ev => {
                ev.stopPropagation()
                let newAcciones = [...acciones()]
                if(newAcciones.includes(id)){
                  newAcciones = newAcciones.filter(x => x !== id)
                } else {
                  newAcciones.push(id)
                }
                console.log("seteando acciones:: ", newAcciones)
                newAcciones.sort((a,b) => b - a)
                if(newAcciones.length === 0){ 
                  props.saveOn.accesosMap.delete(props.acceso.id) 
                } else {
                  props.saveOn.accesosMap.set(props.acceso.id, newAcciones) 
                }
                setAcciones(newAcciones)
              }}
            >
              { accion.short }
            </div>
          })
        }
      </div>
    </Show>
  </div>
}