import type { Directive, DirectiveHook } from 'vue'
import { T } from '@/i18n'

let marked: (s: string) => string
async function getRender() {
  if (marked) return marked
  return Promise.all([
    import('marked'),
    import('@/utils/highlight'),
    import('dompurify'),
  ]).then(
    ([{ marked: marked_ }, { default: hljs }, { default: DOMPurify }]) => {
      marked_.setOptions({
        highlight: (code, language) => {
          const validLanguage = hljs.getLanguage(language)
            ? language
            : 'plaintext'
          return hljs.highlight(code, { language: validLanguage }).value
        },
      })

      DOMPurify.addHook('afterSanitizeAttributes', (node) => {
        if ('target' in node) {
          node.setAttribute('target', '_blank')
        }
      })

      marked = (s) => {
        return DOMPurify.sanitize(marked_.parse(s))
      }
      return marked
    }
  )
}

const render: DirectiveHook = (el, binding) => {
  el._currentMarkdownContent = binding.value
  getRender().then(
    (render) => {
      if (el._currentMarkdownContent === el._renderedMarkdownContent) return
      el.innerHTML = render(el._currentMarkdownContent)
      el._renderedMarkdownContent = el._currentMarkdownContent
    },
    (e) => {
      console.error('markdown render error: ', e)
      el.innerHTML = `<p style="text-align: center;">${T('md.error')}</p>`
    }
  )
}

export default {
  beforeMount: render,
  updated: render,
} as Directive
