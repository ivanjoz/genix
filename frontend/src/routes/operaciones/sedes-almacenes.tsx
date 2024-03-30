import { Loading, Notify } from "~/core/main";
import { For, Show, createEffect, createSignal, on } from "solid-js";
import { CellEditable } from "~/components/Editables";
import { Input } from "~/components/Input";
import { Modal, SideLayer, openLayers, setOpenLayers, setOpenModals } from "~/components/Modals";
import { QTable } from "~/components/QTable";
import { SearchSelect } from "~/components/SearchSelect";
import { formatTime, throttle } from "~/core/main";
import { pageView } from "~/core/menu";
import { PageContainer } from "~/core/page";
import { IAlmacen, IAlmacenLayout, ISede, postAlmacen, postSede, usePaisCiudadesAPI, useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes";
import { max, average } from 'simple-statistics'

export default function SedesAlmacenes() {

  const [filterText, setFilterText] = createSignal("")
  const [sedeForm, setSedeForm] = createSignal({} as ISede)
  const [almacenForm, setAlmacenForm] = createSignal({} as IAlmacen)
  const [almacenes, setAlmacenes] = useSedesAlmacenesAPI()
  const [paisCiudades] = usePaisCiudadesAPI()
  const layerWidth = 0.48
  
  createEffect(()=> {
    for(let e of paisCiudades()?.distritos || []){
      e._nombre = `${e.Departamento?.Nombre||"-"} ► ${e.Provincia?.Nombre||""} ► ${e.Nombre}`
    }
  })

  const saveSede = async (isDelete?: boolean) => {
    const form = sedeForm()
    if((form.Nombre?.length||0) < 4 || (form.Direccion?.length||0) < 4){
      Notify.failure("El nombre y la dirección deben tener al menos 4 caracteres.")
      return
    }

    console.log("guardando sede::", form)

    Loading.standard("Creando /Actualizando Sede...")
    try {
      var result = await postSede(form)
    } catch (error) {
      Notify.failure(error as string); Loading.remove(); return
    }

    let sedes_ = [...almacenes().Sedes]

    if(form.ID){
      const selected = almacenes().Sedes.find(x => x.ID === form.ID)
      if(selected){ Object.assign(selected, form) }
      if(isDelete){ sedes_ = sedes_.filter(x => x.ID !== form.ID) }
    } else {
      form.ID = result.ID
      sedes_.unshift(form)
    }

    setAlmacenes({...almacenes(), sedes: sedes_})
    setOpenModals([])
    Loading.remove()
  }

  const saveAlmacen = async (isDelete?: boolean) => {
    const form = almacenForm()
    if((form.Nombre?.length||0) < 4){
      Notify.failure("El nombre debe tener al menos 4 caracteres.")
      return
    } else if(!form.SedeID){
      Notify.failure("Debe seleccionar una sede.")
      return
    }

    console.log("guardando almacen::", form)

    Loading.standard("Creando /Actualizando Almacén...")
    try {
      var result = await postAlmacen(form)
    } catch (error) {
      Notify.failure(error as string); Loading.remove(); return
    }

    let almacenes_ = [...almacenes().Almacenes]

    if(form.ID){
      const selected = almacenes().Almacenes.find(x => x.ID === form.ID)
      if(selected){ Object.assign(selected, form) }
      if(isDelete){ almacenes_ = almacenes_.filter(x => x.ID !== form.ID) }
    } else {
      form.ID = result.ID
      almacenes_.unshift(form)
    }

    setAlmacenes({...almacenes(), Almacenes: almacenes_})
    setOpenModals([])
    Loading.remove()
  }

  return <PageContainer title="Sedes & Almacenes"
    views={[[1,"Sedes"],[2,"Almacenes"]]} accesos={[5]}
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
            setSedeForm({ ss: 1 } as ISede)
            setOpenModals([1])
          }}>
            <i class="icon-plus"></i>
          </button>
        </div>
      </div>
      <QTable data={almacenes().Sedes || []} css="w100"
        maxHeight="calc(80vh - 13rem)" style={{ height: '100%' }}
        columns={[
          { header: "ID", headerStyle: { width: '3.4rem' }, css: "t-c c-purple",
            getValue: e => e.ID, cardColumn: [1,2],
          },
          { header: "Nombre", cardColumn: [1,1],
            getValue: e => e.Nombre, cardCss: "h5 c-steel",
          },
          { header: "Dirección", cardColumn: [1,2],
            getValue: e => e.Direccion, cardCss: "h5 c-steel",
          },
          { header: "Ciudad", cardColumn: [3,2],
            getValue: e => {
              if(!e.Ciudad){ return "" }
              const arr = e.Ciudad.split("|")
              return arr[1] + " > " + arr[0]
            }
          },
          { header: "Actualizado", headerStyle: { width: '9rem' },
            css: 'nowrap',
            getValue: e => formatTime(e.upd,"Y-m-d h:n") as string
          },
          { header: "...", headerStyle: { width: '2.6rem' }, css: "t-c",
            cardColumn: [1,2],
            render: (e,i) => {
              const onclick = (ev: MouseEvent) => {
                ev.stopPropagation()
                setOpenModals([1])
                setSedeForm({...e})
              }
              return <button class="bnr2 b-blue b-card-1" onClick={onclick}>
                <i class="icon-pencil"></i>
              </button>
            }
          }
        ]}    
      />
    </Show>
    <Show when={pageView() === 2}>
      <div class="w100">
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
              setAlmacenForm({ ss: 1 } as IAlmacen)
              setOpenModals([2])
            }}>
              <i class="icon-plus"></i>
            </button>
          </div>
        </div>
        <QTable data={almacenes().Almacenes || []}
          tableCss="w-page-t"
          style={{ width: openLayers().length > 0 
            ? `calc(var(--page-width) * ${1-layerWidth})` : undefined  
          }}
          maxHeight="calc(80vh - 13rem)" 
          columns={[
            { header: "ID", headerStyle: { width: '3.4rem' }, css: "t-c c-purple",
              getValue: e => e.ID
            },
            { header: "Sede", cardColumn: [3,2],
              getValue: e => {
                const sede = almacenes().SedesMap.get(e.SedeID)
                return sede?.Nombre || `Sede-${e.SedeID}`
              }, 
              cardCss: "h5 c-steel",
            },
            { header: "Nombre", cardColumn: [3,2],
              getValue: e => e.Nombre, cardCss: "h5 c-steel",
            },
            { header: "Layout", cardColumn: [3,2],
              render: (e,i) => {
                const onclick = (ev: MouseEvent) => {
                  ev.stopPropagation()
                  setOpenLayers([1])
                  setAlmacenForm(JSON.parse(JSON.stringify(e)) as IAlmacen)
                }
                return <div class="w100 flex ai-center jc-between">
                  <div class="flex ai-center">
                    <Show when={e.Layout?.length > 0}>
                      <div class="ff-bold h3">{e.Layout.length}</div>
                      <i class="icon-folder-empty"></i>
                      <div class="mr-04 ml-04 h6 c-steel">X</div>
                      <div class="ff-bold h3">{average(e.Layout.map(x => x.ColCant))}</div>
                      <i class="icon-buffer"></i>
                      <div class="mr-04 ml-04 h6 c-steel">X</div>
                      <div class="ff-bold h3">{average(e.Layout.map(x => x.RowCant))}</div>
                      <i class="icon-cube"></i>
                    </Show>
                  </div>
                  <button class="bnr2 b-blue b-card-1" onClick={onclick}>
                    <i class="icon-pencil"></i>
                  </button>
                </div>
              }
            },
            { header: "Estado", cardColumn: [3,2],
              getValue: e => e.ss, cardCss: "h5 c-steel",
            },
            { header: "Actualizado", headerStyle: { width: '9rem' },
              css: 'nowrap',
              getValue: e => formatTime(e.upd,"Y-m-d h:n") as string
            },
            { header: "...", headerStyle: { width: '2.6rem' }, css: "t-c",
              cardColumn: [1,2],
              render: (e,i) => {
                const onclick = (ev: MouseEvent) => {
                  ev.stopPropagation()
                  setOpenModals([2])
                  setAlmacenForm(JSON.parse(JSON.stringify(e)) as IAlmacen)
                }
                return <button class="bnr2 b-blue b-card-1" onClick={onclick}>
                  <i class="icon-pencil"></i>
                </button>
              }
            }
          ]}    
        />
      </div>
    </Show>
    <Modal id={1} css="w56-78 in-s2"
      title={(sedeForm()?.ID > 0 ? "Actualizar" : "Crear") + " Sede"}
      onSave={()=> {
        console.log("sede a guardar::", sedeForm())
        saveSede()
      }}
      onDelete={()=> { saveSede(true) }}
    >
      <div class="flex-wrap ai-start w100-10">
        <Input saveOn={sedeForm()} save="Nombre" 
          css="w-10x mb-10" label="Nombre" required={true}
          disabled={sedeForm()?.ID > 0}
        />
        <Input saveOn={sedeForm()} save="Descripcion" 
          css="w-14x mb-10" label="Descripción" 
        />
        <Input saveOn={sedeForm()} save="Telefono" 
          css="w-10x mb-10" label="Teléfono"
          disabled={sedeForm()?.ID > 0}
        />
        <Input saveOn={sedeForm()} save="Direccion" 
          css="w-14x mb-10" label="Dirección" required={true}
        />
        <SearchSelect options={paisCiudades().distritos}  required={true}
          label="Departamento | Provincia | Distrito" css="w-24x mb-10"
          keys="ID._nombre" saveOn={sedeForm()} save="CiudadID" 
        />
      </div>
    </Modal>
    <Modal id={2} css="w56-78 in-s2"
      title={(almacenForm()?.ID > 0 ? "Actualizar" : "Crear") + " Almacén"}
      onSave={()=> {
        console.log("almacen a guardar::", almacenForm())
        saveAlmacen()
      }}
      onDelete={()=> { saveAlmacen(true) }}
    >
      <div class="flex-wrap ai-start w100-10">
        <SearchSelect options={almacenes().Sedes}  required={true}
          label="Sede" css="w-12x mb-10"
          keys="ID.Nombre" saveOn={almacenForm()} save="SedeID" 
        />
        <Input saveOn={almacenForm()} save="Nombre" 
          css="w-12x mb-10" label="Nombre" required={true}
        />
        <Input saveOn={almacenForm()} save="Descripcion" 
          css="w-12x mb-10" label="Descripción" 
        />
      </div>
    </Modal>
    <SideLayer id={1} style={{ width: `calc(var(--page-width) * ${layerWidth})` }}
      title={"Layout " + (almacenForm()?.Nombre||"-") }
      onClose={() => {

      }}
      onSave={() => {
        for(let layout of almacenForm().Layout||[]){
          layout.Bloques = []
          for(let key in layout){
            if(key.substring(0,3) === 'xy_'){
              const [_, rw, co] = key.split("_")
              layout.Bloques.push({
                nm: layout[key as keyof IAlmacenLayout] as string,
                rw: parseInt(rw),
                co: parseInt(co)
              })
            }
          }
        }
        console.log("almacen a guardar::", almacenForm())
        saveAlmacen()
      }}
    >
      <AlmacenLayout almacen={almacenForm()}/>
    </SideLayer>
  </PageContainer>
}

interface IAlmacenLayoutLayer {
  almacen: IAlmacen
}

const AlmacenLayout = (props: IAlmacenLayoutLayer) => {

  const [layouts, setLayouts] = createSignal([] as IAlmacenLayout[])

  createEffect(on(() => [props.almacen], 
    () => {
      const layouts_ = props.almacen.Layout||[]
      for(let layout of layouts_){
        for(let e of layout.Bloques||[]){
          layout[`xy_${e.rw}_${e.co}` as unknown as keyof IAlmacenLayout] = e.nm as never
        }
      }
      setLayouts(layouts_)
    }
  ))

  createEffect(() => {
    props.almacen.Layout = layouts()
  })

  return <div class="w100 h100 p-rel">
    <div class="flex jc-between w100 mb-08">
      <div></div>
      <div class="flex ai-center">
        <button class="bn1 b-green" onClick={ev => {
          ev.stopPropagation()
          const maxID = layouts().length > 0 ? max(layouts().map(x => x.ID || 0)) : 0
          layouts().push({ RowCant: 2, ColCant: 3, Name: "", ID: maxID + 1 })
          setLayouts([...layouts()])
        }}>
          <i class="icon-plus"></i>
        </button>
      </div>
    </div>
    <div class="overflow-auto pad-sr6" style={{ "max-height": 'calc(100% - 6.4rem)' }}>
      <Show when={layouts().length === 0}>
        <div class="box-error-ms">
          No hay espacios en al almacén. Agregue uno pulsando en (+)
        </div>
      </Show>
      <For each={layouts()}>
      {(e) => {
        const heads: string[] = []
        const rows: string[] = []
        for(let i = 1; i <= (e.ColCant||1); i++){
          heads.push(`${i}`)
        }
        for(let i = 1; i <= (e.RowCant||1); i++){
          rows.push(`${i}`)
        }

        const doChange = () => {
          const layouts_ = []
          for(let layout of layouts()){
            if(layout === e){ layout = {...layout} }
            layouts_.push(layout)
          }
          setLayouts(layouts_)
        }

        return <div class="card-c12p w100 mb-12">
          <div class="w100 flex ai-center jc-between px-08 py-08">
            <div class="flex ai-center">
              <Input label="" save="Name" saveOn={e} css="w-09x mr-12" inputCss="s3"
                required={true}
              />
              <span class="ff-bold c-steel">Filas</span>
              <Input label="" save="RowCant" saveOn={e} css="w-025x" inputCss="s3" 
                onChange={() => { doChange() }} type="number"
              />
              <span class="ff-bold c-steel">Niveles</span>
              <Input label="" save="ColCant" saveOn={e} css="s3 w-025x" inputCss="s3"
                onChange={() => { doChange() }} type="number"
              />
            </div>
            <button class="bnr4 b-red" 
              style={{ "margin-top": '-12px', "margin-right": '-4px' }}
              onClick={ev => {
                ev.stopPropagation()
                const layouts_ = layouts().filter(x => x !== e)
                setLayouts(layouts_)
              }}  
            >
              <i class="icon-trash"></i>
            </button>
          </div>
          <table class="w100">
            <thead>
              <tr>
                <th style={{ width: '3rem' }}>-</th>
                { heads.map(x => <th style={{ width: `calc(92% / ${heads.length})` }}>{x}</th>) }
              </tr>
            </thead>
            <tbody>
              { rows.map(x1 => {
                  return <tr>
                    <td class="t-c">{x1}</td>
                    { heads.map(y1 => {
                        return <td class="p-rel py-02 px-02" style={{ height: '2.6rem' }}>
                          <CellEditable save={`xy_${x1}_${y1}`} saveOn={e} 
                            class="s2 t-c flex ai-center"
                            contentClass="flex ai-center jc-center"
                          />
                        </td>
                      })
                    }
                  </tr>
                }) 
              }
            </tbody>
          </table>
        </div>
      }}
      </For>
    </div>
  </div>

}
