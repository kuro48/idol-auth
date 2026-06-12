import { ApiReferenceReact } from '@scalar/api-reference-react'
import '@scalar/api-reference-react/style.css'

export function DocsApiPage() {
  return (
    <ApiReferenceReact
      configuration={{
        url: '/openapi.json',
        theme: 'default',
        hideDownloadButton: false,
      }}
    />
  )
}
