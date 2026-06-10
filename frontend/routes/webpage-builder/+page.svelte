<script lang="ts">
  // The bare /webpage-builder route edits the default Inicio page (pageID 0,
  // resolved server-side). Per-page editing uses /webpage-builder/[pageID].
  import BuilderPage from './BuilderPage.svelte';
</script>

<BuilderPage pageID={0} />
