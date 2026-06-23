import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import type { ListResponse, Identity } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { statusVariant } from '@/lib/api/statusVariant'
import { roleLabel, statusLabel } from '@/lib/api/displayLabels'
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
      <PageHeader title="ユーザー" description="ユーザー ID を検索・管理します。" />
      <div className={styles.content}>
        <div className={styles.searchRow}>
          <input
            className={styles.searchInput}
            placeholder="メールアドレスまたは ID で検索…"
            value={identifier}
            onChange={e => setIdentifier(e.target.value)}
          />
        </div>
        {isLoading && <p className={styles.empty}>読み込み中…</p>}
        <table className={styles.table}>
          <thead>
            <tr>
              <th>メール</th>
              <th>状態</th>
              <th>ロール</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {data?.items.map(user => (
              <tr key={user.id}>
                <td className={styles.bold}>{user.email || user.id}</td>
                <td><Badge variant={statusVariant(user.state)}>{statusLabel(user.state)}</Badge></td>
                <td className={styles.muted}>{user.roles?.map(roleLabel).join(', ') || '—'}</td>
                <td>
                  <button
                    className={styles.actionBtn}
                    onClick={() => revokeSessions.mutate(user.id)}
                    disabled={revokeSessions.isPending}
                  >
                    セッションを取り消す
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
