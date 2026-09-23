// SUNAT's tipo de documento de identidad (Anexo 1, RS 112-2021). Pure: no Svelte, no fetch.
//
// Split out of client-provider.svelte.ts so the accounting books can convert an identity
// without pulling the service, its HTTP runtime and the by-ids cache into a module that is
// supposed to be a pure function of its arguments. It stays under `services` because that is
// where the //CATALOG: generator looks.

export interface IIdentityDocOption {
  id: number
  /** SUNAT's own character (Anexo 1). Here so a screen can show it; never sent. */
  code: string
  name: string
  /** Abbreviation for a control too narrow for the name — the till's in-input picker. */
  label?: string
}

/** Generated from `IdentityDocOptions` in backend/crm/types/identity_doc_type.go — run
 *  `go run ./scripts sync_struct_interfaces` after changing it there. The backend derives
 *  the type from the registry number's shape when a record is saved with 0. */
//CATALOG:crm.IdentityDocOptions
export const IDENTITY_DOC_OPTIONS: IIdentityDocOption[] = [
  { id: 1, code: '1', name: 'DNI|DNI' },
  { id: 6, code: '6', name: 'RUC|RUC' },
  { id: 4, code: '4', name: 'Foreign ID|Carné de extranjería', label: 'F.ID|C.E.' },
  { id: 7, code: '7', name: 'Passport|Pasaporte', label: 'PSP' },
  { id: 9, code: '0', name: 'No RUC (non-domiciled)|Sin RUC (no domiciliado)' },
  { id: 10, code: 'A', name: 'Diplomatic ID|Cédula diplomática' },
  { id: 11, code: 'B', name: 'Residence country doc|Doc. país de residencia' },
  { id: 12, code: 'C', name: 'TIN|TIN' },
  { id: 13, code: 'D', name: 'IN|IN' },
  { id: 14, code: 'E', name: 'TAM|TAM' },
  { id: 15, code: 'F', name: 'PTP|PTP' },
]

/** The ids the app names directly. Everything else is reached through the catalog. */
export const IdentityDocID = { DNI: 1, RUC: 6, FOREIGN: 4, PASSPORT: 7 } as const

/** What the till offers: the four a counter actually sees. The CRM maintainer shows the
 *  whole catalog, because a diplomatic id has to be reachable somewhere. */
export const TILL_IDENTITY_DOC_OPTIONS = IDENTITY_DOC_OPTIONS.filter(
  (option) => (Object.values(IdentityDocID) as number[]).includes(option.id),
)

/** Guesses the document type from the number's shape, mirroring DeriveIdentityDocType in
 *  backend/crm/types/identity_doc_type.go: in Peru eleven digits is a RUC and eight a DNI.
 *
 *  It exists so the picker shows the type the backend would have derived anyway, instead of
 *  displaying a stale default while the server stores something else. It is a guess the user
 *  can always override — the shape cannot tell a pasaporte from a carné de extranjería. */
export const deriveIdentityDocType = (registryNumber: string): number | undefined => {
  const digits = (registryNumber || '').trim()
  if (digits.length === 11) return IdentityDocID.RUC
  if (digits.length === 8) return IdentityDocID.DNI
  return undefined
}

/** SUNAT's character for a stored document type — column 11 of the sales book, and what the
 *  XML carries. Twin of SunatIdentityDocCode in backend/crm/types/identity_doc_type.go. */
export const sunatIdentityDocCode = (identityDocType: number): string =>
  IDENTITY_DOC_OPTIONS.find((option) => option.id === identityDocType)?.code || ''
