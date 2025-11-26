export interface IProductoStock {
  ID: string
  SKU?: string
  AlmacenID: number
  ProductoID: number
  PresentacionID: number
  Cantidad: number
  SubCantidad: number
  Lote?: string
  CostoUn?: number
  _cantidadPrev?: number
  _isVirtual?: boolean
  _hasUpdated?: boolean
}
