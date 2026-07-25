<script lang="ts">
  // The showroom itself lives in @genix/ui (packages/genix-ui/showroom), so the catalogue
  // ships with the components it documents. This route only supplies the app shell.
  import Showroom from '$components/showroom/Showroom.svelte';
  import Page from '$domain/Page.svelte';

  // Driven by the showroom's own surface toggle, so the flip covers the whole
  // #page-container rather than just the component's box.
  let containerCss = $state('bg-white');
</script>

<Page title="UI Showroom" {containerCss}>
  <Showroom onSurfaceChange={(surfaceCss) => { containerCss = surfaceCss; }} />
</Page>
