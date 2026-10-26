/**
 * 轻量 Markdown 渲染器（公告专用）。
 *
 * 先用 HTML 实体转义所有用户输入，再按 Markdown 规则还原受控标签，
 * 因此不需要额外的 sanitizer。仅支持常用语法：
 * 标题、粗体、斜体、删除线、行内代码、代码块、引用、列表、链接、分隔线。
 */

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

/** 只允许 http(s)、mailto 与站内相对地址，避免 javascript: 注入。 */
function safeUrl(url: string): string | null {
  const value = url.trim();
  if (!value) return null;
  if (/^(https?:\/\/|mailto:|\/|#)/i.test(value)) return value;
  return null;
}

function renderInline(text: string): string {
  let out = escapeHtml(text);

  // 行内代码优先，避免其中的 *、_ 被继续解析。
  const codeSlots: string[] = [];
  out = out.replace(/`([^`]+)`/g, (_, code: string) => {
    const index = codeSlots.push(`<code>${code}</code>`) - 1;
    return `\u0000CODE${index}\u0000`;
  });

  // 链接 [文本](地址)
  out = out.replace(/\[([^\]]+)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g, (whole, label: string, url: string) => {
    const href = safeUrl(url);
    if (!href) return whole;
    return `<a href="${escapeHtml(href)}" target="_blank" rel="noopener noreferrer">${label}</a>`;
  });

  out = out
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/__([^_]+)__/g, '<strong>$1</strong>')
    .replace(/~~([^~]+)~~/g, '<del>$1</del>')
    .replace(/(^|[^*])\*([^*\n]+)\*/g, '$1<em>$2</em>')
    .replace(/(^|[^_])_([^_\n]+)_/g, '$1<em>$2</em>');

  out = out.replace(/\u0000CODE(\d+)\u0000/g, (_, index: string) => codeSlots[Number(index)] || '');
  return out;
}

function isSpecialLine(line: string): boolean {
  return (
    /^\s*#{1,6}\s+/.test(line) ||
    /^\s*>\s?/.test(line) ||
    /^\s*([-*+]|\d+\.)\s+/.test(line) ||
    /^\s*```/.test(line) ||
    /^\s*([-*_])(?:\s*\1){2,}\s*$/.test(line)
  );
}

export function renderMarkdown(source: string): string {
  if (!source) return '';
  const lines = source.replace(/\r\n?/g, '\n').split('\n');
  const html: string[] = [];
  let index = 0;

  while (index < lines.length) {
    const line = lines[index];

    // 代码块
    if (/^\s*```/.test(line)) {
      const lang = line.replace(/^\s*```/, '').trim();
      const code: string[] = [];
      index++;
      while (index < lines.length && !/^\s*```/.test(lines[index])) {
        code.push(lines[index]);
        index++;
      }
      index++; // 跳过结束围栏
      html.push(
        `<pre class="md-code"><code${lang ? ` class="language-${escapeHtml(lang)}"` : ''}>${escapeHtml(code.join('\n'))}</code></pre>`,
      );
      continue;
    }

    // 空行
    if (!line.trim()) {
      index++;
      continue;
    }

    // 分隔线
    if (/^\s*([-*_])(?:\s*\1){2,}\s*$/.test(line)) {
      html.push('<hr />');
      index++;
      continue;
    }

    // 标题
    const heading = line.match(/^\s*(#{1,6})\s+(.*)$/);
    if (heading) {
      const level = heading[1].length;
      html.push(`<h${level}>${renderInline(heading[2].trim())}</h${level}>`);
      index++;
      continue;
    }

    // 引用（连续行合并）
    if (/^\s*>\s?/.test(line)) {
      const quote: string[] = [];
      while (index < lines.length && /^\s*>\s?/.test(lines[index])) {
        quote.push(lines[index].replace(/^\s*>\s?/, ''));
        index++;
      }
      html.push(`<blockquote>${quote.map((item) => renderInline(item)).join('<br />')}</blockquote>`);
      continue;
    }

    // 列表（支持嵌套一层）
    const listMatch = line.match(/^(\s*)([-*+]|\d+\.)\s+(.*)$/);
    if (listMatch) {
      const ordered = /\d+\./.test(listMatch[2]);
      const items: string[] = [];
      let firstIndent = listMatch[1].length;
      while (index < lines.length) {
        const m = lines[index].match(/^(\s*)([-*+]|\d+\.)\s+(.*)$/);
        if (!m) break;
        const indent = m[1].length;
        if (indent < firstIndent) break;
        if (indent > firstIndent) {
          // 嵌套项并入上一项，保持可读即可
          const parent = items.pop() || '';
          items.push(`${parent}<ul><li>${renderInline(m[3])}</li></ul>`);
        } else {
          items.push(`<li>${renderInline(m[3])}</li>`);
        }
        index++;
      }
      const tag = ordered ? 'ol' : 'ul';
      html.push(`<${tag}>${items.join('')}</${tag}>`);
      continue;
    }

    // 普通段落：合并连续的非特殊行，单换行用 <br />
    const paragraph: string[] = [line];
    index++;
    while (index < lines.length && lines[index].trim() && !isSpecialLine(lines[index])) {
      paragraph.push(lines[index]);
      index++;
    }
    html.push(`<p>${paragraph.map((item) => renderInline(item.trim())).join('<br />')}</p>`);
  }

  return html.join('\n');
}
