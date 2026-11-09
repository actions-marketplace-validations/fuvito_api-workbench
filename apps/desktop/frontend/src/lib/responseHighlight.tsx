import { cn } from '@/lib/utils'

// ---- Syntax highlighting ----

export function highlightJSON(json: string): React.ReactNode {
  const nodes: React.ReactNode[] = []
  const re = /("(?:[^"\\]|\\.)*")(?=\s*:)|("(?:[^"\\]|\\.)*")|([-]?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)|(true|false|null)|([{}[\],:])/g
  let last = 0, i = 0
  for (const m of json.matchAll(re)) {
    if (m.index > last) nodes.push(json.slice(last, m.index))
    if      (m[1]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-key)' }}>{m[1]}</span>)
    else if (m[2]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-string)' }}>{m[2]}</span>)
    else if (m[3]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-number)' }}>{m[3]}</span>)
    else if (m[4]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-literal)' }}>{m[4]}</span>)
    else if (m[5]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-punct)' }}>{m[5]}</span>)
    last = m.index + m[0].length
  }
  if (last < json.length) nodes.push(json.slice(last))
  return nodes
}

export function highlightXML(xml: string): React.ReactNode {
  const nodes: React.ReactNode[] = []
  // Order: comments, CDATA, tags (with attrs), entities
  const re = /(<!--[\s\S]*?-->)|(<!\[CDATA\[[\s\S]*?\]\]>)|(<\/?)([a-zA-Z][^\s>/]*)([^>]*)(\/?>)|(&#?[a-zA-Z0-9]+;)/g
  let last = 0, i = 0
  for (const m of xml.matchAll(re)) {
    if (m.index > last) nodes.push(xml.slice(last, m.index))
    if (m[1]) {
      nodes.push(<span key={i++} className="text-[var(--color-muted-foreground)] italic">{m[1]}</span>)
    } else if (m[2]) {
      nodes.push(<span key={i++} className="text-amber-400">{m[2]}</span>)
    } else if (m[4]) {
      // tag: highlight open/close bracket, tag name, attributes, close bracket
      const attrs = m[5] ?? ''
      const attrNodes: React.ReactNode[] = []
      const attrRe = /\s+([a-zA-Z][^\s=]*)(?:=(?:"([^"]*)"|'([^']*)'|([^\s>]*)))?/g
      let al = 0, ai = 0
      for (const am of attrs.matchAll(attrRe)) {
        if (am.index > al) attrNodes.push(attrs.slice(al, am.index))
        attrNodes.push(
          <span key={ai++}>{' '}<span className="text-emerald-400">{am[1]}</span>
          {am[0].includes('=') && <>{'='}<span className="text-amber-400">{`"${am[2] ?? am[3] ?? am[4] ?? ''}"`}</span></>}</span>
        )
        al = am.index + am[0].length
      }
      if (al < attrs.length) attrNodes.push(attrs.slice(al))
      nodes.push(
        <span key={i++}>
          <span className="text-[var(--color-muted-foreground)]">{m[3]}</span>
          <span className="text-blue-400">{m[4]}</span>
          {attrNodes}
          <span className="text-[var(--color-muted-foreground)]">{m[6]}</span>
        </span>
      )
    } else if (m[7]) {
      nodes.push(<span key={i++} className="text-purple-400">{m[7]}</span>)
    }
    last = m.index + m[0].length
  }
  if (last < xml.length) nodes.push(xml.slice(last))
  return nodes
}

export type BodyFormat = 'json' | 'xml' | 'html' | 'plain'

export function detectFormat(body: string, contentType?: string): BodyFormat {
  const ct = contentType?.toLowerCase() ?? ''
  const t = body.trim()
  if (ct.includes('json') || (!ct && (t.startsWith('{') || t.startsWith('[')))) {
    try { JSON.parse(body); return 'json' } catch { /* fall through */ }
  }
  if (ct.includes('xml') || ct.includes('html') || (!ct && t.startsWith('<'))) {
    return ct.includes('html') ? 'html' : 'xml'
  }
  return 'plain'
}

// ---- Search highlight overlay ----

export function highlightSearch(text: string, query: string, activeIdx: number): React.ReactNode {
  if (!query) return text
  const nodes: React.ReactNode[] = []
  const lower = text.toLowerCase()
  const q = query.toLowerCase()
  let pos = 0, matchIdx = 0
  while (pos < text.length) {
    const idx = lower.indexOf(q, pos)
    if (idx === -1) { nodes.push(text.slice(pos)); break }
    if (idx > pos) nodes.push(text.slice(pos, idx))
    nodes.push(
      <mark key={matchIdx} data-match={String(matchIdx)} className={cn('rounded-sm', matchIdx === activeIdx ? 'bg-orange-400 text-black' : 'bg-yellow-300/60 text-inherit')}>
        {text.slice(idx, idx + q.length)}
      </mark>
    )
    matchIdx++
    pos = idx + q.length
  }
  return nodes
}

export function countMatches(text: string, query: string): number {
  if (!query) return 0
  let count = 0, pos = 0
  const lower = text.toLowerCase(), q = query.toLowerCase()
  while ((pos = lower.indexOf(q, pos)) !== -1) { count++; pos += q.length }
  return count
}
