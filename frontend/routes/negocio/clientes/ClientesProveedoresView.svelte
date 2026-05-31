<script lang="ts">
  import Layer from '$components/layers/Layer.svelte'
  import Input from '$components/form/Input.svelte'
  import SearchSelect from '$components/form/SearchSelect.svelte'
  import VTable from '$components/vTable/VTable.svelte'
  import type { ITableColumn } from '$components/vTable/types'
  import { Core, tr } from '$core/store.svelte'
  import T from '$components/misc/T.svelte'
  import Page from '$domain/Page.svelte'
  import { Loading, Notify, formatTime } from '$libs/helpers'
  import FilterInput from '$components/form/FilterInput.svelte'
  import Button from '$components/buttons/Button.svelte'
  import { PaisCiudadesService } from '../sedes-almacenes/sedes-almacenes.svelte'
  import { ClientProviderService, PersonType, postClientProviders, type IClientProvider } from './clientes-proveedores.svelte'

  interface IClientProvidersViewProps {
    clientProviderType: number
    pageTitle: string
    layerTitleSingular: string
  }

  const { clientProviderType, pageTitle, layerTitleSingular }: IClientProvidersViewProps = $props()

  let clientProviderService = $state<ClientProviderService | null>(null)
  const paisCiudadesService = new PaisCiudadesService(true)

  let filterText = $state('')
  let clientProviderForm = $state<IClientProvider>(createEmptyClientProviderForm())

  const filteredClientProviders = $derived.by(() => {
    const normalizedFilterText = filterText.trim().toLowerCase()
    if (!normalizedFilterText) {
      return clientProviderService?.records || []
    }

    return (clientProviderService?.records || []).filter((clientProviderRecord) => {
      return [
        clientProviderRecord.Name,
        clientProviderRecord.RegistryNumber,
        clientProviderRecord.Email,
        String(clientProviderRecord.ID),
      ].some((fieldValue) => String(fieldValue || '').toLowerCase().includes(normalizedFilterText))
    })
  })

  const personTypeOptions = [
    { ID: PersonType.PERSON, Name: 'Person|Persona' },
    { ID: PersonType.COMPANY, Name: 'Company|Empresa' },
  ]

  $effect(() => {
    // Instantiate the shared client-provider source from the page config so both routes reuse the same data flow.
    clientProviderService = new ClientProviderService(clientProviderType, true)
  })

  function createEmptyClientProviderForm(): IClientProvider {
    return {
      ID: 0,
      Type: clientProviderType,
      Name: '',
      RegistryNumber: '',
      PersonType: PersonType.PERSON,
      Email: '',
      CountryID: 604,
      CityID: '',
      ss: 1,
      upd: 0,
    }
  }

  function resetEntityForm() {
    // Keep the route-selected client-provider type fixed so this view never crosses data with the sibling route.
    clientProviderForm = createEmptyClientProviderForm()
  }

  function openCreateClientProviderLayer() {
    resetEntityForm()
    Core.openSideLayer(11)
  }

  function openEditClientProviderLayer(selectedClientProvider: IClientProvider) {
    // Clone the selected record so table state remains immutable until the save succeeds.
    clientProviderForm = {
      ...selectedClientProvider,
      Type: clientProviderType,
    }
    Core.openSideLayer(11)
  }

  async function saveClientProvider() {
    clientProviderForm.Name = (clientProviderForm.Name || '').trim()
    clientProviderForm.Email = (clientProviderForm.Email || '').trim().toLowerCase()
    clientProviderForm.RegistryNumber = (clientProviderForm.RegistryNumber || '').trim()
    clientProviderForm.Type = clientProviderType

    const singularLabel = tr(layerTitleSingular).toLowerCase()
    if (!clientProviderForm.Name) {
      Notify.failure(tr(`Please enter the name of the ${singularLabel}.|Debe ingresar el nombre del ${singularLabel}.`))
      return
    }
    if (!clientProviderForm.Email || !clientProviderForm.Email.includes('@')) {
      Notify.failure(tr('Please enter a valid email.|Debe ingresar un email válido.'))
      return
    }
    if (!clientProviderForm.CountryID || clientProviderForm.CountryID <= 0) {
      Notify.failure(tr('Please enter a valid Country.|Debe ingresar un CountryID válido.'))
      return
    }
    if (!clientProviderForm.CityID || !String(clientProviderForm.CityID).trim()) {
      Notify.failure(tr('Please enter a valid City.|Debe ingresar un CityID válido.'))
      return
    }
    if (clientProviderForm.PersonType === PersonType.COMPANY && !/^\d{7,12}$/.test(clientProviderForm.RegistryNumber || '')) {
      Notify.failure(tr('For companies, Registry Number must be 7–12 digits.|Para empresa, el RegistryNumber debe tener entre 7 y 12 dígitos.'))
      return
    }

    Loading.standard(tr(`Saving ${singularLabel}...|Guardando ${singularLabel}...`))
    console.log('[ClientesProveedoresView] saveClientProvider payload', {
      clientProviderType,
      clientProviderFormSnapshot: $state.snapshot(clientProviderForm),
    })

    try {
      const savedClientProviders = await postClientProviders([clientProviderForm])
      const savedClientProvider = (savedClientProviders?.[0] || clientProviderForm) as IClientProvider

      console.log('[ClientesProveedoresView] saveClientProvider result', {
        clientProviderType,
        savedClientProvider,
      })

      if (!clientProviderService) {
        throw new Error(tr(`Could not initialize service for ${singularLabel}.|No se pudo inicializar el servicio de ${singularLabel}.`))
      }

      const existingClientProviderIndex = clientProviderService.records.findIndex(
        (existingClientProvider) => existingClientProvider.ID === savedClientProvider.ID,
      )

      if (existingClientProviderIndex >= 0) {
        clientProviderService.records = clientProviderService.records.map((existingClientProvider, recordIndex) => {
          if (recordIndex === existingClientProviderIndex) {
            return savedClientProvider
          }
          return existingClientProvider
        })
      } else {
        clientProviderService.records = [savedClientProvider, ...clientProviderService.records]
      }

      clientProviderService.recordsMap.set(savedClientProvider.ID, savedClientProvider)

      Core.hideSideLayer()
      resetEntityForm()
      Notify.success(tr(`${tr(layerTitleSingular)} saved successfully.|${tr(layerTitleSingular)} guardado correctamente.`))
    } catch (saveError) {
      console.warn('[ClientesProveedoresView] saveClientProvider error', {
        clientProviderType,
        saveError,
      })
      Notify.failure(String(saveError))
    } finally {
      Loading.remove()
    }
  }

  const columns: ITableColumn<IClientProvider>[] = [
    {
      header: 'ID',
      headerCss: 'w-64',
      css: 'text-center text-purple-600',
      getValue: (clientProviderRecord) => clientProviderRecord.ID,
    },
    {
      header: 'Name|Nombre',
      getValue: (clientProviderRecord) => clientProviderRecord.Name,
    },
    {
      header: 'Person Type|Tipo Persona',
      headerCss: 'w-144',
      css: 'text-center',
      getValue: (clientProviderRecord) => {
        if (clientProviderRecord.PersonType === PersonType.COMPANY) {
          return tr('Company|Empresa')
        }
        return tr('Person|Persona')
      },
    },
    {
      header: 'Registry / RUC|RUC / Registro',
      getValue: (clientProviderRecord) => clientProviderRecord.RegistryNumber || '-',
    },
    {
      header: 'Email',
      getValue: (clientProviderRecord) => clientProviderRecord.Email || '-',
    },
    {
      header: 'Location|Ubicación',
      headerCss: 'w-144',
      css: 'text-center',
      getValue: (clientProviderRecord) => {
        // Client/provider records still persist CityID as string, so map lookup normalizes to numeric ubigeo.
        const selectedCity = paisCiudadesService.ciudadesMap.get(Number(clientProviderRecord.CityID || 0))
        if (!selectedCity) {
          return clientProviderRecord.CityID || '-'
        }
        const provinceName = selectedCity.Provincia?.Name || '-'
        return `${provinceName} | ${selectedCity.Name}`
      },
    },
    {
      header: 'Updated|Actualizado',
      headerCss: 'w-160',
      css: 'whitespace-nowrap',
      getValue: (clientProviderRecord) => formatTime(clientProviderRecord.upd, 'Y-m-d h:n') as string,
    },
  ]
</script>

<Page title={pageTitle}>
  <Layer type="content">
    <div class="w-full">
      <div class="mb-6 flex items-center justify-between" aria-label="Records list toolbar with search filter and create button">
        <FilterInput bind:value={filterText} placeholder="Search by name, email or registry|Buscar por nombre, email o registro"
          css="mr-16 w-320 max-w-full" />
        <div class="flex items-center">
          <div class="h6 ff-bold pr-8 text-slate-500">
            {filteredClientProviders.length} <T text="records|registros" />
          </div>
          <Button color="green" icon="icon-plus" name="New|Nuevo" hideNameOnMobile
            label={`Opens the side layer to create a new ${layerTitleSingular.toLowerCase()}.`}
            onClick={openCreateClientProviderLayer} />
        </div>
      </div>

      <VTable
        css="w-full"
        maxHeight="calc(80vh - 15rem)"
        columns={columns}
        data={filteredClientProviders}
        selected={clientProviderForm?.ID}
        isSelected={(clientProviderRecord, selectedClientProviderID) => clientProviderRecord.ID === selectedClientProviderID}
        onRowClick={(selectedClientProvider) => {
          openEditClientProviderLayer(selectedClientProvider)
        }}
      />
    </div>
  </Layer>

  <Layer
    id={11}
    type="side"
    sideLayerSize={760}
    title={clientProviderForm.ID ? tr(`Edit ${tr(layerTitleSingular)}|Editar ${tr(layerTitleSingular)}`) : tr(`New ${tr(layerTitleSingular)}|Nuevo ${tr(layerTitleSingular)}`)}
    titleCss="h2 mb-6"
    css="px-12 py-10"
    contentCss="px-0"
    onSave={saveClientProvider}
    onClose={() => {
      resetEntityForm()
    }}
  >
    <div class="mt-8 grid grid-cols-24 gap-10" aria-label="Client or supplier detail form">
      <Input
        label="Name|Nombre"
        saveOn={clientProviderForm}
        save="Name"
        css="col-span-24"
        required={true}
      />
      <SearchSelect
        label="Person Type|Tipo Persona"
        saveOn={clientProviderForm}
        save="PersonType"
        options={personTypeOptions}
        keyId="ID"
        keyName="Name"
        css="col-span-24 md:col-span-12"
      />
      <Input
        label="Email"
        saveOn={clientProviderForm}
        save="Email"
        type="email"
        css="col-span-24 md:col-span-12"
        required={true}
      />
      <Input
        label="Registry Number (RUC)"
        saveOn={clientProviderForm}
        save="RegistryNumber"
        css="col-span-24 md:col-span-12"
        required={clientProviderForm.PersonType === PersonType.COMPANY}
      />
      <SearchSelect
        label="Department | Province | District|Departamento | Provincia | Distrito"
        css="col-span-24"
        options={paisCiudadesService.distritos}
        keyId="ID"
        keyName="_nombre"
        selected={clientProviderForm.CityID || ''}
        required={true}
        onChange={(selectedCity) => {
          // Force the backend contract expected today while the UI keeps a human-readable district picker.
          clientProviderForm.CityID = String(selectedCity.ID || '')
          clientProviderForm.CountryID = 604
        }}
      />
    </div>
  </Layer>
</Page>
