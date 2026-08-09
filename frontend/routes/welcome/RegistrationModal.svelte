<script lang="ts">
  import { onDestroy, untrack } from 'svelte';
  import Input from '$components/form/Input.svelte';
  import Modal from '$components/layers/Modal.svelte';
  import T from '$components/misc/T.svelte';
  import { Env, type IApiEndpointOption } from '$core/env';
  import { tr } from '$core/store.svelte';
  import { formatTime, Notify } from '$libs/helpers';
  import { extractError, security } from '$libs/ui-runtime.svelte';
  import { createSignUpCompany, requestSignUpCode, verifySignUpCode } from '$services/signup';
  import InitialDataForm from '../initial-data/InitialDataForm.svelte';

  interface Props {
    id: number
    // Set when the visitor arrived from the email link (/welcome?req=…&code=…). The modal then
    // opens straight on the code check instead of asking for the address again.
    presetRequestID?: number
    presetCode?: string
    // The server picker is shared with the login form rather than duplicated, so both always agree
    // on which backend the visitor is talking to.
    apiEndpointOptions?: IApiEndpointOption[]
    selectedApiEndpointRoute?: string
    onApiEndpointChange?: (selectedEndpoint: IApiEndpointOption) => void
  }

  let {
    id, presetRequestID = 0, presetCode = '',
    apiEndpointOptions = [], selectedApiEndpointRoute = '', onApiEndpointChange,
  }: Props = $props();

  const steps = [
    { number: 1, title: 'Email verification|Verificación de correo' },
    { number: 2, title: 'Company information|Datos de la empresa' },
    { number: 3, title: 'Initial data|Datos iniciales' },
  ];

  let currentStep = $state(1);
  let isBusy = $state(false);

  let emailForm = $state({ Email: '' });
  let codeForm = $state({ Code: '' });
  let requestID = $state(0);
  let isCodeStage = $state(false);
  let isVerifyingLink = $state(false);
  // Shown inline at the top of step 1 rather than as a toast: an expired or spent code is a normal
  // outcome here, and the message has to stay on screen while the visitor asks for a new email.
  let verificationError = $state('');
  // SUnixTime of the last delivery, and the live countdown until another one is allowed.
  let lastSentAt = $state(0);
  let retrySeconds = $state(0);
  let retryTimer: ReturnType<typeof setInterval> | undefined;

  let companyForm = $state({
    CompanyName: '', Address: '', RUC: '',
    AdminFirstName: '', AdminLastName: '', AdminPassword: '', AdminPasswordRepeat: '',
  });

  const isValidEmail = $derived(/^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(emailForm.Email.trim()));

  // parseLogin decrypts the user payload with the same key the client generated, so it only has
  // to be unpredictable and exactly 32 characters (AES-256).
  const makeCipherKey = () => {
    const randomBytes = crypto.getRandomValues(new Uint8Array(16));
    return [...randomBytes].map((byte) => byte.toString(16).padStart(2, '0')).join('');
  };

  // The backend only reports the seconds left at response time, so the countdown runs here to
  // keep the notice honest and to re-enable the button by itself.
  const startRetryCountdown = (seconds: number) => {
    clearInterval(retryTimer);
    retrySeconds = Math.max(0, Math.round(seconds));
    if (retrySeconds === 0) return;
    retryTimer = setInterval(() => {
      retrySeconds -= 1;
      if (retrySeconds <= 0) clearInterval(retryTimer);
    }, 1000);
  };

  // The separating spaces go outside tr(): it trims what it returns, so padding kept inside the
  // translation string is dropped and the values end up glued to the words around them.
  const cooldownNotice = $derived(
    retrySeconds > 0 && lastSentAt
      ? tr('A registration email was sent at|Se envió un correo de registro a las')
        + ' ' + formatTime(lastSentAt, 'h:n')
        + tr(', wait|, espere') + ' ' + retrySeconds + ' '
        + tr('seconds to send another one.|segundos para enviar otro.')
      : '',
  );

  const sendRegistrationEmail = async () => {
    const email = emailForm.Email.trim();
    if (!isValidEmail) {
      verificationError = tr('Enter a valid email address.|Ingrese un correo electrónico válido.');
      return;
    }

    isBusy = true;
    verificationError = '';
    console.info('[Registration] Requesting sign-up code for:', email);
    try {
      const result = await requestSignUpCode(email);
      requestID = result.RequestID;
      isCodeStage = true;
      lastSentAt = result.SentAt;
      startRetryCountdown(result.RetryAfterSeconds);

      if (result.Sent) Notify.success(tr('Registration email sent.|Correo de registro enviado.'));
      console.info('[Registration] Sign-up request ready:', { requestID, sent: result.Sent });
    } catch (error) {
      // Rejects with the raw response, so String(error) would print "[object Object]".
      console.error('[Registration] Sign-up request failed:', error);
      verificationError = extractError(error);
    }
    isBusy = false;
  };

  const submitVerificationCode = async () => {
    const code = codeForm.Code.trim();
    if (code.length !== 8) {
      verificationError = tr('The verification code has 8 digits.|El código de verificación tiene 8 dígitos.');
      return;
    }

    isBusy = true;
    verificationError = '';
    console.info('[Registration] Verifying code for request:', requestID);
    try {
      await verifySignUpCode(requestID, code);
      currentStep = 2;
      console.info('[Registration] Email verified, moving to company information.');
    } catch (error) {
      // The request rejects with the raw response, so the readable text comes from extractError.
      console.error('[Registration] Code verification failed:', error);
      verificationError = extractError(error);
    }
    isBusy = false;
  };

  const submitCompany = async () => {
    if (companyForm.CompanyName.trim().length < 5) {
      Notify.failure(tr('The company name must be at least 5 characters.|El nombre de la empresa debe tener al menos 5 caracteres.'));
      return;
    }
    if (companyForm.AdminPassword.length < 6) {
      Notify.failure(tr('The password must be at least 6 characters.|La contraseña debe tener al menos 6 caracteres.'));
      return;
    }
    if (companyForm.AdminPassword !== companyForm.AdminPasswordRepeat) {
      Notify.failure(tr('The passwords do not match.|Las contraseñas no coinciden.'));
      return;
    }

    isBusy = true;
    const cipherKey = makeCipherKey();
    console.info('[Registration] Creating company for request:', requestID);
    try {
      const loginInfo = await createSignUpCompany(requestID, codeForm.Code.trim(), {
        CompanyName: companyForm.CompanyName.trim(),
        Address: companyForm.Address.trim(),
        RUC: companyForm.RUC.trim(),
        AdminFirstName: companyForm.AdminFirstName.trim(),
        AdminLastName: companyForm.AdminLastName.trim(),
        AdminPassword: companyForm.AdminPassword,
      }, cipherKey);

      // The response is a full login payload, so the last step runs authenticated against the
      // private initial-data endpoint instead of needing a public one of its own.
      await security.parseLogin(loginInfo, cipherKey);
      if (!security.isTokenValid()) {
        security.clearSession();
        throw new Error(tr('The session could not be started.|No se pudo iniciar la sesión.'));
      }

      localStorage.setItem('genixLastLoginCompanyID', String(loginInfo.CompanyID));
      currentStep = 3;
      console.info('[Registration] Company created and session started:', loginInfo.CompanyID);
    } catch (error) {
      // Server failures already raised their own toast on the way out of the http layer; only the
      // errors thrown right here carry a message that nobody has shown yet.
      console.error('[Registration] Company creation failed:', error);
      if (error instanceof Error) Notify.failure(error.message);
    }
    isBusy = false;
  };

  // The link already carries everything step 1 asks for, so it is checked without the visitor
  // having to press anything and the wizard opens on step 2 already verified.
  const verifyFromEmailLink = async () => {
    isVerifyingLink = true;
    await submitVerificationCode();
    // Still on step 1 means the check failed: the link is expired, already used or malformed, and
    // its code cannot be retried — so fall back to the plain "enter your email" form.
    if (currentStep === 1) {
      isCodeStage = false;
      requestID = 0;
      codeForm.Code = '';
    }
    isVerifyingLink = false;
  };

  onDestroy(() => clearInterval(retryTimer));

  // Arriving from the email link. The parent resolves the query params in its own onMount, so this
  // has to be an effect: the props are still empty when this component mounts.
  $effect(() => {
    const requestIDFromLink = presetRequestID;
    const codeFromLink = presetCode;
    if (!requestIDFromLink) return;

    untrack(() => {
      if (requestID === requestIDFromLink) return;
      requestID = requestIDFromLink;
      codeForm.Code = codeFromLink;
      isCodeStage = true;
      currentStep = 1;
      void verifyFromEmailLink();
    });
  });
</script>

<Modal
  {id}
  size={9}
  title="Create your company|Cree su empresa"
  hideTitle={true}
  css="!min-h-0 h-640 max-h-[86vh] outline-2 outline-solid outline-[#878dffa8]"
  bodyCss="!p-0 flex flex-col"
>
  <!-- grid-rows-1 makes the single row fill the fixed height, so the image column and the step
       content are always as tall as the dialog instead of as tall as the current step. -->
  <div class="grid grow min-h-0 grid-rows-1 md:grid-cols-[minmax(0,320px)_minmax(0,1fr)]">
    <!-- flex-col so the step list can be pushed to the bottom with mt-auto, leaving the photo
         visible between it and the title. -->
    <aside class="relative hidden overflow-hidden bg-slate-950 md:flex md:flex-col">
      <img
        class="absolute inset-0 h-full w-full object-cover object-[60%_center] opacity-70"
        src="/images/welcome-hero-v3.webp"
        alt=""
        aria-hidden="true"
      />
      <div class="absolute inset-0 bg-linear-to-b from-slate-950/70 via-slate-950/55 to-slate-950/85"></div>
      <p class="relative px-24 pt-24 text-xl font-bold leading-tight text-white">
        <T text="Create your company|Cree su empresa" />
      </p>
      <ol class="relative mt-auto grid gap-16 p-24 text-white">
        {#each steps as step}
          <li class="flex items-center gap-12">
            <span
              class="flex size-28 shrink-0 items-center justify-center rounded-full text-sm {currentStep > step.number
                ? 'bg-emerald-500 text-white'
                : currentStep === step.number
                  ? 'bg-indigo-500 text-white'
                  : 'border border-white/30 text-white/60'}"
            >
              {#if currentStep > step.number}
                <i class="icon-[fa--check]" aria-hidden="true"></i>
              {:else}
                {step.number}
              {/if}
            </span>
            <span class="leading-[1.35] {currentStep === step.number ? 'text-white' : 'text-white/60'}">
              <T text={step.title} />
            </span>
          </li>
        {/each}
      </ol>
    </aside>

    <!-- The steps differ a lot in height, so the taller ones scroll here rather than resizing the dialog. -->
    <section class="min-h-0 overflow-y-auto p-20 md:p-28">
      <p class="mb-14 pr-44 text-xl font-bold leading-tight text-slate-900 md:hidden">
        <T text="Create your company|Cree su empresa" />
      </p>
      <div class="mb-18 flex items-center gap-8 md:hidden" aria-label={tr('Progress|Progreso')}>
        {#each steps as step}
          <span class="h-4 grow rounded-full {currentStep >= step.number ? 'bg-indigo-600' : 'bg-slate-200'}"></span>
        {/each}
      </div>

      {#if isVerifyingLink}
        <div class="flex min-h-260 flex-col items-center justify-center gap-14 text-center">
          <i class="icon-[fa--refresh] animate-spin text-3xl text-indigo-600" aria-hidden="true"></i>
          <p class="text-lg font-semibold text-slate-900" aria-live="polite">
            <T text="Verifying registration…|Verificando registro…" />
          </p>
        </div>

      {:else if currentStep === 1}
        <form
          aria-label={tr('Email registration form|Formulario de registro por correo electrónico')}
          onsubmit={(event) => { event.preventDefault(); void (isCodeStage ? submitVerificationCode() : sendRegistrationEmail()); }}
        >
          <!-- pr-44 keeps the picker clear of the modal's absolutely positioned close button. -->
          <div class="mb-10 flex items-start justify-between gap-16 pr-44">
            <img
              class="-ml-8 -mt-24 -mb-12 h-190 w-auto shrink-0"
              src="/images/new-message.svg"
              alt=""
              aria-hidden="true"
            />

            <!-- Hidden once a code has been mailed: that request only exists on the server that
                 issued it, so switching backends mid-flow would silently invalidate it. -->
            {#if apiEndpointOptions.length > 0 && !isCodeStage}
              <fieldset class="shrink-0 mt-24">
                <legend class="mb-6 text-xs text-slate-500"><T text="Server|Servidor" /></legend>
                <div class="flex gap-6">
                  {#each apiEndpointOptions as apiEndpointOption}
                    <button
                      type="button"
                      class="flex min-h-44 w-105 items-center gap-6 rounded-[10px] border px-8 text-left text-xs leading-tight transition focus-visible:outline-3 focus-visible:outline-offset-3 focus-visible:outline-indigo-600 {selectedApiEndpointRoute === apiEndpointOption.route
                        ? 'border-indigo-500 bg-indigo-50 text-indigo-700'
                        : 'border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:bg-slate-50'}"
                      aria-pressed={selectedApiEndpointRoute === apiEndpointOption.route}
                      onclick={() => onApiEndpointChange?.(apiEndpointOption)}
                    >
                      <i class="icon-[fa--server] shrink-0" aria-hidden="true"></i>
                      <span class="min-w-0">{apiEndpointOption.name}</span>
                    </button>
                  {/each}
                </div>
              </fieldset>
            {/if}
          </div>

          <p class="mb-8 text-lg font-semibold text-slate-900">
            <T text="Verify your email address|Verifique su correo electrónico" />
          </p>
          {#if verificationError}
            <p
              class="mb-16 flex items-start gap-10 rounded-[8px] border border-amber-300 bg-amber-50 px-12 py-10 leading-[1.45] text-amber-800"
              role="alert"
            >
              <i class="icon-[fa--exclamation-triangle] mt-2 shrink-0" aria-hidden="true"></i>
              <span>{verificationError}</span>
            </p>
          {/if}
          <p class="mb-20 leading-[1.55] text-slate-600">
            <T text="We will send you a registration email. Click the link in that email, or copy its 8-digit code here, to continue.|Le enviaremos un correo de registro. Para continuar, haga clic en el enlace del correo o copie aquí su código de 8 dígitos." />
          </p>

          {#if !isCodeStage}
            <Input
              saveOn={emailForm}
              save="Email"
              label="Email address|Correo electrónico"
              type="email"
              required={true}
              inputCss="text-base"
            />
          {:else if emailForm.Email.trim()}
            <!-- Once the code is out the address is settled, so it reads as a plain confirmation
                 line instead of a greyed-out input that still looks editable. Skipped when the
                 visitor came from the email link, which carries the code but not the address. -->
            <p class="flex items-center justify-center gap-8 py-4 text-center text-base font-bold text-slate-500">
              <i class="icon-[fa--envelope-o] shrink-0" aria-hidden="true"></i>
              {emailForm.Email.trim()}
            </p>
          {/if}

          {#if cooldownNotice}
            <p class="mt-10 rounded-[8px] bg-amber-50 px-12 py-10 text-sm leading-[1.45] text-amber-800" aria-live="polite">
              {cooldownNotice}
            </p>
          {/if}

          {#if isCodeStage}
            <div class="mt-14">
              <Input
                saveOn={codeForm}
                save="Code"
                label="Verification code|Código de verificación"
                type="text"
                required={true}
                inputCss="text-base tracking-[4px]"
              />
            </div>
          {/if}

          <button
            type="submit"
            class="mt-20 flex h-44 w-full items-center justify-center gap-8 rounded-[10px] bg-indigo-600 px-18 text-white shadow-lg shadow-indigo-900/15 transition hover:bg-indigo-700 focus-visible:outline-3 focus-visible:outline-offset-3 focus-visible:outline-indigo-600 disabled:cursor-not-allowed disabled:bg-slate-300 disabled:shadow-none"
            disabled={isBusy || (!isCodeStage && !isValidEmail)}
          >
            <i class={isBusy ? 'icon-[fa--refresh] animate-spin' : 'icon-[fa--paper-plane-o]'} aria-hidden="true"></i>
            <span class="font-semibold">
              <T text={isCodeStage ? 'Verify code|Verificar código' : 'Send registration email|Enviar correo de registro'} />
            </span>
          </button>

          {#if isCodeStage}
            <div class="mt-10 flex items-center justify-between gap-12 text-sm">
              <button
                type="button"
                class="text-indigo-700 hover:underline disabled:text-slate-400 disabled:no-underline"
                disabled={isBusy || retrySeconds > 0}
                onclick={() => { void sendRegistrationEmail(); }}
              >
                <T text="Send the email again|Enviar el correo otra vez" />
              </button>
              <button
                type="button"
                class="text-indigo-700 hover:underline disabled:text-slate-400"
                disabled={isBusy}
                onclick={() => {
                  clearInterval(retryTimer);
                  isCodeStage = false; requestID = 0; codeForm.Code = '';
                  lastSentAt = 0; retrySeconds = 0;
                }}
              >
                <T text="Use a different email address|Usar otro correo electrónico" />
              </button>
            </div>
          {/if}
        </form>

      {:else if currentStep === 2}
        <form
          aria-label={tr('Company information form|Formulario de datos de la empresa')}
          onsubmit={(event) => { event.preventDefault(); void submitCompany(); }}
        >
          <p class="mb-8 text-lg font-semibold text-slate-900">
            <T text="Company information|Datos de la empresa" />
          </p>
          <p class="mb-16 text-[15px] leading-[1.55] text-slate-600">
            <T text="These details create your company and its administrator account.|Con estos datos se crean su empresa y su cuenta de administrador." />
          </p>

          <div class="grid grid-cols-24 gap-12">
            <Input css="col-span-24" saveOn={companyForm} save="CompanyName" required={true}
              label="Company name|Nombre de la empresa" type="text" inputCss="text-base"
            />
            <Input css="col-span-24 md:col-span-14" saveOn={companyForm} save="Address"
              label="Address (optional)|Dirección (opcional)" type="text" inputCss="text-base"
            />
            <Input css="col-span-24 md:col-span-10" saveOn={companyForm} save="RUC"
              label="Tax ID / RUC (optional)|RUC (opcional)" type="text" inputCss="text-base"
            />
          </div>

          <p class="mb-6 mt-24 border-t border-slate-200 pt-18 text-lg font-semibold text-slate-900">
            <T text='User "admin" details|Datos del usuario "admin"' />
          </p>
          <p class="mb-16 text-[15px] leading-[1.55] text-slate-600">
            <T text="This account administers the company. Its username is always admin; the name is only for display and can be filled in later.|Esta cuenta administra la empresa. Su usuario siempre es admin; el nombre es solo para mostrar y puede completarlo después." />
          </p>

          <div class="grid grid-cols-24 gap-12">
            <Input css="col-span-24 md:col-span-12" saveOn={companyForm} save="AdminFirstName"
              label="First name (optional)|Nombre (opcional)" type="text" inputCss="text-base"
            />
            <Input css="col-span-24 md:col-span-12" saveOn={companyForm} save="AdminLastName"
              label="Last name (optional)|Apellido (opcional)" type="text" inputCss="text-base"
            />
            <Input css="col-span-24 md:col-span-12" saveOn={companyForm} save="AdminPassword" required={true}
              label="Password|Contraseña" type="password" inputCss="text-base"
            />
            <Input css="col-span-24 md:col-span-12" saveOn={companyForm} save="AdminPasswordRepeat" required={true}
              label="Repeat password|Repita la contraseña" type="password" inputCss="text-base"
            />
          </div>

          <button
            type="submit"
            class="mt-20 flex h-44 w-full items-center justify-center gap-8 rounded-[10px] bg-indigo-600 px-18 text-white shadow-lg shadow-indigo-900/15 transition hover:bg-indigo-700 focus-visible:outline-3 focus-visible:outline-offset-3 focus-visible:outline-indigo-600 disabled:cursor-not-allowed disabled:bg-slate-300 disabled:shadow-none"
            disabled={isBusy}
          >
            <i class={isBusy ? 'icon-[fa--refresh] animate-spin' : 'icon-[fa--building-o]'} aria-hidden="true"></i>
            <span class="font-semibold"><T text="Create company|Crear empresa" /></span>
          </button>
        </form>

      {:else}
        <p class="mb-8 text-lg font-semibold text-slate-900">
          <T text="Initial data|Datos iniciales" />
        </p>
        <p class="mb-20 leading-[1.55] text-slate-600">
          <T text="Your company needs a branch, a warehouse and a cash register to start operating. You can accept the suggested names.|Su empresa necesita una sede, un almacén y una caja para poder operar. Puede aceptar los nombres sugeridos." />
        </p>
        <InitialDataForm onSaved={() => Env.navigate('/')} />
      {/if}
    </section>
  </div>
</Modal>
