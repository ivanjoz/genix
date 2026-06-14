import { GetHandler } from '$libs/http.svelte';

export interface IImageAssetSearchRecord {
	ID: number;
	CategoryID: number;
	Bigrams: string;
	upd: number;
}

export interface IImageAssetCategory {
	ID: number;
	Name: string;
	upd: number;
}

export interface IImageAssetsResult {
	images: IImageAssetSearchRecord[];
	categories: IImageAssetCategory[];
}

export class ImageAssetsService extends GetHandler<IImageAssetSearchRecord> {
	route = 'image-assets';
	useCache = { min: 60, ver: 2 };
	categories: IImageAssetCategory[] = $state([]);
	categoriesMap: Map<number, IImageAssetCategory> = $state(new Map());

	handler(result: IImageAssetsResult): void {
		const images = result.images || [];
		const categories = result.categories || [];
		console.debug(`[image-assets] cache loaded; images=${images.length} categories=${categories.length}`);
		this.records = [];
		this.recordsMap = new Map();
		this.addSavedRecords(...images);
		this.categories = categories;
		this.categoriesMap = new Map(categories.map((category) => [category.ID, category]));
	}

	// Decode only when search needs the signature; IndexedDB keeps compact Base64.
	getBigrams(record: IImageAssetSearchRecord): Uint8Array {
		try {
			const binaryBigrams = atob(record.Bigrams);
			return Uint8Array.from(binaryBigrams, (character) => character.charCodeAt(0));
		} catch (decodeError) {
			console.error(`[image-assets] invalid bigrams payload; ID=${record.ID}`, decodeError);
			return new Uint8Array();
		}
	}

	getThumbnailURL(record: IImageAssetSearchRecord): string {
		// Repository thumbnails are grouped by the stable category slug.
		const categoryName = this.categoriesMap.get(record.CategoryID)?.Name;
		if (!categoryName) {
			console.warn(`[image-assets] category not found; imageID=${record.ID} categoryID=${record.CategoryID}`);
			return '';
		}
		return `https://ivanjoz.github.io/genix-assets/images/${encodeURIComponent(categoryName)}/${record.ID}.s.avif`;
	}

	constructor(init: boolean = false) {
		super();
		if (init) this.fetch();
	}
}
