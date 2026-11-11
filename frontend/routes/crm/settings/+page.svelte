<script lang="ts">
  import Page from '#domain/Page.svelte'
  import { useUI } from '@genix/ui'
  import { CrmActivityTypesService } from '#services/crm/crm-activity.svelte.ts'
  import { CrmEmailTemplatesService } from '#services/crm/crm-email-template.svelte.ts'
  import { CrmCustomFieldsService, CrmPipelinesService } from '#services/crm/crm-pipeline.svelte.ts'
  import CustomFieldsTab from './CustomFieldsTab.svelte'
  import PipelinesTab from './PipelinesTab.svelte'

  const ui = useUI()
  const pipelines = new CrmPipelinesService(true)
  const customFields = new CrmCustomFieldsService(true)
  const activityTypes = new CrmActivityTypesService(true)
  const emailTemplates = new CrmEmailTemplatesService(true)

  const pageOptions = [
    { id: 1, name: 'Pipelines|Flujos' },
    { id: 2, name: 'Custom fields|Campos personalizados' },
  ]
</script>

<Page title="Pipelines|Flujos" options={pageOptions}>
  {#if ui.state.pageOptionSelected === 2}
    <CustomFieldsTab {customFields} />
  {:else}
    <PipelinesTab {pipelines} {customFields} {activityTypes} {emailTemplates} />
  {/if}
</Page>
