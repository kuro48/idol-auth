import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'

interface Providers {
  settings_url: string
}

export function SettingsRedirectPage() {
  const { data } = useQuery({
    queryKey: ['auth', 'providers'],
    queryFn: () => api.get<Providers>('/v1/auth/providers'),
  })

  useEffect(() => {
    if (data?.settings_url) {
      window.location.href = data.settings_url
    }
  }, [data])

  return <p style={{ padding: '2rem' }}>Redirecting to settings…</p>
}
