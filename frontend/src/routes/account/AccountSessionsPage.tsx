import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { PageHeader } from '@/components/ui/PageHeader'
import styles from './AccountPage.module.css'

interface SessionInfo {
  id: string
  active: boolean
  authenticated_at: string
  expires_at?: string
  devices?: Array<{ ip_address?: string; user_agent?: string }>
}

export function AccountSessionsPage() {
  const qc = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['account', 'sessions'],
    queryFn: () => api.get<{ items: SessionInfo[] }>('/v1/account/sessions'),
  })

  const revoke = useMutation({
    mutationFn: (sessionId: string) => api.delete(`/v1/account/sessions/${sessionId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['account', 'sessions'] }),
  })

  return (
    <div>
      <PageHeader title="Active Sessions" description="Devices and browsers currently signed in." />
      <div className={styles.content}>
        {isLoading && <p className={styles.empty}>Loading…</p>}
        {data?.items?.length === 0 && <p className={styles.empty}>No active sessions.</p>}
        <div className={styles.cards}>
          {data?.items?.map(session => (
            <div key={session.id} className={styles.appCard}>
              <div className={styles.appInfo}>
                <span className={styles.appName}>
                  {session.devices?.[0]?.user_agent?.slice(0, 60) ?? 'Unknown device'}
                </span>
                <span className={styles.appSlug}>
                  Signed in: {new Date(session.authenticated_at).toLocaleString()}
                </span>
              </div>
              <button
                className={styles.disconnectBtn}
                onClick={() => revoke.mutate(session.id)}
                disabled={revoke.isPending}
              >
                Revoke
              </button>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
