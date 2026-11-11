import { describe, expect, test } from 'vitest';
import { SENT_EMAIL_STATUS, describeDelivery, listAllRecipients, sentEmailMatchesFilter, type ISentEmail } from './email-inbox.ts';

const makeSentEmail = (overrides: Partial<ISentEmail> = {}): ISentEmail => ({
  CompanyID: 1, ID: 1, To: ['ana@genix.pe'], Cc: ['luis@genix.pe'], Bcc: ['x@genix.pe'],
  Subject: 'Convocatoria', Sender: 'noreply@genix.pe', ss: SENT_EMAIL_STATUS.STORED, upd: 0, ...overrides,
})

describe('describeDelivery', () => {
  test('names each status', () => {
    expect(describeDelivery(makeSentEmail()).text).toBe('Only stored, not sent|Solo registrado, no enviado')
    expect(describeDelivery(makeSentEmail({ ss: SENT_EMAIL_STATUS.SENT })).text).toBe('Sent|Enviado')
    expect(describeDelivery(makeSentEmail({ ss: SENT_EMAIL_STATUS.FAILED })).css).toBe('text-red-fg')
  })
})

describe('listAllRecipients', () => {
  test('joins To, Cc and Bcc in that order', () => {
    expect(listAllRecipients(makeSentEmail())).toBe('ana@genix.pe, luis@genix.pe, x@genix.pe')
  })
})

describe('sentEmailMatchesFilter', () => {
  test('matches subject or any address, case-insensitive', () => {
    expect(sentEmailMatchesFilter(makeSentEmail(), 'convo')).toBe(true)
    expect(sentEmailMatchesFilter(makeSentEmail(), 'X@GENIX')).toBe(true)
    expect(sentEmailMatchesFilter(makeSentEmail(), 'pedro')).toBe(false)
    expect(sentEmailMatchesFilter(makeSentEmail(), '  ')).toBe(true)
  })
})
