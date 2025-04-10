import { QTable } from "~/components/QTable"
import { ConfirmWarn, Loading, downloadFile, formatTime } from "~/core/main"
import { PageContainer } from "~/core/page"
import { IBackup, useBackupsAPI } from "~/services/admin/empresas"
import css from "../../css/layout.module.css"
import { formatN } from "~/shared/main"
import { Show, createSignal } from "solid-js"
import { POST } from "~/shared/http"
import { Env } from "~/env"

export default function Backups() {

  const [backups] = useBackupsAPI()
  const [backupSelected, setBackupSelected] = createSignal(null as IBackup)

  const generarBackup = async () => {
    Loading.standard("Generando Backup...")
    try {
      await POST({ data: {}, route: "backup-create" })
    } catch (error) {
      Loading.remove()
      return
    }
    Loading.remove()
  }

  const restaurar = async (name: string) => {
    Loading.standard("Rstaurando Backup...")
    try {
      await POST({
        data: { Name: name  },
        route: "backup-restore",
      })
    } catch (error) {
      Loading.remove()
      return
    }
    Loading.remove()
  }

  return <PageContainer title="Backups & Restore" fetchLoading={true}>
    <div class="w100 grid" style={{ "grid-template-columns": "4fr 3fr" }}>
      <div>
        <div class="flex jc-between ai-center mb-06">
          <div><h2 class="mb-08">Backups</h2></div>
          <div class="flex ai-center">
            <button class="bn1 s4 b-green mr-08" onclick={ev => {
                ev.stopPropagation()
                ConfirmWarn("Generar Backup",
                  `¿Desea generar el backup ahora?`, "SI","NO", 
                  ()=> { generarBackup() })
              }}>
              <i class="icon-plus"></i>
            </button>
            <button class="bn1 s4 d-blue">
              <i class="icon-upload"></i>
            </button>
          </div>
        </div>
        <QTable data={backups()} css="selectable w100"
          maxHeight="calc(80vh - 13rem)"
          selected={backupSelected()}
          isSelected={e => e.Name === backupSelected()?.Name}
          onRowCLick={e => {
            if(backupSelected() === e){ e = null }
            setBackupSelected(e)
          }}
          columns={[
            { header: "Created", headerStyle: { width: '11rem' }, css: "nowrap",
              getValue: e => formatTime(e.upd,"Y-m-d h:n") as string
            },
            { header: "Nombre",
              getValue: e => e.Name
            },
            { header: "Tamaño", css: "t-c",
              getValue: e => {
                return `${formatN(e.Size/1000/1000, 2)} mb`
              }
            },
          ]}    
        />
      </div>
      <div class="" style={{ "margin-left": "20px" }}>
        <h2 class="mb-08">Restore</h2>
        <div  class={`w100 ${css.layer_propiedades} py-10 px-12`}>
          <Show when={!backupSelected()}>
            <div class="w100 flex-wrap box-error-ms mt-16">Seleccione un Backup</div>
          </Show>
          <Show when={backupSelected()}>
            <div class="flex w100 jc-between">
              <div>
                <div class="w100 flex-wrap">{backupSelected().Name}</div>
                <div>{formatN(backupSelected().Size/1000/1000, 2)} mb</div>
              </div>
              <button class="bn1 d-purple" onclick={ev =>{
                ev.stopPropagation()
                const s3key = ["backups",1, backupSelected().Name].join("/")
                const url = Env.S3_URL + s3key
                console.log("url to download::", url)
                downloadFile(url)
              }}>
                <i class="icon-download"></i>
              </button>
            </div>
            <div class="flex jc-center w100 mt-16">
              <button class="bn1 d-blue" onclick={ev => {
                ev.stopPropagation()
                ConfirmWarn("Restaurar Backup",
                  `Restaurar el backup realizado el ${formatTime(backupSelected().upd,"Y-m-d h:n")}`,
                  "SI","NO", ()=> {
                    restaurar(backupSelected().Name)
                  })
              }}>
                Restaurar <i class="icon-database"></i>
              </button>
            </div>
          </Show>
        </div>
      </div>
    </div>
  </PageContainer>
}
