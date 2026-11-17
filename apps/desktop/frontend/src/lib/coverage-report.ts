import type { AccumulatedCoverage, QualityReport } from '@/lib/bridge'

export function parseOp(op: string): { method: string; path: string } {
  const idx = op.indexOf(' ')
  if (idx === -1) return { method: op, path: '' }
  return { method: op.slice(0, idx), path: op.slice(idx + 1) }
}

export function specLabel(specFile: string): string {
  return specFile.replace(/\\/g, '/').split('/').pop() ?? specFile
}

export function csvCell(v: string): string {
  if (v.includes(',') || v.includes('"') || v.includes('\n')) {
    return '"' + v.replace(/"/g, '""') + '"'
  }
  return v
}

export function qualityColumns(
  op: string,
  covered: boolean,
  q: QualityReport | undefined,
): [string, string] {
  if (!covered) return ['Never exercised', '']
  if (!q) return ['', '']
  const happySet  = new Set(q.happyPathOnly)
  const authSet   = new Set(q.securedNeverAuthed)
  const errorMap  = new Map(q.errorsNeverTriggered.map(g => [g.operation, g.missing]))
  const flags: string[] = []
  if (happySet.has(op))  flags.push('Happy-path only')
  if (authSet.has(op))   flags.push('No auth tested')
  if (errorMap.has(op))  flags.push('Errors never triggered')
  return [flags.join('; '), (errorMap.get(op) ?? []).join(' ')]
}

export function buildCsvRows(entry: AccumulatedCoverage): string[] {
  const q = entry.quality
  const rows = ['Method,Path,Coverage,Last Result,Last Run,Quality,Missing Codes']
  const allOps = [...(entry.exercised ?? []), ...(entry.missed ?? [])].sort((a, b) => {
    const pa = parseOp(a), pb = parseOp(b)
    return pa.path !== pb.path ? pa.path.localeCompare(pb.path) : pa.method.localeCompare(pb.method)
  })
  for (const op of allOps) {
    const { method, path } = parseOp(op)
    const covered = (entry.exercised ?? []).includes(op)
    const r = entry.operationResults?.[op]
    const result = covered ? (r ? (r.lastPassed ? 'Pass' : 'Fail') : '') : ''
    const runAt  = r?.lastRunAt ? new Date(r.lastRunAt).toISOString() : ''
    const [quality, missing] = qualityColumns(op, covered, q)
    rows.push([method, path, covered ? 'Covered' : 'Not Covered', result, runAt, quality, missing].map(csvCell).join(','))
  }
  return rows
}

export function generateMarkdownReport(specFile: string, entry: AccumulatedCoverage, now = new Date()): string {
  const label = specLabel(specFile)
  const pct = Math.round(entry.percent)
  const lines: string[] = []

  lines.push(`# Contracts Report — ${label}`)
  lines.push(``)
  lines.push(`Generated: ${now.toLocaleString()}`)
  lines.push(``)
  lines.push(`## Summary`)
  lines.push(``)
  lines.push(`**${entry.covered} / ${entry.total} operations covered (${pct}%)**`)
  lines.push(``)
  lines.push(`| Metric | Value |`)
  lines.push(`|--------|-------|`)
  lines.push(`| Total runs | ${entry.totalRuns} |`)
  lines.push(`| Passed runs | ${entry.passedRuns} |`)
  lines.push(`| Failed runs | ${entry.failedRuns} |`)
  if (entry.lastRunAt) lines.push(`| Last run | ${new Date(entry.lastRunAt).toLocaleString()} |`)
  lines.push(``)
  lines.push(`## Operations`)
  lines.push(``)
  lines.push(`| Method | Path | Coverage | Last Result | Last Run |`)
  lines.push(`|--------|------|----------|-------------|---------|`)

  const allOps = [...(entry.exercised ?? []), ...(entry.missed ?? [])].sort((a, b) => {
    const pa = parseOp(a), pb = parseOp(b)
    return pa.path !== pb.path ? pa.path.localeCompare(pb.path) : pa.method.localeCompare(pb.method)
  })
  for (const op of allOps) {
    const { method, path } = parseOp(op)
    const covered = (entry.exercised ?? []).includes(op)
    const r = entry.operationResults?.[op]
    const result = r ? (r.lastPassed ? 'Pass' : 'Fail') : '—'
    const runAt = r?.lastRunAt ? relTimeFrom(r.lastRunAt, now) : '—'
    lines.push(`| \`${method}\` | \`${path}\` | ${covered ? '✅ Covered' : '—'} | ${result} | ${runAt} |`)
  }

  const q = entry.quality
  if (q) {
    const hasGaps = q.happyPathOnly.length + q.securedNeverAuthed.length + q.errorsNeverTriggered.length > 0
    if (hasGaps) {
      lines.push(``)
      lines.push(`## Quality Gaps`)

      if (q.happyPathOnly.length > 0) {
        lines.push(``)
        lines.push(`### Happy-path only (${q.happyPathOnly.length})`)
        lines.push(``)
        lines.push(`Only 2xx responses observed — declared error paths untested.`)
        lines.push(``)
        for (const op of q.happyPathOnly) lines.push(`- \`${op}\``)
      }

      if (q.errorsNeverTriggered.length > 0) {
        lines.push(``)
        lines.push(`### Error responses never triggered (${q.errorsNeverTriggered.length})`)
        lines.push(``)
        lines.push(`| Operation | Missing status codes |`)
        lines.push(`|-----------|---------------------|`)
        for (const gap of q.errorsNeverTriggered) {
          lines.push(`| \`${gap.operation}\` | ${gap.missing.join(', ')} |`)
        }
      }

      if (q.securedNeverAuthed.length > 0) {
        lines.push(``)
        lines.push(`### Secured — never tested with auth (${q.securedNeverAuthed.length})`)
        lines.push(``)
        lines.push(`Operations that require authentication but were always called without credentials.`)
        lines.push(``)
        for (const op of q.securedNeverAuthed) lines.push(`- \`${op}\``)
      }
    } else {
      lines.push(``)
      lines.push(`## Quality Gaps`)
      lines.push(``)
      lines.push(`_All quality signals clear._`)
    }
  }

  return lines.join('\n')
}

function relTimeFrom(iso: string, now: Date): string {
  const s = Math.floor((now.getTime() - new Date(iso).getTime()) / 1000)
  if (s < 60) return 'just now'
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}
