import { describe, expect, it } from 'bun:test';
import type { ComponentAST } from '$ecommerce/renderer/renderer-types';
import { scopeCustomCss, nextGlobalId } from './scope-custom-css';

// Sequential allocator starting at 1, mirrors nextGlobalId on an empty page.
const alloc = () => {
	let n = 0;
	return () => ++n;
};

describe('scopeCustomCss', () => {
	it('renames a used class in both CSS and AST', () => {
		const ast: ComponentAST[] = [{ tagName: 'div', css: 'flex hero-grad' }];
		const css = scopeCustomCss('.hero-grad { background: linear-gradient(to br, red, blue); }', ast, alloc());
		expect(css).toBe('.x1{background: linear-gradient(to br, red, blue);}');
		expect(ast[0].css).toBe('flex x1'); // utility untouched, custom class renamed
	});

	it('keeps the same id for a class reused across selectors', () => {
		const ast: ComponentAST[] = [{ tagName: 'div', css: 'card' }];
		const css = scopeCustomCss('.card { color: red } .card:hover { color: blue }', ast, alloc());
		expect(css).toBe('.x1{color: red}.x1:hover{color: blue}');
		expect(ast[0].css).toBe('x1');
	});

	it('drops global and unused selectors', () => {
		const ast: ComponentAST[] = [{ tagName: 'div', css: 'box' }];
		const css = scopeCustomCss(
			'body { margin: 0 } * { box-sizing: border-box } .unused { color: red } .box { padding: 4px }',
			ast,
			alloc()
		);
		expect(css).toBe('.x1{padding: 4px}'); // only .box survives
	});

	it('scopes inside @media', () => {
		const ast: ComponentAST[] = [{ tagName: 'div', css: 'panel' }];
		const css = scopeCustomCss('@media (min-width: 700px) { .panel { display: flex } }', ast, alloc());
		expect(css).toBe('@media (min-width: 700px){.x1{display: flex}}');
		expect(ast[0].css).toBe('x1');
	});

	it('namespaces @keyframes and its animation references', () => {
		const ast: ComponentAST[] = [{ tagName: 'div', css: 'spinner' }];
		const css = scopeCustomCss(
			'.spinner { animation: spin 2s linear infinite } @keyframes spin { from { transform: rotate(0) } to { transform: rotate(360deg) } }',
			ast,
			alloc()
		);
		// .spinner -> x1, keyframe spin -> x2, ref rewritten in the animation value.
		expect(css).toContain('.x1{animation: x2 2s linear infinite}');
		expect(css).toContain('@keyframes x2{');
		expect(css).not.toContain('spin');
	});

	it('returns empty and leaves AST untouched when there is no css', () => {
		const ast: ComponentAST[] = [{ tagName: 'div', css: 'flex' }];
		expect(scopeCustomCss('', ast, alloc())).toBe('');
		expect(ast[0].css).toBe('flex');
	});
});

describe('nextGlobalId', () => {
	it('continues past the max existing x{n} across sections', () => {
		const sections = [
			{ id: 'a', Ast: [{ tagName: 'div', css: 'x3 flex' }] },
			{ id: 'b', CustomCss: '.x7{color:red}' },
		] as any;
		const next = nextGlobalId(sections);
		expect(next()).toBe(8);
		expect(next()).toBe(9);
	});

	it('starts at 1 on an empty page', () => {
		const next = nextGlobalId([]);
		expect(next()).toBe(1);
	});
});
