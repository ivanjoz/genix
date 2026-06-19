<script lang="ts">
import Page from '$domain/Page.svelte';
import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
import TestTextIndexes from './TestTextIndexes.svelte';
import TestStockImages from './TestStockImages.svelte';

let view = $state(1)
</script>

<Page title="Testing">
  <OptionsStrip
    selected={view}
    useMobileGrid={true}
    options={[[1, "Text Indexes"], [2, "Stock Images"]]}
    onSelect={(opt) => { view = opt[0] as number }}
  />

  {#if view === 1}
    <TestTextIndexes />
  {:else if view === 2}
    <TestStockImages />
  {/if}
</Page>
