import {
  createElement,
  Fragment,
  type ReactNode,
  type ReactElement,
} from 'react'

type Props = {
  content: string
  className?: string
}

/**
 * Lightweight Markdown renderer for chat replies (no extra deps).
 * Supports: headings, paragraphs, bold/italic/code/links, lists,
 * fenced code blocks, blockquotes, hr, and simple GFM tables.
 */
export function ChatMarkdown({ content, className }: Props) {
  const text = content.replace(/\r\n/g, '\n').trimEnd()
  if (!text) return null

  const blocks = splitBlocks(text)
  return (
    <div className={className ? `ai-chat-md ${className}` : 'ai-chat-md'}>
      {blocks.map((block, i) => (
        <Fragment key={i}>{renderBlock(block)}</Fragment>
      ))}
    </div>
  )
}

type Block =
  | { type: 'code'; lang: string; code: string }
  | { type: 'heading'; level: number; text: string }
  | { type: 'quote'; lines: string[] }
  | { type: 'list'; ordered: boolean; items: string[] }
  | { type: 'table'; header: string[]; rows: string[][] }
  | { type: 'hr' }
  | { type: 'p'; text: string }

function splitBlocks(src: string): Block[] {
  const lines = src.split('\n')
  const out: Block[] = []
  let i = 0

  while (i < lines.length) {
    const line = lines[i]

    // Fenced code
    const fence = line.match(/^```(\w*)\s*$/)
    if (fence) {
      const lang = fence[1] || ''
      i += 1
      const body: string[] = []
      while (i < lines.length && !/^```\s*$/.test(lines[i])) {
        body.push(lines[i])
        i += 1
      }
      if (i < lines.length) i += 1
      out.push({ type: 'code', lang, code: body.join('\n') })
      continue
    }

    if (/^\s*([-*_])\1{2,}\s*$/.test(line)) {
      out.push({ type: 'hr' })
      i += 1
      continue
    }

    const heading = line.match(/^(#{1,4})\s+(.+)$/)
    if (heading) {
      out.push({
        type: 'heading',
        level: heading[1].length,
        text: heading[2].trim(),
      })
      i += 1
      continue
    }

    if (/^\s*>\s?/.test(line)) {
      const q: string[] = []
      while (i < lines.length && /^\s*>\s?/.test(lines[i])) {
        q.push(lines[i].replace(/^\s*>\s?/, ''))
        i += 1
      }
      out.push({ type: 'quote', lines: q })
      continue
    }

    if (isTableHeader(lines, i)) {
      const header = splitTableRow(lines[i])
      i += 2 // skip header + separator
      const rows: string[][] = []
      while (i < lines.length && looksLikeTableRow(lines[i])) {
        rows.push(splitTableRow(lines[i]))
        i += 1
      }
      out.push({ type: 'table', header, rows })
      continue
    }

    if (/^\s*([-*+]|\d+[.)])\s+/.test(line)) {
      const ordered = /^\s*\d+[.)]\s+/.test(line)
      const items: string[] = []
      while (i < lines.length && /^\s*([-*+]|\d+[.)])\s+/.test(lines[i])) {
        items.push(lines[i].replace(/^\s*([-*+]|\d+[.)])\s+/, ''))
        i += 1
      }
      out.push({ type: 'list', ordered, items })
      continue
    }

    if (!line.trim()) {
      i += 1
      continue
    }

    const para: string[] = [line]
    i += 1
    while (
      i < lines.length &&
      lines[i].trim() &&
      !/^```/.test(lines[i]) &&
      !/^(#{1,4})\s+/.test(lines[i]) &&
      !/^\s*>\s?/.test(lines[i]) &&
      !/^\s*([-*+]|\d+[.)])\s+/.test(lines[i]) &&
      !/^\s*([-*_])\1{2,}\s*$/.test(lines[i]) &&
      !isTableHeader(lines, i)
    ) {
      para.push(lines[i])
      i += 1
    }
    out.push({ type: 'p', text: para.join('\n') })
  }

  return out
}

function isTableHeader(lines: string[], i: number): boolean {
  if (i + 1 >= lines.length) return false
  if (!looksLikeTableRow(lines[i])) return false
  return /^\s*\|?[\s:|-]+\|[\s:|-]+\|?\s*$/.test(lines[i + 1])
}

function looksLikeTableRow(line: string): boolean {
  return line.includes('|') && line.trim().length > 0
}

function splitTableRow(line: string): string[] {
  let s = line.trim()
  if (s.startsWith('|')) s = s.slice(1)
  if (s.endsWith('|')) s = s.slice(0, -1)
  return s.split('|').map((c) => c.trim())
}

function renderBlock(block: Block): ReactElement {
  switch (block.type) {
    case 'code':
      return (
        <pre className="ai-chat-md-code">
          {block.lang ? (
            <span className="ai-chat-md-code-lang">{block.lang}</span>
          ) : null}
          <code>{block.code}</code>
        </pre>
      )
    case 'heading': {
      const tag = `h${Math.min(4, Math.max(1, block.level))}` as
        | 'h1'
        | 'h2'
        | 'h3'
        | 'h4'
      return createElement(tag, null, renderInline(block.text))
    }
    case 'quote':
      return (
        <blockquote>
          {block.lines.map((l, i) => (
            <p key={i}>{renderInline(l)}</p>
          ))}
        </blockquote>
      )
    case 'list': {
      const Tag = block.ordered ? 'ol' : 'ul'
      return (
        <Tag>
          {block.items.map((item, i) => (
            <li key={i}>{renderInline(item)}</li>
          ))}
        </Tag>
      )
    }
    case 'table':
      return (
        <div className="ai-chat-md-table-wrap">
          <table>
            <thead>
              <tr>
                {block.header.map((c, i) => (
                  <th key={i}>{renderInline(c)}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {block.rows.map((row, ri) => (
                <tr key={ri}>
                  {row.map((c, ci) => (
                    <td key={ci}>{renderInline(c)}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )
    case 'hr':
      return <hr />
    case 'p':
      return <p>{renderInline(block.text)}</p>
  }
}

/** Inline: `code`, **bold**, *italic*, [link](url), plain text. */
function renderInline(text: string): ReactNode[] {
  const nodes: ReactNode[] = []
  const re =
    /(`[^`]+`)|(\*\*[^*]+\*\*)|(__[^_]+__)|(\*[^*\n]+\*)|(_[^_\n]+_)|(\[[^\]]+\]\([^)\s]+\))/g
  let last = 0
  let m: RegExpExecArray | null
  let key = 0
  while ((m = re.exec(text)) !== null) {
    if (m.index > last) {
      nodes.push(text.slice(last, m.index))
    }
    const raw = m[0]
    if (raw.startsWith('`')) {
      nodes.push(
        <code key={key++} className="ai-chat-md-inline-code">
          {raw.slice(1, -1)}
        </code>
      )
    } else if (raw.startsWith('**') || raw.startsWith('__')) {
      nodes.push(<strong key={key++}>{raw.slice(2, -2)}</strong>)
    } else if (raw.startsWith('*') || raw.startsWith('_')) {
      nodes.push(<em key={key++}>{raw.slice(1, -1)}</em>)
    } else if (raw.startsWith('[')) {
      const link = raw.match(/^\[([^\]]+)\]\(([^)\s]+)\)$/)
      if (link) {
        nodes.push(
          <a
            key={key++}
            href={link[2]}
            target="_blank"
            rel="noopener noreferrer"
          >
            {link[1]}
          </a>
        )
      } else {
        nodes.push(raw)
      }
    } else {
      nodes.push(raw)
    }
    last = m.index + raw.length
  }
  if (last < text.length) nodes.push(text.slice(last))
  return nodes
}
