type Props = {
  description: string
  specFile: string
  onChange: (patch: { description?: string; specFile?: string }) => void
}

export function ScenarioMetaSection({ description, specFile, onChange }: Props) {
  return (
    <div className="shrink-0 flex gap-6 px-4 py-3 border-b border-[var(--color-border)] bg-[var(--color-card)]">
      <div className="flex-1 flex flex-col gap-1 min-w-0">
        <label className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
          Description
        </label>
        <input
          value={description}
          onChange={e => onChange({ description: e.target.value })}
          placeholder="What does this scenario test?"
          className="w-full bg-transparent border-b border-[var(--color-border)] focus:border-[var(--color-accent)] outline-none text-sm text-[var(--color-foreground)] py-1 transition-colors"
        />
      </div>
      <div className="flex-1 flex flex-col gap-1 min-w-0">
        <label className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
          Spec file
        </label>
        <input
          value={specFile}
          onChange={e => onChange({ specFile: e.target.value })}
          placeholder="OpenAPI spec path or URL (optional)"
          className="w-full bg-transparent border-b border-[var(--color-border)] focus:border-[var(--color-accent)] outline-none text-xs text-[var(--color-muted-foreground)] py-1 transition-colors font-mono"
        />
      </div>
    </div>
  )
}
