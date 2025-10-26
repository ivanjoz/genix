export interface IMenuRecord {
  name: string, minName?: string, id?: number, route?: string,
  options?: IMenuRecord[], icon?: string
}
