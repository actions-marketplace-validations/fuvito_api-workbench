import { useState } from 'react'
import { getSettings, saveSettings, resetSettings } from '@/lib/bridge'
import type { AppSettings } from '@/lib/bridge'

export function useSettings() {
  const [settings, setSettings] = useState<AppSettings>({
    theme: 'dark', fontSize: 13, defaultTimeoutMs: 30000, proxyUrl: '', tlsSkipVerify: false,
  })
  const [showSettings, setShowSettings] = useState(false)

  function applySettings(s: AppSettings) {
    const root = document.documentElement
    root.style.setProperty('--font-size-code', `${s.fontSize}px`)
    if (s.theme === 'light') {
      root.classList.add('light')
      root.classList.remove('dark')
    } else {
      root.classList.add('dark')
      root.classList.remove('light')
    }
  }

  async function loadSettings() {
    try {
      const s = await getSettings()
      setSettings(s)
      applySettings(s)
    } catch { /* best-effort */ }
  }

  async function handleToggleTheme() {
    const next = { ...settings, theme: settings.theme === 'dark' ? 'light' : 'dark' } as AppSettings
    await handleSaveSettings(next)
  }

  async function handleSaveSettings(s: AppSettings) {
    try {
      const saved = await saveSettings(s)
      setSettings(saved)
      applySettings(saved)
    } catch (err) {
      console.error('save settings:', err)
    }
  }

  async function handleResetSettings() {
    try {
      const defaults = await resetSettings()
      setSettings(defaults)
      applySettings(defaults)
      setShowSettings(false)
    } catch (err) {
      console.error('reset settings:', err)
    }
  }

  return {
    settings, showSettings, setShowSettings,
    applySettings, loadSettings,
    handleToggleTheme, handleSaveSettings, handleResetSettings,
  }
}
