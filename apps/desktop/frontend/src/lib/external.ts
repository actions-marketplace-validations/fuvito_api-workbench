import { BrowserOpenURL } from '@wailsjs/runtime/runtime'

// The page where problems and ideas are reported.
export const FEEDBACK_URL = 'https://github.com/fuvito/api-workbench/issues/new/choose'

// Opens an address in the system browser. Inside the desktop app that is the Wails runtime; in a
// plain browser (the dev server, the tests) it falls back to a new tab.
export function openExternal(url: string): void {
  try {
    BrowserOpenURL(url)
  } catch {
    window.open(url, '_blank', 'noopener,noreferrer')
  }
}
