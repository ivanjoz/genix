<script lang="ts" generics="T">
import { Core, tr } from '$core/store.svelte';
import { Env } from '$core/env';
import { Agent } from '$components/agent/registry';


  let {
    options, selected, keyId, keyName, buttonCss, onSelect, css, useMobileGrid,
    activeClass = "_3",
    inactiveClass = "",
    itemCss = "_2",
    containerCss = "_1"
  }: {
    options: T[],
    selected: any,
    keyId?: keyof T,
    keyName?: keyof T,
    buttonCss?: string,
    onSelect: (e: T) => void,
    useMobileGrid?: boolean,
    css?: string,
    activeClass?: string,
    inactiveClass?: string,
    itemCss?: string,
    containerCss?: string
  } = $props()

  const getClass = (e: T) => {
    let cn = itemCss
    const id = Array.isArray(e) ? e[0] : (keyId ? e?.[keyId] : e)
    if(id === selected){ 
      cn += " " + activeClass 
    } else {
      cn += " " + inactiveClass
    }
    if(buttonCss){ cn += " " + buttonCss }
    return cn
  }

  const getValue = (e: T): string[] => {
    if(Array.isArray(e)){
      if(Core.deviceType === 3 && Array.isArray(e[2])){
        return e[2]
      }
      return [e[1] as string]
    } else if(keyName){
      return [e[keyName] as string]
    } else {
      return [""]
    }
  }

  const getOptionId = (e: T): string | number => {
    if (Array.isArray(e)) { return e[0] as string | number }
    if (keyId) { return e[keyId] as string | number }
    return e as unknown as string | number
  }

  const componentID = Env.getComponentID()

  $effect(() => {
    return Agent.register({
      id: componentID,
      type: "OptionsStrip",
      label: "",
      select: (...ids) => {
        if (ids.length === 0) { return }
        // Options carry composite ids "<stripID>:<optID>" so the agent
        // addresses them directly via select on each Option. Strip the
        // prefix to recover the option id; bare ids still work for any
        // legacy caller.
        const raw = String(ids[0])
        const colon = raw.lastIndexOf(":")
        const targetId = colon >= 0 ? raw.slice(colon + 1) : raw
        const matched = options.find((opt) => String(getOptionId(opt)) === targetId)
        if (matched) { onSelect(matched) }
      },
    })
  })
</script>

<div data-id="OptionsStrip:{componentID}"
  class="{containerCss} pb-4 md:pb-0 flex items-center shrink-0 max-w-[100%] overflow-x-auto overflow-y-hidden {css}"
  class:_5={useMobileGrid}
  class:grid-cols-2={useMobileGrid && options.length === 2}
  class:grid-cols-3={useMobileGrid && options.length === 3}
  class:grid-cols-4={useMobileGrid && options.length === 4}
  class:grid-cols-5={useMobileGrid && options.length === 5}
>
  {#each options as opt }
    {@const words = getValue(opt)}
    {@const optId = getOptionId(opt)}
    {@const isSelected = optId === selected}
    <button data-id="Option:{componentID}:{optId}"
      data-selected={isSelected ? "true" : undefined}
      class="flex items-center ff-bold _2 {getClass(opt)}" onclick={ev => {
      ev.stopPropagation()
      onSelect(opt)
    }}>
      {#if opt && typeof opt === 'object' && 'icon' in opt}
        <span class="mr-2">{opt.icon}</span>
      {/if}
      {#if words.length === 1}
        <span>{tr(words[0])}</span>
      {:else}
        <span>{tr(words[0])}<br/>{tr(words[1])}</span>
      {/if}
    </button>
  {/each}
</div>

<style>
  ._1 > ._2:not(:last-of-type) {
    margin-right: 6px;
  }
  ._2 {
    padding: 4px 6px 4px 6px;
    min-width: 114px;
    color: #9d9dac;
    border-bottom: 4px solid rgba(0, 0, 0, 0.1);
    user-select: none;
    cursor: pointer;
    display: flex;
    text-align: center;
    justify-content: center;
    align-items: end;
    line-height: 1;
  }
  ._3 {
    color: #4343ad;
    border-bottom: 4px solid rgb(117 108 233);
  }

  @media (max-width: 750px) {
    ._1._5 {
      display: grid;
    }
    ._2 {
      padding: 0 2px 0 2px;
      height: 38px;
      min-width: 86px;
      /* word-break: break-all; */
      align-items: center;
    }
  }
</style>
