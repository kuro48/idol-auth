import { z } from 'zod'

const httpsUrl = z.string().refine(
  v => { try { const u = new URL(v); return u.protocol === 'https:' } catch { return false } },
  { message: 'https:// で始まる有効な URL を入力してください' }
)

export const appRequestFormSchema = z.object({
  name: z.string().min(1, 'アプリ名は必須です').max(100, 'アプリ名は100文字以内で入力してください'),
  type: z.enum(['web', 'spa', 'native', 'm2m']),
  description: z.string().min(1, '説明は必須です').max(1000, '説明は1000文字以内で入力してください'),
  homepage_url: httpsUrl,
  privacy_policy_url: httpsUrl,
  terms_url: httpsUrl.optional().or(z.literal('')),
  contact_email: z.string().email('有効なメールアドレスを入力してください').optional().or(z.literal('')),
  organization: z.string().optional(),
  purpose: z.string().min(50, '利用目的は50文字以上で入力してください').max(2000, '利用目的は2000文字以内で入力してください'),
  redirect_uris: z.string().min(1, 'リダイレクト URI を1つ以上入力してください'),
  scopes: z.string().optional(),
})

export type AppRequestFormValues = z.infer<typeof appRequestFormSchema>
