import { Notify } from "notiflix"
import { createEffect, createSignal } from "solid-js"
import { CellEditable } from "~/components/Editables"
import { ITableColumn, QTable } from "~/components/QTable"
import { Loading } from "~/core/main"
import { IListaRegistro, IListas, postListaRegistros } from "~/services/admin/listas-compartidas"

interface IListasCompartidasLayer {
  listas: IListas
  listaID: number
}

export const ListasCompartidasLayer = (props: IListasCompartidasLayer) => {

  const [registros, setRegistros] = createSignal([] as IListaRegistro[])
  let counter = -1

  createEffect(() => {
    setRegistros((props.listas?.Records||[]).filter(x => x.ListaID === props.listaID))
  })

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
          contentClass="flex ai-center"  required={true}
        />
      }
    },
    { header: 'Descripción', 
      render: e => {
        return <CellEditable saveOn={e} save="Descripcion" 
          contentClass="flex ai-center" 
        />
      }
    },
    { header: "...", headerStyle: { width: '2.6rem' }, css: "t-c",
      cardColumn: [1,2],
      render: (e,i) => {
        const onclick = (ev: MouseEvent) => {
          ev.stopPropagation()
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
    const registrosToSave = registros().filter(x => x.ID < 0)
    if(registrosToSave.length == 0){
      Notify.failure("No hay registros creados / actualizados")
      return
    }
    Loading.standard("Guardando...")
    let result
    try {
      result = await postListaRegistros(registrosToSave)
    } catch (error) {
      console.warn(error)
    }
    Loading.remove()
    if(!result){ return }
    console.log("resultado obtenido::", result)
  }

  return <div class="w100">
    <div class="flex jc-between ai-center w100 mb-08">
      <div>
        <h3>Categorías</h3>
      </div>
      <div class="flex a-center">
        <button class="bn1 b-blue mr-12" onclick={ev => {
          ev.stopPropagation()
          saveRegistros()
        }}>
          Guardar<i class="icon-floppy"></i>
        </button>
        <button class="bn1 b-green" onclick={ev => {
          ev.stopPropagation()
          addRegistro()
        }}>
          <i class="icon-plus"></i>
        </button>
      </div>
    </div>
    <QTable data={registros()} 
      css="" tableCss="w-page" maxHeight="calc(80vh - 13rem)" 
      columns={columns}
    />
  </div>
}