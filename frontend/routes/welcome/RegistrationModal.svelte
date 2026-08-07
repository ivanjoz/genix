<script lang="ts">
  import Input from '$components/form/Input.svelte';
  import Modal from '$components/layers/Modal.svelte';
  import T from '$components/misc/T.svelte';
  import { tr } from '$core/store.svelte';

  interface Props {
    id: number
  }

  interface RegistrationEmailForm {
    Email: string
  }

  let { id }: Props = $props();
  let registrationForm = $state<RegistrationEmailForm>({ Email: '' });

  // The public endpoint is intentionally deferred; validation keeps the UI ready without faking delivery.
  const hasValidEmail = $derived(/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(registrationForm.Email.trim()));
</script>

{#snippet registrationTitle()}
  <T text="Start registration|Iniciar registro" />
{/snippet}

<Modal
  {id}
  size={1}
  title={registrationTitle}
  css="!min-h-0 overflow-hidden"
  bodyCss="!p-0"
>
  <form
    class="p-20 md:p-28"
    aria-label={tr('Email registration form|Formulario de registro por correo electrónico')}
    onsubmit={(event) => event.preventDefault()}
  >
    <div class="mb-20 flex size-48 items-center justify-center rounded-[14px] bg-indigo-50 text-indigo-600">
      <i class="icon-[fa--envelope-o] text-2xl" aria-hidden="true"></i>
    </div>

    <p class="mb-8 text-lg font-semibold text-slate-900">
      <T text="Enter your email address to begin registration.|Ingrese su correo electrónico para iniciar el registro." />
    </p>
    <p class="mb-24 leading-[1.55] text-slate-600">
      <T text="We will send you a registration email. Click the registration link in that email, or copy its verification code, to continue.|Le enviaremos un correo de registro. Para continuar, haga clic en el enlace de registro incluido en el correo o copie su código de verificación." />
    </p>

    <Input
      saveOn={registrationForm}
      save="Email"
      label="Email address|Correo electrónico"
      type="email"
      required={true}
      inputCss="text-base"
    />

    <button
      type="submit"
      class="mt-20 flex h-44 w-full items-center justify-center gap-8 rounded-[10px] bg-indigo-600 px-18 text-white shadow-lg shadow-indigo-900/15 transition hover:bg-indigo-700 focus-visible:outline-3 focus-visible:outline-offset-3 focus-visible:outline-indigo-600 disabled:cursor-not-allowed disabled:bg-slate-300 disabled:shadow-none"
      aria-label={tr('Send registration email|Enviar correo de registro')}
      disabled={true}
      title={tr('Registration email delivery is coming soon|El envío del correo de registro estará disponible próximamente')}
    >
      <i class="icon-[fa--paper-plane-o]" aria-hidden="true"></i>
      <span class="font-semibold"><T text="Send registration email|Enviar correo de registro" /></span>
    </button>

    <p class="mt-10 text-center text-sm text-slate-500" aria-live="polite">
      {#if hasValidEmail}
        <T text="Email delivery will be enabled with the public registration service.|El envío se habilitará con el servicio público de registro." />
      {:else}
        <T text="Registration email delivery will be available soon.|El envío del correo de registro estará disponible próximamente." />
      {/if}
    </p>
  </form>
</Modal>
