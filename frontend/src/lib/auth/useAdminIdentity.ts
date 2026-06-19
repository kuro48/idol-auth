import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'

export interface AdminIdentity {
  email: string
  auth_method: string
  logout_url: string
}

export function useAdminIdentity() {
  return useQuery<AdminIdentity | null>({
    queryKey: ['admin', 'me'],
    queryFn: () => api.get<AdminIdentity>('/v1/admin/me'),
    retry: false,
    enabled: window.location.hostname.startsWith('admin.'),
  })
}
