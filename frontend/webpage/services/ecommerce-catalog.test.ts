import { describe, expect, it } from 'bun:test';
import {
	CATALOG_SCHEMA_VERSION, groupProductsByCategory, isUsableSavedCatalog, mergeCatalogDelta, savedCatalogFromFile,
	toStoreProduct, type ICatalogDelta, type ICatalogProductRecord,
} from './ecommerce-catalog.ts';

const makeProductRecord = (ID: number, overrides: Partial<ICatalogProductRecord> = {}): ICatalogProductRecord => ({
	ID, Name: `Producto ${ID}`, CategoryIDs: [1], BrandID: 2, Price: 1000, FinalPrice: 900, Image: 0,
	Stock: [{ PresentationID: 0, Units: 5 }], ss: 1, ...overrides,
});

const makeDelta = (overrides: Partial<ICatalogDelta> = {}): ICatalogDelta => ({
	UseFile: false, FileCreated: 0, Generation: 3, NextSince: 500,
	Productos: null, Marcas: null, Categorias: null, ...overrides,
});

const fileCatalog = savedCatalogFromFile({
	Meta: { Created: 104, Updated: 100, Generation: 3 },
	Productos: [makeProductRecord(1), makeProductRecord(2)],
	Marcas: [{ ID: 2, Name: 'Marca', ss: 1 }],
	Categorias: null,
});

describe('savedCatalogFromFile', () => {
	it('starts the cursor at the file Updated, and reads a null list as empty', () => {
		expect(fileCatalog.since).toBe(100);
		expect(fileCatalog.generation).toBe(3);
		expect(fileCatalog.categorias).toEqual([]);
		expect(isUsableSavedCatalog(fileCatalog)).toBe(true);
	});

	it('discards a saved catalog of another schema version', () => {
		expect(isUsableSavedCatalog({ ...fileCatalog, schemaVersion: CATALOG_SCHEMA_VERSION + 1 })).toBe(false);
		expect(isUsableSavedCatalog(undefined)).toBe(false);
	});
});

describe('mergeCatalogDelta', () => {
	it('replaces by ID, adds new products, removes ss 0 and moves the cursor', () => {
		const merged = mergeCatalogDelta(fileCatalog, makeDelta({
			Productos: [
				makeProductRecord(1, { Stock: [{ PresentationID: 0, Units: 1 }] }),
				{ ...makeProductRecord(2), ss: 0 },
				makeProductRecord(3),
			],
			Marcas: [{ ID: 2, Name: '', ss: 0 }],
		}));
		expect(merged.productos.map((record) => record.ID)).toEqual([1, 3]);
		expect(merged.productos[0].Stock).toEqual([{ PresentationID: 0, Units: 1 }]);
		expect(merged.marcas).toEqual([]);
		expect(merged.since).toBe(500);
	});

	it('keeps the lists untouched by an empty delta', () => {
		const merged = mergeCatalogDelta(fileCatalog, makeDelta());
		expect(merged.productos).toBe(fileCatalog.productos);
		expect(merged.since).toBe(500);
	});

	it('ignores a removal of a product the browser never had', () => {
		const merged = mergeCatalogDelta(fileCatalog, makeDelta({ Productos: [{ ...makeProductRecord(9), ss: 0 }] }));
		expect(merged.productos.map((record) => record.ID)).toEqual([1, 2]);
	});
});

describe('toStoreProduct and groupProductsByCategory', () => {
	it('turns the image ID into the {n, d} image, empty without one', () => {
		expect(toStoreProduct(makeProductRecord(1, { Image: 44 })).Image).toEqual({ n: '44', d: '' });
		expect(toStoreProduct(makeProductRecord(1)).Image).toEqual({ n: '', d: '' });
		expect(toStoreProduct(makeProductRecord(1, { Stock: null, CategoryIDs: null })).Stock).toEqual([]);
	});

	it('lists a product under each of its categories', () => {
		const products = [makeProductRecord(1, { CategoryIDs: [1, 2] }), makeProductRecord(2, { CategoryIDs: [2] })].map(toStoreProduct);
		const productsByCategory = groupProductsByCategory(products);
		expect(productsByCategory.get(1)?.map((product) => product.ID)).toEqual([1]);
		expect(productsByCategory.get(2)?.map((product) => product.ID)).toEqual([1, 2]);
	});
});
