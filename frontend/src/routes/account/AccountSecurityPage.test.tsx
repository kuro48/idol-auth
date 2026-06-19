import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'
import { AccountSecurityPage } from './AccountSecurityPage'

vi.mock('@/lib/api/client', () => ({
  api: {
    get: vi.fn(),
    patch: vi.fn(),
    post: vi.fn(),
  },
}))

vi.mock('@/lib/kratos/usePasskeys', () => ({
  usePasskeys: () => ({
    passkeys: [],
    isLoading: false,
    error: null,
    register: {
      isPending: false,
      isSuccess: false,
      mutateAsync: vi.fn(),
    },
    remove: {
      isPending: false,
      mutateAsync: vi.fn(),
    },
  }),
}))

vi.mock('@/lib/kratos/useTotp', () => ({
  useTotp: () => ({
    flow: null,
    isLoading: false,
    error: null,
    enroll: {
      isPending: false,
      isSuccess: false,
      mutateAsync: vi.fn(),
    },
    unlink: {
      isPending: false,
      isSuccess: false,
      mutateAsync: vi.fn(),
    },
  }),
}))

describe('AccountSecurityPage', () => {
  it('always renders the passkey add button', () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    })

    const html = renderToStaticMarkup(
      <QueryClientProvider client={queryClient}>
        <AccountSecurityPage />
      </QueryClientProvider>,
    )

    expect(html).toContain('パスキーを追加')
  })
})
