<script lang="ts">
import Input from '$components/Input.svelte';
import OptionsStrip from '$components/OptionsStrip.svelte';
import { accessHelper } from '$core/security';
import pkg from 'notiflix'
const { Loading, Notify } = pkg;
import { postUsuarioPropio } from '$services/services/usuarios.svelte';
import type { IUsuario } from '$core/types/common';


  const options = [
    { id: 1, name: "Usuario" }, { id: 2, name: "Config." }
  ]
  let selected = $state(1)

	function handleLogout() {
		// Clear session/tokens
		localStorage.clear();
		sessionStorage.clear();
		window.location.href = '/login';
	}

  let userInfo = $state(accessHelper.getUserInfo())
  $effect(() => {
    if(selected === 1){ userInfo = userInfo = accessHelper.getUserInfo()}
  })

  const saveUsuario = async () => {
    if(userInfo.password1 && userInfo.password1 !== userInfo.password2){
      Notify.failure("Los password no coinciden.")
    }

    Loading.standard("Creando/Actualizando Usuario...")
    try {
      var result = await postUsuarioPropio(userInfo)
    } catch (error) {
      Notify.failure(error as string)
      Loading.remove()
      return
    }
    Loading.remove()
    accessHelper.setUserInfo(userInfo)
    console.log("usuario result::", result)
  }

</script>

<div class="flex items-center">
  <OptionsStrip options={options} keyId="id" keyName="name"
    selected={selected} onSelect={e => selected = e.id}
  />
</div>
{#if selected === 1}
  <div class="w-full flex mb-12 mt-[-2px]">
    <div class="mr-auto"></div>
    <button class="bx-blue mr-12" aria-label="Guardar Usuario"
      onclick={() => { saveUsuario() }}
    >
      <i class="icon-floppy"></i>
    </button>
    <button class="bx-orange" aria-label="Salir"
      onclick={handleLogout}
    >
      <i class="icon-logout-1"></i>
      <span>Salir</span>
    </button>
  </div>
  <div class="grid grid-cols-24 w-full gap-10">
    <Input label="Nombres" css="col-span-12"
      saveOn={userInfo} save="nombres"
    />
    <Input label="Apellidos" css="col-span-12"
      saveOn={userInfo} save="apellidos"
    />
    <Input label="Email" css="col-span-12"
      saveOn={userInfo} save="email"
    />
    <Input label="Cargo" css="col-span-12"
      saveOn={userInfo} save="cargo"
    />
    <Input label="Nº Documento" css="col-span-12"
      saveOn={userInfo} save="documentoNro"
    />
    <div class="col-span-24">
      <div class="ff-bold mb-[-4px] mt-2">Cambiar Password</div>
    </div>
    <Input label="Password" css="col-span-12"
      saveOn={userInfo} save="password1" type="password"
    />
    <Input label="Repetir Password" css="col-span-12"
      saveOn={userInfo} save="password2" type="password"
    />
  </div>
{/if}
