import type { ScenarioDetail } from '@/lib/bridge'

function yamlScalar(s: string): string {
  if (s === '') return '""'
  if (
    /[:#\[\]{},&*!|>'"%@`\n\r\t]/.test(s) ||
    s.startsWith(' ') || s.endsWith(' ') ||
    s === 'true' || s === 'false' || s === 'null'
  ) return JSON.stringify(s)
  return s
}

function yamlValue(v: unknown): string {
  if (v === null || v === undefined) return 'null'
  if (typeof v === 'number' || typeof v === 'boolean') return String(v)
  return yamlScalar(String(v))
}

export function serializeScenarioToYAML(d: ScenarioDetail): string {
  const lines: string[] = ['schemaVersion: "1"', `title: ${yamlScalar(d.title)}`]
  if (d.description) lines.push(`description: ${yamlScalar(d.description)}`)
  if (!d.steps.length) {
    lines.push('steps: []')
  } else {
    lines.push('steps:')
    for (const step of d.steps) {
      lines.push(`  - title: ${yamlScalar(step.title)}`)
      if (step.description) lines.push(`    description: ${yamlScalar(step.description)}`)
      if (step.disabled) lines.push('    disabled: true')
      if (step.requestRef) {
        lines.push(`    requestRef: ${yamlScalar(step.requestRef)}`)
      } else if (step.method || step.url) {
        lines.push('    request:')
        if (step.method) lines.push(`      method: ${step.method}`)
        if (step.url) lines.push(`      url: ${yamlScalar(step.url)}`)
        if (step.headers?.length) {
          lines.push('      headers:')
          for (const h of step.headers) {
            lines.push(`        - name: ${yamlScalar(h.name)}`)
            lines.push(`          value: ${yamlScalar(h.value)}`)
          }
        }
        if (step.queryParams?.length) {
          lines.push('      queryParams:')
          for (const q of step.queryParams) {
            lines.push(`        - name: ${yamlScalar(q.name)}`)
            lines.push(`          value: ${yamlScalar(q.value)}`)
          }
        }
        if (step.bodyType && step.bodyType !== 'none') {
          lines.push('      body:')
          lines.push(`        type: ${step.bodyType}`)
          if (step.bodyContent) lines.push(`        content: ${yamlScalar(step.bodyContent)}`)
        }
      }
      if (step.failPolicy && step.failPolicy !== 'stop') {
        lines.push(`    failPolicy: ${step.failPolicy}`)
      }
      if (step.timeoutMs && step.timeoutMs > 0) {
        lines.push(`    timeoutMs: ${step.timeoutMs}`)
      }
      if (step.retryMaxAttempts && step.retryMaxAttempts > 0) {
        lines.push('    retry:')
        lines.push(`      maxAttempts: ${step.retryMaxAttempts}`)
        if (step.retryDelayMs && step.retryDelayMs > 0) {
          lines.push(`      delayMs: ${step.retryDelayMs}`)
        }
      }
      if (step.assertions?.length) {
        lines.push('    assertions:')
        for (const a of step.assertions) {
          lines.push(`      - type: ${a.type}`)
          if (a.path) lines.push(`        path: ${yamlScalar(a.path)}`)
          lines.push(`        operator: ${a.operator ?? 'eq'}`)
          if (a.expected !== undefined && a.expected !== null) {
            lines.push(`        expected: ${yamlValue(a.expected)}`)
          }
          if (a.schema) lines.push(`        schema: ${yamlScalar(a.schema)}`)
        }
      }
      if (step.extractions?.length) {
        lines.push('    extractions:')
        for (const e of step.extractions) {
          lines.push(`      - name: ${yamlScalar(e.name)}`)
          lines.push(`        source: ${e.source}`)
          if (e.path) lines.push(`        path: ${yamlScalar(e.path)}`)
        }
      }
    }
  }
  return lines.join('\n') + '\n'
}
