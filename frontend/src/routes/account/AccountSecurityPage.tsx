import { useState } from 'react'
import { usePasskeys } from '@/lib/kratos/usePasskeys'
import { PageHeader } from '@/components/ui/PageHeader'
import styles from './AccountSecurityPage.module.css'

export function AccountSecurityPage() {
  const { passkeys, isLoading, error, canRegister, register, remove } = usePasskeys()
  const [registerError, setRegisterError] = useState<string | null>(null)
  const [removeError, setRemoveError] = useState<string | null>(null)

  async function handleRegister() {
    setRegisterError(null)
    try {
      await register.mutateAsync()
    } catch (err) {
      setRegisterError(err instanceof Error ? err.message : 'パスキーの登録に失敗しました')
    }
  }

  async function handleRemove(passkeyId: string) {
    setRemoveError(null)
    try {
      await remove.mutateAsync(passkeyId)
    } catch (err) {
      setRemoveError(err instanceof Error ? err.message : 'パスキーの削除に失敗しました')
    }
  }

  const isAalError =
    error != null &&
    typeof (error as unknown as { status?: unknown }).status === 'number' &&
    (error as unknown as { status: number }).status === 403

  return (
    <div>
      <PageHeader
        title="セキュリティ設定"
        description="パスキーやログイン方法を管理します。"
      />
      <div className={styles.content}>
        <section className={styles.section}>
          <div className={styles.sectionHeader}>
            <div>
              <h2 className={styles.sectionTitle}>パスキー</h2>
              <p className={styles.sectionDesc}>
                顔認証・指紋・デバイスPINを使って、パスワード不要でログインできます。
              </p>
            </div>
            {canRegister && (
              <button
                className={styles.addBtn}
                onClick={handleRegister}
                disabled={register.isPending}
                aria-busy={register.isPending}
              >
                {register.isPending ? '登録中…' : 'パスキーを追加'}
              </button>
            )}
          </div>

          {isAalError && (
            <div className={styles.alert}>
              セキュリティのため、再度ログインが必要です。
              <a href="/login" className={styles.alertLink}>再ログイン</a>
            </div>
          )}

          {registerError && (
            <div className={styles.alertError}>{registerError}</div>
          )}

          {removeError && (
            <div className={styles.alertError}>{removeError}</div>
          )}

          {register.isSuccess && (
            <div className={styles.alertSuccess}>パスキーを登録しました。</div>
          )}

          {isLoading && <p className={styles.empty}>読み込み中…</p>}

          {!isLoading && !isAalError && passkeys.length === 0 && (
            <p className={styles.empty}>登録済みのパスキーはありません。</p>
          )}

          {passkeys.length > 0 && (
            <ul className={styles.list} aria-label="登録済みパスキー">
              {passkeys.map(pk => (
                <li key={pk.id} className={styles.item}>
                  <div className={styles.itemInfo}>
                    <span className={styles.itemName}>{pk.displayName}</span>
                  </div>
                  <button
                    className={styles.removeBtn}
                    onClick={() => handleRemove(pk.id)}
                    disabled={remove.isPending}
                    aria-label={`${pk.displayName} を削除`}
                  >
                    削除
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section className={styles.section}>
          <div className={styles.sectionHeader}>
            <div>
              <h2 className={styles.sectionTitle}>その他のセキュリティ設定</h2>
              <p className={styles.sectionDesc}>
                パスワード変更・二段階認証（TOTP）などはセキュリティポータルで管理できます。
              </p>
            </div>
            <a href="/settings" className={styles.portalLink}>
              セキュリティポータルを開く →
            </a>
          </div>
        </section>
      </div>
    </div>
  )
}
