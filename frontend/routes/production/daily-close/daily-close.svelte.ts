import { GET, GetHandler, POST } from '#libs/ui-runtime.svelte.ts'
import type { IDailyClosePendingDay, IDailyClosePreview, IProductionClose } from './daily-close.ts'

// The past closes of one warehouse, newest first.
export class ProductionClosesService extends GetHandler<IProductionClose> {
  route = 'production-closes'
  useCache = { min: 0.1, ver: 1 }

  handler(result: { records?: IProductionClose[] }): void {
    // A close with ss = 0 was voided: the delta still sends it (without ss), the list drops it.
    const records = (result?.records || []).filter((productionClose) => productionClose.ss)
    records.sort((leftClose, rightClose) => rightClose.ID - leftClose.ID)
    this.records = records
  }

  constructor(warehouseID: number) {
    super()
    this.route += `?warehouse-id=${warehouseID}`
  }
}

// The days of the warehouse with made products delivered and their stock exit still pending.
export const getDailyClosePending = async (warehouseID: number): Promise<IDailyClosePendingDay[]> => {
  return await GET({ route: `production-close-pending?warehouse-id=${warehouseID}` }) as IDailyClosePendingDay[] || []
}

// The close of the day as it would run now; nothing is written.
export const getDailyClosePreview = async (warehouseID: number, suppliesWarehouseID: number, date: number) => {
  return await GET({
    route: `production-close-preview?warehouse-id=${warehouseID}&supplies-warehouse-id=${suppliesWarehouseID}&date=${date}`,
  }) as IDailyClosePreview
}

// Runs the previewed close. It settles the day's pending sales, so the sales lists re-sync too.
export const postDailyClose = (preview: IDailyClosePreview) => {
  return POST({
    route: 'production-close',
    data: {
      WarehouseID: preview.WarehouseID, SuppliesWarehouseID: preview.SuppliesWarehouseID,
      Date: preview.Date, PreviewHash: preview.PreviewHash,
    },
    refreshRoutes: ['production-closes', 'sale-orders'],
  }) as Promise<IProductionClose>
}
