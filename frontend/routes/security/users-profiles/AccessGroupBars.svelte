<script lang="ts">
  import type { IAccessGroupCatalogEntry, IAccessListCatalogEntry } from "./access-list-catalog"
  import { accesoAcciones, type IProfile, type IUser } from "./users-profiles.svelte"
  import { buildAccessGroupBars, buildUsuarioEffectiveNiveles } from "./users-profiles"

  let { usuario, perfilesByID, accessCatalogEntries, accessGroupEntries }: {
    usuario: IUser
    perfilesByID: Map<number, IProfile>
    accessCatalogEntries: IAccessListCatalogEntry[]
    accessGroupEntries: IAccessGroupCatalogEntry[]
  } = $props()

  const accesoAccionesMap = new Map(accesoAcciones.map((accesoAccion) => [accesoAccion.id, accesoAccion]))

  // The profiles and the user's own grants are merged first, so a bar shows the effective
  // permission rather than where it came from.
  const accessGroupBars = $derived(buildAccessGroupBars(
    buildUsuarioEffectiveNiveles(usuario, perfilesByID),
    accessCatalogEntries,
    accessGroupEntries
  ))
</script>

<div class="_bars">
  {#each accessGroupBars as accessGroupBar (accessGroupBar.groupID)}
    <div class="_bar">
      <div class="_bar-name text-[13px]">{accessGroupBar.groupName}</div>
      <!-- The track is the group's whole catalog; each segment is the share of it the user holds at
           that level, and the empty tail is what it does not hold. -->
      <div class="_bar-track">
        {#each accessGroupBar.segments as segment (segment.nivel)}
          {@const accesoAccion = accesoAccionesMap.get(segment.nivel)}
          <div
            class="_bar-segment"
            style:width={`${(segment.accessCount / accessGroupBar.totalAccessCount) * 100}%`}
            style:background-color={accesoAccion?.color2 || accesoAccion?.color}
            title={`${accesoAccion?.name}: ${segment.accessCount}/${accessGroupBar.totalAccessCount}`}
          >
            <i class={`${accesoAccion?.icon} _bar-icon`}></i>
          </div>
        {/each}
      </div>
    </div>
  {/each}
</div>

<style>
  ._bars {
    display: flex;
    flex-wrap: wrap;
    gap: 4px 10px;
    max-width: 100%;
  }

  ._bar {
    flex: 0 0 auto;
    width: 112px;
  }

  ._bar-name {
    color: #6b7280;
    line-height: 15px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Outlined, not filled: the empty tail has to read as "not granted". */
  ._bar-track {
	  display: flex;
	  height: 18px;
	  overflow: hidden;
	  border-bottom: 1px solid #00000000;
	  background-color: #8080801a;
  }

  ._bar-segment {
    align-items: center;
    display: flex;
    justify-content: center;
    min-width: 0;
    overflow: hidden;
  }

  ._bar-icon {
    color: #00000099;
    flex: 0 0 auto;
    height: 12px;
    width: 12px;
  }

  /* The eye is the one icon from MDI: its glyph fills 62% of its 24×24 box where the FontAwesome
     ones fill theirs edge to edge, so at 12px it reads a size smaller than the shield next to it. */
  ._bar-icon[class*="mdi--eye"] {
    transform: scale(1.25);
  }
</style>
