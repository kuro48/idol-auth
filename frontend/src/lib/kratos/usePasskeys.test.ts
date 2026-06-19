import { describe, expect, it } from 'vitest'
import { parsePasskeyFlow } from './usePasskeys'

type PasskeyFlow = Parameters<typeof parsePasskeyFlow>[0]
type PasskeyFlowNodes = NonNullable<NonNullable<PasskeyFlow['ui']>['nodes']>

function makeFlow(nodes: PasskeyFlowNodes): PasskeyFlow {
  return {
    id: 'flow-123',
    ui: {
      action: '/self-service/settings?flow=flow-123',
      nodes,
    },
  }
}

describe('parsePasskeyFlow', () => {
  it('reads passkey registration data from the Kratos flow', () => {
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
        type: 'input',
        group: 'passkey',
        attributes: {
          name: 'passkey',
          type: 'hidden',
          value: 'registration-options',
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

    const parsed = parsePasskeyFlow(flow)

    expect(parsed.registrationOptionsBase64).toBe('registration-options')
    expect(parsed.registerOnClick).toBe('window.__oryPasskeySettingsRegistration()')
  })

  it('uses the default registration callback when the flow has no register button node', () => {
    const flow = makeFlow([])

    expect(parsePasskeyFlow(flow).registerOnClick).toBe(
      'window.__oryPasskeySettingsRegistration()',
    )
  })
})
