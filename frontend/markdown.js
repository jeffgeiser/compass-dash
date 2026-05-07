// Minimal inline markdown renderer — no external dependencies.
// Supports: headings, bold, italic, inline code, fenced code blocks, blockquotes, unordered lists, paragraphs.
(function (global) {
  function escapeHTML(str) {
    return str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function inlineMarkdown(text) {
    // Code spans (must come before bold/italic to avoid double-processing)
    text = text.replace(/`([^`]+)`/g, (_, code) => `<code>${escapeHTML(code)}</code>`);
    // Bold
    text = text.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
    // Italic
    text = text.replace(/\*(.+?)\*/g, '<em>$1</em>');
    // Links
    text = text.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>');
    return text;
  }

  function render(markdown) {
    const lines = markdown.split('\n');
    const out = [];
    let i = 0;

    while (i < lines.length) {
      const line = lines[i];
      const trimmed = line.trim();

      // Fenced code block
      if (trimmed.startsWith('```')) {
        const lang = trimmed.slice(3).trim();
        const codeLines = [];
        i++;
        while (i < lines.length && !lines[i].trim().startsWith('```')) {
          codeLines.push(escapeHTML(lines[i]));
          i++;
        }
        out.push(`<pre><code${lang ? ` class="language-${lang}"` : ''}>${codeLines.join('\n')}</code></pre>`);
        i++; // skip closing ```
        continue;
      }

      // Headings
      const headingMatch = trimmed.match(/^(#{1,6})\s+(.+)/);
      if (headingMatch) {
        const level = headingMatch[1].length;
        out.push(`<h${level}>${inlineMarkdown(headingMatch[2])}</h${level}>`);
        i++;
        continue;
      }

      // Horizontal rule
      if (/^[-*_]{3,}$/.test(trimmed)) {
        out.push('<hr>');
        i++;
        continue;
      }

      // Blockquote
      if (trimmed.startsWith('>')) {
        const quoteLines = [];
        while (i < lines.length && lines[i].trim().startsWith('>')) {
          quoteLines.push(lines[i].trim().slice(1).trimStart());
          i++;
        }
        out.push(`<blockquote>${render(quoteLines.join('\n'))}</blockquote>`);
        continue;
      }

      // Unordered list
      if (/^[-*+]\s/.test(trimmed)) {
        const listItems = [];
        while (i < lines.length && /^[-*+]\s/.test(lines[i].trim())) {
          listItems.push(`<li>${inlineMarkdown(lines[i].trim().slice(2))}</li>`);
          i++;
        }
        out.push(`<ul>${listItems.join('')}</ul>`);
        continue;
      }

      // Ordered list
      if (/^\d+\.\s/.test(trimmed)) {
        const listItems = [];
        while (i < lines.length && /^\d+\.\s/.test(lines[i].trim())) {
          listItems.push(`<li>${inlineMarkdown(lines[i].trim().replace(/^\d+\.\s/, ''))}</li>`);
          i++;
        }
        out.push(`<ol>${listItems.join('')}</ol>`);
        continue;
      }

      // Blank line — paragraph separator
      if (trimmed === '') {
        i++;
        continue;
      }

      // Paragraph — collect consecutive non-blank, non-special lines
      const paraLines = [];
      while (
        i < lines.length &&
        lines[i].trim() !== '' &&
        !lines[i].trim().startsWith('#') &&
        !lines[i].trim().startsWith('```') &&
        !lines[i].trim().startsWith('>') &&
        !/^[-*+]\s/.test(lines[i].trim()) &&
        !/^\d+\.\s/.test(lines[i].trim())
      ) {
        paraLines.push(inlineMarkdown(lines[i]));
        i++;
      }
      if (paraLines.length) {
        out.push(`<p>${paraLines.join('<br>')}</p>`);
      }
    }

    return out.join('\n');
  }

  global.markdownRender = render;
})(window);
