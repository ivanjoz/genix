<script lang="ts">
  import { Core } from '$core/core/store.svelte'
  import { accesoAcciones, arrayToMapN, type IAcceso, type IPerfil } from "./perfiles-accesos.svelte"

  const accesoAccionesMap = arrayToMapN(accesoAcciones, 'id')

  let {
    acceso,
    perfilForm = $bindable(),
    isEdit,
    onEdit
  }: {
    acceso: IAcceso
    perfilForm?: IPerfil
    isEdit: boolean
    onEdit: () => void
  } = $props()

  const acciones = $derived(perfilForm?.accesosMap?.get(acceso.id) || [])
  
  const accionColor = $derived.by(() => {
    if (acciones.length === 0 || isEdit) return undefined
    const acciones_ = [...acciones].sort().reverse()
    const accion = accesoAccionesMap.get(acciones_[0] || 0)
    return accion?.color2 || accion?.color || ""
  })

  function handleCardClick(ev: MouseEvent) {
    if (Core.deviceType === 1 || !perfilForm) { return }
    ev.stopPropagation()
    const currentAcciones = perfilForm.accesosMap.get(acceso.id) || []
    if (currentAcciones.length >= acceso.acciones.length) {
      perfilForm.accesosMap.delete(acceso.id)
    } else {
      const missing = acceso.acciones.filter(x => !currentAcciones.includes(x))
      const newAcciones = [...currentAcciones, missing[0]].filter(x => x)
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
    if (isEdit) { className += " sel" }
    if (Core.deviceType > 1) { className += " mobile" }
    return className
  })
</script>

<div 
  class={cN}
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
  <div class="mr-4">{acceso.nombre}</div>
  
  {#if Core.deviceType > 1}
    <div class="line-1 p-abs" style:background-color={accionColor}></div>
  {/if}
  
  {#if isEdit}
    <div 
      class="absolute flex items-center justify-center i-edit"
      onclick={ev => {
        ev.stopPropagation()
        onEdit()
      }}
      role="button"
      tabindex="0"
      onkeydown={(ev) => {
        if (ev.key === 'Enter' || ev.key === ' ') {
          ev.preventDefault()
          onEdit()
        }
      }}
    >
      <i class="icon-pencil"></i>
    </div>
    <div class="absolute flex items-center justify-center a-id c-purple">{acceso.id}</div>
  {/if}
  
  {#if !isEdit && perfilForm}
    <div class="acciones-ac1">
      {#each acciones as id}
        {@const accion = accesoAccionesMap.get(id)}
        {#if accion}
          <div 
            class="bnc1"
            style:background-color={accion.color}
          >
            <i class={accion.icon}></i>
          </div>
        {/if}
      {/each}
    </div>
    <div class="acciones-ac2 w-full flex justify-center z-10">
      {#each acceso.acciones as id}
        {@const accion = accesoAccionesMap.get(id)}
        {@const selected = acciones.includes(id)}
        {#if accion}
          <div
            class="accion-btn"
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
            {accion.short}
          </div>
        {/if}
      {/each}
    </div>
  {/if}
</div>

<style>
  .acceso-card {
    height: 38px;
    position: relative;
    box-shadow: 0 1px 8px #5a5e6330;
    background-color: white;
    cursor: pointer;
    display: flex;
    padding: 3px 6px 3px 4px;
    align-items: center;
    line-height: 1.05;
    border-left: 4px solid transparent;
    user-select: none;
  }

  .acceso-card .acciones-ac2 {
    position: absolute;
    left: 0;
    bottom: 0;
  }

  .acceso-card.sel:hover {
    outline: 1px solid rgba(0, 0, 0, 0.54);
  }

  .acciones-ac2 {
    visibility: hidden;
  }

  .acceso-card .i-edit {
    background-color: transparent;
    top: 0;
    right: 0;
    height: 1.7rem;
    font-size: 0.94rem;
    border-radius: 0;
    width: 2.2rem;
    visibility: hidden;
  }

  .acceso-card:hover .i-edit {
    visibility: visible;
  }

  .acceso-card .i-edit:hover {
    background-color: black;
    color: white;
  }

  .acceso-card .a-id {
    background-color: transparent;
    top: 0;
    right: 0;
    height: 1.7rem;
    font-size: 0.94rem;
    border-radius: 0;
    width: 2rem;
  }

  .acceso-card:hover .a-id {
    visibility: hidden;
  }

  .acceso-card:hover:not(.mobile) .acciones-ac2 {
    visibility: visible;
  }

  .acceso-card:hover:not(.mobile) .acciones-ac1 {
    visibility: hidden;
  }

  .acceso-card.mobile {
    padding-top: 0.8rem;
    height: 3rem;
    border-left-width: 0;
  }

  .acceso-card .line-1 {
    height: 4px;
    width: 4rem;
    top: 3px;
    left: 4px;
  }

  .acciones-ac2 > div {
    height: 1.7rem;
    padding: 3px 7px 0 7px;
    margin: 0 2px;
    background-color: #eae8f9;
    border-radius: 7px 7px 0 0;
    min-width: 4rem;
    color: rgb(82, 77, 124);
    border-bottom: 2px solid #eae8f9;
    user-select: none;
  }

  .acciones-ac2 > div:hover {
    border-bottom-color: black;
  }

  .acceso-card .bnc1 {
    color: #fff;
    height: 1.25rem;
    display: inline-flex;
    padding: 0 3px;
    align-items: center;
    border-radius: 2px;
    margin: 0 2px;
    width: 1.6rem;
    justify-content: center;
  }

  /* Mobile specific styles */
  @media (max-width: 749px) {
    .acceso-card {
      width: calc((100% / 2) - 6px);
      margin: 4px 3px;
    }
    .acceso-card .acciones-ac1 {
      position: absolute;
      top: -5px;
      right: 4px;
    }
  }
</style>
