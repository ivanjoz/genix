import { normalizeStringN } from "$libs/helpers"

export const PRODUCT_OPTION_LIST_MONEDA_ID = 1
export const PRODUCT_OPTION_LIST_UNIDAD_ID = 2
export const PRODUCT_SHARED_LIST_CATEGORIA_ID = 1
export const PRODUCT_SHARED_LIST_MARCA_ID = 2

export interface IOption {
	i: number, v: string
}

export interface IOptionList {
	listID: number, options: IOption[]
}

export const productoMonedaOptions: IOption[] = [
  { i: 1, v: "PEN" },
  { i: 2, v: "USD" },
]

export const productoUnidadOptions: IOption[] = [
  { i: 1, v: "Kg" },
  { i: 2, v: "g" },
  { i: 3, v: "Libras" },
]

const optionLists: IOptionList[] = [
	{ listID: PRODUCT_OPTION_LIST_MONEDA_ID, options: productoMonedaOptions },
	{ listID: PRODUCT_OPTION_LIST_UNIDAD_ID, options: productoUnidadOptions }
]

const listOptionNameMap: Map<number,Map<string,IOption>> = new Map()

export const getOptionByName = (optionListID: number, name: string): IOption => {	
	
	if (!listOptionNameMap.has(optionListID)) {
		const namesMap: Map<string, IOption> = new Map()
		const options = optionLists.find(x => x.listID === optionListID)?.options || []
		
		for (const opt of options) {
			namesMap.set(normalizeStringN(opt.v), opt)
		}
		listOptionNameMap.set(optionListID, namesMap)
	}
	
	const namesMap = listOptionNameMap.get(optionListID)
	return namesMap?.get(normalizeStringN(name)) as IOption
}
