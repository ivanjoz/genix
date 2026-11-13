// The web store catalog's pure rules: no Svelte, no fetch. docs/ECOMMERCE_CATALOG_SNAPSHOT_PLAN.md.
//
// The backend (backend/business/ecommerce-catalog.go) sends only what the store can sell: the R2 file
// holds every sellable product, and the delta every product touched since a cursor, whole, or as a
// removal (ss: 0). The browser keeps one saved catalog per company and merges deltas into it.

export interface IProductProperty {
	id: number, nm: string, ss: number
}

export interface IProductProperties {
	ID: number, Name: string, Options: IProductProperty[], Status: number
}

export interface IProductoImage {
	n: string /* image name */
	d: string /* image description */
}

/** What the store can sell of one presentation (PresentationID 0: a product with no presentations). */
export interface ICatalogStock {
	PresentationID: number
	Units: number
}

export interface IProduct {
	ID: number
	Name: string
	Description?: string
	Price?: number
	Discount?: number
	FinalPrice?: number
	ContentHTML?: string
	Properties?: IProductProperties[]
	Weight?: number
	Volume?: number
	SbuQuantity?: number
	SbuUnit?: string
	SbuPrice?: number
	SbuDiscount?: number
	SbuFinalPrice?: number
	Images?: IProductoImage[]
	Image?: IProductoImage
	BrandID?: number
	CategoryIDs?: number[]
	Stock?: ICatalogStock[]
	ss: number
	_moneda?: string
}

export interface IProductCategory {
	ID: number
	Name: string
	Description?: string
}

/** A product as the file and the delta carry it. Colbin sends an empty slice as null. */
export interface ICatalogProductRecord {
	ID: number
	Name: string
	CategoryIDs: number[] | null
	BrandID: number
	Price: number
	FinalPrice: number
	Image: number
	Stock: ICatalogStock[] | null
	ss: number
}

/** A brand or a category. */
export interface ICatalogListRecord {
	ID: number
	Name: string
	ss: number
}

export interface ICatalogFile {
	Meta: { Created: number, Updated: number, Generation: number }
	Productos: ICatalogProductRecord[] | null
	Marcas: ICatalogListRecord[] | null
	Categorias: ICatalogListRecord[] | null
}

/** The delta endpoint's answer. With UseFile set only FileCreated and Generation are meaningful. */
export interface ICatalogDelta {
	UseFile: boolean
	FileCreated: number
	Generation: number
	NextSince: number
	Productos: ICatalogProductRecord[] | null
	Marcas: ICatalogListRecord[] | null
	Categorias: ICatalogListRecord[] | null
}

/** What the browser keeps per company: the catalog and the cursor its next delta starts from. */
export interface ISavedCatalog {
	schemaVersion: number
	generation: number
	since: number
	productos: ICatalogProductRecord[]
	marcas: ICatalogListRecord[]
	categorias: ICatalogListRecord[]
}

// Bump when ISavedCatalog or a record changes shape: a saved catalog of another version is discarded.
export const CATALOG_SCHEMA_VERSION = 1;

/** A saved catalog seeded from the file: its next delta starts at the file's Updated. */
export const savedCatalogFromFile = (catalogFile: ICatalogFile): ISavedCatalog => ({
	schemaVersion: CATALOG_SCHEMA_VERSION,
	generation: catalogFile.Meta.Generation,
	since: catalogFile.Meta.Updated,
	productos: catalogFile.Productos ?? [],
	marcas: catalogFile.Marcas ?? [],
	categorias: catalogFile.Categorias ?? [],
});

/** Whether a saved value can be used as is: written by this code version. */
export const isUsableSavedCatalog = (saved: ISavedCatalog | undefined): saved is ISavedCatalog =>
	!!saved && saved.schemaVersion === CATALOG_SCHEMA_VERSION && saved.since > 0;

// A delta record replaces the saved one with its ID; ss: 0 removes it (no longer sellable, deleted).
const mergeCatalogRecords = <T extends { ID: number, ss: number }>(saved: T[], changed: T[] | null): T[] => {
	if (!changed?.length) return saved;
	const recordByID = new Map(saved.map((record) => [record.ID, record]));
	for (const record of changed) {
		if (record.ss > 0) recordByID.set(record.ID, record);
		else recordByID.delete(record.ID);
	}
	return [...recordByID.values()];
};

/** The saved catalog after a delta, its cursor moved to the delta's NextSince. */
export const mergeCatalogDelta = (saved: ISavedCatalog, delta: ICatalogDelta): ISavedCatalog => ({
	schemaVersion: CATALOG_SCHEMA_VERSION,
	generation: delta.Generation,
	since: delta.NextSince,
	productos: mergeCatalogRecords(saved.productos, delta.Productos),
	marcas: mergeCatalogRecords(saved.marcas, delta.Marcas),
	categorias: mergeCatalogRecords(saved.categorias, delta.Categorias),
});

/** The CDN file of a product image: "<companyID>_<imageID>" (backend/production/products.go, same
 *  as productImageName in the admin). '' when there is no image. */
export const productImageName = (companyID: number, imageID: number): string =>
	imageID > 0 ? `${companyID}_${imageID}` : '';

/** A catalog record in the shape the store's components read (the image as {n, d}).
 *  The catalog carries only the image ID. */
export const toStoreProduct = (record: ICatalogProductRecord, companyID: number): IProduct => ({
	ID: record.ID,
	Name: record.Name,
	CategoryIDs: record.CategoryIDs ?? [],
	BrandID: record.BrandID,
	Price: record.Price,
	FinalPrice: record.FinalPrice,
	Image: { n: productImageName(companyID, record.Image), d: '' },
	Stock: record.Stock ?? [],
	ss: record.ss,
});

/** What GET.p-product-detail adds to the catalog record (backend ProductDetail): the extra images
 *  and the HTML sheet. ss 0: the store no longer sells it. */
export interface IProductDetail {
	ID: number
	ContentHTML: string
	ImageIDs: number[] | null
	ImageDescriptions: string[] | null
	ss: number
	upd: number
}

export const PRODUCT_DETAIL_ROUTE = 'p-product-detail';

/** The detail layer's gallery: the catalog's main image first, then the product's other images in
 *  their stored order. Until the detail arrives (or with no detail images) the main image alone. */
export const productGalleryImages = (
	companyID: number, mainImage: IProductoImage | undefined, productDetail: IProductDetail | null,
): IProductoImage[] => {
	const detailImages = (productDetail?.ImageIDs ?? []).map((imageID, imageIndex) => ({
		n: productImageName(companyID, imageID),
		d: productDetail?.ImageDescriptions?.[imageIndex] ?? '',
	}));
	const mainImageName = mainImage?.n || '';
	if (!mainImageName) return detailImages;
	// The detail's copy of the main image carries its description; the catalog's does not.
	const mainGalleryImage = detailImages.find((image) => image.n === mainImageName) ?? { n: mainImageName, d: '' };
	return [mainGalleryImage, ...detailImages.filter((image) => image.n !== mainImageName)];
};

// The web cart orders a product without choosing a presentation: its lines go to checkout as
// PresentationID 0 (web-checkout.ts), so that is the stock that limits it.
export const CART_PRESENTATION_ID = 0;

/** The units of a product the cart may hold: what the store can sell of the presentation the cart
 *  orders. A product missing from the catalog is not sellable: 0. */
export const cartUnitsLimit = (catalogProduct: IProduct | undefined): number =>
	catalogProduct?.Stock?.find((stock) => stock.PresentationID === CART_PRESENTATION_ID)?.Units ?? 0;

/** The products listed under each category. */
export const groupProductsByCategory = (products: IProduct[]): Map<number, IProduct[]> => {
	const productsByCategory = new Map<number, IProduct[]>();
	for (const product of products) {
		for (const categoryID of product.CategoryIDs ?? []) {
			const categoryProducts = productsByCategory.get(categoryID) ?? [];
			categoryProducts.push(product);
			productsByCategory.set(categoryID, categoryProducts);
		}
	}
	return productsByCategory;
};
