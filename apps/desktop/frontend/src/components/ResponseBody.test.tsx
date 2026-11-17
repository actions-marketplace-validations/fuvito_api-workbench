import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { BodyContent } from './ResponseBody'

const json = [{ name: 'Content-Type', value: 'application/json' }]
const html = [{ name: 'content-type', value: 'text/html; charset=utf-8' }]

function body(props: Partial<React.ComponentProps<typeof BodyContent>> = {}) {
  return render(<BodyContent body='{"a":1}' headers={json} search="" activeMatch={0} {...props} />)
}

describe('BodyContent', () => {
  it('pretty-prints JSON and shows a line-number gutter', () => {
    const { container } = body()
    const pres = container.querySelectorAll('pre')
    expect(pres).toHaveLength(2)
    expect(pres[0].textContent).toBe('1\n2\n3')
    expect(pres[1].textContent).toBe('{\n  "a": 1\n}')
  })

  it('shows raw text unformatted when requested', () => {
    const { container } = body({ raw: true })
    const pres = container.querySelectorAll('pre')
    expect(pres).toHaveLength(1)
    expect(pres[0].textContent).toBe('{"a":1}')
  })

  it('omits the gutter for single-line bodies', () => {
    const { container } = body({ body: 'plain text', headers: [] })
    expect(container.querySelectorAll('pre')).toHaveLength(1)
    expect(container.textContent).toBe('plain text')
  })

  it('does not count a trailing newline as an extra line', () => {
    const { container } = body({ body: 'a\nb\n', headers: [], raw: true })
    expect(container.querySelectorAll('pre')[0].textContent).toBe('1\n2')
  })

  it('syntax-highlights JSON and HTML', () => {
    const j = body()
    expect(j.container.querySelectorAll('span').length).toBeGreaterThan(0)
    const h = render(<BodyContent body={'<a href="x">hi</a>'} headers={html} search="" activeMatch={0} />)
    expect(h.container.querySelectorAll('span').length).toBeGreaterThan(0)
    expect(h.container.textContent).toContain('<a href="x">hi</a>')
  })

  it('skips highlighting for raw mode and for unknown content types', () => {
    expect(body({ raw: true }).container.querySelectorAll('span')).toHaveLength(0)
    const txt = render(<BodyContent body={'hello\nworld'} headers={[{ name: 'Content-Type', value: 'text/plain' }]} search="" activeMatch={0} />)
    expect(txt.container.querySelectorAll('span')).toHaveLength(0)
  })

  it('highlights search matches case-insensitively and marks only the active one', () => {
    const { container } = render(<BodyContent body={'Foo bar foo'} headers={[]} search="foo" activeMatch={1} />)
    const marks = container.querySelectorAll('mark')
    expect(Array.from(marks).map(m => m.textContent)).toEqual(['Foo', 'foo'])
    expect(marks[0].className).toContain('bg-yellow-300/60')
    expect(marks[1].className).toContain('bg-orange-400')
    expect(container.textContent).toBe('Foo bar foo')
  })

  it('finds the content type header case-insensitively', () => {
    const { container } = render(<BodyContent body='{"a":1}' headers={[{ name: 'CONTENT-TYPE', value: 'application/json' }]} search="" activeMatch={0} />)
    expect(container.querySelectorAll('span').length).toBeGreaterThan(0)
  })
})
