import { editorStore } from './editor.svelte';

/**
 * Custom Micro-Compiler for Tailwind-like utilities.
 * Supports: p-, m-, w-, h- (including x, y, t, r, b, l variants and md:, lg:, xl: prefixes)
 * Unit: 1px constant
 */
class LiveCSSStore {
  css = $state('');
  
  // Mapping of prefixes to CSS properties
  private propertyMap: Record<string, string> = {
    'p': 'padding',
    'px': 'padding-inline',
    'py': 'padding-block',
    'pt': 'padding-top',
    'pr': 'padding-right',
    'pb': 'padding-bottom',
    'pl': 'padding-left',
    'm': 'margin',
    'mx': 'margin-inline',
    'my': 'margin-block',
    'mt': 'margin-top',
    'mr': 'margin-right',
    'mb': 'margin-bottom',
    'ml': 'margin-left',
    'w': 'width',
    'h': 'height',
    'max-w': 'max-width',
    'max-h': 'max-height',
    'min-w': 'min-width',
    'min-h': 'min-height'
  };

  // Screen breakpoints
  private breakpoints: Record<string, string> = {
    'md': '749px',
    'lg': '1139px',
    'xl': '1379px',
    '2xl': '1539px'
  };

  async init() {
    // No initialization needed for the micro-compiler
    console.log('Genix Micro-CSS Compiler initialized');
  }

  update() {
    const classes = new Set<string>();
    
    // 1. Collect all classes from sections
    editorStore.sections.forEach(section => {
      if (section.css) {
        Object.values(section.css).forEach(val => {
          if (typeof val === 'string') {
            val.split(/\s+/).filter(Boolean).forEach(c => classes.add(c));
          }
        });
      }
    });

    if (classes.size === 0) {
      this.css = '';
      return;
    }

    // 2. Parse and generate CSS
    const rules: Record<string, string[]> = {
      base: []
    };
    Object.keys(this.breakpoints).forEach(bp => rules[bp] = []);

    classes.forEach(className => {
      let activeClassName = className;
      let breakpoint: string | null = null;

      // Handle breakpoint prefix (e.g., md:p-10)
      const bpMatch = className.match(/^([a-z0-9]+):(.+)$/);
      if (bpMatch && this.breakpoints[bpMatch[1]]) {
        breakpoint = bpMatch[1];
        activeClassName = bpMatch[2];
      }

      // Parse utility (e.g., p-10 or -m-5)
      const utilMatch = activeClassName.match(/^(-)?([a-z-]+)-(\[.+\]|[0-9.]+|full|auto|screen)$/);
      if (!utilMatch) return;

      const isNegative = !!utilMatch[1];
      const prefix = utilMatch[2];
      let value = utilMatch[3];

      const property = this.propertyMap[prefix];
      if (!property) return;

      // Convert value
      if (value.startsWith('[') && value.endsWith(']')) {
        value = value.slice(1, -1); // Arbitrary value [200px]
      } else if (value === 'full') {
        value = '100%';
      } else if (value === 'screen') {
        value = prefix.startsWith('w') ? '100vw' : '100vh';
      } else if (value === 'auto') {
        value = 'auto';
      } else if (!isNaN(Number(value))) {
        value = `${Number(value)}px`; // Our 1px = 1 unit mapping
      } else {
        return; // Unknown value type
      }

      if (isNegative && value !== '0px' && value !== 'auto') {
        value = `calc(${value} * -1)`;
      }

      // Escape class name for CSS selector
      const escapedClass = className.replace(/:/g, '\\:').replace(/\[/g, '\\[').replace(/\]/g, '\\\]');
      const rule = `.${escapedClass} { ${property}: ${value} !important; }`;

      if (breakpoint) {
        rules[breakpoint].push(rule);
      } else {
        rules.base.push(rule);
      }
    });

    // 3. Assemble final CSS string
    let finalCss = "/* Genix Micro-Compiler Output */\n" + rules.base.join('\n');
    
    Object.entries(this.breakpoints).forEach(([bp, width]) => {
      if (rules[bp].length > 0) {
        finalCss += `\n@media (min-width: ${width}) {\n  ${rules[bp].join('\n  ')}\n}\n`;
      }
    });

    this.css = finalCss;
  }
}

export const liveCSS = new LiveCSSStore();
