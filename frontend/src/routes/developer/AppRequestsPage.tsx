import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { api } from '@/lib/api/client'
import type { ListResponse, AppRequest } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { statusVariant } from '@/lib/api/statusVariant'
import { statusLabel } from '@/lib/api/displayLabels'
import styles from './AppRequestsPage.module.css'

export function AppRequestsPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['developer', 'app-requests'],
    queryFn: () => api.get<ListResponse<AppRequest>>('/v1/developer/app-requests'),
  })

  return (
    <div>
      <PageHeader
        title="アプリ申請一覧"
        description="アプリの登録申請を管理します。"
        action={
          <Link to="/developer/app-requests/new" className={styles.primaryBtn}>
            新規申請
          </Link>
        }
      />
      <div className={styles.content}>
        {isLoading && <p className={styles.empty}>読み込み中…</p>}
        {!isLoading && data?.items.length === 0 && (
          <p className={styles.empty}>申請はまだありません。</p>
        )}
        {data?.items.map(req => (
          <a key={req.id} href={`/developer/app-requests/${req.id}`} className={styles.card}>
            <div className={styles.cardMain}>
              <span className={styles.cardName}>{req.name}</span>
              <span className={styles.cardSlug}>{req.slug}</span>
            </div>
            <Badge variant={statusVariant(req.status)}>{statusLabel(req.status)}</Badge>
          </a>
        ))}
      </div>
    </div>
  )
}
