import { describe, expect, it } from 'bun:test';
// parseHTML is the render-side authoring parser (stays in the webpage app); the
// templates and `editable` helpers are editor-only and live here in the builder.
import { parseHTML } from '$ecommerce/html-ast/parse-html';
import { collectRoleNodes, groupSiblings } from './editable';
import { HeroBanner } from '../templates/hero/hero-banner';
import { CategoryShowcase } from '../templates/products/category-showcase';
import { FeatureChecklist } from '../templates/features/feature-checklist';

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

describe('groupSiblings: adjacent text grouping for the WYSIWYG editor', () => {
	const [section] = parseHTML(FeatureChecklist.html!);
	// section > div(grid) > div(text col) + ImageEffect
	const textCol = section.children?.[0].children?.[0]!;
	const ul = textCol.children?.find((n) => n.tagName === 'ul')!;

	it('merges the adjacent <h2> title + <p> content into one 2-line editor', () => {
		const items = groupSiblings(textCol.children!);
		// First item is the merged [h2, p] text group; the <ul> is a separate node item.
		expect(items[0].kind).toBe('lines');
		const first = items[0] as { kind: 'lines'; lines: typeof textCol.children };
		expect(first.lines!.map((l) => l.tagName)).toEqual(['h2', 'p']);
		expect(items[1]).toEqual({ kind: 'node', node: ul });
	});

	it('a <p> still merges with its <h2> after Enter materializes it into spans', () => {
		// Simulate the editor materializing the <p> content into two inline spans.
		const p = textCol.children!.find((n) => n.tagName === 'p')!;
		p.text = undefined;
		p.children = [
			{ tagName: 'span', text: 'One simple price' },
			{ tagName: 'span', text: 'no surprises' },
		];
		const items = groupSiblings(textCol.children!);
		// Eligibility is by tag, so [h2, p] stays one group even though p now has children.
		expect(items[0].kind).toBe('lines');
		expect((items[0] as { lines: typeof textCol.children }).lines!.map((l) => l.tagName)).toEqual(['h2', 'p']);
	});

	it('each <li> becomes its own one-line editor (icon span + text span as fragments)', () => {
		const items = groupSiblings(ul.children!);
		expect(items).toHaveLength(4);
		expect(items.every((it) => it.kind === 'lines')).toBe(true);
		const firstLi = (items[0] as { lines: NonNullable<typeof ul.children> }).lines[0];
		expect(firstLi.tagName).toBe('li');
		// the <li>'s two spans are its fragments (the ✅ icon + the text)
		expect(firstLi.children?.map((c) => c.text)).toEqual(['✅', 'Free worldwide shipping & returns']);
	});
});
