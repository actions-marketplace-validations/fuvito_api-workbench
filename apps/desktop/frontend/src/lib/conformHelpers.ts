import type { ConformFinding } from '@/lib/bridge'

// Display only: the Go service makes the real decision and refuses an unconfirmed remote URL.
export function isLocalUrl(raw: string): boolean {
  try {
    const host = new URL(raw).hostname.toLowerCase()
    return host === 'localhost' || host === '127.0.0.1' || host === '[::1]' || host === '::1'
  } catch {
    return false
  }
}

// Where a finding is, in words: `response header "X-Total-Count"`, `response body, field price`,
// `response status 500`. Empty when the finding names no place.
export function findingWhere(f: Pick<ConformFinding, 'part' | 'where'>): string {
  const part = f.part ?? ''
  const where = f.where ?? ''
  if (!part && !where) return ''
  if (!where) return part
  if (part === 'response body') return `${part}, field ${where}`
  if (part === 'response header') return `${part} "${where}"`
  return `${part} ${where}`.trim()
}

// One name=value per line, blank lines ignored: the page's form of `clrsig conform --var`.
export function parseVars(text: string): { vars: string[]; bad: string[] } {
  const vars: string[] = []
  const bad: string[] = []
  for (const raw of text.split('\n')) {
    const line = raw.trim()
    if (!line) continue
    const eq = line.indexOf('=')
    if (eq > 0 && line.slice(0, eq).trim() !== '') vars.push(line)
    else bad.push(line)
  }
  return { vars, bad }
}

export function splitList(text: string): string[] {
  return text.split(',').map(t => t.trim()).filter(Boolean)
}

// Reads the "/prefix=milliseconds" lines of the per-path response-time limits; `bad` lists the lines
// that do not fit, so the page can say which.
export function parsePathLimits(text: string): { lines: string[]; bad: string[] } {
  const lines: string[] = []
  const bad: string[] = []
  for (const raw of text.split(/\r?\n/)) {
    const line = raw.trim()
    if (!line) continue
    if (/^\/\S*=\d+$/.test(line)) lines.push(line)
    else bad.push(line)
  }
  return { lines, bad }
}

// The values of the page's current choices, as the CLI Command Builder takes them for `clrsig
// conform`. Read-only is on unless writes were ticked, so the command does what the page does.
export function cliValues(p: {
  specPath: string; baseUrl: string; envPath: string; activeEnvironment: string
  tags: string[]; paths: string[]; negative: boolean; failFast: boolean; failOnWarning?: boolean; setupPath?: string; timeoutText: string; maxResponseText?: string; allowWrites: boolean
}): Record<string, string> {
  const v: Record<string, string> = {}
  if (p.specPath) v['spec'] = p.specPath
  if (p.baseUrl.trim()) v['base-url'] = p.baseUrl.trim()
  const env = p.envPath || p.activeEnvironment
  if (env) v['env'] = env
  if (p.tags.length) v['tags'] = p.tags.join(',')
  if (p.paths.length) v['paths'] = p.paths.join(',')
  if (p.negative) v['negative'] = 'true'
  if (p.failFast) v['fail-fast'] = 'true'
  if (p.failOnWarning) v['fail-on'] = 'warning'
  if (p.setupPath) v['setup'] = p.setupPath
  const timeout = Number.parseInt(p.timeoutText, 10)
  if (timeout > 0) v['request-timeout'] = String(timeout)
  const slowest = Number.parseInt(p.maxResponseText ?? '', 10)
  if (slowest > 0) v['max-response-time'] = String(slowest)
  if (!p.allowWrites) v['read-only'] = 'true'
  return v
}
