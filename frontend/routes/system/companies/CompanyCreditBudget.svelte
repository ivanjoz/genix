<script lang="ts">
  import Button from '$components/buttons/Button.svelte';
  import Input from '$components/form/Input.svelte';
  import T from '$components/misc/T.svelte';
  import { tr } from '$core/store.svelte';
  import { formatN, formatTime, Notify } from '$libs/helpers';
  import { untrack } from 'svelte';
  import {
    getCompanyCreditBudget,
    mutateCompanyCreditBudget,
    type CompanyCreditBudgetOperation,
    type ICompanyCreditBudget,
  } from './company-credit-budget';

  let { companyID }: { companyID: number } = $props();

  type CreditPair = { CPU: number; Inference: number };

  let budget = $state<ICompanyCreditBudget | null>(null);
  let dailyForm = $state<CreditPair>({ CPU: 0, Inference: 0 });
  let currentForm = $state<CreditPair>({ CPU: 0, Inference: 0 });
  let increaseForm = $state<CreditPair>({ CPU: 0, Inference: 0 });
  let loadedCompanyID = $state(0);
  let isLoading = $state(false);
  let isSaving = $state(false);
  let loadError = $state('');

  const errorText = (requestError: any): string => String(
    requestError?.errorMessage || requestError?.error || requestError?.message || requestError || 'error'
  );
  const validCredits = (value: string | number): boolean => (
    typeof value === 'number' && Number.isSafeInteger(value) && value >= 0
  );

  const applyBudget = (nextBudget: ICompanyCreditBudget) => {
    budget = nextBudget;
    dailyForm = { CPU: nextBudget.DailyCPU || 0, Inference: nextBudget.DailyInference || 0 };
    currentForm = { CPU: nextBudget.CurrentCPU || 0, Inference: nextBudget.CurrentInference || 0 };
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

  const saveBudget = async (operation: CompanyCreditBudgetOperation, values: CreditPair) => {
    if (!validCredits(values.CPU) || !validCredits(values.Inference)) {
      Notify.failure(tr('Credits must be non-negative whole numbers.|Los créditos deben ser números enteros no negativos.'));
      return;
    }
    isSaving = true;
    console.debug('[company-credit-budget] mutation started', { companyID, operation, ...values });
    try {
      const nextBudget = await mutateCompanyCreditBudget(companyID, operation, values.CPU, values.Inference);
      applyBudget(nextBudget);
      if (operation === 'increase-current') increaseForm = { CPU: 0, Inference: 0 };
      Notify.success(tr('Credit budget updated.|Presupuesto de créditos actualizado.'));
      console.debug('[company-credit-budget] mutation completed', {
        companyID,
        operation,
        currentCPU: nextBudget.CurrentCPU,
        currentInference: nextBudget.CurrentInference,
      });
    } catch (requestError: any) {
      const message = errorText(requestError);
      Notify.failure(message);
      console.error('[company-credit-budget] mutation failed', { companyID, operation, error: message });
      // A lost mutation reply is ambiguous, so always re-read durable state before another action.
      await loadBudget(companyID);
    } finally {
      isSaving = false;
    }
  };

  $effect(() => {
    const targetCompanyID = companyID;
    if (targetCompanyID <= 0 || targetCompanyID === loadedCompanyID) return;
    loadedCompanyID = targetCompanyID;
    untrack(() => { void loadBudget(targetCompanyID); });
  });
</script>

<section class="col-span-24 mt-4 rounded-[10px] border border-slate-200 bg-slate-50 p-10" aria-label="Company credit budget">
  <div class="mb-10 flex flex-wrap items-center gap-8">
    <div>
      <div class="text-[16px] ff-bold"><T text="Credit budget|Presupuesto de créditos" /></div>
      <div class="mt-2 text-[12px] text-slate-600">
        <T text="User daily allowance is 50% of the company allowance.|El límite diario por usuario es el 50% del límite de la empresa." />
      </div>
    </div>
    {#if budget?.Updated}
      <span class="ml-auto text-[12px] text-slate-500">
        <T text="Updated|Actualizado" />: {formatTime(budget.Updated, 'd-M-Y h:n')}
      </span>
    {/if}
  </div>

  {#if loadError}
    <div class="rounded-[8px] border border-red-200 bg-red-50 p-8 text-red-800">{loadError}</div>
  {:else if isLoading || !budget}
    <div class="rounded-[8px] border border-slate-200 bg-white p-12 text-center text-slate-600">
      <T text="Loading credit budget...|Cargando presupuesto de créditos..." />
    </div>
  {:else}
    <div class="mb-10 grid grid-cols-2 gap-8">
      <div class="rounded-[8px] bg-emerald-50 p-9">
        <div class="text-[12px] text-slate-600"><T text="Current CPU|CPU actual" /></div>
        <div class="mt-3 text-[18px] ff-bold ff-mono">{formatN(budget.CurrentCPU) || '0'}</div>
        <div class="mt-2 text-[11px] text-slate-500"><T text="Used this month|Usado este mes" />: {formatN(budget.MonthCPUUsed) || '0'}</div>
      </div>
      <div class="rounded-[8px] bg-purple-50 p-9">
        <div class="text-[12px] text-slate-600"><T text="Current AI|IA actual" /></div>
        <div class="mt-3 text-[18px] ff-bold ff-mono">{formatN(budget.CurrentInference) || '0'}</div>
        <div class="mt-2 text-[11px] text-slate-500"><T text="Used this month|Usado este mes" />: {formatN(budget.MonthInferenceUsed) || '0'}</div>
      </div>
    </div>

    {#if !budget.IsCurrentMonth}
      <div class="mb-10 rounded-[8px] border border-amber-300 bg-amber-50 p-8 text-amber-900">
        <T text="No budget is active for the current month. APIs remain blocked until the required daily allowance and current budget are set.|No hay presupuesto activo para el mes actual. Las APIs permanecen bloqueadas hasta establecer el límite diario requerido y el presupuesto actual." />
      </div>
    {/if}

    <div class="grid grid-cols-1 gap-8 lg:grid-cols-3">
      <div class="rounded-[8px] border border-slate-200 bg-white p-8">
        <div class="mb-7 ff-bold"><T text="Daily allowance|Límite diario" /></div>
        <div class="grid grid-cols-2 gap-7">
          <Input bind:saveOn={dailyForm} save="CPU" type="number" label="CPU" validator={validCredits} />
          <Input bind:saveOn={dailyForm} save="Inference" type="number" label="AI|IA" validator={validCredits} />
        </div>
        <Button
          name="Set daily|Establecer diario"
          icon="icon-[fa--calendar-day]"
          css="mt-8 w-full"
          disabled={isSaving}
          onClick={() => saveBudget('set-daily', dailyForm)}
        />
      </div>

      <div class="rounded-[8px] border border-amber-200 bg-white p-8">
        <div class="mb-7 ff-bold"><T text="Replace current|Reemplazar actual" /></div>
        <div class="grid grid-cols-2 gap-7">
          <Input bind:saveOn={currentForm} save="CPU" type="number" label="CPU" validator={validCredits} />
          <Input bind:saveOn={currentForm} save="Inference" type="number" label="AI|IA" validator={validCredits} />
        </div>
        <Button
          name="Set current|Establecer actual"
          icon="icon-[fa--pen]"
          color="orange"
          css="mt-8 w-full"
          disabled={isSaving}
          onClick={() => saveBudget('set-current', currentForm)}
        />
      </div>

      <div class="rounded-[8px] border border-emerald-200 bg-white p-8">
        <div class="mb-7 ff-bold"><T text="Increase current|Aumentar actual" /></div>
        <div class="grid grid-cols-2 gap-7">
          <Input bind:saveOn={increaseForm} save="CPU" type="number" label="CPU" validator={validCredits} />
          <Input bind:saveOn={increaseForm} save="Inference" type="number" label="AI|IA" validator={validCredits} />
        </div>
        <Button
          name="Add credits|Agregar créditos"
          icon="icon-[fa--plus]"
          color="green"
          css="mt-8 w-full"
          disabled={isSaving || !budget.IsCurrentMonth}
          onClick={() => saveBudget('increase-current', increaseForm)}
        />
      </div>
    </div>
  {/if}
</section>
