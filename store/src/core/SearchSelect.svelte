<script lang="ts" generics="T">
  import { derived } from "svelte/store";
  import { highlString, throttle } from "../functions/helpers";
  import { Core } from "./store.svelte";
  import s1 from "./core.module.css";
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

  const {
    saveOn = $bindable(),
    save,
    css = "",
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
    inputCss = "",
    icon,
    showLoading = false,
  }: SearchSelectProps<T> = $props();

  let show = $state(false);
  let filteredOptions = $state<T[]>([...options]);
  let arrowSelected = $state(-1);
  let avoidhover = $state(false);
  let isValid = $state(0);
  let selectedValue = $state("");

  const [keyId, keyName] = keys.split(".") as [keyof T, keyof T];
  // let searchCardID = Math.random();
  let inputRef: HTMLInputElement;
  let words: string[] = [];

  const isMobile = $derived(Core.deviceType === 3);
  const useLayerPicker = $derived(isMobile);
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
    if (typeof currValue === "undefined" && save && saveOn) {
      currValue = saveOn[save as keyof T] as number | string;
    }
    let selectedItem: T | undefined;
    if (currValue) {
      selectedItem = options.find((x) => x[keyId] === currValue);
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
        changeValue("");
      }
    }

    const newValue = selectedItem ? selectedItem[keyId] : null;
    if (isRequired()) {
      isValid = newValue ? 2 : 1;
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
    } else if (typeof selected !== "undefined") {
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
      if (
        avoidIDSet.size > 0 &&
        avoidIDSet.has(op[keyId as keyof T] as number)
      ) {
        continue;
      }
      const name = op[keyName] as string;
      if (typeof name === "string") {
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
    const text = String(target.value || "").toLowerCase();

    throttle(() => {
      words = String(inputRef.value).split(" ");
      filteredOptions = filter(text);
    }, 120);
  }

  function onKeyDown(ev: KeyboardEvent) {
    console.log("avoid hover:: ", avoidhover);
    ev.stopPropagation();

    let nroR = filteredOptions.length;
    if (nroR > max) nroR = max;

    if (!show || filteredOptions.length === 0) return;

    if (ev.key === "ArrowUp") {
      let arrow = (arrowSelected || 0) - 1;
      if (arrow < 0) arrow = nroR;
      arrowSelected = arrow;
      if (!avoidhover) avoidhover = true;
    } else if (ev.key === "ArrowDown") {
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
    `${s1.input} p-rel${css ? ` ${css}` : ""}${!label ? " no-label" : ""}`,
  );

  function iconValid() {
    if (!isValid) return null;
    if (isValid === 2) {
      return `<i class="v-icon icon-ok c-green"></i>`;
    } else if (isValid === 1) {
      return `<i class="v-icon icon-attention c-red"></i>`;
    }
    return null;
  }

  // Watch for changes in props
  $effect(() => {
    const selectedItem = getSelectedFromProps();
    if (selectedItem) {
      changeValue(selectedItem[keyName] as string);
    } else {
      changeValue("");
    }
    if (isRequired()) {
      isValid = selectedItem ? 2 : 1;
    }
  });

  $effect(() => {
    filteredOptions = filter("");
  });

  function handleOpenMobileLayer() {
    // openSearchLayer = searchCardID;
  }
</script>

<div class={cN}>
  {#if label}
    <div class={s1.input_lab_cell_left}><div></div></div>
    <div class={s1.input_lab}>
      {label}{@html iconValid() || ""}
    </div>
    <div class={s1.input_lab_cell_right}><div></div></div>
  {/if}
  <div class={s1.input_shadow_layer}>
    <div></div>
  </div>
  <div class={`${s1.input_div} flex w-full`}>
    <div class={s1.input_div_1}>
      <div></div>
    </div>
    {#if !useLayerPicker}
      <input
        class={`w-full ${s1.input_inp} ${inputCss}`}
        bind:this={inputRef}
        onkeyup={onKeyUp}
        onpaste={onKeyUp as any}
        oncut={onKeyUp as any}
        onkeydown={(ev) => {
          ev.stopPropagation();
          console.log(ev);
          onKeyDown(ev);
        }}
        placeholder={showLoading ? "" : placeholder || ":: seleccione ::"}
        disabled={isDisabled || isMobile}
        onfocus={(ev) => {
          ev.stopPropagation();
          words = [];
          filteredOptions = filter("");
          show = true;
        }}
        onblur={(ev) => {
          ev.stopPropagation();
          let inputValue = String(inputRef.value || "").toLowerCase();
          const selectedItem = options.find((x) => {
            const itemName = String(x[keyName] || "");
            return itemName.toLowerCase() === inputValue;
          });
          setValueSaveOn(selectedItem, true);
          show = false;
        }}
      />
    {:else}
      <div
        class={`w-full ${s1.input_inp} ${inputCss}`}
        role="button" tabindex="0"
        onkeydown={(ev) => {
          if (ev.key === 'Enter' || ev.key === ' ') {
            handleOpenMobileLayer();
          }
        }}
        onclick={(ev) => {
          ev.stopPropagation();
          handleOpenMobileLayer();
        }}
      >
        <div class="h100 w100 flex items-center">{selectedValue}</div>
      </div>
    {/if}
    {#if !label}
      {@html iconValid() || ""}
    {/if}
  </div>
  {#if showLoading}
    <div>Cargando...</div>
  {/if}
  {#if !isDisabled}
    <div class="absolute bottom-8 right-6 {show && !icon ? 'show' : ''}">
      <i class={icon || "icon-down-open-1"}></i>
    </div>
  {/if}
  {#if show && !useLayerPicker}
    <div class="p-4 _1 left-0 z-40 w-full{arrowSelected >= 0 ? ' on-arrow' : ''}"
      role="button" tabindex="0"
      onmousemove={avoidhover
        ? (ev) => {
            console.log("hover aqui:: ", arrowSelected);
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
        <div class="flex ai-center _highlight{arrowSelected === i
            ? ' _selected'
            : ''}"
          role="button" tabindex="0"
          onmousedown={(ev) => {
            ev.stopPropagation();
            onOptionClick(e);
          }}
        >
          <div>
            {#each highlString(name, words) as w}
              <span class={w.highl ? "_8" : ""}>{w.text}</span>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  /* Import component styles - adjust path as needed */
  ._1 {
    position: absolute;
    top: 100%;
    background: white;
    border: 1px solid #ccc;
    border-radius: 4px;
    max-height: 300px;
    overflow-y: auto;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
    border-radius: 6px;
  }

  ._1 > div {
    display: flex;
    align-items: center;
    height: 36px;
    cursor: pointer;
    padding: 0 6px 0 6px;
    border-radius: 4px;
  }

  ._1 > div:hover {
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

  ._8 {
    color: rgb(196, 71, 71);
    text-decoration: underline;
  }
</style>
