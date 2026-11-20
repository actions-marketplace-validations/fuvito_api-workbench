import { useEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { X, Copy, Download, CheckCircle2 } from 'lucide-react'
import { getUserGuide, getFormatSpec, getWhatWeCheck, getExampleScenario, getAIPromptInstructions, getAIPrompt, saveContentToFile } from '@/lib/bridge'
import { cn } from '@/lib/utils'
import { extractScenarioYaml, extractScenarioFileName } from '@/lib/helpDocs'

export type HelpTab = 'guide' | 'spec' | 'example' | 'prompt' | 'check'
type Tab = HelpTab
type Props = { onClose: () => void; initialTab?: Tab }

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')
}

function textContent(node: React.ReactNode): string {
  if (typeof node === 'string') return node
  if (typeof node === 'number') return String(node)
  if (Array.isArray(node)) return node.map(textContent).join('')
  if (node && typeof node === 'object' && 'props' in (node as React.ReactElement))
    return textContent((node as React.ReactElement<{ children?: React.ReactNode }>).props.children)
  return ''
}

function Heading({ level, children }: { level: 1 | 2 | 3; children: React.ReactNode }) {
  const id = slugify(textContent(children))
  const Tag = `h${level}` as 'h1' | 'h2' | 'h3'
  const cls =
    level === 1
      ? 'text-xl font-bold mt-6 mb-3 text-[var(--color-foreground)]'
      : level === 2
        ? 'text-base font-semibold mt-6 mb-2 text-[var(--color-foreground)] border-b border-[var(--color-border)] pb-1'
        : 'text-sm font-semibold mt-4 mb-1.5 text-[var(--color-foreground)]'
  return <Tag id={id} className={cls}>{children}</Tag>
}

function MarkdownBody({ content, onAnchorClick }: {
  content: string
  onAnchorClick: (href: string | undefined, e: React.MouseEvent) => void
}) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        h1: ({ children }) => <Heading level={1}>{children}</Heading>,
        h2: ({ children }) => <Heading level={2}>{children}</Heading>,
        h3: ({ children }) => <Heading level={3}>{children}</Heading>,
        p: ({ children }) => (
          <p className="text-sm leading-relaxed mb-3 text-[var(--color-foreground)]">{children}</p>
        ),
        a: ({ href, children }) => (
          <a
            href={href}
            onClick={e => onAnchorClick(href, e)}
            className="text-[var(--color-accent)] hover:underline"
          >
            {children}
          </a>
        ),
        ul: ({ children }) => (
          <ul className="list-disc list-inside mb-3 space-y-1 text-sm text-[var(--color-foreground)]">{children}</ul>
        ),
        ol: ({ children }) => (
          <ol className="list-decimal list-inside mb-3 space-y-1 text-sm text-[var(--color-foreground)]">{children}</ol>
        ),
        li: ({ children }) => <li className="leading-relaxed">{children}</li>,
        code: ({ children, className }) => {
          const isBlock = className?.includes('language-')
          return isBlock ? (
            <code className="block bg-[var(--color-muted)] rounded px-3 py-2 text-xs font-mono leading-relaxed text-[var(--color-foreground)] whitespace-pre overflow-x-auto">
              {children}
            </code>
          ) : (
            <code className="bg-[var(--color-muted)] rounded px-1 py-0.5 text-xs font-mono text-[var(--color-foreground)]">
              {children}
            </code>
          )
        },
        pre: ({ children }) => <pre className="mb-3">{children}</pre>,
        table: ({ children }) => (
          <div className="overflow-x-auto mb-3">
            <table className="w-full text-xs border-collapse">{children}</table>
          </div>
        ),
        thead: ({ children }) => <thead>{children}</thead>,
        th: ({ children }) => (
          <th className="text-left px-3 py-1.5 border border-[var(--color-border)] bg-[var(--color-muted)] font-semibold text-[var(--color-foreground)]">
            {children}
          </th>
        ),
        td: ({ children }) => (
          // A check ID such as CS-RESP-008 is meant to be quoted, so it never wraps.
          <td className={cn('px-3 py-1.5 border border-[var(--color-border)] text-[var(--color-foreground)]', isCheckId(children) && 'whitespace-nowrap font-mono')}>
            {children}
          </td>
        ),
        tr: ({ children }) => <tr>{children}</tr>,
        blockquote: ({ children }) => (
          <blockquote className="border-l-2 border-[var(--color-accent)] pl-3 my-3 text-sm text-[var(--color-muted-foreground)] italic">
            {children}
          </blockquote>
        ),
        hr: () => <hr className="border-[var(--color-border)] my-4" />,
        strong: ({ children }) => (
          <strong className="font-semibold text-[var(--color-foreground)]">{children}</strong>
        ),
      }}
    >
      {content}
    </ReactMarkdown>
  )
}

function HeaderAction({ id, done, label, doneLabel, icon, title, onClick }: {
  id: string
  done: string | null
  label: string
  doneLabel: string
  icon: React.ReactNode
  title: string
  onClick: () => void
}) {
  const isDone = done === id
  return (
    <button
      type="button"
      onClick={onClick}
      title={title}
      className={cn(
        'flex items-center gap-1.5 px-3 py-1 rounded text-xs font-medium transition-colors',
        isDone
          ? 'bg-emerald-500/15 text-emerald-400'
          : 'bg-[var(--color-accent)]/10 text-[var(--color-accent)] hover:bg-[var(--color-accent)]/20'
      )}
    >
      {isDone ? <><CheckCircle2 size={12} /> {doneLabel}</> : <>{icon} {label}</>}
    </button>
  )
}

function isCheckId(children: React.ReactNode): boolean {
  return typeof children === 'string' && /^CS-[A-Z]+-\d{3}$/.test(children)
}

export function HelpPanel({ onClose, initialTab = 'guide' }: Props) {
  const [tab, setTab] = useState<Tab>(initialTab)
  const [guide, setGuide] = useState<string | null>(null)
  const [spec, setSpec] = useState<string | null>(null)
  const [example, setExample] = useState<string | null>(null)
  const [promptDoc, setPromptDoc] = useState<string | null>(null)
  const [catalog, setCatalog] = useState<string | null>(null)
  // id of the header action that just succeeded, shown briefly as "Copied!" / "Saved!"
  const [doneAction, setDoneAction] = useState<string | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    getUserGuide().then(setGuide).catch(() => setGuide('Failed to load user guide.'))
  }, [])

  useEffect(() => {
    if (tab === 'spec' && spec === null) {
      getFormatSpec().then(setSpec).catch(() => setSpec('Failed to load format spec.'))
    }
  }, [tab, spec])

  useEffect(() => {
    if (tab === 'example' && example === null) {
      getExampleScenario().then(setExample).catch(() => setExample('Failed to load the example scenario.'))
    }
  }, [tab, example])

  useEffect(() => {
    if (tab === 'check' && catalog === null) {
      getWhatWeCheck().then(setCatalog).catch(() => setCatalog('Failed to load the list of checks.'))
    }
  }, [tab, catalog])

  useEffect(() => {
    if (tab === 'prompt' && promptDoc === null) {
      getAIPromptInstructions().then(setPromptDoc).catch(() => setPromptDoc('Failed to load the AI prompt.'))
    }
  }, [tab, promptDoc])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  useEffect(() => {
    if (scrollRef.current) scrollRef.current.scrollTop = 0
  }, [tab])

  function handleAnchorClick(href: string | undefined, e: React.MouseEvent) {
    if (!href?.startsWith('#')) return
    e.preventDefault()
    const target = document.getElementById(href.slice(1))
    if (target && scrollRef.current) {
      const containerTop = scrollRef.current.getBoundingClientRect().top
      const targetTop = target.getBoundingClientRect().top
      scrollRef.current.scrollBy({ top: targetTop - containerTop - 16, behavior: 'smooth' })
    }
  }

  async function runAction(id: string, action: () => Promise<boolean>) {
    try {
      if (!(await action())) return
      setDoneAction(id)
      setTimeout(() => setDoneAction(d => (d === id ? null : d)), 2000)
    } catch {
      // clipboard or save dialog unavailable — nothing useful to show
    }
  }

  const getSpecText = async () => spec ?? await getFormatSpec()
  const getCatalogText = async () => catalog ?? await getWhatWeCheck()
  const getScenarioYaml = async () => (example ? extractScenarioYaml(example) : null)

  function copyAction(getText: () => Promise<string | null>) {
    return async () => {
      const text = await getText()
      if (!text) return false
      await navigator.clipboard.writeText(text)
      return true
    }
  }

  // The native save dialog returns '' when the user cancels.
  function downloadAction(getText: () => Promise<string | null>, fileName: string) {
    return async () => {
      const text = await getText()
      if (!text) return false
      return (await saveContentToFile(text, fileName)) !== ''
    }
  }

  const tabCls = (active: boolean) => cn(
    'px-4 py-2 text-xs font-medium border-b-2 transition-colors',
    active
      ? 'border-[var(--color-accent)] text-[var(--color-foreground)]'
      : 'border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]'
  )

  const content = tab === 'guide' ? guide : tab === 'spec' ? spec : tab === 'example' ? example : tab === 'check' ? catalog : promptDoc

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-6 bg-black/50 backdrop-blur-sm"
      onClick={e => { if (e.target === e.currentTarget) onClose() }}
    >
      {/* The panel takes most of the window, so wide tables (the catalog's) and long guides use the room there is; the cap keeps it from stretching across a very large screen. */}
      <div
        data-testid="help-panel"
        className="flex flex-col w-[90vw] max-w-[1800px] max-h-[90vh] rounded-xl shadow-2xl bg-[var(--color-background)] border border-[var(--color-border)]"
      >

        {/* Header */}
        <div className="flex items-center justify-between px-4 pt-2 border-b border-[var(--color-border)] shrink-0">
          <div className="flex items-center">
            <button type="button" onClick={() => setTab('guide')} className={tabCls(tab === 'guide')}>
              User Guide
            </button>
            <button type="button" onClick={() => setTab('spec')} className={tabCls(tab === 'spec')}>
              Format Spec
            </button>
            <button type="button" onClick={() => setTab('example')} className={tabCls(tab === 'example')}>
              Example Scenario
            </button>
            <button type="button" onClick={() => setTab('prompt')} className={tabCls(tab === 'prompt')}>
              AI Prompt
            </button>
            <button type="button" onClick={() => setTab('check')} className={tabCls(tab === 'check')}>
              What We Check
            </button>
          </div>
          <div className="flex items-center gap-2 pb-2">
            {tab === 'spec' && (
              <>
                <HeaderAction
                  id="download-spec" done={doneAction} label="Download" doneLabel="Saved!" icon={<Download size={12} />}
                  title="Save the format specification as a Markdown file — attach it to a chat"
                  onClick={() => runAction('download-spec', downloadAction(getSpecText, 'clear-signal-format-spec.md'))}
                />
                <HeaderAction
                  id="copy-spec" done={doneAction} label="Copy Format Spec" doneLabel="Copied!" icon={<Copy size={12} />}
                  title="Copy the format specification to your clipboard"
                  onClick={() => runAction('copy-spec', copyAction(getSpecText))}
                />
              </>
            )}
            {tab === 'check' && (
              <>
                <HeaderAction
                  id="download-catalog" done={doneAction} label="Download" doneLabel="Saved!" icon={<Download size={12} />}
                  title="Save the list of checks as a Markdown file"
                  onClick={() => runAction('download-catalog', downloadAction(getCatalogText, 'clear-signal-what-we-check.md'))}
                />
                <HeaderAction
                  id="copy-catalog" done={doneAction} label="Copy" doneLabel="Copied!" icon={<Copy size={12} />}
                  title="Copy the list of checks to your clipboard, for example to quote an ID in a bug report"
                  onClick={() => runAction('copy-catalog', copyAction(getCatalogText))}
                />
              </>
            )}
            {tab === 'prompt' && (
              <>
                <HeaderAction
                  id="download-prompt" done={doneAction} label="Download" doneLabel="Saved!" icon={<Download size={12} />}
                  title="Save the complete prompt (instructions + format spec + example scenario) as a Markdown file — attach it to a chat"
                  onClick={() => runAction('download-prompt', downloadAction(getAIPrompt, 'clear-signal-ai-prompt.md'))}
                />
                <HeaderAction
                  id="copy-prompt" done={doneAction} label="Copy AI Prompt" doneLabel="Copied!" icon={<Copy size={12} />}
                  title="Copy the complete prompt (instructions + format spec + example scenario) to your clipboard"
                  onClick={() => runAction('copy-prompt', copyAction(getAIPrompt))}
                />
              </>
            )}
            {tab === 'example' && example !== null && (
              <>
                <HeaderAction
                  id="download-scenario" done={doneAction} label="Download" doneLabel="Saved!" icon={<Download size={12} />}
                  title="Save the scenario as a YAML file you can drop into a workspace's scenarios folder"
                  onClick={() => runAction('download-scenario', downloadAction(getScenarioYaml, extractScenarioFileName(example)))}
                />
                <HeaderAction
                  id="copy-scenario" done={doneAction} label="Copy scenario" doneLabel="Copied!" icon={<Copy size={12} />}
                  title="Copy the scenario YAML to your clipboard — save it as scenarios/user-onboarding.yaml in a workspace"
                  onClick={() => runAction('copy-scenario', copyAction(getScenarioYaml))}
                />
              </>
            )}
            <button
              type="button"
              onClick={onClose}
              className="p-1 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
            >
              <X size={14} />
            </button>
          </div>
        </div>

        {/* Copy hint — shown only on spec tab */}
        {tab === 'spec' && (
          <div className="shrink-0 px-6 py-2.5 border-b border-[var(--color-border)] bg-[var(--color-muted)]/50">
            <p className="text-xs text-[var(--color-muted-foreground)]">
              <span className="font-medium text-[var(--color-foreground)]">Copy it anywhere:</span>{' '}
              Click <span className="font-medium">Copy Format Spec</span> to copy this whole document to your clipboard — handy for pasting into an AI assistant, a wiki or a code review.
            </p>
          </div>
        )}

        {/* Content */}
        <div ref={scrollRef} className="overflow-y-auto flex-1 px-8 py-4">
          {content === null ? (
            <div className="text-sm text-[var(--color-muted-foreground)] py-8 text-center">Loading…</div>
          ) : (
            <MarkdownBody
              content={content}
              onAnchorClick={handleAnchorClick}
            />
          )}
        </div>
      </div>
    </div>
  )
}
