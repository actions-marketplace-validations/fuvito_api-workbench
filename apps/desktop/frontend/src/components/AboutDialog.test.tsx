import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { AboutDialog } from './AboutDialog'

describe('AboutDialog', () => {
  it('opens on the About tab with version and stack', () => {
    render(<AboutDialog onClose={vi.fn()} />)
    expect(screen.getByText('Version 0.8.0')).toBeInTheDocument()
    expect(screen.getByText('Go 1.26 + Wails v2')).toBeInTheDocument()
    expect(screen.getByText('MIT')).toBeInTheDocument()
  })

  it('shows release notes with only the newest marked latest', () => {
    render(<AboutDialog onClose={vi.fn()} />)
    fireEvent.click(screen.getByRole('button', { name: 'Release Notes' }))
    expect(screen.getByText('v0.3.0')).toBeInTheDocument()
    expect(screen.getByText('v0.2.0')).toBeInTheDocument()
    expect(screen.getByText('v0.1.0')).toBeInTheDocument()
    expect(screen.getAllByText('Latest')).toHaveLength(1)
  })

  it('shows the privacy statement, license and third-party notices', () => {
    render(<AboutDialog onClose={vi.fn()} />)
    fireEvent.click(screen.getByRole('button', { name: 'Legal' }))
    expect(screen.getByText('No data leaves your device')).toBeInTheDocument()
    expect(screen.getByText(/Permission is hereby granted/)).toBeInTheDocument()
    expect(screen.getByText('Wails v2')).toBeInTheDocument()
    expect(screen.getByText('cobra')).toBeInTheDocument()
  })

  it('switches back to About', () => {
    render(<AboutDialog onClose={vi.fn()} />)
    fireEvent.click(screen.getByRole('button', { name: 'Legal' }))
    fireEvent.click(screen.getByRole('button', { name: 'About' }))
    expect(screen.getByText('Runtime')).toBeInTheDocument()
  })

  it('closes from the backdrop but not from inside the dialog', () => {
    const onClose = vi.fn()
    render(<AboutDialog onClose={onClose} />)
    fireEvent.click(screen.getByText('Clear Signal'))
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.click(screen.getByText('Clear Signal').closest('.fixed')!)
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
