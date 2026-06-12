import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import type { ListResponse, Identity } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { statusVariant } from '@/lib/api/statusVariant'
import styles from './AdminListPage.module.css'

export function AdminUsersPage() {
  const [identifier, setIdentifier] = useState('')
  const qc = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'users', identifier],
    queryFn: () => api.get<ListResponse<Identity>>(`/v1/admin/users${identifier ? `?identifier=${encodeURIComponent(identifier)}` : ''}`),
    enabled: true,
  })

  const revokeSessions = useMutation({
    mutationFn: (ref: string) => api.post(`/v1/admin/users/${ref}/revoke-sessions`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'users'] }),
  })

  return (
    <div>
      <PageHeader title="Users" description="Search and manage user identities." />
      <div className={styles.content}>
        <div className={styles.searchRow}>
          <input
            className={styles.searchInput}
            placeholder="Search by email or identifier…"
            value={identifier}
            onChange={e => setIdentifier(e.target.value)}
          />
        </div>
        {isLoading && <p className={styles.empty}>Loading…</p>}
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Email</th>
              <th>State</th>
              <th>Roles</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {data?.items.map(user => (
              <tr key={user.id}>
                <td className={styles.bold}>{user.email || user.id}</td>
                <td><Badge variant={statusVariant(user.state)}>{user.state}</Badge></td>
                <td className={styles.muted}>{user.roles?.join(', ') || '—'}</td>
                <td>
                  <button
                    className={styles.actionBtn}
                    onClick={() => revokeSessions.mutate(user.id)}
                    disabled={revokeSessions.isPending}
                  >
                    Revoke Sessions
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
