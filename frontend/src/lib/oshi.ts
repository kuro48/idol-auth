export const OSHI_PALETTE = [
  '#ffb2b2',
  '#ffb2d8',
  '#ffb2ff',
  '#d8b2ff',
  '#b2b2ff',
  '#b2d8ff',
  '#b2ffff',
  '#b2ffd8',
  '#b2ffb2',
  '#d8ffb2',
  '#ffffb2',
  '#ffd8b2',
] as const

export const MAX_OSHI_COUNT = 10

export interface OshiEntry {
  idol_id: string
  fan_since?: string
}

export function normalizeOshiColor(raw: string | undefined | null): string {
  if (!raw) return ''
  const normalized = raw.trim().toLowerCase()
  return (OSHI_PALETTE as readonly string[]).includes(normalized) ? normalized : ''
}

const FAN_SINCE_PATTERN = /^\d{4}(-(0[1-9]|1[0-2]))?$/

export function validateFanSince(value: string, now: Date = new Date()): string | null {
  const trimmed = value.trim()
  if (trimmed === '') return null
  if (!FAN_SINCE_PATTERN.test(trimmed)) {
    return 'YYYY または YYYY-MM 形式で入力してください'
  }
  const year = Number(trimmed.slice(0, 4))
  const month = trimmed.length === 7 ? Number(trimmed.slice(5, 7)) : 1
  const start = new Date(Date.UTC(year, month - 1, 1))
  const nowMonth = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1))
  if (start.getTime() > nowMonth.getTime()) {
    return '未来の日付は指定できません'
  }
  if (year < 1900) {
    return '日付が古すぎます'
  }
  return null
}
