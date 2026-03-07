<script lang="ts">
import SearchSelect from '$components/SearchSelect.svelte';
import { useCiudadesAPI, type ICiudad } from '$services/services/ciudades.svelte';

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
    departamentoID: (saveOn?.departamentoID || "") as string,
    provinciaID: (saveOn?.provinciaID || "") as string,
    distritoID: (saveOn?.distritoID || "") as string
  })

  const ciudades = useCiudadesAPI()
  let provincias = $state([] as ICiudad[])
  let distritos = $state([] as ICiudad[])

$effect(() => {
  if(!saveOn || !save || !ciudades.ciudadesMap){ return }

  const ciudad = ciudades.ciudadesMap.get(saveOn[save] as string)

  if(ciudad){
    if(ciudad.ID.length === 2){
      form.departamentoID = ciudad.ID
    } else if(ciudad.ID.length === 4){
      form.departamentoID = ciudad.PadreID
      form.provinciaID = ciudad.ID
    } else {
      const provincia = ciudades.ciudadesMap.get(ciudad.PadreID)
      if(provincia){
        form.departamentoID = provincia.PadreID
        form.provinciaID = ciudad.PadreID
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
    const ciudadID = form.distritoID || form.provinciaID || form.departamentoID || ""
    saveOn[save] = ciudadID as never
    if(onChange){ onChange() }
  }

</script>

<SearchSelect saveOn={form} save="departamentoID" css={css}
  label={"Departamento"} keyId="ID" keyName="Nombre" required={true}
  options={ciudades.departamentos}
  onChange={e => {
    console.log("departamento::", e)
    form.distritoID = ""
    form.provinciaID = ""
    provincias = ciudades.ciudadHijosMap.get(e?.ID) || []
    distritos = []
    doSave()
  }}
/>
<SearchSelect saveOn={form} save="provinciaID" css={css}
  label="Provincia" keyId="ID" keyName="Nombre" required={true}
  options={provincias}
  onChange={e => {
    form.distritoID = ""
    distritos = ciudades.ciudadHijosMap.get(e?.ID) || []
    doSave()
  }}
/>
<SearchSelect saveOn={form} save="distritoID" css={css}
  label="Distrito" keyId="ID" keyName="Nombre" required={true}
  options={distritos}
  onChange={() => { doSave() }}
/>
