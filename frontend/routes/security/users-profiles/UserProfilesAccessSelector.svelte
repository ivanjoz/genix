<script lang="ts">
  import { untrack } from 'svelte';
  import SearchDualCard from '$components/cards/SearchDualCard.svelte';
  import AccessCard from './AccessCard.svelte';
  import Checkbox from '$components/form/Checkbox.svelte';
  import T from '$components/misc/T.svelte';
  import { accesoAcciones } from './users-profiles.svelte';
  import type { IAccessGroupCatalogEntry, IAccessListCatalogEntry } from './access-list-catalog';
  import type { IAccess, IProfile, IUser } from './users-profiles.svelte';
  import {
    buildAccesosCatalog,
    buildRouteCatalogIndex,
    buildSubAccesoOptions,
    fromAccesoGrants,
    isSubAccesoChecked,
    SUB_ACCESO_TODOS_ID,
    SUB_ACCESO_TODOS_NAME,
    toAccesoGrants,
    toggleSubAcceso
  } from './users-profiles';

  interface IProfileAccessSummary {
    readableAccessNames: string[]
    editableAccessNames: string[]
    // "<access>: <sub-access>", because a bare "Recibir Pago" says nothing about what it qualifies.
    subAccessLabels: string[]
  }

  interface IUserProfilesAccessSelectorProps {
    saveOn: IUser
    perfiles: IProfile[]
    accessGroupEntries: IAccessGroupCatalogEntry[]
    accessCatalogEntries: IAccessListCatalogEntry[]
    accessCatalogLoadError?: string
    css?: string
  }

  let {
    saveOn = $bindable(),
    perfiles,
    accessGroupEntries,
    accessCatalogEntries,
    accessCatalogLoadError = '',
    css = ''
  }: IUserProfilesAccessSelectorProps = $props();

  const routeCatalogIndex = buildRouteCatalogIndex();
  const accesosCatalog = $derived(buildAccesosCatalog(accessCatalogEntries, routeCatalogIndex));

  // AccessCard edits a profile's two maps. A user stores the very same AccesosGrants list, so the
  // layer edits through a profile-shaped object and re-derives the list on every change — one
  // component, one editing shape, on both tabs. The subAccesosMap is what makes AccessCard offer
  // the sub-access row here, which a user record can now store.
  const accessEditForm = $state({
    accesosMap: new Map<number, number>(),
    subAccesosMap: new Map<number, number[]>(),
  } as IProfile);

  // Down-sync keys off the form's identity, not its contents: UsersTab replaces the whole object
  // when a layer opens, and reacting to AccesosGrants instead would fight the up-sync below.
  $effect(() => {
    const editedUser = saveOn;
    untrack(() => {
      const { accesosMap, subAccesosMap } = fromAccesoGrants(editedUser?.AccesosGrants);
      accessEditForm.accesosMap = accesosMap;
      accessEditForm.subAccesosMap = subAccesosMap;
    });
  });

  $effect(() => {
    const nextAccesosGrants = toAccesoGrants(accessEditForm.accesosMap, accessEditForm.subAccesosMap);
    untrack(() => {
      if (!saveOn) { return; }
      // Compared as JSON because a grant is a small object with an optional nested list: writing on
      // every tick would restart the down-sync's sibling effects for no change.
      if (JSON.stringify(saveOn.AccesosGrants || []) === JSON.stringify(nextAccesosGrants)) { return; }
      saveOn.AccesosGrants = nextAccesosGrants;
    });
  });

  const accesosCatalogByID = $derived(new Map(accesosCatalog.map((accesoRecord) => [accesoRecord.id, accesoRecord])));

  // One chip per granted access. The chip is a read-only summary; the level and the sub-accesses
  // are both edited in the dropdown above.
  const grantedAccesoIDs = $derived.by(() => {
    return [...accessEditForm.accesosMap.keys()]
      .filter((accesoID) => accesosCatalogByID.has(accesoID))
      .sort((leftAccesoID, rightAccesoID) => {
        const leftName = accesosCatalogByID.get(leftAccesoID)?.nombre || '';
        const rightName = accesosCatalogByID.get(rightAccesoID)?.nombre || '';
        return leftName.localeCompare(rightName);
      });
  });

  // Picking an access out of the list (keyboard, or the product's agent) has no level attached, so
  // it grants the widest one the access declares — the same shortcut AccessCard's card click takes.
  function grantWidestNivel(acceso: IAccess) {
    if (accessEditForm.accesosMap.has(acceso.id)) { return; }

    const widestNivel = [...acceso.acciones].sort((leftNivel, rightNivel) => rightNivel - leftNivel)[0];
    if (!widestNivel) { return; }

    accessEditForm.accesosMap.set(acceso.id, widestNivel);
    // Force reactivity
    accessEditForm.accesosMap = new Map(accessEditForm.accesosMap);
  }

  // Only the granted accesses that actually declare sub-accesses have anything to show, which today
  // is one access in the whole catalog — hence a section that is usually absent entirely.
  const accesosConSubAccesos = $derived(
    grantedAccesoIDs
      .map((accesoID) => accesosCatalogByID.get(accesoID))
      .filter((acceso): acceso is IAccess => !!acceso && acceso.subAccesos.length > 0)
  );

  function toggleSubAccesoOnAcceso(accesoID: number, subAccesoID: number) {
    const declaredSubAccesoIDs = (accesosCatalogByID.get(accesoID)?.subAccesos || []).map((subAcceso) => subAcceso.id);
    const nextSubAccesoIDs = toggleSubAcceso(
      accessEditForm.subAccesosMap.get(accesoID) || [],
      subAccesoID,
      declaredSubAccesoIDs
    );
    nextSubAccesoIDs.length > 0
      ? accessEditForm.subAccesosMap.set(accesoID, nextSubAccesoIDs)
      : accessEditForm.subAccesosMap.delete(accesoID);
    // Force reactivity
    accessEditForm.subAccesosMap = new Map(accessEditForm.subAccesosMap);
  }

  function revokeAcceso(accesoID: number) {
    accessEditForm.accesosMap.delete(accesoID);
    // The sub-accesses go with it: they qualify a permission that no longer exists.
    accessEditForm.subAccesosMap.delete(accesoID);
    // Force reactivity
    accessEditForm.accesosMap = new Map(accessEditForm.accesosMap);
    accessEditForm.subAccesosMap = new Map(accessEditForm.subAccesosMap);
  }

  const accessCatalogNameByID = $derived(new Map(accessCatalogEntries.map((entry) => [entry.id, entry.name])));

  // Keyed by the stored wire shape, accesoID * 100 + subAccesoID, so a profile's SubAccesos list
  // resolves to names without re-deriving the encoding here.
  const subAccessLabelByRef = $derived.by(() => {
    const labelByRef = new Map<number, string>();

    for (const accessCatalogEntry of accessCatalogEntries) {
      const subAccesoOptions = buildSubAccesoOptions(
        accessCatalogEntry.sub_accesses_ids,
        accessCatalogEntry.sub_accesses_names
      );
      if (subAccesoOptions.length === 0) { continue; }

      // "Todos" is never a checkbox, but it is what a full selection is stored as, so a profile
      // holding it still has to read as something here.
      labelByRef.set(
        accessCatalogEntry.id * 100 + SUB_ACCESO_TODOS_ID,
        `${accessCatalogEntry.name}: ${SUB_ACCESO_TODOS_NAME}`
      );

      for (const subAccesoOption of subAccesoOptions) {
        labelByRef.set(
          accessCatalogEntry.id * 100 + subAccesoOption.id,
          `${accessCatalogEntry.name}: ${subAccesoOption.name}`
        );
      }
    }

    return labelByRef;
  });

  function summarizeProfileAccesses(profileRecord?: IProfile): IProfileAccessSummary {
    const readableAccessNames = new Set<string>();
    const editableAccessNames = new Set<string>();

    if (!profileRecord?.accesosMap) {
      return { readableAccessNames: [], editableAccessNames: [], subAccessLabels: [] };
    }

    // Split profile permissions into view and edit buckets for the custom card renderer.
    for (const [accessID, accessLevel] of profileRecord.accesosMap) {
      const accessName = accessCatalogNameByID.get(accessID);
      if (!accessName) { continue; }

      // A level above VER implies VER, so a profile granting TODO reads as both.
      readableAccessNames.add(accessName);
      if (accessLevel > 1) {
        editableAccessNames.add(accessName);
      }
    }

    // Sub-accesses are flags inside an access, not a third level, so they get their own row
    // instead of being folded into the view/edit split.
    const subAccessLabels = new Set<string>();
    for (const [accessID, subAccesoIDs] of profileRecord.subAccesosMap || []) {
      for (const subAccesoID of subAccesoIDs) {
        const subAccessLabel = subAccessLabelByRef.get(accessID * 100 + subAccesoID);
        if (subAccessLabel) { subAccessLabels.add(subAccessLabel); }
      }
    }

    return {
      readableAccessNames: [...readableAccessNames].sort((leftName, rightName) => leftName.localeCompare(rightName)),
      editableAccessNames: [...editableAccessNames].sort((leftName, rightName) => leftName.localeCompare(rightName)),
      subAccessLabels: [...subAccessLabels].sort((leftLabel, rightLabel) => leftLabel.localeCompare(rightLabel))
    };
  }

  function getProfileRecord(profileID: number): IProfile | undefined {
    return perfiles.find((profileRecord) => profileRecord.ID === profileID);
  }

  const accesoAccionesMap = new Map(accesoAcciones.map((accesoAccion) => [accesoAccion.id, accesoAccion]));

  // One icon per granted access, not one per level: the widest level is the one that decides what
  // the user can actually do, so a card holding VER + TODO reads as TODO.
  function getWidestAccion(accesoID: number) {
    return accesoAccionesMap.get(accessEditForm.accesosMap.get(accesoID) || 0);
  }
</script>

<!-- The card owns every click inside the dropdown: preventDefault keeps focus on the search input
     (so the list stays open across several grants) and stopPropagation keeps the row from
     "selecting" an option that has no single level to select. -->
{#snippet accessOptionCard(acceso: IAccess)}
  <div
    class="_access-option"
    role="presentation"
    onmousedown={(ev) => {
      ev.preventDefault();
      ev.stopPropagation();
    }}
  >
    <AccessCard {acceso} perfilForm={accessEditForm} hideSubAccesos={true} />
  </div>
{/snippet}

<!-- Contents only, no wrapper: the chip element itself is the card (see `._chip-right` below). The
     level owns the whole left edge — its icon on top, its colour bar filling the rest — which frees
     the text to run the full width of the card. Levels are edited in the list above; the only
     action here is the chip's own trash button. -->
{#snippet grantedAccessCard(acceso: IAccess)}
  {@const widestAccion = getWidestAccion(acceso.id)}
  <div class="_granted-side">
    {#if widestAccion}
      <i class={`${widestAccion.icon} _granted-icon`} style:color={widestAccion.color} title={widestAccion.name}></i>
    {/if}
    <div class="_granted-line" style:background-color={widestAccion?.color2 || widestAccion?.color}></div>
  </div>
  <div class="_granted-text">
    <div class="text-[15px] ff-semibold _granted-name">{acceso.nombre}</div>
    {#if acceso.descripcion}
      <div class="text-[13px] _granted-route">{acceso.descripcion}</div>
    {/if}
  </div>
{/snippet}

<div class={css}>
  <SearchDualCard
    bind:saveOn
    saveLeft="ProfileIDs"
    css="col-span-24"
    cardCss="mt-8"
    leftOptions={perfiles}
    leftKeyId="ID"
    leftKeyName="Name"
    leftLabel="PERFILES ::"
    rightOptions={accesosCatalog}
    rightKeyId="id"
    rightKeyName="nombre"
    rightLabel="ACCESOS ::"
    rightOptionsCss="w-[calc(150%+7.5px)] max-md:w-full md:-ml-[calc(50%+7.5px)]"
    columns2={2}
    render2={accessOptionCard}
    rightChipIDs={grantedAccesoIDs}
    onRightChipRemove={(accesoID) => revokeAcceso(Number(accesoID))}
    onRightSelect={grantWidestNivel}
    placeholderAsLabel={true}
  >
    {#snippet selectedItem(selectedAccessOrProfile)}
      {#if selectedAccessOrProfile.source === 'left'}
        {@const selectedProfileRecord = getProfileRecord(Number(selectedAccessOrProfile.id))}
        {@const selectedProfileAccessSummary = summarizeProfileAccesses(selectedProfileRecord)}
        <div class="_selected-profile-card">
          <div class="_selected-profile-name text-[15px] ff-semibold text-sky-700">
            {selectedProfileRecord?.Name || ''}
          </div>
          {#if selectedProfileAccessSummary.readableAccessNames.length > 0}
            <div class="_selected-profile-row">
              <i class="icon-[mdi--eye] _selected-profile-icon pt-2 text-blue-600"></i>
              <span class="text-sm">{selectedProfileAccessSummary.readableAccessNames.join(', ')}</span>
            </div>
          {/if}
          {#if selectedProfileAccessSummary.editableAccessNames.length > 0}
            <div class="_selected-profile-row">
              <i class="icon-[fa--pencil] _selected-profile-icon _selected-profile-icon-edit pt-2 text-red-600"></i>
              <span class="text-sm">{selectedProfileAccessSummary.editableAccessNames.join(', ')}</span>
            </div>
          {/if}
          {#if selectedProfileAccessSummary.subAccessLabels.length > 0}
            <div class="_selected-profile-row">
              <i class="icon-[fa--shield] _selected-profile-icon pt-2 text-purple-600"></i>
              <span class="text-sm">{selectedProfileAccessSummary.subAccessLabels.join(', ')}</span>
            </div>
          {/if}
        </div>
      {:else}
        {@const grantedAcceso = accesosCatalogByID.get(Number(selectedAccessOrProfile.id))}
        {#if grantedAcceso}
          {@render grantedAccessCard(grantedAcceso)}
        {:else}
          <div class="_selected-access-orphan">
            <i class={`${accesoAccionesMap.get(4)?.icon} text-red-600`}></i>
            <span class="text-sm">Acceso {selectedAccessOrProfile.id}</span>
          </div>
        {/if}
      {/if}
    {/snippet}
  </SearchDualCard>

  <!-- Sub-accesses live here and not on the option card: the list above is where an access is
       granted, and a checkbox inside a dropdown that closes on the next click is not a place to
       edit anything. Only the granted accesses that declare sub-accesses appear. -->
  {#if accesosConSubAccesos.length > 0}
    <div class="_sub-accesos-section mt-10">
      <div class="_sub-accesos-title text-[15px] leading-[16px] mb-4">
        <T text="Sub-accesses|Sub-accesos" />
      </div>
      <div class="grid grid-cols-2 gap-8 items-start">
        {#each accesosConSubAccesos as acceso (acceso.id)}
          {@const subAccesosSelected = accessEditForm.subAccesosMap.get(acceso.id) || []}
          <div class="_sub-accesos-card">
            <div class="_sub-accesos-card-name text-[15px] ff-semibold">{acceso.nombre}</div>
            <div class="_sub-accesos-card-row text-[14px]">
              {#each acceso.subAccesos as subAcceso}
                <Checkbox
                  size="tiny"
                  label={subAcceso.name}
                  checked={isSubAccesoChecked(subAccesosSelected, subAcceso.id)}
                  underlineOnHover={true}
                  onToggle={() => toggleSubAccesoOnAcceso(acceso.id, subAcceso.id)}
                />
              {/each}
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  {#if accessCatalogLoadError}
    <div class="col-span-24 px-10 py-8 bg-amber-100 border border-amber-400 text-amber-700 rounded">
      {accessCatalogLoadError}
    </div>
  {/if}
</div>

<style>
  :global(._chip-left) {
    align-items: flex-start;
    flex: 0 0 calc(50% - 6px);
    justify-content: flex-start;
    max-width: calc(50% - 6px);
  }

  /* The chip element IS the card. No border in any state: SearchDualCard's shared `._chip:hover`
     ties on specificity with a plain `._chip-right` override and wins on source order, so its frame
     came back on hover and read as the card jumping. The `._container` ancestor breaks that tie, and
     the frame is an outline, which never takes part in layout — hover cannot shift anything. */
  :global(._container ._chip-right),
  :global(._container ._chip-right:hover) {
    align-items: stretch;
    background-color: white;
    border: 0;
    border-radius: 0;
    box-shadow: 0 1px 2px rgba(64, 69, 85, 0.22);
    color: inherit;
    flex: 0 0 calc(33.333% - 8px);
    gap: 6px;
    justify-content: flex-start;
    line-height: 1.15;
    max-width: calc(33.333% - 8px);
    min-height: 0;
    min-width: calc(33.333% - 8px);
    outline: 1px solid transparent;
    overflow: hidden;
    padding: 0;
  }

  :global(._container ._chip-right:hover) {
    outline-color: #c9cee0;
  }

  /* SearchDualCard wraps the snippet in its own span; collapsing it keeps the card one element. */
  :global(._chip-right ._chip-text) {
    display: contents;
  }

  ._access-option {
    height: 100%;
  }

  /* The level column: icon on top, colour bar below it, flush with the card's left edge. */
  ._granted-side {
    align-items: center;
    display: flex;
    flex: 0 0 16px;
    flex-direction: column;
  }

  ._sub-accesos-title {
    color: var(--input-label-color, #6d5dad);
  }

  /* No min-width: the card is sized by its grid column, and a floor wider than the column is what
     would push the second card onto its own row. */
  ._sub-accesos-card {
    background-color: white;
    box-shadow: 0 1px 2px rgba(64, 69, 85, 0.22);
    min-width: 0;
  }

  ._sub-accesos-card-name {
    background-color: #f4f2fd;
    color: #20243b;
    line-height: 19px;
    padding: 4px 10px;
  }

  /* Two fixed columns, not a wrapping row: the checkboxes then line up under each other whatever
     the label lengths are, which is the whole point of a card per access. */
  ._sub-accesos-card-row {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 4px 14px;
    color: #4b4f66;
    line-height: 1.15;
    padding: 6px 10px 7px;
  }

  ._granted-icon {
    flex: 0 0 auto;
    height: 14px;
    margin: 2px 0 1px;
    width: 14px;
  }

  ._granted-line {
    flex: 1 1 auto;
    min-height: 4px;
    width: 100%;
  }

  ._granted-text {
    flex: 1 1 auto;
    min-width: 0;
    padding: 4px 6px 4px 2px;
  }

  /* One line each, cut with an ellipsis: a card that grows to fit its longest access name is what
     made a row of them uneven. The line-heights are explicit because `overflow: hidden` clips the
     line box, and the card's tight 1.15 cut the descenders off "accounting/assets". */
  ._granted-name,
  ._granted-route {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  ._granted-name {
    color: #20243b;
    line-height: 19px;
  }

  ._granted-route {
    color: #6b7280;
    line-height: 17px;
  }

  ._selected-profile-card {
    color: #404555;
    display: flex;
    flex-direction: column;
    gap: 6px;
    width: 100%;
    line-height: 1.2;
  }

  ._selected-profile-name {
    line-height: 18px;
  }

  ._selected-profile-row {
    align-items: flex-start;
    display: flex;
    gap: 6px;
    line-height: 16px;
  }

  ._selected-profile-icon {
    flex: 0 0 auto;
    height: 16px;
    margin-top: 1px;
    margin-right: -2px;
    margin-left: -2px;
    width: 16px;
  }

  ._selected-profile-icon-edit {
    height: 15px;
    margin-left: 0;
    margin-right: 0;
    width: 15px;
  }

  /* The eye is the one icon from MDI: its glyph fills 62% of its 24×24 box where the FontAwesome
     ones fill theirs edge to edge, so at the same box size it reads a size smaller. A transform,
     not a bigger box — it corrects the optical size without moving anything around it. */
  ._selected-profile-icon[class*="mdi--eye"],
  ._granted-icon[class*="mdi--eye"] {
    transform: scale(1.25);
  }

  ._selected-access-orphan {
    align-items: center;
    color: #404555;
    display: flex;
    gap: 6px;
    padding: 6px 8px;
    width: 100%;
  }

  @media (max-width: 768px) {
    :global(._chip-left) {
      flex-basis: calc(100% - 16px);
      max-width: calc(100% - 16px);
    }

    :global(._chip-right) {
      flex-basis: calc(100% - 16px);
      min-width: calc(100% - 16px);
    }
  }
</style>
