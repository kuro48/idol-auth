import styles from './NotFoundPage.module.css'

export function NotFoundPage() {
  return (
    <main className={styles.page}>
      <span className={styles.code}>404</span>
      <h1 className={styles.heading}>ページが見つかりません</h1>
      <a href="/" className={styles.link}>ホームへ戻る</a>
    </main>
  )
}
