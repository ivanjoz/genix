<script lang="ts">
import SearchSelect from '$components/form/SearchSelect.svelte';
import { useCitiesAPI, type ICity } from '$services/services/cities.svelte';

  export interface ICiudades {
    css: string
    saveOn: any
    save: string
    onChange?: () => void
  }

  const {
    css, saveOn, save, onChange
  }: ICiudades = $props();

  // svelte-ignore state_referenced_locally
  let form = $state({
    departamentoID: Number(saveOn?.departamentoID || 0),
    provinciaID: Number(saveOn?.provinciaID || 0),
    distritoID: Number(saveOn?.distritoID || 0)
  })

  const cities = useCitiesAPI()
  let provincias = $state([] as ICity[])
  let distritos = $state([] as ICity[])

$effect(() => {
  if(!saveOn || !save || !cities.citiesMap){ return }

  const ciudad = cities.citiesMap.get(Number(saveOn[save] || 0))

  if(ciudad){
    // Hierarchy defines the selected level now that ubigeos are stored as numbers.
    if(ciudad.Hierarchy === 1){
      form.departamentoID = ciudad.ID
    } else if(ciudad.Hierarchy === 2){
      form.departamentoID = ciudad.ParentID
      form.provinciaID = ciudad.ID
    } else {
      const provincia = cities.citiesMap.get(ciudad.ParentID)
      if(provincia){
        form.departamentoID = provincia.ParentID
        form.provinciaID = ciudad.ParentID
      }
    }
  }

  if(form.departamentoID){
    provincias = cities.cityChildrenMap.get(form.departamentoID) || []
  } else {
    provincias = []
  }

  if(form.provinciaID){
    distritos = cities.cityChildrenMap.get(form.provinciaID) || []
  } else {
    distritos = []
  }
})

  const doSave = () => {
    if(!saveOn || !save){ return }
    const ciudadID = form.distritoID || form.provinciaID || form.departamentoID || 0
    saveOn[save] = ciudadID as never
    if(onChange){ onChange() }
  }

</script>

<SearchSelect saveOn={form} save="departamentoID" css={css}
  label={"Departamento"} keyId="ID" keyName="Name" required={true}
  options={cities.departments}
  onChange={e => {
    console.log("departamento::", e)
    form.distritoID = 0
    form.provinciaID = 0
    provincias = cities.cityChildrenMap.get(e?.ID) || []
    distritos = []
    doSave()
  }}
/>
<SearchSelect saveOn={form} save="provinciaID" css={css}
  label="Provincia" keyId="ID" keyName="Name" required={true}
  options={provincias}
  onChange={e => {
    form.distritoID = 0
    distritos = cities.cityChildrenMap.get(e?.ID) || []
    doSave()
  }}
/>
<SearchSelect saveOn={form} save="distritoID" css={css}
  label="Distrito" keyId="ID" keyName="Name" required={true}
  options={distritos}
  onChange={() => { doSave() }}
/>
