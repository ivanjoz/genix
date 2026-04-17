<script lang="ts">
  interface Props {
    title?: string
    show?: boolean
    closedHeightPx?: number
    onToggle?: (nextState: boolean) => void
    children?: import('svelte').Snippet
  }

  let {
    title = '',
    show = false,
    closedHeightPx = 64,
    onToggle,
    children,
  }: Props = $props()

  // Keep the interaction explicit so the parent owns the open/close state.
  const toggleLayer = () => {
    onToggle?.(!show)
  }
</script>

<div class="mobile-layer-shell" aria-hidden={!show}>
  <button
    class="mobile-layer-backdrop"
    class:is-visible={show}
    aria-label="Cerrar panel"
    onclick={toggleLayer}
  ></button>

  <div
    class="mobile-layer-panel"
    class:is-open={show}
    style={`--mobile-layer-closed-height:${closedHeightPx}px;`}
  >
    <button
      class="mobile-layer-header"
      aria-expanded={show}
      aria-label={show ? 'Ocultar panel' : 'Mostrar panel'}
      onclick={toggleLayer}
    >
      <div class="mobile-layer-handle"></div>
      <div class="mobile-layer-title-row">
        <span class="mobile-layer-title">{title}</span>
        <i class={`icon-${show ? 'down-open' : 'up-open'} mobile-layer-icon`}></i>
      </div>
    </button>

    <div class="mobile-layer-body">
      {@render children?.()}
    </div>
  </div>
</div>

<style>
  .mobile-layer-shell {
    position: fixed;
    inset: 0;
    z-index: var(--layer-zindex);
    pointer-events: none;
  }

  .mobile-layer-backdrop {
    position: absolute;
    inset: 0;
    border: none;
    background: rgb(15 23 42 / 0.22);
    opacity: 0;
    transition: opacity 220ms ease;
    pointer-events: none;
  }

  .mobile-layer-backdrop.is-visible {
    opacity: 1;
    pointer-events: auto;
  }

  .mobile-layer-panel {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    height: calc(100vh - var(--header-height) - 8px);
    background: white;
    border-radius: 18px 18px 0 0;
    box-shadow: 0 -10px 26px rgb(15 23 42 / 0.22);
    overflow: hidden;
    pointer-events: auto;
    transform: translateY(calc(100% - var(--mobile-layer-closed-height)));
    transition: transform 320ms cubic-bezier(.23,.21,.64,.97);
    will-change: transform;
  }

  .mobile-layer-panel.is-open {
    transform: translateY(0);
  }

  .mobile-layer-header {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 8px 16px 12px;
    border: none;
    border-bottom: 1px solid rgb(226 232 240);
    background: linear-gradient(180deg, rgb(248 250 252) 0%, rgb(255 255 255) 100%);
    text-align: left;
  }

  .mobile-layer-handle {
    width: 44px;
    height: 5px;
    margin: 0 auto;
    border-radius: 999px;
    background: rgb(148 163 184);
  }

  .mobile-layer-title-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .mobile-layer-title {
    font-size: 15px;
    font-weight: 700;
    color: rgb(51 65 85);
  }

  .mobile-layer-icon {
    font-size: 16px;
    color: rgb(71 85 105);
  }

  .mobile-layer-body {
    height: calc(100% - 58px);
    overflow: auto;
    background: white;
  }

  @media (min-width: 750px) {
    .mobile-layer-shell {
      display: none;
    }
  }
</style>
