import { describe, expect, it } from 'bun:test';
// parseHTML is the render-side authoring parser (stays in the webpage app); the
// templates and `editable` helpers are editor-only and live here in the builder.
import { parseHTML } from '$ecommerce/html-ast/parse-html';
import { collectRoleNodes } from './editable';
import { HeroBanner } from '../templates/hero/hero-banner';
import { CategoryShowcase } from '../templates/products/category-showcase';

describe('ecommerce templates parse cleanly', () => {
	it('hero banner: native section with color vars and CTA link', () => {
		const [section] = parseHTML(HeroBanner.html!);
		expect(section.tagName).toBe('section');
		expect(section.style).toBe('background-color: var(--color-9);');
		const cta = section.children?.[0].children?.find((n) => n.tagName === 'a');
		expect(cta?.attributes?.href).toBe('/shop');
		expect(cta?.style).toContain('background-color: var(--color-5)');
	});

	it('hero banner: editable roles collected, role attr not emitted to DOM', () => {
		const ast = parseHTML(HeroBanner.html!);
		const roles = collectRoleNodes(ast);
		expect(roles.map((r) => r.role)).toEqual(['title', 'content', 'button']);
		// data-role must not leak into rendered attributes
		const button = roles.find((r) => r.role === 'button')!.node;
		expect(button.attributes).toEqual({ href: '/shop' });
		expect(button.attributes?.['data-role']).toBeUndefined();
	});

	it('category showcase: custom components with coerced props', () => {
		const [section] = parseHTML(CategoryShowcase.html!);
		const inner = section.children?.[0].children ?? [];
		const desc = inner.find((n) => n.tagName === 'CategoryDescription');
		const grid = inner.find((n) => n.tagName === 'ProductsByCategory');
		expect(desc?.props?.categoryIDs).toEqual([22]);
		expect(grid?.props).toEqual({ categoryID: 22, rows: 2, rowsMobile: 3 });
	});
});
