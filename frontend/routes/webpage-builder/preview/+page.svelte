<script lang="ts">
  import Page from '#domain/Page.svelte';
  import { Env } from '#core/env.ts';

  // The storefront is a separate app served under /webpage-app on the same origin; it resolves the
  // tenant from `cid`, so the preview shows the logged-in company's store inside the admin shell.
  const storefrontPreviewURL = `/webpage-app/?cid=${Env.getCompanyID()}`;
</script>

<Page title="Preview|Previsualizar" noPadding fixedFullHeight hideHeader>
  <iframe
    src={storefrontPreviewURL}
    title="Storefront preview"
    class="block h-full w-full border-0 bg-white"
  ></iframe>
</Page>
