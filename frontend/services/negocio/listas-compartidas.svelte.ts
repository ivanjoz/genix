import { GetHandler, POST, type INewIDToID as IBaseNewIDToID } from '$libs/http.svelte';

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
  records: IListaRegistro[];
  recordsMap: Map<number, IListaRegistro>;
}

export const listasCompartidas = [
  { id: 1, name: 'Categoría' },
  { id: 2, name: 'Marca' },
];

export class ListasCompartidasService extends GetHandler<IListaRegistro> {
  route = 'listas-compartidas';
  useCache = { min: 5, ver: 6 };
  inferRemoveFromStatus = true

  ListaRecordsMap: Map<number, IListaRegistro[]> = $state(new Map());
	
	makeName(e: Partial<IListaRegistro>) {
		return [e.ListaID, e.Nombre].join("_")
	}
  
  handler(result: { [k: string]: IListaRegistro[] }): void {
    console.log('result getted::', result);

    this.records = [];
    this.recordsMap = new Map();
    this.ListaRecordsMap = new Map();
    this.nameToRecordMap = new Map();
    const savedRecords: IListaRegistro[] = []

    for (const [key, recordGroups] of Object.entries(result)) {
      const listaID = parseInt(key.split('_')[1]);

      for (const record of recordGroups) {
        if (!record.ss) continue;

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

        savedRecords.push(record)
      }
    }
    
    this.addSavedRecords(...savedRecords)
    this.afterSaveRecords(...savedRecords)
  }

	afterSaveRecords(...records: IListaRegistro[]) {
    for (const record of records) {
      const currentListaRecords = this.ListaRecordsMap.get(record.ListaID) || [];
      const existingRecordPosition = currentListaRecords.findIndex((existingRecord) => existingRecord.ID === record.ID)
      if (existingRecordPosition >= 0) {
        currentListaRecords[existingRecordPosition] = record
      } else {
        currentListaRecords.push(record)
      }
      this.ListaRecordsMap.set(record.ListaID, currentListaRecords);
    }
    
    this.ListaRecordsMap = new Map(this.ListaRecordsMap);
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

export type INewIDToID = IBaseNewIDToID

export const postListaRegistros = (data: IListaRegistro[]): Promise<INewIDToID[]> => {
  return POST({
    data,
    route: 'listas-compartidas',
    refreshRoutes: ['listas-compartidas'],
  }) as Promise<INewIDToID[]>;
};
