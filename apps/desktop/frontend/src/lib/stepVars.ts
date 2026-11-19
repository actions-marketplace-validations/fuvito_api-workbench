import type { StepDetail, ScenarioVariable } from '@/lib/bridge'

export type EnvVars = Record<string, { value: string; secret: boolean }>

export type VarHint = {
  kind: 'extracted' | 'later' | 'scenario' | 'env' | 'missing'
  // shown after the name: "= value", "extracted in step 2", ...
  text: string
}

export type StepVarInfo = {
  // everything that can be typed after {{ in this step
  names: string[]
  hint: (name: string) => VarHint
}

const SECRET = '••••'
const MAX_VALUE = 40

function short(v: string): string {
  return v.length > MAX_VALUE ? v.slice(0, MAX_VALUE) + '…' : v
}

// Where each {{name}} used in step `index` comes from when the scenario runs. The order follows the
// runner: a value extracted by an earlier step wins over a scenario variable, which wins over the
// environment. A name extracted only by this step or a later one is flagged, because it is not
// set yet when this step is sent.
export function buildStepVarInfo(
  steps: StepDetail[],
  index: number,
  scenarioVars: ScenarioVariable[] | undefined,
  env: EnvVars = {},
): StepVarInfo {
  const earlier = new Map<string, number>()
  const later = new Map<string, number>()
  steps.forEach((s, i) => {
    for (const e of s.extractions ?? []) {
      if (!e.name) continue
      if (i < index) earlier.set(e.name, i + 1)
      else if (!later.has(e.name)) later.set(e.name, i + 1)
    }
  })
  const scenario = new Map((scenarioVars ?? []).filter(v => v.name).map(v => [v.name, v]))

  const names = [...new Set([
    ...scenario.keys(),
    ...earlier.keys(),
    ...Object.keys(env),
  ])]

  return {
    names,
    hint(name) {
      const from = earlier.get(name)
      if (from !== undefined) return { kind: 'extracted', text: `extracted in step ${from}` }
      const sv = scenario.get(name)
      if (sv) return { kind: 'scenario', text: sv.secret ? SECRET : short(sv.value) }
      const ev = env[name]
      if (ev) return { kind: 'env', text: ev.secret ? SECRET : short(ev.value) || '(empty)' }
      const l = later.get(name)
      if (l !== undefined) return { kind: 'later', text: `extracted in step ${l}, not set yet here` }
      return { kind: 'missing', text: 'not defined' }
    },
  }
}

const REF = /\{\{\s*([^}$][^}]*?)\s*\}\}/g

// The {{variables}} in a text, without dynamic {{$functions}}.
export function varsIn(text: string): string[] {
  return [...new Set([...text.matchAll(REF)].map(m => m[1]))]
}
