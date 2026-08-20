<script lang="ts">
  import Button from '$components/buttons/Button.svelte';
  import T from '$components/misc/T.svelte';
  import TableGrid from '$components/vTable/TableGrid.svelte';
  import type { ITableColumn } from '$components/vTable/types';
  import { tr } from '$core/store.svelte';
  import { formatN, formatTime, Notify } from '$libs/helpers';
  import { untrack } from 'svelte';
  import {
    getCompanyCreditBudget,
    mutateCompanyCreditBudget,
    type ICompanyCreditBudgetOperation,
    type ICompanyCreditBudget,
  } from './company-credit-budget';

  let { companyID }: { companyID: number } = $props();

  // `saved*` is the durable figure the row was loaded with: it is both the baseline that decides
  // what the save button sends and, for the remaining row, the base the adder row sums onto.
  type BudgetRowKind = 'daily' | 'increase' | 'remaining';
  type BudgetRow = {
    kind: BudgetRowKind;
    concept: string;
    CPU: number;
    Inference: number;
    savedCPU: number;
    savedInference: number;
  };

  let budget = $state<ICompanyCreditBudget | null>(null);
  let rows = $state<BudgetRow[]>([]);
  let loadedCompanyID = $state(0);
  let isLoading = $state(false);
  let isSaving = $state(false);
  let loadError = $state('');

  const rowOf = (kind: BudgetRowKind): BudgetRow | undefined => rows.find((row) => row.kind === kind);
  const hasChanged = (row: BudgetRow | undefined): boolean => (
    !!row && (row.CPU !== row.savedCPU || row.Inference !== row.savedInference)
  );

  // The adder never travels on its own: the remaining row already carries the total it produced, and
  // "set-current" writes that total as an absolute figure, so sending both would apply it twice.
  const pendingOperations = $derived.by<ICompanyCreditBudgetOperation[]>(() => {
    const operations: ICompanyCreditBudgetOperation[] = [];
    const dailyRow = rowOf('daily');
    const remainingRow = rowOf('remaining');
    if (hasChanged(dailyRow)) {
      operations.push({ Operation: 'set-daily', CPU: dailyRow!.CPU, Inference: dailyRow!.Inference });
    }
    if (hasChanged(remainingRow)) {
      operations.push({ Operation: 'set-current', CPU: remainingRow!.CPU, Inference: remainingRow!.Inference });
    }
    return operations;
  });

  const errorText = (requestError: any): string => String(
    requestError?.errorMessage || requestError?.error || requestError?.message || requestError || 'error'
  );

  const makeRow = (
    kind: BudgetRowKind,
    concept: string,
    cpu: number,
    inference: number,
  ): BudgetRow => ({
    kind, concept, CPU: cpu, Inference: inference, savedCPU: cpu, savedInference: inference,
  });

  const applyBudget = (nextBudget: ICompanyCreditBudget) => {
    budget = nextBudget;
    rows = [
      makeRow('daily', 'Daily allowance|Límite diario', nextBudget.DailyCPU || 0, nextBudget.DailyInference || 0),
      // The adder sits above the total it feeds, so the panel reads top to bottom: add this, get that.
      makeRow('increase', 'Credits (add)|Créditos (Aumentar)', 0, 0),
      makeRow('remaining', 'Remaining credits|Créditos Restantes', nextBudget.CurrentCPU || 0, nextBudget.CurrentInference || 0),
    ];
  };

  const loadBudget = async (targetCompanyID: number) => {
    isLoading = true;
    loadError = '';
    console.debug('[company-credit-budget] load started', { companyID: targetCompanyID });
    try {
      const nextBudget = await getCompanyCreditBudget(targetCompanyID);
      if (companyID === targetCompanyID) applyBudget(nextBudget);
      console.debug('[company-credit-budget] load completed', {
        companyID: targetCompanyID,
        currentMonth: nextBudget.IsCurrentMonth,
      });
    } catch (requestError: any) {
      if (companyID === targetCompanyID) loadError = errorText(requestError);
      console.error('[company-credit-budget] load failed', {
        companyID: targetCompanyID,
        error: errorText(requestError),
      });
    } finally {
      if (companyID === targetCompanyID) isLoading = false;
    }
  };

  const editCell = (row: BudgetRow, field: 'CPU' | 'Inference', value: string | number, rerender: () => void) => {
    const credits = typeof value === 'number' ? value : parseFloat(value || '0');
    if (!Number.isSafeInteger(credits) || credits < 0) {
      Notify.failure(tr('Credits must be non-negative whole numbers.|Los créditos deben ser números enteros no negativos.'));
      rerender();
      return;
    }
    row[field] = credits;
    const remainingRow = rowOf('remaining');
    const savedField = field === 'CPU' ? 'savedCPU' : 'savedInference';
    if (row.kind === 'increase' && remainingRow) {
      // The adder is read as "on top of what is left", so the total always restates the stored
      // figure plus this row — that is the number the save writes.
      remainingRow[field] = remainingRow[savedField] + credits;
    } else if (row.kind === 'remaining') {
      // A hand-written total wins: clearing the adder keeps the visible figure and the saved one equal.
      const increaseRow = rowOf('increase');
      if (increaseRow) increaseRow[field] = 0;
    }
    rerender();
  };

  const saveChanges = async () => {
    const operations = pendingOperations;
    if (!operations.length) return;
    isSaving = true;
    console.debug('[company-credit-budget] mutation started', { companyID, operations });
    try {
      const nextBudget = await mutateCompanyCreditBudget(companyID, operations);
      applyBudget(nextBudget);
      Notify.success(tr('Credit budget updated.|Presupuesto de créditos actualizado.'));
      console.debug('[company-credit-budget] mutation completed', {
        companyID,
        currentCPU: nextBudget.CurrentCPU,
        currentInference: nextBudget.CurrentInference,
      });
    } catch (requestError: any) {
      const message = errorText(requestError);
      Notify.failure(message);
      console.error('[company-credit-budget] mutation failed', { companyID, operations, error: message });
      // A lost mutation reply is ambiguous, so always re-read durable state before another action.
      await loadBudget(companyID);
    } finally {
      isSaving = false;
    }
  };

  const creditsColumn = (
    id: string,
    header: string,
    field: 'CPU' | 'Inference',
  ): ITableColumn<BudgetRow> => ({
    id,
    header,
    width: '110px',
    align: 'right',
    cellInputType: 'number',
    css: 'ff-mono',
    getValue: (row) => row[field],
    render: (row) => String(formatN(row[field]) || '0'),
    onCellEdit: (row, value, rerender) => editCell(row, field, value, rerender),
  });

  const columns: ITableColumn<BudgetRow>[] = [
    {
      id: 'concept',
      header: 'Concept|Concepto',
      width: 'minmax(120px, 1fr)',
      getValue: (row) => tr(row.concept),
    },
    creditsColumn('cpu', 'CPU', 'CPU'),
    creditsColumn('inference', 'AI|IA', 'Inference'),
  ];

  $effect(() => {
    const targetCompanyID = companyID;
    if (targetCompanyID <= 0 || targetCompanyID === loadedCompanyID) return;
    loadedCompanyID = targetCompanyID;
    untrack(() => { void loadBudget(targetCompanyID); });
  });
</script>

<section class="col-span-24 mt-4 rounded-[10px] border border-slate-200 bg-slate-50 p-10" aria-label="Company credit budget">
  <div class="flex flex-wrap items-center gap-8">
    <div class="text-[16px] ff-bold"><T text="Credit budget|Presupuesto de créditos" /></div>
    <Button
      name="Save|Guardar"
      icon="icon-[fa--floppy-o]"
      color="blue"
      css="ml-auto"
      disabled={isSaving || !pendingOperations.length}
      onClick={saveChanges}
    />
  </div>

  {#if loadError}
    <div class="mt-8 rounded-[8px] border border-red-200 bg-red-50 p-8 text-red-800">{loadError}</div>
  {:else if isLoading || !budget}
    <div class="mt-8 rounded-[8px] border border-slate-200 bg-white p-12 text-center text-slate-600">
      <T text="Loading credit budget...|Cargando presupuesto de créditos..." />
    </div>
  {:else}
    <div class="mt-6 flex flex-wrap items-start gap-x-12 gap-y-6">
      <div class="w-[420px] max-w-full grow-0">
        <TableGrid {columns} data={rows} height="160px" rowHeight={32} />
      </div>
      <div class="flex-1 min-w-[190px] text-[12px] text-slate-600">
        <div>
          <T text="User daily allowance is 50% of the company allowance.|El límite diario por usuario es el 50% del límite de la empresa." />
        </div>
        <div class="mt-4">
          <T text="CPU used this month|CPU usado este mes" />:
          <span class="ff-bold ff-mono">{formatN(budget.MonthCPUUsed) || '0'}</span>
        </div>
        <div class="mt-2">
          <T text="AI used this month|IA usada este mes" />:
          <span class="ff-bold ff-mono">{formatN(budget.MonthInferenceUsed) || '0'}</span>
        </div>
        {#if budget.Updated}
          <div class="mt-4 text-slate-500">
            <T text="Updated|Actualizado" />: {formatTime(budget.Updated, 'd-M-Y h:n')}
          </div>
        {/if}
      </div>
    </div>

    {#if !budget.IsCurrentMonth}
      <div class="mt-8 rounded-[8px] border border-amber-300 bg-amber-50 p-8 text-amber-900">
        <T text="No budget is active for the current month. APIs remain blocked until the required daily allowance and current budget are set.|No hay presupuesto activo para el mes actual. Las APIs permanecen bloqueadas hasta establecer el límite diario requerido y el presupuesto actual." />
      </div>
    {/if}
  {/if}
</section>
