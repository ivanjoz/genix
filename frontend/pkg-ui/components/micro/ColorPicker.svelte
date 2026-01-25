<script lang="ts" generics="T">
  import { Core } from '$core/core/store.svelte';
  import ColorPicker from 'svelte-awesome-color-picker';
  import s1 from "$components/components.module.css";
    import { untrack } from 'svelte';

	let {
		saveOn = $bindable(),
		save,
		css,
		contentClass = '',
		onChange,
    label
	}: {
    saveOn: T
		save?: keyof T
		css?: string
		contentClass?: string
    label?: string
		onChange?: (newValue: string | number) => void
  } = $props();

  let cN = $derived(`${s1.input} p-rel` + (css ? " " + css : ""));

  // Initialize with white color
  let currentColor = $state('#FFFFFF')
  let hasInit = false

  const setColor = (hexColor?: string) => {
    if(saveOn && save){
      saveOn[save] = hexColor as NonNullable<T>[keyof T]
    }
    if(hexColor) {
      currentColor = hexColor;
    }
  }

  $effect(() => {
    if(saveOn || save){
      console.log("change save on:",$state.snapshot(saveOn))
      untrack(() => {
        currentColor = (saveOn[save as keyof T] as string) || '#FFFFFF'
      })
      hasInit = true
    }
  })

</script>

<div class={cN}>
  {#if label}
  <div class={s1.input_lab_cell_left}><div></div></div>
    <div class={s1.input_lab}>
      {label}
    </div>
    <div class={s1.input_lab_cell_right}><div></div></div>
  {/if}
  <div class={s1.input_shadow_layer}>
    <div></div>
  </div>

  <div class={`_1 ${s1.input_div} flex items-center justify-center w-full`}>
    <div class={s1.input_div_1}>
      <div></div>
    </div>
    <div class="w-full flex items-center justify-center">
      <ColorPicker isAlpha={false} textInputModes={[]}
        position="responsive"
        hex={currentColor}
        onInput={color => {
          if(!hasInit){ return }
          console.log("Setting color:", color.hex)
          setColor(color.hex as string)
        }}
      />
    </div>
  </div>
</div>

<style>
  ._1 {
    --slider-width: 24px;
  }
  ._1 :global(.color-picker .color) {
    border-radius: 0;
    height: calc(var(--input-height) - 16px);
    width: 54px;
    border: 2px solid rgba(0, 0, 0, 0.8);
    margin-bottom: 4px;
  }

  ._1 :global(.color-picker label) {
    font-size: 0;
    line-height: 0;
  }

  ._1 :global(.color-picker label .container) {
    font-size: initial;
    line-height: initial;
  }

</style>