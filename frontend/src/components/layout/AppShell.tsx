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
  { to: '/admin/apps', label: 'アプリ' },
  { to: '/admin/users', label: 'ユーザー' },
  { to: '/admin/app-requests', label: 'アプリ申請' },
  { to: '/admin/audit-logs', label: '監査ログ' },
]

function buildSections(isDeveloper: boolean, isAdminHost: boolean): NavSection[] {
  if (isAdminHost) {
    return [{ title: '管理', items: ADMIN_NAV_ITEMS }]
  }

  const sections: NavSection[] = [
    {
      title: 'アカウント',
      items: [
        { to: '/account', label: '概要' },
        { to: '/account/profile', label: 'プロフィール' },
        { to: '/account/sessions', label: 'デバイス管理' },
        { to: '/account/login-history', label: 'ログイン履歴' },
        { to: '/account/security', label: 'セキュリティ' },
        { to: '/account/privacy', label: 'プライバシー' },
      ],
    },
  ]

  if (isDeveloper) {
    sections.unshift({
      title: '開発者',
      items: [
        { to: '/developer/app-requests', label: 'アプリ申請' },
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
            <a href={logoutHref} className={styles.logoutLink}>ログアウト</a>
          </div>
        )}
      </aside>
      <main className={styles.main}>
        <Outlet />
      </main>
    </div>
  )
}
