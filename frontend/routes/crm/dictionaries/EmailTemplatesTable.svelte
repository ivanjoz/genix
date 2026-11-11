<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import VariableTextInput from '#components/editor/VariableTextInput.svelte'
  import Input from '#components/form/Input.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { EMAIL_PLACEHOLDERS, unknownPlaceholders } from '#services/crm/crm-email-template.ts'
  import {
    postCrmEmailTemplate, type CrmEmailTemplatesService, type ICrmEmailTemplate,
  } from '#services/crm/crm-email-template.svelte.ts'

  // The emails a seller sends from an opportunity, or a stage task sends for them. The body is rich
  // HTML. Placeholders are inserted only from the variable pickers of the subject and the body, shown
  // as "Cliente: Email" labels and saved as {{client.email}}; the server fills them per opportunity.
  let { emailTemplates }: { emailTemplates: CrmEmailTemplatesService } = $props()

  const ui = useUI()
  const TEMPLATE_LAYER_ID = 1
  const makeEmptyTemplate = (): ICrmEmailTemplate => ({ ID: 0, Name: '', Subject: '', BodyHTML: '', ss: 1, upd: 0 })
  let templateForm = $state(makeEmptyTemplate())

  const openTemplate = (emailTemplate: ICrmEmailTemplate) => {
    templateForm = structuredClone($state.snapshot(emailTemplate))
    ui.openSideLayer(TEMPLATE_LAYER_ID)
  }

  const saveTemplate = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({ title: tr('Delete|Eliminar'), message: tr(`Delete "${templateForm.Name}"?|¿Eliminar "${templateForm.Name}"?`) }))) return
    const unknown = isDelete ? [] : unknownPlaceholders(`${templateForm.Subject}\n${templateForm.BodyHTML}`)
    if (unknown.length) {
      notifyFailure(tr(`Unknown placeholder {{${unknown[0]}}}.|La variable {{${unknown[0]}}} no existe.`))
      return
    }
    showLoading(tr('Saving template...|Guardando plantilla...'))
    try {
      const savedTemplate = await postCrmEmailTemplate(emailTemplates, { ...$state.snapshot(templateForm), ss: isDelete ? 0 : 1 })
      if (isDelete) ui.openSideLayer(0)
      else templateForm = structuredClone(savedTemplate)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const PLACEHOLDERS_HINT = 'Quote variables only work when sending a quote.|Las variables de cotización solo funcionan al enviar una cotización.'

  const columns: ITableColumn<ICrmEmailTemplate>[] = [
    { header: 'Template|Plantilla', css: 'px-6', highlight: true, getValue: (emailTemplate) => emailTemplate.Name },
    { header: 'Subject|Asunto', css: 'px-6', getValue: (emailTemplate) => emailTemplate.Subject },
  ]
</script>

<div class="flex items-center justify-between mb-8" aria-label="Email templates toolbar">
  <div class="h3">{tr('Email templates|Plantillas de correo')}</div>
  <Button color="green" icon="icon-[fa--plus]" name="New|Nuevo" label="Opens the side layer to create an email template."
    onClick={() => openTemplate(makeEmptyTemplate())} />
</div>
<VTable {columns} data={emailTemplates.records} css="w-full" maxHeight="calc(100vh - 8rem - 16px)"
  selected={templateForm.ID} isSelected={(emailTemplate, selectedID) => emailTemplate.ID === selectedID} onRowClick={openTemplate} />

<Layer
  id={TEMPLATE_LAYER_ID}
  type="side"
  sideLayerSize={820}
  title={templateForm.ID > 0 ? templateForm.Name : 'New template|Nueva plantilla'}
  titleCss="h2 mb-6"
  css="px-12 py-10"
  contentCss="px-0"
  onSave={() => saveTemplate()}
  onDelete={templateForm.ID > 0 ? () => saveTemplate(true) : undefined}
  onClose={() => { templateForm = makeEmptyTemplate() }}
>
  <div class="flex flex-col gap-10 mt-8" aria-label="Email template form">
    <Input bind:saveOn={templateForm} save="Name" label="Name|Nombre" required={true} />
    <VariableTextInput saveOn={templateForm} save="Subject" label="Subject|Asunto"
      variables={EMAIL_PLACEHOLDERS} variablesHint={PLACEHOLDERS_HINT} />
    {#await import('#components/editor/HTMLEditor.svelte') then htmlEditorModule}
      {@const HTMLEditor = htmlEditorModule.default}
      <HTMLEditor saveOn={templateForm} save="BodyHTML" css="min-h-280"
        variables={EMAIL_PLACEHOLDERS} variablesHint={PLACEHOLDERS_HINT} />
    {/await}
  </div>
</Layer>
