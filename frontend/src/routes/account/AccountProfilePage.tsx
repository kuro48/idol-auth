import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { PageHeader } from '@/components/ui/PageHeader'
import styles from './AccountPage.module.css'

interface Profile {
  identity_id: string
  display_name?: string
  oshi_color?: string
  avatar_url?: string
  bio?: string
}

interface Providers {
  settings_url: string
}

export function AccountProfilePage() {
  const { data, isLoading } = useQuery({
    queryKey: ['account', 'profile'],
    queryFn: () => api.get<Profile>('/v1/account/profile'),
  })

  const { data: providers } = useQuery({
    queryKey: ['auth', 'providers'],
    queryFn: () => api.get<Providers>('/v1/auth/providers'),
  })

  return (
    <div>
      <PageHeader title="Profile" description="Your public profile information." />
      <div className={styles.content}>
        {isLoading && <p className={styles.empty}>Loading…</p>}
        {data && (
          <div className={styles.profileCard}>
            {data.avatar_url && (
              <img src={data.avatar_url} alt="Avatar" className={styles.avatar} />
            )}
            <div className={styles.profileInfo}>
              <div className={styles.profileField}>
                <span className={styles.profileLabel}>Display Name</span>
                <span>{data.display_name ?? '—'}</span>
              </div>
            </div>
            {providers?.settings_url && (
              <a href={providers.settings_url} className={styles.editBtn}>
                Edit Profile &amp; Password
              </a>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
