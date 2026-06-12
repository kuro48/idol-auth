import styles from './Badge.module.css'

export type BadgeVariant = 'default' | 'success' | 'warning' | 'danger' | 'info'

interface BadgeProps {
  variant?: BadgeVariant
  children: React.ReactNode
}

export function Badge({ variant = 'default', children }: BadgeProps) {
  return <span className={`${styles.badge} ${styles[variant]}`}>{children}</span>
}
