<script lang="ts">
import SearchSelect from '$components/form/SearchSelect.svelte';
import { useCiudadesAPI, type ICiudad } from '$services/services/cities.svelte';

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

  const ciudades = useCiudadesAPI()
  let provincias = $state([] as ICiudad[])
  let distritos = $state([] as ICiudad[])

$effect(() => {
  if(!saveOn || !save || !ciudades.ciudadesMap){ return }

  const ciudad = ciudades.ciudadesMap.get(Number(saveOn[save] || 0))

  if(ciudad){
    // Hierarchy defines the selected level now that ubigeos are stored as numbers.
    if(ciudad.Hierarchy === 1){
      form.departamentoID = ciudad.ID
    } else if(ciudad.Hierarchy === 2){
      form.departamentoID = ciudad.ParentID
      form.provinciaID = ciudad.ID
    } else {
      const provincia = ciudades.ciudadesMap.get(ciudad.ParentID)
      if(provincia){
        form.departamentoID = provincia.ParentID
        form.provinciaID = ciudad.ParentID
      }
    }
  }

  if(form.departamentoID){
    provincias = ciudades.ciudadHijosMap.get(form.departamentoID) || []
  } else {
    provincias = []
  }

  if(form.provinciaID){
    distritos = ciudades.ciudadHijosMap.get(form.provinciaID) || []
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
  options={ciudades.departamentos}
  onChange={e => {
    console.log("departamento::", e)
    form.distritoID = 0
    form.provinciaID = 0
    provincias = ciudades.ciudadHijosMap.get(e?.ID) || []
    distritos = []
    doSave()
  }}
/>
<SearchSelect saveOn={form} save="provinciaID" css={css}
  label="Provincia" keyId="ID" keyName="Name" required={true}
  options={provincias}
  onChange={e => {
    form.distritoID = 0
    distritos = ciudades.ciudadHijosMap.get(e?.ID) || []
    doSave()
  }}
/>
<SearchSelect saveOn={form} save="distritoID" css={css}
  label="Distrito" keyId="ID" keyName="Name" required={true}
  options={distritos}
  onChange={() => { doSave() }}
/>
