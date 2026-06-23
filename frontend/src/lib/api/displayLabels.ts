const statusLabels: Record<string, string> = {
  pending: '申請中',
  under_review: '審査中',
  changes_requested: '修正依頼',
  approved: '承認済み',
  rejected: '却下',
  withdrawn: '取り下げ済み',
  active: '有効',
  inactive: '無効',
  revoked: '解除済み',
  disabled: '無効',
  enabled: '有効',
}

const appRequestTypeLabels: Record<string, string> = {
  web: 'Web アプリ',
  spa: 'SPA',
  native: 'ネイティブアプリ',
  m2m: 'M2M',
}

const appPartyTypeLabels: Record<string, string> = {
  first_party: '自社アプリ',
  third_party: '外部アプリ',
}

const auditResultLabels: Record<string, string> = {
  success: '成功',
  failure: '失敗',
  error: 'エラー',
}

const roleLabels: Record<string, string> = {
  admin: '管理者',
  developer: '開発者',
  user: 'ユーザー',
}

function fallbackLabel(value: string): string {
  return value.replaceAll('_', ' ')
}

export function statusLabel(status: string): string {
  return statusLabels[status.toLowerCase()] ?? fallbackLabel(status)
}

export function appRequestTypeLabel(type: string): string {
  return appRequestTypeLabels[type.toLowerCase()] ?? fallbackLabel(type)
}

export function appPartyTypeLabel(type: string): string {
  return appPartyTypeLabels[type.toLowerCase()] ?? fallbackLabel(type)
}

export function auditResultLabel(result: string): string {
  return auditResultLabels[result.toLowerCase()] ?? fallbackLabel(result)
}

export function roleLabel(role: string): string {
  return roleLabels[role.toLowerCase()] ?? fallbackLabel(role)
}
