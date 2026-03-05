<script lang="ts">
  import { browser } from '$app/environment';
  import { Env } from '$core/env';
  import { getToken } from '$core/security';
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

  let metricsRows = $state<MetricsTableRow[]>([]);
  let consoleLogs = $state<ConsoleLine[]>([]);
  let streamConnected = $state(false);
  let streamError = $state('');
  let reconnectAttempts = $state(0);
  let lastEventTimestamp = $state<Date | null>(null);

  let staticMemoryTotal = $state('-');
  let staticDiskTotal = $state('-');

  let reconnectTimer: NodeJS.Timeout | null = null;
  let streamAbortController: AbortController | null = null;
  let lineSequence = 0;
  let metricsSequence = 0;

  const maxConsoleLines = 500;
  const maxMetricsRows = 600;

  const connectionStatusText = $derived.by(() => {
    if (streamConnected) return 'Connected';
    if (streamError) return 'Disconnected';
    return 'Connecting...';
  });

  // Adds one message line to the lower console and caps history for stable memory use.
  const appendConsoleLine = (text: string, level: ConsoleLineLevel = 'info') => {
    lineSequence++;
    const nextLine: ConsoleLine = { id: lineSequence, level, text, at: new Date() };
    consoleLogs = [...consoleLogs, nextLine].slice(-maxConsoleLines);
  };

  // Inserts newest metrics row at the top so operators always see latest values first.
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

    metricsRows = [nextRow, ...metricsRows].slice(0, maxMetricsRows);

    if (level === 'warning') {
      const warnings = Array.isArray(payload?.warnings) ? payload.warnings : [];
      for (const warningMessage of warnings) {
        appendConsoleLine(`warning: ${warningMessage}`, 'warning');
      }
    }
  };

  // Converts raw bytes to compact units for table readability.
  const formatBytes = (valueInBytes: number) => {
    if (!Number.isFinite(valueInBytes) || valueInBytes < 0) return '0 B';
    if (valueInBytes < 1024) return `${valueInBytes} B`;
    if (valueInBytes < 1024 * 1024) return `${(valueInBytes / 1024).toFixed(1)} KB`;
    if (valueInBytes < 1024 * 1024 * 1024) return `${(valueInBytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(valueInBytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  };

  // Parses one SSE block and routes metrics to the table, other messages to the console.
  const processSSEBlock = (sseBlock: string) => {
    const blockLines = sseBlock
      .split('\n')
      .map((line) => line.trimEnd())
      .filter((line) => line.length > 0);

    if (blockLines.length === 0) return;

    let eventName = 'message';
    const dataLines: string[] = [];

    for (const line of blockLines) {
      if (line.startsWith(':')) {
        appendConsoleLine(`[keepalive] ${line.substring(1).trim()}`, 'info');
        return;
      }
      if (line.startsWith('event:')) {
        eventName = line.substring(6).trim();
        continue;
      }
      if (line.startsWith('data:')) {
        dataLines.push(line.substring(5).trim());
      }
    }

    if (dataLines.length === 0) return;
    lastEventTimestamp = new Date();

    const rawDataText = dataLines.join('\n');
    let parsedData: any = rawDataText;
    try {
      parsedData = JSON.parse(rawDataText);
    } catch {
      // Keep raw text when payload is not JSON.
    }

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

  // Consumes the authenticated backend SSE stream using fetch + reader.
  const startMetricsConsoleStream = async () => {
    if (!browser) return;

    stopMetricsConsoleStream();
    streamError = '';
    streamConnected = false;

    const userToken = getToken(true);
    if (!userToken) {
      streamError = 'No se encontró un token válido de sesión.';
      appendConsoleLine(streamError, 'error');
      return;
    }

    const route = Env.makeRoute('system-metrics-stream?interval_ms=1000');
    streamAbortController = new AbortController();

    try {
      appendConsoleLine(`connecting to ${route}`, 'info');
      const response = await fetch(route, {
        method: 'GET',
        headers: {
          Accept: 'text/event-stream',
          Authorization: `Bearer ${userToken}`
        },
        signal: streamAbortController.signal
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`HTTP ${response.status}: ${errorText || 'error al conectar stream'}`);
      }

      if (!response.body) {
        throw new Error('La respuesta SSE no contiene body.');
      }

      streamConnected = true;
      streamError = '';
      reconnectAttempts = 0;

      const reader = response.body.getReader();
      const textDecoder = new TextDecoder();
      let pendingBuffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        pendingBuffer += textDecoder.decode(value, { stream: true });

        const blocks = pendingBuffer.split('\n\n');
        pendingBuffer = blocks.pop() || '';

        for (const block of blocks) {
          processSSEBlock(block);
        }
      }

      streamConnected = false;
      appendConsoleLine('stream closed by server', 'warning');
      scheduleReconnect();
    } catch (streamErrorValue: any) {
      if (streamAbortController?.signal.aborted) {
        appendConsoleLine('stream stopped', 'info');
        return;
      }
      streamConnected = false;
      streamError = String(streamErrorValue?.message || streamErrorValue || 'error al consumir stream');
      appendConsoleLine(streamError, 'error');
      scheduleReconnect();
    }
  };

  // Stops stream and pending reconnect timers to avoid duplicated readers.
  const stopMetricsConsoleStream = () => {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    if (streamAbortController) {
      streamAbortController.abort();
      streamAbortController = null;
    }
    streamConnected = false;
  };

  // Reconnects automatically after short delay to keep the panel alive.
  const scheduleReconnect = () => {
    if (reconnectTimer) return;
    reconnectAttempts++;
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null;
      startMetricsConsoleStream();
    }, 1500);
  };

  // Clears both table and console history without interrupting the stream.
  const clearPanels = () => {
    metricsRows = [];
    consoleLogs = [];
  };

  onMount(() => {
    appendConsoleLine('Server log console initialized', 'success');
    startMetricsConsoleStream();
  });

  onDestroy(() => {
    stopMetricsConsoleStream();
  });
</script>

<section class="server-console-root">
  <div class="console-toolbar">
    <div class="toolbar-left">
      <span class="status-dot {streamConnected ? 'is-on' : 'is-off'}"></span>
      <span class="status-text">{connectionStatusText}</span>
      <span class="status-meta">reconnects: {reconnectAttempts}</span>
      <span class="status-meta">last event: {lastEventTimestamp ? lastEventTimestamp.toLocaleTimeString() : '-'}</span>
      <span class="status-chip">MEM total: {staticMemoryTotal}</span>
      <span class="status-chip">DISK total: {staticDiskTotal}</span>
    </div>

    <div class="toolbar-right">
      <button class="btn" onclick={clearPanels}>Clear</button>
      <button class="btn" onclick={() => startMetricsConsoleStream()}>Reconnect</button>
    </div>
  </div>

  {#if streamError}
    <div class="console-error">{streamError}</div>
  {/if}

  <div class="metrics-table-card">
    <div class="metrics-table-scroll">
      <table class="metrics-table">
        <thead>
          <tr>
            <th>Time</th>
            <th>CPU</th>
            <th>Backend MEM Used</th>
            <th>MEM %</th>
            <th>DISK %</th>
            <th>CONN</th>
            <th>RX/s</th>
            <th>TX/s</th>
          </tr>
        </thead>
        <tbody>
          {#if metricsRows.length === 0}
            <tr>
              <td colspan="8" class="metrics-empty">Waiting for metrics...</td>
            </tr>
          {:else}
            {#each metricsRows as row (row.id)}
              <tr>
                <td>{row.at.toLocaleTimeString()}</td>
                <td>{row.cpuPercent}</td>
                <td>{row.backendMemoryUsed}</td>
                <td>{row.memoryPercent}</td>
                <td>{row.diskPercent}</td>
                <td>{row.connections}</td>
                <td>{row.rxPerSecond}</td>
                <td>{row.txPerSecond}</td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>
  </div>

  <div class="console-card">
    <div class="console-card-header">System Messages</div>
    <div class="console-box">
      {#if consoleLogs.length === 0}
        <div class="console-empty">No messages yet...</div>
      {:else}
        {#each consoleLogs as logLine (logLine.id)}
          <div class="console-line level-{logLine.level}">
            <span class="line-time">[{logLine.at.toLocaleTimeString()}]</span>
            <span class="line-text">{logLine.text}</span>
          </div>
        {/each}
      {/if}
    </div>
  </div>
</section>

<style>
  .server-console-root {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 12px;
  }

  .console-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .toolbar-left {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
    color: #334155;
    font-size: 13px;
  }

  .toolbar-right {
    display: flex;
    gap: 8px;
  }

  .btn {
    border: 1px solid #cbd5e1;
    background: #f8fafc;
    color: #0f172a;
    border-radius: 8px;
    padding: 6px 10px;
    font-size: 12px;
    cursor: pointer;
  }

  .btn:hover {
    background: #e2e8f0;
  }

  .status-dot {
    width: 10px;
    height: 10px;
    border-radius: 999px;
    display: inline-block;
  }

  .status-dot.is-on {
    background: #16a34a;
  }

  .status-dot.is-off {
    background: #dc2626;
  }

  .status-text {
    font-weight: 700;
    color: #0f172a;
  }

  .status-meta {
    color: #475569;
  }

  .status-chip {
    color: #0f172a;
    border: 1px solid #cbd5e1;
    border-radius: 999px;
    padding: 2px 10px;
    background: #f8fafc;
    font-size: 12px;
    font-weight: 600;
  }

  .console-error {
    color: #991b1b;
    background: #fee2e2;
    border: 1px solid #fecaca;
    border-radius: 8px;
    padding: 8px 10px;
    font-size: 12px;
  }

  .metrics-table-card {
    border: 1px solid #cbd5e1;
    border-radius: 10px;
    background: #ffffff;
    overflow: hidden;
  }

  .metrics-table-scroll {
    max-height: 430px;
    overflow: auto;
  }

  .metrics-table {
    width: 100%;
    border-collapse: separate;
    border-spacing: 0;
    font-size: 12px;
    color: #0f172a;
  }

  .metrics-table thead th {
    position: sticky;
    top: 0;
    z-index: 2;
    background: #0f172a;
    color: #e2e8f0;
    text-align: left;
    font-weight: 700;
    padding: 10px 8px;
    border-bottom: 1px solid #1e293b;
    white-space: nowrap;
  }

  .metrics-table tbody td {
    padding: 8px;
    border-bottom: 1px solid #e2e8f0;
    white-space: nowrap;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  }

  .metrics-table tbody tr:nth-child(odd) {
    background: #f8fafc;
  }

  .metrics-empty {
    color: #64748b;
    text-align: center;
    padding: 22px 8px !important;
    font-family: inherit !important;
  }

  .console-card {
    border: 1px solid #1e293b;
    border-radius: 10px;
    overflow: hidden;
  }

  .console-card-header {
    background: #0f172a;
    color: #cbd5e1;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    padding: 9px 12px;
    border-bottom: 1px solid #1e293b;
  }

  .console-box {
    background: #0b1220;
    color: #cbd5e1;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
    font-size: 12px;
    line-height: 1.45;
    height: 220px;
    overflow-y: auto;
    padding: 10px;
  }

  .console-empty {
    color: #64748b;
  }

  .console-line {
    display: flex;
    gap: 8px;
    white-space: pre-wrap;
    word-break: break-word;
    margin-bottom: 2px;
  }

  .line-time {
    color: #64748b;
    flex-shrink: 0;
  }

  .line-text {
    color: #cbd5e1;
  }

  .console-line.level-success .line-text {
    color: #86efac;
  }

  .console-line.level-warning .line-text {
    color: #fde68a;
  }

  .console-line.level-error .line-text {
    color: #fca5a5;
  }
</style>
