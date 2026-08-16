import { describe, expect, it } from 'bun:test'
import { normalizeAccessFrontendRoutes } from './access-list-catalog'

describe('access-list catalog routes', () => {
  it('expands a comma-separated frontend route list', () => {
    // One access can intentionally unlock closely related pages.
    expect(normalizeAccessFrontendRoutes('system/server-panel, /system/observability')).toEqual([
      'system/server-panel',
      'system/observability',
    ])
  })
})
