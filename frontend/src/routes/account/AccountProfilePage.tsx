import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { PageHeader } from '@/components/ui/PageHeader'
import {
  OSHI_PALETTE,
  MAX_OSHI_COUNT,
  normalizeOshiColor,
  validateFanSince,
  type OshiEntry,
} from '@/lib/oshi'
import styles from './AccountPage.module.css'

interface Profile {
  identity_id: string
  display_name?: string
  oshi_color?: string
  avatar_url?: string
  email?: string
  oshis?: OshiEntry[]
}

interface ProfileFormState {
  displayName: string
  oshiColor: string
  oshis: OshiEntry[]
}

function profileToForm(profile: Profile): ProfileFormState {
  return {
    displayName: profile.display_name ?? '',
    oshiColor: normalizeOshiColor(profile.oshi_color),
    oshis: (profile.oshis ?? []).map(o => ({
      idol_id: o.idol_id,
      fan_since: o.fan_since ?? '',
    })),
  }
}

export function AccountProfilePage() {
  const qc = useQueryClient()
  const [isEditing, setIsEditing] = useState(false)
  const [form, setForm] = useState<ProfileFormState>({
    displayName: '',
    oshiColor: '',
    oshis: [],
  })
  const [saveError, setSaveError] = useState<string | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['account', 'profile'],
    queryFn: () => api.get<Profile>('/v1/account/profile'),
  })

  const save = useMutation({
    mutationFn: (payload: {
      display_name: string
      oshi_color: string
      oshis: OshiEntry[]
    }) => api.patch<Profile>('/v1/account/profile', payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['account', 'profile'] })
      setIsEditing(false)
      setSaveError(null)
    },
    onError: err => {
      setSaveError(err instanceof Error ? err.message : '保存に失敗しました')
    },
  })

  function startEdit() {
    if (!data) return
    setForm(profileToForm(data))
    setSaveError(null)
    setIsEditing(true)
  }

  function cancelEdit() {
    setIsEditing(false)
    setSaveError(null)
  }

  function updateOshi(index: number, patch: Partial<OshiEntry>) {
    setForm(prev => ({
      ...prev,
      oshis: prev.oshis.map((o, i) => (i === index ? { ...o, ...patch } : o)),
    }))
  }

  function addOshi() {
    setForm(prev =>
      prev.oshis.length >= MAX_OSHI_COUNT
        ? prev
        : { ...prev, oshis: [...prev.oshis, { idol_id: '', fan_since: '' }] },
    )
  }

  function removeOshi(index: number) {
    setForm(prev => ({
      ...prev,
      oshis: prev.oshis.filter((_, i) => i !== index),
    }))
  }

  function handleSave(e: React.FormEvent) {
    e.preventDefault()
    const trimmedName = form.displayName.trim()
    if (!trimmedName) {
      setSaveError('表示名を入力してください')
      return
    }

    const trimmedOshis = form.oshis
      .map(o => ({ idol_id: o.idol_id.trim(), fan_since: (o.fan_since ?? '').trim() }))
      .filter(o => o.idol_id !== '')

    const ids = new Set<string>()
    for (let i = 0; i < trimmedOshis.length; i++) {
      const o = trimmedOshis[i]
      if (ids.has(o.idol_id)) {
        setSaveError(`推し #${i + 1}: ID が重複しています`)
        return
      }
      ids.add(o.idol_id)
      if (o.fan_since) {
        const err = validateFanSince(o.fan_since)
        if (err) {
          setSaveError(`推し #${i + 1}: ${err}`)
          return
        }
      }
    }

    save.mutate({
      display_name: trimmedName,
      oshi_color: form.oshiColor,
      oshis: trimmedOshis,
    })
  }

  return (
    <div>
      <PageHeader title="プロフィール" description="アカウントの公開情報を管理します。" />
      <div className={styles.content}>
        {isLoading && <p className={styles.empty}>読み込み中…</p>}

        {data && !isEditing && (
          <div className={styles.profileCard}>
            {data.avatar_url && (
              <img src={data.avatar_url} alt="アバター" className={styles.avatar} />
            )}
            <div className={styles.profileInfo}>
              <div className={styles.profileField}>
                <span className={styles.profileLabel}>表示名</span>
                <span>{data.display_name ?? '—'}</span>
              </div>
              {data.email && (
                <div className={styles.profileField}>
                  <span className={styles.profileLabel}>メールアドレス</span>
                  <span>{data.email}</span>
                </div>
              )}
              <div className={styles.profileField}>
                <span className={styles.profileLabel}>推しメンカラー</span>
                {data.oshi_color ? (
                  <span className={styles.oshiColorRow}>
                    <span
                      className={styles.oshiColorChip}
                      style={{ background: data.oshi_color }}
                      aria-hidden
                    />
                    <span className={styles.oshiColorHex}>{data.oshi_color}</span>
                  </span>
                ) : (
                  <span>—</span>
                )}
              </div>
              <div className={styles.profileField}>
                <span className={styles.profileLabel}>推し</span>
                {data.oshis && data.oshis.length > 0 ? (
                  <ul className={styles.oshiList}>
                    {data.oshis.map(o => (
                      <li key={o.idol_id} className={styles.oshiListItem}>
                        <span className={styles.oshiListId}>{o.idol_id}</span>
                        {o.fan_since && (
                          <span className={styles.oshiListSince}>{o.fan_since}〜</span>
                        )}
                      </li>
                    ))}
                  </ul>
                ) : (
                  <span>—</span>
                )}
              </div>
            </div>
            <button className={styles.editBtn} onClick={startEdit}>
              編集
            </button>
          </div>
        )}

        {data && isEditing && (
          <form className={styles.profileCard} onSubmit={handleSave}>
            <div className={styles.profileInfo}>
              <div className={styles.profileField}>
                <label className={styles.profileLabel} htmlFor="display-name">
                  表示名
                </label>
                <input
                  id="display-name"
                  className={styles.input}
                  value={form.displayName}
                  onChange={e =>
                    setForm(prev => ({ ...prev, displayName: e.target.value }))
                  }
                  maxLength={50}
                  required
                  autoFocus
                />
              </div>

              <div className={styles.profileField}>
                <span className={styles.profileLabel}>推しメンカラー</span>
                <div
                  className={styles.swatches}
                  role="radiogroup"
                  aria-label="推しメンカラー"
                >
                  <button
                    type="button"
                    className={styles.swatch}
                    role="radio"
                    aria-checked={form.oshiColor === ''}
                    onClick={() => setForm(prev => ({ ...prev, oshiColor: '' }))}
                    data-selected={form.oshiColor === ''}
                    aria-label="未設定"
                  >
                    <span className={styles.swatchClear}>×</span>
                  </button>
                  {OSHI_PALETTE.map(color => (
                    <button
                      key={color}
                      type="button"
                      className={styles.swatch}
                      role="radio"
                      aria-checked={form.oshiColor === color}
                      onClick={() => setForm(prev => ({ ...prev, oshiColor: color }))}
                      data-selected={form.oshiColor === color}
                      style={{ background: color }}
                      aria-label={color}
                    />
                  ))}
                </div>
              </div>

              <div className={styles.profileField}>
                <div className={styles.oshiHeader}>
                  <span className={styles.profileLabel}>推し（最大{MAX_OSHI_COUNT}件）</span>
                  <button
                    type="button"
                    className={styles.oshiAddBtn}
                    onClick={addOshi}
                    disabled={form.oshis.length >= MAX_OSHI_COUNT}
                  >
                    + 推しを追加
                  </button>
                </div>
                {form.oshis.length === 0 && (
                  <p className={styles.oshiHint}>登録された推しはありません。</p>
                )}
                <ul className={styles.oshiEditList}>
                  {form.oshis.map((entry, index) => (
                    <li key={index} className={styles.oshiEditItem}>
                      <input
                        className={styles.input}
                        placeholder="推し ID（例: member-01）"
                        value={entry.idol_id}
                        onChange={e => updateOshi(index, { idol_id: e.target.value })}
                        maxLength={80}
                        aria-label={`推し ${index + 1} の ID`}
                      />
                      <input
                        className={styles.input}
                        placeholder="ファン歴 開始 (YYYY または YYYY-MM)"
                        value={entry.fan_since ?? ''}
                        onChange={e => updateOshi(index, { fan_since: e.target.value })}
                        maxLength={7}
                        aria-label={`推し ${index + 1} のファン歴開始`}
                      />
                      <button
                        type="button"
                        className={styles.oshiRemoveBtn}
                        onClick={() => removeOshi(index)}
                        aria-label={`推し ${index + 1} を削除`}
                      >
                        削除
                      </button>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
            {saveError && <p className={styles.formError}>{saveError}</p>}
            <div className={styles.formActions}>
              <button type="submit" className={styles.saveBtn} disabled={save.isPending}>
                {save.isPending ? '保存中…' : '保存'}
              </button>
              <button type="button" className={styles.cancelBtn} onClick={cancelEdit}>
                キャンセル
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
