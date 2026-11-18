import type { StepDetail, ScenarioVariable } from '@/lib/bridge'

export type MissingValue = { name: string; extractedIn: number[]; usedIn: number[] }

const REF = /\{\{\s*([A-Za-z_][\w.-]*)\s*\}\}/g

function texts(step: StepDetail): string[] {
  const out: string[] = [step.url ?? '', step.bodyContent ?? '']
  for (const h of step.headers ?? []) out.push(h.name, h.value)
  for (const q of step.queryParams ?? []) out.push(q.name, q.value)
  for (const a of step.assertions ?? []) {
    if (typeof a.expected === 'string') out.push(a.expected)
    if (a.path) out.push(a.path)
  }
  for (const e of step.extractions ?? []) if (e.path) out.push(e.path)
  return out
}

function refs(step: StepDetail): Set<string> {
  const found = new Set<string>()
  for (const t of texts(step)) for (const m of t.matchAll(REF)) found.add(m[1])
  return found
}

// Values that a run starting at `from` would leave unset: extracted by a step before `from` (which
// is skipped) yet referenced by a step that still runs. Scenario variables are always defined, and a
// step at or after `from` that extracts the same name first makes it available again.
export function missingOnRunFrom(steps: StepDetail[], variables: ScenarioVariable[] | undefined, from: number): MissingValue[] {
  if (from <= 0) return []
  const defined = new Set((variables ?? []).map(v => v.name))
  const extractedIn = new Map<string, number[]>()
  steps.forEach((s, i) => {
    if (i >= from) return
    for (const e of s.extractions ?? []) {
      if (!e.name || defined.has(e.name)) continue
      extractedIn.set(e.name, [...(extractedIn.get(e.name) ?? []), i + 1])
    }
  })
  if (extractedIn.size === 0) return []

  const available = new Set<string>()
  const used = new Map<string, number[]>()
  for (let i = from; i < steps.length; i++) {
    const s = steps[i]
    if (s.disabled) continue
    for (const name of refs(s)) {
      if (extractedIn.has(name) && !available.has(name)) used.set(name, [...(used.get(name) ?? []), i + 1])
    }
    for (const e of s.extractions ?? []) if (e.name) available.add(e.name)
  }
  return [...used.entries()].map(([name, usedIn]) => ({ name, extractedIn: extractedIn.get(name) ?? [], usedIn }))
}

export function describeMissing(missing: MissingValue[]): string {
  const names = missing.map(m => `{{${m.name}}}`).join(', ')
  return `Starting here skips earlier steps, so ${names} ${missing.length === 1 ? 'is' : 'are'} not set.`
}
