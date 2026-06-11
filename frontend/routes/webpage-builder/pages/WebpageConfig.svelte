<script lang="ts">
import Button from '$components/buttons/Button.svelte';
import Input from '$components/form/Input.svelte';
import { tr } from '$core/store.svelte';
import { Loading, Notify } from '$libs/helpers';
import {
  getWebsiteConfig,
  postWebsiteDomain,
  postWebsiteSeo,
  type IWebsiteConfig,
} from '$services/webpage/pages.svelte';
import { PUBLIC_ZONE_NAME } from '$env/static/public';
import { onMount } from 'svelte';

  // All storefront config (domain + SEO metatags) lives in parameters group 10.
  let config = $state<IWebsiteConfig>({});
  let domainForm = $state({ subdomain: '' });
  const domainSuffix = `.${PUBLIC_ZONE_NAME}`;

  onMount(async () => {
    // Load the persisted domain + SEO so the form is pre-filled.
    config = (await getWebsiteConfig()) || {};
    const savedSubdomain = config.domain?.endsWith(domainSuffix)
      ? config.domain.slice(0, -domainSuffix.length)
      : config.domain || '';
    // Input reloads its value when the saveOn object reference changes.
    domainForm = { subdomain: savedSubdomain };
  });

  const saveDomain = async () => {
    const subdomain = domainForm.subdomain.trim().toLowerCase();
    if (!subdomain) {
      Notify.failure(tr('Enter a domain.|Ingrese un dominio.'));
      return;
    }
    Loading.standard(tr('Saving domain...|Guardando dominio...'));
    try {
      const result = await postWebsiteDomain(`${subdomain}${domainSuffix}`);
      // Keep the editable value limited to the tenant label.
      config.domain = result?.domain;
      domainForm = {
        subdomain: result?.domain?.slice(0, -domainSuffix.length) || subdomain,
      };
    } catch (error) {
      Notify.failure(error as string);
      Loading.remove();
      return;
    }
    Loading.remove();
  };

  const saveSeo = async () => {
    Loading.standard(tr('Saving SEO...|Guardando SEO...'));
    try {
      // Send only the SEO keys; the domain has its own endpoint.
      const { domain: _domain, ...seo } = config;
      await postWebsiteSeo(seo);
    } catch (error) {
      Notify.failure(error as string);
      Loading.remove();
      return;
    }
    Loading.remove();
  };
</script>

<div class="w-full max-w-4xl p-6" aria-label="Website configuration: domain and SEO metatags">
  <!-- Domain -->
  <h3 class="h3 ff-bold mb-6">{tr('Domain|Dominio')}</h3>
  <div class="grid grid-cols-12 gap-10 items-end mb-16">
    <div class="col-span-9 p-rel">
      <Input label="Domain|Dominio" inputCss="pr-70"
        saveOn={domainForm} save="subdomain" />
      <!-- The zone is deployment configuration, not editable tenant data. -->
      <div class="p-abs right-10 bottom-0 h-44 flex items-center pointer-events-none">
        {domainSuffix}
      </div>
    </div>
    <Button name="Save|Guardar" label="Saves the storefront domain."
      color="green" icon="icon-floppy" css="col-span-3"
      onClick={saveDomain}
    />
  </div>

  <!-- SEO metatags (global, one set for the whole site) -->
  <h3 class="h3 ff-bold mb-6">{tr('SEO Metatags|Metatags SEO')}</h3>
  <div class="grid grid-cols-12 gap-10">
    <Input label="Title|Título" css="col-span-12" saveOn={config} save="title" />
    <Input label="Description|Descripción" css="col-span-12" saveOn={config} save="description" />
    <Input label="Keywords|Palabras clave" css="col-span-12" saveOn={config} save="keywords" />
    <Input label="OG Title|OG Título" css="col-span-6" saveOn={config} save="ogTitle" />
    <Input label="OG Description|OG Descripción" css="col-span-6" saveOn={config} save="ogDescription" />
    <Input label="OG Image URL|URL Imagen OG" css="col-span-6" saveOn={config} save="ogImage" />
    <Input label="Favicon URL|URL Favicon" css="col-span-6" saveOn={config} save="favicon" />
    <Button name="Save SEO|Guardar SEO" label="Saves the SEO metatags for the whole site."
      color="green" icon="icon-floppy" css="col-span-3 mt-4"
      onClick={saveSeo}
    />
  </div>
</div>
