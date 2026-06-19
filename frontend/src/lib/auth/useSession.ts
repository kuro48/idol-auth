import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import type { SessionView } from '@/lib/api/types'

export interface Session {
  identityId: string
  email: string
  roles: string[]
  oshiColor: string
  sessionId: string
  session_id: string
  methods: string[]
  authenticatorAssuranceLevel: string
}

function toSession(view: SessionView): Session | null {
  if (!view.authenticated) return null
  const sessionId = view.session_id ?? ''
  return {
    identityId: view.identity_id ?? '',
    email: view.email ?? '',
    roles: view.roles ?? [],
    oshiColor: view.oshi_color ?? '',
    sessionId,
    session_id: sessionId,
    methods: view.methods ?? [],
    authenticatorAssuranceLevel: view.authenticator_assurance_level ?? '',
  }
}

export function useSession() {
  const { data, isLoading, error } = useQuery<Session | null>({
    queryKey: ['session'],
    queryFn: async () => {
      const view = await api.get<SessionView>('/v1/auth/session')
      return toSession(view)
    },
    staleTime: 5 * 60 * 1000,
    retry: false,
  })

  const roles = data?.roles ?? []
  return {
    session: data ?? null,
    isLoading,
    isAuthenticated: data != null,
    isDeveloper: roles.includes('developer'),
    error,
  }
}
