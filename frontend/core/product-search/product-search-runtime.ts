import { browser } from "$app/environment";
import { ProductSearch } from "./product-search";

let sharedProductSearchInstance: ProductSearch | null = null;
let sharedProductSearchLoadingPromise: Promise<ProductSearch> | null = null;

export const preloadProductSearch = async (): Promise<ProductSearch | null> => {
	// Product index bootstrap requires browser APIs (fetch + IndexedDB), so skip during SSR.
	if (!browser) return null;
	if (sharedProductSearchInstance) return sharedProductSearchInstance;
	if (sharedProductSearchLoadingPromise) return await sharedProductSearchLoadingPromise;

	console.info("[ProductSearchRuntime] preloading product search index");
	sharedProductSearchLoadingPromise = (async () => {
		const productSearchInstance = new ProductSearch();
		await productSearchInstance.readyPromise;
		sharedProductSearchInstance = productSearchInstance;
		console.info("[ProductSearchRuntime] product search index ready", {
			source: productSearchInstance.source,
			productsCount: productSearchInstance.size,
			updatedSunix: productSearchInstance.updated,
		});
		return productSearchInstance;
	})();

	try {
		return await sharedProductSearchLoadingPromise;
	} catch (productSearchLoadError) {
		console.error("[ProductSearchRuntime] preload failed", productSearchLoadError);
		sharedProductSearchLoadingPromise = null;
		throw productSearchLoadError;
	}
};

export const getPreloadedProductSearch = (): ProductSearch | null => {
	return sharedProductSearchInstance;
};
