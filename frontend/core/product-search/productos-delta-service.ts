// Product ecommerce data service. A thin GetHandler subclass: the bulk catalog is bootstrapped
// from a CDN snapshot file (products-c<companyID>.db, ≤30 min stale) on the first sync and then
// kept current by the standard delta cache — so records and per-table watermarks persist in
// IndexedDB across reloads, and the full product list is never pulled from the API.
//
// The endpoint returns three tables (productos, marcas, categorias), each with its own watermark;
// `keysIDs` keys each by `ID`, and `fileSchema` describes the columns of each ">>>" section so the
// cache can parse the snapshot into the same multi-table shape the API returns.
import { GetHandler } from "$libs/http.svelte";
import { Env } from "$core/env";
import type { ISharedListRecord } from "$services/negocio/listas-compartidas.svelte";
import type { IProduct } from "$services/services/productos.svelte";

// Column order MUST match what the backend writes (no header row). Field names map straight to
// record fields, so naming the watermark/status columns "upd"/"ss" feeds the cache directly.
const PRODUCT_ECOMMERCE_FILE_SCHEMA = {
	productos: ["ID:N", "Name:T", "CategoryIDs:AN", "BrandID:N", "Price:N", "FinalPrice:N", "Image:T", "upd:N", "ss:N"],
	marcas: ["ID:N", "Name:T", "upd:N", "ss:N"],
	categorias: ["ID:N", "Name:T", "upd:N", "ss:N"]
};

export class ProductEcommerceDataService extends GetHandler {
	route = "p-productos-ecommerce";
	useCache = { min: 30, ver: 1 };
	keysIDs = { productos: "ID", marcas: "ID", categorias: "ID" };
	fileSchema = PRODUCT_ECOMMERCE_FILE_SCHEMA;

	// Active records published to ProductSearch for indexing.
	productos: IProduct[] = [];
	marcas: ISharedListRecord[] = [];
	categorias: ISharedListRecord[] = [];

	constructor() {
		super();
		this.fileRoute = Env.makeCDNRoute("live", `products-c${Env.getCompanyID()}.db`);
	}

	// handler receives the merged multi-table cache content; we drop inactive (ss=0) rows here
	// since the snapshot retains them as eviction tombstones.
	handler(result: { productos?: IProduct[]; marcas?: ISharedListRecord[]; categorias?: ISharedListRecord[] }): void {
		this.productos = (result?.productos ?? []).filter((record) => (record.ss ?? 0) > 0);
		// Normalize the image into the {n} shape ProductCard expects. The snapshot file carries a flat
		// image name string (Image), while the server delta carries the full Images array.
		for (const product of this.productos) {
			const rawImage = product.Image as unknown;
			const imageName = (typeof rawImage === "string" ? rawImage : product.Image?.n) || product.Images?.[0]?.n || "";
			product.Image = { n: imageName, d: "" };
		}
		this.marcas = (result?.marcas ?? []).filter((record) => (record.ss ?? 0) > 0);
		this.categorias = (result?.categorias ?? []).filter((record) => (record.ss ?? 0) > 0);
	}

	// load delegates to the delta cache: first sync seeds from the .db file then deltas; later
	// syncs read IndexedDB and only fetch records changed since the stored watermarks.
	async load(): Promise<void> {
		await this.fetchOnline();
	}
}
