import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import type { ListResponse, AppRequest } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { statusVariant } from '@/lib/api/statusVariant'
import styles from './AdminListPage.module.css'

export function AdminAppRequestsPage() {
  const qc = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'app-requests'],
    queryFn: () => api.get<ListResponse<AppRequest>>('/v1/admin/app-requests'),
  })

  const approve = useMutation({
    mutationFn: (id: string) => api.post(`/v1/admin/app-requests/${id}/approve`, { party_type: 'third_party' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'app-requests'] }),
  })

  const reject = useMutation({
    mutationFn: (id: string) => api.post(`/v1/admin/app-requests/${id}/reject`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'app-requests'] }),
  })

  return (
    <div>
      <PageHeader title="App Requests" description="Review developer application registration requests." />
      <div className={styles.content}>
        {isLoading && <p className={styles.empty}>Loading…</p>}
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Name</th>
              <th>Type</th>
              <th>Status</th>
              <th>Submitted</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {data?.items.map(req => (
              <tr key={req.id}>
                <td className={styles.bold}>{req.name}</td>
                <td>{req.type}</td>
                <td><Badge variant={statusVariant(req.status)}>{req.status.replace('_', ' ')}</Badge></td>
                <td className={styles.muted}>{new Date(req.created_at).toLocaleDateString()}</td>
                <td className={styles.actions}>
                  {req.status === 'pending' && (
                    <>
                      <button className={styles.approveBtn} onClick={() => approve.mutate(req.id)} disabled={approve.isPending}>Approve</button>
                      <button className={styles.rejectBtn} onClick={() => reject.mutate(req.id)} disabled={reject.isPending}>Reject</button>
                    </>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
