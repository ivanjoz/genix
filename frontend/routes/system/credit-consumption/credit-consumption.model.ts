import { describeDeviceSignature, type IDeviceSignatureLabel } from '#core/device-signature.ts'

export const CREDIT_CONSUMPTION_WINDOW_HOURS = 4

/** One (IP, device) pair over the four hours: the sum of the frames where it made the top twelve. */
export interface ICreditConsumptionPair {
  IP: string
  /** The browser signature in decimal (an int64 does not survive a JSON number). */
  Device: string
  Requests: number
  CPU: number
  Frames: number
}

/** One company's window. Every array has one value per frame of the response's FrameIDs. */
export interface ICreditConsumptionCompany {
  CompanyID: number
  Name: string
  CPU: number[]
  Inference: number[]
  CreditRejected: number[]
  DeviceRejected: number[]
  Devices: number[]
  TotalCPU: number
  TotalInference: number
  TotalRejected: number
  MaxDevices: number
  Pairs: ICreditConsumptionPair[]
}

export interface ICreditConsumptionResponse {
  FrameIDs?: number[]
  Companies?: ICreditConsumptionCompany[]
}

export interface ICreditConsumptionPairRow extends ICreditConsumptionPair {
  Label: IDeviceSignatureLabel
}

/** Refusals of both gates per frame: the credit gates and the device gate. */
export const rejectedPerFrame = (company: ICreditConsumptionCompany): number[] =>
  company.CPU.map((_, index) => (company.CreditRejected[index] || 0) + (company.DeviceRejected[index] || 0))

export const sumValues = (values: number[] | undefined): number =>
  (values || []).reduce((total, value) => total + (Number(value) || 0), 0)

export const companyDisplayName = (company: ICreditConsumptionCompany): string =>
  company.Name || `Empresa #${company.CompanyID}`

export const creditConsumptionMatches = (company: ICreditConsumptionCompany, filterText: string): boolean => {
  const needle = filterText.trim().toLowerCase()
  if (!needle) return true
  return `${company.Name} ${company.CompanyID}`.toLowerCase().includes(needle)
}

export const buildPairRows = (company: ICreditConsumptionCompany | null): ICreditConsumptionPairRow[] =>
  (company?.Pairs || []).map(pair => ({ ...pair, Label: describeDeviceSignature(pair.Device) }))
