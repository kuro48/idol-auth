import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import type { ListResponse, App } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { statusVariant } from '@/lib/api/statusVariant'
import styles from './AdminListPage.module.css'

export function AdminAppsPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'apps'],
    queryFn: () => api.get<ListResponse<App>>('/v1/admin/apps'),
  })

  return (
    <div>
      <PageHeader title="Apps" description="All registered applications." />
      <div className={styles.content}>
        {isLoading && <p className={styles.empty}>Loading…</p>}
        {data?.items.length === 0 && <p className={styles.empty}>No apps found.</p>}
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Name</th>
              <th>Slug</th>
              <th>Type</th>
              <th>Party</th>
              <th>Status</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            {data?.items.map(app => (
              <tr key={app.id}>
                <td className={styles.bold}>{app.name}</td>
                <td className={styles.mono}>{app.slug}</td>
                <td>{app.type}</td>
                <td>{app.party_type}</td>
                <td><Badge variant={statusVariant(app.status)}>{app.status}</Badge></td>
                <td className={styles.muted}>{new Date(app.created_at).toLocaleDateString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
