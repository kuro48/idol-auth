import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { PageHeader } from '@/components/ui/PageHeader'
import styles from './DeveloperRegistrationPage.module.css'

export function DeveloperRegistrationPage() {
  const qc = useQueryClient()
  const [agreed, setAgreed] = useState(false)

  const register = useMutation({
    mutationFn: () => api.post('/v1/account/developer-registration'),
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: ['session'] })
      window.location.replace('/developer/app-requests')
    },
  })

  return (
    <div>
      <PageHeader
        title="開発者登録"
        description="idol-auth の開発者機能を利用するには、以下のポリシーへの同意が必要です。"
      />
      <div className={styles.content}>
        <section className={styles.policyBox}>
          <h2 className={styles.policyTitle}>開発者ポリシー</h2>
          <ol className={styles.policyList}>
            <li>開発者は、idol-auth の API およびクライアント機能を自身のサービス開発目的にのみ使用できます。</li>
            <li>第三者のアカウント情報・セッション情報を不正に取得・利用することを禁止します。</li>
            <li>取得したユーザーデータは、ユーザーが同意した目的の範囲内でのみ使用し、適切に保護してください。</li>
            <li>idol-auth のシステムに過度な負荷を与える行為（スパム・DDoS 等）を禁止します。</li>
            <li>ポリシー違反が確認された場合、開発者権限の停止・アカウントの削除を行う場合があります。</li>
            <li>本ポリシーは予告なく変更される場合があります。継続利用をもって同意とみなします。</li>
          </ol>
        </section>

        {register.isError && (
          <p className={styles.errorMsg}>登録に失敗しました。時間をおいて再度お試しください。</p>
        )}

        <label className={styles.checkLabel}>
          <input
            type="checkbox"
            checked={agreed}
            onChange={e => setAgreed(e.target.checked)}
            className={styles.checkbox}
          />
          上記の開発者ポリシーを読み、内容に同意します
        </label>

        <button
          className={styles.submitBtn}
          disabled={!agreed || register.isPending}
          onClick={() => register.mutate()}
        >
          {register.isPending ? '登録中…' : '同意して開発者登録する'}
        </button>
      </div>
    </div>
  )
}
