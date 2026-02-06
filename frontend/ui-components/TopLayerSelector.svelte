<script lang="ts">
import { Core } from '$core/store.svelte';
import { highlString, include, throttle } from '$libs/helpers';
    import { untrack } from 'svelte';

// Local state for this modal instance
const isOpen = $derived(!!Core.showMobileSearchLayer);
let avoidClose = false
let htmlTextarea: HTMLTextAreaElement | undefined
let searchText = $state("")

const keyName = $derived(Core.showMobileSearchLayer?.keyName) as string
const keyId = $derived(Core.showMobileSearchLayer?.keyID) as string
const searchWords = $derived(searchText.toLowerCase().split(" ").filter(x => x.length > 1))

const optionsFiltered = $derived.by(() => {
  if((searchText||"").trim() === ""){
    console.log("Enviando options::", Core.showMobileSearchLayer?.options)
    return Core.showMobileSearchLayer?.options || []
  } else {
    const filtered = []
    for (const opt of Core.showMobileSearchLayer?.options || []){
      const name = opt[keyName] as string
      if (typeof name === "string") {
        if(include(name.toLowerCase(), searchWords)){
          filtered.push(opt)
        }
      }
    }
    console.log("Enviando filtered::", searchText, filtered)
    return filtered
  }
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
<div class="_1" class:_2={isOpen}
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
    <button class="ml-auto h-40 w-40 shrink-0 _5" aria-label="Buscar"
      onclick={ev => {
        Core.showMobileSearchLayer = null
      }}
    >
      <i class="icon-cancel h1"></i>
    </button>
  </div>
  <div class="_6">
    {#each optionsFiltered as opt }
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <div class="_7" onclick={() => {
        Core.showMobileSearchLayer?.onSelect(opt)
        Core.showMobileSearchLayer = null
      }}>
        <div class="w-full">
          {#each highlString(opt[keyName], searchWords) as w}
            <span class={w.highl ? "_8" : ""}>{w.text}</span>
          {/each}
        </div>
      </div>
    {/each}
  </div>
</div>

<style>
  ._1 { /* background */
    width: 100vw;
    min-height: 50vh;
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

  ._6 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    grid-gap: 4px 8px;
    overflow: auto;
    max-height: 64vh;
    border-radius: 0 0 24px 24px;
    padding: 8px;
  }

  ._7 {
    display: flex;
    color: #fff;
    min-height: 2.5rem;
    border-radius: 7px;
    line-height: 1.1;
    text-align: center;
    padding: 2px 4px;
    border: 1px solid #ffffff36;
    outline: 2px solid #00000080;
    background-color: #00000036;
    align-items: center;
    justify-content: center;
  }
  ._8 {
    color: #ffe98c;
    font-style: normal;
    text-decoration: underline;
  }
</style>
