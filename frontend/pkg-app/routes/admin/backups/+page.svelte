<script lang="ts">
import Page from '$ui/components/Page.svelte';
import type { ITableColumn } from '$ui/components/vTable/types';
import VTable from '$ui/components/vTable/VTable.svelte';
import { ConfirmWarn, formatTime } from '$core/lib/helpers';
import { formatN } from '$core/lib/helpers';
  import pkg from 'notiflix'
const { Loading } = pkg;
  import { BackupsService, createBackup, restoreBackup, type IBackup } from "./backups.svelte";
import { sendServiceMessage } from '$core/lib/sw-cache';
import { Env } from '$core/lib/env';

  const backupsService = new BackupsService()

  let backupSelected = $state(null as IBackup | null)

  const generarBackup = async () => {
    Loading.standard("Generando Backup...")
    try {
      await createBackup()
      // Refresh backups list after creating a new backup
      backupsService.refreshBackups()
    } catch (error) {
      // Error already handled by POST function
    }
    Loading.remove()
  }

  const restaurar = async (name: string) => {
    Loading.standard("Restaurando Backup...")
    try {
      await restoreBackup(name)
      await sendServiceMessage(26, {})
    } catch (error) {
      // Error already handled by POST function
    }
    Loading.remove()
  }

  const downloadBackup = (backup: IBackup) => {
    const s3key = ["backups", 1, backup.Name].join("/")
    const url = Env.S3_URL + s3key
    console.log("url to download::", url)

    const aElement = document.createElement("a")
    aElement.setAttribute("download", backup.Name)
    aElement.href = url
    aElement.setAttribute("target", "_blank")
    aElement.click()
    aElement.remove()
  }

  const columns: ITableColumn<IBackup>[] = [
    {
      header: "Created",
      headerCss: "w-176",
      cellCss: "px-6 nowrap",
      getValue: e => formatTime(e.upd, "Y-m-d h:n") as string
    },
    {
      header: "Nombre",
      highlight: true,
      cellCss: "px-6",
      getValue: e => e.Name
    },
    {
      header: "Tamaño",
      headerCss: "w-120",
      cellCss: "text-center",
      getValue: e => `${formatN(e.Size / 1000 / 1000, 2)} mb`
    }
  ]
</script>

<Page title="Backups & Restore">
  <div class="w-full grid gap-20" style="grid-template-columns: 4fr 3fr;">
    <div>
      <div class="flex items-center justify-between mb-6">
        <div class="h2 ff-bold">Backups</div>
        <div class="flex items-center">
          <button class="bx-green mr-8" onclick={ev => {
            ev.stopPropagation()
            ConfirmWarn(
              "Generar Backup",
              "¿Desea generar el backup ahora?",
              "SI",
              "NO",
              () => { generarBackup() }
            )
          }} aria-label="Generar backup">
            <i class="icon-plus"></i>
          </button>
          <button class="bx-blue" aria-label="Subir backup">
            <i class="icon-upload"></i>
          </button>
        </div>
      </div>
      <VTable
        data={backupsService.backups}
        columns={columns}
        css="selectable w-full"
        tableCss="cursor-pointer"
        maxHeight="calc(80vh - 13rem)"
        selected={backupSelected || undefined}
        isSelected={(e, selected) => {
          if (!selected || typeof selected === 'number') return false
          return e.Name === selected.Name
        }}
        onRowClick={(row) => {
          if (backupSelected?.Name === row.Name) {
            backupSelected = null
          } else {
            backupSelected = row
          }
        }}
      />
    </div>
    <div class="" style="margin-left: 20px;">
      <div class="mb-6 h2 ff-bold">Restore</div>
      <div class="_1 w-full rounded-[8px] min-h-160 py-12 px-16">
        {#if !backupSelected}
          <div class="w-full text-red-500 ff-bold flex-wrap box-error-ms mt-16">
            Seleccione un Backup
          </div>
        {:else}
          <div class="flex w-full justify-between">
            <div>
              <div class="w-full flex-wrap ff-semibold">{backupSelected.Name}</div>
              <div class="text-gray-600 mt-4">{formatN(backupSelected.Size / 1000 / 1000, 2)} mb</div>
            </div>
            <button class="bx-purple" onclick={ev => {
              ev.stopPropagation()
              downloadBackup(backupSelected!)
            }} aria-label="Descargar backup">
              <i class="icon-download"></i>
            </button>
          </div>
          <div class="flex justify-center w-full mt-16">
            <button class="bx-blue" onclick={ev => {
              ev.stopPropagation()
              ConfirmWarn(
                "Restaurar Backup",
                `Restaurar el backup realizado el ${formatTime(backupSelected!.upd, "Y-m-d h:n")}`,
                "SI",
                "NO",
                () => {
                  restaurar(backupSelected!.Name)
                }
              )
            }}>
              Restaurar <i class="icon-database"></i>
            </button>
          </div>
        {/if}
      </div>
    </div>
  </div>
</Page>

<style>
  ._1 {
    background-color: white;
    box-shadow: rgba(99, 99, 99, 0.2) 0px 2px 8px 0px;
  }
</style>
