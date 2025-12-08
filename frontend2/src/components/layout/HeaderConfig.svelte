<script lang="ts">
    import OptionsStrip from "$components/micro/OptionsStrip.svelte";


  const options = [
    { id: 1, name: "Usuario" }, { id: 2, name: "Configuración" }
  ]
  let selected = $state(1)

	function handleLogout() {
		// Clear session/tokens
		localStorage.clear();
		sessionStorage.clear();
		window.location.href = '/login';
	}

</script>

<div class="flex items-center">
  <OptionsStrip options={options} keyId="id" keyName="name" 
    selected={selected} onSelect={e => selected = e.id}
  />
</div>
{#if selected === 1}
  <div class="w-full flex justify-between">
    <div></div>
    <button class="bx-orange"
      onclick={handleLogout}
    >
      <i class="icon-logout-1"></i>
      <span>Salir</span>
    </button>
  </div>
{/if}


