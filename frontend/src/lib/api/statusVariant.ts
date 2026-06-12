import type { BadgeVariant } from '@/components/ui/Badge'

export function statusVariant(status: string): BadgeVariant {
  switch (status.toLowerCase()) {
    case 'approved': return 'success'
    case 'pending': case 'under_review': return 'warning'
    case 'rejected': case 'withdrawn': return 'danger'
    case 'changes_requested': return 'info'
    case 'active': return 'success'
    case 'inactive': case 'revoked': return 'danger'
    default: return 'default'
  }
}
