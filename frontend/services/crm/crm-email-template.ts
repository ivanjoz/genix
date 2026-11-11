// Plain module so pure files can import it without the Svelte runtime.

/**
 * The {{placeholders}} an email template may use. Mirrors EmailPlaceholders in
 * backend/crm/types/email_template_rules.go, which refuses any other name on save. The quote ones
 * only render when the email is sent with a quote. The editor shows each one as "group: field"
 * (e.g. "Cliente: Email") and groups the picker by group.
 */
export const EMAIL_PLACEHOLDERS = [
  { name: 'client.name', group: 'Client|Cliente', field: 'Name|Nombre' },
  { name: 'client.email', group: 'Client|Cliente', field: 'Email|Email' },
  { name: 'opportunity.title', group: 'Opportunity|Oportunidad', field: 'Title|Título' },
  { name: 'opportunity.amount', group: 'Opportunity|Oportunidad', field: 'Amount|Monto' },
  { name: 'owner.name', group: 'Owner|Responsable', field: 'Name|Nombre' },
  { name: 'owner.email', group: 'Owner|Responsable', field: 'Email|Email' },
  { name: 'company.name', group: 'Company|Empresa', field: 'Name|Nombre' },
  { name: 'quote.number', group: 'Quote|Cotización', field: 'Number|Número' },
  { name: 'quote.total', group: 'Quote|Cotización', field: 'Total|Total' },
  { name: 'quote.valid_until', group: 'Quote|Cotización', field: 'Valid until|Validez' },
]

const PLACEHOLDER_PATTERN = /\{\{\s*([a-z_.]+)\s*\}\}/g

/** The placeholder names a text uses that the server would refuse, so the editor can warn before saving. */
export const unknownPlaceholders = (text: string): string[] => {
  const known = new Set<string>(EMAIL_PLACEHOLDERS.map((placeholder) => placeholder.name))
  return [...new Set([...text.matchAll(PLACEHOLDER_PATTERN)].map((match) => match[1]).filter((name) => !known.has(name)))]
}

/** "a@x.pe, b@x.pe; c@x.pe" → the addresses, trimmed, blanks dropped. The server validates each one. */
export const parseEmailAddresses = (text: string): string[] =>
  text.split(/[,;\s]+/).map((address) => address.trim()).filter(Boolean)
