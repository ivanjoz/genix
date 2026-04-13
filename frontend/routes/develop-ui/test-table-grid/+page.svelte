<script lang="ts">
  import Page from '$domain/Page.svelte';
  import TableGrid from '$components/vTable/TableGrid.svelte';
  import type { TableGridColumn } from '$components/vTable/tableGridTypes';

  interface DemoTableGridRow {
    id: string;
    fullName: string;
    email: string;
    city: string;
    age: number;
    status: 'active' | 'blocked' | 'pending';
  }

  // Generate many rows to validate virtualization behavior and row-selection flow.
  const createDemoRows = (totalRows: number): DemoTableGridRow[] => {
    const generatedRows: DemoTableGridRow[] = [];
    const statuses: DemoTableGridRow['status'][] = ['active', 'blocked', 'pending'];

    for (let currentIndex = 0; currentIndex < totalRows; currentIndex += 1) {
      const rowId = `USR-${(currentIndex + 1).toString().padStart(5, '0')}`;
      generatedRows.push({
        id: rowId,
        fullName: `User ${currentIndex + 1}`,
        email: `user${currentIndex + 1}@demo.com`,
        city: currentIndex % 2 === 0 ? 'Lima' : 'Cusco',
        age: 18 + (currentIndex % 55),
        status: statuses[currentIndex % statuses.length],
      });
    }

    return generatedRows;
  };

  let demoRows = $state<DemoTableGridRow[]>(createDemoRows(15000));
  let selectedRecord = $state<DemoTableGridRow | undefined>(undefined);
  let selectedRowId = $state<string | number | undefined>(undefined);

  const columnDefinitions: TableGridColumn<DemoTableGridRow>[] = [
    {
      id: 'id',
      header: 'ID',
      width: '120px',
      mobile: {
        order: 1,
        labelTop: 'Codigo',
        css: 'col-span-full',
      },
      getValue: (rowRecord) => rowRecord.id,
    },
    {
      id: 'fullName',
      header: 'Name',
      width: 'minmax(190px, 1.2fr)',
      mobile: {
        order: 0,
        labelTop: 'Cliente',
        css: 'col-span-full',
      },
      getValue: (rowRecord) => rowRecord.fullName,
    },
    {
      id: 'email',
      header: 'Email',
      width: 'minmax(260px, 1.6fr)',
      mobile: {
        order: 2,
        labelTop: 'Correo',
        css: 'col-span-full',
      },
      getValue: (rowRecord) => rowRecord.email,
    },
    {
      id: 'city',
      header: 'City',
      width: '140px',
      mobile: {
        order: 3,
        labelLeft: 'Ciudad:',
      },
      getValue: (rowRecord) => rowRecord.city,
    },
    {
      id: 'age',
      header: 'Age',
      width: '90px',
      align: 'right',
      mobile: {
        order: 4,
        labelLeft: 'Edad:',
      },
      getValue: (rowRecord) => rowRecord.age,
    },
    {
      id: 'status',
      header: 'Status',
      width: '140px',
      useCellRenderer: true,
      mobile: {
        order: 5,
        labelTop: 'Estado',
      },
      getValue: (rowRecord) => rowRecord.status,
    },
    {
      id: 'actions',
      header: 'Actions',
      width: '150px',
      useCellRenderer: true,
      align: 'center',
      mobile: {
        order: 6,
        labelTop: 'Acciones',
      },
      getValue: () => '',
    },
  ];

  const resolveRowId = (rowRecord: DemoTableGridRow) => rowRecord.id;

  const selectRow = (rowRecord: DemoTableGridRow) => {
    selectedRecord = rowRecord;
    selectedRowId = rowRecord.id;
  };

  const removeFirstRow = () => {
    demoRows = demoRows.slice(1);
  };
</script>

<Page title="TableGrid Demo">
  <div class="mb-10 flex flex-wrap items-center gap-8">
    <button class="bx-red" onclick={removeFirstRow}>Remove First Row</button>
    <div class="text-[13px] text-slate-600">
      Records: <strong>{demoRows.length.toLocaleString()}</strong>
    </div>
    {#if selectedRecord}
      <div class="text-[13px] text-slate-700">
        Selected: <strong>{selectedRecord.fullName}</strong> ({selectedRecord.id})
      </div>
    {/if}
  </div>

  <TableGrid
    columns={columnDefinitions}
    data={demoRows}
    height="calc(100vh - 210px)"
    rowHeight={38}
    selectedRecord={selectedRecord}
    selectedRowId={selectedRowId}
    getRowId={resolveRowId}
    onRowClick={selectRow}
  >
    {#snippet cellRenderer(rowRecord, columnDefinition)}
      {#if columnDefinition.id === 'status'}
        <span class="status-pill status-{rowRecord.status}">{rowRecord.status}</span>
      {:else if columnDefinition.id === 'actions'}
        <button class="bx-blue py-4 px-6 text-[11px]"
          onclick={(eventInfo) => {
            eventInfo.stopPropagation();
            alert(`Open profile for ${rowRecord.fullName}`);
          }}
        >
          Open
        </button>
      {/if}
    {/snippet}
  </TableGrid>
</Page>

<style>
  .status-pill {
    border-radius: 999px;
    padding: 2px 8px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
  }

  .status-active {
    color: #166534;
    background: #dcfce7;
  }

  .status-blocked {
    color: #991b1b;
    background: #fee2e2;
  }

  .status-pending {
    color: #92400e;
    background: #fef3c7;
  }
</style>
