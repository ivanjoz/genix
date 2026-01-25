<script lang="ts" generics="T,E">
import { highlString, include } from '$core/lib/helpers';
import { throttle } from '$core/lib/helpers';
import { Core } from '$core/core/store.svelte';
  import s1 from "./components.module.css";
  import SvelteVirtualList from "@humanspeak/svelte-virtual-list";

  interface SearchSelectProps<T,E> {
    saveOn?: T;
    save?: keyof T;
    css?: string;
    optionsCss?: string
    options: E[];
    keyId: keyof E;
    keyName: keyof E;
    label?: string;
    placeholder?: string;
    max?: number;
    onChange?: (e: E) => void;
    selected?: number | string;
    notEmpty?: boolean;
    required?: boolean;
    disabled?: boolean;
    clearOnSelect?: boolean;
    avoidIDs?: (number|string)[];
    inputCss?: string;
    icon?: string;
    showLoading?: boolean;
  }

  const {
    saveOn = $bindable(),
    save,
    css = "",
    options = [],
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
    keyId,
    optionsCss,
    keyName
  }: SearchSelectProps<T,E> = $props();

  let show = $state(false);
  let filteredOptions = $state<E[]>([...options]);
  let arrowSelected = $state(-1);
  let avoidhover = $state(false);
  let isValid = $state(0);
  let selectedValue = $state("");
  let avoidBlur = false
  let openUp = $state(false);

  // let searchCardID = Math.random();
  let inputRef = $state<HTMLInputElement>();
  let vlRef = $state<any>();
  let words = $state<string[]>([]);

  const isMobile = $derived(Core.deviceType === 3);
  const useLayerPicker = $derived(isMobile);
  const isDisabled = $derived(disabled || showLoading);

  function checkPosition() {
    if (inputRef) {
      const rect = inputRef.getBoundingClientRect();
      const windowHeight = window.innerHeight;
      openUp = rect.top > windowHeight / 2;
    }
  }

  function changeValue(value: string) {
    if (useLayerPicker) {
      selectedValue = value;
    } else {
      if (inputRef) {
        inputRef.value = value;
      }
    }
  }

  function getSelectedFromProps(): E | undefined {
    let currValue = selected;
    if (typeof currValue === "undefined" && save && saveOn) {
      currValue = saveOn[save as keyof T] as number | string;
    }
    let selectedItem: E | undefined;
    if (currValue) {
      selectedItem = options.find((x) => x[keyId] === currValue);
    }
    return selectedItem;
  }

  function isRequired() {
    return required && !disabled;
  }

  let prevSetValueTime = Date.now();

  const setValueSaveOn = (selectedItem?: E, setOnInput?: boolean) => {
    const nowTime = Date.now()
    if (nowTime - prevSetValueTime < 80) { return }
    prevSetValueTime = nowTime

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
        onChange(selectedItem as E);
      }
    } else if (saveOn && save) {
      const current = (saveOn[save as keyof T] || null) as number;
      if (current !== newValue) {
        saveOn[save as keyof T] = newValue as NonNullable<T>[keyof T];
        if (onChange) {
          onChange(selectedItem as E);
        }
      }
    } else if (typeof selected !== "undefined") {
      if ((selected || null) !== newValue) {
        if (onChange) {
          onChange(selectedItem as E);
        }
      }
    }
  }

  const filter = (text: string) => {
    console.log("options filtered::", label, options)
    if (!text && (avoidIDs?.length || 0) === 0) {
      return [...options];
    }
    const avoidIDSet = new Set(avoidIDs || [])
    const filtered: E[] = []
    const searchWords = text.toLowerCase().split(" ").filter(x => x.length > 1)

    for (const opt of options) {
      if (avoidIDSet.size > 0 && avoidIDSet.has(opt[keyId as keyof E] as number)){
        continue;
      }
      if(searchWords.length === 0){
        filtered.push(opt)
      } else {
        const name = opt[keyName] as string
        if (typeof name === "string") {
          if(include(name.toLowerCase(), searchWords)){ filtered.push(opt) }
        }
      }
    }
    console.log("options filtered 2::", label, options, filtered)
    return filtered;
  }

  function onKeyUp(ev: KeyboardEvent) {
    ev.stopPropagation();
    const target = ev.target as HTMLInputElement;
    const text = String(target.value || "").toLowerCase();

    throttle(() => {
      words = String(inputRef.value).toLowerCase().split(" ");
      filteredOptions = filter(text);
      arrowSelected = -1;
    }, 120);
  }

  function onKeyDown(ev: KeyboardEvent) {
    console.log("avoid hover:: ", avoidhover);
    ev.stopPropagation();

    if (!show || filteredOptions.length === 0) return;

    if (ev.key === "ArrowUp") {
      ev.preventDefault();
      arrowSelected = arrowSelected <= 0 ? filteredOptions.length - 1 : arrowSelected - 1;
      avoidhover = true;
      vlRef?.scrollToIndex(arrowSelected, { align: 'auto' });
    } else if (ev.key === "ArrowDown") {
      ev.preventDefault();
      arrowSelected = arrowSelected >= filteredOptions.length - 1 ? 0 : arrowSelected + 1;
      avoidhover = true;
      vlRef?.scrollToIndex(arrowSelected, { align: 'auto' });
    } else if (ev.key === "Enter" && arrowSelected >= 0) {
      ev.preventDefault();
      onOptionClick(filteredOptions[arrowSelected]);
    }
  }

  function onOptionClick(opt: E) {
    setValueSaveOn(opt, true)
    if (inputRef) { inputRef.blur() }
    show = false
  }

  let cN = $derived(
    `${s1.input} p-rel${css ? ` ${css}` : ""}${!label ? " no-label" : ""}`,
  )

  function iconValid() {
    if (!isValid) return null;
    if (isValid === 2) {
      return `<i class="v-icon icon-ok text-green-600"></i>`;
    } else if (isValid === 1) {
      return `<i class="v-icon icon-attention text-red-600"></i>`;
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
    arrowSelected = -1;
  });

  const handleOpenMobileLayer = () => {
    Core.showMobileSearchLayer = {
      options: filteredOptions,
      keyName: keyName as string,
      keyID: keyId as string,
      onSelect: (e) => { onOptionClick(e) },
      onRemove: (e) => {

      }
    }
  }

</script>

<div class={cN}>
  {#if label}
    <div class={s1.input_lab_cell_left}><div></div></div>
    <div class={s1.input_lab}>
      {label}{@html iconValid() || ""}
    </div>
    <div class={s1.input_lab_cell_right}><div></div></div>
    <div class={s1.input_shadow_layer}><div></div></div>
  {/if}
  <div class={`${s1.input_div} flex w-full`}>
    {#if label}
      <div class={s1.input_div_1}><div></div></div>
    {/if}
    {#if !useLayerPicker}
      <input class={`w-full ${s1.input_inp} ${inputCss}`}
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
          if(!show){
            checkPosition();
            words = [];
            filteredOptions = filter("");
            show = true;
          }
        }}
        onblur={(ev) => {
          ev.stopPropagation();
          console.log("avoidBlur 2", avoidBlur)
          if(avoidBlur){
            avoidBlur = false
            inputRef.focus()
            return
          }

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
      <div class={`w-full flex items-center ${s1.input_inp} ${inputCss}`}
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
        <div class="h100 w100 flex items-center">
          {#if selectedValue}
            {selectedValue}
          {:else}
            <div class="fs15 mt-2 _10">{placeholder||""}</div>
          {/if}
        </div>
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
    <div class="absolute bottom-8 right-6 pointer-events-none {show && !icon ? 'show' : ''}">
      <i class={icon || "icon-down-open-1"}></i>
    </div>
  {/if}
  {#if show && !useLayerPicker}
    <div class="_1 p-4 left-0 z-320 {arrowSelected >= 0 ? ' on-arrow' : ''} {optionsCss || "w-full"}"
      style:height={Math.min(filteredOptions.length * 36 + 10, 300) + 'px'}
      class:open-up={openUp}
      role="button" tabindex="0"
      onmousedown={(ev) => {
        ev.stopPropagation()
        avoidBlur = true
        console.log("avoidBlur 1", avoidBlur)
      }}
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
      <SvelteVirtualList bind:this={vlRef} items={filteredOptions}>
        {#snippet renderItem(e, i)}
          {@const name = String(e[keyName])}
          {@const highlighted = highlString(name, words)}
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <div
            class="flex ai-center _highlight{arrowSelected === i ? ' _selected' : ''}"
            role="option"
            aria-selected={arrowSelected === i}
            tabindex="0"
            onclick={(ev) => {
              ev.stopPropagation();
              onOptionClick(e);
            }}
          >
            <div>
              {#each highlighted as w}
                <span class={w.highl ? "_8" : ""} class:mr-4={w.isEnd}>{w.text}</span>
              {/each}
            </div>
          </div>
        {/snippet}
      </SvelteVirtualList>
    </div>
  {/if}
</div>

<style>
  ._1 {
    position: absolute;
    top: 100%;
    background: white;
    border: 1px solid #ccc;
    border-radius: 4px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
    border-radius: 6px;
    overflow: hidden;
  }

  ._1.open-up {
    top: auto;
    bottom: calc(100% - 8px);
    box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.15);
  }

  ._highlight {
    display: flex;
    align-items: center;
    height: 36px;
    cursor: pointer;
    padding: 0 6px 0 6px;
    border-radius: 4px;
  }

  ._highlight:hover {
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
  ._10 {
    color: #6d5dad;
  }
</style>
