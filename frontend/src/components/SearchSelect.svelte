<script lang="ts" generics="T">
import { throttle } from '~/core/main';
import { isMobile } from '~/app';
import { Spinner4 } from './Cards';
import { highlString } from './searchSelectUtils';
import { openSearchLayer, setOpenSearchLayer } from './searchSelectStore.svelte';

interface SearchSelectProps<T> {
  saveOn?: any;
  save?: string | keyof T;
  css?: string;
  options: T[];
  keys: string;
  label?: string;
  placeholder?: string;
  max?: number;
  onChange?: (e: T) => void;
  selected?: number | string;
  notEmpty?: boolean;
  required?: boolean;
  disabled?: boolean;
  clearOnSelect?: boolean;
  avoidIDs?: number[];
  inputCss?: string;
  icon?: string;
  showLoading?: boolean;
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
  onChange,
  selected = $bindable(),
  notEmpty = false,
  required = false,
  disabled = false,
  clearOnSelect = false,
  avoidIDs = [],
  inputCss = '',
  icon,
  showLoading = false
}: SearchSelectProps<T> = $props();

let show = $state(false);
let filteredOptions = $state<T[]>([...options]);
let arrowSelected = $state(-1);
let avoidhover = $state(false);
let isValid = $state(0);
let selectedValue = $state('');

const [keyId, keyName] = keys.split('.') as [keyof T, keyof T];
let searchCardID = Math.random();
let inputRef: HTMLInputElement;
let words: string[] = [];

const useLayerPicker = $derived(isMobile());
const isDisabled = $derived(disabled || showLoading);

function changeValue(value: string) {
  if (useLayerPicker) {
    selectedValue = value;
  } else {
    if (inputRef) {
      inputRef.value = value;
    }
  }
}

function getSelectedFromProps(): T | undefined {
  let currValue = selected;
  if (typeof currValue === 'undefined' && save && saveOn) {
    currValue = saveOn[save as keyof T] as number | string;
  }
  let selectedItem: T | undefined;
  if (currValue) {
    selectedItem = options.find(x => x[keyId] === currValue);
  }
  return selectedItem;
}

function isRequired() {
  return required && !disabled;
}

let prevSetValueTime = Date.now();

function setValueSaveOn(selectedItem?: T, setOnInput?: boolean) {
  const nowTime = Date.now();
  if (nowTime - prevSetValueTime < 80) {
    return;
  }
  prevSetValueTime = nowTime;

  if (notEmpty && !selectedItem) {
    selectedItem = getSelectedFromProps();
  }

  if (setOnInput) {
    if (selectedItem && !clearOnSelect) {
      changeValue(selectedItem[keyName] as string);
    } else {
      changeValue('');
    }
  }

  const newValue = selectedItem ? selectedItem[keyId] : null;
  if (isRequired()) {
    isValid = newValue ? 1 : 2;
  }

  if (clearOnSelect) {
    if (onChange) {
      onChange(selectedItem);
    }
  } else if (saveOn && save) {
    const current = (saveOn[save as keyof T] || null) as number;
    if (current !== newValue) {
      saveOn[save as keyof T] = newValue;
      if (onChange) {
        onChange(selectedItem);
      }
    }
  } else if (typeof selected !== 'undefined') {
    if ((selected || null) !== newValue) {
      if (onChange) {
        onChange(selectedItem);
      }
    }
  }
}

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
  const text = String(target.value || '').toLowerCase();

  throttle(() => {
    words = String(inputRef.value).split(' ');
    filteredOptions = filter(text);
  }, 120);
}

function onKeyDown(ev: KeyboardEvent) {
  console.log('avoid hover:: ', avoidhover);
  ev.stopPropagation();

  let nroR = filteredOptions.length;
  if (nroR > max) nroR = max;

  if (!show || filteredOptions.length === 0) return;

  if (ev.key === 'ArrowUp') {
    let arrow = (arrowSelected || 0) - 1;
    if (arrow < 0) arrow = nroR;
    arrowSelected = arrow;
    if (!avoidhover) avoidhover = true;
  } else if (ev.key === 'ArrowDown') {
    let arrow = (arrowSelected || 0) + 1;
    if (arrow > nroR) arrow = 1;
    arrowSelected = arrow;
    if (!avoidhover) avoidhover = true;
  }
}

function onOptionClick(opt: T) {
  if (inputRef) {
    setValueSaveOn(opt, true);
  }
}

let cN = $derived(
  `in-5c p-rel flex-column a-start ${css}${!label ? ' no-label' : ''}`
);

function iconValid() {
  if (!isValid) return null;
  if (isValid === 1) {
    return { type: 'ok', class: 'c-green' };
  } else {
    return { type: 'attention', class: 'c-red' };
  }
}

// Watch for changes in props
$effect(() => {
  const selectedItem = getSelectedFromProps();
  if (selectedItem) {
    changeValue(selectedItem[keyName] as string);
  } else {
    changeValue('');
  }
  if (isRequired()) {
    isValid = selectedItem ? 1 : 2;
  }
});

$effect(() => {
  filteredOptions = filter('');
});

function handleOpenMobileLayer() {
  openSearchLayer = searchCardID;
}
</script>

<div class={cN}>
  {#if label}
    <div class="input_lab">
      {label}
      {#if iconValid()}
        <i class="v-icon icon-{iconValid().type} {iconValid().class}"></i>
      {/if}
    </div>
    <div class="input_lab1">
      {label}
      {#if iconValid()}
        <i class="v-icon icon-{iconValid().type} {iconValid().class}"></i>
      {/if}
    </div>
  {/if}
  <div class="input_div w100">
    {#if !useLayerPicker}
      <input
        class="input_inp {inputCss}"
        bind:this={inputRef}
        on:keyup={onKeyUp}
        on:paste={onKeyUp}
        on:cut={onKeyUp}
        on:keydown={(ev) => {
          ev.stopPropagation();
          console.log(ev);
          onKeyDown(ev);
        }}
        placeholder={showLoading ? '' : placeholder || ':: seleccione ::'}
        disabled={isDisabled || isMobile()}
        on:focus={(ev) => {
          ev.stopPropagation();
          words = [];
          filteredOptions = filter('');
          show = true;
        }}
        on:blur={(ev) => {
          ev.stopPropagation();
          let inputValue = String(inputRef.value || '').toLowerCase();
          const selectedItem = options.find((x) => {
            const itemName = String(x[keyName] || '');
            return itemName.toLowerCase() === inputValue;
          });
          setValueSaveOn(selectedItem, true);
          show = false;
        }}
      />
    {:else}
      <div
        class="input_inp {inputCss}"
        on:click={(ev) => {
          ev.stopPropagation();
          handleOpenMobileLayer();
        }}
      >
        <div class="h100 w100 flex items-center">{selectedValue}</div>
      </div>
    {/if}
  </div>
  {#if showLoading}
    <Spinner4 style="position: absolute; left: 0.7rem; bottom: 8px" />
  {/if}
  {#if !isDisabled}
    <div class="input_icon1 p-abs {show && !icon ? 'show' : ''}">
      <i class={icon || 'icon-down-open-1'}></i>
    </div>
  {/if}
  {#if show && !useLayerPicker}
    <div
      class="search-ctn z20 w100{arrowSelected >= 0 ? ' on-arrow' : ''}"
      on:mousemove={avoidhover
        ? (ev) => {
            console.log('hover aqui:: ', arrowSelected);
            ev.stopPropagation();
            if (avoidhover) {
              arrowSelected = -1;
              avoidhover = false;
            }
          }
        : undefined}
    >
      {#each filteredOptions as e, i}
        {@const name = String(e[keyName])}
        <div
          class="flex ai-center _highlight{arrowSelected === i ? ' _selected' : ''}"
          on:mousedown={(ev) => {
            ev.stopPropagation();
            onOptionClick(e);
          }}
        >
          <div class="txt">
            {#each highlString(name, words) as part}
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
  {/if}
</div>

<style>
  /* Import component styles - adjust path as needed */
  @import './components.module.css';
  
  .search-ctn {
    position: absolute;
    top: 100%;
    left: 0;
    background: white;
    border: 1px solid #ccc;
    border-radius: 4px;
    max-height: 300px;
    overflow-y: auto;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  }
  
  .search-ctn > div {
    padding: 0.5rem;
    cursor: pointer;
  }
  
  .search-ctn > div:hover {
    background-color: #f0f0f0;
  }
  
  ._selected {
    background-color: #e6f7ff;
  }
  
  ._highlight i {
    font-style: normal;
    font-weight: bold;
    color: #1890ff;
  }
</style>

