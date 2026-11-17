export function parseOp(op: string): { method: string; path: string } {
  const idx = op.indexOf(' ')
  if (idx === -1) return { method: op, path: '' }
  return { method: op.slice(0, idx), path: op.slice(idx + 1) }
}

export function specLabel(specFile: string): string {
  return specFile.replace(/\\/g, '/').split('/').pop() ?? specFile
}
