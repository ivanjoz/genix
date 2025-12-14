<script lang="ts" generics="T">
    import { untrack } from "svelte";
  import s1 from "./components.module.css";
  import { type ElementAST } from "./micro/Renderer.svelte";

  export interface IInput<T> {
    id?: number;
    saveOn: T;
    save: keyof T;
    label?: string;
    css?: string;
    inputCss?: string;
    required?: boolean;
    validator?: (v: string | number) => boolean;
    type?: string;
    placeholder?: string;
    disabled?: boolean;
    onChange?: () => void;
    postValue?: string | ElementAST[];
    baseDecimals?: number;
    content?: string | any;
    transform?: (v: string | number) => string | number;
    useTextArea?: boolean;
    rows?: number;
    dependencyValue?: number | string
  }

  const {
    id,
    saveOn = $bindable(),
    save,
    label,
    css,
    inputCss,
    required,
    validator,
    type,
    placeholder,
    disabled,
    onChange,
    postValue,
    baseDecimals,
    content,
    transform,
    useTextArea,
    rows,
    dependencyValue
  }: IInput<T> = $props();

  /*
  // Shared reactive state
  let inputUpdater = $state(new Map<number, number>());

  export function refreshInput(ids: number[]) {
    const map = new Map(inputUpdater);
    for (let id of ids) {
      const currentValue = map.get(id) || 0;
      map.set(id, currentValue + 1);
    }
    inputUpdater = map;
  }
  */

  const baseDecimalsValue = baseDecimals ? 10 ** baseDecimals : 0;

  const makeValue = (value: number) => {
    if(typeof value !== 'number'){ return value || "" }
    return baseDecimals ? (value as number / baseDecimals) : value
  }

  const checkIfInputIsValid = (): number => {
    if (!required || disabled) return 0;
    if (!saveOn || !save) return 1;
    const value = saveOn[save] as string | number;

    let pass = !required;
    if (validator) {
      pass = validator(value);
    } else {
      if (value || value === 0) pass = true;
    }
    return pass ? 2 : 1;
  }

  let inputValue = $state("" as string | number);
  let isInputValid = $state(checkIfInputIsValid());
  let isChange = 0;

  const onKeyUp = (ev: KeyboardEvent | FocusEvent, isBlur?: boolean) => {
    ev.stopPropagation();
    const target = ev.target as HTMLInputElement | HTMLTextAreaElement;
    let value: string | number = target.value;

    if (type === "number") {
      if (!isBlur && !value && (ev as KeyboardEvent).key === "-") return;
      if (isNaN(value as unknown as number)) {
        value = undefined as any;
      } else {
        value = parseFloat(value as string);
      }
    }

    if (transform && isBlur) {
      value = transform(value);
    }

    untrack(() => {
      if (saveOn && save) {
        let valueSaved = value
        if(baseDecimalsValue && typeof valueSaved === 'number'){
          valueSaved = Math.round(valueSaved * baseDecimalsValue);
        }
        saveOn[save] = valueSaved as NonNullable<T>[keyof T];
        isInputValid = checkIfInputIsValid();
      }
    })

    if(!isBlur){ isChange = 1 }
    inputValue = value;
  }

  const iconValid = () => {
    if (!isInputValid) return null;
    else if (isInputValid === 2)
      return `<i class="v-icon icon-ok c-green"></i>`;
    else if (isInputValid === 1)
      return `<i class="v-icon icon-attention c-red"></i>`;
    return null;
  }

  let lastSaveOn: T | undefined

  const doSave = () => {
    untrack(() => {
      const v = saveOn[save]
      inputValue = typeof v === "number" ? v : (v as string) || ""
      if(baseDecimalsValue && typeof inputValue === 'number'){ 
        inputValue = inputValue / baseDecimalsValue 
      }
      isInputValid = checkIfInputIsValid()
    })
  }

  $effect(() => {
    if(!saveOn || !save){ return }
    if(lastSaveOn === saveOn){ return }
    lastSaveOn = saveOn

    if(saveOn[save] !== inputValue){ doSave() }
  })

  $effect(() => {
    if(dependencyValue){ doSave() }
  })

  let cN = $derived(
    `${s1.input} p-rel${css ? ` ${css}` : ""}${!label ? " no-label" : ""}`,
  )
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
    {#if useTextArea}
      <textarea class={`w-full ${s1.input_inp} ${inputCss || ""}`}
        bind:value={inputValue}
        placeholder={placeholder || ""}
        {disabled}
        {rows}
        onkeyup={ev => { onKeyUp(ev) }}
        onblur={ev => {
          console.log("input saveon:",$state.snapshot(saveOn))
          onKeyUp(ev, true);
          if (onChange && isChange) {
            onChange();
            isChange = 0;
          }
        }}
      ></textarea>
    {:else}
      <input class="w-full {s1.input_inp} {inputCss || ""}"
        bind:value={inputValue}
        type={type || "search"}
        placeholder={placeholder || ""}
        {disabled}
        onkeyup={ev => { onKeyUp(ev) }}
        onblur={(ev) => {
          onKeyUp(ev, true);
          if (onChange && isChange) {
            onChange();
            isChange = 0;
          }
        }}
      />
    {/if}

    {#if !label}
      {@html iconValid() || ""}
    {/if}
    {#if postValue}
      <div class="{s1.input_post_value}">{postValue}</div>
    {/if}
  </div>
</div>
