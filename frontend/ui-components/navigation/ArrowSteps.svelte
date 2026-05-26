<script lang="ts">
  export interface IICardArrowStepsOption {
    id: number;
    name: string;
    icon?: string;
  }

import arrow2Svg from '$components/assets/flecha_fin.svg?raw';
import arrow1Svg from '$components/assets/flecha_inicio.svg?raw';
import { cn, parseSVG } from '$libs/helpers';
import { Core } from '$core/store.svelte'
import { Env } from '$core/env';
import { Agent } from '$components/agent/registry';

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

  const componentID = Env.getComponentID()

  $effect(() => {
    return Agent.register({
      id: componentID,
      type: "ArrowSteps",
      label: "",
      select: (...ids) => {
        if (ids.length === 0) { return }
        const targetId = String(ids[0])
        const matched = options.find((opt) => String(opt.id) === targetId)
        if (matched) { handleSelect(matched) }
      },
    })
  })
</script>

<div data-id="ArrowSteps:{componentID}" class="grid mr-8" style:grid-template-columns={gridTemplateColumns}>
  {#each options as option (option.id)}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div data-id="Option:{option.id}"
      data-selected={option.id === selected ? "true" : undefined}
      onclick={() => handleSelect(option)}
      class={cn(
        "flex relative items-center",
        "card_arrow_ctn",
        option.id === selected && "card_arrow_ctn_selected",
      )}
    >
      <img class="h-full card_arrow_svg" src={parseSVG(arrow1Svg)} alt="" />
      <div class="h-full flex items-center justify-center card_arrow_name">
        {#if optionRender}
          {@render optionRender(option)}
        {:else}
          <div class="ff-semibold">{option.name}</div>
        {/if}
      </div>
      <img class="h-full card_arrow_svg" src={parseSVG(arrow2Svg)} alt="" />
      <div class="card_arrow_line"></div>
    </div>
  {/each}
</div>

<style>
  .card_arrow_ctn {
    height: 2.8rem;
    margin-right: -4px;
    width: calc(100% + 4px);
    cursor: pointer;
  }
  .card_arrow_name {
    background-color: #dfdfdf;
    min-width: 5rem;
    text-align: center;
    overflow: visible;
    z-index: 5;
    padding: 0 6px;
    flex-grow: 1;
    max-width: 100%;
    overflow: hidden;
  }
  .card_arrow_ctn > img:last-of-type {
    margin-right: -4px;
  }
  .card_arrow_line {
    height: 4px;
    width: calc(100% - 9px);
    position: absolute;
    bottom: -4px;
    left: 0;
    background-color: #0cad66;
    visibility: hidden;
  }
  .card_arrow_svg {
    filter: invert(100%) sepia(0%) saturate(5883%) hue-rotate(164deg) brightness(120%) contrast(75%);
  }

  .card_arrow_ctn:hover .card_arrow_line {
   visibility: visible;
  }
  .card_arrow_ctn:hover .card_arrow_svg {
    filter: invert(91%) sepia(5%) saturate(633%) hue-rotate(100deg) brightness(102%) contrast(95%);
  }
  .card_arrow_ctn:hover .card_arrow_name {
    background-color: #d5efe3;
    color: #147e50;
  }
  .card_arrow_ctn_selected .card_arrow_line {
    background-color: #14945c;
  }
  .card_arrow_ctn.card_arrow_ctn_selected .card_arrow_svg {
    filter: invert(49%) sepia(53%) saturate(3777%) hue-rotate(125deg) brightness(95%) contrast(91%);
  }
  .card_arrow_ctn.card_arrow_ctn_selected .card_arrow_name {
    background-color: #0cad66;
    color: white;
  }

  /* Use a literal breakpoint so Lightning CSS can minify this scoped block safely. */
  @media only screen and (max-width: 740px) {
    .card_arrow_name {
      padding: 0;
      font-size: var(--fs2);
      word-break: break-all;
      min-width: unset;
    }
  }
</style>
