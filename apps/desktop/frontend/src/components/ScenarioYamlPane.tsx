import CodeMirror from '@uiw/react-codemirror'
import { yaml } from '@codemirror/lang-yaml'

type Props = {
  theme: 'dark' | 'light'
  value: string
  error: string | null
  onChange: (value: string) => void
}

export function YamlEditorPane({ theme, value, error, onChange }: Props) {
  return (
    <div className="flex-1 overflow-hidden">
      {error
        ? <p className="px-3 py-2 text-xs text-red-400">{error}</p>
        : <CodeMirror value={value} extensions={[yaml()]} theme={theme === 'light' ? 'light' : 'dark'}
            onChange={onChange} height="100%" style={{ height: '100%', fontSize: '12px' }} />}
    </div>
  )
}
