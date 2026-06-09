<script lang="ts">
import type { ColorPalette } from '$ecommerce/renderer/renderer-types';

  interface Props {
    palette?: ColorPalette;
  }

  let { palette }: Props = $props();
</script>

<div class="config-tab">
  {#if palette}
    <div class="field-group">
      <div class="group-header">
        <span class="group-icon">🎨</span>
        <span class="group-label">Color Palette</span>
      </div>
      
      <div class="palette-info">
        <div class="palette-name">{palette.name}</div>
        <div class="palette-colors">
          {#each palette.colors as color, idx}
            <div 
              class="color-swatch" 
              style="background-color: {color}"
              title="COLOR:{idx + 1} - {color}"
            >
              <span class="color-index">{idx + 1}</span>
            </div>
          {/each}
        </div>
      </div>
    </div>
  {/if}

  <div class="field-group">
    <div class="group-header">
      <span class="group-icon">⚙️</span>
      <span class="group-label">Global Configuration</span>
    </div>
    <div class="placeholder-msg">
      Global store configurations like spacing, typography, and theme mode will appear here.
    </div>
  </div>
</div>

<style>
  .config-tab {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .field-group {
    background: #1e293b;
    border-radius: 12px;
    overflow: hidden;
  }

  .group-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 16px;
    background: #334155;
    font-size: 13px;
    font-weight: 600;
    color: #cbd5e1;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .palette-info {
    padding: 16px;
  }

  .palette-name {
    font-size: 14px;
    color: #f1f5f9;
    margin-bottom: 12px;
    font-weight: 500;
  }

  .palette-colors {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(36px, 1fr));
    gap: 6px;
  }

  .color-swatch {
    aspect-ratio: 1;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: transform 0.15s ease;
    position: relative;
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .color-swatch:hover {
    transform: scale(1.1);
    z-index: 1;
  }

  .color-index {
    font-size: 10px;
    font-weight: 600;
    color: white;
    text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
  }

  .placeholder-msg {
    padding: 20px;
    color: #64748b;
    font-size: 13px;
    text-align: center;
    line-height: 1.5;
  }
</style>
