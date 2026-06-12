import { useQuery } from '@tanstack/react-query'
import { api, ApiError } from '@/lib/api/client'

export interface Session {
  identityId: string
  email: string
  roles: string[]
}

export function useSession() {
  const { data, isLoading, error } = useQuery<Session | null>({
    queryKey: ['session'],
    queryFn: async () => {
      try {
        return await api.get<Session>('/v1/auth/session')
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) {
          return null
        }
        throw err
      }
    },
    staleTime: 5 * 60 * 1000,
    retry: false,
  })

  return {
    session: data ?? null,
    isLoading,
    isAuthenticated: data != null,
    isAdmin: data?.roles.includes('admin') ?? false,
    error,
  }
}
