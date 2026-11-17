import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TimingBar, TimingDialog } from './ResponseTiming'
import type { TimingData } from '@/lib/bridge'

const timing: TimingData = { dnsMs: 10, connectMs: 20, tlsMs: 0, ttfbMs: 60, transferMs: 10 }
const zero: TimingData = { dnsMs: 0, connectMs: 0, tlsMs: 0, ttfbMs: 0, transferMs: 0 }

describe('TimingBar', () => {
  it('renders nothing without timing or with a zero total', () => {
    const { container, rerender } = render(<TimingBar timing={undefined as never} total={100} onClick={vi.fn()} />)
    expect(container).toBeEmptyDOMElement()
    rerender(<TimingBar timing={timing} total={0} onClick={vi.fn()} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('says so when no phase has any duration', () => {
    render(<TimingBar timing={zero} total={50} onClick={vi.fn()} />)
    expect(screen.getByText('No timing breakdown available.')).toBeInTheDocument()
  })

  it('shows a legend entry per non-zero phase only', () => {
    render(<TimingBar timing={timing} total={100} onClick={vi.fn()} />)
    expect(screen.getByText('DNS 10ms')).toBeInTheDocument()
    expect(screen.getByText('Connect 20ms')).toBeInTheDocument()
    expect(screen.getByText('TTFB 60ms')).toBeInTheDocument()
    expect(screen.getByText('Transfer 10ms')).toBeInTheDocument()
    expect(screen.queryByText(/TLS/)).not.toBeInTheDocument()
  })

  it('sizes segments proportionally and always totals 100%', () => {
    const { container } = render(<TimingBar timing={timing} total={100} onClick={vi.fn()} />)
    const widths = Array.from(container.querySelectorAll<HTMLElement>('[title$="ms"]')).map(e => parseFloat(e.style.width))
    expect(widths).toHaveLength(4)
    expect(widths.reduce((a, b) => a + b, 0)).toBeCloseTo(100, 5)
    expect(widths[2]).toBeGreaterThan(widths[0])
  })

  it('gives tiny phases a visible minimum width', () => {
    const { container } = render(<TimingBar timing={{ ...zero, dnsMs: 1, ttfbMs: 10000 }} total={10001} onClick={vi.fn()} />)
    const [dns] = Array.from(container.querySelectorAll<HTMLElement>('[title$="ms"]')).map(e => parseFloat(e.style.width))
    expect(dns).toBeGreaterThanOrEqual(0.9)
  })

  it('is clickable', () => {
    const onClick = vi.fn()
    render(<TimingBar timing={timing} total={100} onClick={onClick} />)
    fireEvent.click(screen.getByTitle('Click to view timing breakdown'))
    expect(onClick).toHaveBeenCalledTimes(1)
  })
})

describe('TimingDialog', () => {
  it('lists each non-zero phase with its duration and share, plus the total', () => {
    render(<TimingDialog timing={timing} total={123} onClose={vi.fn()} />)
    expect(screen.getByText('Request Timing')).toBeInTheDocument()
    expect(screen.getByText('60 ms')).toBeInTheDocument()
    expect(screen.getByText('60.0%')).toBeInTheDocument()
    expect(screen.getByText('123 ms')).toBeInTheDocument()
    expect(screen.queryByText('TLS')).not.toBeInTheDocument()
  })

  it('does not divide by zero when every phase is zero', () => {
    render(<TimingDialog timing={zero} total={5} onClose={vi.fn()} />)
    expect(screen.getByText('5 ms')).toBeInTheDocument()
  })

  it('closes via the close button, Escape and a backdrop click, but not an inner click', () => {
    const onClose = vi.fn()
    const { container } = render(<TimingDialog timing={timing} total={100} onClose={onClose} />)
    fireEvent.click(screen.getByText('Request Timing'))
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.keyDown(document, { key: 'Enter' })
    expect(onClose).not.toHaveBeenCalled()

    fireEvent.keyDown(document, { key: 'Escape' })
    fireEvent.click(container.firstElementChild!)
    fireEvent.click(screen.getByRole('button'))
    expect(onClose).toHaveBeenCalledTimes(3)
  })
})
