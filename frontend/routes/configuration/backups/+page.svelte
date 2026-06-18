<script lang="ts">
import Page from '$domain/Page.svelte';
import type { ITableColumn } from '$components/vTable/types';
import VTable from '$components/vTable/VTable.svelte';
import { ConfirmWarn, formatTime } from '$libs/helpers';
import Button from '$components/buttons/Button.svelte';
import { formatN } from '$libs/helpers';
import { tr } from '$core/store.svelte';
import T from '$components/misc/T.svelte';
  import pkg from 'notiflix'
const { Loading } = pkg;
  import { BackupsService, createBackup, restoreBackup, type IBackup } from "./backups.svelte";
import { sendServiceMessage } from '$libs/sw-cache';
import { Env } from '$core/env';

  const backupsService = new BackupsService()

  let backupSelected = $state(null as IBackup | null)

  const generarBackup = async () => {
    Loading.standard(tr("Generating Backup...|Generando Backup..."))
    try {
      await createBackup()
      backupsService.refreshBackups()
    } catch (error) {
      // Error already handled by POST function
    }
    Loading.remove()
  }

  const restaurar = async (name: string) => {
    Loading.standard(tr("Restoring Backup...|Restaurando Backup..."))
    try {
      await restoreBackup(name)
      await sendServiceMessage(26, {})
    } catch (error) {
      // Error already handled by POST function
    }
    Loading.remove()
  }

  const downloadBackup = (backup: IBackup) => {
    const aElement = document.createElement("a")
    aElement.setAttribute("download", backup.Name)
    aElement.href = Env.makeCDNRoute("backups","1", backup.Name)
    console.log("url to download::", aElement.href)
    aElement.setAttribute("target", "_blank")
    aElement.click()
    aElement.remove()
  }

  const columns: ITableColumn<IBackup>[] = [
    {
      header: "Created",
      headerCss: "w-176",
      css: "px-6 nowrap",
      getValue: e => formatTime(e.upd, "Y-m-d h:n") as string
    },
    {
      header: "Name|Nombre",
      highlight: true,
      css: "px-6",
      getValue: e => e.Name
    },
    {
      header: "Size|Tamaño",
      headerCss: "w-120",
      css: "text-center",
      getValue: e => `${formatN(e.Size / 1000 / 1000, 2)} mb`
    }
  ]
</script>

<Page title="Backups & Restore">
  <div class="w-full grid gap-20" style="grid-template-columns: 4fr 3fr;">
    <div>
      <div class="flex items-center justify-between mb-6" aria-label="Backups toolbar with generate and upload buttons">
        <div class="h2 ff-bold">Backups</div>
        <div class="flex items-center">
          <Button color="green" icon="icon-[fa--plus]" label="Generates a new database backup snapshot." css="mr-8" onClick={() => {
            ConfirmWarn(tr("Generate Backup|Generar Backup"), tr("Do you want to generate the backup now?|¿Desea generar el backup ahora?"), "YES|SI", "NO",
              () => { generarBackup() })
          }} />
          <Button color="blue" icon="icon-[fa--upload]" label="Opens dialog to upload an existing backup file." />
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
            <T text="Select a Backup|Seleccione un Backup" />
          </div>
        {:else}
          <div class="flex w-full justify-between">
            <div>
              <div class="w-full flex-wrap ff-semibold">{backupSelected.Name}</div>
              <div class="text-gray-600 mt-4">{formatN(backupSelected.Size / 1000 / 1000, 2)} mb</div>
            </div>
            <Button color="purple" icon="icon-[fa--download]" label="Downloads the selected backup file."
              onClick={() => downloadBackup(backupSelected!)} />
          </div>
          <div class="flex justify-center w-full mt-16">
            <Button color="blue" name="Restore|Restaurar" icon="icon-[fa--database]" label="Restores the database from the selected backup." onClick={() => {
              ConfirmWarn(tr("Restore Backup|Restaurar Backup"),
                tr(`Restore the backup from ${formatTime(backupSelected!.upd, "Y-m-d h:n")}|Restaurar el backup realizado el ${formatTime(backupSelected!.upd, "Y-m-d h:n")}`),
                "YES|SI", "NO", () => { restaurar(backupSelected!.Name) })
            }} />
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
