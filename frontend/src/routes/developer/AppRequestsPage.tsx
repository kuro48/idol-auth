import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { api } from '@/lib/api/client'
import type { ListResponse, AppRequest } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { statusVariant } from '@/lib/api/statusVariant'
import styles from './AppRequestsPage.module.css'

export function AppRequestsPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['developer', 'app-requests'],
    queryFn: () => api.get<ListResponse<AppRequest>>('/v1/developer/app-requests'),
  })

  return (
    <div>
      <PageHeader
        title="App Requests"
        description="Manage your application registration requests."
        action={
          <Link to="/developer/app-requests/new" className={styles.primaryBtn}>
            New Request
          </Link>
        }
      />
      <div className={styles.content}>
        {isLoading && <p className={styles.empty}>Loading…</p>}
        {!isLoading && data?.items.length === 0 && (
          <p className={styles.empty}>No app requests yet. Create one to get started.</p>
        )}
        {data?.items.map(req => (
          <a key={req.ID} href={`/developer/app-requests/${req.ID}`} className={styles.card}>
            <div className={styles.cardMain}>
              <span className={styles.cardName}>{req.Name}</span>
              <span className={styles.cardSlug}>{req.Slug}</span>
            </div>
            <Badge variant={statusVariant(req.Status)}>{req.Status.replace('_', ' ')}</Badge>
          </a>
        ))}
      </div>
    </div>
  )
}
