export interface ParsedUserAgent {
  browser: string
  os: string
  device: 'desktop' | 'mobile' | 'tablet' | 'unknown'
}

const BROWSERS: Array<[RegExp, string]> = [
  [/Edg\/[\d.]+/, 'Edge'],
  [/OPR\/[\d.]+/, 'Opera'],
  [/Firefox\/[\d.]+/, 'Firefox'],
  [/Chrome\/[\d.]+/, 'Chrome'],
  [/Version\/[\d.]+ Safari/, 'Safari'],
  [/Safari\/[\d.]+/, 'Safari'],
]

const OS_PATTERNS: Array<[RegExp, string]> = [
  [/Windows NT 10\.0/, 'Windows 10/11'],
  [/Windows NT 6\.3/, 'Windows 8.1'],
  [/Windows NT/, 'Windows'],
  [/Mac OS X (\d+[._]\d+)/, 'macOS'],
  [/Android (\d+\.\d+)/, 'Android'],
  [/iPhone OS (\d+_\d+)/, 'iOS'],
  [/iPad/, 'iPadOS'],
  [/CrOS/, 'ChromeOS'],
  [/Linux/, 'Linux'],
]

export function parseUserAgent(ua: string | undefined | null): ParsedUserAgent {
  if (!ua) {
    return { browser: 'Unknown browser', os: 'Unknown OS', device: 'unknown' }
  }

  let browser = 'Unknown browser'
  for (const [re, name] of BROWSERS) {
    if (re.test(ua)) {
      browser = name
      break
    }
  }

  let os = 'Unknown OS'
  for (const [re, name] of OS_PATTERNS) {
    if (re.test(ua)) {
      os = name
      break
    }
  }

  let device: ParsedUserAgent['device'] = 'desktop'
  if (/iPad|Tablet/i.test(ua)) {
    device = 'tablet'
  } else if (/Mobi|Android|iPhone/i.test(ua)) {
    device = 'mobile'
  }

  return { browser, os, device }
}

export function deviceIcon(device: ParsedUserAgent['device']): string {
  switch (device) {
    case 'mobile':
      return '📱'
    case 'tablet':
      return '🖥️'
    case 'desktop':
      return '💻'
    default:
      return '❓'
  }
}
