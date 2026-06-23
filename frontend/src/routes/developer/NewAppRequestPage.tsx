import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { api, ApiError } from '@/lib/api/client'
import type { AppRequest } from '@/lib/api/types'
import { PageHeader } from '@/components/ui/PageHeader'
import styles from './NewAppRequestPage.module.css'

const httpsUrl = z.string().refine(
  v => { try { const u = new URL(v); return u.protocol === 'https:' } catch { return false } },
  { message: 'Must be a valid https:// URL' }
)

const schema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name must be at most 100 characters'),
  type: z.enum(['web', 'spa', 'native', 'm2m']),
  description: z.string().min(1, 'Description is required').max(1000, 'Description must be at most 1000 characters'),
  homepage_url: httpsUrl,
  privacy_policy_url: httpsUrl,
  terms_url: httpsUrl.optional().or(z.literal('')),
  contact_email: z.string().email('Must be a valid email').optional().or(z.literal('')),
  organization: z.string().optional(),
  purpose: z.string().min(200, 'Purpose must be at least 200 characters').max(2000, 'Purpose must be at most 2000 characters'),
  redirect_uris: z.string().min(1, 'At least one redirect URI is required'),
  scopes: z.string().optional(),
})

type FormValues = z.infer<typeof schema>

export function NewAppRequestPage() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { register, handleSubmit, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(schema),
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
      <PageHeader title="New App Request" description="Submit a registration request for your application." />
      <form className={styles.form} onSubmit={handleSubmit(v => submit.mutate(v))}>
        {serverError && <p className={styles.serverError}>{serverError}</p>}
        <Field label="App Name" error={errors.name?.message}>
          <input {...register('name')} className={styles.input} placeholder="My App" />
        </Field>
        <Field label="Type" error={errors.type?.message}>
          <select {...register('type')} className={styles.input}>
            <option value="web">Web</option>
            <option value="spa">SPA</option>
            <option value="native">Native</option>
            <option value="m2m">M2M</option>
          </select>
        </Field>
        <Field label="Description" error={errors.description?.message}>
          <textarea {...register('description')} className={styles.textarea} rows={3} />
        </Field>
        <Field label="Purpose" error={errors.purpose?.message}>
          <textarea {...register('purpose')} className={styles.textarea} rows={3} />
        </Field>
        <Field label="Homepage URL" error={errors.homepage_url?.message}>
          <input {...register('homepage_url')} className={styles.input} placeholder="https://example.com" />
        </Field>
        <Field label="Privacy Policy URL" error={errors.privacy_policy_url?.message}>
          <input {...register('privacy_policy_url')} className={styles.input} placeholder="https://example.com/privacy" />
        </Field>
        <Field label="Terms URL" error={errors.terms_url?.message}>
          <input {...register('terms_url')} className={styles.input} placeholder="https://example.com/terms (optional)" />
        </Field>
        <Field label="Contact Email" error={errors.contact_email?.message}>
          <input {...register('contact_email')} className={styles.input} placeholder="contact@example.com (optional)" />
        </Field>
        <Field label="Organization" error={errors.organization?.message}>
          <input {...register('organization')} className={styles.input} placeholder="Acme Corp (optional)" />
        </Field>
        <Field label="Redirect URIs" description="One URI per line" error={errors.redirect_uris?.message}>
          <textarea {...register('redirect_uris')} className={styles.textarea} rows={3} placeholder="https://example.com/callback" />
        </Field>
        <Field label="Scopes" description="Comma-separated (e.g. openid, profile, email)" error={errors.scopes?.message}>
          <input {...register('scopes')} className={styles.input} placeholder="openid, profile, email" />
        </Field>
        <div className={styles.footer}>
          <button type="submit" className={styles.submitBtn} disabled={submit.isPending}>
            {submit.isPending ? 'Submitting…' : 'Submit Request'}
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
