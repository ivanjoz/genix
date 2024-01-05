import { CellEditable } from '~/components/Editables';
import { ITableColumn, QTable } from '~/components/QTable';
import { PageContainer } from '~/core/page';

const columns1: ITableColumn<any>[] = [
  { header: "ID",  id: 101,
    render: e => {
      return <div class="flex ai-center">
        { e._updated &&
          <div class='c-red' style={{ "margin-left": '-4px' }}>
            <i class='icon-arrows-cw'></i>
          </div>
        }
        <div>{e.id}</div>
      </div>
    }
  },
  { header: "Edad",
    getValue: e => e.edad
  },
  { header: "Nombre", id: 103,
    getValue: e => e.nombre
  },
  { header: "Apellidos",
    cellStyle: { padding: '0' },
    render: (e,_,rerender) => <CellEditable save="apellidos" saveOn={e} 
      onChange={() => {
        e._updated = true
        rerender([101])
      }}
    />
  },
  { header: "Edad",
    getValue: e => e.edad
  },
  { header: "...",
    render: (e,idx,rerender) => {

      return <button onClick={ev => {
        ev.stopPropagation()
        e.nombre = e.nombre + "_1"
        rerender([103])
      }}>
        <i class='icon-ok'></i>
      </button>

    }
  }
]

export default function TestTable() {

  const data = makeData()

  return <PageContainer title="Virtualize Scroll">
  <div style={{ padding: '1rem' }}>
      <div style={{ "margin-bottom": '1rem' }}> <h2>VIRTUALIZE SCROLL</h2> </div>
      <QTable columns={columns1} data={data} />
    </div>
  </PageContainer>
}

const makeData = (): any[] => {
  console.log("generando data:: ")

  const records = []

  for(let i = 0; i< 50000; i++){

    const record = {
      id: makeid(12),
      edad: Math.floor(Math.random() * 100),
      nombre: makeid(18),
      apellidos: makeid(23),
      numero: Math.floor(Math.random() * 1000)
    }  
    records.push(record)
  }
  return records
}

function makeid(length: number) {
  let result = '';
  const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  const charactersLength = characters.length;
  let counter = 0;
  while (counter < length) {
    result += characters.charAt(Math.floor(Math.random() * charactersLength));
    counter += 1;
  }
  return result;
}