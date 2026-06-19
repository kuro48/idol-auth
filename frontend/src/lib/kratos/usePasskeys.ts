import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

export interface KratosPasskey {
  id: string
  displayName: string
}

interface PasskeyFlowNode {
  type: string
  group: string
  attributes: {
    name?: string
    type?: string
    value?: string
    onclick?: string
    src?: string
    integrity?: string
    nonce?: string
    disabled?: boolean
  }
  meta?: { label?: { text: string } }
}

interface ParsedPasskeyFlow {
  flowId: string
  action: string
  csrfToken: string
  passkeys: KratosPasskey[]
  registrationOptionsBase64: string
  registerOnClick: string
  scriptSrc: string
  scriptIntegrity?: string
}

interface KratosSettingsFlow {
  id?: string
  ui?: {
    action?: string
    nodes?: PasskeyFlowNode[]
  }
}

export function parsePasskeyFlow(flow: KratosSettingsFlow): ParsedPasskeyFlow {
  const nodes: PasskeyFlowNode[] = flow.ui?.nodes ?? []

  const scriptNode = nodes.find(n => n.type === 'script' && n.group === 'passkey')
  const removeNodes = nodes.filter(
    n => n.group === 'passkey' && n.attributes.name === 'passkey_remove',
  )
  const hiddenPasskeyNode = nodes.find(
    n =>
      n.group === 'passkey' &&
      n.attributes.name === 'passkey' &&
      n.attributes.type === 'hidden',
  )
  const registerBtnNode = nodes.find(
    n => n.group === 'passkey' && n.attributes.name === 'passkey_settings_register',
  )
  const csrfNode = nodes.find(n => n.attributes.name === 'csrf_token')

  return {
    flowId: flow.id ?? '',
    action: flow.ui?.action ?? '',
    csrfToken: csrfNode?.attributes.value ?? '',
    passkeys: removeNodes.map(n => ({
      id: n.attributes.value ?? '',
      displayName: n.meta?.label?.text ?? n.attributes.value ?? 'Passkey',
    })),
    registrationOptionsBase64: hiddenPasskeyNode?.attributes.value ?? '',
    registerOnClick:
      registerBtnNode?.attributes.onclick ?? 'window.__oryPasskeySettingsRegistration()',
    scriptSrc: scriptNode?.attributes.src ?? '/.well-known/ory/webauthn.js',
    scriptIntegrity: scriptNode?.attributes.integrity,
  }
}

async function fetchPasskeyFlow(): Promise<ParsedPasskeyFlow> {
  const res = await fetch('/self-service/settings/browser', {
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })

  if (!res.ok) {
    if (res.status === 401) {
      throw Object.assign(new Error('not_authenticated'), { status: 401 })
    }
    if (res.status === 403) {
      throw Object.assign(new Error('aal_too_low'), { status: 403 })
    }
    throw new Error(`settings_flow: ${res.status}`)
  }

  const flow = await res.json()
  return parsePasskeyFlow(flow)
}

async function loadPasskeyScript(src: string, integrity?: string): Promise<void> {
  if (document.querySelector('script[data-kratos-passkey]')) return

  return new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.src = src
    script.async = true
    script.setAttribute('data-kratos-passkey', '')
    script.crossOrigin = 'anonymous'
    if (integrity) script.integrity = integrity
    script.onload = () => resolve()
    script.onerror = () => reject(new Error('WebAuthnスクリプトの読み込みに失敗しました'))
    document.head.appendChild(script)
  })
}

async function runRegistrationCeremony(flow: ParsedPasskeyFlow): Promise<void> {
  if (!flow.registrationOptionsBase64) {
    throw new Error('このブラウザはパスキーに対応していません')
  }

  await loadPasskeyScript(flow.scriptSrc, flow.scriptIntegrity)

  const container = document.createElement('div')
  container.style.cssText = 'position:absolute;left:-9999px;top:-9999px;'
  container.setAttribute('aria-hidden', 'true')

  const form = document.createElement('form')
  form.action = flow.action
  form.method = 'POST'

  const addHidden = (name: string, value: string): HTMLInputElement => {
    const el = document.createElement('input')
    el.type = 'hidden'
    el.name = name
    el.value = value
    form.appendChild(el)
    return el
  }

  addHidden('csrf_token', flow.csrfToken)
  addHidden('flow', flow.flowId)
  const passkeyInput = addHidden('passkey', flow.registrationOptionsBase64)

  container.appendChild(form)
  document.body.appendChild(container)

  return new Promise<void>((resolve, reject) => {
    const cleanup = () => {
      if (document.body.contains(container)) document.body.removeChild(container)
    }

    const submitHandler = async function (this: HTMLFormElement) {
      const credentialResult = passkeyInput.value
      cleanup()

      try {
        const res = await fetch(flow.action, {
          method: 'POST',
          credentials: 'include',
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
            Accept: 'application/json',
          },
          body: new URLSearchParams({
            csrf_token: flow.csrfToken,
            flow: flow.flowId,
            passkey: credentialResult,
          }).toString(),
        })

        if (!res.ok) {
          const body: { ui?: { messages?: { text: string }[] }; error?: { message?: string } } =
            await res.json().catch(() => ({}))
          const msg =
            body.ui?.messages?.[0]?.text ??
            body.error?.message ??
            `パスキーの登録に失敗しました (${res.status})`
          reject(new Error(msg))
          return
        }
        resolve()
      } catch (err) {
        reject(err instanceof Error ? err : new Error('パスキーの登録に失敗しました'))
      }
    }

    // webauthn.js calls form.submit() — override it to intercept and use fetch
    Object.defineProperty(form, 'submit', {
      value: submitHandler.bind(form),
      writable: true,
    })

    const fnName = flow.registerOnClick
      .replace(/^window\./, '')
      .replace(/\(\s*\)$/, '')
    const fn = (window as unknown as Record<string, unknown>)[fnName]

    if (typeof fn !== 'function') {
      cleanup()
      reject(
        new Error(
          'パスキーの登録を開始できませんでした。ブラウザがパスキーに対応しているか確認してください。',
        ),
      )
      return
    }

    try {
      ;(fn as () => void)()
    } catch (err) {
      cleanup()
      reject(err instanceof Error ? err : new Error('パスキーの登録に失敗しました'))
    }
  })
}

async function removePasskey(flow: ParsedPasskeyFlow, passkeyId: string): Promise<void> {
  const res = await fetch(flow.action, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
      Accept: 'application/json',
    },
    body: new URLSearchParams({
      csrf_token: flow.csrfToken,
      flow: flow.flowId,
      passkey_remove: passkeyId,
    }).toString(),
  })

  if (!res.ok) {
    const body: { ui?: { messages?: { text: string }[] } } = await res.json().catch(() => ({}))
    throw new Error(
      body.ui?.messages?.[0]?.text ?? `パスキーの削除に失敗しました (${res.status})`,
    )
  }
}

export function usePasskeys() {
  const qc = useQueryClient()

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['kratos', 'passkeys'],
    queryFn: fetchPasskeyFlow,
    retry: false,
    staleTime: 0,
  })

  const register = useMutation({
    mutationFn: async () => {
      if (!data) throw new Error('設定が読み込まれていません')
      await runRegistrationCeremony(data)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['kratos', 'passkeys'] }),
  })

  const remove = useMutation({
    mutationFn: async (passkeyId: string) => {
      if (!data) throw new Error('設定が読み込まれていません')
      await removePasskey(data, passkeyId)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['kratos', 'passkeys'] }),
  })

  return {
    passkeys: data?.passkeys ?? [],
    isLoading,
    error,
    register,
    remove,
    refetch,
  }
}
