import { Outlet, Link, useRouterState } from '@tanstack/react-router'
import { useSession } from '@/lib/auth/useSession'
import { useAdminIdentity } from '@/lib/auth/useAdminIdentity'
import styles from './AppShell.module.css'

interface NavItem {
  to: string
  label: string
}

interface NavSection {
  title: string
  items: NavItem[]
}

const ADMIN_NAV_ITEMS: NavItem[] = [
  { to: '/admin/apps', label: 'Apps' },
  { to: '/admin/users', label: 'Users' },
  { to: '/admin/app-requests', label: 'App Requests' },
  { to: '/admin/audit-logs', label: 'Audit Logs' },
]

function buildSections(isDeveloper: boolean, isAdminHost: boolean): NavSection[] {
  if (isAdminHost) {
    return [{ title: 'Admin', items: ADMIN_NAV_ITEMS }]
  }

  const sections: NavSection[] = [
    {
      title: 'Account',
      items: [
        { to: '/account', label: 'Overview' },
        { to: '/account/profile', label: 'Profile' },
        { to: '/account/sessions', label: 'Sessions' },
        { to: '/account/login-history', label: 'Login History' },
        { to: '/account/security', label: 'Security' },
        { to: '/account/privacy', label: 'Privacy' },
      ],
    },
  ]

  if (isDeveloper) {
    sections.unshift({
      title: 'Developer',
      items: [
        { to: '/developer/app-requests', label: 'App Requests' },
      ],
    })
  }

  return sections
}

function NavLink({ to, label }: NavItem) {
  const state = useRouterState()
  const isActive = state.location.pathname.startsWith(to) && (to !== '/' || state.location.pathname === '/')
  return (
    <Link to={to} className={`${styles.navLink} ${isActive ? styles.active : ''}`}>
      {label}
    </Link>
  )
}

export function AppShell() {
  const { session, isDeveloper } = useSession()
  const { data: adminIdentity } = useAdminIdentity()
  const isAdminHost = window.location.hostname.startsWith('admin.')
  const sections = buildSections(isDeveloper, isAdminHost)

  const footerEmail = isAdminHost ? (adminIdentity?.email ?? '') : (session?.email ?? '')
  const logoutHref = isAdminHost
    ? (adminIdentity?.logout_url || '/v1/auth/logout')
    : '/v1/auth/logout'

  return (
    <div className={styles.shell}>
      <aside className={styles.sidebar}>
        <div className={styles.logo}>
          <Link to="/" className={styles.logoLink}>idol-auth</Link>
        </div>
        <nav className={styles.nav}>
          {sections.map(section => (
            <div key={section.title} className={styles.section}>
              <span className={styles.sectionLabel}>{section.title}</span>
              {section.items.map(item => (
                <NavLink key={item.to} {...item} />
              ))}
            </div>
          ))}
        </nav>
        {(isAdminHost ? footerEmail : session) && (
          <div className={styles.footer}>
            <span className={styles.userEmail}>{footerEmail}</span>
            <a href={logoutHref} className={styles.logoutLink}>Sign out</a>
          </div>
        )}
      </aside>
      <main className={styles.main}>
        <Outlet />
      </main>
    </div>
  )
}
