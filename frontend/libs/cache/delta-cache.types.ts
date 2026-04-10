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
}

export interface ICacheRecordRow {
  cR: number
  rK: string
  ID: CacheRecordID
  E: any
  ss: number
}
