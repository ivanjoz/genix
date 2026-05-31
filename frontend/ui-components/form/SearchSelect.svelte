<script lang="ts" generics="T,E">
import { highlString, persistFieldValue, readFieldValue, wordInclude } from '$libs/helpers';
import { throttle } from '$libs/helpers';
import { Core, tr } from '$core/store.svelte';
import T from '$components/misc/T.svelte';
import type { Snippet } from 'svelte';
  import s1 from "../components.module.css";
  import SvelteVirtualList from "@humanspeak/svelte-virtual-list";
    import { Env } from '$core/env';
    import { Agent, type AgentOption } from "$components/agent/registry";

  interface SearchSelectProps<T,E> {
    saveOn?: T;
    save?: keyof T;
    css?: string;
    useStyle?: number;
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
    id?: number;
    useCache?: boolean;
    optionRenderer?: Snippet<[E, string[]]>;
    getSearchText?: (e: E) => string;
    useDividingLine?: boolean;
  }

  const {
    saveOn = $bindable(),
    save,
    css = "",
    useStyle = 0,
    options = [],
    label,
    placeholder,
    max = 200,
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
    keyName,
    id,
    useCache = false,
    optionRenderer,
    getSearchText,
    useDividingLine = false,
  }: SearchSelectProps<T,E> = $props();

  let show = $state(false);
  // svelte-ignore state_referenced_locally
  let filteredOptions = $state<E[]>([...options]);
  let arrowSelected = $state(-1);
  let avoidhover = $state(false);
  let isValid = $state(0);
  let selectedValue = $state("");
  let avoidBlur = false
  let openUp = $state(false);
  let isFocused = $state(false);

  // let searchCardID = Math.random();
  let inputRef = $state<HTMLInputElement>();
  let vlRef = $state<any>();
  let words = $state<string[]>([]);

  const isMobile = $derived(Core.deviceType === 3);
  const useLayerPicker = $derived(isMobile);
  const isDisabled = $derived(disabled || showLoading);
  const useVirtualizedOptions = $derived(filteredOptions.length > 15);
  type SearchOptionID = number | string;

  // Keep id normalization centralized so all caches use the same key format.
  function getOptionId(option: E): SearchOptionID {
    const rawOptionId = option[keyId as keyof E] as unknown;
    if (typeof rawOptionId === "number" || typeof rawOptionId === "string") {
      return rawOptionId;
    }
    return String(rawOptionId ?? "");
  }

  // Normalize input once and reuse it in filter + highlight logic.
  function splitSearchWords(text: string): string[] {
    return String(text).toLowerCase().split(" ").filter((word) => word.length > 1);
  }

  type PreparedOption = {
    id: SearchOptionID;
    label: string;
    normalizedLabel: string;
    option: E;
  };

  let preparedOptions: PreparedOption[] = [];
  let avoidedOptionIdSet = new Set<SearchOptionID>();
  let cacheHydratedForId: number | undefined;

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
    if (!currValue) {
      return undefined;
    }
    const normalizedSelectedId = String(currValue);

    // Resolve against current options so hydration still works after remounts or async option refreshes.
    return options.find((option) => String(getOptionId(option)) === normalizedSelectedId);
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

    let valueChanged = false;
    if (clearOnSelect) {
      valueChanged = true;
      if (onChange) {
        onChange(selectedItem as E);
      }
    } else if (saveOn && save) {
      const current = (saveOn[save as keyof T] || null) as number;
      if (current !== newValue) {
        saveOn[save as keyof T] = newValue as NonNullable<T>[keyof T];
        valueChanged = true;
        if (onChange) {
          onChange(selectedItem as E);
        }
      }
    } else if (typeof selected !== "undefined") {
      if ((selected || null) !== newValue) {
        valueChanged = true;
        if (onChange) {
          onChange(selectedItem as E);
        }
      }
    }

    if (valueChanged && typeof id === "number" && id > 0) {
      persistFieldValue(id, newValue as number | string | null);
    }
  }

  const filter = (text: string, preparedWords?: string[]) => {
    const searchWords = preparedWords || splitSearchWords(text);
    if (searchWords.length === 0 && avoidedOptionIdSet.size === 0) {
      return options
    }
    const filtered: E[] = []

    for (const preparedOption of preparedOptions) {
      if (avoidedOptionIdSet.has(preparedOption.id)) {
        continue;
      }
      if (searchWords.length === 0) {
        filtered.push(preparedOption.option)
      } else {
        if (wordInclude(preparedOption.normalizedLabel, searchWords)) {
          filtered.push(preparedOption.option)
        }
      }
      // Stop as soon as the visible batch is filled to avoid scanning all 10k options.
      if (max > 0 && filtered.length >= max) {
        break;
      }
    }
    return filtered;
  }

  function applySearch(text: string, openList = false) {
    const searchWords = splitSearchWords(text);
    if (inputRef) { inputRef.value = text; }
    words = searchWords;
    filteredOptions = filter(text, searchWords);
    arrowSelected = -1;
    if (openList) { show = true; }
  }

  function onKeyUp(ev: KeyboardEvent) {
    ev.stopPropagation();
    throttle(() => {
      if (!inputRef) { return; }
      applySearch(String(inputRef.value || ""));
    }, 120);
  }

  function onKeyDown(ev: KeyboardEvent) {
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

  function clearSelectedValue() {
    // Keep the clear path explicit so the mobile layer can reset the component without selecting a fake option.
    setValueSaveOn(undefined, true)
    if (inputRef) { inputRef.blur() }
    show = false
  }

  let cN = $derived(
    `${s1.input} p-rel${css ? ` ${css}` : ""}${!label ? " no-label" : ""}${useStyle ? ` use-style-${useStyle}` : ""}`,
  )

  const arrowDirectionClass = $derived(show ? "arrow-up is-open" : "arrow-down");

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
    const nextPreparedOptions: PreparedOption[] = [];

    // Build plain caches once per options change so the keystroke path stays allocation-light.
    for (const option of options) {
      const optionId = getOptionId(option);
      const optionLabel = String(option[keyName] ?? "");
      const searchText = getSearchText ? String(getSearchText(option) ?? "") : optionLabel;
      nextPreparedOptions.push({
        id: optionId,
        label: optionLabel,
        normalizedLabel: searchText.toLowerCase(),
        option,
      });
    }

    preparedOptions = nextPreparedOptions;
    avoidedOptionIdSet = new Set(avoidIDs || []);
    filteredOptions = filter("");
    arrowSelected = -1;

    // Hydrate from localStorage once options are available, but only the first time per id.
    if (useCache && typeof id === "number" && id > 0 && options.length >= 1 && cacheHydratedForId !== id) {
      cacheHydratedForId = id;
      const cachedRaw = readFieldValue(id);
      if (cachedRaw) {
        const matchedOption = options.find((opt) => String(getOptionId(opt)) === cachedRaw);
        if (matchedOption) {
          // Bypass the 80ms throttle so the very-first hydration call is not swallowed.
          prevSetValueTime = 0;
          setValueSaveOn(matchedOption, true);
        } else {
          // Cached id no longer matches any current option; drop the stale entry.
          persistFieldValue(id, null);
        }
      }
    }
  });

  const handleOpenMobileLayer = () => {
    Core.showMobileSearchLayer = {
      options: filteredOptions,
      keyName: keyName as string,
      keyID: keyId as string,
      onSelect: (e) => { onOptionClick(e) },
      onClear: notEmpty ? undefined : () => {
        clearSelectedValue()
      },
      onRemove: (e) => {

      }
    }
  }

  const componentID = Env.getComponentID()

  $effect(() => {
    return Agent.register({
      id: componentID,
      type: "Select",
      label: label || placeholder || "",
      search: (text: string) => {
        applySearch(text, true);
        const out: AgentOption[] = [];
        for (const opt of filteredOptions) {
          if (out.length >= 50) { break; }
          out.push({ ID: getOptionId(opt), Value: String(opt[keyName] ?? "") });
        }
        return out;
      },
      select: (...ids) => {
        if (ids.length === 0) { return; }
        const targetId = String(ids[0]);
        const matched = options.find((opt) => String(getOptionId(opt)) === targetId);
        if (matched) { onOptionClick(matched); }
      },
      getOptions: (maxOptions = 50) => {
        const out: AgentOption[] = [];
        for (const prepared of preparedOptions) {
          if (out.length >= maxOptions) { break; }
          out.push({ ID: prepared.id, Value: prepared.label });
        }
        return out;
      },
    });
  });

  // data-value carries the selected id + label so the agent can read it from the DOM.
  const agentDataValue = $derived.by(() => {
    const item = getSelectedFromProps();
    if (!item) { return ""; }
    return `[${getOptionId(item)}] ${item[keyName as keyof E] as string}`;
  });
  const agentDataLabel = $derived(label || placeholder || "");
</script>

<div data-id="Select:{componentID}" data-value={agentDataValue} data-label={agentDataLabel} data-type="other" data-options-count={options.length} class={cN}>
  {#if label}
    <div class={s1.input_lab_cell_left}><div></div></div>
    <div class={s1.input_lab}>
      <T text={label} />{@html iconValid() || ""}
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
          onKeyDown(ev);
        }}
        placeholder={showLoading ? "" : tr(placeholder || ":: seleccione ::")}
        disabled={isDisabled || isMobile}
        onfocus={(ev) => {
          ev.stopPropagation();
          isFocused = true;
          if(!show){
            checkPosition();
            words = [];
            filteredOptions = filter("");
            show = true;
          }
        }}
        onblur={(ev) => {
          ev.stopPropagation();
          isFocused = false;
          // Ignore blur caused by interactions inside the dropdown list.
          if (avoidBlur) {
            avoidBlur = false;
            return;
          }

          let inputValue = String(inputRef?.value || "").toLowerCase();
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
        onfocus={() => {
          isFocused = true;
          checkPosition();
        }}
        onblur={() => {
          isFocused = false;
        }}
        onclick={(ev) => {
          ev.stopPropagation();
          handleOpenMobileLayer();
        }}
      >
        <div class="h100 w100 min-w-0 flex items-center">
          {#if selectedValue}
            <!-- Keep mobile trigger text on a single line so dense filters stay aligned. -->
            <div class="w-full truncate">
              {selectedValue}
            </div>
          {:else}
            <!-- Apply the same single-line constraint to placeholder text. -->
            <div class="fs15 mt-2 w-full truncate _10"><T text={placeholder || ""} /></div>
          {/if}
        </div>
      </div>
    {/if}
    {#if !label}
      {@html iconValid() || ""}
    {/if}
  </div>
  {#if showLoading}
    <div><T text="Loading...|Cargando..." /></div>
  {/if}
  {#if !isDisabled}
    <div class={`absolute bottom-8 right-6 pointer-events-none select-arrow ${arrowDirectionClass}`}>
      <i class={icon || "icon-down-open-1"}></i>
    </div>
  {/if}
  {#if show && !useLayerPicker}
    <div class="_1 p-4 left-0 z-320 {arrowSelected >= 0 ? ' on-arrow' : ''} {optionsCss || "w-full"}"
      style:height={useVirtualizedOptions ? '300px' : 'auto'}
      style:max-height={'300px'}
      style:overflow-y={useVirtualizedOptions ? 'hidden' : 'auto'}
      class:open-up={openUp}
      role="button" tabindex="0"
      onmousedown={(ev) => {
        // Keep focus on input while interacting with list content.
        ev.preventDefault()
        ev.stopPropagation()
        avoidBlur = true
      }}
      onmousemove={avoidhover
        ? (ev) => {
            ev.stopPropagation();
            if (avoidhover) {
              arrowSelected = -1;
              avoidhover = false;
            }
          }
        : undefined}
    >
      {#snippet optionRow(e: E, i: number)}
        {@const name = String(e[keyName])}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <div
          class="flex ai-center _highlight{arrowSelected === i ? ' _selected' : ''}{useDividingLine ? ' _with_divider' : ''}"
          role="option"
          aria-selected={arrowSelected === i}
          tabindex="0"
          title={name}
          onmousedown={(ev) => {
            // Commit selection before blur/click ordering can clear the input.
            ev.preventDefault();
            ev.stopPropagation();
            onOptionClick(e);
          }}
        >
          {#if optionRenderer}
            {@render optionRenderer(e, words)}
          {:else}
            {@const highlighted = highlString(name, words)}
            <div class="_option_text">
              {#each highlighted as w}
                <span class={w.highl ? "_8" : ""} class:mr-4={w.isEnd}>{w.text}</span>
              {/each}
            </div>
          {/if}
        </div>
      {/snippet}

      {#if useVirtualizedOptions}
        <SvelteVirtualList bind:this={vlRef} items={filteredOptions}>
          {#snippet renderItem(e, i)}
            {@render optionRow(e, i)}
          {/snippet}
        </SvelteVirtualList>
      {:else}
        {#each filteredOptions as e, i}
          {@render optionRow(e, i)}
        {/each}
      {/if}
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
    min-height: 36px;
    cursor: pointer;
    padding: 0 6px 0 6px;
    border-radius: 4px;
  }

  ._highlight._with_divider {
    border-bottom: 1px solid #ececec;
    border-radius: 0;
  }

  ._highlight._with_divider:last-child {
    border-bottom: none;
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

  ._option_text {
    min-width: 0;
    width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  ._8 {
    color: rgb(196, 71, 71);
    text-decoration: underline;
  }
  ._10 {
    color: #6d5dad;
  }

  .select-arrow {
    transition: transform 0.18s ease, color 0.18s ease;
    transform-origin: center;
    transform-style: preserve-3d;
  }

  .select-arrow.arrow-up {
    transform: perspective(120px) rotateX(180deg);
  }

  .select-arrow.arrow-down {
    transform: perspective(120px) rotateX(0deg);
  }

  .select-arrow.is-open {
    color: #3a3945;
  }
</style>
