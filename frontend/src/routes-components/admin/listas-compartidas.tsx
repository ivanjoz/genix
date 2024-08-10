import { Notify } from "~/core/main"
import { createEffect, createSignal } from "solid-js"
import { CellEditable } from "~/components/Editables"
import { ITableColumn, QTable } from "~/components/QTable"
import { Loading } from "~/core/main"
import { IListaRegistro, IListas, INewIDToID, postListaRegistros } from "~/services/admin/listas-compartidas"
import { arrayToMapN } from "~/shared/main"

interface IListasCompartidasLayer {
  listas: IListas
  listaID: number
}

export const ListasCompartidasLayer = (props: IListasCompartidasLayer) => {

  const [registros, setRegistros] = createSignal([] as IListaRegistro[])
  const [registrosUpdated, setRegistrosUpdated] = createSignal(new Map())

  let counter = -1

  createEffect(() => {
    const registros = (props.listas?.Records||[]).filter(x => x.ListaID === props.listaID)
    setRegistros(registros)
    setRegistrosUpdated(new Map())
  })

  const onchange = (e: IListaRegistro) => { 
    registrosUpdated().set(e.ID, e)
    setRegistrosUpdated(new Map(registrosUpdated()))
  }

  const columns: ITableColumn<IListaRegistro>[] = [
    { header: 'ID', headerStyle: { width: '2.6rem' },
      render: (e) => {
        if(e.ID < 0){
          return <i class="c-red icon-arrows-cw"></i>
        }
        return e.ID
      }
    },
    { header: 'Nombre', 
      render: e => {
        return <CellEditable saveOn={e} save="Nombre" 
          contentClass="flex ai-center" required={true}
          onChange={() => { onchange(e) }}
        />
      }
    },
    { header: 'Descripción', 
      render: e => {
        return <CellEditable saveOn={e} save="Descripcion" 
          contentClass="flex ai-center" 
          onChange={() => { onchange(e) }}
        />
      }
    },
    { header: "...", headerStyle: { width: '2.6rem' }, css: "t-c",
      cardColumn: [1,2],
      render: e => {
        const onclick = (ev: MouseEvent) => {
          ev.stopPropagation()
          const newRegistros = registros().filter(x => x.ID !== e.ID)
          if(e.ID < 0){
            registrosUpdated().delete(e.ID)
          } else {
            e.ss = 0
            registrosUpdated().set(e.ID, e)
          }
          setRegistros(newRegistros)
          setRegistrosUpdated(new Map(registrosUpdated()))
        }
        return <button class="bnr2 d-red b-card-1" onClick={onclick}>
          <i class="icon-trash"></i>
        </button>
      }
    }
  ]

  const addRegistro = () => {
    const reg = { 
      ListaID: props.listaID, ID: counter, Nombre: '', ss: 1
    } as IListaRegistro
    registros().push(reg)
    counter--
    setRegistros([...registros()])
  }

  const saveRegistros = async () => {
    const registrosToSave = [...registrosUpdated().values()]
    if(registrosToSave.length == 0){
      Notify.failure("No hay registros creados / actualizados")
      return
    }
    Loading.standard("Guardando...")
    let result: INewIDToID[]
    try {
      result = await postListaRegistros(registrosToSave)
    } catch (error) {
      console.warn(error)
    }
    Loading.remove()
    if(!result){ return }

    const newIDsMap = arrayToMapN(result||[],'TempID')
    for(let e of registros()){
      if(e.ID < 0){ e.ID = newIDsMap.get(e.ID)?.NewID || 0}
    }

    console.log("resultado obtenido::", result)
    setRegistros([...registros()])
    setRegistrosUpdated(new Map(registrosUpdated()))
  }

  return <div class="w100">
    <div class="flex jc-between ai-center w100 mb-08">
      <div>
        <h3>Categorías</h3>
      </div>
      <div class="flex a-center">
        { registrosUpdated().size > 0 &&
          <button class="bn1 b-blue mr-12" onclick={ev => {
            ev.stopPropagation()
            saveRegistros()
          }}>
            Guardar<i class="icon-floppy"></i>
          </button>
        }
        <button class="bn1 b-green" onclick={ev => {
          ev.stopPropagation()
          addRegistro()
        }}>
          <i class="icon-plus"></i>
        </button>
      </div>
    </div>
    <QTable data={registros()} 
      css="" tableCss="w-page-t" maxHeight="calc(80vh - 13rem)" 
      columns={columns}
    />
  </div>
}