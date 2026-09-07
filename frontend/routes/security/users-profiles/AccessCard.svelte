<script lang="ts">
import { arrayToMapN } from '$libs/helpers';
import { useUI } from '@genix/ui';
import Checkbox from '$components/form/Checkbox.svelte';
  import { accesoAcciones, type IAccess, type IProfile } from "./users-profiles.svelte"
  import { isSubAccesoChecked, toggleSubAcceso } from "./users-profiles"
  const ui = useUI()

  const accesoAccionesMap = arrayToMapN(accesoAcciones, 'id')

  let {
    acceso,
    perfilForm = $bindable(),
    hideSubAccesos = false
  }: {
    acceso: IAccess
    perfilForm?: IProfile
    /** Drop the sub-access row when something else on the page owns it — the user layer edits
        sub-accesses in its own section, not inside a dropdown option. */
    hideSubAccesos?: boolean
  } = $props()

  // One nivel per access, not a list: the merge that builds a user's grants keeps only the highest,
  // so a list only ever let the editor express something the backend immediately discarded.
  const nivel = $derived(perfilForm?.accesosMap?.get(acceso.id) || 0)
  const accion = $derived(accesoAccionesMap.get(nivel))

  const accionColor = $derived(accion?.color2 || accion?.color || undefined)

  const cardIsSelected = $derived(nivel > 0)

  const subAccesosSelected = $derived(perfilForm?.subAccesosMap?.get(acceso.id) || [])

  // Only while the access itself is granted: a sub-access qualifies a permission rather than
  // granting one, so offering it on an access nobody holds would be offering nothing. The
  // subAccesosMap check is what keeps the row out of the user layer, which grants levels only.
  const showSubAccesos = $derived(
    !hideSubAccesos && !!perfilForm?.accesosMap && !!perfilForm?.subAccesosMap
      && acceso.subAccesos.length > 0 && cardIsSelected
  )

  const declaredSubAccesoIDs = $derived(acceso.subAccesos.map(subAcceso => subAcceso.id))

  function handleSubAccesoToggle(subAccesoID: number) {
    if (!perfilForm?.subAccesosMap) { return }

    const nextSubAccesoIDs = toggleSubAcceso(subAccesosSelected, subAccesoID, declaredSubAccesoIDs)
    nextSubAccesoIDs.length > 0
      ? perfilForm.subAccesosMap.set(acceso.id, nextSubAccesoIDs)
      : perfilForm.subAccesosMap.delete(acceso.id)
    // Force reactivity
    perfilForm.subAccesosMap = new Map(perfilForm.subAccesosMap)
  }

  function handleCardClick(ev: MouseEvent) {
    if (ui.state.deviceType === 1 || !perfilForm) { return }
    ev.stopPropagation()
    if (perfilForm.accesosMap.has(acceso.id)) {
      perfilForm.accesosMap.delete(acceso.id)
    } else {
      // A card click is a fast toggle: it grants the widest level the access declares.
      const widestNivel = [...acceso.acciones].sort((leftLevel, rightLevel) => rightLevel - leftLevel)[0]
      if (widestNivel) { perfilForm.accesosMap.set(acceso.id, widestNivel) }
    }
    // Force reactivity
    perfilForm.accesosMap = new Map(perfilForm.accesosMap)
  }

  // The levels are a one-of choice, so clicking a level selects it and clicking the selected one
  // revokes the access — which is the only way to give the row an "off" state.
  function handleAccionClick(ev: MouseEvent, id: number) {
    ev.stopPropagation()
    if (!perfilForm?.accesosMap) return

    if (perfilForm.accesosMap.get(acceso.id) === id) {
      perfilForm.accesosMap.delete(acceso.id)
    } else {
      perfilForm.accesosMap.set(acceso.id, id)
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

<div class="acceso-cell">
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
        {#if accion}
          <i class={accion.icon} style:color={accion.color} title={accion.name}></i>
        {/if}
      </div>
    {/if}
    <div class="acciones-ac2 flex justify-start z-10">
      {#each acceso.acciones as id}
        {@const nivelAccion = accesoAccionesMap.get(id)}
        {@const selected = nivel === id}
        {#if nivelAccion}
          <div
            class="accion-btn"
            title={nivelAccion.name}
            aria-label={nivelAccion.name}
            style:background-color={selected ? nivelAccion.color : undefined}
            style:border-color={selected ? nivelAccion.color : undefined}
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
            <i class={nivelAccion.icon}></i>
          </div>
        {/if}
      {/each}
    </div>
  {/if}
</div>

<!-- A row under the card, not inside it: the card is deliberately as tall as its two text lines,
     and only one access in the catalog declares sub-accesses today. -->
{#if showSubAccesos}
  <div class="sub-accesos-row text-[13px]">
    {#each acceso.subAccesos as subAcceso}
      <Checkbox
        size="tiny"
        label={subAcceso.name}
        checked={isSubAccesoChecked(subAccesosSelected, subAcceso.id)}
        underlineOnHover={true}
        onToggle={() => handleSubAccesoToggle(subAcceso.id)}
      />
    {/each}
  </div>
{/if}
</div>

<style>
  .acceso-cell {
    display: flex;
    flex-direction: column;
  }

  .sub-accesos-row {
    display: flex;
    flex-wrap: wrap;
    gap: 4px 14px;
    padding: 5px 8px 3px 9px;
    border: 1px solid #d9dce8;
    border-top: none;
    background: #fcfbff;
    color: #4b4f66;
    line-height: 1.15;
  }

  .acceso-card {
    position: relative;
    background-color: white;
    cursor: pointer;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: 6px;
    /* 9px on the left, not 13: the ::before colour bar takes the first 4px inside the card, so
       anything wider than that plus a normal gutter read as dead space on an ungranted card. */
    padding: 6px 8px 6px 9px;
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

  /* The eye is the one icon from MDI: its glyph fills 62% of its 24×24 box where the FontAwesome
     ones fill theirs edge to edge, so next to the shield it reads a size smaller. A transform, not
     a bigger box — the buttons keep their size and nothing shifts. */
  i[class*="mdi--eye"] {
    transform: scale(1.25);
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
