import { POST } from '$libs/ui-runtime.svelte';

export interface IInitialData {
  SiteID: number
  SiteName: string
  SiteAddress: string
  CityID: number
  WarehouseName: string
  CashBankName: string
}

// Names the user can accept as they are; only the site's address and city must be filled in.
export const initialDataDefaults = (): IInitialData => ({
  SiteID: 0,
  SiteName: "Principal",
  SiteAddress: "",
  CityID: 0,
  WarehouseName: "Central",
  CashBankName: "Caja Principal",
})

export const postInitialData = (data: IInitialData) => {
  return POST({
    data,
    route: "initial-data",
    refreshRoutes: ["locations-warehouses", "cash-banks"]
  })
}
