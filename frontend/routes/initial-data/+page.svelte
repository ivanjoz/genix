<script lang="ts">
  import T from '$components/misc/T.svelte';
  import { Env } from '$core/env';
  import InitialDataForm from './InitialDataForm.svelte';
</script>

<div class="flex items-center justify-center min-h-screen initial-data-bg px-8 py-20">
  <div class="initial-data-card">
    <div class="initial-data-tt flex items-center text-xl">
      <T text="Initial Data|Datos Iniciales" />
    </div>
    <div class="initial-data-logo-c relative mb-2">
      <img class="w-full h-full" src="/images/genix_logo.svg" alt="Genix Logo" />
    </div>
    <p class="text-center text-gray-600 mb-16">
      <T text="Your company needs a branch, a warehouse and a cash register to start operating. You can accept the suggested names.|Su empresa necesita una sede, un almacén y una caja para poder operar. Puede aceptar los nombres sugeridos." />
    </p>

    <InitialDataForm onSaved={() => Env.navigate("/")} />
  </div>
</div>

<style>
  .initial-data-bg {
    background-color: #eef0f8;
  }

  .initial-data-card {
    width: 40rem;
    max-width: 92vw;
    background-color: white;
    box-shadow: rgba(17, 17, 26, 0.1) 0px 4px 16px, rgba(17, 17, 26, 0.1) 0px 8px 24px;
    border-radius: 14px;
    position: relative;
    border-top: 4px solid #5759b1;
    padding: 2rem;
  }

  .initial-data-tt {
    position: absolute;
    top: -2.2rem;
    height: 2.2rem;
    left: 2rem;
    padding: 0 11px;
    background-color: #5759b1;
    color: white;
    border-radius: 11px 11px 0 0;
  }

  .initial-data-logo-c {
    height: 6rem;
  }

  .initial-data-logo-c > img {
    object-fit: contain;
  }
</style>
