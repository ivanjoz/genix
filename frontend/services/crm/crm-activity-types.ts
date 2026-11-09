// The fixed activity catalogs of backend/crm/types/activity.go, filled by the //CATALOG: sync.
// A plain .ts file (no Svelte runtime) so the pure rules in routes/crm/pipeline/pipeline.ts import it.

export interface IActivityTypeOption {
  id: number
  name: string
  icon: string
}

/** The common activity types every company has (IDs 1..40, reserved). Company types start at 41. */
//CATALOG:crm.SystemActivityTypes
export const SYSTEM_ACTIVITY_TYPES: IActivityTypeOption[] = [
  { id: 1, name: 'Note|Nota', icon: 'icon-[fa--sticky-note]' },
  { id: 2, name: 'Call|Llamada', icon: 'icon-[fa--phone]' },
  { id: 3, name: 'Meeting|Reunión', icon: 'icon-[fa--users]' },
  { id: 4, name: 'Email|Email', icon: 'icon-[fa--envelope]' },
  { id: 5, name: 'WhatsApp|WhatsApp', icon: 'icon-[fa--whatsapp]' },
  { id: 6, name: 'Visit|Visita', icon: 'icon-[fa--map-marker]' },
  { id: 7, name: 'Proposal sent|Propuesta enviada', icon: 'icon-[fa--file-text]' },
  { id: 8, name: 'Demo|Demostración', icon: 'icon-[fa--desktop]' },
  { id: 9, name: 'Follow-up|Seguimiento', icon: 'icon-[fa--refresh]' },
]

/** The icons a company activity type may pick: the CSS only ships icons named in the source. */
//CATALOG:crm.ActivityIconOptions
export const ACTIVITY_ICON_OPTIONS: { icon: string }[] = [
  { icon: 'icon-[fa--sticky-note]' },
  { icon: 'icon-[fa--phone]' },
  { icon: 'icon-[fa--users]' },
  { icon: 'icon-[fa--envelope]' },
  { icon: 'icon-[fa--whatsapp]' },
  { icon: 'icon-[fa--map-marker]' },
  { icon: 'icon-[fa--file-text]' },
  { icon: 'icon-[fa--desktop]' },
  { icon: 'icon-[fa--refresh]' },
  { icon: 'icon-[fa--calendar]' },
  { icon: 'icon-[fa--handshake-o]' },
  { icon: 'icon-[fa--truck]' },
  { icon: 'icon-[fa--money]' },
  { icon: 'icon-[fa--video-camera]' },
  { icon: 'icon-[fa--comments]' },
  { icon: 'icon-[fa--flag]' },
]
