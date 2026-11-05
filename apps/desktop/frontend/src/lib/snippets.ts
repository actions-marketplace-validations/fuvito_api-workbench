import type { TabState } from './tabState'

export type SnippetFormat = 'curl' | 'fetch' | 'python'

export const SNIPPET_FORMATS: Array<{ key: SnippetFormat; label: string }> = [
  { key: 'curl',   label: 'curl' },
  { key: 'fetch',  label: 'JavaScript · fetch' },
  { key: 'python', label: 'Python · requests' },
]

export function buildCurl(tab: TabState): string {
  const parts: string[] = [`curl -X ${tab.method}`]

  let url = tab.url.trim()
  const qp = [...tab.queryParams.filter(p => p.enabled && p.name.trim())]
  if (tab.auth.type === 'apikey' && tab.auth.location === 'query') {
    qp.push({ id: '', name: tab.auth.key, value: tab.auth.value, enabled: true })
  }
  if (qp.length) {
    url += (url.includes('?') ? '&' : '?') +
      qp.map(p => `${encodeURIComponent(p.name)}=${encodeURIComponent(p.value)}`).join('&')
  }
  parts.push(`"${url}"`)

  if (tab.auth.type === 'bearer') parts.push(`-H "Authorization: Bearer ${tab.auth.token}"`)
  else if (tab.auth.type === 'basic') parts.push(`--user "${tab.auth.username}:${tab.auth.password}"`)
  else if (tab.auth.type === 'apikey' && tab.auth.location === 'header') parts.push(`-H "${tab.auth.key}: ${tab.auth.value}"`)

  for (const h of tab.headers.filter(h => h.enabled && h.name.trim())) {
    parts.push(`-H "${h.name}: ${h.value.replace(/"/g, '\\"')}"`)
  }

  if (tab.body.type !== 'none' && tab.body.content.trim()) {
    const ct = tab.body.type === 'json' ? 'application/json'
              : tab.body.type === 'form-urlencoded' ? 'application/x-www-form-urlencoded'
              : 'text/plain'
    parts.push(`-H "Content-Type: ${ct}"`)
    parts.push(`--data '${tab.body.content.replace(/'/g, "'\\''")}'`)
  }

  return parts.join(' \\\n  ')
}

export function buildFetch(tab: TabState): string {
  const headers: Record<string, string> = {}
  for (const h of tab.headers.filter(h => h.enabled && h.name.trim())) headers[h.name] = h.value
  if (tab.auth.type === 'bearer') headers['Authorization'] = `Bearer ${tab.auth.token}`
  else if (tab.auth.type === 'basic') headers['Authorization'] = `Basic ${btoa(`${tab.auth.username}:${tab.auth.password}`)}`
  else if (tab.auth.type === 'apikey' && tab.auth.location === 'header') headers[tab.auth.key] = tab.auth.value
  if (tab.body.type === 'json') headers['Content-Type'] = 'application/json'
  else if (tab.body.type === 'form-urlencoded') headers['Content-Type'] = 'application/x-www-form-urlencoded'

  let url = tab.url.trim()
  const qp = [...tab.queryParams.filter(p => p.enabled && p.name.trim())]
  if (tab.auth.type === 'apikey' && tab.auth.location === 'query') {
    qp.push({ id: '', name: tab.auth.key, value: tab.auth.value, enabled: true })
  }
  if (qp.length) {
    url += (url.includes('?') ? '&' : '?') +
      qp.map(p => `${encodeURIComponent(p.name)}=${encodeURIComponent(p.value)}`).join('&')
  }

  const opts: string[] = [`  method: '${tab.method}'`]
  if (Object.keys(headers).length) {
    const hLines = Object.entries(headers).map(([k, v]) => `    '${k}': '${v.replace(/'/g, "\\'")}'`).join(',\n')
    opts.push(`  headers: {\n${hLines}\n  }`)
  }
  if (tab.body.type !== 'none' && tab.body.content.trim()) opts.push(`  body: ${JSON.stringify(tab.body.content)}`)

  return `const response = await fetch('${url}', {\n${opts.join(',\n')}\n});\nconst data = await response.json();`
}

export function buildPython(tab: TabState): string {
  const headers: Record<string, string> = {}
  for (const h of tab.headers.filter(h => h.enabled && h.name.trim())) headers[h.name] = h.value
  if (tab.auth.type === 'bearer') headers['Authorization'] = `Bearer ${tab.auth.token}`
  else if (tab.auth.type === 'apikey' && tab.auth.location === 'header') headers[tab.auth.key] = tab.auth.value
  if (tab.body.type === 'json') headers['Content-Type'] = 'application/json'

  const params: Record<string, string> = {}
  for (const p of tab.queryParams.filter(p => p.enabled && p.name.trim())) params[p.name] = p.value
  if (tab.auth.type === 'apikey' && tab.auth.location === 'query') params[tab.auth.key] = tab.auth.value

  const args: string[] = [`    '${tab.url.trim()}'`]
  if (Object.keys(headers).length) args.push(`    headers=${JSON.stringify(headers, null, 4).replace(/\n/g, '\n    ')}`)
  if (Object.keys(params).length) args.push(`    params=${JSON.stringify(params, null, 4).replace(/\n/g, '\n    ')}`)
  if (tab.body.type !== 'none' && tab.body.content.trim()) {
    args.push(tab.body.type === 'json' ? `    json=${tab.body.content}` : `    data=${JSON.stringify(tab.body.content)}`)
  }
  if (tab.auth.type === 'basic') args.push(`    auth=('${tab.auth.username}', '${tab.auth.password}')`)

  return `import requests\n\nresponse = requests.${tab.method.toLowerCase()}(\n${args.join(',\n')}\n)\nresponse.raise_for_status()\ndata = response.json()`
}
