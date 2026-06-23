import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useParams, useNavigate } from '@tanstack/react-router'
import { api } from '@/lib/api/client'
import type { AppRequest } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { statusVariant } from '@/lib/api/statusVariant'
import { appRequestTypeLabel, statusLabel } from '@/lib/api/displayLabels'
import styles from './AppRequestDetailPage.module.css'

export function AppRequestDetailPage() {
  const { id } = useParams({ strict: false }) as { id: string }
  const navigate = useNavigate()
  const qc = useQueryClient()

  const { data: req, isLoading } = useQuery({
    queryKey: ['developer', 'app-requests', id],
    queryFn: () => api.get<AppRequest>(`/v1/developer/app-requests/${id}`),
  })

  const withdraw = useMutation({
    mutationFn: () => api.post(`/v1/developer/app-requests/${id}/withdraw`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['developer', 'app-requests'] })
      navigate({ to: '/developer/app-requests' })
    },
  })

  if (isLoading) return <div className={styles.loading}>読み込み中…</div>
  if (!req) return <div className={styles.loading}>見つかりません</div>

  return (
    <div>
      <PageHeader
        title={req.name}
        description={req.description}
        action={<Badge variant={statusVariant(req.status)}>{statusLabel(req.status)}</Badge>}
      />
      <div className={styles.content}>
        <dl className={styles.grid}>
          <Field label="種別" value={appRequestTypeLabel(req.type)} />
          <Field label="スラッグ" value={req.slug} mono />
          <Field label="連絡先" value={req.contact_email} />
          <Field label="組織名" value={req.organization} />
          <Field label="ホームページ" value={req.homepage_url} />
          <Field label="利用目的" value={req.purpose} />
          <Field label="リダイレクト URI" value={req.redirect_uris?.join('\n')} mono />
          <Field label="スコープ" value={req.scopes?.join(', ')} mono />
          {req.reviewer_note && <Field label="審査コメント" value={req.reviewer_note} />}
        </dl>
        {(req.status === 'pending' || req.status === 'changes_requested') && (
          <div className={styles.actions}>
            <button
              className={styles.dangerBtn}
              onClick={() => withdraw.mutate()}
              disabled={withdraw.isPending}
            >
              {withdraw.isPending ? '取り下げ中…' : '申請を取り下げる'}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

function Field({ label, value, mono }: { label: string; value?: string; mono?: boolean }) {
  if (!value) return null
  return (
    <>
      <dt className={styles.dt}>{label}</dt>
      <dd className={`${styles.dd} ${mono ? styles.mono : ''}`}>{value}</dd>
    </>
  )
}
