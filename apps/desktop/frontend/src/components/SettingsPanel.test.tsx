import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SettingsPanel, portProblem } from './SettingsPanel'
import type { AppSettings } from '@/lib/bridge'

const defaults: AppSettings = { theme: 'dark', fontSize: 13, defaultTimeoutMs: 30000, proxyUrl: '', tlsSkipVerify: false, telemetryEnabled: false }

function setup(settings: AppSettings = defaults, withPreview = true) {
  const h = { onSave: vi.fn(), onReset: vi.fn(), onClose: vi.fn(), onPreview: withPreview ? vi.fn() : undefined }
  render(<SettingsPanel settings={settings} {...h} />)
  return h
}

const modifiedDots = () => screen.queryAllByTitle('Modified from default')

describe('SettingsPanel', () => {
  it('starts from the supplied settings and flags nothing as modified at defaults', () => {
    setup()
    expect((screen.getByDisplayValue('Dark') as HTMLSelectElement).value).toBe('dark')
    expect(screen.getByText('Font size — 13px')).toBeInTheDocument()
    expect(modifiedDots()).toHaveLength(0)
  })

  it('flags every setting that differs from the defaults', () => {
    setup({ theme: 'light', fontSize: 16, defaultTimeoutMs: 1000, proxyUrl: 'http://p:1', tlsSkipVerify: true, telemetryEnabled: true })
    expect(modifiedDots()).toHaveLength(6)
  })

  it('previews each change live and updates the modified marker', () => {
    const h = setup()
    fireEvent.change(screen.getByDisplayValue('Dark'), { target: { value: 'light' } })
    expect(h.onPreview).toHaveBeenLastCalledWith({ ...defaults, theme: 'light' })
    expect(modifiedDots()).toHaveLength(1)

    fireEvent.change(screen.getByRole('slider'), { target: { value: '18' } })
    expect(h.onPreview).toHaveBeenLastCalledWith({ ...defaults, theme: 'light', fontSize: 18 })
    expect(screen.getByText('Font size — 18px')).toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText('30000'), { target: { value: '5000' } })
    expect(h.onPreview).toHaveBeenLastCalledWith(expect.objectContaining({ defaultTimeoutMs: 5000 }))

    const [tls, telemetry] = screen.getAllByRole('checkbox')
    fireEvent.click(tls)
    fireEvent.click(telemetry)
    expect(h.onPreview).toHaveBeenLastCalledWith(expect.objectContaining({ tlsSkipVerify: true, telemetryEnabled: true }))
  })

  it('works without a preview callback', () => {
    setup(defaults, false)
    expect(() => fireEvent.change(screen.getByDisplayValue('Dark'), { target: { value: 'light' } })).not.toThrow()
  })

  it('saves the edited settings and closes', () => {
    const h = setup()
    fireEvent.change(screen.getByPlaceholderText('http://proxy.example.com:8080'), { target: { value: ' http://proxy.test:3128 ' } })
    fireEvent.click(screen.getByText('Save'))
    expect(h.onSave).toHaveBeenCalledWith({ ...defaults, proxyUrl: ' http://proxy.test:3128 ' })
    expect(h.onClose).toHaveBeenCalled()
  })

  it('rejects an invalid proxy URL without saving, and clears the error when edited', () => {
    const h = setup()
    const input = screen.getByPlaceholderText('http://proxy.example.com:8080')
    fireEvent.change(input, { target: { value: 'not a url' } })
    fireEvent.click(screen.getByText('Save'))
    expect(screen.getByText(/Enter a valid URL/)).toBeInTheDocument()
    expect(h.onSave).not.toHaveBeenCalled()
    expect(h.onClose).not.toHaveBeenCalled()
    expect(input.className).toContain('border-red-500')

    fireEvent.change(input, { target: { value: 'http://ok.test' } })
    expect(screen.queryByText(/Enter a valid URL/)).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('Save'))
    expect(h.onSave).toHaveBeenCalledTimes(1)
  })

  it('allows saving with an empty or whitespace proxy', () => {
    const h = setup()
    fireEvent.change(screen.getByPlaceholderText('http://proxy.example.com:8080'), { target: { value: '   ' } })
    fireEvent.click(screen.getByText('Save'))
    expect(h.onSave).toHaveBeenCalled()
  })

  it('cancel and the close icon close without saving', () => {
    const h = setup()
    fireEvent.click(screen.getByText('Cancel'))
    fireEvent.click(screen.getByLabelText('Close settings'))
    expect(h.onClose).toHaveBeenCalledTimes(2)
    expect(h.onSave).not.toHaveBeenCalled()
  })

  it('reset delegates to the parent', () => {
    const h = setup()
    fireEvent.click(screen.getByText('Reset to defaults'))
    expect(h.onReset).toHaveBeenCalledTimes(1)
  })
})

describe('SettingsPanel — client certificates', () => {
  it('edits the three file paths and saves them', () => {
    const h = setup()
    fireEvent.change(screen.getByLabelText('Client certificate file'), { target: { value: '/c/client.pem' } })
    fireEvent.change(screen.getByLabelText('Client key file'), { target: { value: '/c/client.key' } })
    fireEvent.change(screen.getByLabelText('CA certificate file'), { target: { value: '${CA_FILE}' } })
    fireEvent.click(screen.getByRole('button', { name: /save/i }))
    expect(h.onSave).toHaveBeenCalledWith(expect.objectContaining({
      clientCertFile: '/c/client.pem', clientKeyFile: '/c/client.key', caFile: '${CA_FILE}',
    }))
  })

  it('shows saved paths', () => {
    setup({ ...defaults, clientCertFile: '/c/client.pem' })
    expect((screen.getByLabelText('Client certificate file') as HTMLInputElement).value).toBe('/c/client.pem')
    expect((screen.getByLabelText('Client key file') as HTMLInputElement).value).toBe('')
  })
})

describe('SettingsPanel Dev server ports', () => {
  it('saves the configured ports', () => {
    const h = setup()
    fireEvent.change(screen.getByLabelText('Demo Server port'), { target: { value: '9090' } })
    fireEvent.change(screen.getByLabelText('Conformance Server port'), { target: { value: '9120' } })
    fireEvent.click(screen.getByRole('button', { name: /save/i }))
    expect(h.onSave).toHaveBeenCalledWith(expect.objectContaining({ demoServerPort: 9090, conformanceServerPort: 9120 }))
  })

  it('refuses an out-of-range port or two servers on the same port', () => {
    const h = setup()
    fireEvent.change(screen.getByLabelText('Demo Server port'), { target: { value: '80' } })
    fireEvent.click(screen.getByRole('button', { name: /save/i }))
    expect(screen.getByRole('alert')).toHaveTextContent('between 1024 and 65535')
    expect(h.onSave).not.toHaveBeenCalled()

    fireEvent.change(screen.getByLabelText('Demo Server port'), { target: { value: '9000' } })
    fireEvent.change(screen.getByLabelText('Conformance Server port'), { target: { value: '9000' } })
    fireEvent.click(screen.getByRole('button', { name: /save/i }))
    expect(screen.getByRole('alert')).toHaveTextContent('different ports')
    expect(h.onSave).not.toHaveBeenCalled()
  })

  it('treats 0 as automatic for both', () => {
    expect(portProblem(0, 0)).toBe('')
    expect(portProblem(0, 9000)).toBe('')
    expect(portProblem(1023, 0)).not.toBe('')
    expect(portProblem(70000, 0)).not.toBe('')
  })
})
