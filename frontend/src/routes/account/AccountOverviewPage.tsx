import { Link } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import type { AppMembership } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { statusVariant } from '@/lib/api/statusVariant'
import { useSession } from '@/lib/auth/useSession'
import styles from './AccountPage.module.css'

export function AccountOverviewPage() {
  const qc = useQueryClient()
  const { isDeveloper } = useSession()

  const { data, isLoading } = useQuery({
    queryKey: ['account', 'overview'],
    queryFn: () => api.get<{ memberships: AppMembership[] }>('/v1/account/'),
  })

  const disconnect = useMutation({
    mutationFn: (appId: string) => api.delete(`/v1/account/apps/${appId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['account', 'overview'] }),
  })

  return (
    <div>
      <PageHeader title="Account Overview" description="Apps connected to your account." />
      <div className={styles.content}>
        {isDeveloper ? (
          <div className={styles.devBanner}>
            <span className={styles.devBannerText}>開発者として登録済みです。</span>
            <Link to="/developer/app-requests" className={styles.devBannerLink}>Developer Portal →</Link>
          </div>
        ) : (
          <div className={styles.devBanner}>
            <span className={styles.devBannerText}>アプリ開発者の方はこちら</span>
            <Link to="/account/developer" className={styles.devBannerLink}>開発者登録 →</Link>
          </div>
        )}
        {isLoading && <p className={styles.empty}>Loading…</p>}
        {data?.memberships?.length === 0 && <p className={styles.empty}>No connected apps.</p>}
        <div className={styles.cards}>
          {data?.memberships?.map(m => (
            <div key={m.id} className={styles.appCard}>
              <div className={styles.appInfo}>
                <span className={styles.appName}>{m.app_name}</span>
                <span className={styles.appSlug}>{m.app_slug}</span>
              </div>
              <div className={styles.appMeta}>
                <Badge variant={statusVariant(m.status)}>{m.status}</Badge>
                <button
                  className={styles.disconnectBtn}
                  onClick={() => disconnect.mutate(m.app_id)}
                  disabled={disconnect.isPending}
                >
                  Disconnect
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
