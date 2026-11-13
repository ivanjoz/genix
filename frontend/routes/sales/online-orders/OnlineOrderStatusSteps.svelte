<script lang="ts">
  import ArrowSteps from '#components/navigation/ArrowSteps.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { currentFulfillmentStep, fulfillmentStepOptions, type IOnlineSaleOrder } from './online-orders.ts'

  // "Estado" with the order's steps beside it, as arrows. The current step is highlighted; clicking a
  // later one asks the page to move the order there.
  let { order, onSelectStep }: {
    order: IOnlineSaleOrder
    onSelectStep: (step: number) => void
  } = $props()
</script>

<div class="flex items-center gap-10" aria-label="Order status steps: in preparation, on the way or ready for pickup, delivered">
  <div class="text-fg-muted shrink-0">{tr('Status|Estado')}</div>
  <div class="grow min-w-0">
    <ArrowSteps options={fulfillmentStepOptions(order)} selected={currentFulfillmentStep(order)}
      onSelect={(step) => onSelectStep(step.id)}
    >
      {#snippet optionRender(step)}
        <div class="flex items-center ff-semibold">
          <i class={`text-[16px] mr-6 ${step.icon}`}></i>
          <div class="leading-[1.1]">{tr(step.name)}</div>
        </div>
      {/snippet}
    </ArrowSteps>
  </div>
</div>
