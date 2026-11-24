import { GetHandler } from '#libs/ui-runtime.svelte.ts';
import { Env } from '#core/env.ts';
import type { ImageSource } from '#components/files/ImageUploader.svelte';

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

// Whether a product is bought or made, which decides its accounts and its catalog page
// (Mercaderías / Productos). Mirrors Product.Origin.
export const PRODUCT_ORIGIN_BOUGHT = 1
export const PRODUCT_ORIGIN_MADE = 2

//CATALOG:production.ProductOriginOptions
export const PRODUCT_ORIGIN_OPTIONS: { id: number, name: string }[] = [
  { id: 1, name: 'Bought (merchandise)|Comprado (mercadería)' },
  { id: 2, name: 'Made by the company (finished good)|Fabricado (producto terminado)' },
]

// "Mostrar en" options: IDs are the PRODUCT_CHANNEL_* of #core/product-channels.ts.
//CATALOG:production.ProductSalesChannelOptions
export const PRODUCT_SALES_CHANNEL_OPTIONS: { id: number, name: string }[] = [
  { id: 1, name: 'Physical store|Tienda Física' },
  { id: 2, name: 'Online store|Tienda Online' },
  { id: 3, name: 'Restaurant|Restaurante' },
]

// "Afectación IGV" options: Product.TaxGroup, the IGV treatment a sale line copies.
//CATALOG:finance.ProductTaxGroupOptions
export const PRODUCT_TAX_GROUP_OPTIONS: { id: number, name: string }[] = [
  { id: 1, name: 'Taxed 18 %|Gravado 18 %' },
  { id: 2, name: 'Taxed 10 % (MYPE restaurants, hotels)|Gravado 10 % (MYPE restaurantes, hoteles)' },
  { id: 3, name: 'Exempt|Exonerado' },
  { id: 4, name: 'Unaffected|Inafecto' },
  { id: 5, name: 'Export|Exportación' },
]

// "Detracción (SPOT)" options: Product.DetractionCode, the SUNAT catalog 54 good or service and its rate
// (hundredths of a percent). A factura above S/ 700 with such a line carries a detracción.
//CATALOG:finance.DetractionCodeOptions
export const DETRACTION_CODE_OPTIONS: { id: number, name: string, rate: number }[] = [
  { id: 1, name: '001 Azúcar y melaza de caña', rate: 1000 },
  { id: 4, name: '004 Recursos hidrobiológicos', rate: 400 },
  { id: 5, name: '005 Maíz amarillo duro', rate: 400 },
  { id: 8, name: '008 Madera', rate: 400 },
  { id: 9, name: '009 Arena y piedra', rate: 1000 },
  { id: 10, name: '010 Residuos, subproductos, desechos', rate: 1500 },
  { id: 12, name: '012 Intermediación laboral y tercerización', rate: 1200 },
  { id: 14, name: '014 Carnes y despojos comestibles', rate: 400 },
  { id: 19, name: '019 Arrendamiento de bienes muebles', rate: 1000 },
  { id: 20, name: '020 Mantenimiento y reparación de bienes muebles', rate: 1200 },
  { id: 21, name: '021 Movimiento de carga', rate: 1000 },
  { id: 22, name: '022 Otros servicios empresariales', rate: 1200 },
  { id: 24, name: '024 Comisión mercantil', rate: 1000 },
  { id: 25, name: '025 Fabricación de bienes por encargo', rate: 1000 },
  { id: 26, name: '026 Servicio de transporte de personas', rate: 1000 },
  { id: 27, name: '027 Servicio de transporte de carga', rate: 400 },
  { id: 30, name: '030 Contratos de construcción', rate: 400 },
  { id: 37, name: '037 Demás servicios gravados con el IGV', rate: 1200 },
]

// IProductionSupply mirrors production.ProductionSupply, one line of IProduct.ProductionSupplies.
// Quantity is packed (units*1000 + sub) in SubDivisor, the supply's sub-unit divisor (1 when it has none).
export interface IProductionSupply {
  SupplyID: number
  Quantity: number
  SubDivisor: number
  // Expected waste on top of Quantity, in basis points: 1500 = 15%.
  WasteBasisPoints?: number
}

// IProductionLabor mirrors production.ProductionLabor: minutes of one labor type for the recipe's
// ProductionYieldPortions units. An estimate only; the actual labor comes from the hours logged.
export interface IProductionLabor {
  LaborTypeID: number
  Minutes: number
}

//STRUCT:production.Product
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
  CurrencyID: number
  UnitID: number
  Discount: number
  FinalPrice: number
  Weight: number
  Volume: number
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
  ver: number
  AccountCode: number  /* a supply's default PCGE account; 0 on a product */
  Origin: number       /* 1 bought (merchandise) · 2 made (finished good); on a supply 0, or 2 when semi-finished */
  WebExtraStock: number /* units the web store may sell beyond real stock ("Sobreventa permitida") */
  SalesChannels: number /* "Mostrar en": bitmask of PRODUCT_CHANNEL_* IDs; 0 = every channel */
  TaxGroup: number      /* "Afectación IGV": PRODUCT_TAX_GROUP_OPTIONS, required on a sold product */
  DetractionCode?: number /* "Detracción (SPOT)": DETRACTION_CODE_OPTIONS, 0 when not subject to it */
  /* recipe and cost of a made product (docs/PRODUCTION_PLAN.md §2); zero otherwise. Costs are PEN cents */
  ProductionYieldPortions: number          /* units of the product the recipe makes */
  ProductionSupplies: IProductionSupply[]  /* supplies for ProductionYieldPortions units */
  ProductionExtraCostPerUnit: number
  ProductionStandardCost: number           /* when > 0, the unit cost whatever the supplies cost */
  ProductionLabor: IProductionLabor[]      /* minutes per labor type for ProductionYieldPortions units */
  /* lot policy (docs/MANUFACTURING_PLAN.md §10) */
  TracksLots: boolean          /* a made product gets one lot per order receipt; a bought one is received with a lot */
  ShelfLifeDays: number        /* default expiry of a new lot, days from its date; 0 = none */
  QualityHoldOnEntry: boolean  /* new lots enter in Cuarentena */
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
  /** Holds the raw unit label read from Excel so we can resolve UnitID afterward. */
  _unidadNombre?: string
  /** Holds the raw currency label read from Excel so we can resolve CurrencyID afterward. */
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
  routeByID = "products-ids"
  // ver 14: the recipe moved from Production (one struct) to the flat Production* fields.
  // ver 15: recipe waste and ProductionLabor.
  // ver 16: the lot policy (TracksLots, ShelfLifeDays, QualityHoldOnEntry).
  useCache = { min: 1, ver: 16 }
	inferRemoveFromStatus = true
  prependOnSave = true

	makeName(record: Partial<IProduct>) {
		return record.Name || ""
	}

  handler(result: { records?: IProduct[] }): void {
    const products = result?.records || []
    for(const e of products){
      e.Image = mainProductImage(e)
      e.CategoryIDs = e.CategoryIDs || []
    }
    this.records = []
    this.recordsMap = new Map()
    this.nameToRecordMap = new Map()
		this.addSavedRecords(...products)
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
