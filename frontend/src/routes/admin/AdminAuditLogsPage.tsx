import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import type { ListResponse, AuditLog } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import styles from './AdminListPage.module.css'

export function AdminAuditLogsPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'audit-logs'],
    queryFn: () => api.get<ListResponse<AuditLog>>('/v1/admin/audit-logs?limit=100'),
  })

  return (
    <div>
      <PageHeader title="Audit Logs" description="Recent audit events." />
      <div className={styles.content}>
        {isLoading && <p className={styles.empty}>Loading…</p>}
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Event</th>
              <th>Actor</th>
              <th>Target</th>
              <th>Result</th>
              <th>Time</th>
            </tr>
          </thead>
          <tbody>
            {data?.items.map(log => (
              <tr key={log.ID}>
                <td className={styles.mono}>{log.EventType}</td>
                <td className={styles.muted}>{log.ActorType}:{log.ActorID.slice(0, 8)}</td>
                <td className={styles.muted}>{log.TargetType}:{log.TargetID.slice(0, 8)}</td>
                <td><Badge variant={log.Result === 'success' ? 'success' : 'danger'}>{log.Result}</Badge></td>
                <td className={styles.muted}>{new Date(log.OccurredAt).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
