import { useEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { X, Copy, CheckCircle2 } from 'lucide-react'
import { getUserGuide, getFormatSpec, getAIPrompt } from '@/lib/bridge'
import { cn } from '@/lib/utils'

type Tab = 'guide' | 'spec'
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
          <td className="px-3 py-1.5 border border-[var(--color-border)] text-[var(--color-foreground)]">
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

export function HelpPanel({ onClose, initialTab = 'guide' }: Props) {
  const [tab, setTab] = useState<Tab>(initialTab)
  const [guide, setGuide] = useState<string | null>(null)
  const [spec, setSpec] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
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

  async function handleCopyAIPrompt() {
    try {
      const prompt = await getAIPrompt()
      await navigator.clipboard.writeText(prompt)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // clipboard API may be unavailable in some environments
    }
  }

  const tabCls = (active: boolean) => cn(
    'px-4 py-2 text-xs font-medium border-b-2 transition-colors',
    active
      ? 'border-[var(--color-accent)] text-[var(--color-foreground)]'
      : 'border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]'
  )

  const content = tab === 'guide' ? guide : spec

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-6 bg-black/50 backdrop-blur-sm"
      onClick={e => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="flex flex-col w-full max-w-3xl max-h-[88vh] rounded-xl shadow-2xl bg-[var(--color-background)] border border-[var(--color-border)]">

        {/* Header */}
        <div className="flex items-center justify-between px-4 pt-2 border-b border-[var(--color-border)] shrink-0">
          <div className="flex items-center">
            <button type="button" onClick={() => setTab('guide')} className={tabCls(tab === 'guide')}>
              User Guide
            </button>
            <button type="button" onClick={() => setTab('spec')} className={tabCls(tab === 'spec')}>
              Format Spec
            </button>
          </div>
          <div className="flex items-center gap-2 pb-2">
            {tab === 'spec' && (
              <button
                type="button"
                onClick={handleCopyAIPrompt}
                title="Copy the full AI prompt (spec + instructions) to your clipboard — paste into ChatGPT, Gemini, or any AI assistant"
                className={cn(
                  'flex items-center gap-1.5 px-3 py-1 rounded text-xs font-medium transition-colors',
                  copied
                    ? 'bg-emerald-500/15 text-emerald-400'
                    : 'bg-[var(--color-accent)]/10 text-[var(--color-accent)] hover:bg-[var(--color-accent)]/20'
                )}
              >
                {copied
                  ? <><CheckCircle2 size={12} /> Copied!</>
                  : <><Copy size={12} /> Copy AI Prompt</>}
              </button>
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

        {/* Copy AI Prompt hint — shown only on spec tab */}
        {tab === 'spec' && (
          <div className="shrink-0 px-6 py-2.5 border-b border-[var(--color-border)] bg-[var(--color-muted)]/50">
            <p className="text-xs text-[var(--color-muted-foreground)]">
              <span className="font-medium text-[var(--color-foreground)]">Use with AI:</span>{' '}
              Click <span className="font-medium">Copy AI Prompt</span> to copy the spec + a smart system prompt to your clipboard.
              Paste into ChatGPT, Gemini, or any AI assistant — it will interview you, generate valid YAML files, and self-verify them against this spec.
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
