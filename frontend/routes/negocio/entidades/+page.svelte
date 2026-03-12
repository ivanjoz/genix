<script lang="ts">
  import Layer from '$components/Layer.svelte'
  import Input from '$components/Input.svelte'
  import OptionsStrip from '$components/OptionsStrip.svelte'
  import SearchSelect from '$components/SearchSelect.svelte'
  import VTable from '$components/vTable/VTable.svelte'
  import type { ITableColumn } from '$components/vTable/types'
  import { Core } from '$core/store.svelte'
  import Page from '$domain/Page.svelte'
  import { Loading, Notify, formatTime, throttle } from '$libs/helpers'
  import { PaisCiudadesService } from '../sedes-almacenes/sedes-almacenes.svelte'
  import { EntityType, EntitiesService, PersonType, postEntities, type IEntity } from './entidades.svelte'

  const options = [
    [EntityType.CLIENT, 'Clientes'],
    [EntityType.PROVIDER, 'Proveedores'],
  ]

  const clientsService = new EntitiesService(EntityType.CLIENT)
  const providersService = new EntitiesService(EntityType.PROVIDER)
  const paisCiudadesService = new PaisCiudadesService()

  let selectedEntityType = $state<number>(EntityType.CLIENT)
  let filterText = $state('')
  let entityForm = $state<IEntity>({
    ID: 0,
    Type: EntityType.CLIENT,
    Name: '',
    RegistryNumber: '',
    PersonType: PersonType.PERSON,
    Email: '',
    CountryID: 604,
    CityID: '',
    ss: 1,
    upd: 0,
  })

  const selectedEntities = $derived.by(() => {
    if (selectedEntityType === EntityType.PROVIDER) {
      return providersService.records
    }
    return clientsService.records
  })

  const filteredEntities = $derived.by(() => {
    const normalizedFilter = filterText.trim().toLowerCase()
    if (!normalizedFilter) {
      return selectedEntities
    }

    return selectedEntities.filter((entityRecord) => {
      return [
        entityRecord.Name,
        entityRecord.RegistryNumber,
        entityRecord.Email,
        String(entityRecord.ID),
      ].some((fieldValue) => (fieldValue || '').toLowerCase().includes(normalizedFilter))
    })
  })

  const personTypeOptions = [
    { ID: PersonType.PERSON, Name: 'Persona' },
    { ID: PersonType.COMPANY, Name: 'Empresa' },
  ]

  const resetEntityForm = () => {
    entityForm = {
      ID: 0,
      Type: selectedEntityType,
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

  const openCreateEntityLayer = () => {
    resetEntityForm()
    Core.openSideLayer(11)
  }

  const openEditEntityLayer = (selectedEntityRecord: IEntity) => {
    // Clone the selected record so form edits do not mutate table state before save.
    entityForm = {
      ...selectedEntityRecord,
      Type: selectedEntityType,
    }
    Core.openSideLayer(11)
  }

  const saveEntity = async () => {
    entityForm.Name = (entityForm.Name || '').trim()
    entityForm.Email = (entityForm.Email || '').trim().toLowerCase()
    entityForm.RegistryNumber = (entityForm.RegistryNumber || '').trim()
    entityForm.Type = selectedEntityType

    if (!entityForm.Name) {
      Notify.failure('Debe ingresar el nombre de la entidad.')
      return
    }
    if (!entityForm.Email || !entityForm.Email.includes('@')) {
      Notify.failure('Debe ingresar un email válido.')
      return
    }
    if (!entityForm.CountryID || entityForm.CountryID <= 0) {
      Notify.failure('Debe ingresar un CountryID válido.')
      return
    }
    if (!entityForm.CityID || !String(entityForm.CityID).trim()) {
      Notify.failure('Debe ingresar un CityID válido.')
      return
    }
    if (entityForm.PersonType === PersonType.COMPANY && !/^\d{7,12}$/.test(entityForm.RegistryNumber || '')) {
      Notify.failure('Para empresa, el RegistryNumber debe tener entre 7 y 12 dígitos.')
      return
    }

    Loading.standard('Guardando entidad...')
    console.log('saveEntity payload::', $state.snapshot(entityForm))

    try {
      const saveResult = await postEntities([entityForm])
      const savedEntity = (saveResult?.[0] || entityForm) as IEntity
      console.log('saveEntity result::', saveResult)

      const targetService = savedEntity.Type === EntityType.PROVIDER
        ? providersService
        : clientsService
      const existingEntityIndex = targetService.records.findIndex(
        (existingEntityRecord) => existingEntityRecord.ID === savedEntity.ID,
      )

      if (existingEntityIndex >= 0) {
        targetService.records = targetService.records.map((existingEntityRecord, recordIndex) => {
          if (recordIndex === existingEntityIndex) {
            return savedEntity
          }
          return existingEntityRecord
        })
      } else {
        targetService.records = [savedEntity, ...targetService.records]
      }
      targetService.recordsMap.set(savedEntity.ID, savedEntity)

      Core.hideSideLayer()
      resetEntityForm()
      Notify.success('Entidad guardada correctamente.')
    } catch (saveError) {
      console.warn('saveEntity error::', saveError)
      Notify.failure(String(saveError))
    } finally {
      Loading.remove()
    }
  }

  const columns: ITableColumn<IEntity>[] = [
    {
      header: 'ID',
      headerCss: 'w-64',
      cellCss: 'px-6 text-center text-purple-600',
      getValue: (entityRecord) => entityRecord.ID,
    },
    {
      header: 'Nombre',
      cellCss: 'px-6',
      getValue: (entityRecord) => entityRecord.Name,
    },
    {
      header: 'Tipo Persona',
      headerCss: 'w-144',
      cellCss: 'px-6 text-center',
      getValue: (entityRecord) => {
        if (entityRecord.PersonType === PersonType.COMPANY) {
          return 'Empresa'
        }
        return 'Persona'
      },
    },
    {
      header: 'RUC / Registro',
      cellCss: 'px-6',
      getValue: (entityRecord) => entityRecord.RegistryNumber || '-',
    },
    {
      header: 'Email',
      cellCss: 'px-6',
      getValue: (entityRecord) => entityRecord.Email || '-',
    },
    {
      header: 'Ubicación',
      headerCss: 'w-144',
      cellCss: 'px-6 text-center',
      getValue: (entityRecord) => {
        const selectedCity = paisCiudadesService.ciudadesMap.get(entityRecord.CityID || '')
        if (!selectedCity) {
          return entityRecord.CityID || '-'
        }
        const provinceName = selectedCity.Provincia?.Nombre || '-'
        return `${provinceName} | ${selectedCity.Nombre}`
      },
    },
    {
      header: 'Actualizado',
      headerCss: 'w-160',
      cellCss: 'px-6 whitespace-nowrap',
      getValue: (entityRecord) => formatTime(entityRecord.upd, 'Y-m-d h:n') as string,
    },
  ]
</script>

<Page title="Entidades">
  <Layer type="content">
    <div class="w-full">
      <OptionsStrip
        selected={selectedEntityType}
        options={options}
        useMobileGrid={true}
        onSelect={(selectedOption) => {
          selectedEntityType = selectedOption[0] as number
        }}
      />

      <div class="flex items-center justify-between mb-6">
        <div class="i-search mr-16 w-320 max-w-full">
          <div><i class="icon-search"></i></div>
          <input class="w-full" autocomplete="off" type="text" placeholder="Buscar por nombre, email o registro"
            onkeyup={(event) => {
              event.stopPropagation()
              throttle(() => {
                filterText = ((event.target as HTMLInputElement).value || '').toLowerCase().trim()
              }, 150)
            }}
          />
        </div>
        <div class="flex items-center">
          <div class="h6 text-slate-500 ff-bold pr-8">
            {filteredEntities.length} registros
          </div>
          <button class="bx-green"
            aria-label="Crear entidad"
            onclick={(event) => {
              event.stopPropagation()
              openCreateEntityLayer()
            }}
          >
            <i class="icon-plus"></i><span class="hidden md:block">Nuevo</span>
          </button>
        </div>
      </div>

      <VTable
        css="w-full"
        maxHeight="calc(80vh - 15rem)"
        columns={columns}
        data={filteredEntities}
        selected={entityForm?.ID}
        isSelected={(entityRecord, selectedEntityID) => entityRecord.ID === selectedEntityID}
        onRowClick={(selectedEntityRecord) => {
          openEditEntityLayer(selectedEntityRecord)
        }}
      />
    </div>
  </Layer>

  <Layer
    id={11}
    type="side"
    sideLayerSize={760}
    title={entityForm.ID
      ? (selectedEntityType === EntityType.PROVIDER ? 'Editar Proveedor' : 'Editar Cliente')
      : (selectedEntityType === EntityType.PROVIDER ? 'Nuevo Proveedor' : 'Nuevo Cliente')}
    titleCss="h2 mb-6"
    css="px-12 py-10"
    contentCss="px-0"
    onSave={saveEntity}
    onClose={() => {
      resetEntityForm()
    }}
  >
    <div class="grid grid-cols-24 gap-10 mt-8">
      <Input
        label="Nombre"
        saveOn={entityForm}
        save="Name"
        css="col-span-24"
        required={true}
      />
      <SearchSelect
        label="Tipo Persona"
        saveOn={entityForm}
        save="PersonType"
        options={personTypeOptions}
        keyId="ID"
        keyName="Name"
        css="col-span-24 md:col-span-12"
      />
      <Input
        label="Email"
        saveOn={entityForm}
        save="Email"
        type="email"
        css="col-span-24 md:col-span-12"
        required={true}
      />
      <Input
        label="Registry Number (RUC)"
        saveOn={entityForm}
        save="RegistryNumber"
        css="col-span-24 md:col-span-12"
        required={entityForm.PersonType === PersonType.COMPANY}
      />
      <SearchSelect
        label="Departamento | Provincia | Distrito"
        css="col-span-24"
        options={paisCiudadesService.distritos}
        keyId="ID"
        keyName="_nombre"
        selected={entityForm.CityID || ''}
        required={true}
        onChange={(selectedCity) => {
          entityForm.CityID = String(selectedCity.ID || '')
          entityForm.CountryID = 604
        }}
      />
    </div>
  </Layer>
</Page>
