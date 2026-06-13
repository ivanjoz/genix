<script lang="ts">
import { inMemoryImages, type InMemoryImage } from '$core/inMemoryImages.svelte';
import { tr } from '$core/store.svelte';

// Reactive list of in-flight image processes (SvelteMap → re-renders on change).
const processes = $derived([...inMemoryImages.values()]);

// Human-readable status label per process state.
const statusLabel = (process: InMemoryImage): string => {
  if (process.status === 'converting') { return tr('Converting...|Convirtiendo...'); }
  if (process.status === 'pending') { return tr('Waiting...|En espera...'); }
  if (process.status === 'uploading') {
    return process.progress > 0
      ? `${tr('Saving...|Guardando...')} ${process.progress}%`
      : tr('Saving...|Guardando...');
  }
  return tr('Error|Error');
};
</script>

<div class="w-280 md:w-320 max-h-360 overflow-y-auto">
  <div class="h5 px-8 pt-6 pb-8 c-text-soft">{tr('Processes|Procesos')}</div>

  {#if processes.length === 0}
    <div class="px-8 pb-10 fs14 c-text-soft">{tr('No active processes.|Sin procesos activos.')}</div>
  {/if}

  {#each processes as process (process.name)}
    <div class="flex items-center gap-8 px-8 py-6 process-row">
      <!-- Optimistic preview thumbnail -->
      <img class="h-36 w-36 rounded-6 object-cover shrink-0" src={process.base64} alt="" />
      <div class="flex-1 min-w-0">
        <div class="fs14 ff-bold truncate">{process.description || process.name}</div>
        <div class="fs12 {process.status === 'error' ? 'c-red' : 'c-text-soft'}">
          {statusLabel(process)}
        </div>
      </div>
      {#if process.status === 'error' && process.upload}
        <button class="bnr-1 _retry shrink-0"
          aria-label={tr('Retry|Reintentar')}
          onclick={() => process.upload?.()}
        >
          <i class="icon-cw"></i>
        </button>
      {/if}
    </div>
  {/each}
</div>

<style>
  .process-row:not(:last-child) {
    border-bottom: 1px solid #0000001a;
  }
  :global(body.dark) .process-row:not(:last-child) {
    border-bottom-color: #ffffff1a;
  }
  ._retry {
    background-color: #1277f5;
    color: white;
  }
</style>
