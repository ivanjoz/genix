<script lang="ts">
import { Core } from '$core/store.svelte';
import { highlString, wordInclude, throttle } from '$libs/helpers';
import SvelteVirtualList from '@humanspeak/svelte-virtual-list';
import { untrack } from 'svelte';

// Local state for this modal instance
const isOpen = $derived(!!Core.showMobileSearchLayer);
const canClearSelection = $derived(!!Core.showMobileSearchLayer?.onClear)
let avoidClose = false
let htmlTextarea: HTMLTextAreaElement | undefined
let searchText = $state("")
const mobileColumns = 2
const estimatedRowHeight = 60

const keyName = $derived(Core.showMobileSearchLayer?.keyName) as string
const keyId = $derived(Core.showMobileSearchLayer?.keyID) as string
const searchWords = $derived(searchText.toLowerCase().split(" ").filter(x => x.length > 1))

const optionsFiltered = $derived.by(() => {
  if((searchText||"").trim() === ""){
    return Core.showMobileSearchLayer?.options || []
  } else {
    const filtered: any[] = []
    for (const opt of Core.showMobileSearchLayer?.options || []){
      const name = opt[keyName] as string
      if (typeof name === "string") {
        if(wordInclude(name.toLowerCase(), searchWords)){
          filtered.push(opt)
        }
      }
    }
    return filtered
  }
})

const virtualRows = $derived.by(() => {
  const optionRows: Array<Array<any | null>> = []

  // Group items in fixed rows so the virtualizer only tracks vertical movement.
  for (let optionIndex = 0; optionIndex < optionsFiltered.length; optionIndex += mobileColumns) {
    const rowOptions = optionsFiltered.slice(optionIndex, optionIndex + mobileColumns)
    optionRows.push(Array.from({ length: mobileColumns }, (_, columnIndex) => rowOptions[columnIndex] || null))
  }

  return optionRows
})

$effect(() => {
  if(isOpen){
    untrack(() => {
      if(htmlTextarea){
        htmlTextarea.focus()
        htmlTextarea.value = ""
      }
      searchText = ""
    })
  } else {
    if(htmlTextarea){
      htmlTextarea.blur()
      htmlTextarea.value = ""
    }
  }
})

</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="_1" class:_2={isOpen} data-button-layer-protected="true"
  onmousedown={ev => {
    if((ev.target as HTMLDivElement).tagName === "textarea"){ return }
    avoidClose = true
  }}
>
  <div class="flex items-center p-6">
    <i class="icon-search _3"></i>
    <textarea rows={1} class="h-38 _4" bind:this={htmlTextarea}
      onkeyup={ev => {
        throttle(() => {
          searchText = ((ev.target as any).value||"").toLowerCase()
        },150)
      }}
      onblur={() => {
        if(avoidClose){
          avoidClose = false
          htmlTextarea?.focus()
        } else {
          Core.showMobileSearchLayer = null
        }
      }}
    >
    </textarea>
    <div class="ml-8 flex shrink-0 items-center gap-8">
      {#if canClearSelection}
        <button class="h-40 w-40 shrink-0 _5 _5b" aria-label="Limpiar selección"
          onclick={() => {
            Core.showMobileSearchLayer?.onClear?.()
            Core.showMobileSearchLayer = null
          }}
        >
          <i class="icon-ccw h1 _5c"></i>
        </button>
      {/if}
      <button class="h-40 w-40 shrink-0 _5" aria-label="Cerrar selector"
        onclick={() => {
          Core.showMobileSearchLayer = null
        }}
      >
        <i class="icon-cancel h1"></i>
      </button>
    </div>
  </div>
  <div class="_6">
    <SvelteVirtualList
      items={virtualRows}
      defaultEstimatedItemHeight={estimatedRowHeight}
      bufferSize={4}
      viewportClass="_6_viewport"
      itemsClass="_6_items"
    >
      {#snippet renderItem(optionRow, rowIndex)}
        <div class="_6_row">
          {#each optionRow as opt, columnIndex (`${rowIndex}-${columnIndex}`)}
            {#if opt}
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <div class="_7" onclick={() => {
                Core.showMobileSearchLayer?.onSelect(opt)
                Core.showMobileSearchLayer = null
              }}>
                <div class="_9">
                  {#each highlString(String(opt[keyName] || ''), searchWords) as w}
                    <span class={w.highl ? "_8" : ""}>{w.text}</span>
                  {/each}
                </div>
              </div>
            {:else}
              <div class="_7 _7_empty"></div>
            {/if}
          {/each}
        </div>
      {/snippet}
    </SvelteVirtualList>
  </div>
</div>

<style>
  ._1 { /* background */
    width: 100vw;
    max-height: calc(70vh - 8px);
    background-color: #000000b3;
    overflow-y: auto;
    overflow-x: hidden;
    z-index: 300;
    position: fixed;
    top: 0;
    backdrop-filter: blur(8px);
    border-radius: 0 0 16px 16px;
    border-bottom: 6px solid rgba(0, 0, 0, .25);
    opacity: 0;
    pointer-events: none;
  }
  
  ._6 {
    height: calc(70vh - 68px);
    max-height: calc(70vh - 68px);
    border-radius: 0 0 24px 24px;
    padding: 0;
    overflow: hidden;
    min-height: 0;
  }
  
  ._1._2 {
    opacity: 1;
    z-index: 299;
    pointer-events: all;
  }
  ._3 {
    position: absolute;
    left: 14px;
    color: #ffffff82;
    margin-bottom: 2px;
  }
  ._4 {
    width: calc(100% - 60px);
    border-radius: 18px;
    color: #fff;
    font-size: 18px;
    margin-bottom: 2px;
    background-color: #464672d4;
    border: 1px solid transparent;
    padding-top: calc(.5rem + 2px);
    line-height: 1;
    appearance: none;
    outline: none;
    border-radius: 9px;
    resize: none;
    padding-left: 34px;
  }
  ._4:focus {
    border: 1px solid rgb(117, 118, 214);
    outline: 1px solid rgb(150, 152, 255);
    outline-offset: 0;
  }
  ._5 {
    border-radius: 50%;
    background-color: #e06868;
    color: #fff;
    border: none;
    outline: none;
  }

  ._5b {
    background-color: #30303d;
  }

  ._5c {
    color: #ff9a3d;
  }

  ._6 :global(.virtual-list-container) {
    height: 100%;
    min-height: 0;
    padding: 6px;
    padding-bottom: 0;
    box-sizing: border-box;
  }

  ._6 :global(._6_viewport) {
    height: 100%;
    min-height: 0;
    overflow-y: auto;
    overflow-x: hidden;
    padding: 2px;
    box-sizing: border-box;
  }

  ._6 :global(._6_items) {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-width: 0;
    padding: 2px 2px 10px 2px;
    box-sizing: border-box;
  }

  ._6_row {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
    min-width: 0;
  }

  ._7 {
    display: flex;
    color: #fff;
    min-height: 2.5rem;
    max-height: 80px;
    border-radius: 7px;
    line-height: 1.1;
    text-align: center;
    padding: 6px 4px;
    border: 1px solid #ffffff36;
    outline: 2px solid #00000080;
    background-color: #00000036;
    align-items: center;
    justify-content: center;
    min-width: 0;
    overflow: hidden;
  }

  ._7_empty {
    visibility: hidden;
    pointer-events: none;
  }

  ._9 {
    width: 100%;
    min-width: 0;
    overflow: hidden;
    word-break: break-word;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    text-overflow: ellipsis;
  }
  ._8 {
    color: #ffe98c;
    font-style: normal;
    text-decoration: underline;
  }
</style>
