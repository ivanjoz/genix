<script lang="ts">
  import TableGrid from '$components/vTable/TableGrid.svelte'
  import type { ITableColumn } from '$components/vTable/types'
  import FilterInput from '$components/form/FilterInput.svelte'
  import Layer from '$components/layers/Layer.svelte'
  import { useUI } from '@genix/ui'
  import type { IAccessGroupCatalogEntry, IAccessListCatalogEntry } from "./access-list-catalog"
  import AccessGroupBars from "./AccessGroupBars.svelte"
  import { accesoAcciones, type IProfile, type IUser } from "./users-profiles.svelte"
  import { buildAccessUserRows, buildUsuarioEffectiveNiveles, type IAccessUserRow } from "./users-profiles"

  const ui = useUI()

  let { usuarios, perfilesByID, accessCatalogEntries, accessGroupEntries, filterText = "", onUsuarioSelect }: {
    usuarios: IUser[]
    perfilesByID: Map<number, IProfile>
    accessCatalogEntries: IAccessListCatalogEntry[]
    accessGroupEntries: IAccessGroupCatalogEntry[]
    filterText?: string
    /** Hands the user back to the "Por Usuario" view, which owns editing them. */
    onUsuarioSelect: (usuario: IUser) => void
  } = $props()

  const accesoAccionesMap = new Map(accesoAcciones.map((accesoAccion) => [accesoAccion.id, accesoAccion]))

  const usuariosByID = $derived(new Map(usuarios.map((usuario) => [usuario.ID, usuario])))

  const accessUserRows = $derived(buildAccessUserRows(
    usuarios.map((usuario) => ({
      usuarioID: usuario.ID,
      usuarioName: [usuario.FirstName, usuario.LastName].filter(Boolean).join(" ").trim() || usuario.User,
      nivelByAccesoID: buildUsuarioEffectiveNiveles(usuario, perfilesByID),
    })),
    accessCatalogEntries,
    accessGroupEntries
  ))

  // The filter matches the user names too, so typing a person answers "what does this person hold",
  // which is the same question the other view answers by row.
  const filteredAccessUserRows = $derived.by(() => {
    const normalizedFilterText = filterText.trim().toLowerCase()
    if (!normalizedFilterText) { return accessUserRows }

    return accessUserRows.filter(accessUserRow =>
      [accessUserRow.accesoName, accessUserRow.groupName, ...accessUserRow.usuarios.map(u => u.usuarioName)]
        .join(" ").toLowerCase().includes(normalizedFilterText))
  })

  const ACCESS_USERS_LAYER_ID = 2
  let selectedAccessRow = $state(undefined as IAccessUserRow | undefined)
  let usuarioFilterText = $state("")
  const selectedAccesoAccion = $derived(accesoAccionesMap.get(selectedAccessRow?.nivel || 0))

  // The layer filter matches the login too: an access held by a dozen people is searched by whatever
  // the operator has at hand, and that is as often the username as the full name.
  const selectedAccessUsuarios = $derived.by(() => {
    const rowUsuarios = selectedAccessRow?.usuarios || []
    if (!usuarioFilterText) { return rowUsuarios }

    return rowUsuarios.filter(accessUsuario =>
      [accessUsuario.usuarioName, usuariosByID.get(accessUsuario.usuarioID)?.User || ""]
        .join(" ").toLowerCase().includes(usuarioFilterText))
  })

  const columns: ITableColumn<IAccessUserRow>[] = [
    { id: "acceso", header: "Access|Acceso", width: "300px", useCellRenderer: true, css: "px-8" },
    { id: "usuarios", header: "Users|Usuarios", width: "minmax(300px, 1fr)", useCellRenderer: true, css: "px-8" },
  ]
</script>

<TableGrid
  {columns}
  data={filteredAccessUserRows}
  height="calc(100vh - 8rem - 16px)"
  rowHeight={46}
  getRowId={(accessUserRow) => accessUserRow.rowKey}
  selectedRowId={selectedAccessRow?.rowKey}
  onRowClick={(accessUserRow) => {
    selectedAccessRow = accessUserRow
    usuarioFilterText = ""
    ui.openSideLayer(ACCESS_USERS_LAYER_ID)
  }}
  css="w-full"
>
  {#snippet cellRenderer(accessUserRow: IAccessUserRow, columnDefinition: ITableColumn<IAccessUserRow>)}
    {#if columnDefinition.id === "acceso"}
      {@const accesoAccion = accesoAccionesMap.get(accessUserRow.nivel)}
      <div class="flex items-center gap-7 w-full min-w-0">
        <!-- The MDI eye fills 62% of its box where the FontAwesome icons fill theirs edge to edge,
             so it needs the scale to read the same size as the shield. -->
        <i class="{accesoAccion?.icon} shrink-0 h-14 w-14"
          class:scale-125={accesoAccion?.icon.includes("mdi--eye")}
          style:color={accesoAccion?.color} title={accesoAccion?.name}></i>
        <div class="min-w-0 leading-[1.15]">
          <div class="text-[16px] ff-semibold text-[#20243b] truncate">{accessUserRow.accesoName}</div>
          <div class="text-[15px] text-gray-500">{accessUserRow.groupName}</div>
        </div>
      </div>
    {:else}
      <!-- One line of cards, clipped by the fixed row height: the grid stays uniform and a row
           holding more users than fit scrolls sideways rather than growing. -->
      <div class="flex items-center gap-5 w-full min-w-0 overflow-x-auto">
        {#each accessUserRow.usuarios as accessUsuario (accessUsuario.usuarioID)}
          <!-- Fixed width, not max-width: equal cards line up in columns down the table. -->
          <div class="shrink-0 flex items-center justify-center w-128 min-h-30 px-8 text-[14px]
            text-center bg-white rounded-[4px]" title={accessUsuario.usuarioName}
            style="border: 1px solid #939bd0; box-shadow: rgb(71 71 144 / 31%) 0px 2px 2px -1px;">
            <!-- Two lines then ellipsis so a long name does not stretch the card past its
                 neighbours, on the inner span because the card itself is a flex box.
                 whitespace-normal: the grid cell is nowrap, so without it the name never wraps. -->
            <span class="line-clamp-2 whitespace-normal leading-[15px]">{accessUsuario.usuarioName}</span>
          </div>
        {/each}
      </div>
    {/if}
  {/snippet}
</TableGrid>

<!-- The heading is one stacked block passed as titleSide, not Layer's own `title`: the header row is
     as tall as the close button, so anything rendered after it lands a button's height below the
     access name instead of under it. `titleCss="hidden"` drops the now-empty title div. -->
{#snippet accessLayerHeading()}
  <div class="leading-[1.3]">
    <div class="h2">{selectedAccessRow?.accesoName}</div>
    <div class="text-[14px] text-gray-500 ml-2">
      {selectedAccessRow?.groupName} · {selectedAccesoAccion?.name}
    </div>
  </div>
{/snippet}

<Layer
  id={ACCESS_USERS_LAYER_ID}
  type="side"
  sideLayerSize={620}
  titleIcon="{selectedAccesoAccion?.icon} w-20 h-20 self-start mt-4"
  titleCss="hidden"
  titleSide={accessLayerHeading}
  css="px-12 py-10"
  contentCss="px-0"
>
  {#if selectedAccessRow}
    <div class="flex justify-start mb-8 mt-4">
      <FilterInput bind:value={usuarioFilterText} size="small" css="w-224" />
    </div>
    {#each selectedAccessUsuarios as accessUsuario (accessUsuario.usuarioID)}
      {@const usuario = usuariosByID.get(accessUsuario.usuarioID)}
      {#if usuario}
        <!-- The whole entry opens the user: the bars are what this access looks like in context,
             and the next question after "who holds it" is always "what else do they hold". -->
        <button class="w-full text-left px-8 py-6 mb-6 rounded-[4px] bg-white hover:bg-[#f5f4ff]"
          style="border: 1px solid #dfe1ee;" onclick={() => onUsuarioSelect(usuario)}>
          <div class="text-[15px] ff-semibold text-[#20243b] mb-4">{accessUsuario.usuarioName}</div>
          <AccessGroupBars {usuario} {perfilesByID} {accessCatalogEntries} {accessGroupEntries} />
        </button>
      {/if}
    {/each}
  {/if}
</Layer>
