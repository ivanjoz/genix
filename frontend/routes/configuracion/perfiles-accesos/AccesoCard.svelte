<script lang="ts">
import { arrayToMapN } from '$libs/helpers';
import { Core } from '$core/store.svelte';
  import { accesoAcciones, type IAcceso, type IPerfil } from "./perfiles-accesos.svelte"

  const accesoAccionesMap = arrayToMapN(accesoAcciones, 'id')

  let {
    acceso,
    perfilForm = $bindable()
  }: {
    acceso: IAcceso
    perfilForm?: IPerfil
  } = $props()

  const acciones = $derived(perfilForm?.accesosMap?.get(acceso.id) || [])

  const accionColor = $derived.by(() => {
    if (acciones.length === 0) return undefined
    const acciones_ = [...acciones].sort().reverse()
    const accion = accesoAccionesMap.get(acciones_[0] || 0)
    return accion?.color2 || accion?.color || ""
  })

  const cardIsSelected = $derived((acciones?.length || 0) > 0)

  function handleCardClick(ev: MouseEvent) {
    if (Core.deviceType === 1 || !perfilForm) { return }
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
    if (!perfilForm) return

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
    if (Core.deviceType > 1) { className += " mobile" }
    return className
  })
</script>

<div
  class={cN}
  class:is-selected={cardIsSelected}
  style:border-left-color={accionColor}
  onclick={handleCardClick}
  role="button"
  tabindex="0"
  onkeydown={(ev) => {
    if (ev.key === 'Enter' || ev.key === ' ') {
      ev.preventDefault()
      handleCardClick(ev as any)
    }
  }}
>
  <div class="content-wrap">
    <div class="title-row">
      <div class="title-text">{acceso.nombre}</div>
    </div>
    {#if acceso.descripcion}
      <div class="route-text">{acceso.descripcion}</div>
    {/if}
  </div>

  {#if Core.deviceType > 1}
    <div class="line-1 p-abs" style:background-color={accionColor}></div>
  {/if}

  {#if perfilForm}
    <div class="acciones-ac2 w-full flex justify-start z-10">
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
    min-height: 68px;
    position: relative;
    background-color: white;
    cursor: pointer;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: 6px;
    padding: 8px;
    line-height: 1.15;
    border: 1px solid #d9dce8;
    border-left: 4px solid transparent;
    border-radius: 10px;
    user-select: none;
    transition: border-color 0.12s ease, outline-color 0.12s ease, background-color 0.12s ease;
  }

  .acceso-card:hover {
    outline: 1px solid #9ca3af;
    outline-offset: 0;
  }

  .acceso-card.is-selected {
    border-color: #b4a3ff;
    outline: 1px solid #7c5cff;
    outline-offset: 0;
    background: #f8f6ff;
  }

  .acceso-card .content-wrap {
    width: 100%;
    min-height: 0;
  }

  .acceso-card .title-row {
    display: flex;
    align-items: flex-start;
    justify-content: flex-start;
  }

  .acceso-card .title-text {
    font-weight: 700;
    font-size: 13px;
    color: #20243b;
  }

  .acceso-card .route-text {
    margin-top: 4px;
    font-size: 11px;
    color: #6b7280;
    overflow-wrap: anywhere;
    line-height: 1.15;
  }

  .acceso-card .acciones-ac2 {
    position: static;
    flex-wrap: wrap;
    gap: 6px;
  }

  .acciones-ac2 {
    visibility: visible;
  }

  .acceso-card.mobile {
    padding-top: 0.6rem;
    min-height: 4.25rem;
    border-left-width: 0;
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
    font-size: 11px;
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
