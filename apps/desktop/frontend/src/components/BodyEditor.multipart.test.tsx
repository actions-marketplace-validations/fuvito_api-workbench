import { describe, it, expect, vi } from 'vitest'
import { useState } from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { BodyEditor } from './BodyEditor'
import type { BodyState } from '@/types'

function Harness({ initial, onPickFile, spy }: { initial: BodyState; onPickFile?: () => Promise<string>; spy?: (b: BodyState) => void }) {
  const [body, setBody] = useState(initial)
  return <BodyEditor body={body} onChange={b => { setBody(b); spy?.(b) }} onPickFile={onPickFile} />
}

const last = (spy: ReturnType<typeof vi.fn>) => spy.mock.calls.at(-1)![0] as BodyState

describe('BodyEditor — multipart', () => {
  it('parses the JSON field format into rows', () => {
    const content = JSON.stringify([{ n: 'title', t: 'text', v: 'hello' }, { n: 'doc', t: 'file', v: '/tmp/a.pdf' }])
    render(<Harness initial={{ type: 'multipart', content }} />)
    expect(screen.getByDisplayValue('title')).toBeInTheDocument()
    expect(screen.getByDisplayValue('hello')).toBeInTheDocument()
    expect(screen.getByDisplayValue('/tmp/a.pdf')).toBeInTheDocument()
    expect(screen.getByTitle('Browse')).toBeInTheDocument()
  })

  it('parses the legacy url-encoded format and skips undecodable pairs', () => {
    render(<Harness initial={{ type: 'multipart', content: 'a=1&b=hello%20world&%E0%A4%A=x&flag' }} />)
    expect(screen.getByDisplayValue('a')).toBeInTheDocument()
    expect(screen.getByDisplayValue('hello world')).toBeInTheDocument()
    expect(screen.getByDisplayValue('flag')).toBeInTheDocument()
  })

  it('falls back to legacy parsing for malformed JSON', () => {
    render(<Harness initial={{ type: 'multipart', content: '[not json' }} />)
    expect(screen.getByDisplayValue('[not json')).toBeInTheDocument()
  })

  it('shows no rows for empty content', () => {
    render(<Harness initial={{ type: 'multipart', content: '' }} />)
    expect(screen.queryByPlaceholderText('name')).not.toBeInTheDocument()
  })

  it('adds a field and serializes only named, enabled fields', () => {
    const spy = vi.fn()
    render(<Harness initial={{ type: 'multipart', content: '' }} spy={spy} />)
    fireEvent.click(screen.getByRole('button', { name: /Add field/ }))
    expect(last(spy).content).toBe('')
    fireEvent.change(screen.getByPlaceholderText('name'), { target: { value: ' title ' } })
    fireEvent.change(screen.getByPlaceholderText('value'), { target: { value: 'hi' } })
    expect(JSON.parse(last(spy).content)).toEqual([{ n: 'title', t: 'text', v: 'hi' }])
    expect(last(spy).type).toBe('multipart')
  })

  it('excludes disabled fields', () => {
    const spy = vi.fn()
    render(<Harness initial={{ type: 'multipart', content: JSON.stringify([{ n: 'a', t: 'text', v: '1' }]) }} spy={spy} />)
    fireEvent.click(screen.getAllByRole('checkbox')[0])
    expect(last(spy).content).toBe('')
  })

  it('removes a field', () => {
    const spy = vi.fn()
    render(<Harness initial={{ type: 'multipart', content: JSON.stringify([{ n: 'a', t: 'text', v: '1' }]) }} spy={spy} />)
    fireEvent.click(screen.getAllByRole('button').find(b => b.querySelector('svg.lucide-x'))!)
    expect(screen.queryByDisplayValue('a')).not.toBeInTheDocument()
    expect(last(spy).content).toBe('')
  })

  it('switching a field to File clears its value and shows the browse control', () => {
    const spy = vi.fn()
    render(<Harness initial={{ type: 'multipart', content: JSON.stringify([{ n: 'a', t: 'text', v: 'old' }]) }} spy={spy} />)
    fireEvent.change(screen.getAllByRole('combobox')[1], { target: { value: 'file' } })
    expect(screen.getByTitle('Browse')).toBeInTheDocument()
    expect(JSON.parse(last(spy).content)).toEqual([{ n: 'a', t: 'file', v: '' }])
  })

  it('fills the path from the file picker', async () => {
    const onPickFile = vi.fn(() => Promise.resolve('/picked/file.txt'))
    render(<Harness initial={{ type: 'multipart', content: JSON.stringify([{ n: 'f', t: 'file', v: '' }]) }} onPickFile={onPickFile} />)
    fireEvent.click(screen.getByTitle('Browse'))
    expect(await screen.findByDisplayValue('/picked/file.txt')).toBeInTheDocument()
  })

  it('keeps the value when the picker is cancelled', async () => {
    const onPickFile = vi.fn(() => Promise.resolve(''))
    render(<Harness initial={{ type: 'multipart', content: JSON.stringify([{ n: 'f', t: 'file', v: '/keep' }]) }} onPickFile={onPickFile} />)
    fireEvent.click(screen.getByTitle('Browse'))
    await waitFor(() => expect(onPickFile).toHaveBeenCalled())
    expect(screen.getByDisplayValue('/keep')).toBeInTheDocument()
  })

  it('works without a picker callback', async () => {
    render(<Harness initial={{ type: 'multipart', content: JSON.stringify([{ n: 'f', t: 'file', v: '/keep' }]) }} />)
    fireEvent.click(screen.getByTitle('Browse'))
    expect(screen.getByDisplayValue('/keep')).toBeInTheDocument()
  })

  it('lets the user type a file path directly', () => {
    const spy = vi.fn()
    render(<Harness initial={{ type: 'multipart', content: JSON.stringify([{ n: 'f', t: 'file', v: '' }]) }} spy={spy} />)
    fireEvent.change(screen.getByPlaceholderText('file path…'), { target: { value: '/x/y' } })
    expect(JSON.parse(last(spy).content)[0].v).toBe('/x/y')
  })
})

describe('BodyEditor — form-urlencoded and type switching', () => {
  it('serializes edited rows with encoding', () => {
    const spy = vi.fn()
    render(<Harness initial={{ type: 'form-urlencoded', content: '' }} spy={spy} />)
    fireEvent.click(screen.getByText('+ Add row'))
    fireEvent.change(screen.getAllByRole('textbox')[0], { target: { value: 'a b' } })
    fireEvent.change(screen.getAllByRole('textbox')[1], { target: { value: 'x&y' } })
    expect(last(spy)).toEqual({ type: 'form-urlencoded', content: 'a%20b=x%26y' })
  })

  it('parses existing content into rows', () => {
    render(<Harness initial={{ type: 'form-urlencoded', content: 'name=alice&bad=%E0%A4%A&role=admin&solo' }} />)
    expect(screen.getByDisplayValue('alice')).toBeInTheDocument()
    expect(screen.getByDisplayValue('admin')).toBeInTheDocument()
    expect(screen.getByDisplayValue('solo')).toBeInTheDocument()
  })

  it('re-parses content when switching to form-urlencoded and multipart', () => {
    const spy = vi.fn()
    render(<Harness initial={{ type: 'raw', content: 'k=v' }} spy={spy} />)
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'form-urlencoded' } })
    expect(screen.getByDisplayValue('k')).toBeInTheDocument()
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'multipart' } })
    expect(screen.getByDisplayValue('k')).toBeInTheDocument()
  })

  it('leaves invalid JSON untouched when formatting', () => {
    const spy = vi.fn()
    render(<Harness initial={{ type: 'json', content: '{bad' }} spy={spy} />)
    fireEvent.click(screen.getByText('Format'))
    expect(spy).not.toHaveBeenCalled()
  })

  it('edits raw content through the textarea', () => {
    const spy = vi.fn()
    render(<Harness initial={{ type: 'raw', content: '' }} spy={spy} />)
    fireEvent.change(screen.getByPlaceholderText('Body content…'), { target: { value: 'hello' } })
    expect(last(spy)).toEqual({ type: 'raw', content: 'hello' })
  })
})
