import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { usePasskeys } from '@/lib/kratos/usePasskeys'
import { useTotp } from '@/lib/kratos/useTotp'
import { api } from '@/lib/api/client'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import styles from './AccountSecurityPage.module.css'

interface EmailStatus {
  email: string
  verified: boolean
}

interface RecoveryContacts {
  recovery_email: string
  recovery_phone: string
}

interface SocialProvider {
  provider: string
  subject: string
}

const PROVIDER_LABELS: Record<string, string> = {
  google: 'Google',
  twitter: 'X (Twitter)',
  apple: 'Apple',
}

function isAal2Error(err: unknown): boolean {
  return (
    err != null &&
    typeof (err as { status?: unknown }).status === 'number' &&
    (err as { status: number }).status === 403
  )
}

export function AccountSecurityPage() {
  const qc = useQueryClient()

  const { data: emailStatus } = useQuery({
    queryKey: ['account', 'email-status'],
    queryFn: () => api.get<EmailStatus>('/v1/account/email-status'),
  })

  const { data: socialProvidersData } = useQuery({
    queryKey: ['account', 'social-providers'],
    queryFn: () => api.get<{ items: SocialProvider[] }>('/v1/account/social-providers'),
  })
  const socialProviders = socialProvidersData?.items ?? []

  const { data: recoveryContacts } = useQuery({
    queryKey: ['account', 'recovery-contacts'],
    queryFn: () => api.get<{ email_status?: { recovery_email?: string; recovery_phone?: string } }>('/v1/account/profile').then(p => ({
      recovery_email: '',
      recovery_phone: '',
      ...p,
    } as RecoveryContacts)),
  })

  const [recoveryEmail, setRecoveryEmail] = useState('')
  const [recoveryPhone, setRecoveryPhone] = useState('')
  const [recoveryError, setRecoveryError] = useState<string | null>(null)
  const [isEditingRecovery, setIsEditingRecovery] = useState(false)

  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [isEditingPassword, setIsEditingPassword] = useState(false)

  const savePassword = useMutation({
    mutationFn: (payload: { new_password: string; confirm_password: string }) =>
      api.post('/v1/account/password-change', payload),
    onSuccess: () => {
      setNewPassword('')
      setConfirmPassword('')
      setIsEditingPassword(false)
      setPasswordError(null)
    },
    onError: (err: unknown) => {
      setPasswordError(err instanceof Error ? err.message : 'パスワード変更に失敗しました')
    },
  })

  function handleSavePassword(e: React.FormEvent) {
    e.preventDefault()
    if (newPassword !== confirmPassword) {
      setPasswordError('パスワードが一致しません')
      return
    }
    if (newPassword.length < 8) {
      setPasswordError('パスワードは8文字以上で入力してください')
      return
    }
    setPasswordError(null)
    savePassword.mutate({ new_password: newPassword, confirm_password: confirmPassword })
  }

  const saveRecovery = useMutation({
    mutationFn: (payload: { recovery_email?: string; recovery_phone?: string }) =>
      api.patch<RecoveryContacts>('/v1/account/recovery-contacts', payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['account', 'recovery-contacts'] })
      qc.invalidateQueries({ queryKey: ['account', 'profile'] })
      setIsEditingRecovery(false)
      setRecoveryError(null)
    },
    onError: (err: unknown) => {
      setRecoveryError(err instanceof Error ? err.message : '保存に失敗しました')
    },
  })

  function startEditRecovery() {
    setRecoveryEmail((recoveryContacts as { recovery_email?: string })?.recovery_email ?? '')
    setRecoveryPhone((recoveryContacts as { recovery_phone?: string })?.recovery_phone ?? '')
    setRecoveryError(null)
    setIsEditingRecovery(true)
  }

  function handleSaveRecovery(e: React.FormEvent) {
    e.preventDefault()
    saveRecovery.mutate({
      recovery_email: recoveryEmail.trim() || undefined,
      recovery_phone: recoveryPhone.trim() || undefined,
    })
  }

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
              <h2 className={styles.sectionTitle}>パスワード変更</h2>
              <p className={styles.sectionDesc}>アカウントのパスワードを変更します。</p>
            </div>
            {!isEditingPassword && (
              <button className={styles.addBtn} onClick={() => { setIsEditingPassword(true); setPasswordError(null) }}>
                変更する
              </button>
            )}
          </div>
          {isEditingPassword && (
            <form onSubmit={handleSavePassword} className={styles.totpForm}>
              <label className={styles.totpStepLabel} htmlFor="new-password">新しいパスワード</label>
              <input
                id="new-password"
                className={styles.totpInput}
                type="password"
                value={newPassword}
                onChange={e => setNewPassword(e.target.value)}
                placeholder="8文字以上"
                autoComplete="new-password"
                minLength={8}
                required
              />
              <label className={styles.totpStepLabel} htmlFor="confirm-password">確認（再入力）</label>
              <input
                id="confirm-password"
                className={styles.totpInput}
                type="password"
                value={confirmPassword}
                onChange={e => setConfirmPassword(e.target.value)}
                placeholder="同じパスワードを入力"
                autoComplete="new-password"
                required
              />
              {passwordError && <div className={styles.alertError}>{passwordError}</div>}
              {savePassword.isSuccess && <div className={styles.alertSuccess}>パスワードを変更しました。</div>}
              <div className={styles.totpActions}>
                <button type="submit" className={styles.addBtn} disabled={savePassword.isPending}>
                  {savePassword.isPending ? '変更中…' : '変更する'}
                </button>
                <button
                  type="button"
                  className={styles.removeBtn}
                  onClick={() => { setIsEditingPassword(false); setPasswordError(null) }}
                >
                  キャンセル
                </button>
              </div>
            </form>
          )}
        </section>

        {emailStatus && (
          <section className={styles.section}>
            <div className={styles.sectionHeader}>
              <div>
                <h2 className={styles.sectionTitle}>
                  メールアドレス認証{' '}
                  {emailStatus.verified
                    ? <Badge variant="success">認証済み</Badge>
                    : <Badge variant="default">未認証</Badge>
                  }
                </h2>
                <p className={styles.sectionDesc}>{emailStatus.email}</p>
              </div>
              {!emailStatus.verified && (
                <a
                  className={styles.addBtn}
                  href="/settings"
                >
                  認証メールを送信
                </a>
              )}
            </div>
            {!emailStatus.verified && (
              <p className={styles.sectionDesc}>
                メールアドレスが未認証です。「認証メールを送信」をクリックして認証を完了してください。
              </p>
            )}
          </section>
        )}

        <section className={styles.section}>
          <div className={styles.sectionHeader}>
            <div>
              <h2 className={styles.sectionTitle}>連携済みソーシャルログイン</h2>
              <p className={styles.sectionDesc}>
                Google / X / Apple などのアカウントで直接ログインできます。
              </p>
            </div>
            <a className={styles.addBtn} href="/settings">
              ソーシャルログインを追加
            </a>
          </div>
          {socialProviders.length === 0 ? (
            <p className={styles.empty}>連携済みのソーシャルログインはありません。</p>
          ) : (
            <ul className={styles.list} aria-label="連携済みソーシャルログイン">
              {socialProviders.map(p => (
                <li key={p.provider + ':' + p.subject} className={styles.item}>
                  <div className={styles.itemInfo}>
                    <span className={styles.itemName}>{PROVIDER_LABELS[p.provider] ?? p.provider}</span>
                    <span className={styles.sectionDesc}>{p.subject}</span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section className={styles.section}>
          <div className={styles.sectionHeader}>
            <div>
              <h2 className={styles.sectionTitle}>復旧用連絡先</h2>
              <p className={styles.sectionDesc}>
                メインのメールアドレスにアクセスできない場合の復旧用連絡先です。MFA (二段階認証) が必要です。
              </p>
            </div>
            {!isEditingRecovery && (
              <button className={styles.addBtn} onClick={startEditRecovery}>
                編集
              </button>
            )}
          </div>
          {!isEditingRecovery ? (
            <div>
              <div className={styles.totpStatus}>
                復旧用メール: {(recoveryContacts as { recovery_email?: string })?.recovery_email || '未設定'}
              </div>
              <div className={styles.totpStatus}>
                復旧用電話: {(recoveryContacts as { recovery_phone?: string })?.recovery_phone || '未設定'}
              </div>
            </div>
          ) : (
            <form onSubmit={handleSaveRecovery} className={styles.totpForm}>
              <label className={styles.totpStepLabel} htmlFor="recovery-email">復旧用メールアドレス</label>
              <input
                id="recovery-email"
                className={styles.totpInput}
                type="email"
                value={recoveryEmail}
                onChange={e => setRecoveryEmail(e.target.value)}
                placeholder="backup@example.com"
                autoComplete="email"
              />
              <label className={styles.totpStepLabel} htmlFor="recovery-phone">復旧用電話番号</label>
              <input
                id="recovery-phone"
                className={styles.totpInput}
                type="tel"
                value={recoveryPhone}
                onChange={e => setRecoveryPhone(e.target.value)}
                placeholder="+81-90-1234-5678"
              />
              {recoveryError && <div className={styles.alertError}>{recoveryError}</div>}
              {saveRecovery.isError && (
                <div className={styles.alertError}>
                  MFAが必要です。パスキーまたはTOTPで再認証してください。
                </div>
              )}
              <div className={styles.totpActions}>
                <button type="submit" className={styles.addBtn} disabled={saveRecovery.isPending}>
                  {saveRecovery.isPending ? '保存中…' : '保存'}
                </button>
                <button
                  type="button"
                  className={styles.removeBtn}
                  onClick={() => { setIsEditingRecovery(false); setRecoveryError(null) }}
                >
                  キャンセル
                </button>
              </div>
            </form>
          )}
        </section>

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
