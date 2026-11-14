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
    <div className="flex-1 flex flex-col overflow-hidden">
      {error && (
        <p className="shrink-0 px-3 py-2 text-xs text-red-400 border-b border-red-500/20 bg-red-500/5 whitespace-pre-wrap">{error}</p>
      )}
      <div className="flex-1 overflow-hidden">
        <CodeMirror value={value} extensions={[yaml()]} theme={theme === 'light' ? 'light' : 'dark'}
          onChange={onChange} height="100%" style={{ height: '100%', fontSize: '12px' }} />
      </div>
    </div>
  )
}
