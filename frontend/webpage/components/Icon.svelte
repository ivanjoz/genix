<!--
  Icon — renders an Iconify icon stored inline in the section AST.

  Icons picked in the builder (TextBlockEditor) are saved as AST nodes
  `{ tagName: 'Icon', props: { svg, vb } }`, where `svg` is the sprite id and `vb` the
  viewBox. The body itself is deduplicated into `SectionData.Svgs` and emitted once as a
  `<symbol>` (IconSprite); each icon references it via `<use href="#id">`. (Legacy nodes
  / the editor chip may instead pass `body` — raw SVG inner markup — rendered inline.)
  We render the SVG inline rather than
  via a Tailwind `icon-[set--name]` class because the picker chooses icons at
  runtime, while the Tailwind plugin only emits classes for icons seen statically
  at build time. Inlining also makes each saved page self-contained — no runtime
  dependency on the Iconify JSON or API, so it renders in the storefront prerender.

  Size is 1em so the icon scales with the surrounding font-size (the editor's
  Font-size tool controls it). `mdi` bodies use `currentColor`, so they inherit
  the text color (the Text-color tool tints them); `emojione` / `flat-color-icons`
  carry their own fills and render multicolor as-is.

  `{@html body}` is safe: bodies come from the vendored Iconify packages, not user input.
-->
<script lang="ts">
  interface Props {
    /** Sprite id `icon--<set>-<name>`; referenced via `<use href="#id">` against the
     * section's IconSprite. When set, the body is NOT inlined (deduped in SectionData.Svgs). */
    svg?: string;
    /** SVG inner markup (paths). Fallback when there is no sprite id (legacy inline nodes,
     * or the editor chip rendering standalone). */
    body?: string;
    /** viewBox, e.g. "0 0 24 24". */
    vb?: string;
    css?: string;
    style?: string;
  }

  let { svg = '', body = '', vb = '0 0 24 24', css = '', style = '' }: Props = $props();
</script>

<svg
  class={css}
  {style}
  viewBox={vb}
  width="1em"
  height="1em"
  xmlns="http://www.w3.org/2000/svg"
  aria-hidden="true"
  focusable="false"
>
  {#if svg}<use href="#{svg}" />{:else}{@html body}{/if}
</svg>

<style>
  svg {
    display: inline-block;
    vertical-align: -0.125em;
  }
</style>
