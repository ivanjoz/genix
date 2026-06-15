import { describe, expect, it } from 'bun:test';
import { parseHTML, TEXT_TAG } from './parse-html';
import { compileStyleAttributes } from './style-attributes';
import { coerceValue, coerceProps } from './coerce';
import { componentSchemas } from './component-schemas';

describe('compileStyleAttributes', () => {
	it('maps palette integers to color tokens', () => {
		const { style, consumed } = compileStyleAttributes({ 'background-color': '9' });
		expect(style).toBe('background-color: var(--color-9);');
		expect(consumed.has('background-color')).toBe(true);
	});

	it('passes through explicit colors', () => {
		expect(compileStyleAttributes({ color: '#fff' }).style).toBe('color: #fff;');
	});

	it('ignores non-color attributes (handled by Tailwind)', () => {
		const { style, consumed } = compileStyleAttributes({ 'padding-y': '80', width: '50%', href: '/x' });
		expect(style).toBe('');
		expect(consumed.size).toBe(0);
	});
});

describe('coerceValue', () => {
	it('coerces number[]', () => {
		expect(coerceValue('[1, 2, 3]', 'number[]')).toEqual([1, 2, 3]);
		expect(coerceValue('1,2,3', 'number[]')).toEqual([1, 2, 3]);
	});
	it('coerces number', () => {
		expect(coerceValue('8', 'number')).toBe(8);
	});
	it('coerces boolean', () => {
		expect(coerceValue('', 'boolean')).toBe(true);
		expect(coerceValue('false', 'boolean')).toBe(false);
	});
});

describe('coerceProps', () => {
	it('renames and types via schema, passes unknown through', () => {
		const props = coerceProps(
			{ categoryID: '13', rows: '4', extra: 'raw' },
			componentSchemas.ProductsByCategory,
		);
		expect(props).toEqual({ categoryID: 13, rows: 4, extra: 'raw' });
	});
	it('applies defaults', () => {
		const props = coerceProps({ categoryID: '5' }, componentSchemas.ProductsByCategory);
		expect(props.rows).toBe(3);
	});
});

describe('parseHTML', () => {
	it('parses native tags with class and style attributes', () => {
		const [node] = parseHTML('<section background-color="2" class="mx-auto">Hi</section>');
		expect(node.tagName).toBe('section');
		expect(node.css).toBe('mx-auto');
		expect(node.style).toBe('background-color: var(--color-2);');
		expect(node.text).toBe('Hi');
	});

	it('preserves custom component case and coerces props', () => {
		const [node] = parseHTML('<ProductsByCategory categoryID="13" rows="2" />');
		expect(node.tagName).toBe('ProductsByCategory');
		expect(node.props).toEqual({ categoryID: 13, rows: 2 });
	});

	it('keeps native passthrough attributes', () => {
		const [node] = parseHTML('<a href="/shop">Shop</a>');
		expect(node.attributes).toEqual({ href: '/shop' });
		expect(node.text).toBe('Shop');
	});

	it('builds nested children', () => {
		const [node] = parseHTML('<div><h2>Title</h2><ProductCard productoID="5" /></div>');
		expect(node.children?.length).toBe(2);
		expect(node.children?.[0].tagName).toBe('h2');
		expect(node.children?.[0].text).toBe('Title');
		expect(node.children?.[1].tagName).toBe('ProductCard');
		expect(node.children?.[1].props).toEqual({ productoID: 5 });
	});

	it('represents mixed content with text nodes', () => {
		const [node] = parseHTML('<p>Hello <strong>world</strong></p>');
		expect(node.children?.[0].tagName).toBe(TEXT_TAG);
		expect(node.children?.[0].text).toBe('Hello');
		expect(node.children?.[1].tagName).toBe('strong');
	});

	it('returns multiple top-level nodes', () => {
		const nodes = parseHTML('<div>a</div><div>b</div>');
		expect(nodes.length).toBe(2);
	});
});
