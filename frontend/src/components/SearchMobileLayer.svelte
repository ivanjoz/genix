<script lang="ts" generics="T">
import { throttle } from '~/core/main';
import { openSearchLayer, setOpenSearchLayer } from './searchSelectStore.svelte';
import { Env } from '~/env';
import { highlString } from './searchSelectUtils';

export interface SearchMobileLayerProps<T> {
  id: number;
  css?: string;
  options: T[];
  keys: string;
  label?: string;
  placeholder?: string;
  max?: number;
  onSelect?: (e: T) => void;
  selected?: number | string;
  notEmpty?: boolean;
  required?: boolean;
  clearOnSelect?: boolean;
  avoidIDs?: (number | string)[];
  inputCss?: string;
}

let {
  id,
  css = '',
  options = [],
  keys,
  label,
  placeholder,
  max = 100,
  onSelect,
  selected,
  notEmpty = false,
  required = false,
  clearOnSelect = false,
  avoidIDs = [],
  inputCss = ''
}: SearchMobileLayerProps<T> = $props();

let inputRef: HTMLTextAreaElement;
let onMouseStatus = 1;

const [keyId, keyName] = keys.split('.') as [keyof T, keyof T];
let filteredOptions = $state<T[]>([...options]);
let words = $state<string[]>([]);

const show = $derived(openSearchLayer === id);

function filter(text: string) {
  if (!text && (avoidIDs?.length || 0) === 0) {
    return [...options];
  }
  const avoidIDSet = new Set(avoidIDs || []);

  const filtered: T[] = [];
  for (let op of options) {
    if (avoidIDSet.size > 0 && avoidIDSet.has(op[keyId as keyof T] as number)) {
      continue;
    }
    const name = op[keyName] as string;
    if (typeof name === 'string') {
      const nameN = name.toLowerCase();
      if (!text || nameN.includes(text)) {
        filtered.push(op);
      }
    }
  }
  return filtered;
}

function onKeyUp(ev: KeyboardEvent) {
  ev.stopPropagation();
  const target = ev.target as HTMLInputElement;

  throttle(() => {
    words = String(inputRef.value)
      .split(' ')
      .map((x) => x.toLocaleLowerCase());
    filteredOptions = filter(String(target.value || '').toLowerCase());
  }, 120);
}

$effect(() => {
  if (show) {
    if (inputRef) {
      inputRef.focus();
      setTimeout(() => {
        inputRef.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }, 150);
    }
    words = [];
    filteredOptions = filter('');
    onMouseStatus = 0;
  }
});

$effect(() => {
  if (openSearchLayer > 0) {
    Env.suscribeUrlFlag('mob-search-layer', () => setOpenSearchLayer(0));
  }
});
</script>

{#if show}
  <div id="mob-search-layer" class="search-background">
    <div class="flex items-center">
      <div class="p-rel grow px-08 mt-06">
        <i class="i-search icon-search p-abs"></i>
        <textarea
          rows={1}
          class="w100 increment ta-input"
          bind:this={inputRef}
          on:keyup={onKeyUp}
          placeholder="Seleccione..."
          style="margin-bottom: 2px"
          on:blur={(ev) => {
            ev.stopPropagation();
            console.log('blur:: mouse status', onMouseStatus);
            if (onMouseStatus === 1) {
              onMouseStatus = 0;
            } else {
              setOpenSearchLayer(0);
            }
          }}
        />
      </div>
      <button
        class="mr-08 search-bg-close-btn"
        on:mousedown={(ev) => {
          ev.stopPropagation();
          setOpenSearchLayer(0);
        }}
      >
        <i class="icon-cancel"></i>
      </button>
    </div>
    <div class="w100 px-08 options-c2">
      {#each filteredOptions as opt, i}
        {@const optName = String(opt[keyName])}
        <div
          class="flex ai-center jc-center card-c2{i % 2 === 0 ? ' mr-08' : ''}"
          style="width: 100%"
          on:mousedown={(ev) => {
            ev.stopPropagation();
            onMouseStatus = 1;
          }}
          on:click={(ev) => {
            ev.stopPropagation();
            if (onSelect) {
              onSelect(opt);
            }
            setOpenSearchLayer(0);
          }}
        >
          <div class="_highlight s1">
            {#each highlString(optName, words) as part}
              {#if typeof part === 'string'}
                {part}
              {:else}
                <i>{part.text}</i>
              {/if}
            {/each}
          </div>
        </div>
      {/each}
    </div>
  </div>
{/if}

<style>
  .search-background {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.5);
    z-index: 1000;
    overflow-y: auto;
  }

  ._highlight i {
    font-style: normal;
    font-weight: bold;
    color: #1890ff;
  }
</style>

