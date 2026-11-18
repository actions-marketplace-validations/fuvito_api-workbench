// The {{$function}} catalog comes from the Go core (so editors never drift from what a run
// accepts); App loads it once. Autocomplete stays a pure function of that list.

export type DynamicFunction = {
  name: string
  usage: string
  description: string
  example: string
}

export type Suggestion = {
  kind: 'variable' | 'function'
  label: string   // what the list shows
  insert: string  // the text placed between {{ and }}
  hint?: string
}

let catalog: DynamicFunction[] = []

export function setDynamicFunctions(list: DynamicFunction[]): void {
  catalog = list
}

export function getDynamicFunctions(): DynamicFunction[] {
  return catalog
}

// partial is the text typed after {{. Variables match anywhere in the name; functions are offered
// right after {{ and as soon as a $ is typed.
export function suggest(partial: string, variableNames: string[] = []): Suggestion[] {
  const q = partial.toLowerCase()
  const out: Suggestion[] = variableNames
    .filter(n => n.toLowerCase().includes(q))
    .map(n => ({ kind: 'variable', label: n, insert: n }))
  if (partial === '' || partial.startsWith('$')) {
    const fq = q.replace(/^\$/, '').split(' ')[0]
    for (const f of catalog) {
      if (f.name.toLowerCase().includes(fq)) {
        out.push({ kind: 'function', label: '$' + f.name, insert: f.example, hint: f.description })
      }
    }
  }
  return out
}
