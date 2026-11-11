<script lang="ts">
  import Page from '#domain/Page.svelte'
  import T from '#components/misc/T.svelte'
  import { CrmActivityTypesService } from '#services/crm/crm-activity.svelte.ts'
  import { CrmLostReasonsService } from '#services/crm/crm-pipeline.svelte.ts'
  import { CrmEmailTemplatesService } from '#services/crm/crm-email-template.svelte.ts'
  import ActivityTypesTable from './ActivityTypesTable.svelte'
  import EmailTemplatesTable from './EmailTemplatesTable.svelte'
  import LostReasonsTable from './LostReasonsTable.svelte'

  // Each dictionary is a small list of values the CRM picks from; the left column chooses which one to edit.
  const CRM_DICTIONARIES = [
    { id: 1, name: 'Lost reasons|Motivos de pérdida', icon: 'icon-[fa--times-circle]' },
    { id: 2, name: 'Activity types|Tipos de actividad', icon: 'icon-[fa--comments]' },
    { id: 3, name: 'Email templates|Plantillas de correo', icon: 'icon-[fa--envelope]' },
  ]
  let selectedDictionaryID = $state(1)

  const lostReasons = new CrmLostReasonsService(true)
  const activityTypes = new CrmActivityTypesService(true)
  const emailTemplates = new CrmEmailTemplatesService(true)
</script>

<Page title="Dictionaries|Diccionarios">
  <div class="flex gap-16 items-start">
    <div class="w-240 shrink-0 flex flex-col gap-4" aria-label="CRM dictionaries list">
      {#each CRM_DICTIONARIES as dictionary (dictionary.id)}
        <button type="button"
          class="flex items-center gap-8 px-12 py-8 rounded-md border text-left hover:border-line-strong {selectedDictionaryID === dictionary.id ? 'border-purple-border bg-purple-bg' : 'border-line bg-surface text-fg-soft'}"
          onclick={() => { selectedDictionaryID = dictionary.id }}>
          <i class={dictionary.icon}></i>
          <T text={dictionary.name} />
        </button>
      {/each}
    </div>
    <div class="flex-1 min-w-0">
      {#if selectedDictionaryID === 1}
        <LostReasonsTable {lostReasons} />
      {:else if selectedDictionaryID === 2}
        <ActivityTypesTable {activityTypes} />
      {:else if selectedDictionaryID === 3}
        <EmailTemplatesTable {emailTemplates} />
      {/if}
    </div>
  </div>
</Page>
