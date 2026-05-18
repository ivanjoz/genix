// Product delta service: fetches product/brand/category deltas used by ProductSearch incremental updates.
import { GetHandler } from "$libs/http.svelte";
import type { ISharedListRecord } from "$services/negocio/listas-compartidas.svelte";
import type { IProduct } from "$services/services/productos.svelte";

interface ProductDeltaResult {
	productos: IProduct[];
	marcasCategorias: ISharedListRecord[];
}

export class ProductosDeltaService extends GetHandler<IProduct> {
	route = "p-productos-index-delta";
	useCache = { min: 1, ver: 1 };
	inferRemoveFromStatus = true;
	prependOnSave = true;
	productos: IProduct[] = [];
	categorias: ISharedListRecord[] = [];
	marcas: ISharedListRecord[] = [];

	makeName(record: Partial<IProduct>) {
		return record.Nombre || "";
	}

	handler(result: ProductDeltaResult): void {
		// Rebuild local slices from the payload so product-search can consume a compact taxonomy snapshot.
		const marcasIDs = new Set<number>();
		const categoriasIDs = new Set<number>();

		if ((result.productos || []).length > 0) {
			this.productos = [];
			this.categorias = [];
			this.marcas = [];
		}

		for (const producto of result.productos || []) {
			if (producto.MarcaID) {
				marcasIDs.add(producto.MarcaID);
			}
			for (const categoryID of producto.CategoriasIDs || []) {
				categoriasIDs.add(categoryID);
			}
			this.productos.push(producto);
		}

		for (const sharedRow of result.marcasCategorias || []) {
			if (marcasIDs.has(sharedRow.ID)) {
				this.marcas.push(sharedRow);
			}
			if (categoriasIDs.has(sharedRow.ID)) {
				this.categorias.push(sharedRow);
			}
		}
	}

	constructor(init: boolean = false) {
		super();
		if (init) {
			this.fetch();
		}
	}
}
