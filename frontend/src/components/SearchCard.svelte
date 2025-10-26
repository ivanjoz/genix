<script lang="ts" generics="T">
import { deviceType } from '~/app';
import SearchSelect from './SearchSelect.svelte';
import SearchMobileLayer from './SearchMobileLayer.svelte';
import { setOpenSearchLayer } from './searchSelectStore.svelte';

interface SearchCardProps<T> {
  saveOn?: any;
  save?: string | keyof T;
  css?: string;
  options: T[];
  keys: string;
  label?: string;
  placeholder?: string;
  max?: number;
  inputCss?: string;
}

let {
  saveOn = $bindable(),
  save,
  css = '',
  options = [],
  keys,
  label,
  placeholder,
  max = 100,
  inputCss = ''
}: SearchCardProps<T> = $props();

const [keyId, keyName] = keys.split('.') as [keyof T, keyof T];
let optionsSelected = $state<(number | string)[]>([]);
let id = Math.random();
let optionsMap = $state(new Map<number | string, T>());

function processOptions() {
  optionsMap.clear();
  for (let e of options) {
    optionsMap.set(e[keyId] as number | string, e);
  }
}

function doSetOptionsSelected(ids: (number | string)[]) {
  optionsSelected = [...ids];
  if (saveOn && save) {
    saveOn[save as keyof T] = ids as T[keyof T];
  }
}

$effect(() => {
  processOptions();
});

$effect(() => {
  if (saveOn && save) {
    optionsSelected = [...(saveOn[save] || [])];
  }
});
</script>

<div
  class="flex p-rel mr-06 {css}"
  style:padding-top={[3].includes(deviceType()) ? '0.8rem' : undefined}
  class:ml-06={[3].includes(deviceType())}
>
  {#if [3].includes(deviceType())}
    <div class="in-label p-abs" style="top: -2px">
      {label || placeholder}
    </div>
  {/if}
  {#if [1, 2].includes(deviceType())}
    <SearchSelect
      css={inputCss || 'w-08x'}
      {label}
      {options}
      {keys}
      {placeholder}
      avoidIDs={optionsSelected}
      clearOnSelect={true}
      icon="icon-search"
      onChange={(e) => {
        if (!e) return;
        const itemId = e[keyId];
        optionsSelected.push(itemId);
        doSetOptionsSelected(optionsSelected);
      }}
    />
  {/if}
  {#if [3].includes(deviceType())}
    <button
      class="bn1 d-green s1"
      style="margin-top: 2px"
      on:click={(ev) => {
        ev.stopPropagation();
        setOpenSearchLayer(id);
      }}
    >
      <i class="icon-plus"></i>
    </button>
  {/if}
  <div class="grow-1 flex-wrap card-c12p ml-06">
    {#each optionsSelected as itemId}
      {@const opt = optionsMap.get(itemId)}
      {@const name = String(opt ? opt[keyName] : `ITEM-${itemId}`)}
      <div class="card-c12">
        {name}
        <button
          class="bnr1 b-trash"
          on:click={(ev) => {
            ev.stopPropagation();
            const newSelected = optionsSelected.filter((x) => x !== itemId);
            doSetOptionsSelected(newSelected);
          }}
        >
          <i class="icon-trash"></i>
        </button>
      </div>
    {/each}
  </div>
  <SearchMobileLayer
    {id}
    {options}
    {keys}
    avoidIDs={optionsSelected}
    onSelect={(e) => {
      if (!e) return;
      const itemId = e[keyId];
      optionsSelected.push(itemId);
      doSetOptionsSelected(optionsSelected);
    }}
  />
</div>

