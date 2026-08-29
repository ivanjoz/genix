import { GetHandler } from '$libs/ui-runtime.svelte';

export interface ICityLocation {
  CountryID: number
	ID: number
  Name: string
  ParentID: number
  Hierarchy: number
  Departamento?: ICityLocation
  Provincia?: ICityLocation
  upd: number
  _nombre?: string
}

export interface CountryCityResult {
  ciudades: ICityLocation[]
  distritos: ICityLocation[]
  ciudadesMap: Map<number,ICityLocation>
}

export class CountryCitiesService extends GetHandler {
  route = "country-cities?pais-id=604"
  useCache = { min: 600, ver: 2 }

	ciudades: ICityLocation[] = $state([]) // Departamentos + Provincias + Distritos
	ciudadesMap: Map<number, ICityLocation> = $state(new Map())
	
	distritos: ICityLocation[] = $state([])
	provincias: ICityLocation[] = $state([])
  departamentos: ICityLocation[] = $state([])

  handler(result: ICityLocation[]): void {
		const ciudades = result?.filter(x => !(x as any)._IS_META) || []
		const ciudadesMap = new Map(ciudades.map(e => [e.ID, e]))
		
		const distritos: ICityLocation[] = []
		const provincias: ICityLocation[] = []
		const departamentos: ICityLocation[] = []

		console.log("ciudades", ciudades)
		
		for (const e of ciudades) {			
      const padre = ciudadesMap.get(e.ParentID)
			if (e.Hierarchy === 3) { distritos.push(e) }
			else if (e.Hierarchy === 2) { provincias.push(e) }
			else if(e.Hierarchy === 1){ departamentos.push(e) }

			if (padre) {
        if(padre.Hierarchy === 2){ e.Provincia = padre }
				else if (padre.Hierarchy === 1) {
					e.Departamento = padre
				}
        if(padre.ParentID && ciudadesMap.has(padre.ParentID)){
          const padre2 = ciudadesMap.get(padre.ParentID)
          if(padre2?.Hierarchy === 2){ e.Provincia = padre2 }
          else if(padre2?.Hierarchy === 1){ e.Departamento = padre2 }
        }
      }
    }

    // Build display names
    for(const e of distritos){
      e._nombre = `${e.Departamento?.Name||"-"} ► ${e.Provincia?.Name||""} ► ${e.Name}`
    }

    // console.log("distritos:", distritos)

    this.ciudades = ciudades
		this.distritos = distritos
		this.provincias = provincias
    this.departamentos = departamentos
    this.ciudadesMap = ciudadesMap
  }

  constructor(init?: boolean){
		super()
    if(init){ this.fetch() }
  }
}
