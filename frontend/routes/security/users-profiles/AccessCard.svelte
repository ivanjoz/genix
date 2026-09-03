<script lang="ts">
import { arrayToMapN } from '$libs/helpers';
import { useUI } from '@genix/ui';
  import { accesoAcciones, type IAccess, type IProfile } from "./users-profiles.svelte"
  const ui = useUI()

  const accesoAccionesMap = arrayToMapN(accesoAcciones, 'id')

  let {
    acceso,
    perfilForm = $bindable()
  }: {
    acceso: IAccess
    perfilForm?: IProfile
  } = $props()

  const acciones = $derived(perfilForm?.accesosMap?.get(acceso.id) || [])

  const accionColor = $derived.by(() => {
    if (acciones.length === 0) return undefined
    const acciones_ = [...acciones].sort().reverse()
    const accion = accesoAccionesMap.get(acciones_[0] || 0)
    return accion?.color2 || accion?.color || ""
  })

  const cardIsSelected = $derived((acciones?.length || 0) > 0)

  // Highest-level-first, so the badge reads "Todo, Ver" and not the insertion order.
  const selectedAcciones = $derived(
    [...acciones].sort((leftLevel, rightLevel) => rightLevel - leftLevel)
      .map(id => accesoAccionesMap.get(id))
      .filter(accion => !!accion)
  )

  function handleCardClick(ev: MouseEvent) {
    if (ui.state.deviceType === 1 || !perfilForm) { return }
    ev.stopPropagation()
    const currentAcciones = perfilForm.accesosMap.get(acceso.id) || []
    if (currentAcciones.length > 0) {
      perfilForm.accesosMap.delete(acceso.id)
    } else {
      // A card click should behave like a fast toggle, promoting to full access when that level exists.
      const preferredAccessLevel = acceso.acciones.includes(7)
        ? 7
        : [...acceso.acciones].sort((leftLevel, rightLevel) => rightLevel - leftLevel)[0]
      const newAcciones = preferredAccessLevel ? [preferredAccessLevel] : []
      perfilForm.accesosMap.set(acceso.id, newAcciones)
    }
    // Force reactivity
    perfilForm.accesosMap = new Map(perfilForm.accesosMap)
  }

  function handleAccionClick(ev: MouseEvent, id: number) {
    ev.stopPropagation()
    if (!perfilForm?.accesosMap) return

    let newAcciones = [...(perfilForm.accesosMap.get(acceso.id) || [])]
    if (newAcciones.includes(id)) {
      newAcciones = newAcciones.filter(x => x !== id)
    } else {
      newAcciones.push(id)
    }
    newAcciones.sort((a, b) => b - a)

    if (newAcciones.length === 0) {
      perfilForm.accesosMap.delete(acceso.id)
    } else {
      perfilForm.accesosMap.set(acceso.id, newAcciones)
    }
    // Force reactivity
    perfilForm.accesosMap = new Map(perfilForm.accesosMap)
  }

  const cN = $derived.by(() => {
    let className = "acceso-card"
    if (ui.state.deviceType > 1) { className += " mobile" }
    return className
  })
</script>

<div
  class={cN}
  class:is-selected={cardIsSelected}
  style:--access-color={accionColor}
  onclick={handleCardClick}
  role="button"
  tabindex="0"
  aria-label={`Access control card for ${acceso.nombre}`}
  onkeydown={(ev) => {
    if (ev.key === 'Enter' || ev.key === ' ') {
      ev.preventDefault()
      handleCardClick(ev as any)
    }
  }}
>
  <div class="content-wrap">
    <div class="title-row">
      <div class="title-text text-[15px] ff-semibold">{acceso.nombre}</div>
    </div>
    {#if acceso.descripcion}
      <div class="route-text text-[13px]">{acceso.descripcion}</div>
    {/if}
  </div>

  {#if ui.state.deviceType > 1}
    <div class="line-1 absolute" style:background-color={accionColor}></div>
  {/if}

  {#if perfilForm?.accesosMap}
    {#if cardIsSelected}
      <!-- Compact granted-level badge: replaced by the action buttons while the card is hovered. -->
      <div class="acciones-badge">
        {#each selectedAcciones as accion}
          <i class={accion.icon} style:color={accion.color} title={accion.name}></i>
        {/each}
      </div>
    {/if}
    <div class="acciones-ac2 flex justify-start z-10">
      {#each acceso.acciones as id}
        {@const accion = accesoAccionesMap.get(id)}
        {@const selected = acciones.includes(id)}
        {#if accion}
          <div
            class="accion-btn"
            title={accion.name}
            aria-label={accion.name}
            style:background-color={selected ? accion.color : undefined}
            style:border-color={selected ? accion.color : undefined}
            style:color={selected ? 'white' : undefined}
            onclick={ev => handleAccionClick(ev, id)}
            role="button"
            tabindex="0"
            onkeydown={(ev) => {
              if (ev.key === 'Enter' || ev.key === ' ') {
                ev.preventDefault()
                handleAccionClick(ev as any, id)
              }
            }}
          >
            <i class={accion.icon}></i>
          </div>
        {/if}
      {/each}
    </div>
  {/if}
</div>

<style>
  .acceso-card {
    position: relative;
    background-color: white;
    cursor: pointer;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: 6px;
    padding: 6px 8px 6px 13px;
    line-height: 1.15;
    border: 1px solid #d9dce8;
    user-select: none;
  }

  /* Flush against the outer edge, full height, square ends — no border miter. */
  .acceso-card::before {
    content: '';
    position: absolute;
    top: -1px;
    bottom: -1px;
    left: -1px;
    width: 5px;
    background-color: var(--access-color, transparent);
  }

  .acceso-card:hover {
    outline: 1px solid #9ca3af;
    outline-offset: 0;
  }

  .acceso-card.is-selected {
    background: #f8f6ff;
  }

  .acceso-card .content-wrap {
    width: 100%;
    min-height: 0;
    /* Keeps the text clear of the badge sitting in the top-right corner. */
    padding-right: 46px;
  }

  .acceso-card .title-row {
    display: flex;
    align-items: flex-start;
    justify-content: flex-start;
  }

  .acceso-card .title-text {
    color: #20243b;
  }

  .acceso-card .route-text {
    margin-top: 2px;
    color: #6b7280;
    overflow-wrap: anywhere;
    line-height: 1.15;
  }

  .acciones-badge {
    position: absolute;
    top: 5px;
    right: 7px;
    display: flex;
    gap: 3px;
  }

  .acciones-badge i {
    height: 14px;
    width: 14px;
  }

  /* Desktop: the buttons float over the card centre-right and only on hover, so the
     card itself only has to be as tall as its two text lines. */
  .acceso-card:not(.mobile) .acciones-ac2 {
    position: absolute;
    top: 50%;
    right: 5px;
    transform: translateY(-50%);
    gap: 4px;
    opacity: 0;
    pointer-events: none;
    padding-left: 10px;
    border-radius: 10px;
    background-color: white;
    box-shadow: -10px 0 10px -4px white;
  }

  .acceso-card.is-selected:not(.mobile) .acciones-ac2 {
    background-color: #f8f6ff;
    box-shadow: -10px 0 10px -4px #f8f6ff;
  }

  .acceso-card:not(.mobile):hover .acciones-ac2 {
    opacity: 1;
    pointer-events: auto;
  }

  .acceso-card:not(.mobile):hover .acciones-badge {
    opacity: 0;
  }

  .acceso-card:not(.mobile) .acciones-ac2 > div {
    height: 1.7rem;
    min-width: 42px;
    padding: 0 10px;
    font-size: 15px;
  }

  .acceso-card.mobile {
    padding: 0.6rem 8px 6px 8px;
    min-height: 4.25rem;
  }

  /* Touch keeps the horizontal .line-1 marker instead. */
  .acceso-card.mobile::before {
    display: none;
  }

  .acceso-card.mobile .acciones-ac2 {
    position: static;
    width: 100%;
    flex-wrap: wrap;
    gap: 6px;
  }

  /* On touch there is no hover, so the badge would just duplicate the buttons. */
  .acceso-card.mobile .acciones-badge {
    display: none;
  }

  .acceso-card .line-1 {
    height: 4px;
    width: 2.5rem;
    top: 3px;
    left: 4px;
  }

  .acciones-ac2 > div {
    height: 1.55rem;
    min-width: 50px;
    padding: 0 10px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background-color: #eae8f9;
    border-radius: 10px;
    color: rgb(82, 77, 124);
    border: 1px solid #eae8f9;
    user-select: none;
    font-size: 13px;
  }

  .acciones-ac2 > div:hover {
    border-color: black;
  }

  /* Mobile specific styles */
  @media (max-width: 749px) {
    .acceso-card {
      width: 100%;
      margin: 0;
    }
  }
</style>
