/**
 * The browser's device signature: one int64 the frontend appends to the session token
 * (`Bearer <token>.<signature>`) and fareward counts per company as (IP, device) pairs.
 *
 * The layout and every list below mirror fareward/DEVICE_SIGNATURE.md, which is the contract. IDs
 * are append-only: never renumber or reuse one, or every stored signature changes meaning.
 * Fields are packed from the most significant bit down, in LAYOUT order.
 */

export const DEVICE_SIGNATURE_LAYOUT = [
  ['browser', 6], ['version', 8], ['os', 5], ['timeZone', 6], ['language', 6], ['hasCountry', 1],
  ['countryOrLanguage', 6], ['screenWidth', 6], ['screenHeight', 6], ['cores', 6],
  ['pixelRatio', 3], ['touch', 1], ['automation', 1], ['memory', 3],
] as const

export type DeviceSignatureField = typeof DEVICE_SIGNATURE_LAYOUT[number][0]
export type IDeviceSignature = Record<DeviceSignatureField, number>

/** Index = ID. Empty strings are IDs not assigned yet. */
export const DEVICE_BROWSERS: string[] = [
  '', 'Chrome', 'Edge', 'Firefox', 'Safari', 'Opera', 'Samsung Internet', 'Brave', 'Vivaldi', 'Yandex',
  'UC Browser', 'Chrome iOS', 'Firefox iOS', 'Android WebView', 'Electron', 'Opera GX', 'Opera Mini',
  'Arc', 'DuckDuckGo', 'Chromium', 'Edge iOS', 'Edge Android', 'Opera iOS', 'Huawei Browser',
  'Mi Browser', 'HeyTap', 'Vivo Browser', 'QQ Browser', 'Baidu', '360 Browser', 'Sogou', 'Whale',
  'Silk', 'Puffin', 'Maxthon', 'Kiwi', 'Firefox Focus', 'Ecosia', 'Avast Secure', 'AVG Secure',
  'Waterfox', 'LibreWolf', 'Floorp', 'Zen', 'Tor Browser', 'Mullvad Browser', 'Comet',
  'ChatGPT Atlas', 'Google app', 'Facebook', 'Instagram', 'TikTok', 'Snapchat', 'X', 'LinkedIn',
  'WeChat', 'LINE', 'iOS WebView', 'Internet Explorer', 'Edge Legacy', 'Headless Chrome', 'Bot',
  '', 'Other',
]

export const DEVICE_OPERATING_SYSTEMS: string[] = [
  '', 'Windows', 'macOS', 'Linux', 'Android', 'iOS', 'iPadOS', 'ChromeOS', 'HarmonyOS',
]
DEVICE_OPERATING_SYSTEMS[31] = 'Other'

const REGION_GROUPS_LANGUAGE = [
  'Europe|Europa', 'Asia|Asia', 'Africa|África', 'Americas|América', 'Oceania|Oceanía',
  'Middle East|Medio Oriente',
]
const REGION_GROUPS_COUNTRY = [
  'Americas|América', 'Europe|Europa', 'Asia|Asia', 'Middle East & N. Africa|Medio Oriente y N. África',
  'Africa|África', 'Oceania|Oceanía',
]

/** ISO 639-1. 1–55 listed, 56–61 regional groups, 62 reserved, 63 unreadable. */
export const DEVICE_LANGUAGES: string[] = [
  '', 'es', 'en', 'pt', 'fr', 'de', 'it', 'nl', 'zh', 'ja', 'ko', 'ru', 'ar',
  'hi', 'tr', 'pl', 'sv', 'da', 'no', 'fi', 'cs', 'el', 'he', 'id', 'ms',
  'th', 'vi', 'uk', 'ro', 'hu', 'ca', 'eu', 'gl', 'qu', 'ay', 'gn', 'fil',
  'bn', 'ur', 'fa', 'sk', 'bg', 'hr', 'sr', 'sl', 'lt', 'lv', 'et', 'ta',
  'te', 'sw', 'af', 'zu', 'ka', 'hy', 'kk',
]

/** ISO 3166-1 alpha-2 plus 419 (Latin America). Same ID ranges as the languages. */
export const DEVICE_COUNTRIES: string[] = [
  '', 'PE', 'US', 'MX', 'CL', 'CO', 'AR', 'EC', 'BO', 'BR', 'VE',
  'UY', 'PY', 'CR', 'PA', 'GT', 'DO', 'SV', 'HN', 'NI', 'CU',
  'PR', 'ES', 'CA', 'GB', 'FR', 'DE', 'IT', 'PT', 'NL', 'BE',
  'CH', 'AT', 'IE', 'SE', 'NO', 'DK', 'FI', 'PL', 'CN', 'JP',
  'KR', 'IN', 'AU', 'NZ', 'MA', 'ZA', 'TW', 'HK', 'SG', 'PH',
  'RU', 'TR', 'IL', 'AE', '419',
]

const PIXEL_RATIOS = ['', '1x', '1.25x', '1.5x', '2x', '2.5x', '3x', 'other']
const MEMORY_SIZES = ['', '0.25 GB', '0.5 GB', '1 GB', '2 GB', '4 GB', '8+ GB', 'other']

/** Splits a signature (decimal, signed or unsigned, as the backend sends it) into its fields. */
export const decodeDeviceSignature = (signature: string | number | bigint): IDeviceSignature => {
  let bits: bigint
  try {
    bits = BigInt.asUintN(64, BigInt(signature))
  } catch {
    bits = 0n
  }
  const fields = {} as IDeviceSignature
  let shift = 64n
  for (const [name, width] of DEVICE_SIGNATURE_LAYOUT) {
    shift -= BigInt(width)
    fields[name] = Number((bits >> shift) & ((1n << BigInt(width)) - 1n))
  }
  return fields
}

const listedOrGroup = (list: string[], groups: string[], id: number): string => {
  if (id <= 0) return ''
  if (list[id]) return list[id]
  if (id >= 56 && id <= 61) return `${groups[id - 56].split('|')[1]} (otro)`
  return '?'
}

/** UTC offset of a time-zone field: 30-minute steps around 24. */
const formatTimeZone = (value: number): string => {
  if (value === 63) return ''
  const minutes = (value - 24) * 30
  const sign = minutes < 0 ? '-' : '+'
  const absolute = Math.abs(minutes)
  const hours = Math.floor(absolute / 60)
  const rest = absolute % 60
  return `UTC${sign}${hours}${rest ? `:${String(rest).padStart(2, '0')}` : ''}`
}

/** The parts a person reads: browser and version, OS, locale, screen, and the warning flags. */
export interface IDeviceSignatureLabel {
  Browser: string
  OS: string
  Locale: string
  Screen: string
  TimeZone: string
  Hardware: string
  IsAutomated: boolean
  IsTouch: boolean
  IsEmpty: boolean
}

export const describeDeviceSignature = (signature: string | number | bigint): IDeviceSignatureLabel => {
  const fields = decodeDeviceSignature(signature)
  const isEmpty = Object.values(fields).every(value => value === 0)
  const browserName = DEVICE_BROWSERS[fields.browser] || (fields.browser ? `#${fields.browser}` : '')
  const version = fields.version ? ` ${fields.version === 255 ? '255+' : fields.version}` : ''
  const language = listedOrGroup(DEVICE_LANGUAGES, REGION_GROUPS_LANGUAGE, fields.language)
  const second = fields.hasCountry
    ? listedOrGroup(DEVICE_COUNTRIES, REGION_GROUPS_COUNTRY, fields.countryOrLanguage)
    : listedOrGroup(DEVICE_LANGUAGES, REGION_GROUPS_LANGUAGE, fields.countryOrLanguage)
  const locale = fields.hasCountry
    ? [language, second].filter(Boolean).join('-')
    : [language, second].filter(Boolean).join(', ')
  const screen = fields.screenWidth && fields.screenHeight
    ? `${fields.screenWidth * 40}×${fields.screenHeight * 40}`
    : ''
  const hardware = [
    fields.cores && fields.cores < 63 ? `${fields.cores} cores` : '',
    PIXEL_RATIOS[fields.pixelRatio] || '',
    MEMORY_SIZES[fields.memory] || '',
  ].filter(Boolean).join(' · ')
  return {
    Browser: browserName ? `${browserName}${version}` : '',
    OS: DEVICE_OPERATING_SYSTEMS[fields.os] || (fields.os ? `#${fields.os}` : ''),
    Locale: locale,
    Screen: screen,
    TimeZone: formatTimeZone(fields.timeZone),
    Hardware: hardware,
    IsAutomated: fields.automation === 1 || fields.browser === 60 || fields.browser === 61,
    IsTouch: fields.touch === 1,
    IsEmpty: isEmpty,
  }
}
