import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import React from 'react'
import { detectFormat, countMatches, highlightJSON, highlightXML, highlightSearch } from './responseHighlight'

// ---- detectFormat ----

describe('detectFormat', () => {
  it('returns json for application/json content-type', () => {
    expect(detectFormat('{"a":1}', 'application/json')).toBe('json')
  })

  it('returns json when body starts with { and no content-type', () => {
    expect(detectFormat('{"a":1}')).toBe('json')
  })

  it('returns json when body starts with [ and no content-type', () => {
    expect(detectFormat('[1,2,3]')).toBe('json')
  })

  it('returns plain if body looks like JSON but is invalid', () => {
    expect(detectFormat('{not valid json}')).toBe('plain')
  })

  it('returns xml for application/xml content-type', () => {
    expect(detectFormat('<root/>', 'application/xml')).toBe('xml')
  })

  it('returns html for text/html content-type', () => {
    expect(detectFormat('<html/>', 'text/html; charset=utf-8')).toBe('html')
  })

  it('returns xml when body starts with < and no content-type', () => {
    expect(detectFormat('<root/>')).toBe('xml')
  })

  it('returns plain for unrecognised content', () => {
    expect(detectFormat('hello world')).toBe('plain')
  })

  it('prefers content-type over body shape for json', () => {
    expect(detectFormat('<not-xml>', 'application/json; charset=utf-8')).toBe('plain')
  })
})

// ---- countMatches ----

describe('countMatches', () => {
  it('returns 0 for empty query', () => {
    expect(countMatches('hello world', '')).toBe(0)
  })

  it('returns 0 when no matches', () => {
    expect(countMatches('hello world', 'xyz')).toBe(0)
  })

  it('counts single match', () => {
    expect(countMatches('hello world', 'world')).toBe(1)
  })

  it('counts multiple matches', () => {
    expect(countMatches('abcabcabc', 'abc')).toBe(3)
  })

  it('is case-insensitive', () => {
    expect(countMatches('Hello HELLO hello', 'hello')).toBe(3)
  })

  it('handles overlapping non-overlapping matches correctly', () => {
    expect(countMatches('aaa', 'aa')).toBe(1)
  })
})

// ---- highlightJSON ----

describe('highlightJSON', () => {
  function renderNodes(nodes: React.ReactNode) {
    const { container } = render(<pre>{nodes}</pre>)
    return container
  }

  it('wraps JSON keys in a span', () => {
    const nodes = highlightJSON('{"name": "Alice"}')
    const el = renderNodes(nodes)
    const spans = el.querySelectorAll('span')
    const keySpan = Array.from(spans).find(s => s.textContent === '"name"')
    expect(keySpan).toBeTruthy()
    expect(keySpan!.style.color).toBe('var(--color-json-key)')
  })

  it('wraps string values in a span', () => {
    const nodes = highlightJSON('{"name": "Alice"}')
    const el = renderNodes(nodes)
    const spans = el.querySelectorAll('span')
    const valSpan = Array.from(spans).find(s => s.textContent === '"Alice"')
    expect(valSpan).toBeTruthy()
    expect(valSpan!.style.color).toBe('var(--color-json-string)')
  })

  it('wraps numbers in a span', () => {
    const nodes = highlightJSON('{"n": 42}')
    const el = renderNodes(nodes)
    const spans = el.querySelectorAll('span')
    const numSpan = Array.from(spans).find(s => s.textContent === '42')
    expect(numSpan).toBeTruthy()
    expect(numSpan!.style.color).toBe('var(--color-json-number)')
  })

  it('wraps true/false/null in a span', () => {
    const nodes = highlightJSON('{"ok": true, "x": null}')
    const el = renderNodes(nodes)
    const spans = el.querySelectorAll('span')
    expect(Array.from(spans).some(s => s.textContent === 'true')).toBe(true)
    expect(Array.from(spans).some(s => s.textContent === 'null')).toBe(true)
  })

  it('handles plain string input without throwing', () => {
    expect(() => highlightJSON('not json at all')).not.toThrow()
  })
})

// ---- highlightXML ----

describe('highlightXML', () => {
  function renderNodes(nodes: React.ReactNode) {
    const { container } = render(<pre>{nodes}</pre>)
    return container
  }

  it('highlights a tag name in blue', () => {
    const nodes = highlightXML('<root/>')
    const el = renderNodes(nodes)
    const spans = el.querySelectorAll('span')
    expect(Array.from(spans).some(s => s.textContent === 'root' && s.className.includes('text-blue-400'))).toBe(true)
  })

  it('highlights an XML comment in muted italic', () => {
    const nodes = highlightXML('<!-- comment -->')
    const el = renderNodes(nodes)
    const spans = el.querySelectorAll('span')
    const comment = Array.from(spans).find(s => s.textContent === '<!-- comment -->')
    expect(comment).toBeTruthy()
    expect(comment!.className).toContain('italic')
  })

  it('handles plain text without throwing', () => {
    expect(() => highlightXML('no tags here')).not.toThrow()
  })
})

// ---- highlightSearch ----

describe('highlightSearch', () => {
  function renderNodes(nodes: React.ReactNode) {
    const { container } = render(<pre>{nodes}</pre>)
    return container
  }

  it('returns plain string when query is empty', () => {
    const result = highlightSearch('hello world', '', 0)
    expect(result).toBe('hello world')
  })

  it('wraps match in a mark element', () => {
    const nodes = highlightSearch('hello world', 'world', 0)
    const el = renderNodes(nodes)
    const marks = el.querySelectorAll('mark')
    expect(marks).toHaveLength(1)
    expect(marks[0].textContent).toBe('world')
  })

  it('marks active match with orange background class', () => {
    const nodes = highlightSearch('one two one', 'one', 0)
    const el = renderNodes(nodes)
    const marks = el.querySelectorAll('mark')
    expect(marks[0].className).toContain('bg-orange-400')
    expect(marks[1].className).not.toContain('bg-orange-400')
  })

  it('marks non-active matches with yellow background class', () => {
    const nodes = highlightSearch('one two one', 'one', 0)
    const el = renderNodes(nodes)
    const marks = el.querySelectorAll('mark')
    expect(marks[1].className).toContain('bg-yellow-300')
  })

  it('is case-insensitive', () => {
    const nodes = highlightSearch('Hello HELLO', 'hello', 0)
    const el = renderNodes(nodes)
    expect(el.querySelectorAll('mark')).toHaveLength(2)
  })

  it('handles query not found', () => {
    const nodes = highlightSearch('hello world', 'xyz', 0)
    const el = renderNodes(nodes)
    expect(el.querySelectorAll('mark')).toHaveLength(0)
  })
})
