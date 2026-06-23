import { describe, expect, it } from 'vitest'
import {
  auditResultLabel,
  appPartyTypeLabel,
  appRequestTypeLabel,
  statusLabel,
} from './displayLabels'

describe('display label helpers', () => {
  it('translates known statuses', () => {
    expect(statusLabel('pending')).toBe('申請中')
    expect(statusLabel('under_review')).toBe('審査中')
    expect(statusLabel('changes_requested')).toBe('修正依頼')
    expect(statusLabel('approved')).toBe('承認済み')
    expect(statusLabel('rejected')).toBe('却下')
    expect(statusLabel('withdrawn')).toBe('取り下げ済み')
    expect(statusLabel('active')).toBe('有効')
    expect(statusLabel('inactive')).toBe('無効')
    expect(statusLabel('revoked')).toBe('解除済み')
  })

  it('translates app request and party types', () => {
    expect(appRequestTypeLabel('web')).toBe('Web アプリ')
    expect(appRequestTypeLabel('spa')).toBe('SPA')
    expect(appRequestTypeLabel('native')).toBe('ネイティブアプリ')
    expect(appRequestTypeLabel('m2m')).toBe('M2M')
    expect(appPartyTypeLabel('first_party')).toBe('自社アプリ')
    expect(appPartyTypeLabel('third_party')).toBe('外部アプリ')
  })

  it('translates audit results', () => {
    expect(auditResultLabel('success')).toBe('成功')
    expect(auditResultLabel('failure')).toBe('失敗')
  })
})
