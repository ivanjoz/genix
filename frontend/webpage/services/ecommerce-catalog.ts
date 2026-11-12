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

/** A catalog record in the shape the store's components read (the image as {n, d}). */
export const toStoreProduct = (record: ICatalogProductRecord): IProduct => ({
	ID: record.ID,
	Name: record.Name,
	CategoryIDs: record.CategoryIDs ?? [],
	BrandID: record.BrandID,
	Price: record.Price,
	FinalPrice: record.FinalPrice,
	Image: { n: record.Image > 0 ? String(record.Image) : '', d: '' },
	Stock: record.Stock ?? [],
	ss: record.ss,
});

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
