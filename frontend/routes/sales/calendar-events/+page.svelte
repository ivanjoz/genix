<script lang="ts">
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Page from '#domain/Page.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import Button from '#components/buttons/Button.svelte'
  import FilterInput from '#components/form/FilterInput.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import DateInput from '#components/form/DateInput.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatTime } from '#libs/helpers.ts'
  import { security } from '#libs/ui-runtime.svelte.ts'
  import { useUI } from '@genix/ui'
  import {
    CALENDAR_EVENT_TYPES,
    CalendarEventsService,
    postCalendarEvent,
    SUPER_ADMIN_USER_ID,
    type ICalendarEvent,
  } from '#services/sales/calendar-events.svelte.ts'

  const ui = useUI()
  const calendarEvents = new CalendarEventsService()
  const isSuperAdmin = (security.getUserInfo()?.ID || 0) === SUPER_ADMIN_USER_ID
  const eventTypeOptions = $derived(CALENDAR_EVENT_TYPES.map((eventType) => ({ id: eventType.id, name: tr(eventType.name) })))
  const eventTypeNameByID = $derived(new Map(eventTypeOptions.map((eventType) => [eventType.id, eventType.name])))

  let filterText = $state('')
  let form = $state({} as ICalendarEvent)
  // A global holiday is shown to everyone but edited only by the super-admin.
  const isFormReadOnly = $derived(!!form.IsGlobal && !isSuperAdmin)

  const openEvent = (event: ICalendarEvent) => {
    form = { ...event }
    ui.openSideLayer(1)
  }

  const saveEvent = async (eventToSave: ICalendarEvent) => {
    showLoading(tr('Saving event...|Guardando evento...'))
    try {
      await postCalendarEvent(eventToSave)
      await calendarEvents.fetchOnline()
      ui.openSideLayer(0)
    } catch (saveError) {
      console.error('[calendar-events] save error', { eventID: eventToSave.ID, saveError })
    } finally {
      hideLoading()
    }
  }

  const onSave = () => {
    if (!form.Date) { notifyFailure(tr('Select a date.|Seleccione una fecha.')); return }
    if (!form.EventType) { notifyFailure(tr('Select the event type.|Seleccione el tipo de evento.')); return }
    if (!form.Name?.trim()) { notifyFailure(tr('Enter a name.|Ingrese un nombre.')); return }
    saveEvent({ ...form, ss: 1 })
  }

  const onDelete = async () => {
    const confirmed = await confirmWarn({
      title: tr('Delete event|Eliminar evento'),
      message: tr(`Delete "${form.Name}"?|¿Eliminar "${form.Name}"?`),
    })
    if (confirmed) saveEvent({ ...form, ss: 0 })
  }

  const columns: ITableColumn<ICalendarEvent>[] = [
    {
      header: 'Date|Fecha', headerCss: 'w-120', css: 'px-6 whitespace-nowrap',
      getValue: (event) => formatTime(event.Date, 'Y-m-d') as string,
      mobile: { order: 1, css: 'col-span-12 ff-bold', icon: '[fa--calendar]' },
    },
    {
      header: 'Type|Tipo', headerCss: 'w-160', css: 'px-6',
      getValue: (event) => eventTypeNameByID.get(event.EventType) || '',
      mobile: { order: 2, css: 'col-span-12', contentCss: 'text-right' },
    },
    {
      header: 'Name|Nombre', css: 'px-6', getValue: (event) => event.Name,
      mobile: { order: 3, css: 'col-span-24' },
    },
    {
      header: 'Scope|Alcance', headerCss: 'w-120', css: 'px-6',
      getValue: (event) => tr(event.IsGlobal ? 'Global|Global' : 'Company|Empresa'),
      mobile: { order: 4, css: 'col-span-24', labelLeft: 'Alcance:' },
    },
  ]
</script>

<Page title="Commercial Calendar|Calendario Comercial">
  <div class="flex items-center justify-between mb-8">
    <FilterInput bind:value={filterText} css="w-256" />
    <Button color="green" icon="icon-[fa--plus]" name="New|Nuevo"
      onClick={() => { form = { ss: 1, EventType: 2, IsGlobal: false } as ICalendarEvent; ui.openSideLayer(1) }} />
  </div>

  <Layer type="content">
    <VTable css="w-full" maxHeight="calc(100vh - 13rem)"
      columns={columns} data={calendarEvents.records}
      {filterText} getFilterContent={(event) => event.Name}
      selected={form} isSelected={(event, selected) => typeof selected === 'object'
        && event.ID === selected.ID && !!event.IsGlobal === !!selected.IsGlobal}
      onRowClick={openEvent}
    />
  </Layer>

  <Layer type="side" id={1} width={560}
    css="px-8 py-8 md:px-14 md:py-10"
    title={form?.ID ? form.Name : tr('New event|Nuevo evento')}
    titleCss="h2 ff-bold"
    onSave={isFormReadOnly ? undefined : onSave}
    onDelete={form?.ID && !isFormReadOnly ? onDelete : undefined}
    onClose={() => { form = {} as ICalendarEvent }}
  >
    {#if isFormReadOnly}
      <div class="text-fg-muted mt-8">
        {tr('Global holidays are shared by every company and can only be edited by the administrator.|Los feriados globales son compartidos por todas las empresas y solo el administrador puede editarlos.')}
      </div>
    {/if}
    <div class="grid grid-cols-24 gap-10 mt-12">
      <DateInput bind:saveOn={form} save="Date" css="col-span-24 md:col-span-12" label="Date|Fecha"
        required={true} disabled={isFormReadOnly} />
      <SearchSelect bind:saveOn={form} save="EventType" css="col-span-24 md:col-span-12" label="Type|Tipo"
        keyId="id" keyName="name" options={eventTypeOptions} required={true} disabled={isFormReadOnly} />
      <Input bind:saveOn={form} save="Name" css="col-span-24" label="Name|Nombre"
        required={true} disabled={isFormReadOnly} />
      <!-- Only on a new event: moving a saved event between partitions would leave its copy behind. -->
      {#if isSuperAdmin && !form?.ID}
        <Checkbox bind:saveOn={form} save="IsGlobal" css="col-span-24"
          label="Global holiday (every company)|Feriado global (todas las empresas)" />
      {/if}
    </div>
  </Layer>
</Page>
