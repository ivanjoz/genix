<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    src: string;
    /** Controls the composition and space for text (Geometric shapes or soft gradients) */
    layout?: 'fade-left' | 'fade-right' | 'fade-up' | 'fade-down' | 'slash-left' | 'slash-right' | 'curve-left' | 'curve-right' | 'curve-convex-left' | 'curve-convex-right' | string;
    /** Controls the visual filter/treatment applied to the image */
    effect?: 'duotone' | 'duotone-matrix' | 'glass' | 'vignette' | 'overlay' | string;
    color?: string;
    color2?: string;
    intensity?: number;
    blur?: number;
    css?: string;
    imgCss?: string;
    aspectRatio?: string;
    children?: Snippet;
  }

  let {
    src,
    layout = '',
    effect = '',
    color = '#ffffff',
    color2 = '#000000',
    intensity = 1,
    blur = 0,
    css = '',
    imgCss = '',
    aspectRatio = 'auto',
    children
  }: Props = $props();

  const filterId = $derived(`duotone-${Math.random().toString(36).substring(2, 9)}`);
  const curveRightId = $derived(`curve-right-${Math.random().toString(36).substring(2, 9)}`);
  const curveLeftId = $derived(`curve-left-${Math.random().toString(36).substring(2, 9)}`);
  const curveConvexRightId = $derived(`curve-convex-right-${Math.random().toString(36).substring(2, 9)}`);
  const curveConvexLeftId = $derived(`curve-convex-left-${Math.random().toString(36).substring(2, 9)}`);

  const hexToNorm = (hex: string, def = 0) => {
    try {
      const h = hex.replace('#', '');
      const fullHex = h.length === 3 ? h.split('').map(c => c + c).join('') : h;
      return {
        r: (parseInt(fullHex.substring(0, 2), 16) / 255).toFixed(2),
        g: (parseInt(fullHex.substring(2, 4), 16) / 255).toFixed(2),
        b: (parseInt(fullHex.substring(4, 6), 16) / 255).toFixed(2)
      };
    } catch (e) {
      return { r: def, g: def, b: def };
    }
  };

  const c1 = $derived(hexToNorm(color, 0));
  const c2 = $derived(hexToNorm(color2, 1));

  /** Styles related to the Layout (gradients for fade effects) */
  const layoutOverlayStyle = $derived.by(() => {
    const op = intensity;
    const blurVal = blur > 0 ? `backdrop-filter: blur(${blur}px);` : '';
    
    switch (layout) {
      case 'fade-right':
        return `background: linear-gradient(to right, ${color} 0%, transparent 100%); opacity: ${op}; ${blurVal}`;
      case 'fade-left':
        return `background: linear-gradient(to left, ${color} 0%, transparent 100%); opacity: ${op}; ${blurVal}`;
      case 'fade-up':
        return `background: linear-gradient(to top, ${color} 0%, transparent 100%); opacity: ${op}; ${blurVal}`;
      case 'fade-down':
        return `background: linear-gradient(to bottom, ${color} 0%, transparent 100%); opacity: ${op}; ${blurVal}`;
      default:
        return blurVal ? `opacity: 1; ${blurVal}` : '';
    }
  });

  /** Styles related to Visual Effects (Glass, Vignette, etc.) */
  const effectOverlayStyle = $derived.by(() => {
    const op = intensity;
    switch (effect) {
      case 'overlay':
        return `background-color: ${color}; opacity: ${op * 0.5};`;
      case 'glass':
        return `backdrop-filter: blur(${blur || 8 * op}px); background-color: ${color}${Math.round(op * 50).toString(16).padStart(2, '0')};`;
      case 'vignette':
        return `background: radial-gradient(circle, transparent 40%, ${color} 100%); opacity: ${op};`;
      default:
        return '';
    }
  });

  const imgContainerStyle = $derived.by(() => {
    switch (layout) {
      case 'slash-right':
        return `clip-path: polygon(35% 0, 100% 0, 100% 100%, 55% 100%);`;
      case 'slash-left':
        return `clip-path: polygon(0 0, 65% 0, 45% 100%, 0 100%);`;
      case 'curve-right':
        return `clip-path: url(#${curveRightId});`;
      case 'curve-left':
        return `clip-path: url(#${curveLeftId});`;
      case 'curve-convex-right':
        return `clip-path: url(#${curveConvexRightId});`;
      case 'curve-convex-left':
        return `clip-path: url(#${curveConvexLeftId});`;
      default:
        return '';
    }
  });

  const imgStyle = $derived.by(() => {
    let styles = [];
    if (effect === 'duotone') {
      styles.push(`filter: grayscale(100%) contrast(1.2);`);
    }
    if (effect === 'duotone-matrix') {
      styles.push(`filter: url(#${filterId});`);
    }
    return styles.join(' ');
  });

  const isLeveraged = $derived(layout.includes('slash') || layout.includes('curve'));
</script>

<div class={["relative overflow-hidden", css].join(" ")} 
     style="aspect-ratio: {aspectRatio}; isolation: isolate; {isLeveraged ? `background-color: ${color};` : ''}">
  
  <svg style="position: absolute; width: 0; height: 0; pointer-events: none;">
    <defs>
      {#if effect === 'duotone-matrix'}
        <filter id={filterId}>
          <feColorMatrix type="matrix" values="0.2126 0.7152 0.0722 0 0
                                               0.2126 0.7152 0.0722 0 0
                                               0.2126 0.7152 0.0722 0 0
                                               0 0 0 1 0" result="gray" />
          <feComponentTransfer in="gray" result="contrast">
            <feFuncR type="linear" slope="1.2" intercept="-0.1" />
            <feFuncG type="linear" slope="1.2" intercept="-0.1" />
            <feFuncB type="linear" slope="1.2" intercept="-0.1" />
          </feComponentTransfer>
          <feComponentTransfer in="contrast" color-interpolation-filters="sRGB">
            <feFuncR type="table" tableValues="{c1.r} {c2.r}" />
            <feFuncG type="table" tableValues="{c1.g} {c2.g}" />
            <feFuncB type="table" tableValues="{c1.b} {c2.b}" />
          </feComponentTransfer>
        </filter>
      {/if}
      <clipPath id={curveRightId} clipPathUnits="objectBoundingBox">
        <path d="M 1,0 L 0.65,0 C 0.65,0.4 0.45,0.8 0.15,1 L 1,1 Z" />
      </clipPath>
      <clipPath id={curveLeftId} clipPathUnits="objectBoundingBox">
        <path d="M 0,0 L 0.35,0 C 0.35,0.4 0.55,0.8 0.85,1 L 0,1 Z" />
      </clipPath>
      <clipPath id={curveConvexRightId} clipPathUnits="objectBoundingBox">
        <path d="M 1,0 L 0.65,0 C 0.45,0.3 0.35,0.7 0.4,1 L 1,1 Z" />
      </clipPath>
      <clipPath id={curveConvexLeftId} clipPathUnits="objectBoundingBox">
        <path d="M 0,0 L 0.35,0 C 0.55,0.3 0.65,0.7 0.6,1 L 0,1 Z" />
      </clipPath>
    </defs>
  </svg>

  <!-- Background Image Layer -->
  <div class="absolute inset-0 z-0 pointer-events-none" style={imgContainerStyle}>
    {#if isLeveraged}
      <div class="flex h-full w-full">
        {#if layout.includes('right')}
          <div class="h-full w-[35%] overflow-hidden">
            <img {src} alt="" class="h-full w-full object-cover scale-x-[-1]" style="object-position: left; {imgStyle}" />
          </div>
          <div class="h-full w-[65%] overflow-hidden">
            <img {src} alt="" class="h-full w-full object-cover" style="object-position: right; {imgStyle}" />
          </div>
        {:else}
          <div class="h-full w-[65%] overflow-hidden">
            <img {src} alt="" class="h-full w-full object-cover" style="object-position: left; {imgStyle}" />
          </div>
          <div class="h-full w-[35%] overflow-hidden">
            <img {src} alt="" class="h-full w-full object-cover scale-x-[-1]" style="object-position: right; {imgStyle}" />
          </div>
        {/if}
      </div>
    {:else}
      <img 
        {src} 
        alt="" 
        class={["w-full h-full object-cover", imgCss].join(" ")}
        style={imgStyle}
      />
    {/if}
    
    {#if effect === 'duotone'}
      <div class="absolute inset-0" style="background-color: {color}; mix-blend-mode: screen; opacity: {intensity};"></div>
      <div class="absolute inset-0" style="background-color: {color2}; mix-blend-mode: multiply; opacity: {intensity};"></div>
    {/if}
  </div>

  <!-- Effect Overlay Layer (Glass, Vignette, etc.) -->
  {#if effectOverlayStyle}
    <div class="absolute inset-0 z-5 pointer-events-none" style={effectOverlayStyle}></div>
  {/if}

  <!-- Layout Overlay Layer (Fade Gradients) -->
  {#if layoutOverlayStyle}
    <div class="absolute inset-0 z-10 pointer-events-none" 
         style={(imgContainerStyle.includes('url') || imgContainerStyle.includes('polygon') ? imgContainerStyle : '') + layoutOverlayStyle}>
    </div>
  {/if}
  
  <!-- Content Layer -->
  {#if children}
    <div class="relative z-20 w-full h-full">
      {@render children()}
    </div>
  {/if}
</div>

<style>
  .relative {
    min-height: 1px;
  }
</style>
