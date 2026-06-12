import { Outlet } from '@tanstack/react-router'
import styles from './RootLayout.module.css'

export function RootLayout() {
  return (
    <div className={styles.root}>
      <Outlet />
    </div>
  )
}
