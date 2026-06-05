import { GetHandler } from '$libs/http.svelte';
import type { ImageSource } from '$components/files/ImageUploader.svelte';

export interface IProductProperty {
  id: number, nm: string, ss: number
}

export interface IProductProperties {
  ID: number, Name: string, Options: IProductProperty[], Status: number
}

export interface IProductPresentation {
  id: number,
  at: number, /* atributo */
  nm: string, /* nombre */
  cl: string, /* color */
  pc: number, /* precio */
  pd: number, /* precio diferencial */
  sk?: string, /* sku */
  ss: number  /* estado */
}

export interface IProductoImage {
  n: string /* Nombre del imagen */
  d: string /* descripcion de la imagen */
}

//STRUCT:negocio.Product
export interface IProduct {
  ID: number,
  TempID: number,
  Name: string
  Description: string
  ContentHTML: string
  CategoryIDs: number[]
  BrandID: number
  Params: number[]
  Price: number
  MonedaID: number
  UnidadID: number
  Discount: number
  FinalPrice: number
  Peso: number
  Volumen: number
  SbuQuantity: number
  SbuUnit: string
  SbuPrice: number
  SbuDiscount: number
  SbuFinalPrice: number
  SKU: string
  NameHash: number
  Properties: IProductProperties[]
  Presentations: IProductPresentation[]
  Images: IProductoImage[]
  Stock: any
  ReservedStock: any
  StockStatus: number
  NameUpdated: number
  ss: number
  upd: number
  UpdatedBy: number
  Created: number
  CreatedBy: number
  CategoriesWithStock: number[]
  ccv: number
  /* extra fields */
  Image?: IProductoImage
  AtributosIDs?: number[]
  _stock?: number
  _moneda?: string
  _imageSource?: ImageSource
  /** Holds the raw category names read from Excel so we can resolve IDs afterward. */
  _categoriasNames?: string
  /** Holds the raw brand label read from Excel so we can resolve MarcaID afterward. */
  _marcaNombre?: string
  /** Holds the raw unit label read from Excel so we can resolve UnidadID afterward. */
  _unidadNombre?: string
  /** Holds the raw currency label read from Excel so we can resolve MonedaID afterward. */
	_monedaNombre?: string
  _updatedFields?: string[]
}

export interface IProductResult {
  Records?: IProduct[]
  records: IProduct[]
  recordsMap: Map<number,IProduct>
}

export class ProductosService extends GetHandler<IProduct> {
  route = "productos"
  routeByID = "p-productos-ids"
  useCache = { min: 1, ver: 10 }
	inferRemoveFromStatus = true
  prependOnSave = true
	
	makeName(record: Partial<IProduct>) {
		return record.Name || ""
	}

  handler(result: IProduct[]): void {
    for(const e of result){
      e.Image = e.Images?.[0]
      e.CategoryIDs = e.CategoryIDs || []
    }
    this.records = []
    this.recordsMap = new Map()
    this.nameToRecordMap = new Map()
		this.addSavedRecords(...result)
		this.records.sort((a, b) => b.ID - a.ID)
  }

  constructor(init: boolean = false){
    super()
    if (init) {
      this.fetch()
    }
  }
}

export const productoAtributos = [
  { id: 1, name: "Color" },
  { id: 2, name: "Talla" },
  { id: 3, name: "Tamaño" },
  { id: 4, name: "Forma" },
  { id: 5, name: "Presentación" },
]
