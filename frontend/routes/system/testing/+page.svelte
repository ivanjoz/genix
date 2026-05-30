<script lang="ts">
import Page from '$domain/Page.svelte';
import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
import TestTextIndexes from './TestTextIndexes.svelte';

// Single sub-view for now; OptionsStrip is in place so more testing tools
// can be added as extra options later.
let view = $state(1)
</script>

<Page title="Testing">
  <OptionsStrip
    selected={view}
    useMobileGrid={true}
    options={[[1, "Text Indexes"]]}
    onSelect={(opt) => { view = opt[0] as number }}
  />

  {#if view === 1}
    <TestTextIndexes />
  {/if}
</Page>
