<script lang="ts">
  import { onMount } from "svelte";
import Input from '$ui/components/Input.svelte';
import { Notify } from '$core/lib/helpers';
import { sendUserLogin, ILogin } from '$services/services/admin/login';
import { checkIsLogin } from '$core/lib/security';
import { Env } from '$core/lib/env';

  let form = $state({ Usuario: "", Password: "", EmpresaID: 1, CipherKey: "" } as ILogin)
  let isLoading = $state(false)

  const sendLogin = async () => {
    if(form.Usuario.length < 4 || form.Password.length < 4){
      Notify.failure('Debe proporcionar un usuario y una contraseña válidos');
      return
    }

    isLoading = true
    Notify.info('Enviando Credenciales...')

    const result = await sendUserLogin(form)
    console.log(result)

    isLoading = false

    if(result.error){
      Notify.failure('Error al iniciar sesión')
    }
  }

  onMount(() => {
    if(checkIsLogin() === 2){
      Env.navigate("/")
    }
  })

</script>
<div class="relative">
  <img class="bg-image-1" src="/images/background-1.webp" alt="" />
  <div class="flex items-center h-screen login-bg-c1 relative">
    <div class="login-bg-1 w-full">
      <div class="login-tt flex items-center text-xl">
        Iniciar Sesión
      </div>
      <div class="login-logo-c relative mb-2">
        <img class="w-full h-full" src="/images/genix_logo.svg" alt="Genix Logo" />
      </div>
      <div class="flex justify-center items-center text-xl font-semibold text-[#686caa] mb-4">
        Gestor Empresarial en la Nube para MyPes
      </div>
      <Input
        required={true}
        css="mb-12 w-full text-lg"
        label="Usuario"
        saveOn={form}
        save="Usuario"
        type="text"
      />
      <Input
        required={true}
        css="mb-12 w-full text-lg"
        label="Contraseña"
        saveOn={form}
        save="Password"
        type="password"
      />
      <div class="flex w-full justify-center items-center mt-[1.4rem]">
        <button
          class="bn1 big d-blue"
          disabled={isLoading}
          onclick={ev => {
            ev.stopPropagation()
            sendLogin()
          }}
        >
          <i class="icon-login"></i> Ingresar
        </button>
      </div>
    </div>
  </div>
</div>

<style>
  .bg-image-1 {
    position: absolute;
    left: 0;
    top: 0;
    height: 100vh;
    object-fit: cover;
    max-width: 100vw;
  }

  .login-bg-c1 {
    position: absolute;
    top: 0px;
    right: 4rem;
    height: 100vh;
  }

  .login-logo-c {
    height: 8rem;
  }

  .login-logo-c > img {
    object-fit: contain;
  }

  .login-bg-1 {
    height: 38rem;
    width: 30rem;
    background-color: white;
    box-shadow: rgba(17, 17, 26, 0.1) 0px 4px 16px, rgba(17, 17, 26, 0.1) 0px 8px 24px, rgba(17, 17, 26, 0.1) 0px 16px 56px;
    border-radius: 14px 14px 14px 14px;
    position: relative;
    border-top: 4px solid #5759b1;
    padding: 2rem 2rem;
  }

  .login-tt {
    position: absolute;
    top: -2.2rem;
    height: 2.2rem;
    left: 2rem;
    padding: 0 11px;
    background-color: #5759b1;
    color: white;
    border-radius: 11px 11px 0 0;
  }

  .bn1 {
    display: inline-flex;
    height: 2.3rem;
    justify-content: center;
    align-items: center;
    padding-left: 1rem;
    padding-right: 1rem;
    background-color: white;
    color: #475262;
    box-shadow: rgba(149, 157, 165, 0.2) 0px 8px 24px;
    border-radius: 8px;
    font-weight: 500;
    gap: 5px;
    border: none;
    cursor: pointer;
    outline: none;
  }

  .bn1.big {
    height: 2.5rem;
    font-size: 1.1rem;
  }

  .d-blue {
    background-color: #5e83f2;
    color: #ffffff;
    border: 2px solid #5475dc;
  }

  .d-blue:hover {
    background-color: #6a8df7;
    border: 2px solid #6a8df7;
  }

  .d-blue:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  @media (max-width: 768px) {
    .login-bg-c1 {
      right: unset;
      left: 0;
      width: 100%;
      display: flex;
      justify-content: center;
    }

    .login-bg-1 {
      width: 90vw;
      padding: 1rem 1.2rem 1.4rem 1.2rem;
      height: fit-content;
    }

    .login-tt {
      height: 2.5rem;
      font-size: 1.1rem;
    }

    .login-logo-c {
      height: 6.5rem;
    }
  }
</style>
