<script lang="ts">
  // The sales tool's charts, then its tables, under the agent's message (agent-answer.ts lays them out).
  import { Chart } from '@genix/ui/charts';
  import {
    formatAgentCell, formatAgentChartValue, inlineBarPercents, planAgentChart,
    type AgentChart, type AgentTable,
  } from './agent-answer';

  interface Props {
    tables?: AgentTable[]
    charts?: AgentChart[]
  }

  const { tables = [], charts = [] }: Props = $props()

  const isNumericFormat = (format: string) => format === 'number' || format === 'money'
</script>

{#each charts as chart, chartIndex (chartIndex)}
  {#each planAgentChart(chart) as plan, planIndex (planIndex)}
    <div class="mt-8 bg-surface rounded-lg p-6">
      <Chart type={plan.type} series={plan.series} labels={plan.labels} height={planIndex === 0 ? 200 : 140}        formatValue={(value) => formatAgentChartValue(value, plan.format)}
      />
    </div>
  {/each}
{/each}

{#each tables as table, tableIndex (tableIndex)}
  {@const rowCount = table.Columns[0]?.Values.length || 0}
  {@const barColumn = table.Columns.find((column) => column.InlineBar)}
  {@const barPercents = barColumn ? inlineBarPercents(barColumn) : []}
  <div class="mt-8 max-h-[360px] overflow-auto rounded-lg bg-surface">
    <table class="w-full text-sm border-collapse">
      <thead class="sticky top-0 bg-surface">
        <tr>
          {#each table.Columns as column, columnIndex (columnIndex)}
            <th class="px-6 py-4 font-semibold whitespace-nowrap {isNumericFormat(column.Format) ? 'text-right' : 'text-left'}">{column.Label}</th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#each { length: rowCount } as _, rowIndex (rowIndex)}
          <tr class="_agent-answer-row">
            {#each table.Columns as column, columnIndex (columnIndex)}
              <td class="px-6 pt-4 {barColumn ? 'pb-0' : 'pb-4'} {isNumericFormat(column.Format) ? 'text-right font-mono whitespace-nowrap' : ''}">
                {formatAgentCell(column.Values[rowIndex], column.Format)}
              </td>
            {/each}
          </tr>
          {#if barColumn}
            <tr>
              <td colspan={table.Columns.length} class="px-6 pb-4">
                <div class="h-3 rounded-full bg-accent-solid opacity-60" style:width="{barPercents[rowIndex]}%"></div>
              </td>
            </tr>
          {/if}
        {/each}
      </tbody>
    </table>
  </div>
{/each}

<style>
  ._agent-answer-row:hover {
    background-color: var(--surface-muted);
  }
</style>
