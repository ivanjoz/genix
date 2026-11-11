// Plain module so pure files can import it without the Svelte runtime.

/**
 * The {{placeholders}} an email template may use. Mirrors EmailPlaceholders in
 * backend/crm/types/email_template_rules.go, which refuses any other name on save. The quote ones
 * only render when the email is sent with a quote.
 */
export const EMAIL_PLACEHOLDERS = [
  { name: 'client.name', label: 'Client name|Nombre del cliente' },
  { name: 'client.email', label: 'Client email|Correo del cliente' },
  { name: 'opportunity.title', label: 'Opportunity title|Título de la oportunidad' },
  { name: 'opportunity.amount', label: 'Opportunity amount|Monto de la oportunidad' },
  { name: 'owner.name', label: 'Owner name|Nombre del responsable' },
  { name: 'owner.email', label: 'Owner email|Correo del responsable' },
  { name: 'company.name', label: 'Company name|Nombre de la empresa' },
  { name: 'quote.number', label: 'Quote number|Número de cotización' },
  { name: 'quote.total', label: 'Quote total|Total de la cotización' },
  { name: 'quote.valid_until', label: 'Quote valid until|Validez de la cotización' },
] as const

const PLACEHOLDER_PATTERN = /\{\{\s*([a-z_.]+)\s*\}\}/g

/** The placeholder names a text uses that the server would refuse, so the editor can warn before saving. */
export const unknownPlaceholders = (text: string): string[] => {
  const known = new Set<string>(EMAIL_PLACEHOLDERS.map((placeholder) => placeholder.name))
  return [...new Set([...text.matchAll(PLACEHOLDER_PATTERN)].map((match) => match[1]).filter((name) => !known.has(name)))]
}

/** "a@x.pe, b@x.pe; c@x.pe" → the addresses, trimmed, blanks dropped. The server validates each one. */
export const parseEmailAddresses = (text: string): string[] =>
  text.split(/[,;\s]+/).map((address) => address.trim()).filter(Boolean)
