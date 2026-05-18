import { GetHandler, POST, type INewIDToID as IBaseNewIDToID } from '$libs/http.svelte';

export interface ISharedListRecord {
  ID: number;
  ListaID: number;
  Nombre: string;
  Images?: string[];
  Descripcion?: string;
  UpdatedBy?: number;
  ss: number;
  upd: number;
}

export interface ISharedLists {
  records: ISharedListRecord[];
  recordsMap: Map<number, ISharedListRecord>;
}

export const listasCompartidas = [
  { id: 1, name: 'Categoría' },
  { id: 2, name: 'Marca' },
];

export class ListasCompartidasService extends GetHandler<ISharedListRecord> {
  route = 'listas-compartidas';
  useCache = { min: 5, ver: 6 };
  inferRemoveFromStatus = true

  ListaRecordsMap: Map<number, ISharedListRecord[]> = $state(new Map());
	
	makeName(e: Partial<ISharedListRecord>) {
		return [e.ListaID, e.Nombre].join("_")
	}
  
  handler(result: { [k: string]: ISharedListRecord[] }): void {
    console.log('result getted::', result);

    this.records = [];
    this.recordsMap = new Map();
    this.ListaRecordsMap = new Map();
    this.nameToRecordMap = new Map();
    const savedRecords: ISharedListRecord[] = []

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

	afterSaveRecords(...records: ISharedListRecord[]) {
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

  constructor(ids: number[] = [], init: boolean = false) {
    super();
    if (ids.length > 0) {
      this.route = `listas-compartidas?ids=${ids.join(',')}`;
    }
    if (init) {
      this.fetch();
    }
  }
}

export type INewIDToID = IBaseNewIDToID

export const postListaRegistros = (data: ISharedListRecord[]): Promise<INewIDToID[]> => {
  return POST({
    data,
    route: 'listas-compartidas',
    refreshRoutes: ['listas-compartidas'],
  }) as Promise<INewIDToID[]>;
};
