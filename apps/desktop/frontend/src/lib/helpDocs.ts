// Pulls the scenario file out of the example document: the fenced block whose first line is "# file: scenarios/…".
export function extractScenarioYaml(markdown: string): string | null {
  const m = markdown.replace(/\r\n/g, '\n').match(/```yaml\n# file: scenarios\/[^\n]*\n([\s\S]*?)```/)
  return m ? m[1].trimEnd() + '\n' : null
}

// File name (without folders) of that scenario block, taken from its "# file: scenarios/…" line.
export function extractScenarioFileName(markdown: string, fallback = 'scenario.yaml'): string {
  const m = markdown.replace(/\r\n/g, '\n').match(/```yaml\n# file: scenarios\/([^\n]+)/)
  if (!m) return fallback
  const name = m[1].trim().split('/').pop()
  return name || fallback
}
