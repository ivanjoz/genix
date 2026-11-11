import { describe, expect, it } from 'vitest'
import { parseEmailAddresses, unknownPlaceholders } from '#services/crm/crm-email-template.ts'

describe('email template helpers', () => {
  it('reports only the placeholders the server would refuse, once each', () => {
    expect(unknownPlaceholders('Hi {{client.name}}, {{ quote.total }} {{client.nmae}} {{client.nmae}} {{owner}}')).toEqual(['client.nmae', 'owner'])
    expect(unknownPlaceholders('no placeholders')).toEqual([])
    // The editor's variable markup: only the {{name}} inside counts.
    expect(unknownPlaceholders('<span data-variable="client.email">{{client.email}}</span>')).toEqual([])
  })

  it('splits recipients on commas, semicolons and spaces', () => {
    expect(parseEmailAddresses(' a@x.pe, b@x.pe;c@x.pe  d@x.pe ,')).toEqual(['a@x.pe', 'b@x.pe', 'c@x.pe', 'd@x.pe'])
    expect(parseEmailAddresses('  ')).toEqual([])
  })
})
