import { describe, expect, it } from 'vitest'
import {
  docTypeName, isNoteDocType, sunatDocCode, DOC_TYPES, DOC_TYPE_BOLETA, DOC_TYPE_CREDIT_NOTE,
  DOC_TYPE_DEBIT_NOTE, DOC_TYPE_FACTURA,
} from './sunat-doc-type'

// The ids are SUNAT's own codes as numbers, which is what lets sunatDocCode be a pad rather
// than a lookup. A renumbering that broke that would send the wrong type to SUNAT.
it('pads the id into the two-character code the annex wants', () => {
  expect(sunatDocCode(DOC_TYPE_FACTURA)).toBe("01")
  expect(sunatDocCode(DOC_TYPE_BOLETA)).toBe("03")
  expect(sunatDocCode(DOC_TYPE_CREDIT_NOTE)).toBe("07")
  expect(sunatDocCode(DOC_TYPE_DEBIT_NOTE)).toBe("08")
})

// 0 is "no type", which a row with an unresolved series carries. It must not become "00",
// which is a real SUNAT code for something else.
it('has no code for no type', () => {
  expect(sunatDocCode(0)).toBe("")
})

describe('the catalog is the only place the four types are written down', () => {
  it('names every one of them', () => {
    for (const docType of DOC_TYPES) {
      expect(docTypeName(docType.ID)).toBe(docType.Name)
    }
    expect(docTypeName(99)).toBe("—")
  })

  it('knows which types correct another document', () => {
    expect(isNoteDocType(DOC_TYPE_CREDIT_NOTE)).toBe(true)
    expect(isNoteDocType(DOC_TYPE_DEBIT_NOTE)).toBe(true)
    expect(isNoteDocType(DOC_TYPE_FACTURA)).toBe(false)
    expect(isNoteDocType(DOC_TYPE_BOLETA)).toBe(false)
  })
})
