import { Outlet, useRouterState } from '@tanstack/react-router'
import styles from './DocsLayout.module.css'

const NAV = [
  { to: '/docs', label: 'Overview' },
  { to: '/docs/start', label: 'Getting Started' },
  { to: '/docs/concepts', label: 'Core Concepts' },
  { to: '/docs/sdk', label: 'SDK' },
  { to: '/docs/management', label: 'User Management' },
  { to: '/docs/account', label: 'Account Center' },
  { to: '/docs/security', label: 'Security' },
  { to: '/docs/api', label: 'API Reference' },
]

function DocNavLink({ to, label }: { to: string; label: string }) {
  const state = useRouterState()
  const isActive = state.location.pathname === to || (to !== '/docs' && state.location.pathname.startsWith(to))
  return (
    <a href={to} className={`${styles.navLink} ${isActive ? styles.active : ''}`}>{label}</a>
  )
}

export function DocsLayout() {
  return (
    <div className={styles.layout}>
      <nav className={styles.sidebar}>
        <span className={styles.sidebarTitle}>Documentation</span>
        <div className={styles.links}>
          {NAV.map(item => <DocNavLink key={item.to} {...item} />)}
        </div>
      </nav>
      <main className={styles.content}>
        <Outlet />
      </main>
    </div>
  )
}
