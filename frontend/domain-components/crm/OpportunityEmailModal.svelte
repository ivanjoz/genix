<script lang="ts">
  import { untrack } from 'svelte'
  import { useUI } from '@genix/ui'
  import { hideLoading, notifyFailure, notifySuccess, showLoading } from '@genix/ui/notify'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import { tr } from '#core/store.svelte.ts'
  import type { ICrmActivity } from '#services/crm/crm-activity.svelte.ts'
  import { parseEmailAddresses } from '#services/crm/crm-email-template.ts'
  import {
    getCrmOpportunityEmailDraft, postCrmOpportunityEmail, type CrmEmailTemplatesService,
  } from '#services/crm/crm-email-template.svelte.ts'

  // Writes and sends one email from an opportunity: pick a template, the server renders it for the
  // opportunity (client, owner, company), the seller edits it and sends. The email is logged as an
  // Email activity. With taskID it fulfils that "send email" task (presetTemplateID comes from it).
  let { modalID, opportunityID, emailTemplates, presetTemplateID = 0, taskID = 0, onSent }: {
    modalID: number
    opportunityID: number
    emailTemplates: CrmEmailTemplatesService
    presetTemplateID?: number
    taskID?: number
    onSent: (activity: ICrmActivity) => void
  } = $props()

  const ui = useUI()
  const makeEmptyEmailForm = () => ({ TemplateID: 0, To: '', Cc: '', Subject: '', HTMLBody: '' })
  let emailForm = $state(makeEmptyEmailForm())

  const templateOptions = $derived(emailTemplates.records.map((emailTemplate) => ({ id: emailTemplate.ID, name: emailTemplate.Name })))

  // Renders the template for this opportunity. The recipients typed so far are kept; the subject and
  // body are replaced, since they are what the template is for.
  const loadDraft = async (templateID: number) => {
    emailForm.TemplateID = templateID
    if (!templateID) return
    showLoading(tr('Preparing email...|Preparando correo...'))
    try {
      const draft = await getCrmOpportunityEmailDraft(opportunityID, templateID)
      if (!emailForm.To.trim()) emailForm.To = draft.To.join(', ')
      emailForm.Subject = draft.Subject
      emailForm.HTMLBody = draft.HTMLBody
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  // Another opportunity or task starts a fresh email, from the task's template when it has one.
  $effect(() => {
    const openedFor = [opportunityID, presetTemplateID, taskID]
    untrack(() => {
      emailForm = makeEmptyEmailForm()
      if (openedFor[1]) loadDraft(presetTemplateID)
    })
  })

  const sendEmail = async () => {
    const recipients = parseEmailAddresses(emailForm.To)
    if (!recipients.length) { notifyFailure(tr('Enter at least one recipient.|Ingrese al menos un destinatario.')); return }
    if (!emailForm.Subject.trim()) { notifyFailure(tr('Enter the subject.|Ingrese el asunto.')); return }
    showLoading(tr('Sending email...|Enviando correo...'))
    try {
      const activity = await postCrmOpportunityEmail({
        OpportunityID: opportunityID, TemplateID: emailForm.TemplateID, To: recipients, Cc: parseEmailAddresses(emailForm.Cc),
        Subject: emailForm.Subject, HTMLBody: emailForm.HTMLBody, TaskID: taskID,
      })
      notifySuccess(tr('Email sent.|Correo enviado.'))
      ui.closeModal(modalID)
      onSent(activity)
    } catch {
      // genix-ui already notified the backend's error message; the email stays to retry.
    }
    hideLoading()
  }
</script>

<Modal id={modalID} size={7} bodyCss="px-16 py-14" title={tr('Send email|Enviar correo')} onSave={sendEmail}>
  <div class="grid grid-cols-24 gap-10" aria-label="Send email form">
    <SearchSelect options={templateOptions} keyId="id" keyName="name" label="Template|Plantilla" css="col-span-24"
      selected={emailForm.TemplateID} onChange={(option) => loadDraft(option?.id ?? 0)} />
    <Input bind:saveOn={emailForm} save="To" label="To|Para" required={true} css="col-span-24 md:col-span-12" />
    <Input bind:saveOn={emailForm} save="Cc" label="CC" css="col-span-24 md:col-span-12" />
    <Input bind:saveOn={emailForm} save="Subject" label="Subject|Asunto" required={true} css="col-span-24" />
    <div class="col-span-24">
      {#await import('#components/editor/HTMLEditor.svelte') then htmlEditorModule}
        {@const HTMLEditor = htmlEditorModule.default}
        <HTMLEditor saveOn={emailForm} save="HTMLBody" css="min-h-280" />
      {/await}
    </div>
  </div>
</Modal>
