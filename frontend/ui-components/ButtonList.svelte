<script lang="ts">
  import { parseSVG } from '$libs/helpers';
  import angleSvg from '$domain/assets/angle.svg?raw';

  interface ActionItem {
    id: number;
    name: string;
    icon: string;
    handler: () => void;
  }

  let {
    name = '',
    icon = '',
    css = '',
    items = [],
  }: {
    name?: string;
    icon?: string;
    css?: string;
    items: ActionItem[];
  } = $props();
</script>

<div class="bl-wrapper">
  <button class="bx-purple {css}" type="button">
    {#if icon}<i class={icon}></i>{/if}
    {#if name}<span>{name}</span>{/if}
  </button>

  <!-- Dropdown: hidden by default, shown on parent hover via CSS -->
  <div class="bl-dropdown">
    <div class="bl-angle">
      <img class="bl-angle-img" alt="" src={parseSVG(angleSvg)} />
    </div>
    <div class="bl-content">
      {#each items as item (item.id)}
        <button class="bl-item" type="button" onclick={item.handler}>
          <i class={item.icon}></i>
          <span>{item.name}</span>
        </button>
      {/each}
    </div>
  </div>
</div>

<style>
  .bl-wrapper {
    position: relative;
    display: inline-block;
  }

  .bl-dropdown {
    position: absolute;
    top: calc(100% + 6px);
    right: 0;
    z-index: 360;
    min-width: 160px;
    background-color: white;
    border-radius: 8px;
    outline: 4px solid #4d447424;
    border: 1px solid #4d447452;
    box-shadow:
      0 4px 6px -1px rgba(0, 0, 0, 0.1),
      0 2px 4px -1px rgba(0, 0, 0, 0.06);
    opacity: 0;
    visibility: hidden;
    pointer-events: none;
    transform: translateY(-4px);
    transition: opacity 0.15s ease, transform 0.15s ease, visibility 0.15s;
  }

  /* Invisible bridge filling the gap so hover doesn't break when moving to the dropdown */
  .bl-wrapper::after {
    content: '';
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    height: 6px;
  }

  /* Show dropdown when hovering over the wrapper (button, gap bridge, or dropdown itself) */
  .bl-wrapper:hover .bl-dropdown {
    opacity: 1;
    visibility: visible;
    pointer-events: auto;
    transform: translateY(0);
  }

  /* Triangle pointer aligned to the right of the dropdown (above the button) */
  .bl-angle {
    position: absolute;
    top: -18px;
    right: 10px;
    overflow: hidden;
    height: 18px;
    width: 24px;
    display: flex;
    justify-content: center;
    z-index: 361;
  }

  .bl-angle-img {
    width: 24px;
    height: 24px;
    margin-top: 2px;
    filter: drop-shadow(0 -1px 1px rgba(0, 0, 0, 0.05));
  }

  .bl-content {
    padding: 4px 0;
  }

  .bl-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 8px 14px;
    background: none;
    border: none;
    cursor: pointer;
    text-align: left;
    color: inherit;
    white-space: nowrap;
  }

  .bl-item:hover {
    background-color: #f3f0ff;
  }

  .bl-item i {
    color: #6b7280;
  }
</style>
