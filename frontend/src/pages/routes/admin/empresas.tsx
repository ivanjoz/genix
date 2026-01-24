import { QTable } from "~/components/QTable"
import { formatTime } from "~/core/main"
import { PageContainer } from "~/core/page"
import { useEmpresasAPI, useUsuariosAPI } from "~/services/admin/empresas"

export default function Empresas() {

  const [empresas] = useEmpresasAPI()
  const [usuarios] = useUsuariosAPI()

  console.log("empresas", empresas())
  
  return <PageContainer title="Empresas" fetchLoading={true}>
    <div>
      <h1 class="mb-08">Empresas</h1>
      <QTable data={empresas()} css="w100"
        maxHeight="calc(80vh - 13rem)"
        columns={[
          { header: "ID", headerStyle: { width: '3.4rem' }, css: "c",
            getValue: e => e.id
          },
          { header: "Nombre",
            getValue: e => e.nombre
          },
          { header: "Razón Social",
            getValue: e => e.razonSocial
          },
          { header: "RUC",
            getValue: e => e.ruc
          },
          { header: "Estado",
            getValue: e => e.ss
          },
          { header: "Actualizado",
            getValue: e => formatTime(e.upd,"Y-m-d h:n") as string
          },
          { header: "...", headerStyle: { width: '3rem' }, css: "c",
            render: e => {
              return <button class="bnr2 b-blue">
                <i class="icon-pencil"></i>
              </button>
            }
          }
        ]}    
      />
    </div>
  </PageContainer>
}
