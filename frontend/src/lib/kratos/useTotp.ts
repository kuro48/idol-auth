import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

interface TotpFlowNode {
  type: string
  group: string
  attributes: {
    name?: string
    type?: string
    value?: string
    src?: string
    text?: { text?: string } | string
  }
  meta?: { label?: { text: string } }
}

export interface ParsedTotpFlow {
  flowId: string
  action: string
  csrfToken: string
  hasTotp: boolean
  qrSrc: string
  secretKey: string
}

async function fetchTotpFlow(): Promise<ParsedTotpFlow> {
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
  const nodes: TotpFlowNode[] = flow.ui?.nodes ?? []

  const csrfNode = nodes.find(n => n.attributes.name === 'csrf_token')
  const qrNode = nodes.find(n => n.group === 'totp' && n.attributes.name === 'totp_qr')
  const secretNode = nodes.find(
    n => n.group === 'totp' && n.attributes.name === 'totp_secret_key',
  )
  const unlinkNode = nodes.find(
    n => n.group === 'totp' && n.attributes.name === 'totp_unlink',
  )

  let secretKey = ''
  if (secretNode) {
    const text = secretNode.attributes.text
    if (typeof text === 'string') {
      secretKey = text
    } else if (text && typeof text === 'object' && 'text' in text) {
      secretKey = text.text ?? ''
    }
    if (secretKey === '' && secretNode.attributes.value) {
      secretKey = secretNode.attributes.value
    }
  }

  return {
    flowId: flow.id,
    action: flow.ui.action,
    csrfToken: csrfNode?.attributes.value ?? '',
    hasTotp: Boolean(unlinkNode),
    qrSrc: qrNode?.attributes.src ?? '',
    secretKey,
  }
}

async function submitTotpCode(flow: ParsedTotpFlow, code: string): Promise<void> {
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
      method: 'totp',
      totp_code: code,
    }).toString(),
  })

  if (!res.ok) {
    const body: { ui?: { messages?: { text: string }[] } } = await res
      .json()
      .catch(() => ({}))
    throw new Error(
      body.ui?.messages?.[0]?.text ?? `TOTP の登録に失敗しました (${res.status})`,
    )
  }
}

async function submitTotpUnlink(flow: ParsedTotpFlow): Promise<void> {
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
      method: 'totp',
      totp_unlink: 'true',
    }).toString(),
  })

  if (!res.ok) {
    const body: { ui?: { messages?: { text: string }[] } } = await res
      .json()
      .catch(() => ({}))
    throw new Error(
      body.ui?.messages?.[0]?.text ?? `TOTP の解除に失敗しました (${res.status})`,
    )
  }
}

export function useTotp() {
  const qc = useQueryClient()

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['kratos', 'totp'],
    queryFn: fetchTotpFlow,
    retry: false,
    staleTime: 0,
  })

  const enroll = useMutation({
    mutationFn: async (code: string) => {
      if (!data) throw new Error('設定が読み込まれていません')
      const trimmed = code.trim()
      if (!/^\d{6}$/.test(trimmed)) {
        throw new Error('6桁の数字コードを入力してください')
      }
      await submitTotpCode(data, trimmed)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['kratos', 'totp'] }),
  })

  const unlink = useMutation({
    mutationFn: async () => {
      if (!data) throw new Error('設定が読み込まれていません')
      await submitTotpUnlink(data)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['kratos', 'totp'] }),
  })

  return {
    flow: data,
    isLoading,
    error,
    enroll,
    unlink,
    refetch,
  }
}
