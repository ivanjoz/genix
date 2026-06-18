<script lang="ts">
  import { untrack } from 'svelte'
  import Input from '$components/form/Input.svelte'
  import LayerStatic from '$components/layers/LayerStatic.svelte'
  import FilterInput from '$components/form/FilterInput.svelte'
  import Button from '$components/buttons/Button.svelte'
  import TableTree, { type TableTreeNode } from '$components/vTable/TableTree.svelte'
  import type { ITableColumn } from '$components/vTable/types'
  import Page from '$domain/Page.svelte'
  import { GetHandler, POST } from '$libs/http.svelte'
  import { formatN, Loading } from '$libs/helpers'
  import { tr } from '$core/store.svelte'
  import T from '$components/misc/T.svelte'
  import {
    PaisCiudadesService,
    type ICityLocation,
  } from '$routes/negocio/sedes-almacenes/sedes-almacenes.svelte'

  type DeliveryCostField = 'Fijo' | 'PorKg'

  type DeliveryCost = {
    CityID: number
    FlatCost: number
    CostPerKg: number
    hasUpdated?: boolean
    upd?: number
  }

  class ShippingCostsService extends GetHandler<any> {
    route = 'shipping-costs'
    keyID = 'CityID'
    useCache = { min: 5, ver: 1 }

    records: DeliveryCost[] = $state([])
    recordsMap: Map<number, DeliveryCost> = $state(new Map())

    handler(result: DeliveryCost[]): void {
      // Backend rows are keyed by CityID; addSavedRecords expects ID, so this service owns its maps directly.
      this.records = (result || []).map((shippingCost) => ({ ...shippingCost, hasUpdated: false }))
      this.recordsMap = new Map(this.records.map((shippingCost) => [shippingCost.CityID, shippingCost]))
      console.debug('[delivery-costs] fetched costs', { count: this.records.length })
    }

    constructor(init?: boolean) {
      super()
      if (init) { this.fetch() }
    }
  }

  const paisCiudadesService = new PaisCiudadesService(true)
  const shippingCostsService = new ShippingCostsService(true)

  let selectedDepartamento = $state<ICityLocation | null>(null)
  let selectedProvincia = $state<ICityLocation | null>(null)
  let departamentoCostForm = $state<DeliveryCost>(makeEmptyDeliveryCost(0))
  let deliveryCostByCiudadID = $state<Record<string, DeliveryCost>>({})
  let filterText = $state('')
  let lastAutoSelectedFilterText = ''

  const normalizedFilterText = $derived(normalizeSearchText(filterText))

  const distritosByProvinciaID = $derived.by(() => {
    const groupedDistritos = new Map<number, ICityLocation[]>()

    for (const distrito of paisCiudadesService.distritos) {
      // District rows are attached to their province through PadreID.
      const currentList = groupedDistritos.get(distrito.ParentID) || []
      currentList.push(distrito)
      groupedDistritos.set(distrito.ParentID, currentList)
    }

    return groupedDistritos
  })

  const provinciasByDepartamentoID = $derived.by(() => {
    const groupedProvincias = new Map<number, ICityLocation[]>()

    for (const provincia of paisCiudadesService.provincias) {
      // Province summaries are grouped once so every departamento card can read the same source.
      const currentList = groupedProvincias.get(provincia.ParentID) || []
      currentList.push(provincia)
      groupedProvincias.set(provincia.ParentID, currentList)
    }

    return groupedProvincias
  })

  const ciudadColumns: ITableColumn<ICityLocation>[] = [
    {
      id: 'name',
      header: 'Name|Nombre',
      width: 'minmax(180px, 1fr)',
      css: 'text-[14px] leading-[1.1] pl-10 pr-24',
      getValue: (ciudad) => ciudad.Name,
    },
    {
      id: 'fixed-cost',
      header: 'Fixed|Fijo',
      width: '110px',
      align: 'right',
      cellInputType: 'number',
      css: 'text-[14px] font-mono',
      getValue: (ciudad) => getDeliveryCost(ciudad.ID, 'Fijo'),
      formatInputValue: value => value ? formatN(value as number,2) as string : "-",
      onCellEdit: (ciudad, value) => {
        // Fixed costs are kept locally for now; backend persistence can consume this map later.
        updateCiudadCost(ciudad, 'Fijo', value)
      },
    },
    {
      id: 'weight-cost',
      header: 'Per Kg|Por Kg',
      width: '110px',
      align: 'right',
      cellInputType: 'number',
      css: 'text-[14px] font-mono',
      getValue: (ciudad) => getDeliveryCost(ciudad.ID, 'PorKg'),
      formatInputValue: value => value ? formatN(value as number,2) as string : "-",
      onCellEdit: (ciudad, value) => {
        // Weight costs share the same city record so fixed and per-kg prices save together.
        updateCiudadCost(ciudad, 'PorKg', value)
      },
    },
  ]

  const provinciaTree = $derived.by<TableTreeNode<ICityLocation>[]>(() => {
    const departamento = selectedDepartamento
    if (!departamento) { return [] }

    return (provinciasByDepartamentoID.get(departamento.ID) || [])
      .map((provincia) => makeProvinciaTreeNode(provincia))
      .filter((node) => !normalizedFilterText || cityMatchesFilter(node.record) || node.children.length > 0)
  })

  const filteredDepartamentos = $derived.by(() => {
    if (!normalizedFilterText) { return paisCiudadesService.departamentos }

    return paisCiudadesService.departamentos.filter((departamento) => {
      // Departamento cards stay visible when the filter matches the departamento or its child places.
      if (cityMatchesFilter(departamento)) { return true }
      return (provinciasByDepartamentoID.get(departamento.ID) || []).some((provincia) => {
        if (cityMatchesFilter(provincia)) { return true }
        return (distritosByProvinciaID.get(provincia.ID) || []).some(cityMatchesFilter)
      })
    })
  })

  $effect(() => {
    const departamentoID = selectedDepartamento?.ID
    const departamentoCost = departamentoID ? getDeliveryCostRecord(departamentoID) : makeEmptyDeliveryCost(0)

    untrack(() => {
      // Keep the header input aligned with the selected departamento without feeding the effect back into itself.
      departamentoCostForm = { ...departamentoCost }
    })
  })

  $effect(() => {
    const readyTick = shippingCostsService.isReady

    untrack(() => {
      // Hydrate local editable state from the delta cache without overwriting unsaved edits.
      const nextCostByCityID = { ...deliveryCostByCiudadID }
      for (const shippingCost of shippingCostsService.records) {
        const currentCost = nextCostByCityID[shippingCost.CityID]
        if (currentCost?.hasUpdated) { continue }
        nextCostByCityID[shippingCost.CityID] = { ...shippingCost, hasUpdated: false }
      }
      deliveryCostByCiudadID = nextCostByCityID
      console.debug('[delivery-costs] local costs hydrated', { readyTick, count: shippingCostsService.records.length })
    })
  })

  $effect(() => {
    if (!normalizedFilterText || normalizedFilterText === lastAutoSelectedFilterText) { return }
    lastAutoSelectedFilterText = normalizedFilterText

    if (filteredDepartamentos.length > 0 && filteredDepartamentos.length <= 6) {
      untrack(() => {
        // Narrow filters should immediately open the first matching departamento for faster editing.
        selectDepartamento(filteredDepartamentos[0])
      })
    }
  })

  function selectDepartamento(departamento: ICityLocation) {
    // Departamento click controls the right-side province/district table.
    selectedDepartamento = departamento
    selectedProvincia = null
    console.debug('[delivery-costs] departamento selected', {
      id: departamento.ID,
      nombre: departamento.Name,
    })
  }

  function toggleProvincia(node: TableTreeNode<ICityLocation>) {
    // The clicked province opens its district rows as the second table level.
    selectedProvincia = node.isOpen ? node.record : null
    console.debug('[delivery-costs] provincia selected', {
      id: node.record.ID,
      nombre: node.record.Name,
      isOpen: node.isOpen,
      distritos: node.children.length,
    })
  }

  function updateDepartamentoCost() {
    if (!selectedDepartamento) { return }

    // Departamento cost is stored with the same city-ID map used by provincia and distrito rows.
    const cost = normalizeDeliveryCostRecord({
      ...departamentoCostForm,
      CityID: selectedDepartamento.ID,
      hasUpdated: true,
    })
    deliveryCostByCiudadID = { ...deliveryCostByCiudadID, [selectedDepartamento.ID]: cost }
    console.debug('[delivery-costs] departamento cost edited', {
      id: selectedDepartamento.ID,
      nombre: selectedDepartamento.Name,
      cost,
    })
  }

  function updateCiudadCost(ciudad: ICityLocation, field: DeliveryCostField, value: string | number) {
    // Editing one column preserves the other cost field for the same city row.
    const nextCost = Number(value || 0)
    const currentCost = getDeliveryCostRecord(ciudad.ID)
    const backendField = getBackendCostField(field)
    const cost = {
      ...currentCost,
      CityID: ciudad.ID,
      [backendField]: isNaN(nextCost) ? 0 : nextCost,
      hasUpdated: true,
    }
    deliveryCostByCiudadID = { ...deliveryCostByCiudadID, [ciudad.ID]: cost }
    console.debug('[delivery-costs] cost edited', { id: ciudad.ID, nombre: ciudad.Name, field, cost })
  }

  function normalizeSearchText(value: string) {
    // Accent-insensitive filtering keeps Spanish place names searchable with plain keyboard input.
    return value
      .trim()
      .toLowerCase()
      .normalize('NFD')
      .replace(/\p{Diacritic}/gu, '')
  }

  function cityMatchesFilter(ciudad: ICityLocation) {
    // Every filter target uses the same normalized comparison to avoid inconsistent results.
    return normalizeSearchText(ciudad.Name).includes(normalizedFilterText)
  }

  function makeProvinciaTreeNode(provincia: ICityLocation): TableTreeNode<ICityLocation> {
    const provinciaMatches = cityMatchesFilter(provincia)
    const allDistritos = distritosByProvinciaID.get(provincia.ID) || []
    const visibleDistritos = !normalizedFilterText || provinciaMatches
      ? allDistritos
      : allDistritos.filter(cityMatchesFilter)

    return {
      id: provincia.ID,
      record: provincia,
      children: visibleDistritos,
      isOpen: selectedProvincia?.ID === provincia.ID,
    }
  }

  async function saveDeliveryCosts() {
    const changedCosts = Object.values(deliveryCostByCiudadID)
      .filter((shippingCost) => shippingCost.hasUpdated)
      .map(normalizeDeliveryCostRecord)

    console.debug('[delivery-costs] save requested', { changedCount: changedCosts.length, changedCosts })
    if (changedCosts.length === 0) { return }

    Loading.standard(tr("Saving records...|Guardando Registros..."))
    try {
      const savedCosts = await POST({
        route: 'shipping-costs',
        data: changedCosts,
        refreshRoutes: ['shipping-costs'],
      }) as DeliveryCost[]

      for (const savedCost of savedCosts || []) {
        deliveryCostByCiudadID = {
          ...deliveryCostByCiudadID,
          [savedCost.CityID]: { ...savedCost, hasUpdated: false },
        }
      }
      shippingCostsService.fetchOnline()
    } finally {
      Loading.remove()
    }
  }

  function normalizeDeliveryCostRecord(cost: DeliveryCost) {
    // Numeric normalization avoids storing NaN when a number input is cleared or receives invalid text.
    return {
      ...cost,
      CityID: Number(cost.CityID || 0),
      FlatCost: Number.isNaN(Number(cost.FlatCost || 0)) ? 0 : Number(cost.FlatCost || 0),
      CostPerKg: Number.isNaN(Number(cost.CostPerKg || 0)) ? 0 : Number(cost.CostPerKg || 0),
    }
  }

  function getDeliveryCostRecord(ciudadID: number) {
    // Missing city costs behave as an empty two-column price record.
    return deliveryCostByCiudadID[ciudadID] || makeEmptyDeliveryCost(ciudadID)
  }

  function getDeliveryCost(ciudadID: number, field: DeliveryCostField) {
    // Empty cost cells are treated as unset and excluded from min/max summaries.
    const cost = Number(getDeliveryCostRecord(ciudadID)[getBackendCostField(field)] || 0)
    return isNaN(cost) ? 0 : cost
  }

  function getBackendCostField(field: DeliveryCostField) {
    return field === 'Fijo' ? 'FlatCost' : 'CostPerKg'
  }

  function makeEmptyDeliveryCost(cityID: number): DeliveryCost {
    return { CityID: cityID, FlatCost: 0, CostPerKg: 0, hasUpdated: false }
  }

  function formatDeliveryCost(cost: number) {
    // Cards use the same two-decimal presentation as the editable table cells.
    return cost > 0 ? formatN(cost, 2) as string : '-'
  }

  function formatCostRange(ciudades: ICityLocation[], field: DeliveryCostField) {
    // Range summaries only compare configured costs, so empty cities do not collapse the minimum to zero.
    const configuredCosts = ciudades
      .map((ciudad) => getDeliveryCost(ciudad.ID, field))
      .filter((cost) => cost > 0)

    if (!configuredCosts.length) { return '-' }

    const lowestCost = Math.min(...configuredCosts)
    const highestCost = Math.max(...configuredCosts)
    return `${formatDeliveryCost(lowestCost)} - ${formatDeliveryCost(highestCost)}`
  }

  function getProvinciaCostRange(departamento: ICityLocation, field: DeliveryCostField) {
    // Provincia range summarizes direct province-level shipping costs for this departamento.
    return formatCostRange(provinciasByDepartamentoID.get(departamento.ID) || [], field)
  }

  function getDistritoCostRange(departamento: ICityLocation, field: DeliveryCostField) {
    // Distrito range summarizes every district under the departamento's provinces.
    const departamentoDistritos = (provinciasByDepartamentoID.get(departamento.ID) || [])
      .flatMap((provincia) => distritosByProvinciaID.get(provincia.ID) || [])
    return formatCostRange(departamentoDistritos, field)
  }
</script>

<Page title="Delivery Costs|Costos de Delivery">
  <div class="flex h-full gap-20">
    <div class="flex-1 min-w-0 h-[calc(100vh-var(--header-height)-20px)] bg-gray-100 rounded-md border border-gray-200 p-8 flex flex-col min-h-0">
      <div class="mb-8 flex items-center gap-8 w-full justify-between shrink-0" aria-label="Shipping costs toolbar with filter and save">
        <FilterInput
          bind:value={filterText}
          placeholder="Filter|Filtrar"
          icon="icon-[fa--search]"
          css="w-240"
        />
        <Button color="purple" icon="icon-[fa--floppy-o]" name="Save|Guardar"
          label="Saves all edited shipping cost records to the server."
          onClick={saveDeliveryCosts} />
      </div>
      <div class="flex-1 min-h-0 overflow-auto">
        <div class="grid grid-cols-2 gap-8">
          {#each filteredDepartamentos as departamento (departamento.ID)}
            <button
              type="button"
              class="text-left min-h-72 px-10 py-8 rounded-md border bg-white hover:border-slate-500 {selectedDepartamento?.ID === departamento.ID ? 'border-violet-400 bg-violet-50 shadow-[inset_0_0_3px_#9987c8]' : 'border-gray-200 text-gray-700'}"
              onclick={() => selectDepartamento(departamento)}
            >
              <div class="flex items-center gap-8">
                <div class="ff-semibold leading-[1.1] truncate">{departamento.Name}</div>
                <div class="rounded bg-blue-50 px-8 py-3 text-xs leading-[1.0] text-blue-700 font-mono">
                  {formatDeliveryCost(getDeliveryCost(departamento.ID, 'Fijo'))}
                </div>
              </div>
              <div class="mt-8 grid grid-cols-2 gap-12 leading-[1.1]">
                <div>
                  <div class="uppercase text-[12px] text-gray-500"><T text="Province|Provincia" /></div>
                  <div class="text-[13px] mt-2 font-mono text-gray-800">{getProvinciaCostRange(departamento, 'Fijo')}</div>
                </div>
                <div>
                  <div class="uppercase text-[12px] text-gray-500"><T text="District|Distrito" /></div>
                  <div class="text-[13px] mt-2 font-mono text-gray-800">{getDistritoCostRange(departamento, 'Fijo')}</div>
                </div>
              </div>
            </button>
          {:else}
            <div class="col-span-2 rounded-md border border-gray-200 bg-white p-12 text-[14px] text-gray-400">
              <T text="No departments|Sin departamentos" />
            </div>
          {/each}
        </div>
      </div>
    </div>

    <LayerStatic
      css="w-[50%] min-w-350 bg-white border-l border-gray-200 flex flex-col h-[calc(100vh-var(--header-height))] shadow-lg md:-m-10"
      mobileLayerTitle="Zone Details|Detalle de Zona"
      useMobileLayerVertical={120}
    >
      <div class="px-12 py-10 border-b border-gray-100 bg-gray-50/50">
        <div class="hidden md:block">
          <div class="text-[15px] font-bold text-gray-800">
            <T text="Delivery Costs|Costos de Delivery" />
          </div>
        </div>
        <div class="mt-6 grid grid-cols-[minmax(160px,1fr)_110px_110px] gap-8" aria-label="Departamento delivery cost form with flat and per-kg rates">
          <div class="bg-gray-100 rounded-md p-8">
            <div class="text-[10px] uppercase text-gray-500"><T text="Department|Departamento" /></div>
            <div class="text-[15px] text-gray-800 truncate">{selectedDepartamento?.Name || tr('Select|Seleccione')}</div>
          </div>
          <Input
            bind:saveOn={departamentoCostForm}
            save="FlatCost"
            label="Fixed|Fijo"
            type="number"
            disabled={!selectedDepartamento}
            css="bg-blue-50 rounded-md"
            inputCss="text-[15px] text-blue-700 font-bold"
            onChange={updateDepartamentoCost}
          />
          <Input
            bind:saveOn={departamentoCostForm}
            save="CostPerKg"
            label="Per Kg|Por Kg"
            type="number"
            disabled={!selectedDepartamento}
            css="bg-cyan-50 rounded-md"
            inputCss="text-[15px] text-cyan-700 font-bold"
            onChange={updateDepartamentoCost}
          />
        </div>
      </div>

      <div class="px-12 py-10 flex-1 min-h-0 overflow-auto">
        {#if selectedDepartamento}
          <TableTree
            columns={ciudadColumns}
            data={provinciaTree}
            selectedId={selectedProvincia?.ID}
            getChildId={(distrito) => distrito.ID}
            onNodeClick={toggleProvincia}
            onChildClick={(distrito) => console.debug('[delivery-costs] distrito click', { id: distrito.ID, nombre: distrito.Name })}
            emptyMessage="No provinces|Sin provincias"
            css=""
            rowCss="text-[14px]"
          />
        {:else}
          <div class="h-full flex flex-col items-center justify-center text-gray-300 gap-8">
            <i class="icon-[fa--map-marker] text-4xl"></i>
            <span class="text-[14px]"><T text="Select a department|Seleccione un departamento" /></span>
          </div>
        {/if}
      </div>
    </LayerStatic>
  </div>
</Page>
