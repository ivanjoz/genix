<script lang="ts" generics="T">
import { Core } from '$core/core/store.svelte';


  let { 
    options, selected, keyId, keyName, buttonCss, onSelect, css, useMobileGrid
  }: { 
    options: T[], 
    selected: number,
    keyId?: keyof T,
    keyName?: keyof T,
    buttonCss?: string,
    onSelect: (e: T) => void,
    useMobileGrid?: boolean
    css?: string
  } = $props()

  const getClass = (e: T) => {
    let cn = ""
    const id = Array.isArray(e) ? e[0] : (keyId ? e?.[keyId] : 0) || 0
    if(id === selected){ cn += " _3" }
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

</script>

<div class="_1 pb-4 md:pb-0 flex items-center shrink-0 max-w-[100%] overflow-x-auto overflow-y-hidden {css}"
  class:_5={useMobileGrid}
  class:grid-cols-2={useMobileGrid && options.length === 2}
  class:grid-cols-3={useMobileGrid && options.length === 3}
  class:grid-cols-4={useMobileGrid && options.length === 4}
  class:grid-cols-5={useMobileGrid && options.length === 5}
>
  {#each options as opt }
    {@const words = getValue(opt)}
    <button class="flex items-center ff-bold _2 {getClass(opt)}" onclick={ev => {
      ev.stopPropagation()
      onSelect(opt)
    }}>
      {#if words.length === 1}
        <span>{words[0]}</span>
      {:else}
        <span>{words[0]}<br/>{words[1]}</span>
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
      word-break: break-all;
      align-items: center;
    }
  }
</style>