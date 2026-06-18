<!--
  IconPicker — the searchable icon grid shown inside the TextBlockEditor toolbar's
  options row. Lazy-loads one Iconify set (mdi / emojione / flat-color-icons),
  filters by name, and renders a virtualized grid. Picking an icon hands its SVG
  body + viewBox back to the editor, which splices an `Icon` fragment into the line.

  Preview cells render at 18px; the inserted icon itself renders at 1em (Icon.svelte).
-->
<script lang="ts">
  import SvelteVirtualList from '@humanspeak/svelte-virtual-list';
  import { loadIconSets, type IconSetId, type PickerIcon } from './icon-sets';

  interface Props {
    /** One or more sets, shown concatenated in this order (e.g. color icons, then material). */
    sets: IconSetId[];
    onpick: (icon: { id: string; body: string; vb: string }) => void;
  }

  let { sets, onpick }: Props = $props();

  const COLS = 8;

  let all = $state<PickerIcon[]>([]);
  let loading = $state(true);
  let query = $state('');

  $effect(() => {
    loading = true;
    loadIconSets(sets).then((icons) => {
      all = icons;
      loading = false;
    });
  });

  const filtered = $derived.by(() => {
    const q = query.trim().toLowerCase();
    const list = q ? all.filter((i) => i.name.includes(q)) : all;
    // Cap the unfiltered view so the very first paint stays light; search reaches the rest.
    return q ? list : list.slice(0, 600);
  });

  // Chunk into rows for the vertical virtual list (one row = COLS icons).
  const rows = $derived.by(() => {
    const out: PickerIcon[][] = [];
    for (let i = 0; i < filtered.length; i += COLS) out.push(filtered.slice(i, i + COLS));
    return out;
  });
</script>

<div class="icon-picker">
  <input
    class="icon-search"
    type="text"
    placeholder="Buscar…"
    bind:value={query}
    autocomplete="off"
    spellcheck="false"
  />
  {#if loading}
    <div class="icon-status">Cargando…</div>
  {:else if rows.length === 0}
    <div class="icon-status">Sin resultados</div>
  {:else}
    <div class="icon-grid">
      <SvelteVirtualList items={rows}>
        {#snippet renderItem(row)}
          <div class="icon-row">
            {#each row as icon (icon.name + icon.vb)}
              <button
                type="button"
                class="icon-cell"
                title={icon.name}
                onclick={() => onpick({ id: icon.id, body: icon.body, vb: icon.vb })}
              >
                <svg viewBox={icon.vb} width="18" height="18" aria-hidden="true">{@html icon.body}</svg>
              </button>
            {/each}
          </div>
        {/snippet}
      </SvelteVirtualList>
    </div>
  {/if}
</div>

<style>
  .icon-picker {
    display: flex;
    flex-direction: column;
    gap: 6px;
    width: 100%;
    min-width: 220px;
  }
  .icon-search {
    width: 100%;
    box-sizing: border-box;
    padding: 5px 8px;
    font-size: 12px;
    border: 1px solid rgba(255, 255, 255, 0.18);
    border-radius: 6px;
    background: rgba(0, 0, 0, 0.25);
    color: inherit;
    outline: none;
  }
  .icon-search:focus {
    border-color: #3b82f6;
  }
  .icon-status {
    padding: 16px 0;
    text-align: center;
    font-size: 12px;
    opacity: 0.6;
  }
  .icon-grid {
    height: 220px;
  }
  .icon-row {
    display: flex;
    gap: 2px;
  }
  .icon-cell {
    flex: 1 1 0;
    display: flex;
    align-items: center;
    justify-content: center;
    height: 30px;
    padding: 0;
    border: none;
    border-radius: 5px;
    background: transparent;
    color: inherit;
    cursor: pointer;
  }
  .icon-cell:hover {
    background: rgba(255, 255, 255, 0.12);
  }
</style>
