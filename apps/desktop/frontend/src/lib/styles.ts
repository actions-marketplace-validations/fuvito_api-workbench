// Shared input/select/textarea visual affordance classes.
// Use as the base and add layout classes with cn(): cn(inputCls, 'w-full font-mono px-3 py-1.5')
export const inputCls =
  'bg-[var(--color-input)] border border-[var(--color-input-border)] rounded ' +
  'text-[var(--color-foreground)] text-xs ' +
  'outline-none focus:border-[var(--color-accent)] transition-colors ' +
  'placeholder:text-[var(--color-muted-foreground)]'

export const selectCls = inputCls + ' cursor-pointer'
