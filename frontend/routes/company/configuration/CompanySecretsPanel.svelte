<script lang="ts">
import Button from '$components/buttons/Button.svelte';
import Checkbox from '$components/form/Checkbox.svelte';
import FileDropZone from '$components/files/FileDropZone.svelte';
import Input from '$components/form/Input.svelte';
import T from '$components/misc/T.svelte';
import { tr } from '$core/store.svelte';
import { formatTime, Loading, Notify, readFileAsBase64 } from '$libs/helpers';
import { untrack } from 'svelte';
import { CompanySecretsService, postCompanySecrets, testCompanySecrets } from './company-secrets.svelte';
import {
  activeSecrets, certificateDaysLeft, EXPIRY_WARNING_DAYS, retiredSecrets,
  SUNAT_ENV_BETA, SUNAT_ENV_PRODUCTION, validateSecretsForm,
  type ICompanySecretsForm,
} from './company-secrets';

  const service = new CompanySecretsService()

  const current = $derived(activeSecrets(service.all))
  const retired = $derived(retiredSecrets(service.all))

  // Passwords and certificate are never prefilled — the server does not return them, and
  // an empty one means "keep the stored one" to the endpoint.
  let form = $state({
    ID: 0, SolUser: "", SolPassword: "", CertPassword: "", Environment: SUNAT_ENV_BETA,
  } as Omit<ICompanySecretsForm, "Certificate">)

  let certificateFile = $state<File | undefined>(undefined)
  let isSaving = $state(false)

  // Only when the row being edited changes — a rotation inserts a new id — so a reload
  // never overwrites what is half typed in the form.
  let syncedSecretsID = -1
  $effect(() => {
    const secrets = current
    if (!secrets || secrets.ID === syncedSecretsID) return

    untrack(() => {
      syncedSecretsID = secrets.ID
      // Replaced wholesale rather than mutated field by field. Input only re-reads `saveOn` when
      // the object identity changes — its own effect returns early while `lastSaveOn === saveOn` —
      // so assigning form.SolUser after the inputs have mounted left them showing the empty value
      // they mounted with, and a stored SOL user looked like it had not been saved.
      form = {
        ID: secrets.ID,
        SolUser: secrets.SolUser,
        SolPassword: "",
        CertPassword: "",
        Environment: secrets.Environment || SUNAT_ENV_BETA,
      }
    })
  })

  const daysLeft = $derived(current?.HasCert ? certificateDaysLeft(current, Date.now() / 1000) : 0)

  const saveSecrets = async () => {
    const payload: ICompanySecretsForm = {
      ...form,
      Certificate: certificateFile ? await readFileAsBase64(certificateFile) : "",
    }

    // Checked here so an unusable row never reaches the endpoint; the backend checks the
    // certificate itself again — password, expiry and RUC — because it is the one that can.
    const problem = validateSecretsForm(payload, current)
    if (problem) {
      Notify.failure(tr(problem))
      return
    }

    isSaving = true
    Loading.standard(tr("Saving...|Guardando..."))
    try {
      await postCompanySecrets(payload)
      // Reloaded rather than patched: uploading a certificate retires the previous row and
      // inserts a new one, so both ends of the list moved.
      await service.load()
      form.SolPassword = ""
      form.CertPassword = ""
      certificateFile = undefined
      Notify.success(tr("Credentials saved|Credenciales guardadas"))
    } catch (error) {
      // Reported by POST.
    }
    Loading.remove()
    isSaving = false
  }

  const testSecrets = async () => {
    Loading.standard(tr("Checking with SUNAT...|Consultando a SUNAT..."))
    try {
      const result = await testCompanySecrets()
      Notify.success(tr("SUNAT accepted the credentials|SUNAT aceptó las credenciales") + `: ${result.Message}`)
    } catch (error) {
      // Reported by POST.
    }
    Loading.remove()
  }
</script>

<section class="rounded-[12px] border border-slate-200 bg-white p-16 shadow-sm"
  aria-label="SUNAT credentials: SOL user, environment and digital certificate">
  <div class="flex items-start gap-10 mb-12">
    <div class="flex-1 min-w-0">
      <div class="h3 ff-bold mb-4"><T text="SUNAT Credentials|Credenciales SUNAT" /></div>
      <div class="text-[13px] text-slate-500">
        <T text="The certificate is stored encrypted and is never returned. Uploading a new one retires the previous certificate instead of replacing it.|El certificado se guarda cifrado y nunca se devuelve. Cargar uno nuevo retira el anterior en vez de reemplazarlo." />
      </div>
    </div>
    <Button color="purple" icon="icon-[fa--plug]" name="Test|Probar"
      label="Checks the credentials against SUNAT without issuing any document."
      disabled={!current?.HasCert} onClick={testSecrets} />
  </div>

  {#if current?.HasCert}
    <!-- Red once expired, amber inside the last month: SUNAT rejects every document signed
         with an expired certificate, and buying the next one takes days. -->
    {@const isExpired = daysLeft < 0}
    {@const isExpiring = !isExpired && daysLeft <= EXPIRY_WARNING_DAYS}
    <div class="rounded-[8px] border p-12 mb-14 {isExpired
      ? 'border-red-300 bg-red-50'
      : isExpiring ? 'border-amber-300 bg-amber-50' : 'border-slate-200 bg-slate-50'}">
      <div class="flex flex-wrap items-baseline gap-x-14 gap-y-4">
        <span class="text-[14px] ff-bold">RUC {current.CertRUC || "—"}</span>
        <span class="text-[14px] text-slate-600">
          <T text="Valid until|Vigente hasta" /> {formatTime(current.CertValidTo, "d-M-Y")}
        </span>
        {#if isExpired}
          <span class="text-[14px] ff-bold text-red-600"><T text="EXPIRED|VENCIDO" /></span>
        {:else if isExpiring}
          <span class="text-[14px] ff-bold text-amber-700">
            {daysLeft} <T text="days left|días restantes" />
          </span>
        {/if}
      </div>
      <div class="text-[13px] text-slate-500 mt-4 truncate">{current.CertSubject}</div>
    </div>
  {:else}
    <div class="rounded-[8px] border border-dashed border-slate-300 p-12 mb-14 text-center text-slate-500">
      <T text="No certificate loaded. A sale cannot be invoiced until the company has one.|Sin certificado cargado. No se puede facturar una venta hasta que la empresa tenga uno." />
    </div>
  {/if}

  <div class="grid grid-cols-24 gap-10 content-start">
    <Input css="col-span-24 md:col-span-12" label="SOL User|Usuario SOL" save="SolUser"
      bind:saveOn={form} required={true} />
    <Input css="col-span-24 md:col-span-12" label="SOL Password|Clave SOL" type="password"
      save="SolPassword" bind:saveOn={form}
      placeholder={current ? tr("Unchanged|Sin cambios") : ""} />

    <div class="col-span-24 md:col-span-12">
      <FileDropZone bind:selectedFile={certificateFile} extensions={['pfx', 'p12']}
        label="Drop the certificate here or click to select it|Suelte el certificado aquí o haga clic para seleccionarlo"
        hint="A .pfx or .p12 file|Un archivo .pfx o .p12" />
    </div>

    <!-- Beside the drop zone, not under it: the password belongs to the file that was
         just dropped, and the environment is the other half of the same decision. -->
    <div class="col-span-24 md:col-span-12 flex flex-col justify-center gap-10">
      <Input label="Certificate Password|Clave del Certificado"
        type="password" save="CertPassword" bind:saveOn={form}
        placeholder={current?.HasCert ? tr("Unchanged|Sin cambios") : ""} />

      <Checkbox label="Use SUNAT Beta|Usar Beta SUNAT"
        checked={form.Environment === SUNAT_ENV_BETA}
        onToggle={(isChecked) => {
          form.Environment = isChecked ? SUNAT_ENV_BETA : SUNAT_ENV_PRODUCTION
        }} />
    </div>

    <div class="col-span-24 text-[13px] text-slate-500">
      <T text="Beta issues against SUNAT's test environment: the documents are valid for testing and have no tax effect.|Beta emite contra el ambiente de pruebas de SUNAT: los comprobantes sirven para probar y no tienen efecto tributario." />
    </div>

    <div class="col-span-24 flex justify-end">
      <Button color="blue" icon="icon-[fa--floppy-o]" name="Save|Guardar"
        label="Saves the SUNAT credentials and uploads the certificate."
        disabled={isSaving} onClick={saveSecrets} />
    </div>
  </div>

  {#if retired.length > 0}
    <div class="mt-14 border-t border-slate-200 pt-10">
      <div class="text-[13px] ff-bold text-slate-500 mb-4">
        <T text="Retired certificates|Certificados retirados" />
      </div>
      {#each retired as retiredSecret (retiredSecret.ID)}
        <div class="text-[13px] text-slate-500">
          RUC {retiredSecret.CertRUC || "—"} ·
          {formatTime(retiredSecret.CertValidFrom, "d-M-Y")} →
          {formatTime(retiredSecret.CertValidTo, "d-M-Y")}
        </div>
      {/each}
    </div>
  {/if}
</section>
