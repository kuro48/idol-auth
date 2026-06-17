import { useState } from 'react'
import { usePasskeys } from '@/lib/kratos/usePasskeys'
import { useTotp } from '@/lib/kratos/useTotp'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import styles from './AccountSecurityPage.module.css'

function isAal2Error(err: unknown): boolean {
  return (
    err != null &&
    typeof (err as { status?: unknown }).status === 'number' &&
    (err as { status: number }).status === 403
  )
}

export function AccountSecurityPage() {
  const { passkeys, isLoading, error, canRegister, register, remove } = usePasskeys()
  const {
    flow: totpFlow,
    isLoading: totpLoading,
    error: totpError,
    enroll: totpEnroll,
    unlink: totpUnlink,
  } = useTotp()
  const [registerError, setRegisterError] = useState<string | null>(null)
  const [removeError, setRemoveError] = useState<string | null>(null)
  const [totpCode, setTotpCode] = useState('')
  const [totpFormError, setTotpFormError] = useState<string | null>(null)

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

  async function handleTotpEnroll(e: React.FormEvent) {
    e.preventDefault()
    setTotpFormError(null)
    try {
      await totpEnroll.mutateAsync(totpCode)
      setTotpCode('')
    } catch (err) {
      setTotpFormError(err instanceof Error ? err.message : 'TOTP の登録に失敗しました')
    }
  }

  async function handleTotpUnlink() {
    setTotpFormError(null)
    if (!window.confirm('二段階認証 (TOTP) を解除します。よろしいですか？')) {
      return
    }
    try {
      await totpUnlink.mutateAsync()
    } catch (err) {
      setTotpFormError(err instanceof Error ? err.message : 'TOTP の解除に失敗しました')
    }
  }

  const passkeyAalError = isAal2Error(error)
  const totpAalError = isAal2Error(totpError)

  return (
    <div>
      <PageHeader
        title="セキュリティ設定"
        description="パスキーや二段階認証でアカウントを保護します。"
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

          {passkeyAalError && (
            <div className={styles.alert}>
              セキュリティのため、再度ログインが必要です。
              <a href="/login" className={styles.alertLink}>再ログイン</a>
            </div>
          )}

          {registerError && <div className={styles.alertError}>{registerError}</div>}
          {removeError && <div className={styles.alertError}>{removeError}</div>}
          {register.isSuccess && (
            <div className={styles.alertSuccess}>パスキーを登録しました。</div>
          )}

          {isLoading && <p className={styles.empty}>読み込み中…</p>}
          {!isLoading && !passkeyAalError && passkeys.length === 0 && (
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
              <h2 className={styles.sectionTitle}>
                二段階認証 (TOTP) {totpFlow?.hasTotp && <Badge variant="success">有効</Badge>}
              </h2>
              <p className={styles.sectionDesc}>
                認証アプリ (Google Authenticator / 1Password など) で生成したコードを
                ログイン時に追加で要求します。
              </p>
            </div>
          </div>

          {totpAalError && (
            <div className={styles.alert}>
              セキュリティのため、再度ログインが必要です。
              <a href="/login" className={styles.alertLink}>再ログイン</a>
            </div>
          )}

          {totpFormError && <div className={styles.alertError}>{totpFormError}</div>}
          {totpEnroll.isSuccess && (
            <div className={styles.alertSuccess}>二段階認証を有効化しました。</div>
          )}
          {totpUnlink.isSuccess && (
            <div className={styles.alertSuccess}>二段階認証を解除しました。</div>
          )}

          <div className={styles.totpBody}>
            {totpLoading && <p className={styles.empty}>読み込み中…</p>}

            {!totpLoading && !totpAalError && totpFlow?.hasTotp && (
              <>
                <div className={styles.totpStatus}>
                  ✓ このアカウントは二段階認証で保護されています。
                </div>
                <div className={styles.totpActions}>
                  <button
                    className={styles.removeBtn}
                    onClick={handleTotpUnlink}
                    disabled={totpUnlink.isPending}
                  >
                    {totpUnlink.isPending ? '解除中…' : '二段階認証を解除'}
                  </button>
                </div>
              </>
            )}

            {!totpLoading && !totpAalError && totpFlow && !totpFlow.hasTotp && (
              <div className={styles.totpEnrollGrid}>
                {totpFlow.qrSrc ? (
                  <img className={styles.totpQr} src={totpFlow.qrSrc} alt="TOTP QR Code" />
                ) : (
                  <div className={styles.empty}>QRコードが利用できません</div>
                )}
                <div className={styles.totpDetails}>
                  <div>
                    <div className={styles.totpStepLabel}>Step 1</div>
                    <p className={styles.sectionDesc}>
                      認証アプリでQRコードを読み取るか、下記キーを手動入力してください。
                    </p>
                    {totpFlow.secretKey && (
                      <div className={styles.totpSecret} aria-label="TOTP secret key">
                        {totpFlow.secretKey}
                      </div>
                    )}
                  </div>
                  <form className={styles.totpForm} onSubmit={handleTotpEnroll}>
                    <div className={styles.totpStepLabel}>Step 2 — 確認コード</div>
                    <input
                      className={styles.totpInput}
                      value={totpCode}
                      onChange={e => setTotpCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                      placeholder="000000"
                      inputMode="numeric"
                      autoComplete="one-time-code"
                      aria-label="認証コード"
                      maxLength={6}
                      required
                    />
                    <div className={styles.totpActions}>
                      <button
                        className={styles.addBtn}
                        type="submit"
                        disabled={totpEnroll.isPending || totpCode.length !== 6}
                      >
                        {totpEnroll.isPending ? '登録中…' : '二段階認証を有効化'}
                      </button>
                    </div>
                  </form>
                </div>
              </div>
            )}
          </div>
        </section>
      </div>
    </div>
  )
}
