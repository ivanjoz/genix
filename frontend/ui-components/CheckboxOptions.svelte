<script lang="ts" generics="T,E">
    import { untrack } from 'svelte';

  const {
		options, saveOn = $bindable(), save, keyId, keyName, css, type, useButtons = false
	}: {
    saveOn?: T
		save?: keyof T
    options?: E[]
    keyId: keyof E
    keyName: keyof E
    css?: string
    type?: "single" | "multiple"
    useButtons?: boolean
  } = $props();

  let optionsSelected: (number|string)[] = $state([])

  const onSelect = (e: E) => {
    const id = e[keyId] as number|string
    if(type == 'multiple'){
      if(optionsSelected.includes(id)){
        optionsSelected = optionsSelected.filter(x => x !== id)
      } else {
        optionsSelected.push(id)
      }
    } else {
      if(optionsSelected.includes(id)){
        optionsSelected = []
      } else {
        optionsSelected = [id]
      }
    }

    if(saveOn && save){
      if(type === 'multiple'){
        saveOn[save] = optionsSelected as NonNullable<T>[keyof T]
      } else {
        saveOn[save] = (optionsSelected[0] || undefined) as NonNullable<T>[keyof T]
      }
    }
  }

  let lastSaveOn: T | undefined

  $effect(() => {
    if(!saveOn || !save){ return }
    if(lastSaveOn === saveOn){ return }
    lastSaveOn = saveOn

    untrack(() => {
      if(type === 'multiple'){
        optionsSelected = (saveOn[save] || []) as (number|string)[]
      } else {
        optionsSelected = [(saveOn[save] || []) as (number|string)]        
      }
    })
  })

</script>

<div class="flex {css}">
  {#each options as opt }
  {@const isSelected = optionsSelected.includes(opt[keyId] as (number|string))}
    {#if useButtons}
      <button class="_button ff-semibold mr-10"
        class:_buttonSelected={isSelected}
        aria-label={opt[keyName] as string}
        onclick={ev => {
          ev.stopPropagation()
          onSelect(opt)
        }}
      >
        {opt[keyName] as string}
      </button>
    {:else}
      <div class="flex items-center mr-10">
        <button class="flex mr-4 pt-1 items-center p-0 lh-10 justify-center rounded-[4px] shrink-0 w-28 h-26 _1" 
          class:_2={isSelected}
          aria-label={opt[keyName] as string}
          onclick={ev => {
            ev.stopPropagation()
            onSelect(opt)
          }}
        >        
          {#if isSelected}
            <i class="icon-ok"></i>
          {/if}
        </button>
        <!-- svelte-ignore a11y_label_has_associated_control -->
        <label>{opt[keyName] as string}</label>
      </div>
    {/if}
  {/each}

</div>

<style>
  ._1 {
    background-color: var(--white);
    border: 1px solid rgb(143, 143, 143);
    color: white;
  }
  ._1._2 {
    background-color: #09cb70;
    border-color: #19965b;
  }
  ._1:hover {
    border: 2px solid #0987eb;
  }
  ._1._2:hover {
    border: 2px solid #61778b;
    background-color: #98aec5;
  }

  ._button {
    background-color: var(--white);
    opacity: 0.8;
    border-radius: 8px;
    min-height: 30px;
    padding: 0 8px;
    box-shadow: rgba(0, 0, 0, 0.16) 0px 1px 3px;
    border: 1px solid transparent;
    font-size: 15px;
    line-height: 1;
  }

  ._buttonSelected {
		opacity: 1;
	  outline: 1px solid #bc91ffcf;
	  box-shadow: rgb(151 112 242 / 70%) 0px 2px 1px;
	  background-color: #f7f2ff;
	  color: #6f42b8;
	  border: 1px solid #ece1ff;
  }
</style>
