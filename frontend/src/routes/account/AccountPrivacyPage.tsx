import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { PageHeader } from '@/components/ui/PageHeader'
import styles from './AccountPage.module.css'

interface DataPreferences {
  analytics_consent: boolean
  personalization: boolean
  third_party_sharing: boolean
}

export function AccountPrivacyPage() {
  const qc = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['account', 'data-preferences'],
    queryFn: () => api.get<DataPreferences>('/v1/account/data-preferences'),
  })

  const save = useMutation({
    mutationFn: (payload: DataPreferences) =>
      api.patch<DataPreferences>('/v1/account/data-preferences', payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['account', 'data-preferences'] })
    },
  })

  function toggle(field: keyof DataPreferences) {
    if (!data) return
    save.mutate({ ...data, [field]: !data[field] })
  }

  const items: { field: keyof DataPreferences; label: string; description: string }[] = [
    {
      field: 'analytics_consent',
      label: 'アクセス解析',
      description: 'サービス改善のために匿名の利用データを収集します。',
    },
    {
      field: 'personalization',
      label: 'パーソナライズ',
      description: 'あなたの利用履歴に基づいてコンテンツをカスタマイズします。',
    },
    {
      field: 'third_party_sharing',
      label: '第三者へのデータ提供',
      description: '連携アプリに匿名統計データを提供します。',
    },
  ]

  return (
    <div>
      <PageHeader
        title="データ利用設定"
        description="データの収集・利用に関するオプトイン設定を管理します。"
      />
      <div className={styles.content}>
        {isLoading && <p className={styles.empty}>読み込み中…</p>}
        {data && (
          <div className={styles.profileCard}>
            {items.map(({ field, label, description }) => (
              <div key={field} className={styles.profileField} style={{ alignItems: 'center' }}>
                <div>
                  <span className={styles.profileLabel}>{label}</span>
                  <p style={{ fontSize: '0.85rem', color: 'var(--color-text-muted, #888)', margin: 0 }}>
                    {description}
                  </p>
                </div>
                <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', cursor: 'pointer' }}>
                  <input
                    type="checkbox"
                    checked={data[field]}
                    onChange={() => toggle(field)}
                    disabled={save.isPending}
                    aria-label={label}
                  />
                  <span style={{ fontSize: '0.875rem' }}>{data[field] ? '有効' : '無効'}</span>
                </label>
              </div>
            ))}
            {save.isError && (
              <p className={styles.formError}>保存に失敗しました。</p>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
