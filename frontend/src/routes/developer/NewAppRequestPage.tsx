import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { api, ApiError } from '@/lib/api/client'
import type { AppRequest } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import { appRequestFormSchema, type AppRequestFormValues } from './appRequestSchema'
import styles from './NewAppRequestPage.module.css'

type FormValues = AppRequestFormValues

export function NewAppRequestPage() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { register, handleSubmit, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(appRequestFormSchema),
    defaultValues: { type: 'web' },
  })

  const submit = useMutation({
    mutationFn: (values: FormValues) => api.post<AppRequest>('/v1/developer/app-requests', {
      name: values.name,
      type: values.type,
      description: values.description,
      homepage_url: values.homepage_url,
      privacy_policy_url: values.privacy_policy_url,
      terms_url: values.terms_url || undefined,
      contact_email: values.contact_email || undefined,
      organization: values.organization || undefined,
      purpose: values.purpose,
      redirect_uris: values.redirect_uris.split('\n').map(s => s.trim()).filter(Boolean),
      scopes: values.scopes ? values.scopes.split(',').map(s => s.trim()).filter(Boolean) : [],
    }),
    onSuccess: (req) => {
      qc.invalidateQueries({ queryKey: ['developer', 'app-requests'] })
      navigate({ to: `/developer/app-requests/${req.id}` })
    },
  })

  const serverError = submit.error instanceof ApiError ? submit.error.message : null

  return (
    <div>
      <PageHeader title="アプリ申請" description="アプリの登録申請を送信します。" />
      <form className={styles.form} onSubmit={handleSubmit(v => submit.mutate(v))}>
        {serverError && <p className={styles.serverError}>{serverError}</p>}
        <Field label="アプリ名" error={errors.name?.message}>
          <input {...register('name')} className={styles.input} placeholder="マイアプリ" />
        </Field>
        <Field label="種別" error={errors.type?.message}>
          <select {...register('type')} className={styles.input}>
            <option value="web">Web</option>
            <option value="spa">SPA</option>
            <option value="native">ネイティブアプリ</option>
            <option value="m2m">M2M</option>
          </select>
        </Field>
        <Field label="説明" error={errors.description?.message}>
          <textarea {...register('description')} className={styles.textarea} rows={3} />
        </Field>
        <Field label="利用目的" description="50文字以上" error={errors.purpose?.message}>
          <textarea {...register('purpose')} className={styles.textarea} rows={5} />
        </Field>
        <Field label="ホームページ URL" error={errors.homepage_url?.message}>
          <input {...register('homepage_url')} className={styles.input} placeholder="https://example.com" />
        </Field>
        <Field label="プライバシーポリシー URL" error={errors.privacy_policy_url?.message}>
          <input {...register('privacy_policy_url')} className={styles.input} placeholder="https://example.com/privacy" />
        </Field>
        <Field label="利用規約 URL" error={errors.terms_url?.message}>
          <input {...register('terms_url')} className={styles.input} placeholder="https://example.com/terms（任意）" />
        </Field>
        <Field label="連絡先メール" error={errors.contact_email?.message}>
          <input {...register('contact_email')} className={styles.input} placeholder="contact@example.com（任意）" />
        </Field>
        <Field label="組織名" error={errors.organization?.message}>
          <input {...register('organization')} className={styles.input} placeholder="株式会社〇〇（任意）" />
        </Field>
        <Field label="リダイレクト URI" description="1行に1つ入力" error={errors.redirect_uris?.message}>
          <textarea {...register('redirect_uris')} className={styles.textarea} rows={3} placeholder="https://example.com/callback" />
        </Field>
        <Field label="スコープ" description="カンマ区切り（例: openid, profile, email）" error={errors.scopes?.message}>
          <input {...register('scopes')} className={styles.input} placeholder="openid, profile, email" />
        </Field>
        <div className={styles.footer}>
          <button type="submit" className={styles.submitBtn} disabled={submit.isPending}>
            {submit.isPending ? '送信中…' : '申請する'}
          </button>
        </div>
      </form>
    </div>
  )
}

function Field({ label, description, error, children }: { label: string; description?: string; error?: string; children: React.ReactNode }) {
  return (
    <div className={styles.field}>
      <label className={styles.label}>{label}{description && <span className={styles.hint}> — {description}</span>}</label>
      {children}
      {error && <span className={styles.error}>{error}</span>}
    </div>
  )
}
