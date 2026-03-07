import { GetHandler } from '$libs/http.svelte';
import type { ImageSource } from '$components/ImageUploader.svelte';

export interface IProductoPropiedad {
  id: number, nm: string, ss: number
}

export interface IProductoPropiedades {
  ID: number, Nombre: string, Options: IProductoPropiedad[], Status: number
}

export interface IProductoPresentacion {
  id: number, 
  at: number, /* atributo */
  nm: string, /* nombre */
  cl: string, /* color */
  pc: number, /* precio */
  pd: number, /* precio diferencial */
  ss: number  /* estado */
}

export interface IProductoImage {
  n: string /* Nombre del imagen */
  d: string /* descripcion de la imagen */
}

export interface IProducto {
  ID: number,
  TempID?: number,
  Nombre: string
  Descripcion: string
  Precio: number
  Descuento: number
  MonedaID: number
  PrecioFinal: number
  ContentHTML?: string
  Propiedades: IProductoPropiedades[]
  Presentaciones: IProductoPresentacion[]
  Peso?: number
  Volumen?: number
  SbnCantidad?: number
  SbnUnidad?: string
  SbnPrecio?: number
  SbnDescuento?: number
  SbnPreciFinal?: number
  Images?: IProductoImage[]
  Image?: IProductoImage
  CategoriasIDs: number[]
  AtributosIDs?: number[]
  MarcaID: number
  UnidadID: number
  Stock?: {a /* almacen */: number, c /* cantidad */: number}[]
  Params: number[]
  ss: number
  upd: number
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

export interface IProductoResult {
  Records?: IProducto[]
  records: IProducto[]
  recordsMap: Map<number,IProducto>
}

export class ProductosService extends GetHandler<IProducto> {
  route = "productos"
  useCache = { min: 5, ver: 9 }
	inferRemoveFromStatus = true
  prependOnSave = true
	
	makeName(record: Partial<IProducto>) {
		return record.Nombre || ""
	}

  handler(result: IProducto[]): void {
    for(const e of result){
      e.Image = e.Images?.[0]
      e.CategoriasIDs = e.CategoriasIDs || []
    }
    this.records = []
    this.recordsMap = new Map()
    this.nameToRecordMap = new Map()
		this.addSavedRecords(...result)
    this.records.sort((a,b) => b.ID - a.ID)
  }

  constructor(){
    super()
    this.fetch()
  }
}

export const productoAtributos = [
  { id: 1, name: "Color" },
  { id: 2, name: "Talla" },
  { id: 3, name: "Tamaño" },
  { id: 4, name: "Forma" },
  { id: 5, name: "Presentación" },
]
