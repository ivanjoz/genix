<script lang="ts">
  import Button from '#components/buttons/Button.svelte';
  import Card from '#components/cards/Card.svelte';
  import FilterInput from '#components/form/FilterInput.svelte';
  import SearchSelect from '#components/form/SearchSelect.svelte';
  import T from '#components/misc/T.svelte';
  import Page from '#domain/Page.svelte';
  import { tr } from '#core/store.svelte.ts';
  import { formatTime } from '#libs/helpers.ts';
  import { hideLoading, showLoading } from '@genix/ui/notify';
  import { EmpresasService } from '#routes/system/companies/empresas.svelte.ts';
  import { SENT_EMAILS_PAGE_SIZE, describeDelivery, listAllRecipients, sentEmailMatchesFilter, type ISentEmail } from './email-inbox.ts';
  import { loadSentEmail, loadSentEmails } from './email-inbox.svelte.ts';

  const empresasService = new EmpresasService()

  // Company 1 is the SaaS: platform emails (sign-up, contact form) are stored there.
  let companyID = $state(1)
  let sentEmails = $state<ISentEmail[]>([])
  let hasOlderEmails = $state(false)
  let filterText = $state('')
  let openedEmail = $state<ISentEmail | undefined>(undefined)
  let openedEmailID = $state(0)

  const shownEmails = $derived(sentEmails.filter((sentEmail) => sentEmailMatchesFilter(sentEmail, filterText)))

  // Reload starts again from the newest email; Load older appends the page before the oldest shown.
  const loadPage = async (appendOlder: boolean) => {
    showLoading(tr('Loading the emails...|Cargando los correos...'))
    try {
      const beforeID = appendOlder ? sentEmails[sentEmails.length - 1]?.ID : undefined
      const page = await loadSentEmails(companyID, beforeID)
      sentEmails = appendOlder ? [...sentEmails, ...page] : page
      hasOlderEmails = page.length === SENT_EMAILS_PAGE_SIZE
    } catch {
      // genix-ui already notified the backend's error message.
    }
    hideLoading()
  }
  void loadPage(false)

  const selectCompany = (selectedCompanyID: number) => {
    companyID = selectedCompanyID
    openedEmailID = 0
    openedEmail = undefined
    void loadPage(false)
  }

  // The list carries no bodies: the preview fetches the opened email. A click on another email while
  // one loads wins, the late answer is dropped.
  const openEmail = async (sentEmailID: number) => {
    openedEmailID = sentEmailID
    openedEmail = undefined
    try {
      const loadedEmail = await loadSentEmail(companyID, sentEmailID)
      if (openedEmailID === sentEmailID) { openedEmail = loadedEmail }
    } catch {
      // genix-ui already notified the backend's error message.
    }
  }
</script>

<Page title="Email Inbox|Buzón Email">
  <!-- Two columns that fill the viewport below the header: the list and the preview scroll apart. -->
  <div class="flex flex-col md:flex-row gap-12 md:h-[calc(100vh-var(--header-height)-20px)]">
    <div class="flex flex-col gap-8 md:w-400 shrink-0 min-h-0 max-h-[50vh] md:max-h-none bg-surface border border-line rounded p-10">
      <SearchSelect options={empresasService.empresas} keyId="id" keyName="Name" label="Company|Empresa"
        selected={companyID} onChange={(company) => { if (company) { selectCompany(company.id) } }} />
      <div class="flex gap-8 items-center">
        <FilterInput bind:value={filterText} placeholder="Filter by subject or address|Filtrar por asunto o correo" css="flex-1" />
        <Button color="blue" icon="icon-[fa--refresh]" label="Reloads the newest emails.|Recarga los correos más recientes." onClick={() => { void loadPage(false) }} />
      </div>
      <div class="flex flex-col gap-2 flex-1 min-h-0 overflow-y-auto">
        {#each shownEmails as sentEmail (sentEmail.ID)}
          {@const delivery = describeDelivery(sentEmail)}
          <Card label={`Opens the email ${sentEmail.Subject}.|Abre el correo ${sentEmail.Subject}.`} onClick={() => { void openEmail(sentEmail.ID) }}
            css="rounded-r border-l-3 px-10 py-6 cursor-pointer {sentEmail.ID === openedEmailID ? 'border-blue-solid bg-blue-bg text-blue-fg' : 'border-transparent hover:bg-surface-muted'}">
            <div class="flex justify-between gap-8 text-[14px]">
              <span class="truncate text-fg-muted">{listAllRecipients(sentEmail)}</span>
              <span class="shrink-0 text-fg-muted">{formatTime(sentEmail.upd, 'Y-m-d h:n')}</span>
            </div>
            <div class="font-semibold leading-tight truncate">{sentEmail.Subject || tr('(no subject)|(sin asunto)')}</div>
            <div class="text-[14px] {delivery.css}">{tr(delivery.text)}</div>
          </Card>
        {:else}
          <div class="text-fg-muted px-4 py-6"><T text="No emails.|No hay correos." /></div>
        {/each}
        {#if hasOlderEmails}
          <Button css="mt-6 self-center" icon="icon-[fa--chevron-down]" name="Load older|Cargar anteriores"
            label="Loads the previous emails.|Carga los correos anteriores." onClick={() => { void loadPage(true) }} />
        {/if}
      </div>
    </div>

    <div class="flex-1 min-w-0 min-h-[60vh] md:min-h-0 flex flex-col bg-surface border border-line rounded">
      {#if openedEmail}
        {@const delivery = describeDelivery(openedEmail)}
        <div class="px-16 py-12 border-b border-line flex flex-col gap-4">
          <div class="h2">{openedEmail.Subject || tr('(no subject)|(sin asunto)')}</div>
          <div><span class="text-fg-muted"><T text="From|De" />:</span> {openedEmail.Sender}</div>
          <div><span class="text-fg-muted"><T text="To|Para" />:</span> {(openedEmail.To || []).join(', ')}</div>
          {#if openedEmail.Cc?.length}<div><span class="text-fg-muted">Cc:</span> {openedEmail.Cc.join(', ')}</div>{/if}
          {#if openedEmail.Bcc?.length}<div><span class="text-fg-muted">Cco:</span> {openedEmail.Bcc.join(', ')}</div>{/if}
          {#if openedEmail.AttachmentNames?.length}
            <div><span class="text-fg-muted"><T text="Attachments|Adjuntos" />:</span> {openedEmail.AttachmentNames.join(', ')}</div>
          {/if}
          <div class="text-[14px]">
            <span class="text-fg-muted">{formatTime(openedEmail.upd, 'Y-m-d h:n')} ·</span>
            <span class={delivery.css}>{tr(delivery.text)}</span>
          </div>
          {#if openedEmail.SendError}<div class="text-[14px] text-red-fg">{openedEmail.SendError}</div>{/if}
        </div>
        <!-- An empty sandbox: the email's HTML and styles render apart from the app and run no scripts. -->
        <iframe title={tr('Email body|Cuerpo del correo')} sandbox="" srcdoc={openedEmail.HTMLBody || ''} class="flex-1 w-full min-h-0 bg-white rounded-b"></iframe>
      {:else}
        <div class="m-auto text-fg-muted px-12 py-10">
          {#if openedEmailID}<T text="Loading the email...|Cargando el correo..." />{:else}<T text="Pick an email to see it.|Elige un correo para verlo." />{/if}
        </div>
      {/if}
    </div>
  </div>
</Page>
