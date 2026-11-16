<script lang="ts">
  import { tr } from '#core/store.svelte.ts'
  import { formatN, highlString } from '#libs/helpers.ts'
  import { addToRoundDraft, filterMenuPickItems, menuPickItems } from './floor-service.ts'
  import type { RoundDraftState } from './floor-service.svelte.ts'

  // The layout's menu beside "Nuevo pedido": every presentation with units left, in one list to
  // scroll; a click adds one unit to the round. The optional filter narrows it by words of the name.
  let { round }: { round: RoundDraftState } = $props()

  let filterText = $state('')
  const pickItems = $derived(menuPickItems(round.menu))
  const shownItems = $derived(filterMenuPickItems(pickItems, filterText))
  const filterWords = $derived(filterText.toLowerCase().split(' ').filter(Boolean))
</script>

<div class="h-full flex flex-col gap-8" aria-label="The menu: click a product to add one unit to the round">
  <div class="flex items-center gap-8">
    <i class="icon-[mdi--silverware-fork-knife] text-[20px] text-fg-muted"></i>
    <span class="h3">{tr('Menu|Carta')}</span>
    <span class="ml-auto text-fg-muted">{shownItems.length}</span>
  </div>
  <input type="search" bind:value={filterText} placeholder={tr('Filter...|Filtrar...')} aria-label="Filter the menu by name"
    class="w-full px-12 py-6 bg-surface-soft border border-line rounded-lg focus:outline-none focus:border-blue-solid" />
  <div class="flex-1 min-h-0 overflow-y-auto overflow-x-hidden flex flex-col gap-4 pr-4">
    {#each shownItems as item (item.ID)}
      <button type="button" aria-label="Add one {item.Name} to the round"
        class="flex items-center gap-8 px-8 py-6 rounded-md bg-surface text-left border border-line hover:border-blue-solid hover:bg-blue-bg"
        onclick={() => { round.draft = addToRoundDraft(round.draft, round.menu, item.ID) }}>
        <span class="flex-1 leading-tight">{#each highlString(item.Name, filterWords) as part}{#if part.highl}<span class="bg-yellow-bg-strong ff-bold">{part.text}</span>{:else}{part.text}{/if}{/each}</span>
        <span class="shrink-0 flex flex-col items-end leading-tight">
          <span class="ff-mono">{formatN(item.Price / 100, 2)}</span>
          <span class="text-fg-muted text-nowrap">{item.Sellable} {tr('left|disp.')}</span>
        </span>
      </button>
    {:else}
      <div class="p-12 rounded-md bg-surface-soft text-fg-muted">
        {round.isMenuLoading ? tr('Loading the menu...|Cargando la carta...') : tr('No product to order.|Ningún producto para pedir.')}
      </div>
    {/each}
  </div>
</div>
