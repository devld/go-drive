import { Marked } from 'marked'
import { markedHighlight } from 'marked-highlight'
import hljs from './highlight'
import DOMPurify from 'dompurify'

const marked = new Marked(
  markedHighlight({
    langPrefix: 'hljs language-',
    highlight(code, lang) {
      const validLanguage: string = hljs.getLanguage(lang) ? lang : 'plaintext'
      return hljs.highlight(code, { language: validLanguage }).value
    },
  })
)

DOMPurify.addHook('afterSanitizeAttributes', (node) => {
  if ('target' in node) {
    node.setAttribute('target', '_blank')
  }
  if ('href' in node) {
    const a = node as HTMLAnchorElement
    const href = a.getAttribute('href') ?? ''
    if (href.startsWith('#')) {
      a.dataset.href = href
      a.href = 'javascript:;'
    }
  }
})

export default (s: string) =>
  DOMPurify.sanitize(marked.parse(s, { async: false }))
