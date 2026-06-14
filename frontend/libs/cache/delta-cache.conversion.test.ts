import { describe, expect, it } from 'bun:test';
import { applyCacheConversions, unpackWordBigrams } from './delta-cache.conversion';

const toBase64 = (bytes: number[]): string => {
	return btoa(String.fromCharCode(...bytes));
};

describe('uint8_packed cache conversion', () => {
	it('unpacks word units with zero only between words', () => {
		// Two words with lengths 2 and 1: header fields are 01 and 00.
		const result = unpackWordBigrams(toBase64([2, 0b01000000, 10, 11, 12]));
		expect([...result]).toEqual([10, 11, 0, 12]);
	});

	it('converts configured response fields to Uint8Array', () => {
		const response = {
			images: [{ ID: 36, Bigrams: toBase64([1, 0, 255]) }],
			categories: [{ ID: 1, Name: 'electronics-tech' }],
		};
		applyCacheConversions(response, { Bigrams: 'uint8_packed' }, 'image-assets');
		expect(response.images[0].Bigrams).toBeInstanceOf(Uint8Array);
		expect([...(response.images[0].Bigrams as unknown as Uint8Array)]).toEqual([255]);
		expect(response.categories[0].Name).toBe('electronics-tech');
	});

	it('rejects malformed packed lengths before persistence', () => {
		expect(() => unpackWordBigrams(toBase64([2, 0b01000000, 10]))).toThrow();
	});
});
