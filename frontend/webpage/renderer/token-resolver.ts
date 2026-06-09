import type { ComponentVariable, ColorPalette } from './renderer-types';

/**
 * Resolves variables and color tokens in a CSS string.
 * Example: "w-[__v1__] bg-[__COLOR:1__]" -> "w-[500px] bg-[#0f172a]"
 */
export function resolveTokens(
    css: string | undefined,
    variables: ComponentVariable[] = [],
    values: Record<string, string> = {},
    palette?: ColorPalette
): string {
    if (!css) return '';

    try {
        let resolved = css;

        // 1. Resolve Component Variables (__v1__, __v2__, etc.)
        if (variables && variables.length > 0) {
            variables.forEach(v => {
                if (!v.key) return;
                const value = values[v.key] ?? v.defaultValue;
                if (value !== undefined) {
                    // Escape key for regex
                    const regex = new RegExp(v.key.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g');
                    resolved = resolved.replace(regex, value);
                }
            });
        }

        // 2. Resolve Color Tokens (__COLOR:1__ to __COLOR:10__)
        if (palette && palette.colors) {
            resolved = resolved.replace(/__COLOR:(\d+)__/g, (match, index) => {
                const idx = parseInt(index) - 1;
                const color = palette.colors[idx];
                if (color) return color;
                console.warn(`Color token ${match} not found in palette ${palette.name}`);
                return 'transparent';
            });
        }

        return resolved;
    } catch (error) {
        console.error('Error resolving tokens:', error);
        return css;
    }
}

/**
 * Generates a style string containing CSS variables from a color palette.
 * Useful for the root element of a section.
 */
export function generatePaletteStyles(palette: ColorPalette | undefined): string {
    if (!palette || !palette.colors) return '';
    
    return palette.colors
        .map((color, i) => `--color-${i + 1}: ${color};`)
        .join(' ');
}
