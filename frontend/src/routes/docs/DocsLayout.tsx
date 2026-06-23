import { Outlet, useRouterState } from '@tanstack/react-router'
import styles from './DocsLayout.module.css'

const NAV = [
  { to: '/docs', label: '概要' },
  { to: '/docs/start', label: 'はじめに' },
  { to: '/docs/concepts', label: '基本概念' },
  { to: '/docs/sdk', label: 'SDK' },
  { to: '/docs/management', label: 'ユーザー管理' },
  { to: '/docs/account', label: 'アカウントセンター' },
  { to: '/docs/security', label: 'セキュリティ' },
  { to: '/docs/api', label: 'API リファレンス' },
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
        <span className={styles.sidebarTitle}>ドキュメント</span>
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
