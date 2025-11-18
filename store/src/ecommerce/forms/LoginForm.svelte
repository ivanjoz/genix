<script lang="ts">
    import Input from "$components/Input.svelte";


  let { 
    css, isMobile 
  } = $props<{ css?: boolean, isMobile?: boolean }>();

  let form = { usuario: "", password: "" }

</script>

<div class="grid grid-cols-1 gap-8 p-2 md:grid-cols-2">
  <div class="mt-4"></div>
  <Input saveOn={form} css="mb-2" save="usuario" label="Usuario"/>
  <Input saveOn={form} css="mb-2" type="password" save="password" label="Password"/>

</div>