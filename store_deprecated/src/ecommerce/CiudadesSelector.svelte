<script lang="ts">
  import SearchSelect from "$components/SearchSelect.svelte";
  import { useCiudadesAPI, type ICiudad } from "../services/ciudades.svelte";
  
  export interface ICiudades {
    css: string
    saveOn: any
    save: string
    onChange?: () => void
  }
  
  const {
    css, saveOn, save, onChange
  }: ICiudades = $props();

  let form = $state({ 
    departamentoID: saveOn.departamentoID || "", 
    provinciaID: saveOn.provinciaID || "", 
    distritoID: saveOn.distritoID || "" 
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
        form.departamentoID = provincia.PadreID
        form.provinciaID = ciudad.PadreID
        form.distritoID = ciudad.ID
      }
      if(form.departamentoID){
        provincias = ciudades.ciudadHijosMap.get(form.departamentoID)
      } else {
        provincias = []
      }
      if(form.provinciaID){
        distritos = ciudades.ciudadHijosMap.get(form.provinciaID)
      } else {
        distritos = []
      }
    } else {
      provincias = []
      distritos = []
      form.provinciaID = ""
      form.distritoID = ""
      form.provinciaID = ""
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
  label={"Departamento "+ciudades.departamentos.length} keys="ID.Nombre" required={true}
  options={ciudades.departamentos}
  onChange={e => {
    console.log("departamento::", e)
    form.distritoID = null
    form.provinciaID = null
    provincias = ciudades.ciudadHijosMap.get(e?.ID) || []
    distritos = []
    doSave()
  }}
/>
<SearchSelect saveOn={form} save="provinciaID" css={css}
  label="Provincia" keys="ID.Nombre" required={true}
  options={provincias}
  onChange={e => {
    form.distritoID = null
    distritos = ciudades.ciudadHijosMap.get(e?.ID) || []
    doSave()
  }}
/>
<SearchSelect saveOn={form} save="distritoID" css={css}
  label="Distrito" keys="ID.Nombre" required={true}
  options={distritos}
  onChange={() => { doSave() }}
/>
