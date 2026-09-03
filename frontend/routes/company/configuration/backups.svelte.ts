import { GetHandler, POST } from '$libs/ui-runtime.svelte';

// Access id from backend/access.toml. It gates the "Backups" tab of this route, whose other
// tab (My Company) carries its own id.
export const BACKUPS_ACCESS_ID = 4

export interface IBackup {
  Name: string
  Size: number
  upd: number
}

export class BackupsService extends GetHandler {
  route = "backups"
  useCache = { min: 0, ver: 1 }
  keyID = "Name"

  backups: IBackup[] = $state([])

  handler(records: IBackup[]) {
    records.sort((a,b) => a.Name > b.Name ? 1 : -1)
    this.backups = records || []
  }

  constructor() {
    super()
    this.fetch()
  }

  refreshBackups() {
    this.fetch()
  }
}

export const createBackup = () => {
  return POST({
    data: {},
    route: "backup-create",
    successMessage: "Backup generado exitosamente",
    errorMessage: "Error al generar el backup"
  })
}

export const restoreBackup = (name: string) => {
  return POST({
    data: { Name: name },
    route: "backup-restore",
    successMessage: "Backup restaurado exitosamente",
    errorMessage: "Error al restaurar el backup"
  })
}
