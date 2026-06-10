<script lang="ts">
  // Per-page builder route: /webpage-builder/<PageID>. The PageID comes from the
  // pages list (pencil button) and selects which page's content to load/save.
  import { page } from '$app/state';
  import BuilderPage from '../BuilderPage.svelte';

  // params.pageID is a string; coerce to a number (0 falls back to the default page).
  const pageID = $derived(Number(page.params.pageID) || 0);
</script>

<BuilderPage {pageID} />
