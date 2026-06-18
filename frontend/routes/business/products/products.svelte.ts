import { GetHandler } from '$libs/http.svelte';
import { Env } from '$core/env';
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
  id?: number /* imageID (autoincrement*10 + configDigit) */
  n: string /* Nombre base de la imagen: "<companyID>_<imageID>" (sin carpeta ni extensión) */
  d: string /* descripcion de la imagen */
}

/** Builds the CDN base name (without folder/extension) for a product imageID. */
export const productImageName = (imageID: number) => `${Env.getCompanyID()}_${imageID}`

/** Builds the display image ({id, n, d}) for the product's main image, or undefined when none. */
export const mainProductImage = (e: IProduct): IProductoImage | undefined => {
  if (!e.ImageMain) return undefined
  const index = (e.ImageIDs || []).indexOf(e.ImageMain)
  return { id: e.ImageMain, n: productImageName(e.ImageMain), d: e.ImageDescriptions?.[index] || "" }
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
  ImageMain: number          /* imageID of the primary image */
  ImageIDs: number[]         /* every imageID */
  ImageDescriptions: string[] /* parallel to ImageIDs */
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

export class ProductsService extends GetHandler<IProduct> {
  route = "products"
  routeByID = "p-products-ids"
  useCache = { min: 1, ver: 10 }
	inferRemoveFromStatus = true
  prependOnSave = true
	
	makeName(record: Partial<IProduct>) {
		return record.Name || ""
	}

  handler(result: IProduct[]): void {
    for(const e of result){
      e.Image = mainProductImage(e)
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
