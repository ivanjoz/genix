<script lang="ts">
  import { browser } from '$app/environment';
  import TableStream from '$components/vTable/TableStream.svelte';
  import type { ITableColumn } from '$components/vTable/types';
  import { Env } from '$core/env';
  import { getToken } from '$core/security';
  import { onDestroy, onMount } from 'svelte';

  interface MemoryPackagesTableRow {
    id: number;
    packageName: string;
    inuseBytesText: string;
    inuseMiBText: string;
    inusePercentText: string;
  }

  let memoryTableRef = $state<any>(null);
  let memoryLastSyncTimestamp = $state<Date | null>(null);
  let memoryRefreshError = $state('');
  let memoryWarnings = $state<string[]>([]);
  let memoryRefreshInProgress = $state(false);

  let memoryHeapInuse = $state('-');
  let memoryHeapObjects = $state('-');
  let memoryBackendRSS = $state('-');

  let memoryRefreshTimer: NodeJS.Timeout | null = null;

  const maxMemoryRows = 20;
  const memoryRefreshIntervalMilliseconds = 2000;

  const memoryPackagesColumns: ITableColumn<MemoryPackagesTableRow>[] = [
    { id: 'rank', header: '#', getValue: (rowRecord) => rowRecord.id },
    { id: 'package_name', header: 'Package', getValue: (rowRecord) => rowRecord.packageName },
    { id: 'inuse_bytes', header: 'In-Use (bytes)', getValue: (rowRecord) => rowRecord.inuseBytesText },
    { id: 'inuse_mib', header: 'In-Use (MiB)', getValue: (rowRecord) => rowRecord.inuseMiBText },
    { id: 'inuse_percent', header: 'Heap %', getValue: (rowRecord) => rowRecord.inusePercentText }
  ];

  // Converts raw bytes to compact units for table readability.
  const formatBytes = (valueInBytes: number) => {
    if (!Number.isFinite(valueInBytes) || valueInBytes < 0) return '0 B';
    if (valueInBytes < 1024) return `${valueInBytes} B`;
    if (valueInBytes < 1024 * 1024) return `${(valueInBytes / 1024).toFixed(1)} KB`;
    if (valueInBytes < 1024 * 1024 * 1024) return `${(valueInBytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(valueInBytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  };

  const refreshMemoryPackages = async () => {
    if (!browser || memoryRefreshInProgress) return;

    const userToken = getToken(true);
    if (!userToken) {
      memoryRefreshError = 'No se encontró un token válido de sesión.';
      return;
    }

    memoryRefreshInProgress = true;
    memoryRefreshError = '';

    try {
      const route = Env.makeRoute('system-memory-packages?limit=20');
      const response = await fetch(route, {
        method: 'GET',
        headers: {
          Accept: 'application/json',
          Authorization: `Bearer ${userToken}`
        }
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`HTTP ${response.status}: ${errorText || 'error al consultar memoria por paquetes'}`);
      }

      const responsePayload = await response.json();
      const reportPayload = responsePayload?.report || {};
      const topPackages = Array.isArray(reportPayload?.top_packages) ? reportPayload.top_packages : [];

      memoryHeapInuse = formatBytes(Number(reportPayload?.go_heap_inuse_bytes || 0));
      memoryHeapObjects = Number(reportPayload?.go_heap_objects || 0).toLocaleString();
      memoryBackendRSS = formatBytes(Number(reportPayload?.backend_rss_bytes || 0));

      const mappedMemoryRows: MemoryPackagesTableRow[] = topPackages.map((packageEntry: any, index: number) => ({
        id: index + 1,
        packageName: String(packageEntry?.package_name || 'unknown'),
        inuseBytesText: formatBytes(Number(packageEntry?.inuse_bytes || 0)),
        inuseMiBText: `${Number(packageEntry?.inuse_mib || 0).toFixed(2)} MiB`,
        inusePercentText: `${Number(packageEntry?.percent || 0).toFixed(2)}%`
      }));

      memoryTableRef?.replaceRecords(mappedMemoryRows);
      memoryWarnings = Array.isArray(responsePayload?.warnings) ? responsePayload.warnings : [];
      memoryLastSyncTimestamp = new Date();
    } catch (memoryError: any) {
      memoryRefreshError = String(memoryError?.message || memoryError || 'error al consultar memoria');
    } finally {
      memoryRefreshInProgress = false;
    }
  };

  const startMemoryRefresh = () => {
    stopMemoryRefresh();
    void refreshMemoryPackages();
    memoryRefreshTimer = setInterval(() => {
      void refreshMemoryPackages();
    }, memoryRefreshIntervalMilliseconds);
  };

  const stopMemoryRefresh = () => {
    if (!memoryRefreshTimer) return;
    clearInterval(memoryRefreshTimer);
    memoryRefreshTimer = null;
  };

  const clearMemoryPanels = () => {
    memoryTableRef?.clearRecords();
    memoryWarnings = [];
  };

  onMount(() => {
    startMemoryRefresh();
  });

  onDestroy(() => {
    stopMemoryRefresh();
  });
</script>

<div class="flex flex-col gap-12">
  <div class="flex flex-wrap items-center justify-between gap-12">
    <div class="flex flex-wrap items-center gap-10 text-[13px] text-slate-700">
      <span class={["inline-block h-10 w-10 rounded-full", memoryRefreshError ? "bg-red-600" : "bg-green-600"].join(' ')}></span>
      <span class="font-bold text-slate-900">{memoryRefreshError ? 'Disconnected' : 'Connected'}</span>
      <span class="text-slate-600">updated: {memoryLastSyncTimestamp ? memoryLastSyncTimestamp.toLocaleTimeString() : '-'}</span>
      <span class="rounded-full border border-slate-300 bg-slate-50 px-10 py-2 text-[12px] font-semibold text-slate-900">Heap in-use: {memoryHeapInuse}</span>
      <span class="rounded-full border border-slate-300 bg-slate-50 px-10 py-2 text-[12px] font-semibold text-slate-900">Heap objects: {memoryHeapObjects}</span>
      <span class="rounded-full border border-slate-300 bg-slate-50 px-10 py-2 text-[12px] font-semibold text-slate-900">Backend RSS: {memoryBackendRSS}</span>
    </div>

    <div class="flex gap-8">
      <button class="cursor-pointer rounded-lg border border-slate-300 bg-slate-50 px-10 py-6 text-[12px] text-slate-900 hover:bg-slate-200" onclick={clearMemoryPanels}>Clear</button>
      <button class="cursor-pointer rounded-lg border border-slate-300 bg-slate-50 px-10 py-6 text-[12px] text-slate-900 hover:bg-slate-200" onclick={() => refreshMemoryPackages()}>Refresh</button>
    </div>
  </div>

  {#if memoryRefreshError}
    <div class="rounded-lg border border-red-200 bg-red-100 px-10 py-8 text-[12px] text-red-900">{memoryRefreshError}</div>
  {/if}

  <TableStream
    bind:this={memoryTableRef}
    columns={memoryPackagesColumns}
    tableCss="text-sm text-slate-900"
    maxRecords={maxMemoryRows}
    maxHeight="430px"
    emptyMessage={memoryRefreshInProgress ? 'Refreshing memory report...' : 'Waiting for memory data...'}
  />

  <div class="overflow-hidden rounded-[10px] border border-slate-900">
    <div class="border-b border-slate-900 bg-slate-900 px-12 py-9 text-[12px] font-bold uppercase tracking-[0.08em] text-slate-300">Memory Warnings</div>
    <div class="h-220 overflow-y-auto bg-slate-950 px-10 py-10 font-mono text-[12px] leading-[1.45] text-slate-300">
      {#if memoryWarnings.length === 0}
        <div class="text-slate-500">No warnings.</div>
      {:else}
        {#each memoryWarnings as warningText, warningIndex (`${warningText}-${warningIndex}`)}
          <div class="mb-2 flex gap-8 whitespace-pre-wrap break-words">
            <span class="shrink-0 text-slate-500">[warn]</span>
            <span class="text-amber-200">{warningText}</span>
          </div>
        {/each}
      {/if}
    </div>
  </div>
</div>
