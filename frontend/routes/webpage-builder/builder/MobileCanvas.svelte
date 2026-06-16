<script lang="ts">
import { editorStore } from '../stores/editor.svelte';
import BuilderSectionRender from './BuilderSectionRender.svelte';

  interface Props {
    // Palette CSS custom properties, forwarded reactively from the parent so a
    // palette change in the Config tab reflects in the mobile preview.
    paletteStyles: string;
  }

  let { paletteStyles }: Props = $props();
</script>

<!-- Root mounted into the preview iframe's <body>. It reads the same shared
     editorStore as the desktop canvas, so click-to-select drives the same
     selection state and the parent editor panel reacts unchanged. Limited view:
     each section renders in 'selectOnly' mode (no drag wiring). -->
<div class="mobile-canvas">
  {#each editorStore.sections as section, idx (section.id)}
    <BuilderSectionRender
      {section}
      index={idx}
      {paletteStyles}
      interaction="selectOnly"
    />
  {/each}

  {#if editorStore.sections.length === 0}
    <div class="empty-canvas">
      <p>Your store is empty</p>
    </div>
  {/if}
</div>

<style>
  .mobile-canvas {
    min-height: 100vh;
    background: #f8fafc;
  }

  .empty-canvas {
    padding: 80px 24px;
    text-align: center;
    color: #94a3b8;
    font-size: 16px;
  }
</style>
