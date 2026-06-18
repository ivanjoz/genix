<!--
  IconSprite — emits a section's deduplicated icon bodies (SectionData.Svgs) as one hidden
  SVG containing a `<symbol id="…">` per entry. Every Icon in the section references its
  symbol via `<use href="#id">`, so each unique body is stored and shipped exactly once.

  Symbols are left WITHOUT their own viewBox: each referencing `<svg viewBox={vb}>` (Icon.svelte)
  supplies the viewBox, so sets with different coordinate systems (mdi 24, emojione 64, fc 48)
  all render correctly from the same sprite.

  `{@html body}` is safe: bodies come from the vendored Iconify packages, not user input.
-->
<script lang="ts">
  interface Props {
    svgs?: Record<string, string>;
  }

  let { svgs }: Props = $props();

  const entries = $derived(Object.entries(svgs ?? {}));
</script>

{#if entries.length}
  <svg
    aria-hidden="true"
    style="position:absolute;width:0;height:0;overflow:hidden"
    xmlns="http://www.w3.org/2000/svg"
  >
    {#each entries as [id, body] (id)}
      <symbol {id}>{@html body}</symbol>
    {/each}
  </svg>
{/if}
