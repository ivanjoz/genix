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

  get(id: number) {
    return this.RecordsMap.get(id);
  }

  getByName(listaID: number, name: string): IListaRegistro {
    if (!this.nameToRecordMap.has(listaID)) {
      const namesMap: Map<string, IListaRegistro> = new Map();
      for (const record of this.ListaRecordsMap.get(listaID) || []) {
        namesMap.set(normalizeStringN(record.Nombre), record);
      }
      this.nameToRecordMap.set(listaID, namesMap);
    }

    const namesMap = this.nameToRecordMap.get(listaID);
    return namesMap?.get(normalizeStringN(name)) as IListaRegistro;
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

  addNew(record: IListaRegistro) {
    const records = this.ListaRecordsMap.get(record.ListaID) || [];
    records.unshift(record);
    this.ListaRecordsMap.set(record.ListaID, [...records]);
    this.ListaRecordsMap = new Map(this.ListaRecordsMap);
    this.RecordsMap.set(record.ID, record);
    this.Records.push(record);
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

export const postListaRegistros = (data: IListaRegistro[]) => {
  return POST({
    data,
    route: 'listas-compartidas',
    refreshRoutes: ['listas-compartidas'],
  });
};
