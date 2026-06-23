import { describe, expect, it } from 'vitest'
import { appRequestFormSchema } from './appRequestSchema'

describe('appRequestFormSchema', () => {
  it('allows a 50 character purpose', () => {
    const result = appRequestFormSchema.safeParse({
      name: 'マイアプリ',
      type: 'web',
      description: 'テスト用アプリ',
      homepage_url: 'https://example.com',
      privacy_policy_url: 'https://example.com/privacy',
      terms_url: '',
      contact_email: '',
      organization: '',
      purpose: 'x'.repeat(50),
      redirect_uris: 'https://example.com/callback',
      scopes: 'openid, profile, email',
    })

    expect(result.success).toBe(true)
  })
})
