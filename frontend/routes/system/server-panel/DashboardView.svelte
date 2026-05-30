<script lang="ts">
  import { browser } from '$app/environment';
  import TableStream from '$components/vTable/TableStream.svelte';
  import type { ITableColumn } from '$components/vTable/types';
  import { Env } from '$core/env';
  import { getToken } from '$core/security';
  import { SSEClient, type SSEClientEvent } from '$libs/sse-client';
  import { onDestroy, onMount } from 'svelte';

  type ConsoleLineLevel = 'info' | 'warning' | 'error' | 'success';

  interface ConsoleLine {
    id: number;
    level: ConsoleLineLevel;
    text: string;
    at: Date;
  }

  interface MetricsTableRow {
    id: number;
    at: Date;
    cpuPercent: string;
    backendMemoryUsed: string;
    memoryPercent: string;
    diskPercent: string;
    connections: number;
    rxPerSecond: string;
    txPerSecond: string;
  }

  let dashboardTableRef = $state<any>(null);
  let consoleLogs = $state<ConsoleLine[]>([]);
  let streamConnected = $state(false);
  let streamError = $state('');
  let reconnectAttempts = $state(0);
  let lastEventTimestamp = $state<Date | null>(null);
  let staticMemoryTotal = $state('-');
  let staticDiskTotal = $state('-');

  let reconnectTimer: NodeJS.Timeout | null = null;
  let metricsSSEClient: SSEClient<any> | null = null;
  let shouldReconnect = true;
  let lineSequence = 0;
  let metricsSequence = 0;

  const maxConsoleLines = 500;
  const maxMetricsRows = 600;

  const connectionStatusText = $derived.by(() => {
    if (streamConnected) return 'Connected';
    if (streamError) return 'Disconnected';
    return 'Connecting...';
  });

  const dashboardMetricsColumns: ITableColumn<MetricsTableRow>[] = [
    { id: 'time', header: 'Time', getValue: (rowRecord) => rowRecord.at.toLocaleTimeString() },
    { id: 'cpu', header: 'CPU', getValue: (rowRecord) => rowRecord.cpuPercent },
    { id: 'backend_memory_used', header: 'Backend MEM Used', getValue: (rowRecord) => rowRecord.backendMemoryUsed },
    { id: 'memory_percent', header: 'MEM %', getValue: (rowRecord) => rowRecord.memoryPercent },
    { id: 'disk_percent', header: 'DISK %', getValue: (rowRecord) => rowRecord.diskPercent },
    { id: 'connections', header: 'CONN', getValue: (rowRecord) => rowRecord.connections },
    { id: 'rx_per_second', header: 'RX/s', getValue: (rowRecord) => rowRecord.rxPerSecond },
    { id: 'tx_per_second', header: 'TX/s', getValue: (rowRecord) => rowRecord.txPerSecond }
  ];

  // Converts raw bytes to compact units for table readability.
  const formatBytes = (valueInBytes: number) => {
    if (!Number.isFinite(valueInBytes) || valueInBytes < 0) return '0 B';
    if (valueInBytes < 1024) return `${valueInBytes} B`;
    if (valueInBytes < 1024 * 1024) return `${(valueInBytes / 1024).toFixed(1)} KB`;
    if (valueInBytes < 1024 * 1024 * 1024) return `${(valueInBytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(valueInBytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  };

  // Adds one message line and caps history to keep the console memory-bounded.
  const appendConsoleLine = (text: string, level: ConsoleLineLevel = 'info') => {
    lineSequence++;
    const nextLine: ConsoleLine = { id: lineSequence, level, text, at: new Date() };
    consoleLogs = [...consoleLogs, nextLine].slice(-maxConsoleLines);
  };

  // Maps one metrics event to a row and appends it at the top.
  const prependMetricsRow = (payload: any, level: ConsoleLineLevel) => {
    const metrics = payload?.metrics || {};

    metricsSequence++;

    const cpuPercentValue = Number(metrics?.cpu?.percent_used || 0);
    const backendMemoryUsedBytes = Number(metrics?.backend_process?.rss_bytes || 0);
    const memoryTotalBytes = Number(metrics?.memory?.total_bytes || 0);
    const memoryPercentValue = Number(metrics?.memory?.percent_used || 0);
    const diskTotalBytes = Number(metrics?.disk?.total_bytes || 0);
    const diskPercentValue = Number(metrics?.disk?.percent_used || 0);

    if (memoryTotalBytes > 0) {
      staticMemoryTotal = formatBytes(memoryTotalBytes);
    }
    if (diskTotalBytes > 0) {
      staticDiskTotal = formatBytes(diskTotalBytes);
    }

    const nextRow: MetricsTableRow = {
      id: metricsSequence,
      at: new Date(),
      cpuPercent: `${cpuPercentValue.toFixed(1)}%`,
      backendMemoryUsed: formatBytes(backendMemoryUsedBytes),
      memoryPercent: `${memoryPercentValue.toFixed(1)}%`,
      diskPercent: `${diskPercentValue.toFixed(1)}%`,
      connections: Number(metrics?.connections?.http_active || 0),
      rxPerSecond: `${formatBytes(Number(metrics?.bandwidth?.rx_bytes_per_sec || 0))}/s`,
      txPerSecond: `${formatBytes(Number(metrics?.bandwidth?.tx_bytes_per_sec || 0))}/s`
    };

    dashboardTableRef?.appendTop(nextRow);

    if (level === 'warning') {
      const warnings = Array.isArray(payload?.warnings) ? payload.warnings : [];
      for (const warningMessage of warnings) {
        appendConsoleLine(`warning: ${warningMessage}`, 'warning');
      }
    }
  };

  // Routes normalized SSE events to dashboard reducers.
  const handleMetricsStreamEvent = (sseEvent: SSEClientEvent<any>) => {
    const eventName = sseEvent.eventName;
    const parsedData = sseEvent.payload;
    lastEventTimestamp = sseEvent.receivedAt;

    if (eventName === 'connected') {
      appendConsoleLine(
        `stream connected | interval_ms=${parsedData?.interval_ms || 1000} | mount=${parsedData?.mount_path || '/'} | iface=${parsedData?.interface_name || 'auto'}`,
        'success'
      );
      return;
    }

    if (eventName === 'warning') {
      prependMetricsRow(parsedData, 'warning');
      return;
    }

    if (eventName === 'metrics') {
      prependMetricsRow(parsedData, 'info');
      return;
    }

    appendConsoleLine(`[${eventName}] ${typeof parsedData === 'string' ? parsedData : JSON.stringify(parsedData)}`, 'info');
  };

  const stopStream = (allowFutureReconnect = false) => {
    shouldReconnect = allowFutureReconnect;
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    if (metricsSSEClient) {
      metricsSSEClient.disconnect();
      metricsSSEClient = null;
      appendConsoleLine('stream stopped', 'info');
    }
    streamConnected = false;
  };

  const scheduleReconnect = () => {
    if (reconnectTimer) return;
    reconnectAttempts++;
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null;
      void startStream();
    }, 1500);
  };

  const startStream = async () => {
    if (!browser) return;

    stopStream(false);
    shouldReconnect = true;
    streamError = '';
    streamConnected = false;

    const userToken = getToken(true);
    if (!userToken) {
      streamError = 'No se encontró un token válido de sesión.';
      appendConsoleLine(streamError, 'error');
      return;
    }

    const route = Env.makeRoute('system-metrics-stream?interval_ms=1000');
    metricsSSEClient = new SSEClient({
      endpointUrl: route,
      requestHeaders: {
        Authorization: `Bearer ${userToken}`
      },
      onOpen: () => {
        streamConnected = true;
        streamError = '';
        reconnectAttempts = 0;
      },
      onComment: (sseComment) => {
        appendConsoleLine(`[keepalive] ${sseComment.commentText}`, 'info');
      },
      onMessage: handleMetricsStreamEvent,
      onError: (sseError) => {
        streamConnected = false;
        streamError = String(sseError?.message || sseError || 'error al consumir stream');
        appendConsoleLine(streamError, 'error');
      }
    });

    try {
      appendConsoleLine(`connecting to ${route}`, 'info');
      await metricsSSEClient.connect();
      streamConnected = false;
      if (shouldReconnect) {
        if (!streamError) {
          appendConsoleLine('stream closed by server', 'warning');
        }
        scheduleReconnect();
      }
    } catch (streamErrorValue: any) {
      if (!shouldReconnect) return;
      streamConnected = false;
      streamError = String(streamErrorValue?.message || streamErrorValue || 'error al consumir stream');
      appendConsoleLine(streamError, 'error');
      scheduleReconnect();
    }
  };

  const clearDashboardPanels = () => {
    consoleLogs = [];
    dashboardTableRef?.clearRecords();
  };

  onMount(() => {
    appendConsoleLine('Server dashboard initialized', 'success');
    void startStream();
  });

  onDestroy(() => {
    stopStream(false);
  });
</script>

<div class="flex flex-col gap-12">
  <div class="flex flex-wrap items-center justify-between gap-12">
    <div class="flex flex-wrap items-center gap-10 text-[13px] text-slate-700">
      <span class={["inline-block h-10 w-10 rounded-full", streamConnected ? "bg-green-600" : "bg-red-600"].join(' ')}></span>
      <span class="font-bold text-slate-900">{connectionStatusText}</span>
      <span class="text-slate-600">reconnects: {reconnectAttempts}</span>
      <span class="text-slate-600">last event: {lastEventTimestamp ? lastEventTimestamp.toLocaleTimeString() : '-'}</span>
      <span class="rounded-full border border-slate-300 bg-slate-50 px-10 py-2 text-[12px] font-semibold text-slate-900">MEM total: {staticMemoryTotal}</span>
      <span class="rounded-full border border-slate-300 bg-slate-50 px-10 py-2 text-[12px] font-semibold text-slate-900">DISK total: {staticDiskTotal}</span>
    </div>

    <div class="flex gap-8">
      <button class="cursor-pointer rounded-lg border border-slate-300 bg-slate-50 px-10 py-6 text-[12px] text-slate-900 hover:bg-slate-200" onclick={clearDashboardPanels}>Clear</button>
      <button class="cursor-pointer rounded-lg border border-slate-300 bg-slate-50 px-10 py-6 text-[12px] text-slate-900 hover:bg-slate-200" onclick={() => startStream()}>Reconnect</button>
    </div>
  </div>

  {#if streamError}
    <div class="rounded-lg border border-red-200 bg-red-100 px-10 py-8 text-[12px] text-red-900">{streamError}</div>
  {/if}

  <TableStream
    bind:this={dashboardTableRef}
    columns={dashboardMetricsColumns}
    tableCss="text-sm text-slate-900"
    maxRecords={maxMetricsRows}
    maxHeight="430px"
    emptyMessage="Waiting for metrics..."
  />

  <div class="overflow-hidden rounded-[10px] border border-slate-900">
    <div class="border-b border-slate-900 bg-slate-900 px-12 py-9 text-[12px] font-bold uppercase tracking-[0.08em] text-slate-300">System Messages</div>
    <div class="h-220 overflow-y-auto bg-slate-950 px-10 py-10 font-mono text-[12px] leading-[1.45] text-slate-300">
      {#if consoleLogs.length === 0}
        <div class="text-slate-500">No messages yet...</div>
      {:else}
        {#each consoleLogs as logLine (logLine.id)}
          <div class="mb-2 flex gap-8 whitespace-pre-wrap break-words">
            <span class="shrink-0 text-slate-500">[{logLine.at.toLocaleTimeString()}]</span>
            <span class={[
              logLine.level === 'success' ? 'text-green-300' : '',
              logLine.level === 'warning' ? 'text-amber-200' : '',
              logLine.level === 'error' ? 'text-red-300' : '',
              !['success', 'warning', 'error'].includes(logLine.level) ? 'text-slate-300' : ''
            ].join(' ')}>{logLine.text}</span>
          </div>
        {/each}
      {/if}
    </div>
  </div>
</div>
