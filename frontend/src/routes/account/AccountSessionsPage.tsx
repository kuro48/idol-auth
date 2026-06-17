import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { useSession } from '@/lib/auth/useSession'
import { parseUserAgent, deviceIcon } from '@/lib/userAgent'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import styles from './AccountPage.module.css'

interface SessionInfo {
  id: string
  active: boolean
  authenticated_at: string
  expires_at?: string
  created_at?: string
  device?: string
  ip_address?: string
  location?: string
  authenticator_assurance_level?: string
  methods?: string[]
}

function formatTime(value: string | undefined): string {
  if (!value) return '—'
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString('ja-JP')
}

export function AccountSessionsPage() {
  const qc = useQueryClient()
  const { session } = useSession()
  const currentSessionId = session?.session_id

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
      <PageHeader
        title="デバイス管理"
        description="このアカウントでログイン中のデバイスとブラウザの一覧です。"
      />
      <div className={styles.content}>
        {isLoading && <p className={styles.empty}>読み込み中…</p>}
        {data?.items?.length === 0 && (
          <p className={styles.empty}>アクティブなセッションはありません。</p>
        )}
        <ul className={styles.sessionList}>
          {data?.items?.map(s => {
            const ua = parseUserAgent(s.device)
            const isCurrent = s.id === currentSessionId
            return (
              <li key={s.id} className={styles.sessionItem}>
                <div className={styles.sessionIcon} aria-hidden>
                  {deviceIcon(ua.device)}
                </div>
                <div className={styles.sessionMain}>
                  <div className={styles.sessionTitleRow}>
                    <span className={styles.sessionTitle}>
                      {ua.browser} on {ua.os}
                    </span>
                    {isCurrent && <Badge variant="success">現在のセッション</Badge>}
                    {s.authenticator_assurance_level === 'aal2' && (
                      <Badge variant="info">AAL2</Badge>
                    )}
                  </div>
                  <div className={styles.sessionMeta}>
                    <span>ログイン: {formatTime(s.authenticated_at)}</span>
                    {s.expires_at && <span>期限: {formatTime(s.expires_at)}</span>}
                    {s.ip_address && <span>IP: {s.ip_address}</span>}
                    {s.location && <span>場所: {s.location}</span>}
                    {s.methods && s.methods.length > 0 && (
                      <span>方式: {s.methods.join(', ')}</span>
                    )}
                  </div>
                </div>
                <button
                  className={styles.disconnectBtn}
                  onClick={() => revoke.mutate(s.id)}
                  disabled={revoke.isPending || isCurrent}
                  title={isCurrent ? 'このセッションは現在使用中のため取り消せません' : ''}
                >
                  取り消す
                </button>
              </li>
            )
          })}
        </ul>
      </div>
    </div>
  )
}
