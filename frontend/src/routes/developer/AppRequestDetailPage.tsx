import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useParams, useNavigate } from '@tanstack/react-router'
import { api } from '@/lib/api/client'
import type { AppRequest } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { statusVariant } from '@/lib/api/statusVariant'
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

  if (isLoading) return <div className={styles.loading}>Loading…</div>
  if (!req) return <div className={styles.loading}>Not found</div>

  return (
    <div>
      <PageHeader
        title={req.name}
        description={req.description}
        action={<Badge variant={statusVariant(req.status)}>{req.status.replace('_', ' ')}</Badge>}
      />
      <div className={styles.content}>
        <dl className={styles.grid}>
          <Field label="Type" value={req.type} />
          <Field label="Slug" value={req.slug} mono />
          <Field label="Contact" value={req.contact_email} />
          <Field label="Organization" value={req.organization} />
          <Field label="Homepage" value={req.homepage_url} />
          <Field label="Purpose" value={req.purpose} />
          <Field label="Redirect URIs" value={req.redirect_uris?.join('\n')} mono />
          <Field label="Scopes" value={req.scopes?.join(', ')} mono />
          {req.reviewer_note && <Field label="Reviewer Note" value={req.reviewer_note} />}
        </dl>
        {(req.status === 'pending' || req.status === 'changes_requested') && (
          <div className={styles.actions}>
            <button
              className={styles.dangerBtn}
              onClick={() => withdraw.mutate()}
              disabled={withdraw.isPending}
            >
              {withdraw.isPending ? 'Withdrawing…' : 'Withdraw Request'}
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
