import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import type { AppSettings } from '@/lib/bridge'

vi.mock('@/lib/bridge', () => ({
  getSettings: vi.fn(),
  saveSettings: vi.fn(),
  resetSettings: vi.fn(),
}))

import { getSettings, saveSettings, resetSettings } from '@/lib/bridge'
import { useSettings } from './useSettings'

const light: AppSettings = { theme: 'light', fontSize: 16, defaultTimeoutMs: 1000, proxyUrl: '', tlsSkipVerify: false, telemetryEnabled: false }
const dark: AppSettings = { ...light, theme: 'dark', fontSize: 13 }

beforeEach(() => {
  vi.clearAllMocks()
  document.documentElement.className = ''
  document.documentElement.style.removeProperty('--font-size-code')
})

describe('useSettings', () => {
  it('starts with dark defaults and a closed dialog', () => {
    const { result } = renderHook(() => useSettings())
    expect(result.current.settings).toMatchObject({ theme: 'dark', fontSize: 13, defaultTimeoutMs: 30000, telemetryEnabled: false })
    expect(result.current.showSettings).toBe(false)
  })

  it('loadSettings applies theme and font size to the document', async () => {
    vi.mocked(getSettings).mockResolvedValue(light)
    const { result } = renderHook(() => useSettings())
    await act(() => result.current.loadSettings())

    expect(result.current.settings).toEqual(light)
    expect(document.documentElement.classList.contains('light')).toBe(true)
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(document.documentElement.style.getPropertyValue('--font-size-code')).toBe('16px')
  })

  it('loadSettings keeps the defaults when the backend fails', async () => {
    vi.mocked(getSettings).mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useSettings())
    await act(() => result.current.loadSettings())
    expect(result.current.settings.theme).toBe('dark')
  })

  it('handleSaveSettings stores what the backend returns, not what was sent', async () => {
    vi.mocked(saveSettings).mockResolvedValue({ ...light, fontSize: 24 })
    const { result } = renderHook(() => useSettings())
    await act(() => result.current.handleSaveSettings({ ...light, fontSize: 99 }))

    expect(saveSettings).toHaveBeenCalledWith({ ...light, fontSize: 99 })
    expect(result.current.settings.fontSize).toBe(24)
    expect(document.documentElement.style.getPropertyValue('--font-size-code')).toBe('24px')
  })

  it('handleSaveSettings leaves state alone on failure', async () => {
    const err = vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.mocked(saveSettings).mockRejectedValue(new Error('disk full'))
    const { result } = renderHook(() => useSettings())
    await act(() => result.current.handleSaveSettings(light))
    expect(result.current.settings.theme).toBe('dark')
    expect(err).toHaveBeenCalled()
    err.mockRestore()
  })

  it('handleToggleTheme flips dark and light through the backend', async () => {
    vi.mocked(saveSettings).mockImplementation(async s => s)
    const { result } = renderHook(() => useSettings())
    await act(() => result.current.handleToggleTheme())
    expect(saveSettings).toHaveBeenLastCalledWith(expect.objectContaining({ theme: 'light' }))
    expect(result.current.settings.theme).toBe('light')
    await act(() => result.current.handleToggleTheme())
    expect(saveSettings).toHaveBeenLastCalledWith(expect.objectContaining({ theme: 'dark' }))
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('handleResetSettings restores defaults and closes the dialog', async () => {
    vi.mocked(resetSettings).mockResolvedValue(dark)
    const { result } = renderHook(() => useSettings())
    act(() => result.current.setShowSettings(true))
    expect(result.current.showSettings).toBe(true)
    await act(() => result.current.handleResetSettings())
    expect(result.current.settings).toEqual(dark)
    expect(result.current.showSettings).toBe(false)
  })

  it('handleResetSettings keeps the dialog open if the reset fails', async () => {
    const err = vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.mocked(resetSettings).mockRejectedValue(new Error('nope'))
    const { result } = renderHook(() => useSettings())
    act(() => result.current.setShowSettings(true))
    await act(() => result.current.handleResetSettings())
    expect(result.current.showSettings).toBe(true)
    err.mockRestore()
  })
})
