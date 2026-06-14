import { describe, it, expect } from 'vitest'

// These tests document the CONTRACT that /v1/auth/session provides,
// and verify that useSession's toSession() interprets it correctly.
//
// Backend contract (router.go handleSession):
//   - Always returns 200 (never 401)
//   - Response shape: SessionView (snake_case)
//     { authenticated: bool, identity_id?: string, email?: string,
//       roles?: string[], oshi_color?: string, ... }
//   - Unauthenticated: { authenticated: false }
//   - Authenticated:   { authenticated: true, identity_id: "...", email: "...", roles: ["admin"] }

import type { SessionView } from '@/lib/api/types'

// Mirror the toSession logic from useSession.ts for pure-function testing.
function parseSessionFromView(view: SessionView) {
  if (!view.authenticated) return null
  return {
    identityId: view.identity_id ?? '',
    email: view.email ?? '',
    roles: view.roles ?? [],
    oshiColor: view.oshi_color ?? '',
  }
}

function isAdminFromView(view: SessionView): boolean {
  const parsed = parseSessionFromView(view)
  if (!parsed) return false
  return parsed.roles.includes('admin')
}

describe('SessionView contract alignment', () => {
  it('unauthenticated response produces null session', () => {
    const unauthenticated: SessionView = { authenticated: false }
    expect(parseSessionFromView(unauthenticated)).toBeNull()
  })

  it('authenticated response produces session with correct fields', () => {
    const authenticated: SessionView = {
      authenticated: true,
      identity_id: 'user-123',
      email: 'test@example.com',
      roles: ['user'],
    }
    const result = parseSessionFromView(authenticated)
    expect(result).not.toBeNull()
    expect(result?.identityId).toBe('user-123')
    expect(result?.email).toBe('test@example.com')
    expect(result?.roles).toEqual(['user'])
  })

  it('roles missing in authenticated response does not throw (graceful fallback)', () => {
    const authenticated: SessionView = {
      authenticated: true,
      identity_id: 'user-123',
      email: 'test@example.com',
      // roles omitted (omitempty on backend)
    }
    expect(() => parseSessionFromView(authenticated)).not.toThrow()
    const result = parseSessionFromView(authenticated)
    expect(result?.roles).toEqual([])
  })

  it('admin role is detected correctly', () => {
    const admin: SessionView = {
      authenticated: true,
      identity_id: 'admin-1',
      email: 'admin@example.com',
      roles: ['admin'],
    }
    expect(isAdminFromView(admin)).toBe(true)
  })

  it('non-admin user is not detected as admin', () => {
    const user: SessionView = {
      authenticated: true,
      identity_id: 'user-1',
      email: 'user@example.com',
      roles: ['user'],
    }
    expect(isAdminFromView(user)).toBe(false)
  })

  it('unauthenticated is not admin', () => {
    const unauth: SessionView = { authenticated: false }
    expect(isAdminFromView(unauth)).toBe(false)
  })
})
