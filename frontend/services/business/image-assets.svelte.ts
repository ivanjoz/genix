import { GetHandler } from '$libs/http.svelte';

export interface IImageAssetSearchRecord {
	ID: number;
	CategoryID: number;
	Bigrams: Uint8Array;
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
	useCache = { min: 60, ver: 3 };
	categories: IImageAssetCategory[] = $state([]);
	categoriesMap: Map<number, IImageAssetCategory> = $state(new Map());
	conversion = { Bigrams: 'uint8_packed' } as const;

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

	getBigrams(record: IImageAssetSearchRecord): Uint8Array {
		return record.Bigrams;
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

	getImageURL(record: IImageAssetSearchRecord): string {
		// Full-resolution variant (the `.s` thumbnail suffix dropped) for use as the live image src.
		const categoryName = this.categoriesMap.get(record.CategoryID)?.Name;
		if (!categoryName) { return ''; }
		return `https://ivanjoz.github.io/genix-assets/images/${encodeURIComponent(categoryName)}/${record.ID}.avif`;
	}

	constructor(init: boolean = false) {
		super();
		if (init) this.fetch();
	}
}
