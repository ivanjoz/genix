<script lang="ts">
import Page from '$domain/Page.svelte';
import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
import TestTextIndexes from './TestTextIndexes.svelte';
import TestStockImages from './TestStockImages.svelte';
import { useUI } from '@genix/ui';

const ui = useUI()

const pageOptions = [
  { id: 1, name: "Testing" },
  { id: 2, name: "UI Showroom" },
]

let testingView = $state(1)

// Flipped by the showroom's own surface toggle, so the audit covers the whole
// page container and not just the catalogue's box.
let containerCss = $state('')
</script>

<Page title="Developer" options={pageOptions} {containerCss}>
  {#if ui.state.pageOptionSelected === 1 /* Testing */}
    <OptionsStrip
      selected={testingView}
      useMobileGrid={true}
      options={[[1, "Text Indexes"], [2, "Stock Images"]]}
      onSelect={(opt) => { testingView = opt[0] as number }}
    />

    {#if testingView === 1}
      <TestTextIndexes />
    {:else if testingView === 2}
      <TestStockImages />
    {/if}
  {/if}

  {#if ui.state.pageOptionSelected === 2 /* UI Showroom */}
    <!-- Imported on demand: the showroom drags in every component of the package —
         5k-row tables, canvas charts, the RoosterJS editor — and this route is in the
         menu, so a static import would load all of it for the Testing section too. -->
    {#await import('$components/showroom/Showroom.svelte') then showroomModule}
      <showroomModule.default onSurfaceChange={(surfaceCss) => { containerCss = surfaceCss }} />
    {/await}
  {/if}
</Page>
