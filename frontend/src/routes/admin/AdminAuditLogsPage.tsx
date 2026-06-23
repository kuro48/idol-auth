import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import type { ListResponse, AuditLog } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { auditResultLabel } from '@/lib/api/displayLabels'
import styles from './AdminListPage.module.css'

export function AdminAuditLogsPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'audit-logs'],
    queryFn: () => api.get<ListResponse<AuditLog>>('/v1/admin/audit-logs?limit=100'),
  })

  return (
    <div>
      <PageHeader title="監査ログ" description="最近の監査イベント" />
      <div className={styles.content}>
        {isLoading && <p className={styles.empty}>読み込み中…</p>}
        <table className={styles.table}>
          <thead>
            <tr>
              <th>イベント</th>
              <th>実行者</th>
              <th>対象</th>
              <th>結果</th>
              <th>日時</th>
            </tr>
          </thead>
          <tbody>
            {data?.items?.map(log => (
              <tr key={log.id}>
                <td className={styles.mono}>{log.event_type}</td>
                <td className={styles.muted}>{log.actor_type}:{log.actor_id.slice(0, 8)}</td>
                <td className={styles.muted}>{log.target_type}:{log.target_id.slice(0, 8)}</td>
                <td><Badge variant={log.result === 'success' ? 'success' : 'danger'}>{auditResultLabel(log.result)}</Badge></td>
                <td className={styles.muted}>{new Date(log.occurred_at).toLocaleString('ja-JP')}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
