<script lang="ts" generics="T,E">
    import { untrack } from 'svelte';

  const {
		 saveOn = $bindable(), save, css, label
	}: {
    saveOn?: T
		save?: keyof T
    css?: string
    label?: string
  } = $props();

  let isSelected = $state(false)

  const onSelect = () => {
    isSelected = !isSelected
    if(saveOn && save){
      saveOn[save] = isSelected as NonNullable<T>[keyof T]
    }
  }

  let lastSaveOn: T | undefined

  $effect(() => {
    if(!saveOn || !save){ return }
    if(lastSaveOn === saveOn){ return }
    lastSaveOn = saveOn

    untrack(() => {
      isSelected = !!saveOn[save]
    })
  })

</script>

<div class="flex items-center {css}">
  <button class="flex mr-4 pt-1 items-center p-0 lh-10 justify-center rounded-[4px] shrink-0 w-28 h-26 _1" 
    class:_2={isSelected}
    aria-label="{label as string}"
    onclick={ev => {
      onSelect()
    }}
  >        
    {#if isSelected}
      <i class="icon-ok"></i>
    {/if}
  </button>
  <!-- svelte-ignore a11y_label_has_associated_control -->
  <label>{label as string}</label>
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
</style>