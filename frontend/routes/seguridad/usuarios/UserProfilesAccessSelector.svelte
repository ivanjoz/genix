<script lang="ts">
  import SearchDualCard from '$components/SearchDualCard.svelte';
  import { accesoAcciones } from '../perfiles-accesos/perfiles-accesos.svelte';
  import type {
    IAccessGroupCatalogEntry,
    IAccessListCatalogEntry
  } from '../perfiles-accesos/access-list-catalog';
  import type { IPerfil, IUsuario } from './usuarios.svelte';

  interface IProfileAccessSummary {
    readableAccessNames: string[]
    editableAccessNames: string[]
  }

  interface ISelectedAccessPresentation {
    accessName: string
    accessGroupName: string
    actionIcon: string
    actionColorClass: string
  }

  interface IUserProfilesAccessSelectorProps {
    saveOn: IUsuario
    perfiles: IPerfil[]
    accessGroupEntries: IAccessGroupCatalogEntry[]
    accessCatalogEntries: IAccessListCatalogEntry[]
    accessLevelOptions: { ID: number; Nombre: string }[]
    accessCatalogLoadError?: string
    css?: string
  }

  let {
    saveOn = $bindable(),
    perfiles,
    accessGroupEntries,
    accessCatalogEntries,
    accessLevelOptions,
    accessCatalogLoadError = '',
    css = ''
  }: IUserProfilesAccessSelectorProps = $props();

  const accessActionPresentationByID = new Map(
    accesoAcciones.map((accessActionRecord) => [
      accessActionRecord.id,
      {
        actionIcon: accessActionRecord.icon,
        actionColorClass: accessActionRecord.id === 1 ? 'text-blue-600' : 'text-red-600'
      }
    ])
  );


  const accessCatalogNameByID = $derived.by(() => {
    const accessNameByID = new Map<number, string>();

    for (const accessCatalogEntry of accessCatalogEntries) {
      accessNameByID.set(accessCatalogEntry.id, accessCatalogEntry.name);
    }

    return accessNameByID;
  });

  const accessGroupNameByID = $derived.by(() => {
    const groupNameByID = new Map<number, string>();

    for (const accessGroupEntry of accessGroupEntries) {
      groupNameByID.set(accessGroupEntry.id, accessGroupEntry.name);
    }

    return groupNameByID;
  });

  function summarizeProfileAccesses(profileRecord?: IPerfil): IProfileAccessSummary {
    const readableAccessNames = new Set<string>();
    const editableAccessNames = new Set<string>();

    if (!profileRecord?.accesosMap) {
      return { readableAccessNames: [], editableAccessNames: [] };
    }

    // Split profile permissions into view and edit buckets for the custom card renderer.
    for (const [accessID, accessLevels] of profileRecord.accesosMap) {
      const accessName = accessCatalogNameByID.get(accessID);
      if (!accessName) { continue; }

      if (accessLevels.includes(1)) {
        readableAccessNames.add(accessName);
      }

      if (accessLevels.some((accessLevel) => accessLevel > 1)) {
        editableAccessNames.add(accessName);
      }
    }

    return {
      readableAccessNames: [...readableAccessNames].sort((leftName, rightName) => leftName.localeCompare(rightName)),
      editableAccessNames: [...editableAccessNames].sort((leftName, rightName) => leftName.localeCompare(rightName))
    };
  }

  function getProfileRecord(profileID: number): IPerfil | undefined {
    return perfiles.find((profileRecord) => profileRecord.ID === profileID);
  }

  function getSelectedAccessPresentation(encodedAccessLevelID: number): ISelectedAccessPresentation {
    // Decode the combined access+level identifier so the selected chip shows catalog metadata instead of the raw suffix.
    const accessID = Math.floor(encodedAccessLevelID / 10);
    const accessLevel = encodedAccessLevelID % 10;
    const accessActionPresentation = accessActionPresentationByID.get(accessLevel);
    const accessCatalogEntry = accessCatalogEntries.find((currentAccessCatalogEntry) => currentAccessCatalogEntry.id === accessID);

    return {
      accessName: accessCatalogNameByID.get(accessID) || `Acceso ${accessID}`,
      accessGroupName: accessGroupNameByID.get(accessCatalogEntry?.group || 0) || '',
      actionIcon: accessActionPresentation?.actionIcon || 'icon-shield',
      actionColorClass: accessActionPresentation?.actionColorClass || 'text-red-600'
    };
  }

</script>

<div class={css}>
  <SearchDualCard
    bind:saveOn
    saveLeft="PerfilesIDs"
    saveRight="AccesosNivelIDs"
    css="col-span-24"
    cardCss="mt-8"
    leftOptions={perfiles}
    leftKeyId="ID"
    leftKeyName="Nombre"
    leftLabel="PERFILES ::"
    rightOptions={accessLevelOptions}
    rightKeyId="ID"
    rightKeyName="Nombre"
    rightLabel="ACCESOS ::"
  >
    {#snippet selectedItem(selectedAccessOrProfile)}
      {#if selectedAccessOrProfile.source === 'left'}
        {@const selectedProfileRecord = getProfileRecord(Number(selectedAccessOrProfile.id))}
        {@const selectedProfileAccessSummary = summarizeProfileAccesses(selectedProfileRecord)}
        <div class="_selected-profile-card">
          <div class="_selected-profile-name ff-semibold text-sky-700">
            {String(selectedAccessOrProfile.option.Nombre || '')}
          </div>
          {#if selectedProfileAccessSummary.readableAccessNames.length > 0}
            <div class="_selected-profile-row">
              <i class="icon-eye _selected-profile-icon pt-2 text-blue-600"></i>
              <span class="text-sm">{selectedProfileAccessSummary.readableAccessNames.join(', ')}</span>
            </div>
          {/if}
          {#if selectedProfileAccessSummary.editableAccessNames.length > 0}
            <div class="_selected-profile-row">
              <i class="icon-pencil _selected-profile-icon pt-2 text-red-600"></i>
              <span class="text-sm">{selectedProfileAccessSummary.editableAccessNames.join(', ')}</span>
            </div>
          {/if}
        </div>
      {:else}
        {@const selectedAccessPresentation = getSelectedAccessPresentation(Number(selectedAccessOrProfile.id))}
        <div class="_selected-chip-access">
          <i class={`${selectedAccessPresentation.actionIcon} _selected-access-icon ${selectedAccessPresentation.actionColorClass}`}></i>
          <div class="_selected-access-text">
            <div class="_selected-access-name">{selectedAccessPresentation.accessName}</div>
            {#if selectedAccessPresentation.accessGroupName}
              <div class="text-xs _selected-access-group-name">{selectedAccessPresentation.accessGroupName}</div>
            {/if}
          </div>
        </div>
      {/if}
    {/snippet}
  </SearchDualCard>

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

  :global(._chip-right) {
    align-items: flex-start;
    flex: 0 0 calc(25% - 8px);
    justify-content: flex-start;
    max-width: calc(25% - 8px);
    min-width: calc(25% - 8px);
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
    margin-top: 1px;
    margin-right: -2px;
    margin-left: -2px;
  }

  ._selected-chip-access {
    align-items: center;
    color: #404555;
    display: flex;
    gap: 6px;
    width: 100%;
  }

  ._selected-access-icon {
    margin-top: 1px;
    margin-right: -2px;
    margin-left: -2px;
  }

  ._selected-access-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  ._selected-access-name {
    line-height: 16px;
  }

  ._selected-access-group-name {
    color: #7b8294;
    line-height: 14px;
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
