import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import styles from './AccountPage.module.css'

interface LoginEvent {
  id: string
  session_id: string
  authenticated_at: string
  issued_at: string
  aal: string
  methods: string[]
  ip_address?: string
  user_agent?: string
  recorded_at: string
}

function formatMethods(methods: string[]): string {
  if (methods.length === 0) return '—'
  return methods.join(', ')
}

function shortUA(ua: string | undefined): string {
  if (!ua) return 'Unknown device'
  return ua.length > 80 ? ua.slice(0, 80) + '…' : ua
}

export function AccountLoginHistoryPage() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['account', 'login-history'],
    queryFn: () =>
      api.get<{ items: LoginEvent[] }>('/v1/account/login-history?limit=100'),
  })

  return (
    <div>
      <PageHeader
        title="ログイン履歴"
        description="このアカウントで観測された認証済みセッションの一覧です。"
      />
      <div className={styles.content}>
        {isLoading && <p className={styles.empty}>読み込み中…</p>}
        {error && (
          <p className={styles.formError}>
            ログイン履歴の取得に失敗しました。
          </p>
        )}
        {data?.items?.length === 0 && (
          <p className={styles.empty}>履歴はまだありません。</p>
        )}
        <ul className={styles.loginEventList}>
          {data?.items?.map(event => (
            <li key={event.id} className={styles.loginEventItem}>
              <div className={styles.loginEventHeader}>
                <span className={styles.loginEventTime}>
                  {new Date(event.authenticated_at).toLocaleString('ja-JP')}
                </span>
                {event.aal && (
                  <Badge variant={event.aal === 'aal2' ? 'success' : 'default'}>
                    {event.aal.toUpperCase()}
                  </Badge>
                )}
              </div>
              <div className={styles.loginEventDevice}>{shortUA(event.user_agent)}</div>
              <div className={styles.loginEventMeta}>
                <span>方式: {formatMethods(event.methods)}</span>
                {event.ip_address && <span>IP: {event.ip_address}</span>}
              </div>
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}
