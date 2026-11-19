export function parseOp(op: string): { method: string; path: string } {
  const idx = op.indexOf(' ')
  if (idx === -1) return { method: op, path: '' }
  return { method: op.slice(0, idx), path: op.slice(idx + 1) }
}

export function specLabel(specFile: string): string {
  return specFile.replace(/\\/g, '/').split('/').pop() ?? specFile
}

export type OperationOutcomes = {
  total: number
  covered: number
  passing: number
  failing: number
  // covered, but the last result was not recorded (older data)
  noResult: number
  notCovered: number
}

type OutcomeSource = {
  total: number
  exercised?: string[]
  missed?: string[]
  operationResults?: Record<string, { lastPassed: boolean }>
}

// Where every operation of the spec stands: passing, failing (by its last result) or not covered.
// passing + failing + noResult = covered, and covered + notCovered = total.
export function operationOutcomes(e: OutcomeSource): OperationOutcomes {
  const exercised = e.exercised ?? []
  let passing = 0, failing = 0, noResult = 0
  for (const op of exercised) {
    const r = e.operationResults?.[op]
    if (!r) noResult++
    else if (r.lastPassed) passing++
    else failing++
  }
  const covered = exercised.length
  return { total: e.total, covered, passing, failing, noResult, notCovered: e.missed?.length ?? Math.max(e.total - covered, 0) }
}

export function share(count: number, total: number): number {
  return total > 0 ? Math.round((count / total) * 100) : 0
}
