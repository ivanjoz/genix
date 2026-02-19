import { normalizeStringN } from '$libs/helpers';
import { GetHandler, POST } from '$libs/http.svelte';

export interface IListaRegistro {
  ID: number;
  ListaID: number;
  Nombre: string;
  Images?: string[];
  Descripcion?: string;
  UpdatedBy?: number;
  ss: number;
  upd: number;
}

export interface IListas {
  Records: IListaRegistro[];
  RecordsMap: Map<number, IListaRegistro>;
}

export const listasCompartidas = [
  { id: 1, name: 'Categoría' },
  { id: 2, name: 'Marca' },
];

export class ListasCompartidasService extends GetHandler {
  route = 'listas-compartidas';
  useCache = { min: 5, ver: 6 };

  Records: IListaRegistro[] = $state([]);
  RecordsMap: Map<number, IListaRegistro> = $state(new Map());
  ListaRecordsMap: Map<number, IListaRegistro[]> = $state(new Map());

  nameToRecordMap: Map<number, Map<string, IListaRegistro>> = new Map();
  nextTempID = -1;

  private ensureNamesMap(listaID: number): Map<string, IListaRegistro> {
		if (!this.nameToRecordMap.has(listaID)) {
			const records = this.ListaRecordsMap.get(listaID) || []
      this.nameToRecordMap.set(listaID, new Map(records.map(e => [normalizeStringN(e.Nombre), e])));
    }
    return this.nameToRecordMap.get(listaID) as Map<string, IListaRegistro>;
  }

  get(id: number) {
    return this.RecordsMap.get(id);
  }

  getByName(listaID: number, name: string): IListaRegistro {
    const normalizedName = normalizeStringN(name);
    const namesMap = this.ensureNamesMap(listaID);
    return namesMap.get(normalizedName) as IListaRegistro;
  }

  handler(result: { [k: string]: IListaRegistro[] }): void {
    console.log('result getted::', result);

    this.Records = [];
    this.ListaRecordsMap = new Map();

    for (const [key, records] of Object.entries(result)) {
      const listaID = parseInt(key.split('_')[1]);

      for (const record of records) {
        if (!record.ss) continue;
        this.Records.push(record);

        if ([1, 2].includes(listaID)) {
          // Preserve category/brand image ordering by numeric suffix.
          const imagesMap = new Map(
            (record.Images || []).filter((imageName) => imageName).map((imageName) => [parseInt(imageName.split('-')[1]), imageName]),
          );

          record.Images = [];
          for (const order of [1, 2, 3]) {
            record.Images.push(imagesMap.get(order) || '');
          }
        }

        this.ListaRecordsMap.has(record.ListaID)
          ? this.ListaRecordsMap.get(record.ListaID)?.push(record)
          : this.ListaRecordsMap.set(record.ListaID, [record]);
      }
    }

    this.RecordsMap = new Map(this.Records.map((record) => [record.ID, record]));
    this.nameToRecordMap = new Map();
  }

	addNew(record: IListaRegistro, partial?: boolean, avoidRerender?: boolean) {
		if (!partial) {
			this.ensureNamesMap(record.ListaID).set(normalizeStringN(record.Nombre), record);
		}
		
		this.RecordsMap.set(record.ID, record);
    const records = this.ListaRecordsMap.get(record.ListaID) || [];
    records.unshift(record);
		this.Records.push(record);
		
		if (!avoidRerender) {
   		this.ListaRecordsMap.set(record.ListaID, [...records]);
    	this.ListaRecordsMap = new Map(this.ListaRecordsMap);
    }
  }

  addNewTemp(record: Omit<IListaRegistro, 'ID'>): IListaRegistro {
    const normalizedName = normalizeStringN(record.Nombre || '');
    const namesMap = this.ensureNamesMap(record.ListaID);
    const existingRecord = namesMap.get(normalizedName);
		if (existingRecord) return existingRecord;
    
    const tempRecord: IListaRegistro = {
      ...record,
      ID: this.nextTempID--,
      ss: record.ss || 1,
      upd: record.upd || 0,
    };
    this.RecordsMap.set(tempRecord.ID, tempRecord);
    namesMap.set(normalizedName, tempRecord);

    console.log('[listas-compartidas] temp record created:', tempRecord);
    return tempRecord;
  }

  clearTempRecords() {
    for (const [recordID, record] of this.RecordsMap.entries()) {
      if (recordID >= 0) continue;
			this.RecordsMap.delete(recordID);
			
			const namesMap = this.nameToRecordMap.get(record.ListaID)			
			for (const [key, e] of namesMap || new Map() as  Map<string, IListaRegistro>){
				if(e.ID <= 0){ namesMap?.delete(key) }
			}
    }
  }

  getTempRecordsCount(): number {
    return [...this.RecordsMap.keys()].filter(x => x < 0).length
  }

  async syncTemp(): Promise<Map<number, number>> {
    let tempToNewIDs = new Map<number, number>();
    const pendingRecordsByLista = new Map<number, IListaRegistro[]>();

    for (const pendingRecord of this.RecordsMap.values()) {
      if (pendingRecord.ID >= 0) continue;
      if (!pendingRecordsByLista.has(pendingRecord.ListaID)) {
        pendingRecordsByLista.set(pendingRecord.ListaID, []);
      }
      pendingRecordsByLista.get(pendingRecord.ListaID)?.push(pendingRecord);
    }

    for (const pendingRecords of pendingRecordsByLista.values()) {
      if (pendingRecords.length === 0) continue;
			const createdMappings = await postListaRegistros(pendingRecords);
			for(const e  of createdMappings){ tempToNewIDs.set(e.TempID, e.NewID) }
      
			for (const [id, record] of this.RecordsMap) {
				if (id <= 0 && tempToNewIDs.has(id)) {
					record.ID = tempToNewIDs.get(id) as number
					this.addNew(record, true, true)
				}
			}
		}
    
    // Clear synced temp records from in-memory indexes.
    this.clearTempRecords();
    return tempToNewIDs;
  }

  constructor(ids: number[] = []) {
    super();
    if (ids.length > 0) {
      this.route = `listas-compartidas?ids=${ids.join(',')}`;
    }
    if (ids) {
      this.fetch();
    }
  }
}

export interface INewIDToID {
  NewID: number;
  TempID: number;
}

export const postListaRegistros = (data: IListaRegistro[]): Promise<INewIDToID[]> => {
  return POST({
    data,
    route: 'listas-compartidas',
    refreshRoutes: ['listas-compartidas'],
  }) as Promise<INewIDToID[]>;
};
