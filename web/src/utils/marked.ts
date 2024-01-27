import { marked } from 'marked'
import hljs from './highlight'
import DOMPurify from 'dompurify'

marked.setOptions({
  highlight: (code, language) => {
    const validLanguage: string = hljs.getLanguage(language)
      ? language
      : 'plaintext'
    return hljs.highlight(code, { language: validLanguage }).value
  },
})

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

export default (s: string) => DOMPurify.sanitize(marked.parse(s))
