<script lang="ts">
  export interface IICardArrowStepsOption {
    id: number;
    name: string;
    icon?: string;
  }

import arrow2Svg from '$domain/assets/flecha_fin.svg';
import arrow1Svg from '$domain/assets/flecha_inicio.svg';
import { cn, parseSVG } from '$libs/helpers';
import { Core } from '$core/store.svelte'
import s1 from './styles.module.css';

  let {
    options,
    onSelect = () => {},
    selected,
    optionRender,
    columnsTemplate,
  }: {
    options: IICardArrowStepsOption[];
    onSelect?: (e: IICardArrowStepsOption) => void;
    selected?: number;
    optionRender?: (e: IICardArrowStepsOption) => any;
    columnsTemplate?: string;
  } = $props();

  function handleSelect(option: IICardArrowStepsOption) {
    onSelect(option);
  }

  $effect(() => {
    console.log("Ecommerce changed::", Core.ecommerce.cartOption, "|", selected);
  });

  // Use a derived state or a getter for reactive values
  const gridTemplateColumns = $derived(
    columnsTemplate || options.map(() => "1fr").join(" "),
  );
</script>

<div class="grid mr-8" style:grid-template-columns={gridTemplateColumns}>
  {#each options as option (option.id)}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div onclick={() => handleSelect(option)}
      class={cn(
        "flex relative items-center",
        s1.card_arrow_ctn,
        option.id === selected && s1.card_arrow_ctn_selected,
      )}
    >
      <img class="h-full {s1.card_arrow_svg}" src={parseSVG(arrow1Svg)} alt="" />
      <div class="h-full flex items-center justify-center {s1.card_arrow_name}">
        {#if optionRender}
          {@render optionRender(option)}
        {:else}
          <div class="ff-semibold">{option.name}</div>
        {/if}
      </div>
      <img class="h-full {s1.card_arrow_svg}" src={parseSVG(arrow2Svg)} alt="" />
      <div class={s1.card_arrow_line}></div>
    </div>
  {/each}
</div>
