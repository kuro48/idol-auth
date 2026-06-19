import { describe, expect, it } from 'vitest'
import { parsePasskeyFlow } from './usePasskeys'

function makeFlow(nodes: unknown[]) {
  return {
    id: 'flow-123',
    ui: {
      action: '/self-service/settings?flow=flow-123',
      nodes,
    },
  }
}

describe('parsePasskeyFlow', () => {
  it('allows registration when Kratos exposes a passkey register button', () => {
    const flow = makeFlow([
      {
        type: 'input',
        group: 'default',
        attributes: {
          name: 'csrf_token',
          type: 'hidden',
          value: 'csrf-123',
        },
      },
      {
        type: 'script',
        group: 'passkey',
        attributes: {
          src: '/.well-known/ory/webauthn.js',
        },
      },
      {
        type: 'input',
        group: 'passkey',
        attributes: {
          name: 'passkey_settings_register',
          type: 'button',
          onclick: 'window.__oryPasskeySettingsRegistration()',
        },
        meta: {
          label: {
            text: 'Add passkey',
          },
        },
      },
    ])

    expect(parsePasskeyFlow(flow).canRegister).toBe(true)
  })

  it('does not allow registration when the register button is disabled', () => {
    const flow = makeFlow([
      {
        type: 'input',
        group: 'passkey',
        attributes: {
          name: 'passkey_settings_register',
          type: 'button',
          onclick: 'window.__oryPasskeySettingsRegistration()',
          disabled: true,
        },
      },
    ])

    expect(parsePasskeyFlow(flow).canRegister).toBe(false)
  })
})
