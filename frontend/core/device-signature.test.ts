import { describe, expect, it } from 'bun:test'
import {
  DEVICE_BROWSERS,
  DEVICE_COUNTRIES,
  DEVICE_LANGUAGES,
  DEVICE_SIGNATURE_LAYOUT,
  decodeDeviceSignature,
  describeDeviceSignature,
  type IDeviceSignature,
} from './device-signature'

/** Packs fields the way DEVICE_SIGNATURE.md's reference packing does. */
const pack = (fields: Partial<IDeviceSignature>): bigint => {
  let signature = 0n
  for (const [name, width] of DEVICE_SIGNATURE_LAYOUT) {
    signature = (signature << BigInt(width)) | BigInt(fields[name] || 0)
  }
  return signature
}

describe('device signature', () => {
  it('fills exactly 64 bits and keeps the lists inside their fields', () => {
    expect(DEVICE_SIGNATURE_LAYOUT.reduce((total, [, width]) => total + width, 0)).toBe(64)
    expect(DEVICE_BROWSERS.length).toBeLessThanOrEqual(64)
    expect(DEVICE_LANGUAGES.length).toBe(56)
    expect(DEVICE_COUNTRIES.length).toBe(56)
  })

  it('reads Chrome 131 on Windows in es-PE at 1920×1080 back, signed or unsigned', () => {
    const fields: Partial<IDeviceSignature> = {
      browser: 1, version: 131, os: 1, timeZone: 14, language: 1, hasCountry: 1,
      countryOrLanguage: 1, screenWidth: 48, screenHeight: 27, cores: 8, pixelRatio: 1,
      memory: 6,
    }
    const unsigned = pack(fields)
    const signed = BigInt.asIntN(64, unsigned)
    expect(decodeDeviceSignature(unsigned.toString())).toEqual(decodeDeviceSignature(signed.toString()))
    expect(describeDeviceSignature(signed.toString())).toEqual({
      Browser: 'Chrome 131',
      OS: 'Windows',
      Locale: 'es-PE',
      Screen: '1920×1080',
      TimeZone: 'UTC-5',
      Hardware: '8 cores · 1x · 8+ GB',
      IsAutomated: false,
      IsTouch: false,
      IsEmpty: false,
    })
  })

  it('shows a secondary language when there is no country, and flags automation', () => {
    const label = describeDeviceSignature(pack({
      browser: 60, language: 2, hasCountry: 0, countryOrLanguage: 1, timeZone: 63,
    }).toString())
    expect(label.Browser).toBe('Headless Chrome')
    expect(label.Locale).toBe('en, es')
    expect(label.TimeZone).toBe('')
    expect(label.IsAutomated).toBe(true)
  })

  it('names the regional groups and treats 0 or garbage as no signature', () => {
    expect(describeDeviceSignature(pack({ language: 1, hasCountry: 1, countryOrLanguage: 57 })).Locale)
      .toBe('es-Europa (otro)')
    expect(describeDeviceSignature('0').IsEmpty).toBe(true)
    expect(describeDeviceSignature('not a number').IsEmpty).toBe(true)
  })
})
