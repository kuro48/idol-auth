import { Link } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import type { AppMembership, DeletionRequest } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { statusVariant } from '@/lib/api/statusVariant'
import { statusLabel } from '@/lib/api/displayLabels'
import { useSession } from '@/lib/auth/useSession'
import styles from './AccountPage.module.css'

export function AccountOverviewPage() {
  const qc = useQueryClient()
  const { isDeveloper } = useSession()

  const { data, isLoading } = useQuery({
    queryKey: ['account', 'overview'],
    queryFn: () => api.get<{ memberships: AppMembership[] }>('/v1/account/'),
  })

  const { data: deletionData } = useQuery({
    queryKey: ['account', 'deletion'],
    queryFn: () => api.get<{ request: DeletionRequest | null }>('/v1/account/deletion'),
  })

  const disconnect = useMutation({
    mutationFn: (appId: string) => api.delete(`/v1/account/apps/${appId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['account', 'overview'] }),
  })

  const scheduleDeletion = useMutation({
    mutationFn: () => api.post('/v1/account/deletion', { reason: '' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['account', 'deletion'] }),
  })

  const cancelDeletion = useMutation({
    mutationFn: () => api.delete('/v1/account/deletion'),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['account', 'deletion'] }),
  })

  const deletion = deletionData?.request
  const isDeletionScheduled = deletion?.status === 'scheduled'

  return (
    <div>
      <PageHeader title="アカウント概要" description="連携アプリとアカウント管理" />
      <div className={styles.content}>
        {isDeveloper ? (
          <div className={styles.devBanner}>
            <span className={styles.devBannerText}>開発者として登録済みです。</span>
            <Link to="/developer/app-requests" className={styles.devBannerLink}>開発者ポータル →</Link>
          </div>
        ) : (
          <div className={styles.devBanner}>
            <span className={styles.devBannerText}>アプリ開発者の方はこちら</span>
            <Link to="/account/developer" className={styles.devBannerLink}>開発者登録 →</Link>
          </div>
        )}

        {isLoading && <p className={styles.empty}>読み込み中…</p>}
        {data?.memberships?.length === 0 && (
          <p className={styles.empty}>連携済みのアプリはありません。</p>
        )}
        <div className={styles.cards}>
          {data?.memberships?.map(m => (
            <div key={m.id} className={styles.appCard}>
              <div className={styles.appInfo}>
                <span className={styles.appName}>{m.app_name}</span>
                <span className={styles.appSlug}>{m.app_slug}</span>
              </div>
              <div className={styles.appMeta}>
                <Badge variant={statusVariant(m.status)}>{statusLabel(m.status)}</Badge>
                <button
                  className={styles.disconnectBtn}
                  onClick={() => disconnect.mutate(m.app_id)}
                  disabled={disconnect.isPending}
                >
                  連携解除
                </button>
              </div>
            </div>
          ))}
        </div>

        <div className={styles.dangerSection}>
          <h2 className={styles.dangerTitle}>危険ゾーン</h2>
          {isDeletionScheduled && deletion ? (
            <>
              <div className={styles.deletionStatus}>
                <span className={styles.deletionDate}>
                  削除予定日: {new Date(deletion.scheduled_for).toLocaleDateString('ja-JP')}
                </span>
                <span className={styles.dangerText}>
                  期日までにキャンセルしなければアカウントと全データが完全に削除されます。
                </span>
              </div>
              <button
                className={styles.dangerBtn}
                onClick={() => cancelDeletion.mutate()}
                disabled={cancelDeletion.isPending}
              >
                {cancelDeletion.isPending ? '処理中…' : '削除をキャンセル'}
              </button>
            </>
          ) : (
            <>
              <p className={styles.dangerText}>
                アカウントを削除すると、すべての連携・データが完全に削除されます。リクエスト後30日間の猶予期間があり、その間はキャンセル可能です。猶予期間後に確定すると元に戻せません。
              </p>
              <button
                className={styles.dangerBtn}
                onClick={() => {
                  if (window.confirm('本当にアカウントを削除しますか？30日後に確定します。猶予期間中はキャンセル可能です。')) {
                    scheduleDeletion.mutate()
                  }
                }}
                disabled={scheduleDeletion.isPending}
              >
                {scheduleDeletion.isPending ? '処理中…' : 'アカウントを削除'}
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
