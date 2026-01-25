<script lang="ts" generics="T,E">
  import SvelteVirtualList from '@humanspeak/svelte-virtual-list';
  import { untrack } from "svelte";
  import { throttle } from "$core/lib/helpers";
  import { Core, WeakSearchRef } from '$core/core/store.svelte';
import { Popover2 } from '$components/popover2/Popover2';

	export interface ICellSelector<T,E> {
    id?: number | string;
		saveOn?: T;
		save?: keyof T;
		css?: string;
		contentClass?: string;
		inputClass?: string;
		onChange?: (newValue: string | number) => void;
		render?: (value: E, isEditing: boolean) => any;
		getValue?: (e: T) => number | string;
    options: E[]
    keyId: keyof E,
    keyName: keyof E,
		required?: boolean;
		type?: string;
	}

  let {
		id, options, saveOn, save, keyId, keyName, contentClass, required, render
	}: ICellSelector<T,E> = $props();

  let show = $derived(Core.popoverShowID === id);
  let refElement: HTMLDivElement | null = $state(null);
  let inputRef: HTMLInputElement | null = $state(null);
  let filterValue: string = $state("");
  let selected: E = $state(null as E)
  let avoidBlur = false

  const buildMap = () => {
    if(WeakSearchRef.has(options)){ return }
    const idToRecord: Map<string|number, E> = new Map()
    const valueToRecord: Map<string,E> = new Map()

    for(const e of options){
      idToRecord.set(e[keyId] as (string|number), e)
      valueToRecord.set((e[keyName] as string).toLowerCase(), e)
    }

    WeakSearchRef.set(options, { idToRecord, valueToRecord })
  }

  buildMap()

  let optionsFiltered = $derived.by(() => {
    console.log("filtrando valores::", filterValue)

    if(!filterValue){ return options }
    else {
      return options.filter(x => String(x[keyName]||"").toLowerCase().includes(filterValue))
    }
  })

  const clearValueIfNotFound = (target: HTMLInputElement) => {
    const valueToRecord = WeakSearchRef.get(options)?.valueToRecord || new Map()
    selected = valueToRecord.get(String(target.value||"").toLowerCase())
    if(selected){
      target.value = selected[keyName] as string
    } else {
      target.value = ""
    }
    if(saveOn && save){
      if(selected){ saveOn[save] = selected[keyId] as unknown as NonNullable<T>[keyof T] }
      else {
        delete saveOn[save]
      }
    }
  }

  const onSelect = (e: E) => {
    console.log("selected", e)
    selected = e
    if(inputRef){ inputRef.value = e[keyName] as string }
    if(saveOn && save){
      saveOn[save] = e[keyId] as unknown as NonNullable<T>[keyof T]
    }
    Core.popoverShowID = 0
  }

  $effect(() => {
    if(selected || !selected){
      console.log("selected 2",selected)
    }
	})

  const renderContent = $derived(
    render ? render(selected, show) : selected?.[keyName] || "" as string
  );

  const handlwShowClick = () => {
    filterValue = ""
    Core.popoverShowID = id as (number|string)
	}

  $effect(() => {
		if (show && inputRef) {
      if(selected){ inputRef.value = selected[keyName] as string }
			inputRef.focus();
		}
	})

  $effect(() => {
    if(saveOn && save){
      buildMap()
      untrack(() => {
        const idToRecord = WeakSearchRef.get(options)?.idToRecord || new Map()
        selected = saveOn[save] ? idToRecord.get(saveOn[save]) as E : null as E
        console.log("selected 3",selected)
        if (selected) {
          if(inputRef){ inputRef.value = selected[keyName] as string }
        } else {
          if(inputRef){ inputRef.value = "" }
        }
      })
    }
  })

</script>

<div class="_2">{renderContent}</div>
<div class="_1" bind:this={refElement}
  role="button"
  tabindex="0"
  onkeydown={(ev) => {
    if (ev.key === 'Enter' || ev.key === ' ') {
      ev.preventDefault();
    }
  }}
  onclick={ev => {
    ev.stopPropagation()
    handlwShowClick()
  }}
>
  <div class="{contentClass}"
    style:visibility={show ? 'hidden' : 'visible'}
  >
    { renderContent }
    {#if !selected && required}
      <i class="icon-attention text-red-500"></i>
    {/if}
  </div>
  {#if show}
    <input bind:this={inputRef} id="{String(id)}"
      onkeyup={ev => {
        const value = ((ev.target as HTMLInputElement).value||"").trim();
        throttle(() => {
          filterValue = String(value||"").toLowerCase()
        },200)
      }}
      onfocus={() => {
        // Core.popoverShowID = id
      }}
      onblur={ev => {
        if(avoidBlur){ avoidBlur = false; return }
        clearValueIfNotFound(ev.target as HTMLInputElement)
        Core.popoverShowID = 0
        filterValue = ""
      }}
    />
  {/if}
  <Popover2
    referenceElement={refElement}
    open={show}
    placement="bottom"
  >
    <div class="h-200 w-400 p-4 _4 overflow-auto">
      <SvelteVirtualList items={optionsFiltered}>
        {#snippet renderItem(item)}
          <div class="_3 min-h-28 px-8 py-4 flex items-center"
            role="button"
            tabindex="0"
            onclick={ev => {
              ev.stopPropagation()
              onSelect(item)
            }}
            onmousedown={() => {
              avoidBlur = true
            }}
            onkeydown={(ev) => {
              if (ev.key === 'Enter' || ev.key === ' ') {
                ev.preventDefault()
                onSelect(item)
              }
            }}
          >
            {item[keyName]}
          </div>
        {/snippet}
      </SvelteVirtualList>
    </div>
  </Popover2>
</div>

<style>
  ._1 {
		position: absolute;
		top: 0;
		left: 0;
		display: flex;
		align-items: center;
		padding: 0 6px;
		width: 100%;
		height: 100%;
		border: 1px solid transparent;
	}
  ._1:hover {
		border: 1px solid rgba(0, 0, 0, 0.596);
	}
	._1:focus-within {
		box-shadow: inset 0 0 0px 1px #dbc1ff;
    border-color: #b17bff;
		background-color: #f9f4ff;
	}
  ._1 > input:first-of-type {
		border: none;
		outline: none;
		position: absolute;
		top: 0;
		left: 0;
		height: 100%;
		width: 100%;
		padding-left: 6px;
	}
  ._2 {
		opacity: 0;
		pointer-events: none;
	}
  ._3 {
    user-select: none;
    cursor: pointer;
  }
  ._3:hover {
    background-color: rgb(244, 244, 244);
  }
  ._4 {
    background-color: rgb(255, 255, 255);
    border-radius: 7px;
    box-shadow: rgba(0, 0, 0, 0.24) 0px 3px 8px;
  }
  ._5 {

  }
</style>
