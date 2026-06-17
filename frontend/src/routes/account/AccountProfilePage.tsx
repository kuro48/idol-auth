import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { PageHeader } from '@/components/ui/PageHeader'
import styles from './AccountPage.module.css'

interface Profile {
  identity_id: string
  display_name?: string
  oshi_color?: string
  avatar_url?: string
  email?: string
}

export function AccountProfilePage() {
  const qc = useQueryClient()
  const [isEditing, setIsEditing] = useState(false)
  const [displayName, setDisplayName] = useState('')
  const [saveError, setSaveError] = useState<string | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['account', 'profile'],
    queryFn: () => api.get<Profile>('/v1/account/profile'),
  })

  const save = useMutation({
    mutationFn: (name: string) =>
      api.patch<Profile>('/v1/account/profile', { display_name: name }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['account', 'profile'] })
      setIsEditing(false)
      setSaveError(null)
    },
    onError: (err) => {
      setSaveError(err instanceof Error ? err.message : '保存に失敗しました')
    },
  })

  function startEdit() {
    setDisplayName(data?.display_name ?? '')
    setSaveError(null)
    setIsEditing(true)
  }

  function cancelEdit() {
    setIsEditing(false)
    setSaveError(null)
  }

  function handleSave(e: React.FormEvent) {
    e.preventDefault()
    const trimmed = displayName.trim()
    if (!trimmed) return
    save.mutate(trimmed)
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
                  value={displayName}
                  onChange={e => setDisplayName(e.target.value)}
                  maxLength={50}
                  required
                  autoFocus
                />
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
