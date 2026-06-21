import type { ComponentAST } from '$ecommerce/renderer/renderer-types';

// The agent introduces a NEW color (one not in the palette) as a Tailwind arbitrary
// value with a hex, e.g. `bg-[#aabbcc]`, `text-[#abc]`, `hover:border-[#112233]`.
// absorbColors funnels those into the page palette: every such hex is added to the
// palette (deduped) and the class is rewritten to reference it as `[var(--color-N)]`,
// so a single palette is the source of truth for all colors and the swatch grid grows.
// Existing palette colors the agent reuses arrive as `color="N"` attributes (already
// compiled to var(--color-N) at parse time) and need no handling here.

// Matches the hex inside a Tailwind arbitrary-value bracket. Any utility prefix is
// fine — an arbitrary `[#hex]` value is always a color.
const ARBITRARY_HEX_RE = /\[#([0-9a-fA-F]{3,8})\]/g;

// canon normalizes a color to a comparable canonical form: lowercase, '#'-prefixed,
// shorthand expanded (#abc -> #aabbcc, #abcd -> #aabbccdd) so the same color written
// different ways maps to one palette entry.
function canon(color: string): string {
  let h = color.trim().toLowerCase().replace(/^#/, '');
  if (h.length === 3 || h.length === 4) h = [...h].map((c) => c + c).join('');
  return '#' + h;
}

// indexFor returns the 1-based palette index of a hex, appending it when new.
function indexFor(colors: string[], rawHex: string): number {
  const c = canon(rawHex);
  let i = colors.findIndex((col) => canon(col) === c);
  if (i < 0) {
    colors.push(c);
    i = colors.length - 1;
  }
  return i + 1; // var(--color-N) is 1-based
}

// absorbColors walks the section AST in place, rewriting arbitrary hex color classes
// to palette references and growing `colors` (the live editor palette array) as it
// discovers new ones.
export function absorbColors(nodes: ComponentAST[] | undefined, colors: string[]): void {
  const walk = (node: ComponentAST) => {
    if (node.css && node.css.includes('[#')) {
      node.css = node.css.replace(ARBITRARY_HEX_RE, (_m, hex) => `[var(--color-${indexFor(colors, hex)})]`);
    }
    node.children?.forEach(walk);
  };
  nodes?.forEach(walk);
}
