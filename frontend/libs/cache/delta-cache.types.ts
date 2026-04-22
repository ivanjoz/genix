export type CacheRecordID = string | number

export interface ILastSync {
  fetchTime: number
  updatedStatus: { [key: string]: number }
  fetchedRecordsCount: number
  fetchedBytes: number
  forceNetwork?: boolean
  __version__: number
}

export interface IDeltaCacheRouteRef {
  env: string
  companyID: number
  dbName: string
  module: string
  route: string
  partitionValue: string
  cacheKey: string
  routeLookupKey: string
  version: number
}

export interface ICacheRouteRow extends ILastSync {
  id?: number
  dbName: string
  routeLookupKey: string
  env: string
  module: string
  route: string
  partitionValue: string
  cacheKey: string
  responseKeys: string[]
  // Routes whose backend responses are bare arrays (normalized to `_default`) persist records
  // in `cacheRecordsSingle` and skip the `_k` field. The flag is set on the first fetch.
  isSingle?: boolean
}

export interface ICacheRecordRowMulti {
  _r: number
  _k: number
  ID: CacheRecordID
  ss: number
  [field: string]: any
}

export interface ICacheRecordRowSingle {
  _r: number
  ID: CacheRecordID
  ss: number
  [field: string]: any
}

export type ICacheRecordRow = ICacheRecordRowMulti | ICacheRecordRowSingle

export interface IRequestLogRow {
  id: number /* unix milliseconds timestamp */
  route: string
  qp: string /* query params */
  sPs: number /* server pre-parsing request time */
  sF: number /* server final request time */
  req: number /* client request time */
  spc: number /* client serialize, parse and cache loading time  */
  size: number /* size in kb */
}
