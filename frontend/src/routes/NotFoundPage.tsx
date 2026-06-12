import styles from './NotFoundPage.module.css'

export function NotFoundPage() {
  return (
    <main className={styles.page}>
      <span className={styles.code}>404</span>
      <h1 className={styles.heading}>Page not found</h1>
      <a href="/" className={styles.link}>Go home</a>
    </main>
  )
}
