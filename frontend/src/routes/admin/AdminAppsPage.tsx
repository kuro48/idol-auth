import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import type { ListResponse, App } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { statusVariant } from '@/lib/api/statusVariant'
import { appPartyTypeLabel, appRequestTypeLabel, statusLabel } from '@/lib/api/displayLabels'
import styles from './AdminListPage.module.css'

export function AdminAppsPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'apps'],
    queryFn: () => api.get<ListResponse<App>>('/v1/admin/apps'),
  })

  return (
    <div>
      <PageHeader title="アプリ一覧" description="登録されたすべてのアプリケーション" />
      <div className={styles.content}>
        {isLoading && <p className={styles.empty}>読み込み中…</p>}
        {data?.items.length === 0 && <p className={styles.empty}>アプリが見つかりません。</p>}
        <table className={styles.table}>
          <thead>
            <tr>
              <th>名前</th>
              <th>スラッグ</th>
              <th>種別</th>
              <th>区分</th>
              <th>ステータス</th>
              <th>作成日</th>
            </tr>
          </thead>
          <tbody>
            {data?.items.map(app => (
              <tr key={app.id}>
                <td className={styles.bold}>{app.name}</td>
                <td className={styles.mono}>{app.slug}</td>
                <td>{appRequestTypeLabel(app.type)}</td>
                <td>{appPartyTypeLabel(app.party_type)}</td>
                <td><Badge variant={statusVariant(app.status)}>{statusLabel(app.status)}</Badge></td>
                <td className={styles.muted}>{new Date(app.created_at).toLocaleDateString('ja-JP')}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
