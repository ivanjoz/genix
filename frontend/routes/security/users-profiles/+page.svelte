<script lang="ts">
import Page from '$domain/Page.svelte';
import { security } from '$libs/ui-runtime.svelte';
import { useUI } from '@genix/ui';
import { onMount } from 'svelte';
import ProfilesTab from './ProfilesTab.svelte';
import UsersTab from './UsersTab.svelte';
import {
  fetchAccessListCatalog,
  type IAccessGroupCatalogEntry,
  type IAccessListCatalogEntry
} from './access-list-catalog';
import { PerfilesService, PROFILES_ACCESS_ID, USERS_ACCESS_ID } from './users-profiles.svelte';

  const ui = useUI()
  const perfilesService = new PerfilesService()

  let accessGroups = $state([] as IAccessGroupCatalogEntry[])
  let accessListEntries = $state([] as IAccessListCatalogEntry[])
  let accessCatalogLoadError = $state("")

  // Both accesses point at this route, so canAccessRoute lets anyone holding either one in. The
  // tab a user cannot read is dropped from the strip instead — Page falls back to the first option.
  const pageOptions = [
    { id: 1, name: "Users|Usuarios", accessID: USERS_ACCESS_ID },
    { id: 2, name: "Profiles|Perfiles", accessID: PROFILES_ACCESS_ID }
  ].filter(pageOption => security.checkAcceso(pageOption.accessID, 1))

  onMount(async () => {
    try {
      // One load for both tabs: the catalog is the shared, read-only source of truth for the
      // access cards on Profiles and the per-user access selector on Users.
      const accessCatalogPayload = await fetchAccessListCatalog()
      accessGroups = accessCatalogPayload.groups || []
      accessListEntries = accessCatalogPayload.access || []
    } catch (error) {
      accessCatalogLoadError = error as string
      console.error('[access-list] Catalog load failed', { error })
    }
  })
</script>

<Page title="Users & Profiles|Usuarios & Perfiles" options={pageOptions}>
  {#if ui.state.pageOptionSelected === 1 /* Usuarios */}
    <UsersTab
      {perfilesService}
      accessGroupEntries={accessGroups}
      accessCatalogEntries={accessListEntries}
      {accessCatalogLoadError}
    />
  {/if}
  {#if ui.state.pageOptionSelected === 2 /* Perfiles */}
    <ProfilesTab
      {perfilesService}
      {accessGroups}
      {accessListEntries}
      accessListLoadError={accessCatalogLoadError}
    />
  {/if}
</Page>
